package aliyun

import (
	"fmt"
	"time"
	"github.com/yunionio/log"
	"github.com/yunionio/pkg/util/secrules"
	"github.com/yunionio/pkg/utils"
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

type SSecurityGroup struct {
	CreationTime      time.Time
	Description       string
	SecurityGroupId   string
	SecurityGroupName string
	VpcId             string
	InnerAccessPolicy string
	Permissions       SPermissions
	RegionId          string
}

func (self *SRegion) GetSecurityGroups(vpcId string, offset int, limit int) ([]SSecurityGroup, int, error) {
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

func (self *SRegion) createSecurityGroup(vpcId string, name string, desc string) (string, error) {
	params := make(map[string]string)
	if len(vpcId) > 0 {
		params["VpcId"] = vpcId
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

func (self *SRegion) addSecurityGroupRule(secGrpId string, rule *secrules.SecurityRule) error {
	params := make(map[string]string)
	params["SecurityGroupId"] = secGrpId
	protocol := rule.Protocol
	if len(rule.Protocol) == 0 {
		protocol = "all"
	}
	params["IpProtocol"] = protocol
	params["PortRange"] = fmt.Sprintf("%d/%d", rule.PortStart, rule.PortEnd)
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
		params["DestCidrIp"] = "0.0.0.0/0"
		_, err := self.ecsRequest("AuthorizeSecurityGroup", params)
		return err
	} else { // rule.Direction == secrules.SecurityRuleEgress {
		if rule.IPNet != nil {
			params["DestCidrIp"] = rule.IPNet.String()
		} else {
			params["DestCidrIp"] = "0.0.0.0/0"
		}
		params["SourceCidrIp"] = "0.0.0.0/0"
		_, err := self.ecsRequest("AuthorizeSecurityGroupEgress", params)
		return err
	}
}

func (self *SRegion) createDefaultSecurityGroup(vpcId string) (string, error) {
	secId, err := self.createSecurityGroup(vpcId, "", "")
	if err != nil {
		return "", err
	}
	inRule := secrules.SecurityRule{
		Priority:  1,
		Action:    secrules.SecurityRuleAllow,
		Protocol:  "",
		Direction: secrules.SecurityRuleIngress,
		PortStart: -1,
		PortEnd:   -1,
	}
	err = self.addSecurityGroupRule(secId, &inRule)
	if err != nil {
		return "", err
	}
	outRule := secrules.SecurityRule{
		Priority:  1,
		Action:    secrules.SecurityRuleAllow,
		Protocol:  "",
		Direction: secrules.SecurityRuleEgress,
		PortStart: -1,
		PortEnd:   -1,
	}
	err = self.addSecurityGroupRule(secId, &outRule)
	if err != nil {
		return "", err
	}
	return secId, nil
}
