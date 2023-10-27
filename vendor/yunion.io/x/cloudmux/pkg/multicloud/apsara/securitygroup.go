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

package apsara

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

// {"CreationTime":"2017-03-19T13:37:48Z","Description":"System created security group.","SecurityGroupId":"sg-j6cannq0xxj2r9z0yxwl","SecurityGroupName":"sg-j6cannq0xxj2r9z0yxwl","Tags":{"Tag":[]},"VpcId":"vpc-j6c86z3sh8ufhgsxwme0q"}
// {"Description":"System created security group.","InnerAccessPolicy":"Accept","Permissions":{"Permission":[{"CreateTime":"2017-03-19T13:37:54Z","Description":"","DestCidrIp":"","DestGroupId":"","DestGroupName":"","DestGroupOwnerAccount":"","Direction":"ingress","IpProtocol":"ALL","NicType":"intranet","Policy":"Accept","PortRange":"-1/-1","Priority":110,"SourceCidrIp":"0.0.0.0/0","SourceGroupId":"","SourceGroupName":"","SourceGroupOwnerAccount":""},{"CreateTime":"2017-03-19T13:37:55Z","Description":"","DestCidrIp":"0.0.0.0/0","DestGroupId":"","DestGroupName":"","DestGroupOwnerAccount":"","Direction":"egress","IpProtocol":"ALL","NicType":"intranet","Policy":"Accept","PortRange":"-1/-1","Priority":110,"SourceCidrIp":"","SourceGroupId":"","SourceGroupName":"","SourceGroupOwnerAccount":""}]},"RegionId":"cn-hongkong","RequestId":"FBFE0950-5F2D-40DE-8C3C-E5A62AE7F7DA","SecurityGroupId":"sg-j6cannq0xxj2r9z0yxwl","SecurityGroupName":"sg-j6cannq0xxj2r9z0yxwl","VpcId":"vpc-j6c86z3sh8ufhgsxwme0q"}

type SecurityGroupPermissionNicType string

const (
	IntranetNicType SecurityGroupPermissionNicType = "intranet"
	InternetNicType SecurityGroupPermissionNicType = "internet"
)

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
	multicloud.SSecurityGroup
	ApsaraTags

	region            *SRegion
	CreationTime      time.Time
	Description       string
	SecurityGroupId   string
	SecurityGroupName string
	VpcId             string
	InnerAccessPolicy string
	Permissions       SPermissions
	RegionId          string
	Tags              Tags

	DepartmentInfo
}

func (self *SSecurityGroup) GetVpcId() string {
	return self.VpcId
}

func (self *SSecurityGroup) GetTags() (map[string]string, error) {
	tags := map[string]string{}
	for _, value := range self.Tags.Tag {
		tags[value.TagKey] = value.TagValue
	}
	return tags, nil
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

func (self *SSecurityGroup) GetCreatedAt() time.Time {
	return self.CreationTime
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	ret := make([]cloudprovider.ISecurityGroupRule, 0)
	rules, err := self.region.GetSecurityGroupRules(self.SecurityGroupId)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		rules[i].region = self.region
		ret = append(ret, &rules[i])
	}
	return ret, nil
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
	group, err := self.region.GetSecurityGroup(self.SecurityGroupId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, group)
}

func (self *SRegion) GetSecurityGroup(id string) (*SSecurityGroup, error) {
	groups, _, err := self.GetSecurityGroups("", "", []string{id}, 0, 1)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].SecurityGroupId == id {
			groups[i].region = self
			return &groups[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
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

func (self *SRegion) CreateSecurityGroup(vpcId string, name string, desc, projectId string) (string, error) {
	params := make(map[string]string)
	if len(vpcId) > 0 {
		params["VpcId"] = vpcId
	}

	if len(projectId) > 0 {
		params["ResourceGroupId"] = projectId
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
		return "", errors.Wrap(err, "CreateSecurityGroup")
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

func (self *SRegion) AddSecurityGroupRules(secGrpId string, rule secrules.SecurityRule) error {
	if len(rule.Ports) != 0 {
		for _, port := range rule.Ports {
			rule.PortStart, rule.PortEnd = port, port
			err := self.addSecurityGroupRule(secGrpId, rule)
			if err != nil {
				return errors.Wrapf(err, "addSecurityGroupRule %s", rule.String())
			}
		}
		return nil
	}
	return self.addSecurityGroupRule(secGrpId, rule)
}

func (self *SRegion) addSecurityGroupRule(secGrpId string, rule secrules.SecurityRule) error {
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

	// 忽略地址为0.0.0.0/32这样的阿里云规则
	if rule.IPNet.IP.String() == "0.0.0.0" && rule.IPNet.String() != "0.0.0.0/0" {
		return nil
	}

	params["Priority"] = fmt.Sprintf("%d", rule.Priority)
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

func (self *SRegion) DelSecurityGroupRule(secGrpId string, rule secrules.SecurityRule) error {
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
	return self.region.DeleteSecurityGroup(self.SecurityGroupId)
}
