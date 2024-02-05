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

package cloudpods

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SVpc struct {
	multicloud.SVpc
	CloudpodsTags
	region *SRegion

	api.VpcDetails
}

func (self *SVpc) GetName() string {
	return self.Name
}

func (self *SVpc) GetId() string {
	return self.Id
}

func (self *SVpc) GetGlobalId() string {
	return self.Id
}

func (self *SVpc) GetStatus() string {
	return self.Status
}

func (self *SVpc) Refresh() error {
	vpc, err := self.region.GetVpc(self.Id)
	if err != nil {
		return errors.Wrapf(err, "GetVpc(%s)", self.Id)
	}
	return jsonutils.Update(self, vpc)
}

func (self *SVpc) GetCidrBlock() string {
	return self.CidrBlock
}

func (self *SVpc) GetCidrBlock6() string {
	return self.CidrBlock6
}

func (self *SVpc) GetIRouteTableById(id string) (cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SVpc) GetIsDefault() bool {
	return self.IsDefault
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SVpc) GetExternalAccessMode() string {
	return self.ExternalAccessMode
}

func (self *SVpc) Delete() error {
	return self.region.cli.delete(&modules.Vpcs, self.Id)
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	vpcs, err := self.GetVpcs()
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpcs")
	}
	ret := []cloudprovider.ICloudVpc{}
	for i := range vpcs {
		vpcs[i].region = self
		ret = append(ret, &vpcs[i])
	}
	return ret, nil
}

func (self *SVpc) CreateIWire(opts *cloudprovider.SWireCreateOptions) (cloudprovider.ICloudWire, error) {
	wire, err := self.region.CreateWire(opts, self.Id, self.DomainId, self.PublicScope, self.IsPublic)
	if err != nil {
		return nil, err
	}
	wire.vpc = self
	return wire, nil
}

func (self *SRegion) CreateWire(opts *cloudprovider.SWireCreateOptions, vpcId, domainId, publicScope string, isPublic bool) (*SWire, error) {
	input := api.WireCreateInput{}
	input.GenerateName = opts.Name
	input.Mtu = opts.Mtu
	input.Bandwidth = opts.Bandwidth
	input.VpcId = vpcId
	input.DomainId = domainId
	input.PublicScope = publicScope
	input.IsPublic = &isPublic
	input.ZoneId = opts.ZoneId
	t := true
	input.IsEmulated = &t
	wire := &SWire{}
	return wire, self.create(&modules.Wires, input, wire)
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	return []cloudprovider.ICloudSecurityGroup{}, nil
}

func (self *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	input := api.VpcCreateInput{}
	input.Name = opts.NAME
	input.Description = opts.Desc
	input.CidrBlock = opts.CIDR
	input.CloudregionId = self.Id
	vpc := &SVpc{region: self}
	return vpc, self.create(&modules.Vpcs, input, vpc)
}

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	vpc, err := self.GetVpc(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpc(%s)", id)
	}
	return vpc, nil
}

func (self *SRegion) GetVpcs() ([]SVpc, error) {
	vpcs := []SVpc{}
	return vpcs, self.list(&modules.Vpcs, nil, &vpcs)
}

func (self *SRegion) GetVpc(id string) (*SVpc, error) {
	vpc := &SVpc{region: self}
	return vpc, self.cli.get(&modules.Vpcs, id, nil, vpc)
}
