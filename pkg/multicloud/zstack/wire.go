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

package zstack

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SWire struct {
	vpc *SVpc

	inetworks []cloudprovider.ICloudNetwork

	ZStackBasic
	Vlan              int    `json:"vlan"`
	ZoneUUID          string `json:"zoneUuid"`
	PhysicalInterface string `json:"physicalInterface"`
	Type              string `json:"type"`
	ZStackTime
	AttachedClusterUUIDs []string `json:"attachedClusterUuids"`
}

func (region *SRegion) GetWire(wireId string) (*SWire, error) {
	wire := &SWire{vpc: region.GetVpc()}
	return wire, region.client.getResource("l2-networks", wireId, wire)
}

func (region *SRegion) GetWires(zoneId string, wireId string, clusterId string) ([]SWire, error) {
	wires := []SWire{}
	clusterIds, err := region.GetClusterIds()
	if err != nil {
		return nil, err
	}
	params := []string{"q=attachedClusterUuids?=" + strings.Join(clusterIds, ",")}
	if len(clusterId) > 0 {
		params = []string{"q=attachedClusterUuids?=" + clusterId}
	}
	if len(zoneId) > 0 {
		params = append(params, "q=zone.uuid="+zoneId)
	}
	if len(wireId) > 0 {
		params = append(params, "q=uuid="+wireId)
	}
	err = region.client.listAll("l2-networks", params, &wires)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(wires); i++ {
		wires[i].vpc = region.GetVpc()
	}
	return wires, nil
}

func (wire *SWire) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (wire *SWire) GetId() string {
	return wire.UUID
}

func (wire *SWire) GetName() string {
	return wire.Name
}

func (wire *SWire) IsEmulated() bool {
	return false
}

func (wire *SWire) GetStatus() string {
	return "available"
}

func (wire *SWire) Refresh() error {
	return nil
}

func (wire *SWire) GetGlobalId() string {
	return wire.UUID
}

func (wire *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return nil
}

func (wire *SWire) GetIZone() cloudprovider.ICloudZone {
	zone, _ := wire.vpc.region.GetZone(wire.ZoneUUID)
	return zone
}

func (wire *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	if wire.inetworks == nil || len(wire.inetworks) == 0 {
		networks, err := wire.vpc.region.GetNetworks(wire.ZoneUUID, wire.UUID, "", "")
		if err != nil {
			return nil, err
		}
		wire.inetworks = []cloudprovider.ICloudNetwork{}
		for i := 0; i < len(networks); i++ {
			networks[i].wire = wire
			wire.inetworks = append(wire.inetworks, &networks[i])
		}
	}
	return wire.inetworks, nil
}

func (wire *SWire) GetBandwidth() int {
	return 10000
}

func (wire *SWire) CreateINetwork(name string, cidr string, desc string) (cloudprovider.ICloudNetwork, error) {
	network, err := wire.vpc.region.CreateNetwork(name, cidr, wire.UUID, desc)
	if err != nil {
		return nil, err
	}
	network.wire = wire
	return network, nil
}

func (wire *SWire) GetINetworkById(netid string) (cloudprovider.ICloudNetwork, error) {
	idInfo := strings.Split(netid, "/")
	if len(idInfo) == 2 {
		network, err := wire.vpc.region.GetNetwork(wire.ZoneUUID, wire.UUID, idInfo[0], idInfo[1])
		if err != nil {
			return nil, err
		}
		network.wire = wire
		return network, nil
	}
	return nil, fmt.Errorf("invalid netid %s", netid)
}
