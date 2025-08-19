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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SAlb struct {
	multicloud.SLoadbalancerBase
	AliyunTags
	region *SRegion

	LoadBalancerId               string                   `json:"LoadBalancerId"`
	LoadBalancerName             string                   `json:"LoadBalancerName"`
	LoadBalancerStatus           string                   `json:"LoadBalancerStatus"`
	LoadBalancerBizStatus        string                   `json:"LoadBalancerBizStatus"`
	LoadBalancerEdition          string                   `json:"LoadBalancerEdition"`
	LoadBalancerPayType          string                   `json:"LoadBalancerPayType"`
	AddressType                  string                   `json:"AddressType"`
	AddressAllocatedMode         string                   `json:"AddressAllocatedMode"`
	Ipv6AddressType              string                   `json:"Ipv6AddressType"`
	DNSName                      string                   `json:"DNSName"`
	VpcId                        string                   `json:"VpcId"`
	LoadBalancerBussinessStatus  string                   `json:"LoadBalancerBussinessStatus"`
	CreateTime                   time.Time                `json:"CreateTime"`
	ResourceGroupId              string                   `json:"ResourceGroupId"`
	CanBeDeleted                 bool                     `json:"CanBeDeleted"`
	ModificationProtectionConfig map[string]interface{}   `json:"ModificationProtectionConfig"`
	DeletionProtectionConfig     map[string]interface{}   `json:"DeletionProtectionConfig"`
	AccessLogConfig              map[string]interface{}   `json:"AccessLogConfig"`
	LoadBalancerOperationLocks   []map[string]interface{} `json:"LoadBalancerOperationLocks"`
	ZoneMappings                 []ZoneMapping            `json:"ZoneMappings"`
	RegionId                     string                   `json:"RegionId"`
}

type ZoneMapping struct {
	ZoneId                string                `json:"ZoneId"`
	VSwitchId             string                `json:"VSwitchId"`
	AllocationId          string                `json:"AllocationId"`
	EipType               string                `json:"EipType"`
	LoadBalancerAddresses []LoadBalancerAddress `json:"LoadBalancerAddresses"`
}

type LoadBalancerAddress struct {
	Address         string `json:"Address"`
	AddressType     string `json:"AddressType"`
	AllocationId    string `json:"AllocationId"`
	EipType         string `json:"EipType"`
	IntranetAddress string `json:"IntranetAddress"`
	InternetAddress string `json:"InternetAddress"`
	Ipv6Address     string `json:"Ipv6Address"`
}

func (alb *SAlb) GetName() string {
	return alb.LoadBalancerName
}

func (alb *SAlb) GetId() string {
	return alb.LoadBalancerId
}

func (alb *SAlb) GetGlobalId() string {
	return alb.LoadBalancerId
}

func (alb *SAlb) GetStatus() string {
	switch alb.LoadBalancerStatus {
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

func (alb *SAlb) GetAddress() string {
	return alb.DNSName
}

func (alb *SAlb) GetAddressType() string {
	return alb.AddressType
}

func (alb *SAlb) GetNetworkType() string {
	return "vpc"
}

func (alb *SAlb) GetNetworkIds() []string {
	ret := []string{}
	for _, zone := range alb.ZoneMappings {
		if len(zone.VSwitchId) > 0 {
			ret = append(ret, zone.VSwitchId)
		}
	}
	return ret
}

func (alb *SAlb) GetZoneId() string {
	if len(alb.ZoneMappings) > 0 {
		zone, err := alb.region.getZoneById(alb.ZoneMappings[0].ZoneId)
		if err != nil {
			log.Errorf("failed to find zone for alb %s error: %v", alb.LoadBalancerName, err)
			return ""
		}
		return zone.GetGlobalId()
	}
	return ""
}

func (alb *SAlb) GetZone1Id() string {
	if len(alb.ZoneMappings) > 1 {
		zone, err := alb.region.getZoneById(alb.ZoneMappings[1].ZoneId)
		if err != nil {
			log.Errorf("failed to find zone for alb %s error: %v", alb.LoadBalancerName, err)
			return ""
		}
		return zone.GetGlobalId()
	}
	return ""
}

func (alb *SAlb) IsEmulated() bool {
	return false
}

func (alb *SAlb) GetVpcId() string {
	return alb.VpcId
}

func (alb *SAlb) Refresh() error {
	loadbalancer, err := alb.region.GetAlbDetail(alb.LoadBalancerId)
	if err != nil {
		return err
	}
	return jsonutils.Update(alb, loadbalancer)
}

func (alb *SAlb) Delete(ctx context.Context) error {
	return alb.region.DeleteAlb(alb.LoadBalancerId)
}

func (alb *SAlb) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	groups, err := alb.region.GetAlbServerGroups()
	if err != nil {
		return nil, err
	}
	igroups := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	for i := 0; i < len(groups); i++ {
		// 过滤出属于当前负载均衡器的服务器组
		for _, lbId := range groups[i].RelatedLoadBalancerIds {
			if lbId == alb.LoadBalancerId {
				groups[i].alb = alb
				igroups = append(igroups, &groups[i])
				break
			}
		}
	}
	return igroups, nil
}

func (alb *SAlb) CreateILoadBalancerBackendGroup(group *cloudprovider.SLoadbalancerBackendGroup) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	serverGroup, err := alb.region.CreateAlbServerGroup(group, alb.VpcId)
	if err != nil {
		return nil, err
	}
	serverGroup.alb = alb
	return serverGroup, nil
}

