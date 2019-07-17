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

type SNatDTableEntry struct {
	multicloud.SResourceBase
	gateway *SNatGateway

	ID           string `json:"id"`
	NatGatewayID string `json:"nat_gateway_id"`
	Protocol     string `json:"protocol"`
	Status       string `json:"status"`
	ExternalIP   string `json:"floating_ip_address"`
	ExternalPort int    `json:"external_service_port"`
	InternalIP   string `json:"private_ip"`
	InternalPort int    `json:"internal_service_port"`
}

func (nat *SNatDTableEntry) GetId() string {
	return nat.ID
}

func (nat *SNatDTableEntry) GetName() string {
	// No name so return id
	return nat.GetId()
}

func (nat *SNatDTableEntry) GetGlobalId() string {
	return nat.GetId()
}

func (nat *SNatDTableEntry) GetStatus() string {
	return NatResouceStatusTransfer(nat.Status)
}

func (nat *SNatDTableEntry) GetIpProtocol() string {
	return nat.Protocol
}

func (nat *SNatDTableEntry) GetExternalIp() string {
	return nat.ExternalIP
}

func (nat *SNatDTableEntry) GetExternalPort() int {
	return nat.ExternalPort
}

func (nat *SNatDTableEntry) GetInternalIp() string {
	return nat.InternalIP
}

func (nat *SNatDTableEntry) GetInternalPort() int {
	return nat.InternalPort
}

// getSNatEntries return all snat rules of gateway
func (gateway *SNatGateway) getDNatEntries() ([]SNatDTableEntry, error) {
	queuies := map[string]string{
		"nat_gateway_id": gateway.GetId(),
	}
	dNatSTableEntris := make([]SNatDTableEntry, 0, 2)
	// can't make true that restapi support marker para in Huawei Cloud
	err := doListAllWithMarker(gateway.region.ecsClient.DNatRules.List, queuies, &dNatSTableEntris)
	if err != nil {
		return nil, errors.Wrapf(err, `get dnat rule of gateway %q`, gateway.GetId())
	}

	for i := range dNatSTableEntris {
		dNatSTableEntris[i].gateway = gateway
	}
	return dNatSTableEntris, nil
}

func (region *SRegion) GetDNatTable(natGatewayID string) ([]SNatDTableEntry, error) {
	queuies := map[string]string{
		"nat_gateway_id": natGatewayID,
	}
	dNatSTableEntris := make([]SNatDTableEntry, 0, 2)
	// can't make true that restapi support marker para in Huawei Cloud
	err := doListAllWithMarker(region.ecsClient.DNatRules.List, queuies, &dNatSTableEntris)
	if err != nil {
		return nil, errors.Wrapf(err, `get dnat rule of gateway %q`, natGatewayID)
	}
	return dNatSTableEntris, nil
}
