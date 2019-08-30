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

type SNatDEntry struct {
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

func (region *SRegion) GetNatDTable(natGatewayID string) ([]SNatDEntry, error) {
	queuies := map[string]string{
		"nat_gateway_id": natGatewayID,
	}
	dNatSTableEntries := make([]SNatDEntry, 0, 2)
	// can't make true that restapi support marker para in Huawei Cloud
	err := doListAllWithMarker(region.ecsClient.DNatRules.List, queuies, &dNatSTableEntries)
	if err != nil {
		return nil, errors.Wrapf(err, `get dnat rule of gateway %q`, natGatewayID)
	}
	for i := range dNatSTableEntries {
		nat := &dNatSTableEntries[i]
		if len(nat.InternalIP) == 0 {
			port, err := region.GetPort(nat.PortID)
			if err != nil {
				return nil, errors.Wrapf(err, `get port info for transfer to ip of port_id %q error`, nat.PortID)
			}
			nat.InternalIP = port.FixedIps[0].IpAddress
		}
	}
	return dNatSTableEntries, nil
}

func (region *SRegion) DeleteNatDEntry(entryID string) error {
	_, err := region.ecsClient.DNatRules.Delete(entryID, nil)
	if err != nil {
		return errors.Wrapf(err, `delete dnat rule %q failed`, entryID)
	}
	return nil
}

func (nat *SNatDEntry) Refresh() error {
	new, err := nat.gateway.region.GetNatDEntryByID(nat.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(nat, new)
}
