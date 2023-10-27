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
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"
)

type SSecurityGroupRule struct {
	region *SRegion

	ZStackBasic
	SecurityGroupUUID       string `json:"securityGroupUuid"`
	Type                    string `json:"type"`
	IPVersion               int    `json:"ipVersion"`
	StartPort               int    `json:"startPort"`
	EndPort                 int    `json:"endPort"`
	Protocol                string `json:"protocol"`
	State                   string `json:"state"`
	AllowedCIDR             string `json:"allowedCidr"`
	RemoteSecurityGroupUUID string `json:"remoteSecurityGroupUuid"`
	ZStackTime
}

func (self *SSecurityGroupRule) GetGlobalId() string {
	return self.UUID
}

func (self *SSecurityGroupRule) GetAction() secrules.TSecurityRuleAction {
	return secrules.SecurityRuleAllow
}

func (self *SSecurityGroupRule) GetDescription() string {
	return self.Description
}

func (self *SSecurityGroupRule) GetDirection() secrules.TSecurityRuleDirection {
	if self.Type == "Egress" {
		return secrules.DIR_OUT
	}
	return secrules.DIR_IN
}

func (self *SSecurityGroupRule) GetCIDRs() []string {
	ip := self.AllowedCIDR + self.RemoteSecurityGroupUUID
	if len(ip) == 0 {
		ip = "0.0.0.0/0"
	}
	ret := []string{ip}
	return ret
}

func (self *SSecurityGroupRule) GetProtocol() string {
	if self.Protocol == "ALL" {
		return secrules.PROTO_ANY
	}
	return strings.ToLower(self.Protocol)
}

func (self *SSecurityGroupRule) GetPorts() string {
	if self.StartPort > 0 && self.EndPort > 0 {
		return fmt.Sprintf("%d-%d", self.StartPort, self.EndPort)
	}
	return ""
}

func (self *SSecurityGroupRule) GetPriority() int {
	return 0
}

func (self *SSecurityGroupRule) Delete() error {
	return self.region.DeleteSecurityGroupRules([]string{self.UUID})
}

func (self *SSecurityGroupRule) Update(opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	return cloudprovider.ErrNotImplemented
}
