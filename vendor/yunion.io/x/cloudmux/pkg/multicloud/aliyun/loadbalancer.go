// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aliyun

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
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
	multicloud.SLoadbalancerBase
	AliyunTags
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
	CreateTime               time.Time           //负载均衡实例的创建时间。
	MasterZoneId             string              //实例的主可用区ID。
	SlaveZoneId              string              //实例的备可用区ID。
	InternetChargeType       TInternetChargeType //公网实例的计费方式。取值：paybybandwidth：按带宽计费 paybytraffic：按流量计费（默认值） 说明 当 PayType参数的值为PrePay时，只支持按带宽计费。
	InternetChargeTypeAlias  TInternetChargeType

	PayType          string //实例的计费类型，取值：PayOnDemand：按量付费 PrePay：预付费
	ResourceGroupId  string //企业资源组ID。
	LoadBalancerSpec string //负载均衡实例的的性能规格
	Bandwidth        int    //按带宽计费的公网型实例的带宽峰值
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
		return api.LB_STATUS_ENABLED
	}
	return api.LB_STATUS_DISABLED
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

func (lb *SLoadbalancer) GetNetworkIds() []string {
	return []string{lb.VSwitchId}
}

func (lb *SLoadbalancer) GetZoneId() string {
	zone, err := lb.region.getZoneById(transZoneIdToEcsZoneId(lb.region, "elb", lb.MasterZoneId))
	if err != nil {
		log.Errorf("failed to find zone for lb %s error: %v", lb.LoadBalancerName, err)
		return ""
	}
	return zone.GetGlobalId()
}

func (self *SLoadbalancer) GetZone1Id() string {
	return ""
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
	lb := SLoadbalancer{region: region}
	return &lb, body.Unmarshal(&lb)
}

func (lb *SLoadbalancer) Delete(ctx context.Context) error {
	params := map[string]string{}
	params["RegionId"] = lb.region.RegionId
	params["LoadBalancerId"] = lb.LoadBalancerId
	_, err := lb.region.lbRequest("DeleteLoadBalancer", params)
	return err
}

func (lb *SLoadbalancer) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
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

func (lb *SLoadbalancer) CreateILoadBalancerBackendGroup(group *cloudprovider.SLoadbalancerBackendGroup) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	switch group.GroupType {
	case api.LB_BACKENDGROUP_TYPE_NORMAL:
		group, err := lb.region.CreateLoadbalancerBackendGroup(group.Name, lb.LoadBalancerId, group.Backends)
		if err != nil {
			return nil, err
		}
		group.lb = lb
		return group, nil
	case api.LB_BACKENDGROUP_TYPE_MASTER_SLAVE:
		group, err := lb.region.CreateLoadbalancerMasterSlaveBackendGroup(group.Name, lb.LoadBalancerId, group.Backends)
		if err != nil {
			return nil, err
		}
		group.lb = lb
		return group, nil
	default:
		return nil, fmt.Errorf("Unsupport backendgroup type %s", group.GroupType)
	}
}

func (lb *SLoadbalancer) CreateILoadBalancerListener(ctx context.Context, listener *cloudprovider.SLoadbalancerListenerCreateOptions) (cloudprovider.ICloudLoadbalancerListener, error) {
	switch listener.ListenerType {
	case api.LB_LISTENER_TYPE_TCP:
		return lb.region.CreateLoadbalancerTCPListener(lb, listener)
	case api.LB_LISTENER_TYPE_UDP:
		return lb.region.CreateLoadbalancerUDPListener(lb, listener)
	case api.LB_LISTENER_TYPE_HTTP:
		return lb.region.CreateLoadbalancerHTTPListener(lb, listener)
	case api.LB_LISTENER_TYPE_HTTPS:
		return lb.region.CreateLoadbalancerHTTPSListener(lb, listener)
	}
	return nil, fmt.Errorf("unsupport listener type %s", listener.ListenerType)
}

func (lb *SLoadbalancer) GetLoadbalancerSpec() string {
	if len(lb.LoadBalancerSpec) == 0 {
		lb.Refresh()
	}
	return lb.LoadBalancerSpec
}

func (lb *SLoadbalancer) GetChargeType() string {
	chargeType := lb.InternetChargeType
	if len(lb.InternetChargeTypeAlias) > 0 {
		chargeType = lb.InternetChargeTypeAlias
	}
	switch chargeType {
	case "paybybandwidth":
		return api.LB_CHARGE_TYPE_BY_BANDWIDTH
	case "paybytraffic":
		return api.LB_CHARGE_TYPE_BY_TRAFFIC
	default:
		return string(chargeType)
	}
}

func (lb *SLoadbalancer) GetCreatedAt() time.Time {
	return lb.CreateTime
}

func (lb *SLoadbalancer) GetEgressMbps() int {
	if lb.Bandwidth < 1 {
		return 0
	}
	return lb.Bandwidth
}

