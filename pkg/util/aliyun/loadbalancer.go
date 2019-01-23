package aliyun

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type ListenerProtocol string

const (
	ListenerProtocolTCP   ListenerProtocol = "tcp"
	ListenerProtocolUDP   ListenerProtocol = "udp"
	ListenerProtocolHTTP  ListenerProtocol = "http"
	ListenerProtocolHTTPS ListenerProtocol = "https"
)

type ListenerPorts struct {
	ListenerPort []int
}

type ListenerPortsAndProtocol struct {
	ListenerPortAndProtocol []ListenerPortAndProtocol
}

type ListenerPortAndProtocol struct {
	Description      string
	ListenerPort     int
	ListenerProtocol ListenerProtocol
}

type BackendServers struct {
	BackendServer []SLoadbalancerDefaultBackend
}

type SLoadbalancer struct {
	region *SRegion

	LoadBalancerId           string //负载均衡实例ID。
	LoadBalancerName         string //负载均衡实例的名称。
	LoadBalancerStatus       string //负载均衡实例状态：inactive: 此状态的实例监听不会再转发流量。active: 实例创建后，默认状态为active。 locked: 实例已经被锁定。
	Address                  string //负载均衡实例的服务地址。
	RegionId                 string //负载均衡实例的地域ID。
	RegionIdAlias            string //负载均衡实例的地域名称。
	AddressType              string //负载均衡实例的网络类型。
	VSwitchId                string //私网负载均衡实例的交换机ID。
	VpcId                    string //私网负载均衡实例的专有网络ID。
	NetworkType              string //私网负载均衡实例的网络类型：vpc：专有网络实例 classic：经典网络实例
	ListenerPorts            ListenerPorts
	ListenerPortsAndProtocol ListenerPortsAndProtocol
	BackendServers           BackendServers
	CreateTime               string //负载均衡实例的创建时间。
	MasterZoneId             string //实例的主可用区ID。
	SlaveZoneId              string //实例的备可用区ID。
	InternetChargeType       string //公网实例的计费方式。取值：paybybandwidth：按带宽计费 paybytraffic：按流量计费（默认值） 说明 当 PayType参数的值为PrePay时，只支持按带宽计费。
	PayType                  string //实例的计费类型，取值：PayOnDemand：按量付费 PrePay：预付费
	ResourceGroupId          string //企业资源组ID。

}

func (lb *SLoadbalancer) GetName() string {
	return lb.LoadBalancerName
}

func (lb *SLoadbalancer) GetId() string {
	return lb.LoadBalancerId
}

func (lb *SLoadbalancer) GetGlobalId() string {
	return lb.LoadBalancerId
}

func (lb *SLoadbalancer) GetStatus() string {
	if lb.LoadBalancerStatus == "active" {
		return models.LB_STATUS_ENABLED
	}
	return models.LB_STATUS_DISABLED
}

func (lb *SLoadbalancer) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (lb *SLoadbalancer) GetAddress() string {
	return lb.Address
}

func (lb *SLoadbalancer) GetAddressType() string {
	return lb.AddressType
}

func (lb *SLoadbalancer) GetNetworkType() string {
	return lb.NetworkType
}

func (lb *SLoadbalancer) GetNetworkId() string {
	return lb.VSwitchId
}

func (lb *SLoadbalancer) GetZoneId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_ALIYUN, lb.MasterZoneId)
}

func (lb *SLoadbalancer) IsEmulated() bool {
	return false
}

func (lb *SLoadbalancer) GetVpcId() string {
	return lb.VpcId
}

func (lb *SLoadbalancer) Refresh() error {
	loadbalancer, err := lb.region.GetLoadbalancerDetail(lb.LoadBalancerId)
	if err != nil {
		return err
	}
	return jsonutils.Update(lb, loadbalancer)
}

func (region *SRegion) GetLoadbalancers(ids []string) ([]SLoadbalancer, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	if ids != nil && len(ids) > 0 {
		params["LoadBalancerId"] = strings.Join(ids, ",")
	}
	body, err := region.lbRequest("DescribeLoadBalancers", params)
	if err != nil {
		return nil, err
	}
	lbs := []SLoadbalancer{}
	return lbs, body.Unmarshal(&lbs, "LoadBalancers", "LoadBalancer")
}

func (region *SRegion) GetLoadbalancerDetail(loadbalancerId string) (*SLoadbalancer, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	body, err := region.lbRequest("DescribeLoadBalancerAttribute", params)
	if err != nil {
		return nil, err
	}
	lb := SLoadbalancer{}
	return &lb, body.Unmarshal(&lb)
}

func (lb *SLoadbalancer) GetILoadbalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	ibackendgroups := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	{
		backendgroups, err := lb.region.GetLoadbalancerBackendgroups(lb.LoadBalancerId)
		if err != nil {
			return nil, err
		}

		for i := 0; i < len(backendgroups); i++ {
			backendgroups[i].lb = lb
			ibackendgroups = append(ibackendgroups, &backendgroups[i])
		}
	}
	{
		iDefaultBackendgroup := SLoadbalancerDefaultBackendGroup{lb: lb}
		ibackendgroups = append(ibackendgroups, &iDefaultBackendgroup)

	}
	{
		backendgroups, err := lb.region.GetLoadbalancerMasterSlaveBackendgroups(lb.LoadBalancerId)
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(backendgroups); i++ {
			backendgroups[i].lb = lb
			ibackendgroups = append(ibackendgroups, &backendgroups[i])
		}
	}
	return ibackendgroups, nil
}

func (lb *SLoadbalancer) GetILoadbalancerListeners() ([]cloudprovider.ICloudLoadbalancerListener, error) {
	loadbalancer, err := lb.region.GetLoadbalancerDetail(lb.LoadBalancerId)
	if err != nil {
		return nil, err
	}
	listeners := []cloudprovider.ICloudLoadbalancerListener{}
	for _, listenerInfo := range loadbalancer.ListenerPortsAndProtocol.ListenerPortAndProtocol {
		switch listenerInfo.ListenerProtocol {
		case ListenerProtocolHTTP:
			listener, err := lb.region.GetLoadbalancerHTTPListener(lb.LoadBalancerId, listenerInfo.ListenerPort)
			if err != nil {
				return nil, err
			}
			listener.lb = lb
			listeners = append(listeners, listener)
		case ListenerProtocolHTTPS:
			listener, err := lb.region.GetLoadbalancerHTTPSListener(lb.LoadBalancerId, listenerInfo.ListenerPort)
			if err != nil {
				return nil, err
			}
			listener.lb = lb
			listeners = append(listeners, listener)
		case ListenerProtocolTCP:
			listener, err := lb.region.GetLoadbalancerTCPListener(lb.LoadBalancerId, listenerInfo.ListenerPort)
			if err != nil {
				return nil, err
			}
			listener.lb = lb
			listeners = append(listeners, listener)
		case ListenerProtocolUDP:
			listener, err := lb.region.GetLoadbalancerUDPListener(lb.LoadBalancerId, listenerInfo.ListenerPort)
			if err != nil {
				return nil, err
			}
			listener.lb = lb
			listeners = append(listeners, listener)
		default:
			return nil, fmt.Errorf("failed to recognize %s type listener", listenerInfo.ListenerProtocol)
		}
	}
	return listeners, nil
}
