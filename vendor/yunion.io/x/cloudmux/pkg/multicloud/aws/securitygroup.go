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

package aws

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	AwsTags
	vpc *SVpc

	RegionId          string
	VpcId             string
	SecurityGroupId   string
	Description       string
	SecurityGroupName string
	Permissions       []cloudprovider.SecurityRule

	// CreationTime      time.Time
	// InnerAccessPolicy string
}

func randomString(prefix string, length int) string {
	bytes := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return prefix + string(result)
}

func (self *SSecurityGroup) GetId() string {
	return self.SecurityGroupId
}

func (self *SSecurityGroup) GetVpcId() string {
	return self.VpcId
}

func (self *SSecurityGroup) GetName() string {
	if len(self.SecurityGroupName) > 0 {
		return self.SecurityGroupName
	}
	return self.SecurityGroupId
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.SecurityGroupId
}

func (self *SSecurityGroup) GetStatus() string {
	return ""
}

func (self *SSecurityGroup) Refresh() error {
	if new, err := self.vpc.region.GetSecurityGroupDetails(self.SecurityGroupId); err != nil {
		return err
	} else {
		return jsonutils.Update(self, new)
	}
}

func (self *SSecurityGroup) IsEmulated() bool {
	return false
}

func (self *SSecurityGroup) GetDescription() string {
	return self.Description
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.SecurityRule, error) {
	secgrp, err := self.vpc.region.GetSecurityGroupDetails(self.SecurityGroupId)
	if err != nil {
		return nil, errors.Wrap(err, "GetSecurityGroupDetails")
	}
	return secgrp.Permissions, nil
}

func (self *SRegion) addSecurityGroupRules(secGrpId string, rule cloudprovider.SecurityRule) error {
	if len(rule.Ports) != 0 {
		for _, port := range rule.Ports {
			rule.PortStart, rule.PortEnd = port, port
			err := self.addSecurityGroupRule(secGrpId, rule)
			if err != nil {
				return errors.Wrapf(err, "addSecurityGroupRule(%d %s)", rule.Priority, rule.String())
			}
		}
		return nil
	}
	return self.addSecurityGroupRule(secGrpId, rule)
}

func (self *SRegion) addSecurityGroupRule(secGrpId string, rule cloudprovider.SecurityRule) error {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}

	ipPermissions, err := YunionSecRuleToAws(rule)
	log.Debugf("Aws security group rule: %s", ipPermissions)
	if err != nil {
		return err
	}

	if rule.Direction == secrules.SecurityRuleIngress {
		params := &ec2.AuthorizeSecurityGroupIngressInput{}
		params.SetGroupId(secGrpId)
		params.SetIpPermissions(ipPermissions)
		_, err = ec2Client.AuthorizeSecurityGroupIngress(params)
	}

	if rule.Direction == secrules.SecurityRuleEgress {
		params := &ec2.AuthorizeSecurityGroupEgressInput{}
		params.SetGroupId(secGrpId)
		params.SetIpPermissions(ipPermissions)
		_, err = ec2Client.AuthorizeSecurityGroupEgress(params)
	}

	if err != nil && strings.Contains(err.Error(), "InvalidPermission.Duplicate") {
		log.Debugf("addSecurityGroupRule %s %s", rule.Direction, err.Error())
		return nil
	}

	return err
}

func (self *SRegion) DelSecurityGroupRule(secGrpId string, rule cloudprovider.SecurityRule) error {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}

	ipPermissions, err := YunionSecRuleToAws(rule)
	if err != nil {
		return err
	}

	if rule.Direction == secrules.SecurityRuleIngress {
		params := &ec2.RevokeSecurityGroupIngressInput{}
		params.SetGroupId(secGrpId)
		params.SetIpPermissions(ipPermissions)
		_, err = ec2Client.RevokeSecurityGroupIngress(params)
	}

	if rule.Direction == secrules.SecurityRuleEgress {
		params := &ec2.RevokeSecurityGroupEgressInput{}
		params.SetGroupId(secGrpId)
		params.SetIpPermissions(ipPermissions)
		_, err = ec2Client.RevokeSecurityGroupEgress(params)
	}

	if err != nil {
		log.Debugf("delSecurityGroupRule %s %s", rule.Direction, err.Error())
		return err
	}
	return nil
}

