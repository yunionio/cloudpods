package aws

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/secrules"
	"github.com/aws/aws-sdk-go/service/ec2"
	"yunion.io/x/log"
)

type SecurityGroupPermissionNicType string

const (
	IntranetNicType SecurityGroupPermissionNicType = "intranet"
	InternetNicType SecurityGroupPermissionNicType = "internet"
)

type SPermission struct {
	Description       		string
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

	RegionId          string
	VpcId             string
	SecurityGroupId   string
	Description       string
	SecurityGroupName string
	Permissions       SPermissions
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
	// todo: implement me
	rules := make([]secrules.SecurityRule, 0)
	if secgrp, err := self.vpc.region.GetSecurityGroupDetails(self.SecurityGroupId); err != nil {
		return rules, err
	} else {
		for _, permission := range secgrp.Permissions.Permission {
			if rule, err := secrules.ParseSecurityRule(""); err != nil {
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

func (self *SRegion) addSecurityGroupRules(secGrpId string, rule *secrules.SecurityRule) error {
	// todo: add sercurity rules
	return nil
}

func (self *SRegion) addSecurityGroupRule(secGrpId string, rule *secrules.SecurityRule) error {
	// todo: add sercurity rules
	return nil
}

func (self *SRegion) createSecurityGroup(vpcId string, name string, desc string) (string, error) {
	params := &ec2.CreateSecurityGroupInput{}
	params.SetVpcId(vpcId)
	params.SetDescription(desc)
	params.SetGroupName(name)

	group, err := self.ec2Client.CreateSecurityGroup(params)
	if err != nil {
		return "", err
	}

	return *group.GroupId, nil
}

func (self *SRegion) createDefaultSecurityGroup(vpcId string) (string, error) {
	secId, err := self.createSecurityGroup(vpcId, "vpc default", "vpc default group")
	if err != nil {
		return "", err
	}
	// todo : add sercurity rules
	return secId, nil
}

func (self *SRegion) GetSecurityGroupDetails(secGroupId string) (*SSecurityGroup, error) {
	return nil, nil
}

func (self *SRegion) getSecurityGroupByTag(vpcId, secgroupId string) (*SSecurityGroup, error) {
	return nil, nil
}

func (self *SRegion) addTagToSecurityGroup(secgroupId, key, value string, index int) error {
	return nil
}

func (self *SRegion) modifySecurityGroup(secGrpId string, name string, desc string) error {
	return nil
}

func (self *SRegion) syncSecgroupRules(secgroupId string, rules []secrules.SecurityRule) error {
	return nil
}

func (self *SRegion) getSecurityGroupsPermissions(ingress []*ec2.IpPermission, egress []*ec2.IpPermission) SPermissions {
	return SPermissions{}
}

func (self *SRegion) GetSecurityGroups(vpcId string, offset int, limit int) ([]SSecurityGroup, int, error) {
	params := &ec2.DescribeSecurityGroupsInput{}
	filters := make([]*ec2.Filter, 0)
	if len(vpcId) > 0 {
		filters = AppendSingleValueFilter(filters, "vpc-id", vpcId)
	}

	if len(filters) > 0 {
		params.SetFilters(filters)
	}

	ret, err := self.ec2Client.DescribeSecurityGroups(params)
	if err != nil {
		return nil, 0 , err
	}

	securityGroups := []SSecurityGroup{}
	for _,item := range ret.SecurityGroups {
		vpc, err := self.getVpc(*item.VpcId)
		if err != nil {
			log.Errorf("vpc %s not found", *item.VpcId)
		}

		permissions := self.getSecurityGroupsPermissions(item.IpPermissions, item.IpPermissionsEgress)


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