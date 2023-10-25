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
	"fmt"
	"net/url"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

const (
	VPC_STATUS_ACTIVE = "ACTIVE"
	VPC_STATUS_DOWN   = "DOWN"
	VPC_STATUS_BUILD  = "BUILD"
	VPC_STATUS_ERROR  = "ERROR"
)

type SVpc struct {
	multicloud.SVpc
	OpenStackTags
	region *SRegion

	AdminStateUp          bool
	AvailabilityZoneHints []string
	AvailabilityZones     []string
	CreatedAt             time.Time
	DnsDomain             string
	Id                    string
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
	NetworkType           string `json:"provider:network_type"` // flat, vlan, vxlan, or gre ...
	PhysicalNetwork       string `json:"provider:physical_network"`
	SegmentationId        string `json:"provider:segmentation_id"`
}

func (vpc *SVpc) GetId() string {
	return vpc.Id
}

func (vpc *SVpc) GetName() string {
	if len(vpc.Name) > 0 {
		return vpc.Name
	}
	return vpc.Id
}

func (vpc *SVpc) GetGlobalId() string {
	return vpc.Id
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
	return vpc.region.DeleteVpc(vpc.Id)
}

func (region *SRegion) DeleteVpc(vpcId string) error {
	resource := fmt.Sprintf("/v2.0/networks/%s", vpcId)
	_, err := region.vpcDelete(resource)
	return err
}

func (vpc *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	return []cloudprovider.ICloudSecurityGroup{}, nil
}

func (vpc *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	if vpc.PhysicalNetwork == "public" {
		return []cloudprovider.ICloudRouteTable{}, nil
	}
	err := vpc.region.fetchrouters()
	if err != nil {
		return nil, errors.Wrap(err, "vpc.region.fetchrouters()")
	}
	routeTables := []SRouteTable{}
	for index, router := range vpc.region.routers {
		if len(router.Routes) < 1 {
			continue
		}
		for _, port := range router.ports {
			if port.NetworkID == vpc.GetId() {
				routeTable := SRouteTable{}
				routeTable.entries = router.Routes
				routeTable.router = &vpc.region.routers[index]
				routeTable.vpc = vpc
				routeTables = append(routeTables, routeTable)
				break
			}
		}
	}
	ret := []cloudprovider.ICloudRouteTable{}
	for i := range routeTables {
		ret = append(ret, &routeTables[i])
	}
	return ret, nil
}

func (self *SVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (vpc *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	iwires, err := vpc.GetIWires()
	if err != nil {
		return nil, errors.Wrap(err, "GetIWires")
	}
	for i := range iwires {
		if iwires[i].GetGlobalId() == wireId {
			return iwires[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (vpc *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return []cloudprovider.ICloudWire{&SWire{vpc: vpc}}, nil
}

func (vpc *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return vpc.region
}

func (region *SRegion) GetVpc(vpcId string) (*SVpc, error) {
	vpc := &SVpc{region: region}
	resource := fmt.Sprintf("/v2.0/networks/%s", vpcId)
	resp, err := region.vpcGet(resource)
	if err != nil {
		return nil, errors.Wrapf(err, "vpcGet(%s)", resource)
	}
	err = resp.Unmarshal(vpc, "network")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return vpc, nil
}

func (region *SRegion) GetVpcs(projectId string) ([]SVpc, error) {
	vpcs := []SVpc{}
	resource := "/v2.0/networks"
	query := url.Values{}
	if len(projectId) > 0 {
		query.Set("tenant_id", projectId)
	}
	for {
		resp, err := region.vpcList(resource, query)
		if err != nil {
			return nil, errors.Wrapf(err, "vpcList.%s", resource)
		}

		part := struct {
			Networks      []SVpc
			NetworksLinks SNextLinks
		}{}

		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		vpcs = append(vpcs, part.Networks...)
		marker := part.NetworksLinks.GetNextMark()
		if len(marker) == 0 {
			break
		}
		query.Set("marker", marker)
	}
	return vpcs, nil
}

func (vpc *SVpc) Refresh() error {
	_vpc, err := vpc.region.GetVpc(vpc.Id)
	if err != nil {
		return errors.Wrapf(err, "GetVpc(%s)", vpc.Id)
	}
	return jsonutils.Update(vpc, _vpc)
}

func (region *SRegion) CreateVpc(name, desc string) (*SVpc, error) {
	params := map[string]map[string]string{
		"network": {
			"name":        name,
			"description": desc,
		},
	}
	resource := "/v2.0/networks"
	resp, err := region.vpcPost(resource, params)
	if err != nil {
		return nil, errors.Wrap(err, "vpcPost")
	}
	vpc := &SVpc{region: region}
	err = resp.Unmarshal(vpc, "network")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return vpc, nil
}
