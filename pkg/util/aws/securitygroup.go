package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"sort"
	"strings"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/util/secrules"
)

type Tags struct {
	Tag []Tag
}

type Tag struct {
	TagKey   string
	TagValue string
}

type SSecurityGroup struct {
	vpc *SVpc

	RegionId          string
	VpcId             string
	SecurityGroupId   string
	Description       string
	SecurityGroupName string
	Permissions       []secrules.SecurityRule
	Tags              Tags

	// CreationTime      time.Time
	// InnerAccessPolicy string
}

func (self *SSecurityGroup) GetId() string {
	return self.SecurityGroupId
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

func (self *SSecurityGroup) GetDescription() string {
	return self.Description
}

func (self *SSecurityGroup) GetRules() ([]secrules.SecurityRule, error) {
	rules := make([]secrules.SecurityRule, 0)
	if secgrp, err := self.vpc.region.GetSecurityGroupDetails(self.SecurityGroupId); err != nil {
		return rules, err
	} else {
		rules = secgrp.Permissions
	}

	return rules, nil
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
	ipPermissions, err := YunionSecRuleToAws(*rule)
	log.Debugf("Aws security group rule: %s", ipPermissions)
	if err != nil {
		return err
	}

	if rule.Direction == secrules.SecurityRuleIngress {
		params := &ec2.AuthorizeSecurityGroupIngressInput{}
		params.SetGroupId(secGrpId)
		params.SetIpPermissions(ipPermissions)
		_, err := self.ec2Client.AuthorizeSecurityGroupIngress(params)
		if err != nil {
			return err
		}
	}

	if rule.Direction == secrules.SecurityRuleEgress {
		params := &ec2.AuthorizeSecurityGroupEgressInput{}
		params.SetGroupId(secGrpId)
		params.SetIpPermissions(ipPermissions)
		_, err := self.ec2Client.AuthorizeSecurityGroupEgress(params)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SRegion) delSecurityGroupRule(secGrpId string, rule *secrules.SecurityRule) error {
	ipPermissions, err := YunionSecRuleToAws(*rule)
	if err != nil {
		return err
	}

	if rule.Direction == secrules.SecurityRuleIngress {
		params := &ec2.RevokeSecurityGroupIngressInput{}
		params.SetGroupId(secGrpId)
		params.SetIpPermissions(ipPermissions)
		_, err := self.ec2Client.RevokeSecurityGroupIngress(params)
		if err != nil {
			return err
		}
	}

	if rule.Direction == secrules.SecurityRuleEgress {
		params := &ec2.RevokeSecurityGroupEgressInput{}
		params.SetGroupId(secGrpId)
		params.SetIpPermissions(ipPermissions)
		_, err := self.ec2Client.RevokeSecurityGroupEgress(params)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SRegion) updateSecurityGroupRuleDescription(secGrpId string, rule *secrules.SecurityRule) error {
	ipPermissions, err := YunionSecRuleToAws(*rule)
	if err != nil {
		return err
	}

	if rule.Direction == secrules.SecurityRuleIngress {
		params := &ec2.UpdateSecurityGroupRuleDescriptionsIngressInput{}
		params.SetGroupId(secGrpId)
		params.SetIpPermissions(ipPermissions)
		ret, err := self.ec2Client.UpdateSecurityGroupRuleDescriptionsIngress(params)
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
		ret, err := self.ec2Client.UpdateSecurityGroupRuleDescriptionsEgress(params)
		if err != nil {
			return err
		} else if ret.Return != nil && *ret.Return == false {
			log.Debugf("update security group %s rule description failed: %s", secGrpId, ipPermissions)
		}
	}
	return nil
}

func (self *SRegion) createSecurityGroup(vpcId string, name string, secgroupIdTag string, desc string) (string, error) {
	params := &ec2.CreateSecurityGroupInput{}
	params.SetVpcId(vpcId)
	params.SetDescription(desc)
	params.SetGroupName(name)

	group, err := self.ec2Client.CreateSecurityGroup(params)
	if err != nil {
		return "", err
	}

	tagspec := TagSpec{ResourceType: "security-group"}
	tagspec.SetTag("id", secgroupIdTag)
	tags, _ := tagspec.GetTagSpecifications()
	tagParams := &ec2.CreateTagsInput{}
	tagParams.SetResources([]*string{group.GroupId})
	tagParams.SetTags(tags.Tags)
	_, err = self.ec2Client.CreateTags(tagParams)
	if err != nil {
		return "", err
	}

	return *group.GroupId, nil
}

func (self *SRegion) createDefaultSecurityGroup(vpcId string) (string, error) {
	secId, err := self.createSecurityGroup(vpcId, "vpc default", fmt.Sprintf("%s-default", vpcId), "vpc default group")
	if err != nil {
		return "", err
	}

	rule := &secrules.SecurityRule{
		Priority:  1,
		Action:    secrules.SecurityRuleAllow,
		Protocol:  "",
		Direction: secrules.SecurityRuleIngress,
		PortStart: -1,
		PortEnd:   -1,
	}

	err = self.addSecurityGroupRule(secId, rule)
	if err != nil {
		return "", err
	}
	return secId, nil
}

func (self *SRegion) GetSecurityGroupDetails(secGroupId string) (*SSecurityGroup, error) {
	params := &ec2.DescribeSecurityGroupsInput{}
	params.SetGroupIds([]*string{&secGroupId})

	ret, err := self.ec2Client.DescribeSecurityGroups(params)
	if err != nil {
		return nil, err
	}
	if len(ret.SecurityGroups) == 1 {
		s := ret.SecurityGroups[0]
		vpc, err := self.getVpc(*s.VpcId)
		if err != nil {
			fmt.Errorf("vpc %s not found", *s.VpcId)
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

func (self *SRegion) getSecurityGroupByTag(vpcId, secgroupId string) (*SSecurityGroup, error) {
	secgroups, total, err := self.GetSecurityGroups(vpcId, secgroupId, 0, 0)
	if err != nil {
		return nil, err
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

	_, err := self.ec2Client.CreateTags(params)
	if err != nil {
		return err
	}

	return nil
}

func (self *SRegion) syncSecgroupRules(secgroupId string, rules []secrules.SecurityRule) error {
	if secgroup, err := self.GetSecurityGroupDetails(secgroupId); err != nil {
		return err
	} else {

		sort.Sort(secrules.SecurityRuleSet(rules))
		sort.Sort(secrules.SecurityRuleSet(secgroup.Permissions))

		i, j := 0, 0
		for i < len(rules) || j < len(secgroup.Permissions) {
			if i < len(rules) && j < len(secgroup.Permissions) {
				permissionStr := secgroup.Permissions[j].String()
				ruleStr := rules[i].String()
				cmp := strings.Compare(permissionStr, ruleStr)
				if cmp == 0 {
					if secgroup.Permissions[j].Description != rules[i].Description {
						if err := self.updateSecurityGroupRuleDescription(secgroupId, &rules[i]); err != nil {
							log.Errorf("updateSecurityGroupRuleDescription %v error: %s", rules[i], err.Error())
							return err
						}
					}
					i += 1
					j += 1
				} else if cmp > 0 {
					if err := self.delSecurityGroupRule(secgroupId, &secgroup.Permissions[j]); err != nil {
						log.Errorf("delSecurityGroupRule %v error: %s", secgroup.Permissions[j], err.Error())
						return err
					}
					j += 1
				} else {
					if err := self.addSecurityGroupRules(secgroupId, &rules[i]); err != nil {
						log.Errorf("addSecurityGroupRule %v error: %s", rules[i], err.Error())
						return err
					}
					i += 1
				}
			} else if i >= len(rules) {
				if err := self.delSecurityGroupRule(secgroupId, &secgroup.Permissions[j]); err != nil {
					log.Errorf("delSecurityGroupRule %v error: %s", secgroup.Permissions[j], err.Error())
					return err
				}
				j += 1
			} else if j >= len(secgroup.Permissions) {
				if err := self.addSecurityGroupRules(secgroupId, &rules[i]); err != nil {
					log.Errorf("addSecurityGroupRule %v error: %s", rules[i], err.Error())
					return err
				}
				i += 1
			}
		}
	}
	return nil
}

func (self *SRegion) getSecRules(ingress []*ec2.IpPermission, egress []*ec2.IpPermission) []secrules.SecurityRule {
	rules := []secrules.SecurityRule{}
	for _, p := range ingress {
		ret, err := AwsIpPermissionToYunion(secrules.SecurityRuleIngress, *p)
		if err != nil {
			log.Debugf(err.Error())
		}

		for _, rule := range ret {
			rules = append(rules, rule)
		}
	}

	for _, p := range egress {
		ret, err := AwsIpPermissionToYunion(secrules.SecurityRuleEgress, *p)
		if err != nil {
			log.Debugf(err.Error())
		}

		for _, rule := range ret {
			rules = append(rules, rule)
		}
	}

	return rules
}

func (self *SRegion) GetSecurityGroups(vpcId string, secgroupIdTag string, offset int, limit int) ([]SSecurityGroup, int, error) {
	params := &ec2.DescribeSecurityGroupsInput{}
	filters := make([]*ec2.Filter, 0)
	if len(vpcId) > 0 {
		filters = AppendSingleValueFilter(filters, "vpc-id", vpcId)
	}

	if len(secgroupIdTag) > 0 {
		filters = AppendSingleValueFilter(filters, "tag:id", secgroupIdTag)
	}

	if len(filters) > 0 {
		params.SetFilters(filters)
	}

	ret, err := self.ec2Client.DescribeSecurityGroups(params)
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
