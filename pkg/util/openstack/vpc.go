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

package openstack

import (
	"time"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	VPC_STATUS_ACTIVE = "ACTIVE"
	VPC_STATUS_DOWN   = "DOWN"
	VPC_STATUS_BUILD  = "BUILD"
	VPC_STATUS_ERROR  = "ERROR"
)

type SVpc struct {
	region *SRegion

	iwires    []cloudprovider.ICloudWire
	secgroups []cloudprovider.ICloudSecurityGroup

	AdminStateUp          bool
	AvailabilityZoneHints []string
	AvailabilityZones     []string
	CreatedAt             time.Time
	DnsDomain             string
	ID                    string
	Ipv4AddressScope      string
	Ipv6AddressScope      string
	L2Adjacency           bool
	Mtu                   int
	Name                  string
	PortSecurityEnabled   bool
	ProjectID             string
	QosPolicyID           string
	RevisionNumber        int
	External              bool `json:"router:external"`
	Shared                bool
	Status                string
	Subnets               []string
	TenantID              string
	UpdatedAt             time.Time
	VlanTransparent       bool
	Fescription           string
	IsDefault             bool
}

func (vpc *SVpc) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (vpc *SVpc) GetId() string {
	return vpc.ID
}

func (vpc *SVpc) GetName() string {
	if len(vpc.Name) > 0 {
		return vpc.Name
	}
	return vpc.ID
}

func (vpc *SVpc) GetGlobalId() string {
	return vpc.ID
}

func (vpc *SVpc) IsEmulated() bool {
	return false
}

func (vpc *SVpc) GetIsDefault() bool {
	return vpc.IsDefault
}

func (vpc *SVpc) GetCidrBlock() string {
	return ""
}

func (vpc *SVpc) GetStatus() string {
	switch vpc.Status {
	case VPC_STATUS_ACTIVE:
		return api.VPC_STATUS_AVAILABLE
	case VPC_STATUS_BUILD, VPC_STATUS_DOWN:
		return api.VPC_STATUS_PENDING
	case VPC_STATUS_ERROR:
		return api.VPC_STATUS_FAILED
	default:
		return api.VPC_STATUS_UNKNOWN
	}
}

func (vpc *SVpc) Delete() error {
	return vpc.region.DeleteVpc(vpc.ID)
}

func (region *SRegion) DeleteVpc(vpcId string) error {
	_, err := region.Delete("network", "/v2.0/networks/"+vpcId, "")
	return err
}

func (vpc *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := vpc.region.GetSecurityGroups()
	if err != nil {
		return nil, err
	}
	iSecgroups := []cloudprovider.ICloudSecurityGroup{}
	for i := 0; i < len(secgroups); i++ {
		secgroups[i].vpc = vpc
		iSecgroups = append(iSecgroups, &secgroups[i])
	}
	return iSecgroups, nil
}

func (vpc *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	rts := []cloudprovider.ICloudRouteTable{}
	return rts, nil
}

func (vpc *SVpc) fetchWires() error {
	if len(vpc.region.izones) == 0 {
		if err := vpc.region.fetchZones(); err != nil {
			return err
		}
	}
	wire := SWire{zone: vpc.region.izones[0].(*SZone), vpc: vpc}
	vpc.iwires = []cloudprovider.ICloudWire{&wire}
	return nil
}

func (vpc *SVpc) getWire() *SWire {
	if vpc.iwires == nil {
		vpc.fetchWires()
	}
	return vpc.iwires[0].(*SWire)
}

func (vpc *SVpc) fetchNetworks() error {
	networks, err := vpc.region.GetNetworks(vpc.ID)
	if err != nil {
		return err
	}
	for i := 0; i < len(networks); i++ {
		wire := vpc.getWire()
		networks[i].wire = wire
		wire.addNetwork(&networks[i])
	}
	return nil
}

func (vpc *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	if vpc.iwires == nil {
		err := vpc.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	for i := 0; i < len(vpc.iwires); i++ {
		if vpc.iwires[i].GetGlobalId() == wireId {
			return vpc.iwires[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (vpc *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if vpc.iwires == nil {
		err := vpc.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	return vpc.iwires, nil
}

func (vpc *SVpc) GetManagerId() string {
	return vpc.region.client.providerID
}

func (vpc *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return vpc.region
}

func (region *SRegion) GetVpc(vpcId string) (*SVpc, error) {
	_, resp, err := region.Get("network", "/v2.0/networks/"+vpcId, "", nil)
	if err != nil {
		return nil, err
	}
	vpc := SVpc{}
	return &vpc, resp.Unmarshal(&vpc, "network")
}

func (region *SRegion) GetVpcs() ([]SVpc, error) {
	_, resp, err := region.List("network", "/v2.0/networks", "", nil)
	if err != nil {
		return nil, err
	}
	vpcs := []SVpc{}
	return vpcs, resp.Unmarshal(&vpcs, "networks")
}

func (vpc *SVpc) Refresh() error {
	new, err := vpc.region.GetVpc(vpc.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(vpc, new)
}

func (vpc *SVpc) addWire(wire *SWire) {
	if vpc.iwires == nil {
		vpc.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	vpc.iwires = append(vpc.iwires, wire)
}
