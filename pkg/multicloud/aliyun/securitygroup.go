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
	"sort"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"
)

// {"CreationTime":"2017-03-19T13:37:48Z","Description":"System created security group.","SecurityGroupId":"sg-j6cannq0xxj2r9z0yxwl","SecurityGroupName":"sg-j6cannq0xxj2r9z0yxwl","Tags":{"Tag":[]},"VpcId":"vpc-j6c86z3sh8ufhgsxwme0q"}
// {"Description":"System created security group.","InnerAccessPolicy":"Accept","Permissions":{"Permission":[{"CreateTime":"2017-03-19T13:37:54Z","Description":"","DestCidrIp":"","DestGroupId":"","DestGroupName":"","DestGroupOwnerAccount":"","Direction":"ingress","IpProtocol":"ALL","NicType":"intranet","Policy":"Accept","PortRange":"-1/-1","Priority":110,"SourceCidrIp":"0.0.0.0/0","SourceGroupId":"","SourceGroupName":"","SourceGroupOwnerAccount":""},{"CreateTime":"2017-03-19T13:37:55Z","Description":"","DestCidrIp":"0.0.0.0/0","DestGroupId":"","DestGroupName":"","DestGroupOwnerAccount":"","Direction":"egress","IpProtocol":"ALL","NicType":"intranet","Policy":"Accept","PortRange":"-1/-1","Priority":110,"SourceCidrIp":"","SourceGroupId":"","SourceGroupName":"","SourceGroupOwnerAccount":""}]},"RegionId":"cn-hongkong","RequestId":"FBFE0950-5F2D-40DE-8C3C-E5A62AE7F7DA","SecurityGroupId":"sg-j6cannq0xxj2r9z0yxwl","SecurityGroupName":"sg-j6cannq0xxj2r9z0yxwl","VpcId":"vpc-j6c86z3sh8ufhgsxwme0q"}

type SecurityGroupPermissionNicType string

const (
	IntranetNicType SecurityGroupPermissionNicType = "intranet"
	InternetNicType SecurityGroupPermissionNicType = "internet"
)

type SPermission struct {
	CreateTime              time.Time
	Description             string
	DestCidrIp              string
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
	SourceGroupId           string
	SourceGroupName         string
	SourceGroupOwnerAccount string
}

type SPermissions struct {
	Permission []SPermission
}

type Tags struct {
	Tag []Tag
}

type Tag struct {
	TagKey   string
	TagValue string
}

type SSecurityGroup struct {
	vpc               *SVpc
	CreationTime      time.Time
	Description       string
	SecurityGroupId   string
	SecurityGroupName string
	VpcId             string
	InnerAccessPolicy string
	Permissions       SPermissions
	RegionId          string
	Tags              Tags
}

type PermissionSet []SPermission

func (v PermissionSet) Len() int {
	return len(v)
}

func (v PermissionSet) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v PermissionSet) Less(i, j int) bool {
	if v[i].Priority < v[j].Priority {
		return true
	} else if v[i].Priority == v[j].Priority {
		return strings.Compare(v[i].String(), v[j].String()) <= 0
	}
	return false
}

func (self *SSecurityGroup) GetVpcId() string {
	return self.VpcId
}

func (self *SSecurityGroup) GetMetadata() *jsonutils.JSONDict {
	if len(self.Tags.Tag) == 0 {
		return nil
	}
	data := jsonutils.NewDict()
	for _, value := range self.Tags.Tag {
		data.Add(jsonutils.NewString(value.TagValue), value.TagKey)
	}
	return data
}

func (self *SSecurityGroup) GetId() string {
	return self.SecurityGroupId
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.SecurityGroupId
}

func (self *SSecurityGroup) GetDescription() string {
	return self.Description
}

func (self *SSecurityGroup) GetRules() ([]secrules.SecurityRule, error) {
	rules := make([]secrules.SecurityRule, 0)
	if secgrp, err := self.vpc.region.GetSecurityGroupDetails(self.SecurityGroupId); err != nil {
		return rules, err
	} else {
		for _, permission := range secgrp.Permissions.Permission {
			if rule, err := secrules.ParseSecurityRule(permission.String()); err != nil {
				return rules, err
			} else {
				priority := permission.Priority
				if priority > 100 {
					priority = 100
				}
				rule.Priority = 101 - priority
				rule.Description = permission.Description
				rules = append(rules, *rule)
			}
		}
	}
	return rules, nil
}

func (self *SSecurityGroup) GetName() string {
	if len(self.SecurityGroupName) > 0 {
		return self.SecurityGroupName
	}
	return self.SecurityGroupId
}

func (self *SSecurityGroup) GetStatus() string {
	return ""
}

