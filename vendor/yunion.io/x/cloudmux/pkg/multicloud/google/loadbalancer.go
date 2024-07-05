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

package google

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

// 全球负载均衡 https://cloud.google.com/compute/docs/reference/rest/v1/globalAddresses/list
// 区域负载均衡 https://cloud.google.com/compute/docs/reference/rest/v1/addresses
// https://cloud.google.com/compute/docs/reference/rest/v1/targetHttpProxies/get
// https://cloud.google.com/compute/docs/reference/rest/v1/targetGrpcProxies/get
// https://cloud.google.com/compute/docs/reference/rest/v1/targetHttpsProxies/get
// https://cloud.google.com/compute/docs/reference/rest/v1/targetSslProxies/get
// https://cloud.google.com/compute/docs/reference/rest/v1/targetTcpProxies/get

type SLoadbalancer struct {
	multicloud.SLoadbalancerBase
	SResourceBase
	region          *SRegion
	urlMap          *SUrlMap           // http & https LB
	backendServices []SBackendServices // tcp & udp LB. 或者 http & https 后端
	instanceGroups  []SInstanceGroup
	healthChecks    []HealthChecks

	forwardRules []SForwardingRule // 服务IP地址
	isHttpLb     bool              // 标记是否为http/https lb
}

func (self *SLoadbalancer) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SLoadbalancer) Refresh() error {
	lb, err := self.region.GetLoadbalancer(self.GetGlobalId())
	if err != nil {
		return errors.Wrap(err, "GetLoadbalancer")
	}

	err = jsonutils.Update(self, &lb)
	if err != nil {
		return errors.Wrap(err, "Refresh.Update")
	}

	self.healthChecks = nil
	self.instanceGroups = nil
	self.forwardRules = nil
	if self.isHttpLb {
		bss, err := self.GetBackendServices()
		if err != nil {
			return errors.Wrap(err, "GetForwardingRules")
		}
		self.backendServices = bss
	}
	return nil
}

func (self *SLoadbalancer) IsEmulated() bool {
	return true
}

func (self *SLoadbalancer) GetSysTags() map[string]string {
	frs, err := self.GetForwardingRules()
	if err != nil {
		return nil
	}

	ips := []string{}
	for i := range frs {
		if len(frs[i].IPAddress) > 0 && !utils.IsInStringArray(frs[i].IPAddress, ips) {
			ips = append(ips, frs[i].IPAddress)
		}
	}
	data := map[string]string{}
	data["FrontendIPs"] = strings.Join(ips, ",")
	return data
}

func (self *SLoadbalancer) GetTags() (map[string]string, error) {
	return map[string]string{}, nil
}

func (self *SLoadbalancer) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancer) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SLoadbalancer) GetProjectId() string {
	return self.region.GetProjectId()
}

/*
对应forwardingRules地址,存在多个前端IP的情况下，只展示按拉丁字母排序最前的一个地址。其他地址需要到详情中查看
External
Internal
InternalManaged
InternalSelfManaged
Invalid
UndefinedLoadBalancingScheme
*/
func (self *SLoadbalancer) GetAddress() string {
	frs, err := self.GetForwardingRules()
	if err != nil {
		log.Errorf("GetAddress.GetForwardingRules %s", err)
	}

	for i := range frs {
		sche := strings.ToLower(frs[i].LoadBalancingScheme)
		if !utils.IsInStringArray(sche, []string{"invalid", "undefinedloadbalancingscheme"}) {
			return frs[i].IPAddress
		}
	}
	return ""
}

func (self *SLoadbalancer) GetAddressType() string {
	frs, err := self.GetForwardingRules()
	if err != nil {
		return api.LB_ADDR_TYPE_INTRANET
	}

	for i := range frs {
		sche := strings.ToLower(frs[i].LoadBalancingScheme)
		if !utils.IsInStringArray(sche, []string{"invalid", "undefinedloadbalancingscheme"}) {
			if sche == "external" {
				return api.LB_ADDR_TYPE_INTERNET
			} else {
				return api.LB_ADDR_TYPE_INTRANET
			}
		}
	}

	return api.LB_ADDR_TYPE_INTERNET
}

func (self *SLoadbalancer) GetNetworkType() string {
	return api.LB_NETWORK_TYPE_VPC
}

