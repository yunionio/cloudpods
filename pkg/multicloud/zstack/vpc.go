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

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SVpc struct {
	multicloud.SVpc

	region *SRegion

	iwires []cloudprovider.ICloudWire
}

func (vpc *SVpc) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (vpc *SVpc) GetId() string {
	return fmt.Sprintf("%s/vpc", vpc.region.GetGlobalId())
}

func (vpc *SVpc) GetName() string {
	return fmt.Sprintf("%s-VPC", vpc.region.client.providerName)
}

func (vpc *SVpc) GetGlobalId() string {
	return vpc.GetId()
}

func (vpc *SVpc) IsEmulated() bool {
	return true
}

func (vpc *SVpc) GetIsDefault() bool {
	return true
}

func (vpc *SVpc) GetCidrBlock() string {
	return ""
}

func (vpc *SVpc) GetStatus() string {
	return api.VPC_STATUS_AVAILABLE
}

func (vpc *SVpc) Refresh() error {
	return nil
}

func (vpc *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return vpc.region
}

func (vpc *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if vpc.iwires == nil || len(vpc.iwires) == 0 {
		vpc.iwires = []cloudprovider.ICloudWire{}
		wires, err := vpc.region.GetWires("", "", "")
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(wires); i++ {
			wires[i].vpc = vpc
			vpc.iwires = append(vpc.iwires, &wires[i])
		}
	}
	return vpc.iwires, nil
}

func (vpc *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	wire, err := vpc.region.GetWire(wireId)
	if err != nil {
		return nil, err
	}
	wire.vpc = vpc
	return wire, nil
}

func (vpc *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := vpc.region.GetSecurityGroups("", "")
	if err != nil {
		return nil, err
	}
	isecgroups := []cloudprovider.ICloudSecurityGroup{}
	for i := 0; i < len(secgroups); i++ {
		isecgroups = append(isecgroups, &secgroups[i])
	}
	return isecgroups, nil
}

func (vpc *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (vpc *SVpc) Delete() error {
	return nil
}
