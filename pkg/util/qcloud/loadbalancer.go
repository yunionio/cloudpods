package qcloud

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

const (
	LB_ADDR_TYPE_INTERNAL = "INTERNAL"
	LB_ADDR_TYPE_OPEN     = "OPEN"
)

type LB_TYPE int64

const (
	LB_TYPE_CLASSIC     = LB_TYPE(0)
	LB_TYPE_APPLICATION = LB_TYPE(1)
)

/*
目前存在的问题：
1.HTTP\HTTPS监听调度算法没同步成功，原因是http、https监听并不关联转发策略，因此本身就没有调度算法
2.不能创建后端服务器组。原因是没办法找到关联的listener及生成external id
*/
// https://cloud.tencent.com/document/api/214/30694#LoadBalancer
type SLoadbalancer struct {
	region *SRegion

	Status           int64     `json:"Status"` // 0：创建中，1：正常运行
	Domain           string    `json:"Domain"`
	VpcID            string    `json:"VpcId"`
	Log              string    `json:"Log"`
	ProjectID        int64     `json:"ProjectId"`
	Snat             bool      `json:"Snat"`
	LoadBalancerID   string    `json:"LoadBalancerId"`
	LoadBalancerVips []string  `json:"LoadBalancerVips"`
	LoadBalancerType string    `json:"LoadBalancerType"` // 负载均衡实例的网络类型： OPEN：公网属性， INTERNAL：内网属性。
	LoadBalancerName string    `json:"LoadBalancerName"`
	Forward          LB_TYPE   `json:"Forward"` // 应用型负载均衡标识，1：应用型负载均衡，0：传统型的负载均衡。
	StatusTime       time.Time `json:"StatusTime"`
	OpenBGP          int64     `json:"OpenBgp"` // 高防 LB 的标识，1：高防负载均衡 0：非高防负载均衡。
	CreateTime       time.Time `json:"CreateTime"`
	Isolation        int64     `json:"Isolation"` // 0：表示未被隔离，1：表示被隔离。
	SubnetId         string    `json:"SubnetId"`
}

func (self *SLoadbalancer) GetLoadbalancerSpec() string {
	return ""
}

func (self *SLoadbalancer) GetChargeType() string {
	return models.LB_CHARGE_TYPE_BY_HOUR
}

// https://cloud.tencent.com/document/product/214/30689
func (self *SLoadbalancer) Delete() error {
	_, err := self.region.DeleteLoadbalancer(self.GetId())
	if err != nil {
		return err
	}

	return cloudprovider.WaitDeleted(self, 5*time.Second, 60*time.Second)
}

// 腾讯云loadbalance不支持启用/禁用
func (self *SLoadbalancer) Start() error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancer) Stop() error {
	return cloudprovider.ErrNotSupported
}