func (self *SLoadbalancer) GetNetworkIds() []string {
	igs, err := self.GetInstanceGroups()
	if err != nil {
		log.Errorf("GetInstanceGroups %s", err)
		return nil
	}

	selfLinks := make([]string, 0)
	networkIds := make([]string, 0)
	for i := range igs {
		if utils.IsInStringArray(igs[i].Subnetwork, selfLinks) {
			selfLinks = append(selfLinks, igs[i].Subnetwork)
			network := SResourceBase{
				Name:     "",
				SelfLink: igs[i].Network,
			}
			networkIds = append(networkIds, network.GetGlobalId())
		}
	}

	return networkIds
}

func (self *SLoadbalancer) GetVpcId() string {
	networkIds := self.GetNetworkIds()
	if len(networkIds) == 0 {
		return ""
	}
	if len(networkIds) >= 1 {
		vpc, err := self.region.GetVpc(networkIds[0])
		if err == nil && vpc != nil {
			return vpc.GetGlobalId()
		}
	}
	return ""
}

func (self *SLoadbalancer) GetZoneId() string {
	igs, err := self.GetInstanceGroups()
	if err != nil {
		log.Errorf("GetInstanceGroups %s", err)
		return ""
	}

	for i := range igs {
		if len(igs[i].Zone) > 0 {
			zone := SResourceBase{
				Name:     "",
				SelfLink: igs[i].Zone,
			}

			return zone.GetGlobalId()
		}
	}

	return ""
}

func (self *SLoadbalancer) GetZone1Id() string {
	return ""
}

func (self *SLoadbalancer) GetLoadbalancerSpec() string {
	if self.isHttpLb {
		return "regional_http_lb"
	}

	return fmt.Sprintf("regional_%s", strings.ToLower(self.backendServices[0].Protocol))
}

func (self *SLoadbalancer) GetChargeType() string {
	return api.LB_CHARGE_TYPE_BY_TRAFFIC
}

func (self *SLoadbalancer) GetEgressMbps() int {
	return 0
}

func (self *SLoadbalancer) GetIEIP() (cloudprovider.ICloudEIP, error) {
	frs, err := self.GetForwardingRules()
	if err != nil {
		log.Errorf("GetAddress.GetForwardingRules %s", err)
	}

	for i := range frs {
		if strings.ToLower(frs[i].LoadBalancingScheme) == "external" {
			eips, err := self.region.GetEips(frs[i].IPAddress, 0, "")
			if err != nil {
				log.Errorf("GetEips %s", err)
			}

			if len(eips) > 0 {
				return &eips[0], nil
			}
		}
	}

	return nil, nil
}

func (self *SLoadbalancer) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SLoadbalancer) Start() error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancer) Stop() error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancer) GetILoadBalancerListeners() ([]cloudprovider.ICloudLoadbalancerListener, error) {
	lbls, err := self.GetLoadbalancerListeners()
	if err != nil {
		return nil, errors.Wrap(err, "GetLoadbalancerListeners")
	}

	ilbls := make([]cloudprovider.ICloudLoadbalancerListener, len(lbls))
	for i := range lbls {
		ilbls[i] = &lbls[i]
	}

	return ilbls, nil
}

func (self *SLoadbalancer) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	lbbgs, err := self.GetLoadbalancerBackendGroups()
	if err != nil {
		return nil, errors.Wrap(err, "GetLoadbalancerBackendGroups")
	}

	ilbbgs := make([]cloudprovider.ICloudLoadbalancerBackendGroup, len(lbbgs))
	for i := range lbbgs {
		ilbbgs[i] = &lbbgs[i]
	}

	return ilbbgs, nil
}