func (self *SSecurityGroup) IsEmulated() bool {
	return false
}

func (self *SSecurityGroup) Refresh() error {
	if new, err := self.vpc.region.GetSecurityGroupDetails(self.SecurityGroupId); err != nil {
		return err
	} else {
		return jsonutils.Update(self, new)
	}
}

func (self *SRegion) GetSecurityGroups(vpcId, name string, securityGroupIds []string, offset int, limit int) ([]SSecurityGroup, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)
	if len(vpcId) > 0 {
		params["VpcId"] = vpcId
	}
	if len(name) > 0 {
		params["SecurityGroupName"] = name
	}

	if securityGroupIds != nil && len(securityGroupIds) > 0 {
		params["SecurityGroupIds"] = jsonutils.Marshal(securityGroupIds).String()
	}

	body, err := self.ecsRequest("DescribeSecurityGroups", params)
	if err != nil {
		log.Errorf("GetSecurityGroups fail %s", err)
		return nil, 0, err
	}

	secgrps := make([]SSecurityGroup, 0)
	err = body.Unmarshal(&secgrps, "SecurityGroups", "SecurityGroup")
	if err != nil {
		log.Errorf("Unmarshal security groups fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	return secgrps, int(total), nil
}

func (self *SRegion) GetSecurityGroupDetails(secGroupId string) (*SSecurityGroup, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["SecurityGroupId"] = secGroupId

	body, err := self.ecsRequest("DescribeSecurityGroupAttribute", params)
	if err != nil {
		log.Errorf("DescribeSecurityGroupAttribute fail %s", err)
		return nil, err
	}

	log.Debugf("%s", body)
	secgrp := SSecurityGroup{}
	err = body.Unmarshal(&secgrp)
	if err != nil {
		log.Errorf("Unmarshal security group details fail %s", err)
		return nil, err
	}
	return &secgrp, nil
}

func (self *SRegion) CreateSecurityGroup(vpcId string, name string, desc string) (string, error) {
	params := make(map[string]string)
	if len(vpcId) > 0 {
		params["VpcId"] = vpcId
	}

	if name == "Default" {
		name = "Default-copy"
	}

	if len(name) > 0 {
		params["SecurityGroupName"] = name
	}
	if len(desc) > 0 {
		params["Description"] = desc
	}
	params["ClientToken"] = utils.GenRequestId(20)

	body, err := self.ecsRequest("CreateSecurityGroup", params)
	if err != nil {
		return "", err
	}
	return body.GetString("SecurityGroupId")
}

func (self *SRegion) modifySecurityGroupRule(secGrpId string, rule *secrules.SecurityRule) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["SecurityGroupId"] = secGrpId
	params["NicType"] = string(IntranetNicType)
	params["Description"] = rule.Description
	params["PortRange"] = fmt.Sprintf("%d/%d", rule.PortStart, rule.PortEnd)
	protocol := rule.Protocol
	if len(rule.Protocol) == 0 || rule.Protocol == secrules.PROTO_ANY {
		protocol = "all"
	}
	params["IpProtocol"] = protocol
	if rule.PortStart < 1 && rule.PortEnd < 1 {
		if protocol == "udp" || protocol == "tcp" {
			params["PortRange"] = "1/65535"
		} else {
			params["PortRange"] = "-1/-1"
		}
	}
	if rule.Action == secrules.SecurityRuleAllow {
		params["Policy"] = "accept"
	} else {
		params["Policy"] = "drop"
	}
	params["Priority"] = fmt.Sprintf("%d", rule.Priority)
	if rule.Direction == secrules.SecurityRuleIngress {
		if rule.IPNet != nil {
			params["SourceCidrIp"] = rule.IPNet.String()
		} else {
			params["SourceCidrIp"] = "0.0.0.0/0"
		}
		_, err := self.ecsRequest("ModifySecurityGroupRule", params)
		return err
	} else { // rule.Direction == secrules.SecurityRuleEgress {
		//阿里云不支持出方向API接口调用
		return nil
		// if rule.IPNet != nil {
		// 	params["DestCidrIp"] = rule.IPNet.String()
		// } else {
		// 	params["DestCidrIp"] = "0.0.0.0/0"
		// }
		// _, err := self.ecsRequest("ModifySecurityGroupRule", params)
		// return err
	}
}

func (self *SRegion) modifySecurityGroup(secGrpId string, name string, desc string) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["SecurityGroupId"] = secGrpId
	params["SecurityGroupName"] = name
	if len(desc) > 0 {
		params["Description"] = desc
	}
	_, err := self.ecsRequest("ModifySecurityGroupAttribute", params)
	return err
}

