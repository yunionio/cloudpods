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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/multicloud"
)

type SNatSEntry struct {
	multicloud.SResourceBase
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

func (region *SRegion) GetNatSTable(natGatewayID string) ([]SNatSEntry, error) {
	queuies := map[string]string{
		"nat_gateway_id": natGatewayID,
	}
	sNatSTableEntris := make([]SNatSEntry, 0, 2)
	err := doListAllWithMarker(region.ecsClient.SNatRules.List, queuies, &sNatSTableEntris)
	if err != nil {
		return nil, errors.Wrapf(err, `get snat rule of gateway %q`, natGatewayID)
	}
	for i := range sNatSTableEntris {
		nat := &sNatSTableEntris[i]
		if len(nat.SourceCIDR) != 0 {
			continue
		}
		subnet := SNetwork{}
		err := DoGet(region.ecsClient.Subnets.Get, nat.NetworkID, map[string]string{}, &subnet)
		if err != nil {
			return nil, errors.Wrapf(err, `get cidr of subnet %q`, nat.NetworkID)
		}
		nat.SourceCIDR = subnet.CIDR
	}
	return sNatSTableEntris, nil
}

func (region *SRegion) DeleteNatSEntry(entryID string) error {
	_, err := region.ecsClient.SNatRules.Delete(entryID, nil)
	if err != nil {
		return errors.Wrapf(err, `delete snat rule %q failed`, entryID)
	}
	return nil
}

func (nat *SNatSEntry) Refresh() error {
	new, err := nat.gateway.region.GetNatSEntryByID(nat.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(nat, new)
}
