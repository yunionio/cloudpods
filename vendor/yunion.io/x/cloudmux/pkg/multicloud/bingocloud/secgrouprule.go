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

package bingocloud

import (
	"fmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"
)

type IPPermissions struct {
	direction secrules.TSecurityRuleDirection

	BoundType   string `json:"boundType"`
	Description string `json:"description"`
	FromPort    int    `json:"fromPort"`
	IPProtocol  string `json:"ipProtocol"`
	Groups      []struct {
		GroupId   string
		GroupName string
	} `json:"groups"`
	IPRanges []struct {
		CIDRIP string `json:"cidrIp"`
	} `json:"ipRanges"`
	L2Accept     string `json:"l2Accept"`
	PermissionId string `json:"permissionId"`
	Policy       string `json:"policy"`
	ToPort       int    `json:"toPort"`
}

func (self *IPPermissions) GetGlobalId() string {
	return self.PermissionId
}

func (self *IPPermissions) GetDescription() string {
	return self.Description
}

func (self *IPPermissions) GetAction() secrules.TSecurityRuleAction {
	if self.Policy == "DROP" {
		return secrules.SecurityRuleDeny
	}
	return secrules.SecurityRuleAllow
}

func (self *IPPermissions) GetProtocol() string {
	protocol := secrules.PROTO_ANY
	if self.IPProtocol != "all" {
		protocol = self.IPProtocol
	}
	return protocol
}

func (self *IPPermissions) GetPorts() string {
	if self.GetProtocol() == secrules.PROTO_TCP || self.GetProtocol() == secrules.PROTO_UDP {
		return fmt.Sprintf("%d-%d", self.FromPort, self.ToPort)
	}
	return ""
}

func (self *IPPermissions) GetPriority() int {
	return 0
}

func (self *IPPermissions) GetCIDRs() []string {
	nets := []string{}
	for _, ip := range self.IPRanges {
		nets = append(nets, ip.CIDRIP)
	}
	return nets
}

func (self *IPPermissions) GetDirection() secrules.TSecurityRuleDirection {
	return self.direction
}

func (self *IPPermissions) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *IPPermissions) Update(opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	return cloudprovider.ErrNotImplemented
}
