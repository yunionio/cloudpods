package aws

import (
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/secrules"
	"github.com/aws/aws-sdk-go/service/ec2"
)

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

func (self *SSecurityGroup) GetId() string {
	panic("implement me")
}

func (self *SSecurityGroup) GetName() string {
	panic("implement me")
}

func (self *SSecurityGroup) GetGlobalId() string {
	panic("implement me")
}

func (self *SSecurityGroup) GetStatus() string {
	panic("implement me")
}

func (self *SSecurityGroup) Refresh() error {
	panic("implement me")
}

func (self *SSecurityGroup) IsEmulated() bool {
	panic("implement me")
}

func (self *SSecurityGroup) GetMetadata() *jsonutils.JSONDict {
	panic("implement me")
}

func (self *SSecurityGroup) GetDescription() string {
	panic("implement me")
}

func (self *SSecurityGroup) GetRules() ([]secrules.SecurityRule, error) {
	panic("implement me")
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