func (self *SRegion) updateSecurityGroupRuleDescription(secGrpId string, rule cloudprovider.SecurityRule) error {
	ipPermissions, err := YunionSecRuleToAws(rule)
	if err != nil {
		return err
	}

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}

	if rule.Direction == secrules.SecurityRuleIngress {
		params := &ec2.UpdateSecurityGroupRuleDescriptionsIngressInput{}
		params.SetGroupId(secGrpId)
		params.SetIpPermissions(ipPermissions)
		ret, err := ec2Client.UpdateSecurityGroupRuleDescriptionsIngress(params)
		if err != nil {
			return err
		} else if ret.Return != nil && *ret.Return == false {
			log.Debugf("update security group %s rule description failed: %s", secGrpId, ipPermissions)
		}
	}

	if rule.Direction == secrules.SecurityRuleEgress {
		params := &ec2.UpdateSecurityGroupRuleDescriptionsEgressInput{}
		params.SetGroupId(secGrpId)
		params.SetIpPermissions(ipPermissions)
		ret, err := ec2Client.UpdateSecurityGroupRuleDescriptionsEgress(params)
		if err != nil {
			return err
		} else if ret.Return != nil && *ret.Return == false {
			log.Debugf("update security group %s rule description failed: %s", secGrpId, ipPermissions)
		}
	}
	return nil
}

func (self *SRegion) CreateSecurityGroup(vpcId string, name string, secgroupIdTag string, desc string) (string, error) {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return "", errors.Wrap(err, "getEc2Client")
	}

	params := &ec2.CreateSecurityGroupInput{}
	params.SetVpcId(vpcId)
	// 这里的描述aws 上层代码拼接的描述。并非用户提交的描述，用户描述放置在Yunion本地数据库中。）
	if len(desc) == 0 {
		desc = "vpc default group"
	}
	params.SetDescription(desc)
	if strings.ToLower(name) == "default" {
		name = randomString(fmt.Sprintf("%s-", vpcId), 9)
	}
	params.SetGroupName(name)

	group, err := ec2Client.CreateSecurityGroup(params)
	if err != nil {
		return "", err
	}

	tagspec := TagSpec{ResourceType: "security-group"}
	if len(secgroupIdTag) > 0 {
		tagspec.SetTag("id", secgroupIdTag)
	}
	tagspec.SetNameTag(name)
	tagspec.SetDescTag(desc)
	tags, _ := tagspec.GetTagSpecifications()
	tagParams := &ec2.CreateTagsInput{}
	tagParams.SetResources([]*string{group.GroupId})
	tagParams.SetTags(tags.Tags)
	_, err = ec2Client.CreateTags(tagParams)
	if err != nil {
		return "", err
	}

	return *group.GroupId, nil
}

func (self *SRegion) createDefaultSecurityGroup(vpcId string) (string, error) {
	name := randomString(fmt.Sprintf("%s-", vpcId), 9)
	secId, err := self.CreateSecurityGroup(vpcId, name, "default", "vpc default group")
	if err != nil {
		return "", err
	}

	rule := cloudprovider.SecurityRule{
		SecurityRule: secrules.SecurityRule{
			Priority:  1,
			Action:    secrules.SecurityRuleAllow,
			Protocol:  "",
			Direction: secrules.SecurityRuleIngress,
			PortStart: -1,
			PortEnd:   -1,
		},
	}

	err = self.addSecurityGroupRule(secId, rule)
	if err != nil {
		return "", err
	}
	return secId, nil
}

func (self *SRegion) GetSecurityGroupDetails(secGroupId string) (*SSecurityGroup, error) {
	if len(secGroupId) == 0 {
		return nil, fmt.Errorf("GetSecurityGroupDetails security group id should not be empty.")
	}
	params := &ec2.DescribeSecurityGroupsInput{}
	params.SetGroupIds([]*string{&secGroupId})

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return nil, errors.Wrap(err, "getEc2Client")
	}
	ret, err := ec2Client.DescribeSecurityGroups(params)
	err = parseNotFoundError(err)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeSecurityGroups")
	}

	if len(ret.SecurityGroups) == 1 {
		s := ret.SecurityGroups[0]
		vpc, err := self.getVpc(*s.VpcId)
		if err != nil {
			return nil, fmt.Errorf("vpc %s not found", *s.VpcId)
		}

		permissions := self.getSecRules(s.IpPermissions, s.IpPermissionsEgress)

		return &SSecurityGroup{
			vpc:               vpc,
			Description:       *s.Description,
			SecurityGroupId:   *s.GroupId,
			SecurityGroupName: *s.GroupName,
			VpcId:             *s.VpcId,
			Permissions:       permissions,
			RegionId:          self.RegionId,
		}, nil
	} else {
		return nil, fmt.Errorf("required one security group. but found: %d", len(ret.SecurityGroups))
	}
}

func (self *SRegion) getSecurityGroupById(vpcId, secgroupId string) (*SSecurityGroup, error) {
	if len(secgroupId) == 0 {
		return nil, httperrors.NewInputParameterError("security group id should not be empty")
	}

	secgroups, total, err := self.GetSecurityGroups(vpcId, "", secgroupId, 0, 0)
	if err != nil {
		log.Errorf("GetSecurityGroups vpc %s secgroupId %s: %s", vpcId, secgroupId, err)
		return nil, errors.Wrap(err, "GetSecurityGroups")
	}

	if total != 1 {
		log.Debugf("failed to find  SecurityGroup %s: %d found", secgroupId, total)
		return nil, httperrors.NewNotFoundError("failed to find SecurityGroup %s", secgroupId)
	}
	return &secgroups[0], nil
}

