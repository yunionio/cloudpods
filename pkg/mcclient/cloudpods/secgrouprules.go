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
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SecurityGroupRule struct {
	multicloud.SResourceBase
	CloudpodsTags
	region *SRegion

	api.SecgroupRuleDetails
}

func (self *SecurityGroupRule) GetGlobalId() string {
	return self.Id
}

func (self *SecurityGroupRule) GetAction() secrules.TSecurityRuleAction {
	return secrules.TSecurityRuleAction(self.Action)
}

func (self *SecurityGroupRule) GetDescription() string {
	return self.Description
}

func (self *SecurityGroupRule) GetDirection() secrules.TSecurityRuleDirection {
	return secrules.TSecurityRuleDirection(self.Direction)
}

func (self *SecurityGroupRule) GetCIDRs() []string {
	return []string{self.CIDR}
}

func (self *SecurityGroupRule) GetProtocol() string {
	return self.Protocol
}

func (self *SecurityGroupRule) GetPorts() string {
	return self.Ports
}

func (self *SecurityGroupRule) GetPriority() int {
	return int(self.Priority)
}

func (self *SecurityGroupRule) Delete() error {
	return self.region.cli.delete(&modules.SecGroupRules, self.Id)
}

func (self *SecurityGroupRule) Update(opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	return cloudprovider.ErrNotImplemented
}
