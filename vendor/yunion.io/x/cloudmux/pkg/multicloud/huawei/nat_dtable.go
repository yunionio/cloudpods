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

type SNatDTable struct {
	multicloud.SResourceBase
	multicloud.STagBase
	nat *SNatgateway

	Id                       string
	Description              string
	PortId                   string
	PrivateIp                string
	InternalServicePort      int
	NatGatewayId             string
	FloatingIpId             string
	FloatingIpAddress        string
	ExternalServicePort      int
	Status                   string
	AdminStateUp             bool
	InternalServicePortRange string
	ExternalServicePortRange string
	Protocol                 string
	CreatedAt                string
	GlobalEipId              string
	GlobalEipAddress         string
}

func (nat *SNatDTable) GetId() string {
	return nat.Id
}

func (nat *SNatDTable) GetName() string {
	return nat.GetId()
}

func (nat *SNatDTable) GetGlobalId() string {
	return nat.GetId()
}

func (nat *SNatDTable) GetStatus() string {
	return NatResouceStatusTransfer(nat.Status)
}

func (nat *SNatDTable) Refresh() error {
	table, err := nat.nat.region.GetNatDTable(nat.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(nat, table)
}

func (nat *SNatDTable) GetIpProtocol() string {
	return nat.Protocol
}

func (nat *SNatDTable) GetExternalIp() string {
	return nat.FloatingIpAddress
}

func (nat *SNatDTable) GetExternalPort() int {
	return nat.ExternalServicePort
}

func (nat *SNatDTable) GetInternalIp() string {
	return nat.PrivateIp
}

func (nat *SNatDTable) GetInternalPort() int {
	return nat.InternalServicePort
}

func (nat *SNatDTable) Delete() error {
	return nat.nat.region.DeletePublicNatDEntry(nat.nat.Id, nat.GetId())
}

func (region *SRegion) DeletePublicNatDEntry(natId, id string) error {
	res := fmt.Sprintf("nat_gateways/%s/dnat_rules/%s", natId, id)
	_, err := region.delete(SERVICE_NAT_V2, res)
	return err
}

func (region *SRegion) GetPublicNatDTable(natId string) ([]SNatDTable, error) {
	query := url.Values{}
	if len(natId) > 0 {
		query.Set("nat_gateway_id", natId)
	}
	ret := []SNatDTable{}
	for {
		resp, err := region.list(SERVICE_NAT_V2, "dnat_rules", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			DnatRules []SNatDTable
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.DnatRules...)
		if len(part.DnatRules) == 0 {
			break
		}
		query.Set("marker", part.DnatRules[len(part.DnatRules)-1].Id)
	}
	return ret, nil
}

func (gateway *SNatgateway) CreateINatDEntry(rule cloudprovider.SNatDRule) (cloudprovider.ICloudNatDEntry, error) {
	dnat, err := gateway.region.CreateNatDTable(rule, gateway.GetId())
	if err != nil {
		return nil, err
	}
	dnat.nat = gateway
	return dnat, nil
}

func (region *SRegion) CreateNatDTable(rule cloudprovider.SNatDRule, gatewayID string) (*SNatDTable, error) {
	params := make(map[string]interface{})
	params["nat_gateway_id"] = gatewayID
	params["private_ip"] = rule.InternalIP
	params["internal_service_port"] = rule.InternalPort
	params["floating_ip_id"] = rule.ExternalIPID
	params["external_service_port"] = rule.ExternalPort
	params["protocol"] = rule.Protocol

	resp, err := region.post(SERVICE_NAT_V2, "dnat_rules", map[string]interface{}{"dnat_rule": params})
	if err != nil {
		return nil, errors.Wrapf(err, "create dnat")
	}
	ret := &SNatDTable{}
	err = resp.Unmarshal(ret, "dnat_rule")
	if err != nil {
		return nil, err
	}
	return ret, nil
}
