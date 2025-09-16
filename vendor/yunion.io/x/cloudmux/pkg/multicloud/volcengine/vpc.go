// Copyright 2023 Yunion
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

package volcengine

import (
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SUserCIDRs []string

type SVpc struct {
	multicloud.SVpc
	VolcEngineTags

	region *SRegion

	routeTables []cloudprovider.ICloudRouteTable

	RegionId                string
	VpcId                   string
	VpcName                 string
	CidrBlock               string
	Ipv6CidrBlock           string
	CidrBlockAssociationSet []string
	IsDefault               bool
	Status                  string
	InstanceTenancy         string
}

func (vpc *SVpc) GetId() string {
	return vpc.VpcId
}

func (vpc *SVpc) GetName() string {
	if len(vpc.VpcName) > 0 {
		return vpc.VpcName
	}
	return vpc.VpcId
}

func (vpc *SVpc) GetGlobalId() string {
	return vpc.VpcId
}

func (vpc *SVpc) GetIsDefault() bool {
	return vpc.IsDefault
}

func (vpc *SVpc) GetCidrBlock() string {
	return vpc.CidrBlock
}

func (vpc *SVpc) GetCidrBlock6() string {
	return vpc.Ipv6CidrBlock
}

func (vpc *SVpc) GetStatus() string {
	return strings.ToLower(vpc.Status)
}

func (vpc *SVpc) Refresh() error {
	new, err := vpc.region.getVpc(vpc.VpcId)
	if err != nil {
		return err
	}
	return jsonutils.Update(vpc, new)
}

func (vpc *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return vpc.region
}

func (vpc *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	zones, err := vpc.region.GetZones("")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range zones {
		zones[i].region = vpc.region
		ret = append(ret, &SWire{zone: &zones[i], vpc: vpc})
	}
	return ret, nil
}

func (vpc *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	wires, err := vpc.GetIWires()
	if err != nil {
		return nil, err
	}
	for i := range wires {
		if wires[i].GetGlobalId() == wireId {
			return wires[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, wireId)
}

func (vpc *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	secgroups := make([]SSecurityGroup, 0)
	pageNumber := 1
	for {
		parts, total, err := vpc.region.GetSecurityGroups(vpc.VpcId, "", nil, 50, pageNumber)
		if err != nil {
			return nil, err
		}
		secgroups = append(secgroups, parts...)
		if len(secgroups) >= total {
			break
		}
		pageNumber += 1
	}
	ret := []cloudprovider.ICloudSecurityGroup{}
	for i := 0; i < len(secgroups); i++ {
		secgroups[i].region = vpc.region
		ret = append(ret, &secgroups[i])
	}
	return ret, nil
}

func (vpc *SVpc) fetchRouteTables() error {
	routeTables := make([]*SRouteTable, 0)
	pageNumber := 1
	for {
		parts, total, err := vpc.RemoteGetRouteTableList(pageNumber, 50)
		if err != nil {
			return err
		}
		routeTables = append(routeTables, parts...)
		if len(routeTables) >= total {
			break
		}
		pageNumber += 1
	}
	vpc.routeTables = make([]cloudprovider.ICloudRouteTable, len(routeTables))
	for i := 0; i < len(routeTables); i++ {
		routeTables[i].vpc = vpc
		vpc.routeTables[i] = routeTables[i]
	}
	return nil
}

func (vpc *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	if vpc.routeTables == nil {
		err := vpc.fetchRouteTables()
		if err != nil {
			return nil, err
		}
	}
	return vpc.routeTables, nil
}

func (vpc *SVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	tables, err := vpc.GetIRouteTables()
	if err != nil {
		return nil, errors.Wrapf(err, "GetIRouteTables")
	}
	for i := range tables {
		if tables[i].GetGlobalId() == routeTableId {
			return tables[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, routeTableId)
}

func (vpc *SVpc) Delete() error {
	return vpc.region.DeleteVpc(vpc.VpcId)
}

func (vpc *SVpc) getNatGateways() ([]SNatGateway, error) {
	nats := make([]SNatGateway, 0)
	pageNumber := 1
	for {
		parts, total, err := vpc.region.GetNatGateways(vpc.VpcId, "", pageNumber, 50)
		if err != nil {
			return nil, err
		}
		nats = append(nats, parts...)
		if len(nats) >= total {
			break
		}
		pageNumber += 1
	}
	for i := 0; i < len(nats); i += 1 {
		nats[i].vpc = vpc
	}
	return nats, nil
}

func (vpc *SVpc) getINatGateways() ([]cloudprovider.ICloudNatGateway, error) {
	nats := make([]SNatGateway, 0)
	pageNumber := 1
	for {
		parts, total, err := vpc.region.GetNatGateways(vpc.VpcId, "", pageNumber, 50)
		if err != nil {
			return nil, err
		}
		nats = append(nats, parts...)
		if len(nats) >= total {
			break
		}
		pageNumber += 1
	}
	inats := []cloudprovider.ICloudNatGateway{}
	for i := 0; i < len(nats); i++ {
		nats[i].vpc = vpc
		inats = append(inats, &nats[i])
	}
	return inats, nil
}

func (vpc *SVpc) CreateINatGateway(opts *cloudprovider.NatGatewayCreateOptions) (cloudprovider.ICloudNatGateway, error) {
	nat, err := vpc.region.CreateNatGateway(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateNatGateway")
	}
	nat.vpc = vpc
	return nat, nil
}

func (vpc *SVpc) GetAuthorityOwnerId() string {
	return vpc.region.client.ownerId
}
