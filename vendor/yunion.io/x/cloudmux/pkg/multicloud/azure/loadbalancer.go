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

package azure

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SLoadbalancer struct {
	multicloud.SLoadbalancerBase
	AzureTags
	region *SRegion

	Name       string                   `json:"name"`
	ID         string                   `json:"id"`
	Etag       string                   `json:"etag"`
	Type       string                   `json:"type"`
	Location   string                   `json:"location"`
	Properties *SLoadbalancerProperties `json:"properties"`
	Sku        Sku                      `json:"sku"`
}

func (self *SLoadbalancer) GetId() string {
	return self.ID
}

func (self *SLoadbalancer) GetName() string {
	return self.Name
}

func (self *SLoadbalancer) GetGlobalId() string {
	return strings.ToLower(self.GetId())
}

func (self *SLoadbalancer) GetProperties() (*SLoadbalancerProperties, error) {
	if self.Properties != nil {
		return self.Properties, nil
	}
	lb, err := self.region.GetLoadbalancer(self.ID)
	if err != nil {
		return nil, err
	}
	self.Properties = lb.Properties
	return self.Properties, nil
}

func (self *SLoadbalancer) GetStatus() string {
	properties, err := self.GetProperties()
	if err != nil {
		return api.LB_STATUS_UNKNOWN
	}
	switch properties.ProvisioningState {
	case "Deleting":
		return api.LB_STATUS_DELETING
	case "Failed":
		return api.LB_STATUS_START_FAILED
	case "Updating":
		return api.LB_SYNC_CONF
	}

	switch self.Properties.OperationalState {
	case "Running":
		return api.LB_STATUS_ENABLED
	case "Stopped":
		return api.LB_STATUS_DISABLED
	case "Starting", "Stopping":
		return api.LB_SYNC_CONF
	default:
		if self.Properties.ProvisioningState == "Succeeded" {
			return api.LB_STATUS_ENABLED
		}

		return api.LB_STATUS_UNKNOWN
	}
}

func (self *SLoadbalancer) Refresh() error {
	lb, err := self.region.GetLoadbalancer(self.ID)
	if err != nil {
		return errors.Wrap(err, "GetLoadbalancer")
	}
	return jsonutils.Update(self, lb)
}

func (self *SLoadbalancer) GetProjectId() string {
	return getResourceGroup(self.ID)
}

func (self *SLoadbalancer) GetAddress() string {
	properties, err := self.GetProperties()
	if err != nil {
		return ""
	}
	for _, front := range properties.FrontendIPConfigurations {
		if len(front.Properties.PrivateIPAddress) > 0 {
			return front.Properties.PrivateIPAddress
		}
	}
	return ""
}

func (self *SLoadbalancer) GetAddressType() string {
	netIds := self.GetNetworkIds()
	if len(netIds) > 0 {
		return api.LB_ADDR_TYPE_INTRANET
	}
	return api.LB_ADDR_TYPE_INTERNET
}

func (self *SLoadbalancer) GetNetworkType() string {
	return api.LB_NETWORK_TYPE_VPC
}

func (self *SLoadbalancer) GetNetworkIds() []string {
	properties, err := self.GetProperties()
	if err != nil {
		return []string{}
	}
	for _, front := range properties.FrontendIPConfigurations {
		if len(front.Properties.Subnet.ID) > 0 {
			return []string{strings.ToLower(front.Properties.Subnet.ID)}
		}
	}
	for _, gateway := range properties.GatewayIPConfigurations {
		if len(gateway.Properties.Subnet.ID) > 0 {
			return []string{strings.ToLower(gateway.Properties.Subnet.ID)}
		}
	}
	return []string{}
}

func (self *SLoadbalancer) GetVpcId() string {
	ids := self.GetNetworkIds()
	for _, netId := range ids {
		if strings.Contains(netId, "/subnets") {
			return strings.Split(netId, "/subnets")[0]
		}
	}
	return ""
}

func (self *SLoadbalancer) GetZoneId() string {
	return ""
}

func (self *SLoadbalancer) GetZone1Id() string {
	return ""
}

func (self *SLoadbalancer) GetLoadbalancerSpec() string {
	if len(self.Sku.Name) > 0 {
		return self.Sku.Name
	}
	properties, err := self.GetProperties()
	if err != nil {
		return ""
	}
	return properties.Sku.Name
}

func (self *SLoadbalancer) GetChargeType() string {
	return api.LB_CHARGE_TYPE_BY_TRAFFIC
}

func (self *SLoadbalancer) GetEgressMbps() int {
	return 0
}

