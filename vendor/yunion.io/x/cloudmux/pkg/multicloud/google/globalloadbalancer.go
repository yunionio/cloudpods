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

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SGlobalLoadbalancer struct {
	multicloud.SLoadbalancerBase
	GoogleTags
	SResourceBase
	region          *SGlobalRegion
	urlMap          *SUrlMap           // http & https LB
	backendServices []SBackendServices // tcp & udp LB. 或者 http & https 后端
	instanceGroups  []SGlobalInstanceGroup
	healthChecks    []HealthChecks

	forwardRules []SForwardingRule // 服务IP地址
	isHttpLb     bool              // 标记是否为http/https lb
}

func (self *SGlobalLoadbalancer) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SGlobalLoadbalancer) Refresh() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SGlobalLoadbalancer) GetNetworkType() string {
	return api.LB_NETWORK_TYPE_VPC
}

func (self *SGlobalLoadbalancer) GetVpcId() string {
	return ""
}

func (self *SGlobalLoadbalancer) GetZoneId() string {
	return ""
}

func (self *SGlobalLoadbalancer) GetZone1Id() string {
	return ""
}

func (self *SGlobalLoadbalancer) GetChargeType() string {
	return api.LB_CHARGE_TYPE_BY_TRAFFIC
}

func (self *SGlobalLoadbalancer) GetEgressMbps() int {
	return 0
}

func (self *SGlobalLoadbalancer) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SGlobalLoadbalancer) Start() error {
	return cloudprovider.ErrNotSupported
}

func (self *SGlobalLoadbalancer) Stop() error {
	return cloudprovider.ErrNotSupported
}

func (self *SGlobalLoadbalancer) GetSysTags() map[string]string {
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

func (self *SGlobalLoadbalancer) GetProjectId() string {
	return ""
}

func (self *SGlobalLoadbalancer) GetIEIP() (cloudprovider.ICloudEIP, error) {
	frs, err := self.GetForwardingRules()
	if err != nil {
		log.Errorf("GetAddress.GetForwardingRules %s", err)
	}

	for i := range frs {
		if strings.ToLower(frs[i].LoadBalancingScheme) == "external" {
			eips, err := self.region.GetEips(frs[i].IPAddress)
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

func (self *SGlobalLoadbalancer) GetAddress() string {
	var ret []SForwardingRule
	if err := self.region.getGlobalLoadbalancerComponents("forwardingRules", "", &ret); err != nil {
		return ""
	}

	for _, item := range ret {
		targetResponse, err := _jsonRequest(self.region.client.client, "GET", item.Target, nil, false)
		if err != nil {
			return ""
		}

		var target interface{}
		switch {
		case strings.Contains(item.Target, "targetHttpProxies"):
			target = new(STargetHttpProxy)
		case strings.Contains(item.Target, "targetHttpsProxies"):
			target = new(STargetHttpsProxy)
		case strings.Contains(item.Target, "targetTcpProxies"):
			target = new(STargetTcpProxy)
		default:
			continue
		}

		if err := targetResponse.Unmarshal(target); err != nil {
			return ""
		}

		match, err := self.isNameMatch(target)
		if err != nil {
			log.Errorf("isNameMatch error: %v", err)
			continue
		}
		if match {
			return item.IPAddress
		}
	}

	return ""
}

func (self *SGlobalLoadbalancer) GetAddressType() string {
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

func (self *SGlobalLoadbalancer) GetNetworkIds() []string {
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

func (self *SGlobalLoadbalancer) GetLoadbalancerSpec() string {
	if self.isHttpLb {
		return "global_http_lb"
	}

	return fmt.Sprintf("global_%s", strings.ToLower(self.backendServices[0].Protocol))
}

func (self *SGlobalLoadbalancer) GetILoadBalancerListeners() ([]cloudprovider.ICloudLoadbalancerListener, error) {
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

func (self *SGlobalLoadbalancer) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
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

func (self *SGlobalLoadbalancer) CreateILoadBalancerBackendGroup(group *cloudprovider.SLoadbalancerBackendGroup) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SGlobalLoadbalancer) GetILoadBalancerBackendGroupById(groupId string) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
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

func (self *SGlobalLoadbalancer) CreateILoadBalancerListener(ctx context.Context, listener *cloudprovider.SLoadbalancerListenerCreateOptions) (cloudprovider.ICloudLoadbalancerListener, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SGlobalLoadbalancer) GetILoadBalancerListenerById(listenerId string) (cloudprovider.ICloudLoadbalancerListener, error) {
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

func (self *SGlobalRegion) getGlobalLoadbalancerComponents(resource string, filter string, result interface{}) error {
	url := fmt.Sprintf("global/%s", resource)
	params := map[string]string{}
	if len(filter) > 0 {
		params["filter"] = filter
	}

	err := self.ListAll(url, params, result)
	if err != nil {
		return errors.Wrap(err, "ListAll")
	}

	return nil
}

func (self *SGlobalRegion) GetGlobalHTTPLoadbalancers() ([]SGlobalLoadbalancer, error) {
	ums, err := self.GetGlobalUrlMaps("")
	if err != nil {
		return nil, errors.Wrap(err, "GetGlobalUrlMaps")
	}

	lbs := make([]SGlobalLoadbalancer, len(ums))
	for i := range ums {
		lbs[i] = SGlobalLoadbalancer{
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

func (self *SGlobalRegion) GetGlobalTcpLoadbalancers() ([]SGlobalLoadbalancer, error) {
	bss, err := self.GetGlobalBackendServices("protocol eq TCP")
	if err != nil {
		return nil, errors.Wrap(err, "GetGlobalBackendServices")
	}

	lbs := make([]SGlobalLoadbalancer, len(bss))
	for i := range bss {
		lbs[i] = SGlobalLoadbalancer{
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

func (self *SGlobalRegion) GetGlobalUdpLoadbalancers() ([]SGlobalLoadbalancer, error) {
	bss, err := self.GetGlobalBackendServices("protocol eq UDP")
	if err != nil {
		return nil, errors.Wrap(err, "GetGlobalBackendServices")
	}

	lbs := make([]SGlobalLoadbalancer, len(bss))
	for i := range bss {
		lbs[i] = SGlobalLoadbalancer{
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

func (self *SGlobalRegion) GetGlobalLoadbalancers() ([]SGlobalLoadbalancer, error) {
	lbs := make([]SGlobalLoadbalancer, 0)
	funcs := []func() ([]SGlobalLoadbalancer, error){self.GetGlobalHTTPLoadbalancers, self.GetGlobalTcpLoadbalancers, self.GetGlobalUdpLoadbalancers}
	for i := range funcs {
		_lbs, err := funcs[i]()
		if err != nil {
			return nil, errors.Wrap(err, "GetGlobalLoadbalancers")
		}
		lbs = append(lbs, _lbs...)
	}

	return lbs, nil
}

func (self *SGlobalRegion) GetLoadbalancer(resourceId string) (SGlobalLoadbalancer, error) {
	lb := SGlobalLoadbalancer{}
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

func (self *SGlobalRegion) GetILoadBalancerById(loadbalancerId string) (cloudprovider.ICloudLoadbalancer, error) {
	lb, err := self.GetLoadbalancer(loadbalancerId)
	if err != nil {
		return nil, errors.Wrap(err, "GetLoadbalancer")
	}
	return &lb, nil
}
