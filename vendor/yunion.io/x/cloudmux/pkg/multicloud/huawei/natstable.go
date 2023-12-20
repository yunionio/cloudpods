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

type SNatSEntry struct {
	multicloud.SResourceBase
	HuaweiTags
	gateway *SNatGateway

	ID           string `json:"id"`
	NatGatewayID string `json:"nat_gateway_id"`
	NetworkID    string `json:"network_id"`
	SourceCIDR   string `json:"cidr"`
	Status       string `json:"status"`
	SNatIP       string `json:"floating_ip_address"`
	AdminStateUp bool   `json:"admin_state_up"`
}

func (nat *SNatSEntry) GetId() string {
	return nat.ID
}

func (nat *SNatSEntry) GetName() string {
	// Snat rule has no name in Huawei Cloud, so return ID
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
	return nat.NetworkID
}

func (nat *SNatSEntry) Delete() error {
	return nat.gateway.region.DeleteNatSEntry(nat.GetId())
}

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

func (region *SRegion) GetNatSTable(natId string) ([]SNatSEntry, error) {
	query := url.Values{}
	if len(natId) > 0 {
		query.Set("gateway_id", natId)
	}
	ret := []SNatSEntry{}
	for {
		resp, err := region.list(SERVICE_NAT, "private-nat/snat-rules", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			SnatRules []SNatSEntry
			PageInfo  sPageInfo
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.SnatRules...)
		if len(part.PageInfo.NextMarker) == 0 || len(part.SnatRules) == 0 {
			break
		}
		query.Set("marker", part.PageInfo.NextMarker)
	}
	return ret, nil
}

func (region *SRegion) DeleteNatSEntry(id string) error {
	_, err := region.delete(SERVICE_NAT, "private-nat/snat-rules/"+id)
	return err
}

func (nat *SNatSEntry) Refresh() error {
	ret, err := nat.gateway.region.GetNatSEntryByID(nat.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(nat, ret)
}