func (self *SRegion) addSecurityGroupRules(secGrpId string, rule *secrules.SecurityRule) error {
	if len(rule.Ports) != 0 {
		for _, port := range rule.Ports {
			rule.PortStart, rule.PortEnd = port, port
			if err := self.addSecurityGroupRule(secGrpId, rule); err != nil {
				return err
			}
		}
	} else {
		return self.addSecurityGroupRule(secGrpId, rule)
	}
	return nil
}

func (self *SRegion) addSecurityGroupRule(secGrpId string, rule *secrules.SecurityRule) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["SecurityGroupId"] = secGrpId
	params["NicType"] = string(IntranetNicType)
	params["Description"] = rule.Description
	params["PortRange"] = fmt.Sprintf("%d/%d", rule.PortStart, rule.PortEnd)
	protocol := rule.Protocol
	if len(rule.Protocol) == 0 || rule.Protocol == secrules.PROTO_ANY {
		protocol = "all"
	}
	params["IpProtocol"] = protocol
	if rule.PortStart < 1 && rule.PortEnd < 1 {
		if protocol == "udp" || protocol == "tcp" {
			params["PortRange"] = "1/65535"
		} else {
			params["PortRange"] = "-1/-1"
		}
	}
	if rule.Action == secrules.SecurityRuleAllow {
		params["Policy"] = "accept"
	} else {
		params["Policy"] = "drop"
	}
	params["Priority"] = fmt.Sprintf("%d", 101-rule.Priority)
	if rule.Direction == secrules.SecurityRuleIngress {
		if rule.IPNet != nil {
			params["SourceCidrIp"] = rule.IPNet.String()
		} else {
			params["SourceCidrIp"] = "0.0.0.0/0"
		}
		_, err := self.ecsRequest("AuthorizeSecurityGroup", params)
		return err
	} else { // rule.Direction == secrules.SecurityRuleEgress {
		if rule.IPNet != nil {
			params["DestCidrIp"] = rule.IPNet.String()
		} else {
			params["DestCidrIp"] = "0.0.0.0/0"
		}
		_, err := self.ecsRequest("AuthorizeSecurityGroupEgress", params)
		return err
	}
}

func (self *SRegion) delSecurityGroupRule(secGrpId string, rule *secrules.SecurityRule) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["SecurityGroupId"] = secGrpId
	params["NicType"] = string(IntranetNicType)
	params["PortRange"] = fmt.Sprintf("%d/%d", rule.PortStart, rule.PortEnd)
	protocol := rule.Protocol
	if len(rule.Protocol) == 0 || rule.Protocol == secrules.PROTO_ANY {
		protocol = "all"
	}
	params["IpProtocol"] = protocol
	if rule.PortStart < 1 && rule.PortEnd < 1 {
		if protocol == "udp" || protocol == "tcp" {
			params["PortRange"] = "1/65535"
		} else {
			params["PortRange"] = "-1/-1"
		}
	}
	if rule.Action == secrules.SecurityRuleAllow {
		params["Policy"] = "accept"
	} else {
		params["Policy"] = "drop"
	}
	params["Priority"] = fmt.Sprintf("%d", rule.Priority)
	if rule.Direction == secrules.SecurityRuleIngress {
		if rule.IPNet != nil {
			params["SourceCidrIp"] = rule.IPNet.String()
		} else {
			params["SourceCidrIp"] = "0.0.0.0/0"
		}
		_, err := self.ecsRequest("RevokeSecurityGroup", params)
		return err
	} else { // rule.Direction == secrules.SecurityRuleEgress {
		if rule.IPNet != nil {
			params["DestCidrIp"] = rule.IPNet.String()
		} else {
			params["DestCidrIp"] = "0.0.0.0/0"
		}
		_, err := self.ecsRequest("RevokeSecurityGroupEgress", params)
		return err
	}
}

func (self *SPermission) String() string {
	action := secrules.SecurityRuleDeny
	if strings.ToLower(self.Policy) == "accept" {
		action = secrules.SecurityRuleAllow
	}
	direction := "in"
	if self.Direction == "egress" {
		direction = "out"
	}
	cidr := self.SourceCidrIp
	if direction == "out" {
		cidr = self.DestCidrIp
	}
	if cidr == "0.0.0.0/0" {
		cidr = ""
	}
	protocol := strings.ToLower(self.IpProtocol)
	if protocol == "all" {
		protocol = "any"
	}
	port, ports := "", strings.Split(self.PortRange, "/")
	if ports[0] == ports[1] {
		if ports[0] != "-1" {
			port = ports[0]
		}
	} else if ports[0] != "1" && ports[1] != "65535" {
		port = fmt.Sprintf("%s-%s", ports[0], ports[1])
	}
	result := fmt.Sprintf("%s:%s", direction, string(action))
	if len(cidr) > 0 {
		result += fmt.Sprintf(" %s", cidr)
	}
	result += fmt.Sprintf(" %s", protocol)
	if len(port) > 0 {
		result += fmt.Sprintf(" %s", port)
	}
	return result
}

