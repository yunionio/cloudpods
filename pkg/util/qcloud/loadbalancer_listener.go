package qcloud

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

var HTTP_CODES = []string{
	models.LB_HEALTH_CHECK_HTTP_CODE_1xx,
	models.LB_HEALTH_CHECK_HTTP_CODE_2xx,
	models.LB_HEALTH_CHECK_HTTP_CODE_3xx,
	models.LB_HEALTH_CHECK_HTTP_CODE_4xx,
	models.LB_HEALTH_CHECK_HTTP_CODE_5xx,
}

type certificate struct {
	SSLMode  string `json:"SSLMode"`
	CERTCAID string `json:"CertCaId"`
	CERTID   string `json:"CertId"`
}

/*
健康检查状态码（仅适用于HTTP/HTTPS转发规则）。可选值：1~31，默认 31。
1 表示探测后返回值 1xx 表示健康，2 表示返回 2xx 表示健康，4 表示返回 3xx 表示健康，8 表示返回 4xx 表示健康，16 表示返回 5xx 表示健康。
若希望多种码都表示健康，则将相应的值相加。
*/
type healthCheck struct {
	HTTPCheckDomain string `json:"HttpCheckDomain"`
	HealthSwitch    int    `json:"HealthSwitch"`
	HTTPCheckPath   string `json:"HttpCheckPath"`
	HTTPCheckMethod string `json:"HttpCheckMethod"`
	UnHealthNum     int    `json:"UnHealthNum"`
	IntervalTime    int    `json:"IntervalTime"`
	HTTPCode        int    `json:"HttpCode"` // 健康检查状态码（仅适用于HTTP/HTTPS转发规则）。可选值：1~31，默认 31。
	HealthNum       int    `json:"HealthNum"`
	TimeOut         int    `json:"TimeOut"`
}

type SLBListener struct {
	lb *SLoadbalancer

	Protocol          string            `json:"Protocol"` // 监听器协议类型，取值 TCP | UDP | HTTP | HTTPS | TCP_SSL
	Certificate       certificate       `json:"Certificate"`
	SniSwitch         int64             `json:"SniSwitch"`   // 是否开启SNI特性（本参数仅对于HTTPS监听器有意义）
	HealthCheck       healthCheck       `json:"HealthCheck"` // 仅适用于TCP/UDP/TCP_SSL监听器
	ListenerID        string            `json:"ListenerId"`
	ListenerName      string            `json:"ListenerName"`
	Rules             []SLBListenerRule `json:"Rules"` // 监听器下的全部转发规则（本参数仅对于HTTP/HTTPS监听器有意义）
	Scheduler         string            `json:"Scheduler"`
	SessionExpireTime int               `json:"SessionExpireTime"` // 会话保持时间，单位：秒。可选值：30~3600，默认 0，表示不开启。此参数仅适用于TCP/UDP监听器。
	Port              int               `json:"Port"`
	ClassicListener   bool              // 这个字段是在qcloud返回字段基础上,额外增加的字段。用于区分listener 是否是classic。
}

// todo: 腾讯云后端端口不是与listener绑定的
func (self *SLBListener) GetBackendServerPort() int {
	return 0
}

