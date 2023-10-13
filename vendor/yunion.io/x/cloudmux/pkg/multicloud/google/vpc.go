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

package google

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SVpc struct {
	multicloud.SVpc
	GoogleTags
	SResourceBase

	region *SRegion

	CreationTimestamp     time.Time
	Network               string
	IpCidrRange           string
	Region                string
	GatewayAddress        string
	Status                string
	AvailableCpuPlatforms []string
	PrivateIpGoogleAccess bool
	Fingerprint           string
	Purpose               string
	Kind                  string
}

func (self *SVpc) GetGlobalVpcId() string {
	gvpc := &SGlobalNetwork{}
	err := self.region.GetBySelfId(self.Network, gvpc)
	if err != nil {
		return ""
	}
	return gvpc.Id
}

func (self *SVpc) Refresh() error {
	vpc, err := self.region.GetVpc(self.Id)
	if err != nil {
		return errors.Wrapf(err, "GetVpc")
	}
	return jsonutils.Update(self, vpc)
}

func (self *SRegion) GetVpc(id string) (*SVpc, error) {
	vpc := &SVpc{region: self}
	return vpc, self.Get("subnetworks", id, vpc)
}

func (vpc *SVpc) GetStatus() string {
	return api.VPC_STATUS_AVAILABLE
}

func (vpc *SVpc) Delete() error {
	return vpc.region.Delete(vpc.SelfLink)
}

func (vpc *SVpc) GetCidrBlock() string {
	return vpc.IpCidrRange
}

func (vpc *SVpc) IsEmulated() bool {
	return false
}

func (vpc *SVpc) GetIsDefault() bool {
	return false
}

func (vpc *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return vpc.region
}

func (vpc *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (vpc *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	return []cloudprovider.ICloudSecurityGroup{}, nil
}

func (vpc *SVpc) getWire() *SWire {
	return &SWire{vpc: vpc}
}

func (vpc *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	wire := vpc.getWire()
	return []cloudprovider.ICloudWire{wire}, nil
}

func (vpc *SVpc) GetIWireById(id string) (cloudprovider.ICloudWire, error) {
	if id != vpc.getWire().GetGlobalId() {
		return nil, cloudprovider.ErrNotFound
	}
	return &SWire{vpc: vpc}, nil
}

func (self *SRegion) CreateVpc(name string, gvpcId string, cidr string, desc string) (*SVpc, error) {
	body := map[string]interface{}{
		"name":        name,
		"description": desc,
		"network":     gvpcId,
		"ipCidrRange": cidr,
	}
	resource := fmt.Sprintf("regions/%s/subnetworks", self.Name)
	vpc := &SVpc{region: self}
	err := self.Insert(resource, jsonutils.Marshal(body), vpc)
	if err != nil {
		return nil, err
	}
	return vpc, nil
}

func (self *SRegion) GetVpcs() ([]SVpc, error) {
	vpcs := []SVpc{}
	resource := fmt.Sprintf("regions/%s/subnetworks", self.Name)
	return vpcs, self.List(resource, nil, 0, "", &vpcs)
}