func (self *SLoadbalancer) CreateILoadBalancerBackendGroup(group *cloudprovider.SLoadbalancerBackendGroup) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SLoadbalancer) GetILoadBalancerBackendGroupById(groupId string) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	lbbgs, err := self.GetLoadbalancerBackendGroups()
	if err != nil {
		return nil, errors.Wrap(err, "GetLoadbalancerBackendGroups")
	}

	for i := range lbbgs {
		if lbbgs[i].GetGlobalId() == groupId {
			return &lbbgs[i], nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func (self *SLoadbalancer) CreateILoadBalancerListener(ctx context.Context, listener *cloudprovider.SLoadbalancerListenerCreateOptions) (cloudprovider.ICloudLoadbalancerListener, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SLoadbalancer) GetILoadBalancerListenerById(listenerId string) (cloudprovider.ICloudLoadbalancerListener, error) {
	lbls, err := self.GetLoadbalancerListeners()
	if err != nil {
		return nil, errors.Wrap(err, "GetLoadbalancerBackendGroups")
	}

	for i := range lbls {
		if lbls[i].GetGlobalId() == listenerId {
			return &lbls[i], nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

// GET https://compute.googleapis.com/compute/v1/projects/{project}/aggregated/targetHttpProxies 前端监听
//
//	tcp lb backend type: backend service
func (self *SRegion) GetRegionalTcpLoadbalancers() ([]SLoadbalancer, error) {
	bss, err := self.GetRegionalBackendServices("protocol eq TCP")
	if err != nil {
		return nil, errors.Wrap(err, "GetRegionalBackendServices")
	}

	lbs := make([]SLoadbalancer, len(bss))
	for i := range bss {
		lbs[i] = SLoadbalancer{
			region: self,
			SResourceBase: SResourceBase{
				Name:     bss[i].Name,
				SelfLink: bss[i].SelfLink,
			},
			urlMap:          nil,
			backendServices: []SBackendServices{bss[i]},
			forwardRules:    nil,
		}
	}

	return lbs, nil
}

// udp lb backend type: backend service
func (self *SRegion) GetRegionalUdpLoadbalancers() ([]SLoadbalancer, error) {
	bss, err := self.GetRegionalBackendServices("protocol eq UDP")
	if err != nil {
		return nil, errors.Wrap(err, "GetRegionalBackendServices")
	}

	lbs := make([]SLoadbalancer, len(bss))
	for i := range bss {
		lbs[i] = SLoadbalancer{
			region: self,
			SResourceBase: SResourceBase{
				Name:     bss[i].Name,
				SelfLink: bss[i].SelfLink,
			},
			urlMap:          nil,
			backendServices: []SBackendServices{bss[i]},
			forwardRules:    nil,
		}
	}

	return lbs, nil
}

// http&https lb: urlmaps
func (self *SRegion) GetRegionalHTTPLoadbalancers() ([]SLoadbalancer, error) {
	ums, err := self.GetRegionalUrlMaps("")
	if err != nil {
		return nil, errors.Wrap(err, "GetRegionalUrlMaps")
	}

	lbs := make([]SLoadbalancer, len(ums))
	for i := range ums {
		lbs[i] = SLoadbalancer{
			region: self,
			SResourceBase: SResourceBase{
				Name:     ums[i].Name,
				SelfLink: ums[i].SelfLink,
			},
			urlMap:          &ums[i],
			backendServices: nil,
			forwardRules:    nil,
			isHttpLb:        true,
		}
	}

	return lbs, nil
}

func (self *SRegion) GetRegionalLoadbalancers() ([]SLoadbalancer, error) {
	lbs := make([]SLoadbalancer, 0)
	funcs := []func() ([]SLoadbalancer, error){self.GetRegionalHTTPLoadbalancers, self.GetRegionalTcpLoadbalancers, self.GetRegionalUdpLoadbalancers}
	for i := range funcs {
		_lbs, err := funcs[i]()
		if err != nil {
			return nil, errors.Wrap(err, "GetRegionalLoadbalancers")
		}
		lbs = append(lbs, _lbs...)
	}

	return lbs, nil
}

func (self *SRegion) GetLoadbalancer(resourceId string) (SLoadbalancer, error) {
	lb := SLoadbalancer{}
	var err error
	if strings.Contains(resourceId, "/urlMaps/") {
		ret := SUrlMap{}
		err = self.GetBySelfId(resourceId, &ret)
		lb.isHttpLb = true
		lb.urlMap = &ret
		lb.SResourceBase = SResourceBase{
			Name:     ret.Name,
			SelfLink: ret.SelfLink,
		}
	} else {
		ret := SBackendServices{}
		err = self.GetBySelfId(resourceId, &ret)
		lb.backendServices = []SBackendServices{ret}
		lb.SResourceBase = SResourceBase{
			Name:     ret.Name,
			SelfLink: ret.SelfLink,
		}
	}

	if err != nil {
		return lb, errors.Wrapf(err, "get")
	}

	lb.region = self
	return lb, nil
}

func (self *SRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	lbs, err := self.GetRegionalLoadbalancers()
	if err != nil {
		return nil, errors.Wrap(err, "GetRegionalLoadbalancers")
	}
	ilbs := []cloudprovider.ICloudLoadbalancer{}
	for i := range lbs {
		ilbs = append(ilbs, &lbs[i])
	}
	return ilbs, nil
}

func (self *SRegion) GetILoadBalancerById(loadbalancerId string) (cloudprovider.ICloudLoadbalancer, error) {
	lb, err := self.GetLoadbalancer(loadbalancerId)
	if err != nil {
		return nil, errors.Wrap(err, "GetLoadbalancer")
	}
	return &lb, nil
}
