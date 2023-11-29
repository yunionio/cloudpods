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

package apsara

import (
	"strings"
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

const (
	VpcAvailable = "Available"
	VpcPending   = "Pending"
)

// "CidrBlock":"172.31.0.0/16","CreationTime":"2017-03-19T13:37:40Z","Description":"System created default VPC.","IsDefault":true,"RegionId":"cn-hongkong","Status":"Available","UserCidrs":{"UserCidr":[]},"VRouterId":"vrt-j6c00qrol733dg36iq4qj","VSwitchIds":{"VSwitchId":["vsw-j6c3gig5ub4fmi2veyrus"]},"VpcId":"vpc-j6c86z3sh8ufhgsxwme0q","VpcName":""

type SUserCIDRs struct {
	UserCidr []string
}

type SVSwitchIds struct {
	VSwitchId []string
}

type SVpc struct {
	multicloud.SVpc
	ApsaraTags

	region *SRegion

	iwires []cloudprovider.ICloudWire

	secgroups   []cloudprovider.ICloudSecurityGroup
	routeTables []cloudprovider.ICloudRouteTable

	CidrBlock    string
	CreationTime time.Time
	Description  string
	IsDefault    bool
	RegionId     string
	Status       string
	UserCidrs    SUserCIDRs
	VRouterId    string
	VSwitchIds   SVSwitchIds
	VpcId        string
	VpcName      string

	DepartmentInfo
}

func (self *SVpc) GetId() string {
	return self.VpcId
}

func (self *SVpc) GetName() string {
	if len(self.VpcName) > 0 {
		return self.VpcName
	}
	return self.VpcId
}

func (self *SVpc) GetGlobalId() string {
	return self.VpcId
}

func (self *SVpc) IsEmulated() bool {
	return false
}

func (self *SVpc) GetIsDefault() bool {
	return self.IsDefault
}

func (self *SVpc) GetCidrBlock() string {
	return self.CidrBlock
}

func (self *SVpc) GetStatus() string {
	return strings.ToLower(self.Status)
}

func (self *SVpc) GetCreatedAt() time.Time {
	return self.CreationTime
}

func (self *SVpc) Refresh() error {
	new, err := self.region.getVpc(self.VpcId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SVpc) GetSysTags() map[string]string {
	tags := self.ApsaraTags.GetSysTags()
	if len(self.ResourceGroup) > 0 {
		tags["ResourceGroup"] = self.ResourceGroup
	}
	if len(self.ResourceGroupId) > 0 {
		tags["ResourceGroupId"] = self.ResourceGroupId
	}
	if len(self.Department) > 0 {
		tags["Department"] = self.Department
	}
	if len(self.DepartmentName) > 0 {
		tags["DepartmentName"] = self.DepartmentName
	}
	if len(self.ResourceGroupName) > 0 {
		groupName := strings.TrimPrefix(self.ResourceGroupName, "ResourceSet(")
		groupName = strings.TrimSuffix(groupName, ")")
		tags["ResourceGroupName"] = groupName
	}
	return tags
}

func (self *SVpc) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SVpc) getWireByZoneId(zoneId string) *SWire {
	for i := 0; i <= len(self.iwires); i += 1 {
		wire := self.iwires[i].(*SWire)
		if wire.zone.ZoneId == zoneId {
			return wire
		}
	}
	return nil
}

func (self *SVpc) fetchVSwitches() error {
	switches, total, err := self.region.GetVSwitches(nil, self.VpcId, 0, 50)
	if err != nil {
		return err
	}
	if total > len(switches) {
		switches, _, err = self.region.GetVSwitches(nil, self.VpcId, 0, total)
		if err != nil {
			return err
		}
	}
	for i := 0; i < len(switches); i += 1 {
		wire := self.getWireByZoneId(switches[i].ZoneId)
		switches[i].wire = wire
		wire.addNetwork(&switches[i])
	}
	return nil
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		err := self.fetchVSwitches()
		if err != nil {
			return nil, err
		}
	}
	return self.iwires, nil
}

func (self *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		err := self.fetchVSwitches()
		if err != nil {
			return nil, err
		}
	}
	for i := 0; i < len(self.iwires); i += 1 {
		if self.iwires[i].GetGlobalId() == wireId {
			return self.iwires[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SVpc) fetchSecurityGroups() error {
	secgroups := make([]SSecurityGroup, 0)
	for {
		parts, total, err := self.region.GetSecurityGroups(self.VpcId, "", []string{}, len(secgroups), 50)
		if err != nil {
			return err
		}
		secgroups = append(secgroups, parts...)
		if len(secgroups) >= total || len(parts) == 0 {
			break
		}
	}
	self.secgroups = make([]cloudprovider.ICloudSecurityGroup, len(secgroups))
	for i := 0; i < len(secgroups); i++ {
		secgroups[i].region = self.region
		self.secgroups[i] = &secgroups[i]
	}
	return nil
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	if self.secgroups == nil {
		err := self.fetchSecurityGroups()
		if err != nil {
			return nil, err
		}
	}
	return self.secgroups, nil
}

func (self *SVpc) fetchRouteTables() error {
	routeTables := make([]*SRouteTable, 0)
	for {
		parts, total, err := self.RemoteGetRouteTableList(len(routeTables), 50)
		if err != nil {
			return err
		}
		routeTables = append(routeTables, parts...)
		if len(routeTables) >= total || len(parts) == 0 {
			break
		}
	}
	self.routeTables = make([]cloudprovider.ICloudRouteTable, len(routeTables))
	for i := 0; i < len(routeTables); i++ {
		routeTables[i].vpc = self
		self.routeTables[i] = routeTables[i]
	}
	return nil
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	if self.routeTables == nil {
		err := self.fetchRouteTables()
		if err != nil {
			return nil, err
		}
	}
	return self.routeTables, nil
}

func (self *SVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SVpc) Delete() error {
	return self.region.DeleteVpc(self.VpcId)
}

func (self *SVpc) getNatGateways() ([]SNatGetway, error) {
	natgatways := make([]SNatGetway, 0)
	gwTotal := -1
	for gwTotal < 0 || len(natgatways) < gwTotal {
		parts, total, err := self.region.GetNatGateways(self.VpcId, "", len(natgatways), 50)
		if err != nil {
			return nil, err
		}
		if len(parts) > 0 {
			natgatways = append(natgatways, parts...)
		}
		gwTotal = total
	}
	for i := 0; i < len(natgatways); i += 1 {
		natgatways[i].vpc = self
	}
	return natgatways, nil
}

func (self *SVpc) GetINatGateways() ([]cloudprovider.ICloudNatGateway, error) {
	nats := []SNatGetway{}
	for {
		parts, total, err := self.region.GetNatGateways(self.VpcId, "", len(nats), 50)
		if err != nil {
			return nil, err
		}
		nats = append(nats, parts...)
		if len(nats) >= total || len(parts) == 0 {
			break
		}
	}
	inats := []cloudprovider.ICloudNatGateway{}
	for i := 0; i < len(nats); i++ {
		nats[i].vpc = self
		inats = append(inats, &nats[i])
	}
	return inats, nil
}
