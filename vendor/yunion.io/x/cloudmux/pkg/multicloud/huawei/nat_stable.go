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

package huawei

import (
	"fmt"
	"net/url"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SNatSTable struct {
	multicloud.SResourceBase
	multicloud.STagBase

	nat               *SNatgateway
	Id                string
	Cidr              string
	SourceType        int
	FloatingIpId      string
	Description       string
	Status            string
	CreatedAt         string
	NetworkId         string
	AdminStateUp      bool
	FloatingIpAddress string
	FreezedIpAddress  string
	GlobalEipId       string
	GlobalEipAddress  string
}

func (nat *SNatSTable) GetId() string {
	return nat.Id
}

func (nat *SNatSTable) GetName() string {
	return nat.GetId()
}

func (nat *SNatSTable) GetGlobalId() string {
	return nat.GetId()
}

func (nat *SNatSTable) GetStatus() string {
	return NatResouceStatusTransfer(nat.Status)
}

func (nat *SNatSTable) GetIP() string {
	return nat.FloatingIpAddress
}

func (nat *SNatSTable) Refresh() error {
	table, err := nat.nat.region.GetNatSEntry(nat.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(nat, table)
}

func (nat *SNatSTable) GetSourceCIDR() string {
	return nat.Cidr
}

func (nat *SNatSTable) GetNetworkId() string {
	return nat.NetworkId
}

func (nat *SNatSTable) Delete() error {
	return nat.nat.region.DeleteNatSTable(nat.nat.Id, nat.GetId())
}

func (region *SRegion) DeleteNatSTable(natId, id string) error {
	res := fmt.Sprintf("nat_gateways/%s/snat_rules/%s", natId, id)
	_, err := region.delete(SERVICE_NAT_V2, res)
	return err
}

func (region *SRegion) GetNatSEntry(id string) (*SNatSTable, error) {
	resp, err := region.list(SERVICE_NAT_V2, "snat_rules/"+id, nil)
	if err != nil {
		return nil, err
	}
	ret := &SNatSTable{}
	err = resp.Unmarshal(ret, "snat_rule")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (region *SRegion) GetPublicNatSTable(natId string) ([]SNatSTable, error) {
	query := url.Values{}
	if len(natId) > 0 {
		query.Set("nat_gateway_id", natId)
	}
	ret := []SNatSTable{}
	for {
		resp, err := region.list(SERVICE_NAT_V2, "snat_rules", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			SnatRules []SNatSTable
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.SnatRules...)
		if len(part.SnatRules) == 0 {
			break
		}
		query.Set("marker", part.SnatRules[len(part.SnatRules)-1].Id)
	}
	return ret, nil
}

func (gateway *SNatgateway) CreateINatSEntry(rule cloudprovider.SNatSRule) (cloudprovider.ICloudNatSEntry, error) {
	snat, err := gateway.region.CreateNatSTable(rule, gateway.GetId())
	if err != nil {
		return nil, err
	}
	snat.nat = gateway
	return snat, nil
}

func (region *SRegion) CreateNatSTable(rule cloudprovider.SNatSRule, gatewayId string) (*SNatSTable, error) {
	params := make(map[string]interface{})
	params["nat_gateway_id"] = gatewayId
	if len(rule.NetworkID) != 0 {
		params["network_id"] = rule.NetworkID
	}
	if len(rule.SourceCIDR) != 0 {
		params["cidr"] = rule.SourceCIDR
	}
	params["floating_ip_id"] = rule.ExternalIPID
	resp, err := region.post(SERVICE_NAT_V2, "snat_rules", map[string]interface{}{"snat_rule": params})
	if err != nil {
		return nil, errors.Wrapf(err, "create snat")
	}
	ret := &SNatSTable{}
	err = resp.Unmarshal(ret, "snat_rule")
	if err != nil {
		return nil, err
	}
	return ret, nil
}
