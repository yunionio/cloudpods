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
	"net/url"

	"yunion.io/x/jsonutils"

	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNatDEntry struct {
	multicloud.SResourceBase
	HuaweiTags
	gateway *SNatGateway

	ID           string `json:"id"`
	NatGatewayID string `json:"nat_gateway_id"`
	Protocol     string `json:"protocol"`
	Status       string `json:"status"`
	ExternalIP   string `json:"floating_ip_address"`
	ExternalPort int    `json:"external_service_port"`
	InternalIP   string `json:"private_ip"`
	InternalPort int    `json:"internal_service_port"`
	PortID       string `json:"port_id"`
	AdminStateUp bool   `json:"admin_state_up"`
}

func (nat *SNatDEntry) GetId() string {
	return nat.ID
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

func (region *SRegion) GetNatDTable(natId string) ([]SNatDEntry, error) {
	query := url.Values{}
	if len(natId) > 0 {
		query.Set("gateway_id", natId)
	}
	ret := []SNatDEntry{}
	for {
		resp, err := region.list(SERVICE_NAT, "private-nat/dnat-rules", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			DnatRules []SNatDEntry
			PageInfo  sPageInfo
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.DnatRules...)
		if len(part.PageInfo.NextMarker) == 0 || len(part.DnatRules) == 0 {
			break
		}
		query.Set("marker", part.PageInfo.NextMarker)
	}
	return ret, nil
}

func (region *SRegion) DeleteNatDEntry(id string) error {
	_, err := region.delete(SERVICE_NAT, "private-nat/dnat-rules/"+id)
	return err
}

func (nat *SNatDEntry) Refresh() error {
	ret, err := nat.gateway.region.GetNatDEntryByID(nat.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(nat, ret)
}