func (self *SLoadbalancer) GetIEIP() (cloudprovider.ICloudEIP, error) {
	properties, err := self.GetProperties()
	if err != nil {
		return nil, errors.Wrapf(err, "GetProperties")
	}
	for _, front := range properties.FrontendIPConfigurations {
		if len(front.Properties.PublicIPAddress.ID) > 0 {
			eip, err := self.region.GetEip(front.Properties.PublicIPAddress.ID)
			if err != nil {
				return nil, err
			}
			return eip, nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SLoadbalancer) Delete(ctx context.Context) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Delete")
}

func (self *SLoadbalancer) Start() error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Start")
}

func (self *SLoadbalancer) Stop() error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Stop")
}

func (self *SLoadbalancer) GetILoadBalancerListeners() ([]cloudprovider.ICloudLoadbalancerListener, error) {
	properties, err := self.GetProperties()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudLoadbalancerListener{}
	idMap := map[string]bool{}

	for _, listeners := range [][]SLoadBalancerListener{
		properties.InboundNatRules,
		properties.OutboundRules,
		properties.LoadBalancingRules,
	} {
		for i := range listeners {
			listener := listeners[i]

			listener.lb = self
			if _, ok := idMap[listener.GetGlobalId()]; !ok {
				ret = append(ret, &listener)
				idMap[listener.GetGlobalId()] = true
			}
		}
	}

	for i := range properties.HTTPListeners {
		listener := properties.HTTPListeners[i]

		listener.lb = self
		if _, ok := idMap[listener.GetGlobalId()]; !ok {
			ret = append(ret, &listener)
			idMap[listener.GetGlobalId()] = true
		}
	}

	return ret, nil
}

func (self *SLoadbalancer) CreateILoadBalancerBackendGroup(group *cloudprovider.SLoadbalancerBackendGroup) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotImplemented, "CreateILoadBalancerBackendGroup")
}

func (self *SLoadbalancer) CreateILoadBalancerListener(ctx context.Context, listener *cloudprovider.SLoadbalancerListenerCreateOptions) (cloudprovider.ICloudLoadbalancerListener, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotImplemented, "CreateILoadBalancerListener")
}

func (self *SLoadbalancer) GetILoadBalancerListenerById(id string) (cloudprovider.ICloudLoadbalancerListener, error) {
	lblis, err := self.GetILoadBalancerListeners()
	if err != nil {
		return nil, err
	}
	for i := range lblis {
		if lblis[i].GetGlobalId() == id {
			return lblis[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SLoadbalancer) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SLoadbalancer) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SLoadbalancer) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	ret := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	properties, err := self.GetProperties()
	if err != nil {
		return nil, err
	}
	idMap := map[string]bool{}
	for _, rule := range properties.InboundNatRules {
		if len(rule.Properties.BackendIPConfiguration.Id) == 0 {
			continue
		}
		lbbg := &SLoadbalancerBackendGroup{lb: self, Id: rule.Properties.BackendIPConfiguration.Id}
		if _, ok := idMap[lbbg.GetGlobalId()]; !ok {
			ret = append(ret, lbbg)
			idMap[lbbg.GetGlobalId()] = true
		}
	}

	for _, pool := range properties.BackendAddressPools {
		lbbg := &SLoadbalancerBackendGroup{lb: self, Id: pool.Id, BackendIPConfigurations: pool.Properties.BackendIPConfigurations}
		if _, ok := idMap[lbbg.GetGlobalId()]; !ok {
			ret = append(ret, lbbg)
			idMap[lbbg.GetGlobalId()] = true
		}
	}

	return ret, nil
}

func (self *SLoadbalancer) GetILoadBalancerBackendGroupById(groupId string) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	lbbgs, err := self.GetILoadBalancerBackendGroups()
	if err != nil {
		return nil, err
	}
	for i := range lbbgs {
		if lbbgs[i].GetGlobalId() == groupId {
			return lbbgs[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, groupId)
}

func (self *SRegion) GetLoadbalancers() ([]SLoadbalancer, error) {
	params := url.Values{}
	params.Add("$filter", fmt.Sprintf("location eq '%s'", self.Name))
	params.Add("$filter", "(resourceType eq 'Microsoft.Network/applicationGateways' or resourceType eq 'Microsoft.Network/loadBalancers')")
	resp, err := self.client.list_v2("resources", "2024-03-01", params)
	if err != nil {
		return nil, err
	}
	ret := []SLoadbalancer{}
	err = resp.Unmarshal(&ret, "value")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) GetLoadbalancer(id string) (*SLoadbalancer, error) {
	resp, err := self.show(id, "2023-11-01")
	if err != nil {
		return nil, err
	}
	ret := &SLoadbalancer{region: self}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