// 腾讯云无后端服务器组
// todo: 是否返回一个fake的后端服务器组
func (self *SLoadbalancer) CreateILoadBalancerBackendGroup(group *cloudprovider.SLoadbalancerBackendGroup) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SLoadbalancer) GetILoadBalancerBackendGroupById(groupId string) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	groups, err := self.GetILoadBalancerBackendGroups()
	if err != nil {
		return nil, err
	}

	for _, group := range groups {
		if group.GetId() == groupId {
			return group, nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func onecloudHealthCodeToQcloud(codes string) int {
	qcode := 0
	for i, code := range HTTP_CODES {
		if strings.Contains(code, codes) {
			c := 1 << uint(i)
			qcode += c
		}
	}

	return qcode
}

// https://cloud.tencent.com/document/product/214/30693
// Onecloud 不支持双向证书
/*
todo:  限制比较多必须加参数校验
HealthSwitch	Integer	否	是否开启健康检查：1（开启）、0（关闭）。
TimeOut	Integer	否	健康检查的响应超时时间，可选值：2~60，默认值：2，单位：秒。响应超时时间要小于检查间隔时间。
IntervalTime	Integer	否	健康检查探测间隔时间，默认值：5，可选值：5~300，单位：秒。
HealthNum	Integer	否	健康阈值，默认值：3，表示当连续探测三次健康则表示该转发正常，可选值：2~10，单位：次。
UnHealthNum	Integer	否	不健康阈值，默认值：3，表示当连续探测三次不健康则表示该转发异常，可选值：2~10，单位：次。
HttpCode	Integer	否	健康检查状态码（仅适用于HTTP/HTTPS转发规则）。可选值：1~31，默认 31。
1 表示探测后返回值 1xx 表示健康，2 表示返回 2xx 表示健康，4 表示返回 3xx 表示健康，8 表示返回 4xx 表示健康，16 表示返回 5xx 表示健康。若希望多种码都表示健康，则将相应的值相加。
HttpCheckPath	String	否	健康检查路径（仅适用于HTTP/HTTPS转发规则）。
HttpCheckDomain	String	否	健康检查域名（仅适用于HTTP/HTTPS转发规则）。
HttpCheckMethod	String	否	健康检查方法（仅适用于HTTP/HTTPS转发规则），取值为HEAD或GET。

SSLMode	String	是	认证类型，UNIDIRECTIONAL：单向认证，MUTUAL：双向认证
CertId	String	否	服务端证书的 ID，如果不填写此项则必须上传证书，包括 CertContent，CertKey，CertName。
CertCaId	String	否	客户端证书的 ID，如果 SSLMode=mutual，监听器如果不填写此项则必须上传客户端证书，包括 CertCaContent，CertCaName。
*/
func (self *SLoadbalancer) CreateILoadBalancerListener(listener *cloudprovider.SLoadbalancerListener) (cloudprovider.ICloudLoadbalancerListener, error) {
	sniSwitch := 0
	hc := getHealthCheck(listener)
	cert := getCertificate(listener)

	listenId, err := self.region.CreateLoadbalancerListener(self.GetId(),
		listener.Name,
		getProtocol(listener),
		listener.ListenerPort,
		getScheduler(listener),
		&listener.StickySessionCookieTimeout,
		&sniSwitch,
		hc,
		cert)

	if err != nil {
		return nil, err
	}

	time.Sleep(3 * time.Second)
	return self.GetILoadBalancerListenerById(listenId)
}

func (self *SLoadbalancer) GetILoadBalancerListenerById(listenerId string) (cloudprovider.ICloudLoadbalancerListener, error) {
	listeners, err := self.GetLoadbalancerListeners("")
	if err != nil {
		return nil, err
	}

	for _, listener := range listeners {
		if listener.GetId() == listenerId {
			return &listener, nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func (self *SLoadbalancer) GetId() string {
	return self.LoadBalancerID
}

func (self *SLoadbalancer) GetName() string {
	return self.LoadBalancerName
}

// add region?
func (self *SLoadbalancer) GetGlobalId() string {
	return self.LoadBalancerID
}

func (self *SLoadbalancer) GetStatus() string {
	switch self.Status {
	case 0:
		return models.LB_STATUS_INIT
	case 1:
		return models.LB_STATUS_ENABLED
	default:
		return models.LB_STATUS_UNKNOWN
	}
}

func (self *SLoadbalancer) Refresh() error {
	lb, err := self.region.GetLoadbalancer(self.GetId())
	if err != nil {
		return err
	}

	return jsonutils.Update(self, lb)
}

func (self *SLoadbalancer) IsEmulated() bool {
	return false
}

func (self *SLoadbalancer) GetMetadata() *jsonutils.JSONDict {
	meta := jsonutils.NewDict()
	meta.Add(jsonutils.NewInt(int64(self.Forward)), "Forward")
	meta.Add(jsonutils.NewInt(self.OpenBGP), "OpenBGP")
	meta.Add(jsonutils.NewString(self.Domain), "Domain")
	meta.Add(jsonutils.NewInt(self.ProjectID), "ProjectID")

	return meta
}

// todo： 腾讯云支持绑定多个地址。目前未找到相关文档描述。需要提工单询问。
// 目前先当作只能绑定一个IP处理
func (self *SLoadbalancer) GetAddress() string {
	return self.LoadBalancerVips[0]
}

func (self *SLoadbalancer) GetAddressType() string {
	switch self.LoadBalancerType {
	case LB_ADDR_TYPE_INTERNAL:
		return models.LB_ADDR_TYPE_INTRANET
	case LB_ADDR_TYPE_OPEN:
		return models.LB_ADDR_TYPE_INTERNET
	default:
		return ""
	}
}

func (self *SLoadbalancer) GetNetworkType() string {
	return models.LB_NETWORK_TYPE_VPC
}

func (self *SLoadbalancer) GetNetworkId() string {
	return self.SubnetId
}

func (self *SLoadbalancer) GetVpcId() string {
	return self.VpcID
}

func (self *SLoadbalancer) GetZoneId() string {
	return ""
}

func (self *SLoadbalancer) GetLoadbalancerListeners(protocal string) ([]SLBListener, error) {
	listeners, err := self.region.GetLoadbalancerListeners(self.GetId(), self.Forward, protocal)
	if err != nil {
		return nil, err
	}

	for i := range listeners {
		listeners[i].lb = self
	}

	return listeners, nil
}

func (self *SLoadbalancer) GetILoadBalancerListeners() ([]cloudprovider.ICloudLoadbalancerListener, error) {
	listeners, err := self.GetLoadbalancerListeners("")
	if err != nil {
		return nil, err
	}

	ilisteners := make([]cloudprovider.ICloudLoadbalancerListener, len(listeners))
	for i := range listeners {
		l := listeners[i]
		ilisteners[i] = &l
	}

	return ilisteners, nil
}

func (self *SLoadbalancer) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	listeners, err := self.GetLoadbalancerListeners("")
	if err != nil {
		return nil, err
	}

	bgs := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	for i := range listeners {
		listener := listeners[i]
		t := listener.GetListenerType()
		if t == models.LB_LISTENER_TYPE_HTTP || t == models.LB_LISTENER_TYPE_HTTPS {
			rules := listener.Rules
			for i := range rules {
				rule := rules[i]
				rule.listener = &listener
				bg := rule.GetBackendGroup()
				bgs = append(bgs, bg)
			}
		} else {
			bg := listener.GetBackendGroup()
			bgs = append(bgs, bg)
		}
	}

	ibgs := make([]cloudprovider.ICloudLoadbalancerBackendGroup, len(bgs))
	for i := range bgs {
		ibgs[i] = bgs[i]
	}

	return ibgs, nil
}

func (self *SRegion) GetLoadbalancers(ids []string) ([]SLoadbalancer, error) {
	params := map[string]string{}
	if ids != nil {
		for i, id := range ids {
			params[fmt.Sprintf("LoadBalancerIds.%d", i)] = id
		}
	}

	offset := 0
	limit := 100
	lbs := make([]SLoadbalancer, 0)
	for {
		params["Limit"] = strconv.Itoa(limit)
		params["Offset"] = strconv.Itoa(offset)

		resp, err := self.clbRequest("DescribeLoadBalancers", params)
		if err != nil {
			return nil, err
		}

		parts := make([]SLoadbalancer, 0)
		err = resp.Unmarshal(&parts, "LoadBalancerSet")
		if err != nil {
			return nil, err
		}

		_total, err := resp.Float("TotalCount")
		if err != nil {
			return nil, err
		}

		total := int(_total)
		if err != nil {
			return nil, err
		}

		lbs = append(lbs, parts...)
		offset += limit
		if offset >= total {
			for i := range lbs {
				lbs[i].region = self
			}

			return lbs, err
		}
	}
}

func (self *SRegion) GetLoadbalancer(id string) (*SLoadbalancer, error) {
	if len(id) == 0 {
		return nil, fmt.Errorf("GetLoadbalancer id should not empty")
	}

	lbs, err := self.GetLoadbalancers([]string{id})
	if err == nil && len(lbs) == 0 {
		return nil, cloudprovider.ErrNotFound
	}

	if err != nil && len(lbs) == 1 {
		return &lbs[0], nil
	}

	return nil, err
}

/*
返回requstid 用于异步任务查询
https://cloud.tencent.com/document/product/214/30689
*/
func (self *SRegion) DeleteLoadbalancer(lbid string) (string, error) {
	if len(lbid) == 0 {
		return "", fmt.Errorf("loadbalancer id should not be empty")
	}

	params := map[string]string{"LoadBalancerIds.0": lbid}
	resp, err := self.clbRequest("DeleteLoadBalancer", params)
	if err != nil {
		return "", err
	}

	return resp.GetString("RequestId")
}

/*
https://cloud.tencent.com/document/product/214/30693
SNI 特性是什么？？
*/
func (self *SRegion) CreateLoadbalancerListener(lbid, name, protocol string, port int, scheduler *string, sessionExpireTime, sniSwitch *int, healthCheck *healthCheck, cert *certificate) (string, error) {
	if len(lbid) == 0 {
		return "", fmt.Errorf("loadbalancer id should not be empty")
	}

	params := map[string]string{
		"LoadBalancerId": lbid,
		"Ports.0":        strconv.Itoa(port),
		"Protocol":       protocol,
	}

	if len(name) > 0 {
		params["ListenerNames.0"] = name
	}

	if sniSwitch != nil {
		params["SniSwitch"] = strconv.Itoa(*sniSwitch)
	}

	if sessionExpireTime != nil {
		params["SessionExpireTime"] = strconv.Itoa(*sessionExpireTime)
	}

	if scheduler != nil && len(*scheduler) > 0 {
		params["Scheduler"] = *scheduler
	}

	params = healthCheckParams(params, healthCheck)
	params = certificateParams(params, cert)

	resp, err := self.clbRequest("CreateListener", params)
	if err != nil {
		return "", err
	}

	listeners, err := resp.GetArray("ListenerIds")
	if err != nil {
		return "", err
	}

	if len(listeners) == 0 {
		return "", fmt.Errorf("CreateLoadbalancerListener no listener id returned: %s", resp.String())
	} else if len(listeners) == 1 {
		return listeners[0].GetString()
	} else {
		return "", fmt.Errorf("CreateLoadbalancerListener mutliple listener id returned: %s", resp.String())
	}
}

// https://cloud.tencent.com/document/product/214/30683
// 任务的当前状态。 0：成功，1：失败，2：进行中
func (self *SRegion) GetLBTaskStatus(requestId string) (string, error) {
	if len(requestId) == 0 {
		return "", fmt.Errorf("WaitTaskSuccess requestId should not be emtpy")
	}

	params := map[string]string{"TaskId": requestId}
	resp, err := self.clbRequest("DescribeTaskStatus", params)
	if err != nil {
		return "", err
	}

	status, err := resp.Get("Status")
	if err != nil {
		log.Debugf("WaitTaskSuccess failed %s: %s", err, resp.String())
		return "", err
	}

	_status, err := status.Float()
	return fmt.Sprintf("%1.f", _status), err
}

func (self *SRegion) WaitLBTaskSuccess(requestId string, interval time.Duration, timeout time.Duration) error {
	startTime := time.Now()
	for time.Now().Sub(startTime) < timeout {
		status, err := self.GetLBTaskStatus(requestId)
		if err != nil {
			return err
		}
		if status == "0" {
			return nil
		}

		if status == "1" {
			return fmt.Errorf("Task %s failed.", requestId)
		}

		time.Sleep(interval)
	}

	return cloudprovider.ErrTimeout
}
