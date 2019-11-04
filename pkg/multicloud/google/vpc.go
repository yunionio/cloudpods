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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SVpc struct {
	multicloud.SVpc
	globalnetwork *SGlobalNetwork

	region *SRegion
}

func (vpc *SVpc) GetName() string {
	return fmt.Sprintf("%s(%s)", vpc.globalnetwork.Name, vpc.region.Name)
}

func (vpc *SVpc) GetId() string {
	return getGlobalId(vpc.globalnetwork.SelfLink)
}

func (vpc *SVpc) GetGlobalId() string {
	return vpc.GetId()
}

func (vpc *SVpc) Refresh() error {
	return nil
}

func (vpc *SVpc) GetStatus() string {
	return api.VPC_STATUS_AVAILABLE
}

func (vpc *SVpc) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (vpc *SVpc) GetCidrBlock() string {
	return ""
}

func (vpc *SVpc) GetIGlobalNetworkId() string {
	return vpc.globalnetwork.GetGlobalId()
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

func (vpc *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	firewalls, err := vpc.region.GetFirewalls(vpc.globalnetwork.SelfLink, 0, "")
	if err != nil {
		return nil, errors.Wrap(err, "GetFirewalls")
	}
	isecgroups := []cloudprovider.ICloudSecurityGroup{}
	tags := []string{}
	allInstance := false
	for _, firewall := range firewalls {
		if len(firewall.TargetServiceAccounts) > 0 {
			secgroup := &SSecurityGroup{vpc: vpc, ServiceAccount: firewall.TargetServiceAccounts[0]}
			isecgroups = append(isecgroups, secgroup)
		} else if len(firewall.TargetTags) > 0 && !utils.IsInStringArray(firewall.TargetTags[0], tags) {
			secgroup := &SSecurityGroup{vpc: vpc, Tag: firewall.TargetTags[0]}
			tags = append(tags, firewall.TargetTags[0])
			isecgroups = append(isecgroups, secgroup)
		} else if !allInstance {
			secgroup := &SSecurityGroup{vpc: vpc}
			isecgroups = append(isecgroups, secgroup)
			allInstance = true
		}
	}
	return isecgroups, nil
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
