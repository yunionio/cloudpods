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

package oracle

import (
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SVpc struct {
	multicloud.SVpc
	SOracleTag

	region *SRegion

	Id             string
	CidrBlock      string    `json:"cidr-block"`
	TimeCreated    time.Time `json:"time-created"`
	DisplayName    string    `json:"display-name"`
	DnsLabel       string    `json:"dns-label"`
	LifecycleState string    `json:"lifecycle-state"`
}

func (self *SVpc) GetId() string {
	return self.Id
}

func (self *SVpc) GetName() string {
	return self.DisplayName
}

func (self *SVpc) GetGlobalId() string {
	return self.Id
}

func (self *SVpc) GetIsDefault() bool {
	return true
}

func (self *SVpc) GetCidrBlock() string {
	return self.CidrBlock
}

func (self *SVpc) GetStatus() string {
	if self.LifecycleState != "AVAILABLE" {
		return api.VPC_STATUS_UNKNOWN
	}
	return api.VPC_STATUS_AVAILABLE
}

func (self *SVpc) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	groups, err := self.region.GetSecurityGroups(self.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSecurityGroup{}
	for i := range groups {
		groups[i].region = self.region
		ret = append(ret, &groups[i])
	}
	return ret, nil
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SVpc) GetINatGateways() ([]cloudprovider.ICloudNatGateway, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SVpc) GetIWireById(id string) (cloudprovider.ICloudWire, error) {
	wires, err := self.GetIWires()
	if err != nil {
		return nil, err
	}
	for i := range wires {
		if wires[i].GetGlobalId() == id {
			return wires[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SVpc) getRegionWire() *SWire {
	return &SWire{vpc: self}
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	ret := []cloudprovider.ICloudWire{self.getRegionWire()}
	zones, err := self.region.GetZones()
	if err != nil {
		return nil, err
	}
	for i := range zones {
		zones[i].region = self.region
		ret = append(ret, &SWire{vpc: self, zone: &zones[i]})
	}
	return ret, nil
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SVpc) Refresh() error {
	vpc, err := self.region.GetVpc(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, vpc)
}

func (self *SRegion) GetVpcs() ([]SVpc, error) {
	resp, err := self.list(SERVICE_IAAS, "vcns", nil)
	if err != nil {
		return nil, err
	}
	ret := []SVpc{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) GetVpc(id string) (*SVpc, error) {
	resp, err := self.get(SERVICE_IAAS, "vcns", id, nil)
	if err != nil {
		return nil, err
	}
	ret := &SVpc{region: self}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
