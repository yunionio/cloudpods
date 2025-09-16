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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNlb struct {
	multicloud.SLoadbalancerBase
	AliyunTags
	region *SRegion

	LoadBalancerId               string                   `json:"LoadBalancerId"`
	LoadBalancerName             string                   `json:"LoadBalancerName"`
	LoadBalancerStatus           string                   `json:"LoadBalancerStatus"`
	LoadBalancerBusinessStatus   string                   `json:"LoadBalancerBusinessStatus"`
	LoadBalancerType             string                   `json:"LoadBalancerType"`
	AddressType                  string                   `json:"AddressType"`
	AddressIpVersion             string                   `json:"AddressIpVersion"`
	Ipv6AddressType              string                   `json:"Ipv6AddressType"`
	VpcId                        string                   `json:"VpcId"`
	RegionId                     string                   `json:"RegionId"`
	ZoneId                       string                   `json:"ZoneId"`
	DNSName                      string                   `json:"DNSName"`
	NetworkType                  string                   `json:"NetworkType"`
	InternetChargeType           string                   `json:"InternetChargeType"`
	LoadBalancerBillingConfig    map[string]interface{}   `json:"LoadBalancerBillingConfig"`
	ModificationProtectionConfig map[string]interface{}   `json:"ModificationProtectionConfig"`
	DeletionProtectionConfig     map[string]interface{}   `json:"DeletionProtectionConfig"`
	LoadBalancerOperationLocks   []map[string]interface{} `json:"LoadBalancerOperationLocks"`
	ZoneMappings                 []NlbZoneMapping         `json:"ZoneMappings"`
	CreateTime                   time.Time                `json:"CreateTime"`
	ResourceGroupId              string                   `json:"ResourceGroupId"`
	CpsEnabled                   bool                     `json:"CpsEnabled"`
	CrossZoneEnabled             bool                     `json:"CrossZoneEnabled"`
	TrafficAffinityEnabled       bool                     `json:"TrafficAffinityEnabled"`
	BandwidthPackageId           string                   `json:"BandwidthPackageId"`
}

type NlbZoneMapping struct {
	ZoneId                string                   `json:"ZoneId"`
	VSwitchId             string                   `json:"VSwitchId"`
	AllocationId          string                   `json:"AllocationId"`
	PrivateIPv4Address    string                   `json:"PrivateIPv4Address"`
	IPv6Address           string                   `json:"IPv6Address"`
	LoadBalancerAddresses []NlbLoadBalancerAddress `json:"LoadBalancerAddresses"`
}

type NlbLoadBalancerAddress struct {
	AllocationId       string `json:"AllocationId"`
	EipType            string `json:"EipType"`
	PrivateIPv4Address string `json:"PrivateIPv4Address"`
	PrivateIPv6Address string `json:"PrivateIPv6Address"`
	PublicIPv4Address  string `json:"PublicIPv4Address"`
	PublicIPv6Address  string `json:"PublicIPv6Address"`
}

func (nlb *SNlb) GetName() string {
	return nlb.LoadBalancerName
}

func (nlb *SNlb) GetId() string {
	return nlb.LoadBalancerId
}

func (nlb *SNlb) GetGlobalId() string {
	return nlb.LoadBalancerId
}

func (nlb *SNlb) GetStatus() string {
	switch nlb.LoadBalancerStatus {
	case "Active":
		return api.LB_STATUS_ENABLED
	case "Provisioning", "Configuring":
		return api.LB_STATUS_UNKNOWN
	case "Stopped":
		return api.LB_STATUS_DISABLED
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (nlb *SNlb) GetAddress() string {
	return nlb.DNSName
}

func (nlb *SNlb) GetAddressType() string {
	return nlb.AddressType
}

func (nlb *SNlb) GetNetworkType() string {
	return "vpc"
}

func (nlb *SNlb) GetNetworkIds() []string {
	ret := []string{}
	for _, zone := range nlb.ZoneMappings {
		if len(zone.VSwitchId) > 0 {
			ret = append(ret, zone.VSwitchId)
		}
	}
	return ret
}

func (nlb *SNlb) GetZoneId() string {
	if len(nlb.ZoneMappings) > 0 {
		zone, err := nlb.region.getZoneById(nlb.ZoneMappings[0].ZoneId)
		if err != nil {
			log.Errorf("failed to find zone for nlb %s error: %v", nlb.LoadBalancerName, err)
			return ""
		}
		return zone.GetGlobalId()
	}
	return ""
}

func (nlb *SNlb) GetZone1Id() string {
	if len(nlb.ZoneMappings) > 1 {
		zone, err := nlb.region.getZoneById(nlb.ZoneMappings[1].ZoneId)
		if err != nil {
			log.Errorf("failed to find zone for nlb %s error: %v", nlb.LoadBalancerName, err)
			return ""
		}
		return zone.GetGlobalId()
	}
	return ""
}

func (nlb *SNlb) IsEmulated() bool {
	return false
}

func (nlb *SNlb) GetVpcId() string {
	return nlb.VpcId
}

func (nlb *SNlb) Refresh() error {
	loadbalancer, err := nlb.region.GetNlbDetail(nlb.LoadBalancerId)
	if err != nil {
		return err
	}
	return jsonutils.Update(nlb, loadbalancer)
}

func (nlb *SNlb) Delete(ctx context.Context) error {
	return nlb.region.DeleteNlb(nlb.LoadBalancerId)
}

func (nlb *SNlb) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	groups, err := nlb.region.GetNlbServerGroups()
	if err != nil {
		return nil, err
	}
	igroups := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	for i := 0; i < len(groups); i++ {
		// 过滤出属于当前负载均衡器的服务器组
		for _, lbId := range groups[i].RelatedLoadBalancerIds {
			if lbId == nlb.LoadBalancerId {
				groups[i].nlb = nlb
				igroups = append(igroups, &groups[i])
				break
			}
		}
	}
	return igroups, nil
}

func (nlb *SNlb) CreateILoadBalancerBackendGroup(group *cloudprovider.SLoadbalancerBackendGroup) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	serverGroup, err := nlb.region.CreateNlbServerGroup(group, nlb.VpcId)
	if err != nil {
		return nil, err
	}
	serverGroup.nlb = nlb
	return serverGroup, nil
}

