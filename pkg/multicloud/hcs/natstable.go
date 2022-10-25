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

type SNatSEntry struct {
	multicloud.SResourceBase
	multicloud.HuaweiTags
	gateway *SNatGateway

	Id           string `json:"id"`
	NatGatewayId string `json:"nat_gateway_id"`
	NetworkId    string `json:"network_id"`
	SourceCIDR   string `json:"cidr"`
	Status       string `json:"status"`
	SNatIP       string `json:"floating_ip_address"`
	AdminStateUp bool   `json:"admin_state_up"`
}

func (nat *SNatSEntry) GetId() string {
	return nat.Id
}

func (nat *SNatSEntry) GetName() string {
	// Snat rule has no name in Huawei Cloud, so return Id
	return nat.GetId()
}

func (nat *SNatSEntry) GetGlobalId() string {
	return nat.GetId()
}

func (nat *SNatSEntry) GetStatus() string {
	return NatResouceStatusTransfer(nat.Status)
}

func (nat *SNatSEntry) GetIP() string {
	return nat.SNatIP
}

func (nat *SNatSEntry) GetSourceCIDR() string {
	return nat.SourceCIDR
}

func (nat *SNatSEntry) GetNetworkId() string {
	return nat.NetworkId
}

func (nat *SNatSEntry) Delete() error {
	return nat.gateway.region.DeleteNatSEntry(nat.GetId())
}

// getNatSTable return all snat rules of gateway
func (gateway *SNatGateway) getNatSTable() ([]SNatSEntry, error) {
	ret, err := gateway.region.GetNatSTable(gateway.GetId())
	if err != nil {
		return nil, err
	}
	for i := range ret {
		ret[i].gateway = gateway
	}
	return ret, nil
}

func (self *SRegion) GetNatSTable(natGatewayId string) ([]SNatSEntry, error) {
	query := url.Values{}
	query.Set("nat_gateway_id", natGatewayId)
	ret := []SNatSEntry{}
	err := self.list("nat", "v2.0", "snat_rules", query, &ret)
	if err != nil {
		return nil, err
	}
	for i := range ret {
		nat := &ret[i]
		if len(nat.SourceCIDR) != 0 {
			continue
		}
		net, err := self.GetNetwork(nat.NetworkId)
		if err != nil {
			return nil, errors.Wrapf(err, "GetNetwork(%s)", nat.NetworkId)
		}
		nat.SourceCIDR = net.CIDR
	}
	return ret, nil
}

func (self *SRegion) DeleteNatSEntry(entryId string) error {
	return self.delete("nat", "v2.0", "snat_rules/"+entryId)
}

func (nat *SNatSEntry) Refresh() error {
	ret, err := nat.gateway.region.GetNatSEntry(nat.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(nat, ret)
}

func (self *SRegion) CreateNatSEntry(rule cloudprovider.SNatSRule, gatewayId string) (*SNatSEntry, error) {
	params := make(map[string]interface{})
	params["nat_gateway_id"] = gatewayId
	if len(rule.NetworkID) != 0 {
		params["network_id"] = rule.NetworkID
	}
	if len(rule.SourceCIDR) != 0 {
		params["cidr"] = rule.SourceCIDR
	}
	params["floating_ip_id"] = rule.ExternalIPID

	packParams := map[string]interface{}{
		"snat_rule": params,
	}

	ret := &SNatSEntry{}
	return ret, self.create("nat", "v2.0", "snat_rules", packParams, ret)
}

func (self *SRegion) GetNatSEntry(id string) (*SNatSEntry, error) {
	ret := &SNatSEntry{}
	return ret, self.get("nat", "v2.0", "snat_rules/"+id, ret)
}