func (alb *SAlb) GetILoadBalancerBackendGroupById(groupId string) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	groups, err := alb.GetILoadBalancerBackendGroups()
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

func (alb *SAlb) CreateILoadBalancerListener(ctx context.Context, listener *cloudprovider.SLoadbalancerListenerCreateOptions) (cloudprovider.ICloudLoadbalancerListener, error) {
	return alb.region.CreateAlbListener(alb, listener)
}

func (alb *SAlb) GetILoadBalancerListenerById(listenerId string) (cloudprovider.ICloudLoadbalancerListener, error) {
	listeners, err := alb.GetILoadBalancerListeners()
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

func (alb *SAlb) GetILoadBalancerListeners() ([]cloudprovider.ICloudLoadbalancerListener, error) {
	listeners, err := alb.region.GetAlbListeners(alb.LoadBalancerId)
	if err != nil {
		return nil, err
	}
	ilisteners := []cloudprovider.ICloudLoadbalancerListener{}
	for i := 0; i < len(listeners); i++ {
		listeners[i].alb = alb
		ilisteners = append(ilisteners, &listeners[i])
	}
	return ilisteners, nil
}

func (alb *SAlb) GetChargeType() string {
	return api.LB_CHARGE_TYPE_BY_TRAFFIC
}

func (alb *SAlb) GetCreatedAt() time.Time {
	return alb.CreateTime
}

func (alb *SAlb) GetEgressMbps() int {
	return 0
}

func (alb *SAlb) GetIEIPs() ([]cloudprovider.ICloudEIP, error) {
	ret := []cloudprovider.ICloudEIP{}
	info, err := alb.region.GetAlbDetail(alb.LoadBalancerId)
	if err != nil {
		return nil, err
	}
	for _, zone := range info.ZoneMappings {
		for _, addr := range zone.LoadBalancerAddresses {
			if len(addr.Address) > 0 {
				eips, err := alb.region.GetEips("", "", addr.Address)
				if err != nil {
					return nil, err
				}
				for i := range eips {
					eips[i].region = alb.region
					ret = append(ret, &eips[i])
				}
			}
		}
	}
	return ret, nil
}

func (alb *SAlb) Start() error {
	return cloudprovider.ErrNotSupported
}

func (alb *SAlb) Stop() error {
	return cloudprovider.ErrNotSupported
}

func (alb *SAlb) GetLoadbalancerSpec() string {
	return alb.LoadBalancerEdition
}

func (alb *SAlb) GetProjectId() string {
	return alb.ResourceGroupId
}

// region methods
func (region *SRegion) GetAlbs() ([]SAlb, error) {
	params := map[string]string{
		"RegionId":   region.RegionId,
		"MaxResults": "100",
	}

	albs := []SAlb{}
	nextToken := ""

	for {
		if nextToken != "" {
			params["NextToken"] = nextToken
		}

		body, err := region.albRequest("ListLoadBalancers", params)
		if err != nil {
			return nil, err
		}

		pageAlbs := []SAlb{}
		err = body.Unmarshal(&pageAlbs, "LoadBalancers")
		if err != nil {
			return nil, err
		}

		for i := 0; i < len(pageAlbs); i++ {
			pageAlbs[i].region = region
		}
		albs = append(albs, pageAlbs...)

		nextToken, _ = body.GetString("NextToken")
		if nextToken == "" {
			break
		}
	}

	return albs, nil
}

func (region *SRegion) GetAlbDetail(albId string) (*SAlb, error) {
	params := map[string]string{
		"RegionId":       region.RegionId,
		"LoadBalancerId": albId,
	}

	body, err := region.albRequest("GetLoadBalancerAttribute", params)
	if err != nil {
		return nil, err
	}

	alb := &SAlb{region: region}
	err = body.Unmarshal(alb)
	if err != nil {
		return nil, err
	}

	return alb, nil
}

func (region *SRegion) DeleteAlb(albId string) error {
	params := map[string]string{
		"RegionId":       region.RegionId,
		"LoadBalancerId": albId,
	}

	_, err := region.albRequest("DeleteLoadBalancer", params)
	return err
}