// https://cloud.tencent.com/document/product/214/30691
// todo: 调度规则原用监听器的Scheduler ok？ https 协议怎么兼容？
func (self *SLBListener) CreateILoadBalancerListenerRule(rule *cloudprovider.SLoadbalancerListenerRule) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	requestId, err := self.lb.region.CreateLoadbalancerListenerRule(self.lb.GetId(),
		self.GetId(),
		rule.Domain,
		rule.Path,
		&self.Scheduler,
		&self.SessionExpireTime)
	if err != nil {
		return nil, err
	}

	err = self.lb.region.WaitLBTaskSuccess(requestId, 5*time.Second, 60*time.Second)
	if err != nil {
		return nil, err
	}

	err = self.Refresh()
	if err != nil {
		return nil, err
	}

	for _, r := range self.Rules {
		if r.GetPath() == rule.Path {
			return &r, nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func (self *SLBListener) GetILoadBalancerListenerRuleById(ruleId string) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rules, err := self.GetILoadbalancerListenerRules()
	if err != nil {
		return nil, err
	}

	for _, rule := range rules {
		if rule.GetId() == ruleId {
			return rule, nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func (self *SLBListener) Start() error {
	return nil
}

func (self *SLBListener) Stop() error {
	return cloudprovider.ErrNotSupported
}

// https://cloud.tencent.com/document/product/214/30677
func (self *SLBListener) Sync(listener *cloudprovider.SLoadbalancerListener) error {
	hc := getHealthCheck(listener)
	cert := getCertificate(listener)
	requestId, err := self.lb.region.UpdateLoadbalancerListener(
		self.lb.GetId(),
		self.GetId(),
		&listener.Name,
		getScheduler(listener),
		&listener.StickySessionCookieTimeout,
		hc,
		cert)
	if err != nil {
		return err
	}

	return self.lb.region.WaitLBTaskSuccess(requestId, 5*time.Second, 60*time.Second)
}

func (self *SLBListener) Delete() error {
	requestId, err := self.lb.region.DeleteLoadbalancerListener(self.lb.GetId(), self.GetId())
	if err != nil {
		return err
	}

	return self.lb.region.WaitLBTaskSuccess(requestId, 5*time.Second, 60*time.Second)
}

// https://cloud.tencent.com/document/api/214/30694#ClassicalListener
type SLBClassicListener struct {
	InstancePort  int64  `json:"InstancePort"`
	CERTCAID      string `json:"CertCaId"`
	Status        int64  `json:"Status"`
	CERTID        string `json:"CertId"`
	Protocol      string `json:"Protocol"`
	TimeOut       int    `json:"TimeOut"`
	HTTPHash      string `json:"HttpHash"` // 公网固定IP型的 HTTP、HTTPS 协议监听器的轮询方法。wrr 表示按权重轮询，ip_hash 表示根据访问的源 IP 进行一致性哈希方式来分发
	UnhealthNum   int    `json:"UnhealthNum"`
	IntervalTime  int    `json:"IntervalTime"`
	ListenerID    string `json:"ListenerId"`
	ListenerPort  int    `json:"ListenerPort"`
	HTTPCheckPath string `json:"HttpCheckPath"`
	HealthNum     int    `json:"HealthNum"`
	ListenerName  string `json:"ListenerName"`
	HealthSwitch  int    `json:"HealthSwitch"`
	SSLMode       string `json:"SSLMode"`
	SessionExpire int    `json:"SessionExpire"`
	HTTPCode      int    `json:"HttpCode"`
}

func (self *SLBClassicListener) ToLBListener() SLBListener {
	// 转换之后丢弃了 InstancePort、Status、HttpHash
	return SLBListener{
		Protocol: self.Protocol,
		Certificate: certificate{
			SSLMode:  self.SSLMode,
			CERTCAID: self.CERTCAID,
			CERTID:   self.CERTID,
		},
		HealthCheck: healthCheck{
			HTTPCheckDomain: "",
			HealthSwitch:    self.HealthSwitch,
			HTTPCheckPath:   self.HTTPCheckPath,
			HTTPCheckMethod: "",
			UnHealthNum:     self.UnhealthNum,
			IntervalTime:    self.IntervalTime,
			HTTPCode:        self.HTTPCode,
			HealthNum:       self.HealthNum,
			TimeOut:         self.TimeOut,
		},
		ListenerID:        self.ListenerID,
		ListenerName:      self.ListenerName,
		Rules:             nil,
		Scheduler:         self.HTTPHash,
		SessionExpireTime: self.SessionExpire,
		Port:              self.ListenerPort,
		ClassicListener:   true,
	}
}

func (self *SLBListener) GetId() string {
	return self.ListenerID
}

func (self *SLBListener) GetName() string {
	return self.ListenerName
}

func (self *SLBListener) GetGlobalId() string {
	return self.ListenerID
}

// 腾讯云负载均衡没有启用禁用操作
func (self *SLBListener) GetStatus() string {
	return models.LB_STATUS_ENABLED
}

func (self *SLBListener) Refresh() error {
	listeners, err := self.lb.region.GetLoadbalancerListeners(self.lb.GetId(), self.lb.Forward, "")
	if err != nil {
		return err
	}

	for _, listener := range listeners {
		if listener.GetId() == self.GetId() {
			listener.lb = self.lb
			err := jsonutils.Update(self, listener)
			if err != nil {
				return err
			}
		}
	}

	return cloudprovider.ErrNotFound
}

func (self *SLBListener) IsEmulated() bool {
	return false
}

func (self *SLBListener) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SLBListener) GetListenerType() string {
	switch self.Protocol {
	case "TCP":
		return models.LB_LISTENER_TYPE_TCP
	case "UDP":
		return models.LB_LISTENER_TYPE_UDP
	case "HTTP":
		return models.LB_LISTENER_TYPE_HTTP
	case "HTTPS":
		return models.LB_LISTENER_TYPE_HTTPS
	case "TCP_SSL":
		return models.LB_LISTENER_TYPE_TCP
	default:
		return ""
	}
}

func (self *SLBListener) GetListenerPort() int {
	return self.Port
}

func (self *SLBListener) GetScheduler() string {
	switch strings.ToLower(self.Scheduler) {
	case "wrr":
		return models.LB_SCHEDULER_WRR
	case "ip_hash":
		return models.LB_SCHEDULER_SCH
	case "least_conn":
		return models.LB_SCHEDULER_WLC
	default:
		return ""
	}
}

func (self *SLBListener) GetAclStatus() string {
	return models.LB_BOOL_OFF
}

func (self *SLBListener) GetAclType() string {
	return ""
}

func (self *SLBListener) GetAclId() string {
	return ""
}

func (self *SLBListener) GetHealthCheck() string {
	if self.HealthCheck.HealthSwitch == 0 {
		return models.LB_HEALTH_CHECK_DISABLE
	} else {
		return models.LB_HEALTH_CHECK_ENABLE
	}
}

// todo: ?待确认
func (self *SLBListener) GetHealthCheckType() string {
	if len(self.HealthCheck.HTTPCheckMethod) > 0 {
		return models.LB_HEALTH_CHECK_HTTP
	} else {
		return models.LB_HEALTH_CHECK_TCP
	}
}

func (self *SLBListener) GetHealthCheckTimeout() int {
	return self.HealthCheck.TimeOut
}

func (self *SLBListener) GetHealthCheckInterval() int {
	return self.HealthCheck.IntervalTime
}

func (self *SLBListener) GetHealthCheckRise() int {
	return self.HealthCheck.HealthNum
}

func (self *SLBListener) GetHealthCheckFail() int {
	return self.HealthCheck.UnHealthNum
}

func (self *SLBListener) GetHealthCheckReq() string {
	return ""
}

func (self *SLBListener) GetHealthCheckExp() string {
	return ""
}

func (self *SLBListener) GetBackendGroup() *SLBBackendGroup {
	t := self.GetListenerType()
	// http、https类型的监听不能直接绑定服务器
	if t == models.LB_LISTENER_TYPE_HTTP || t == models.LB_LISTENER_TYPE_HTTPS {
		return nil
	} else {
		return &SLBBackendGroup{lb: self.lb, listener: self}
	}
}

func (self *SLBListener) GetBackendGroupId() string {
	bg := self.GetBackendGroup()
	if bg == nil {
		return ""
	}

	return bg.GetId()
}

func (self *SLBListener) GetHealthCheckDomain() string {
	return self.HealthCheck.HTTPCheckDomain
}

func (self *SLBListener) GetHealthCheckURI() string {
	return self.HealthCheck.HTTPCheckPath
}

func (self *SLBListener) GetHealthCheckCode() string {
	codes := []string{}
	for i := uint8(0); i < 5; i++ {
		n := 1 << i
		if (self.HealthCheck.HTTPCode & n) == n {
			codes = append(codes, HTTP_CODES[i])
		}
	}

	return strings.Join(codes, ",")
}

// 仅http、https类型监听包含rules
func (self *SLBListener) GetILoadbalancerListenerRules() ([]cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rules := self.Rules
	iRules := []cloudprovider.ICloudLoadbalancerListenerRule{}
	for i := 0; i < len(rules); i++ {
		rules[i].listener = self
		iRules = append(iRules, &rules[i])
	}
	return iRules, nil
}

func (self *SLBListener) GetStickySession() string {
	if self.SessionExpireTime == 0 {
		return models.LB_BOOL_OFF
	} else {
		return models.LB_BOOL_ON
	}
}

// 支持基于 cookie 插入的会话保持能力 https://cloud.tencent.com/document/product/214/6154
func (self *SLBListener) GetStickySessionType() string {
	return models.LB_STICKY_SESSION_TYPE_INSERT
}

// todo: 腾讯云不支持？
func (self *SLBListener) GetStickySessionCookie() string {
	return ""
}

func (self *SLBListener) GetStickySessionCookieTimeout() int {
	return self.SessionExpireTime
}

/*
7层负载均衡系统提供 X-Forwarded-For 的方式获取访问者真实 IP，LB 侧默认开启

https://cloud.tencent.com/document/product/214/6151
七层转发获取来访真实IP的方法 https://cloud.tencent.com/document/product/214/3728
*/
func (self *SLBListener) XForwardedForEnabled() bool {
	switch self.GetListenerType() {
	case models.LB_LISTENER_TYPE_HTTP, models.LB_LISTENER_TYPE_HTTPS:
		return true
	default:
		return false
	}
}

// HTTP/HTTPS协议默认支持用户开启gzip压缩功能
// 负载均衡开启Gzip配置及检测方法说明 https://cloud.tencent.com/document/product/214/5404
func (self *SLBListener) GzipEnabled() bool {
	switch self.GetListenerType() {
	case models.LB_LISTENER_TYPE_HTTP, models.LB_LISTENER_TYPE_HTTPS:
		return true
	default:
		return false
	}
}

func (self *SLBListener) GetCertificateId() string {
	return self.Certificate.CERTID
}

// todo: ??
func (self *SLBListener) GetTLSCipherPolicy() string {
	return ""
}

// 负载均衡能力说明 https://cloud.tencent.com/document/product/214/6534
func (self *SLBListener) HTTP2Enabled() bool {
	return true
}

func (self *SRegion) GetLoadbalancerListeners(lbid string, t LB_TYPE, protocol string) ([]SLBListener, error) {
	params := map[string]string{"LoadBalancerId": lbid}
	if len(protocol) > 0 {
		params["Protocol"] = protocol
	}

	listeners := []SLBListener{}
	if t == LB_TYPE_CLASSIC {
		resp, err := self.clbRequest("DescribeClassicalLBListeners", params)
		if err != nil {
			return nil, err
		}

		clisteners := []SLBClassicListener{}
		err = resp.Unmarshal(&clisteners, "Listeners")
		if err != nil {
			return nil, err
		}

		for _, l := range clisteners {
			listeners = append(listeners, l.ToLBListener())
		}

		return listeners, nil
	}

	resp, err := self.clbRequest("DescribeListeners", params)
	if err != nil {
		return nil, err
	}

	err = resp.Unmarshal(&listeners, "Listeners")
	if err != nil {
		return nil, err
	}

	return listeners, nil
}

//  返回requestID
func (self *SRegion) CreateLoadbalancerListenerRule(lbid string, listenerId string, domain string, url string, scheduler *string, sessionExpireTime *int) (string, error) {
	if len(lbid) == 0 {
		return "", fmt.Errorf("loadbalancer id should not be empty")
	}

	params := map[string]string{
		"LoadBalancerId": lbid,
		"ListenerId":     listenerId,
		"Rules.0.Domain": domain,
		"Rules.0.Url":    url,
	}

	if scheduler != nil && len(*scheduler) > 0 {
		params["Rules.0.Scheduler"] = *scheduler
	}

	if sessionExpireTime != nil {
		params["Rules.0.SessionExpireTime"] = strconv.Itoa(*sessionExpireTime)
	}

	resp, err := self.clbRequest("CreateRule", params)
	if err != nil {
		return "", err
	}

	return resp.GetString("RequestId")
}

//  返回requestID
func (self *SRegion) DeleteLoadbalancerListener(lbid string, listenerId string) (string, error) {
	if len(lbid) == 0 {
		return "", fmt.Errorf("loadbalancer id should not be empty")
	}

	params := map[string]string{
		"LoadBalancerId": lbid,
		"ListenerId":     listenerId,
	}

	resp, err := self.clbRequest("DeleteListener", params)
	if err != nil {
		return "", err
	}

	return resp.GetString("RequestId")
}

// https://cloud.tencent.com/document/product/214/30681
func (self *SRegion) UpdateLoadbalancerListener(lbid string, listenerId string,listenerName *string, scheduler *string, sessionExpireTime *int, healthCheck *healthCheck,cert *certificate) (string, error) {
	if len(lbid) == 0 {
		return "", fmt.Errorf("loadbalancer id should not be empty")
	}

	if len(listenerId) == 0 {
		return "", fmt.Errorf("loadbalancer listener id should not be empty")
	}

	params := map[string]string{
		"LoadBalancerId": lbid,
		"ListenerId":     listenerId,
	}

	if listenerName != nil && len(*listenerName) > 0 {
		params["ListenerName"] = *listenerName
	}

	if scheduler != nil && len(*scheduler) > 0 {
		params["Scheduler"] = *scheduler
	}

	if sessionExpireTime != nil {
		params["SessionExpireTime"] = strconv.Itoa(*sessionExpireTime)
	}

	params = healthCheckParams(params, healthCheck)
	params = certificateParams(params, cert)

	resp, err := self.clbRequest("ModifyListener", params)
	if err != nil {
		return "", err
	}

	return resp.GetString("RequestId")
}

func getHealthCheck(listener *cloudprovider.SLoadbalancerListener) *healthCheck {
	var hc *healthCheck
	if listener.HealthCheck == models.LB_HEALTH_CHECK_ENABLE {
		hc = &healthCheck{
			HealthSwitch:    1,
			UnHealthNum:     listener.HealthCheckFail,
			IntervalTime:    listener.HealthCheckInterval,
			HealthNum:       listener.HealthCheckRise,
			TimeOut:         listener.HealthCheckTimeout,
		}

		httpCode := onecloudHealthCodeToQcloud(listener.HealthCheckHttpCode)
		if httpCode > 0 {
			hc.HTTPCode = httpCode
			hc.HTTPCheckMethod = "HEAD"  // todo: add column HttpCheckMethod in model
		    hc.HTTPCheckDomain = listener.HealthCheckDomain
		    hc.HTTPCheckPath = listener.HealthCheckURI
		}
	}

	return hc
}

func getCertificate(listener *cloudprovider.SLoadbalancerListener) *certificate {
	var cert *certificate
	if len(listener.CertificateID) > 0 {
		cert = &certificate{
			SSLMode:  "UNIDIRECTIONAL",
			CERTCAID: listener.CertificateID,
			CERTID:   "",
		}
	}

	return cert
}

func getProtocol(listener *cloudprovider.SLoadbalancerListener) string {
	switch listener.ListenerType {
	case models.LB_LISTENER_TYPE_HTTPS:
		return "HTTPS"
	case models.LB_LISTENER_TYPE_HTTP:
		return "HTTP"
	case models.LB_LISTENER_TYPE_TCP:
		return "TCP"
	case models.LB_LISTENER_TYPE_UDP:
		return "UDP"
	case "tcp_ssl":
		return "TCP_SSL"
	default:
		return ""
	}
}

func getScheduler(listener *cloudprovider.SLoadbalancerListener) *string {
	var sch string
	switch listener.Scheduler {
	case models.LB_SCHEDULER_WRR:
		sch = "WRR"
	case models.LB_SCHEDULER_WLC:
		sch = "LEAST_CONN"
	default:
		return nil
	}

	return &sch
}

func healthCheckParams(params map[string]string,hc *healthCheck) map[string]string {
	if hc != nil {
		params["HealthCheck.HealthSwitch"] = strconv.Itoa(hc.HealthSwitch)
		params["HealthCheck.TimeOut"] = strconv.Itoa(hc.TimeOut)
		params["HealthCheck.IntervalTime"] = strconv.Itoa(hc.IntervalTime)
		params["HealthCheck.HealthNum"] = strconv.Itoa(hc.HealthNum)
		params["HealthCheck.UnHealthNum"] = strconv.Itoa(hc.UnHealthNum)
		if hc.HTTPCode > 0 {
			params["HealthCheck.HttpCode"] = strconv.Itoa(hc.HTTPCode)
			params["HealthCheck.HttpCheckPath"] = hc.HTTPCheckPath
			params["HealthCheck.HttpCheckDomain"] = hc.HTTPCheckDomain
			params["HealthCheck.HttpCheckMethod"] = hc.HTTPCheckMethod
		}
	}

	return params
}

func certificateParams(params map[string]string,cert *certificate) map[string]string {
	if cert != nil {
		params["Certificate.SSLMode"] = cert.SSLMode
		params["Certificate.CertId"] = cert.CERTID
		if len(cert.CERTCAID) > 0 {
			params["Certificate.CertCaId"] = cert.CERTCAID
		}
	}

	return params
}