func (self *SRegion) syncSecgroupRules(secgroupId string, rules []secrules.SecurityRule) error {
	if secgroup, err := self.GetSecurityGroupDetails(secgroupId); err != nil {
		return err
	} else {

		sort.Sort(secrules.SecurityRuleSet(rules))
		sort.Sort(PermissionSet(secgroup.Permissions.Permission))

		i, j := 0, 0
		for i < len(rules) || j < len(secgroup.Permissions.Permission) {
			if i < len(rules) && j < len(secgroup.Permissions.Permission) {
				permissionStr := secgroup.Permissions.Permission[j].String()
				ruleStr := rules[i].String()
				cmp := strings.Compare(permissionStr, ruleStr)
				if cmp == 0 {
					if secgroup.Permissions.Permission[j].Description != rules[i].Description {
						rules[i].Priority = secgroup.Permissions.Permission[j].Priority
						if err := self.modifySecurityGroupRule(secgroupId, &rules[i]); err != nil {
							log.Errorf("modifySecurityGroupRule error %v", rules[i])
							return err
						}
					}
					i += 1
					j += 1
				} else if cmp > 0 {
					if rule, err := secrules.ParseSecurityRule(permissionStr); err != nil {
						return err
					} else {
						rule.Priority = secgroup.Permissions.Permission[j].Priority
						if err := self.delSecurityGroupRule(secgroupId, rule); err != nil {
							log.Errorf("delSecurityGroupRule error %v", rule)
							return err
						}
					}
					j += 1
				} else {
					if err := self.addSecurityGroupRules(secgroupId, &rules[i]); err != nil {
						log.Errorf("addSecurityGroupRule error %v", rules[i])
						return err
					}
					i += 1
				}
			} else if i >= len(rules) {
				permissionStr := secgroup.Permissions.Permission[j].String()
				if rule, err := secrules.ParseSecurityRule(permissionStr); err != nil {
					return err
				} else {
					rule.Priority = secgroup.Permissions.Permission[j].Priority
					if err := self.delSecurityGroupRule(secgroupId, rule); err != nil {
						log.Errorf("delSecurityGroupRule error %v", rule)
						return err
					}
				}
				j += 1
			} else if j >= len(secgroup.Permissions.Permission) {
				if err := self.addSecurityGroupRules(secgroupId, &rules[i]); err != nil {
					log.Errorf("addSecurityGroupRule error %v", rules[i])
					return err
				}
				i += 1
			}
		}
	}
	return nil
}

func (self *SRegion) AssignSecurityGroup(secgroupId, instanceId string) error {
	return self.SetSecurityGroups([]string{secgroupId}, instanceId)
}

func (self *SRegion) SetSecurityGroups(secgroupIds []string, instanceId string) error {
	params := map[string]string{"InstanceId": instanceId}
	for _, secgroupId := range secgroupIds {
		params["SecurityGroupId"] = secgroupId
		if _, err := self.ecsRequest("JoinSecurityGroup", params); err != nil {
			return err
		}
	}
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return err
	}
	for _, _secgroupId := range instance.SecurityGroupIds.SecurityGroupId {
		if !utils.IsInStringArray(_secgroupId, secgroupIds) {
			if err := self.leaveSecurityGroup(_secgroupId, instanceId); err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *SRegion) leaveSecurityGroup(secgroupId, instanceId string) error {
	params := map[string]string{"InstanceId": instanceId, "SecurityGroupId": secgroupId}
	_, err := self.ecsRequest("LeaveSecurityGroup", params)
	return err
}

func (self *SRegion) DeleteSecurityGroup(secGrpId string) error {
	params := make(map[string]string)
	params["SecurityGroupId"] = secGrpId

	_, err := self.ecsRequest("DeleteSecurityGroup", params)
	if err != nil {
		log.Errorf("Delete security group fail %s", err)
		return err
	}
	return nil
}

func (self *SSecurityGroup) Delete() error {
	return self.vpc.region.DeleteSecurityGroup(self.SecurityGroupId)
}

func (self *SSecurityGroup) GetProjectId() string {
	return ""
}

func (self *SSecurityGroup) SyncRules(rules []secrules.SecurityRule) error {
	return self.vpc.region.syncSecgroupRules(self.SecurityGroupId, rules)
}