func (self *SRegion) addTagToSecurityGroup(secgroupId, key, value string, index int) error {
	return nil
}

func (self *SRegion) modifySecurityGroup(secGrpId string, name string, desc string) error {
	tagspec := TagSpec{ResourceType: "security-group"}
	tagspec.SetNameTag(name)
	tagspec.SetDescTag(desc)
	ec2Tags, _ := tagspec.GetTagSpecifications()
	params := &ec2.CreateTagsInput{}
	params.SetTags(ec2Tags.Tags)
	params.SetResources([]*string{&secGrpId})

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}
	_, err = ec2Client.CreateTags(params)
	if err != nil {
		return errors.Wrap(err, "CreateTags")
	}

	return nil
}

func (self *SRegion) getSecRules(ingress []*ec2.IpPermission, egress []*ec2.IpPermission) []cloudprovider.SecurityRule {
	rules := []cloudprovider.SecurityRule{}
	for _, p := range ingress {
		ret, err := AwsIpPermissionToYunion(secrules.SecurityRuleIngress, *p)
		if err != nil {
			log.Debugln(err)
		}

		for _, rule := range ret {
			rules = append(rules, rule)
		}
	}

	for _, p := range egress {
		ret, err := AwsIpPermissionToYunion(secrules.SecurityRuleEgress, *p)
		if err != nil {
			log.Debugln(err)
		}

		for _, rule := range ret {
			rules = append(rules, rule)
		}
	}

	return rules
}

func (self *SRegion) GetSecurityGroups(vpcId string, name string, secgroupId string, offset int, limit int) ([]SSecurityGroup, int, error) {
	params := &ec2.DescribeSecurityGroupsInput{}
	filters := make([]*ec2.Filter, 0)
	if len(vpcId) > 0 {
		filters = AppendSingleValueFilter(filters, "vpc-id", vpcId)
	}

	if len(name) > 0 {
		filters = AppendSingleValueFilter(filters, "group-name", name)
	}

	if len(secgroupId) > 0 {
		params.SetGroupIds([]*string{&secgroupId})
	}

	if len(filters) > 0 {
		params.SetFilters(filters)
	}

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return nil, 0, errors.Wrap(err, "getEc2Client")
	}
	ret, err := ec2Client.DescribeSecurityGroups(params)
	err = parseNotFoundError(err)
	if err != nil {
		return nil, 0, err
	}

	securityGroups := []SSecurityGroup{}
	for _, item := range ret.SecurityGroups {
		if err := FillZero(item); err != nil {
			return nil, 0, err
		}

		if len(*item.VpcId) == 0 {
			log.Debugf("ingored: security group with no vpc binded")
			continue
		}

		vpc, err := self.getVpc(*item.VpcId)
		if err != nil {
			log.Errorf("vpc %s not found", *item.VpcId)
			continue
		}

		permissions := self.getSecRules(item.IpPermissions, item.IpPermissionsEgress)
		group := SSecurityGroup{
			vpc:               vpc,
			Description:       *item.Description,
			SecurityGroupId:   *item.GroupId,
			SecurityGroupName: *item.GroupName,
			VpcId:             *item.VpcId,
			Permissions:       permissions,
			RegionId:          self.RegionId,
			// Tags:              *item.Tags,
		}

		securityGroups = append(securityGroups, group)
	}

	return securityGroups, len(securityGroups), nil
}

func (self *SSecurityGroup) GetProjectId() string {
	return ""
}

func (self *SSecurityGroup) SyncRules(common, inAdds, outAdds, inDels, outDels []cloudprovider.SecurityRule) error {
	for _, r := range append(inDels, outDels...) {
		err := self.vpc.region.DelSecurityGroupRule(self.SecurityGroupId, r)
		if err != nil {
			if strings.Contains(err.Error(), "InvalidPermission.NotFound") {
				continue
			}
			return errors.Wrapf(err, "delSecurityGroupRule %s %d %s", r.Name, r.Priority, r.String())
		}
	}

	for _, r := range append(inAdds, outAdds...) {
		err := self.vpc.region.addSecurityGroupRules(self.SecurityGroupId, r)
		if err != nil {
			return errors.Wrapf(err, "addSecurityGroupRules %d %s", r.Priority, r.String())
		}
	}
	return nil
}

func (self *SSecurityGroup) Delete() error {
	return self.vpc.region.DeleteSecurityGroup(self.SecurityGroupId)
}