func (nlb *SNlb) GetILoadBalancerBackendGroupById(groupId string) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	groups, err := nlb.GetILoadBalancerBackendGroups()
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

func (nlb *SNlb) CreateILoadBalancerListener(ctx context.Context, listener *cloudprovider.SLoadbalancerListenerCreateOptions) (cloudprovider.ICloudLoadbalancerListener, error) {
	return nlb.region.CreateNlbListener(nlb, listener)
}

func (nlb *SNlb) GetILoadBalancerListenerById(listenerId string) (cloudprovider.ICloudLoadbalancerListener, error) {
	listeners, err := nlb.GetILoadBalancerListeners()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(listeners); i++ {
		if listeners[i].GetGlobalId() == listenerId {
			return listeners[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (nlb *SNlb) GetILoadBalancerListeners() ([]cloudprovider.ICloudLoadbalancerListener, error) {
	listeners, err := nlb.region.GetNlbListeners(nlb.LoadBalancerId)
	if err != nil {
		return nil, err
	}
	ilisteners := []cloudprovider.ICloudLoadbalancerListener{}
	for i := 0; i < len(listeners); i++ {
		listeners[i].nlb = nlb
		ilisteners = append(ilisteners, &listeners[i])
	}
	return ilisteners, nil
}

func (nlb *SNlb) GetChargeType() string {
	return api.LB_CHARGE_TYPE_BY_TRAFFIC
}

func (nlb *SNlb) GetCreatedAt() time.Time {
	return nlb.CreateTime
}

func (nlb *SNlb) GetEgressMbps() int {
	return 0
}

func (nlb *SNlb) GetIEIPs() ([]cloudprovider.ICloudEIP, error) {
	ret := []cloudprovider.ICloudEIP{}
	for _, zone := range nlb.ZoneMappings {
		for _, addr := range zone.LoadBalancerAddresses {
			if strings.HasPrefix(addr.AllocationId, "eip-") {
				eip, err := nlb.region.GetEip(addr.AllocationId)
				if err != nil {
					return nil, err
				}
				eip.region = nlb.region
				ret = append(ret, eip)
			}
		}
	}
	return ret, nil
}

func (nlb *SNlb) Start() error {
	return cloudprovider.ErrNotSupported
}

func (nlb *SNlb) Stop() error {
	return cloudprovider.ErrNotSupported
}

func (nlb *SNlb) GetLoadbalancerSpec() string {
	return nlb.LoadBalancerType
}

func (nlb *SNlb) GetProjectId() string {
	return nlb.ResourceGroupId
}

// region methods
func (region *SRegion) GetNlbs() ([]SNlb, error) {
	params := map[string]string{
		"RegionId":   region.RegionId,
		"MaxResults": "100",
	}

	nlbs := []SNlb{}
	nextToken := ""

	for {
		if nextToken != "" {
			params["NextToken"] = nextToken
		}

		body, err := region.nlbRequest("ListLoadBalancers", params)
		if err != nil {
			return nil, err
		}

		pageNlbs := []SNlb{}
		err = body.Unmarshal(&pageNlbs, "LoadBalancers")
		if err != nil {
			return nil, err
		}

		for i := 0; i < len(pageNlbs); i++ {
			pageNlbs[i].region = region
		}
		nlbs = append(nlbs, pageNlbs...)

		nextToken, _ = body.GetString("NextToken")
		if nextToken == "" {
			break
		}
	}

	return nlbs, nil
}

func (region *SRegion) GetNlbDetail(nlbId string) (*SNlb, error) {
	params := map[string]string{
		"RegionId":       region.RegionId,
		"LoadBalancerId": nlbId,
	}

	body, err := region.nlbRequest("GetLoadBalancerAttribute", params)
	if err != nil {
		return nil, err
	}

	nlb := &SNlb{region: region}
	err = body.Unmarshal(nlb)
	if err != nil {
		return nil, err
	}

	return nlb, nil
}

func (region *SRegion) DeleteNlb(nlbId string) error {
	params := map[string]string{
		"RegionId":       region.RegionId,
		"LoadBalancerId": nlbId,
	}

	_, err := region.nlbRequest("DeleteLoadBalancer", params)
	return err
}