func (lb *SLoadbalancer) GetILoadBalancerBackendGroupById(groupId string) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	groups, err := lb.GetILoadBalancerBackendGroups()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(groups); i++ {
		if groups[i].GetGlobalId() == groupId {
			return groups[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (lb *SLoadbalancer) GetIEIP() (cloudprovider.ICloudEIP, error) {
	if lb.AddressType == "internet" {
		eip := SEipAddress{
			region:         lb.region,
			IpAddress:      lb.Address,
			InstanceId:     lb.GetGlobalId(),
			InstanceType:   EIP_INTANNCE_TYPE_SLB,
			Status:         EIP_STATUS_INUSE,
			AllocationId:   lb.GetGlobalId(),
			AllocationTime: lb.CreateTime,
			Bandwidth:      lb.Bandwidth,
		}
		switch lb.GetChargeType() {
		case api.LB_CHARGE_TYPE_BY_BANDWIDTH:
			eip.InternetChargeType = InternetChargeByBandwidth
		case api.LB_CHARGE_TYPE_BY_TRAFFIC:
			eip.InternetChargeType = InternetChargeByTraffic
		}
		return &eip, nil
	}
	eips, total, err := lb.region.GetEips("", lb.LoadBalancerId, "", 0, 1)
	if err != nil {
		return nil, errors.Wrapf(err, "lb.region.GetEips(%s)", lb.LoadBalancerId)
	}
	if total != 1 {
		return nil, cloudprovider.ErrNotFound
	}
	eips[0].region = lb.region
	return &eips[0], nil
}

func (region *SRegion) loadbalancerOperation(loadbalancerId, status string) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	params["LoadBalancerStatus"] = status
	_, err := region.lbRequest("SetLoadBalancerStatus", params)
	return err
}

func (lb *SLoadbalancer) Start() error {
	if lb.LoadBalancerStatus != "active" {
		return lb.region.loadbalancerOperation(lb.LoadBalancerId, "active")
	}
	return nil
}

func (lb *SLoadbalancer) Stop() error {
	if lb.LoadBalancerStatus != "inactive" {
		return lb.region.loadbalancerOperation(lb.LoadBalancerId, "inactive")
	}
	return nil
}

func (lb *SLoadbalancer) GetILoadBalancerListenerById(listenerId string) (cloudprovider.ICloudLoadbalancerListener, error) {
	listener, err := lb.GetILoadBalancerListeners()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(listener); i++ {
		if listener[i].GetGlobalId() == listenerId {
			return listener[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (lb *SLoadbalancer) GetILoadBalancerListeners() ([]cloudprovider.ICloudLoadbalancerListener, error) {
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

func (lb *SLoadbalancer) GetProjectId() string {
	return lb.ResourceGroupId
}

func (lb *SLoadbalancer) SetTags(tags map[string]string, replace bool) error {
	return lb.region.SetResourceTags(ALIYUN_SERVICE_SLB, "instance", lb.LoadBalancerId, tags, replace)
}

func (lb *SLoadbalancer) GetTags() (map[string]string, error) {
	_, tags, err := lb.region.ListSysAndUserTags("slb", "instance", lb.LoadBalancerId)
	return tags, err
}

// mapping aliyun finance zoneId to aliyun finance ecs zoneId
func transZoneIdToEcsZoneId(region *SRegion, service, zoneId string) string {
	if region.GetCloudEnv() == ALIYUN_FINANCE_CLOUDENV {
		switch service {
		case "elb", "redis":
			if utils.IsInStringArray(zoneId, []string{"cn-hangzhou-finance-b", "cn-hangzhou-finance-c", "cn-hangzhou-finance-d"}) {
				return strings.Replace(zoneId, "-finance", "", -1)
			}
		default:
			return zoneId
		}
	}

	return zoneId
}

// mapping aliyun finance ecs zoneId to dest service zone id
func transZoneIdFromEcsZoneId(region *SRegion, service, zoneId string) string {
	if region.GetCloudEnv() == ALIYUN_FINANCE_CLOUDENV {
		switch service {
		case "elb", "redis":
			if utils.IsInStringArray(zoneId, []string{"cn-hangzhou-b", "cn-hangzhou-c", "cn-hangzhou-d"}) {
				return strings.Replace(zoneId, "cn-hangzhou", "cn-hangzhou-finance", -1)
			}
		default:
			return zoneId
		}
	}

	return zoneId
}

// mapping aliyun finance regionId to aliyun finance ecs regionId
func transRegionIdToEcsRegionId(region *SRegion, service string) string {
	if region.GetCloudEnv() == ALIYUN_FINANCE_CLOUDENV {
		switch service {
		case "redis":
			if region.GetId() == "cn-hangzhou-finance" {
				return "cn-hangzhou"
			}
		default:
			return region.GetId()
		}
	}

	return region.GetId()
}

// mapping aliyun finance regionId from aliyun finance ecs regionId
func transRegionIdFromEcsRegionId(region *SRegion, service string) string {
	if region.GetCloudEnv() == ALIYUN_FINANCE_CLOUDENV {
		switch service {
		case "redis":
			if region.GetId() == "cn-hangzhou" {
				return "cn-hangzhou-finance"
			}
		default:
			return region.GetId()
		}
	}

	return region.GetId()
}

func fetchMasterZoneId(zoneId string) string {
	// cn-shenzhen-finance-1MAZ2(d,e)
	i := strings.Index(zoneId, "MAZ")
	if i > 0 {
		s := strings.Index(zoneId, "(")
		e := strings.Index(zoneId, ",")
		if s >= 0 && e >= 0 {
			return zoneId[0:i] + zoneId[s+1:e]
		}
	}

	return zoneId
}
