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

package ctyun

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SVpc struct {
	multicloud.SVpc

	region *SRegion

	iwires    []cloudprovider.ICloudWire
	secgroups []cloudprovider.ICloudSecurityGroup

	ResVpcID   string `json:"resVpcId"`
	Name       string `json:"name"`
	CIDR       string `json:"cidr"`
	ZoneID     string `json:"zoneId"`
	ZoneName   string `json:"zoneName"`
	VpcStatus  string `json:"vpcStatus"`
	RegionID   string `json:"regionId"`
	CreateDate int64  `json:"createDate"`
}

func (self *SVpc) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SVpc) GetId() string {
	return self.ResVpcID
}

func (self *SVpc) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.ResVpcID
}

func (self *SVpc) GetGlobalId() string {
	return self.GetId()
}

func (self *SVpc) GetStatus() string {
	return api.VPC_STATUS_AVAILABLE
}

func (self *SVpc) Refresh() error {
	new, err := self.region.GetVpc(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SVpc) IsEmulated() bool {
	return false
}

func (self *SVpc) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SVpc) GetIsDefault() bool {
	return false
}

func (self *SVpc) GetCidrBlock() string {
	return self.CIDR
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		err := self.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	return self.iwires, nil
}

// http://ctyun-api-url/apiproxy/v3/getSecurityGroupRules
// http://ctyun-api-url/apiproxy/v3/getSecurityGroups
func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	if self.secgroups == nil {
		err := self.fetchSecurityGroups()
		if err != nil {
			return nil, err
		}
	}
	return self.secgroups, nil
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	rts := []cloudprovider.ICloudRouteTable{}
	return rts, nil
}

func (self *SVpc) Delete() error {
	return self.region.DeleteVpc(self.GetId())
}

func (self *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		err := self.fetchNetworks()
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
	secgroups, err := self.region.GetSecurityGroups("")
	if err != nil {
		return err
	}

	self.secgroups = make([]cloudprovider.ICloudSecurityGroup, len(secgroups))
	for i := 0; i < len(secgroups); i++ {
		secgroups[i].vpc = self
		self.secgroups[i] = &secgroups[i]
	}

	return nil
}

func (self *SVpc) getWireByRegionId(regionId string) *SWire {
	if len(regionId) == 0 {
		return nil
	}

	for i := 0; i < len(self.iwires); i++ {
		wire := self.iwires[i].(*SWire)

		if wire.region.GetId() == regionId {
			return wire
		}
	}

	return nil
}

func (self *SVpc) fetchNetworks() error {
	networks, err := self.region.GetNetwroks(self.GetId())
	if err != nil {
		return err
	}

	if len(networks) == 0 {
		self.iwires = append(self.iwires, &SWire{region: self.region, vpc: self})
		return nil
	}

	for i := 0; i < len(networks); i += 1 {
		wire := self.getWireByRegionId(self.region.GetId())
		networks[i].wire = wire
		wire.addNetwork(&networks[i])
	}

	return nil
}

func (self *SRegion) GetVpc(vpcId string) (*SVpc, error) {
	params := map[string]string{
		"vpcId":    vpcId,
		"regionId": self.GetId(),
	}

	vpc := &SVpc{}
	resp, err := self.client.DoGet("/apiproxy/v3/queryVPCDetail", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetVpc.DoGet")
	}

	err = resp.Unmarshal(vpc, "returnObj")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetVpc.Unmarshal")
	}

	vpc.region = self
	return vpc, nil
}

func (self *SRegion) DeleteVpc(vpcId string) error {
	params := map[string]jsonutils.JSONObject{
		"regionId": jsonutils.NewString(self.GetId()),
		"vpcId":    jsonutils.NewString(vpcId),
	}

	_, err := self.client.DoPost("/apiproxy/v3/deleteVPC", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.DeleteVpc.DoPost")
	}

	return nil
}
