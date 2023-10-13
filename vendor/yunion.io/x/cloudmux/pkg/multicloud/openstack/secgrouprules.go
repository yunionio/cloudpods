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

package openstack

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"
)

type SSecurityGroupRule struct {
	region *SRegion

	Direction       string
	Ethertype       string
	Id              string
	PortRangeMax    int
	PortRangeMin    int
	Protocol        string
	RemoteGroupId   string
	RemoteIpPrefix  string
	SecurityGroupId string
	ProjectId       string
	RevisionNumber  int
	Tags            []string
	TenantId        string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Description     string
}

func (self *SSecurityGroupRule) GetGlobalId() string {
	return self.Id
}

func (self *SSecurityGroupRule) GetAction() secrules.TSecurityRuleAction {
	return secrules.SecurityRuleAllow
}

func (self *SSecurityGroupRule) GetDescription() string {
	return self.Description
}

func (self *SSecurityGroupRule) GetDirection() secrules.TSecurityRuleDirection {
	if self.Direction == "egress" {
		return secrules.DIR_OUT
	}
	return secrules.DIR_IN
}

func (self *SSecurityGroupRule) GetCIDRs() []string {
	ip := self.RemoteIpPrefix + self.RemoteGroupId
	if len(ip) == 0 {
		ip = "0.0.0.0/0"
	}
	ret := []string{ip}
	return ret
}

func (self *SSecurityGroupRule) GetProtocol() string {
	if len(self.Protocol) == 0 || self.Protocol == "-1" {
		return secrules.PROTO_ANY
	}
	strings.ReplaceAll(self.Protocol, "6", "tcp")
	strings.ReplaceAll(self.Protocol, "17", "udp")
	strings.ReplaceAll(self.Protocol, "1", "icmp")
	return self.Protocol
}

func (self *SSecurityGroupRule) GetPorts() string {
	if self.PortRangeMax > 0 && self.PortRangeMin > 0 {
		return fmt.Sprintf("%d-%d", self.PortRangeMin, self.PortRangeMax)
	}
	return ""
}

func (self *SSecurityGroupRule) GetPriority() int {
	return 0
}

func (self *SSecurityGroupRule) Delete() error {
	return self.region.DeleteSecurityGroupRule(self.Id)
}

func (self *SSecurityGroupRule) Update(opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	return cloudprovider.ErrNotImplemented
}
