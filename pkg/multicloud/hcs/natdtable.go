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

package hcs

import (
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SNatDEntry struct {
	multicloud.SResourceBase
	multicloud.HcsTags
	gateway *SNatGateway

	Id           string `json:"id"`
	NatGatewayId string `json:"nat_gateway_id"`
	Protocol     string `json:"protocol"`
	Status       string `json:"status"`
	ExternalIP   string `json:"floating_ip_address"`
	ExternalPort int    `json:"external_service_port"`
	InternalIP   string `json:"private_ip"`
	InternalPort int    `json:"internal_service_port"`
	PortId       string `json:"port_id"`
	AdminStateUp bool   `json:"admin_state_up"`
}

func (nat *SNatDEntry) GetId() string {
	return nat.Id
}

func (nat *SNatDEntry) GetName() string {
	// No name so return id
	return nat.GetId()
}

func (nat *SNatDEntry) GetGlobalId() string {
	return nat.GetId()
}

func (nat *SNatDEntry) GetStatus() string {
	return NatResouceStatusTransfer(nat.Status)
}

func (nat *SNatDEntry) GetIpProtocol() string {
	return nat.Protocol
}

func (nat *SNatDEntry) GetExternalIp() string {
	return nat.ExternalIP
}

func (nat *SNatDEntry) GetExternalPort() int {
	return nat.ExternalPort
}

func (nat *SNatDEntry) GetInternalIp() string {
	return nat.InternalIP
}

func (nat *SNatDEntry) GetInternalPort() int {
	return nat.InternalPort
}

func (nat *SNatDEntry) Delete() error {
	return nat.gateway.region.DeleteNatDEntry(nat.GetId())
}

// getNatSTable return all snat rules of gateway
func (gateway *SNatGateway) getNatDTable() ([]SNatDEntry, error) {
	ret, err := gateway.region.GetNatDTable(gateway.GetId())
	if err != nil {
		return nil, err
	}
	for i := range ret {
		ret[i].gateway = gateway
	}
	return ret, nil
}

func (self *SRegion) GetNatDTable(natGatewayId string) ([]SNatDEntry, error) {
	query := url.Values{}
	query.Set("nat_gateway_id", natGatewayId)
	ret := []SNatDEntry{}
	err := self.list("nat", "v2.0", "dnat_rules", query, &ret)
	if err != nil {
		return nil, err
	}
	for i := range ret {
		nat := &ret[i]
		if len(nat.InternalIP) == 0 {
			port, err := self.GetPort(nat.PortId)
			if err != nil {
				return nil, errors.Wrapf(err, `get port info for transfer to ip of port_id %q error`, nat.PortId)
			}
			nat.InternalIP = port.FixedIps[0].IpAddress
		}
	}
	return ret, nil
}

func (self *SRegion) DeleteNatDEntry(entryId string) error {
	return self.delete("nat", "v2.0", "dnat_rules/"+entryId)
}

func (nat *SNatDEntry) Refresh() error {
	ret, err := nat.gateway.region.GetNatDEntry(nat.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(nat, ret)
}

func (self *SRegion) CreateNatDEntry(rule cloudprovider.SNatDRule, gatewayId string) (*SNatDEntry, error) {
	params := make(map[string]interface{})
	params["nat_gateway_id"] = gatewayId
	params["private_ip"] = rule.InternalIP
	params["internal_service_port"] = rule.InternalPort
	params["floating_ip_id"] = rule.ExternalIPID
	params["external_service_port"] = rule.ExternalPort
	params["protocol"] = rule.Protocol

	packParams := map[string]interface{}{
		"dnat_rule": params,
	}
	ret := &SNatDEntry{}
	return ret, self.create("nat", "v2.0", "dnat_rules", packParams, ret)
}

func (self *SRegion) GetNatDEntry(id string) (*SNatDEntry, error) {
	ret := &SNatDEntry{}
	return ret, self.get("nat", "v2.0", "dnat_rules/"+id, ret)
}
