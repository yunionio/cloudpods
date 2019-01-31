package qcloud

import (
	"fmt"
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
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
1.HTTP\HTTPS监听调度算法没同步成功
2.后端服务器端口权重没同步成功
3.后端服务器组待确认

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
	panic("implement me")
}

func (self *SLoadbalancer) GetChargeType() string {
	panic("implement me")
}

func (self *SLoadbalancer) Delete() error {
	panic("implement me")
}

func (self *SLoadbalancer) Start() error {
	panic("implement me")
}

func (self *SLoadbalancer) Stop() error {
	panic("implement me")
}

func (self *SLoadbalancer) GetILoadBalancerListeners() ([]cloudprovider.ICloudLoadbalancerListener, error) {
	panic("implement me")
}

func (self *SLoadbalancer) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	panic("implement me")
}

func (self *SLoadbalancer) CreateILoadBalancerBackendGroup(group *cloudprovider.SLoadbalancerBackendGroup) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	panic("implement me")
}

func (self *SLoadbalancer) GetILoadBalancerBackendGroupById(groupId string) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	panic("implement me")
}

func (self *SLoadbalancer) CreateILoadBalancerListener(listener *cloudprovider.SLoadbalancerListener) (cloudprovider.ICloudLoadbalancerListener, error) {
	panic("implement me")
}

func (self *SLoadbalancer) GetILoadBalancerListenerById(listenerId string) (cloudprovider.ICloudLoadbalancerListener, error) {
	panic("implement me")
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
	listeners, err := self.region.GetLoadbalancerListeners(self.GetId(), self.Forward, "")
	if err != nil {
		return nil, err
	}

	for i := range listeners {
		listeners[i].lb = self
	}

	return listeners, nil
}

func (self *SLoadbalancer) GetILoadbalancerListeners() ([]cloudprovider.ICloudLoadbalancerListener, error) {
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

func (self *SLoadbalancer) GetILoadbalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
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
	if err != nil && len(lbs) == 1 {
		return &lbs[0], nil
	}

	return nil, err
}
