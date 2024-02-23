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

package aliyun

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"
)

type SPermission struct {
	region *SRegion

	CreateTime              time.Time
	Description             string
	DestCidrIp              string
	Ipv6DestCidrIp          string
	DestGroupId             string
	DestGroupName           string
	DestGroupOwnerAccount   string
	Direction               string
	IpProtocol              string
	NicType                 SecurityGroupPermissionNicType
	Policy                  string
	PortRange               string
	Priority                int
	SourceCidrIp            string
	Ipv6SourceCidrIp        string
	SourceGroupId           string
	SourceGroupName         string
	SourceGroupOwnerAccount string
	SecurityGroupRuleId     string
	SecurityGroupId         string
}

func (self *SPermission) GetGlobalId() string {
	return self.SecurityGroupRuleId
}

func (self *SPermission) GetAction() secrules.TSecurityRuleAction {
	if self.Policy == "Drop" {
		return secrules.SecurityRuleDeny
	}
	return secrules.SecurityRuleAllow
}

func (self *SPermission) GetDescription() string {
	return self.Description
}

func (self *SPermission) GetDirection() secrules.TSecurityRuleDirection {
	if self.Direction == "ingress" {
		return secrules.DIR_IN
	}
	return secrules.DIR_OUT
}

func (self *SPermission) GetCIDRs() []string {
	ret := []string{}
	if len(self.SourceCidrIp) > 0 {
		ret = append(ret, self.SourceCidrIp)
	}
	if len(self.SourceGroupId) > 0 {
		ret = append(ret, self.SourceGroupId)
	}
	if len(self.DestGroupId) > 0 {
		ret = append(ret, self.SourceGroupId)
	}
	if len(self.DestCidrIp) > 0 {
		ret = append(ret, self.DestCidrIp)
	}
	if len(self.Ipv6DestCidrIp) > 0 {
		ret = append(ret, self.Ipv6DestCidrIp)
	}
	if len(self.Ipv6SourceCidrIp) > 0 {
		ret = append(ret, self.Ipv6SourceCidrIp)
	}
	return ret
}

func (self *SPermission) GetProtocol() string {
	if strings.ToLower(self.IpProtocol) == "all" {
		return secrules.PROTO_ANY
	}
	return strings.ToLower(self.IpProtocol)
}

func (self *SPermission) GetPorts() string {
	if self.PortRange == "-1/-1" || self.PortRange == "1/65535" || self.PortRange == "" {
		return ""
	}
	info := strings.Split(self.PortRange, "/")
	if len(info) != 2 {
		return ""
	}
	if info[0] == info[1] {
		if info[0] == "-1" {
			return ""
		}
		return info[0]
	}
	return fmt.Sprintf("%s-%s", info[0], info[1])
}

func (self *SPermission) GetPriority() int {
	return self.Priority
}

func (self *SPermission) Delete() error {
	return self.region.DeleteSecurityGroupRule(self.SecurityGroupId, self.GetDirection(), self.SecurityGroupRuleId)
}

func (self *SPermission) Update(opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	return self.region.UpdateSecurityGroupRule(self.SecurityGroupId, self.SecurityGroupRuleId, self.GetDirection(), opts.Desc)
}

func (self *SRegion) GetSecurityGroupRules(id string) ([]SPermission, error) {
	params := map[string]string{
		"SecurityGroupId": id,
		"RegionId":        self.RegionId,
	}
	resp, err := self.ecsRequest("DescribeSecurityGroupAttribute", params)
	if err != nil {
		return nil, err
	}
	ret := struct {
		Permissions struct {
			Permission []SPermission
		}
		SecurityGroupId string
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	for i := range ret.Permissions.Permission {
		ret.Permissions.Permission[i].SecurityGroupId = ret.SecurityGroupId
	}
	return ret.Permissions.Permission, nil
}

func (self *SRegion) DeleteSecurityGroupRule(groupId string, direction secrules.TSecurityRuleDirection, ruleId string) error {
	action := "RevokeSecurityGroup"
	if direction == secrules.DIR_OUT {
		action = "RevokeSecurityGroupEgress"
	}
	params := map[string]string{
		"RegionId":              self.RegionId,
		"ClientToken":           utils.GenRequestId(20),
		"SecurityGroupId":       groupId,
		"SecurityGroupRuleId.1": ruleId,
	}
	_, err := self.ecsRequest(action, params)
	return err
}

func (self *SRegion) UpdateSecurityGroupRule(groupId, ruleId string, direction secrules.TSecurityRuleDirection, desc string) error {
	params := map[string]string{
		"ClientToken":         utils.GenRequestId(20),
		"SecurityGroupId":     groupId,
		"SecurityGroupRuleId": ruleId,
	}
	action := "ModifySecurityGroupRule"
	if direction == secrules.DIR_OUT {
		action = "ModifySecurityGroupEgressRule"
	}
	_, err := self.ecsRequest(action, params)
	return err
}
