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
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SNatSTableEntry struct {
	multicloud.SResourceBase
	gateway *SNatGateway

	ID           string `json:"id"`
	NatGatewayID string `json:"nat_gateway_id"`
	NetworkID    string `json:"network_id"`
	SourceCIDR   string `json:"cidr"`
	Status       string `json:"status"`
	SNatIP       string `json:"floating_ip_address"`
}

func (nat *SNatSTableEntry) GetId() string {
	return nat.ID
}

func (nat *SNatSTableEntry) GetName() string {
	// Snat rule has no name in Huawei Cloud, so return ID
	return nat.GetId()
}

func (nat *SNatSTableEntry) GetGlobalId() string {
	return nat.GetId()
}

func (nat *SNatSTableEntry) GetStatus() string {
	return NatResouceStatusTransfer(nat.Status)
}

func (nat *SNatSTableEntry) GetIP() string {
	return nat.SNatIP
}

func (nat *SNatSTableEntry) GetSourceCIDR() string {
	return nat.SourceCIDR
}

func (nat *SNatSTableEntry) GetNetworkId() string {
	return nat.NetworkID
}

// getSNatEntries return all snat rules of gateway
func (gateway *SNatGateway) getSNatEntries() ([]SNatSTableEntry, error) {
	queuies := map[string]string{
		"nat_gateway_id": gateway.GetId(),
	}
	sNatSTableEntris := make([]SNatSTableEntry, 0, 2)
	err := doListAllWithMarker(gateway.region.ecsClient.SNatRules.List, queuies, &sNatSTableEntris)
	if err != nil {
		return nil, errors.Wrapf(err, `get snat rule of gateway %q`, gateway.GetId())
	}

	for i := range sNatSTableEntris {
		sNatSTableEntris[i].gateway = gateway
	}
	return sNatSTableEntris, nil
}

func (region *SRegion) getSNatTable(natGatewayID string) ([]SNatSTableEntry, error) {
	queuies := map[string]string{
		"nat_gateway_id": natGatewayID,
	}
	sNatSTableEntris := make([]SNatSTableEntry, 0, 2)
	err := doListAllWithMarker(region.ecsClient.SNatRules.List, queuies, &sNatSTableEntris)
	if err != nil {
		return nil, errors.Wrapf(err, `get snat rule of gateway %q`, natGatewayID)
	}

	return sNatSTableEntris, nil
}
