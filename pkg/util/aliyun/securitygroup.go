package aliyun

import (
	"fmt"
	"strings"
	"time"

	"github.com/deckarep/golang-set"
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/onecloud/pkg/httperrors"
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
	params["RegionId"] = self.RegionId
	params["SecurityGroupId"] = secGrpId
	params["NicType"] = string(IntranetNicType)
	params["PortRange"] = fmt.Sprintf("%d/%d", rule.PortStart, rule.PortEnd)
	protocol := rule.Protocol
	if len(rule.Protocol) == 0 || rule.Protocol == secrules.PROTO_ANY {
		protocol = "all"
		params["PortRange"] = "-1/-1"
	}
	params["IpProtocol"] = protocol
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
		log.Debugf("add Security Group params: %v", params)
		_, err := self.ecsRequest("AuthorizeSecurityGroup", params)
		return err
	} else { // rule.Direction == secrules.SecurityRuleEgress {
		if rule.IPNet != nil {
			params["DestCidrIp"] = rule.IPNet.String()
		} else {
			params["DestCidrIp"] = "0.0.0.0/0"
		}
		log.Debugf("add Security Group params: %v", params)
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
		params["PortRange"] = "-1/-1"
	}
	params["IpProtocol"] = protocol
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
		log.Debugf("del Security Group params: %v", params)
		_, err := self.ecsRequest("RevokeSecurityGroup", params)
		return err
	} else { // rule.Direction == secrules.SecurityRuleEgress {
		if rule.IPNet != nil {
			params["DestCidrIp"] = rule.IPNet.String()
		} else {
			params["DestCidrIp"] = "0.0.0.0/0"
		}
		log.Debugf("del Security Group params: %v", params)
		_, err := self.ecsRequest("RevokeSecurityGroupEgress", params)
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

func (self *SRegion) getSecurityGroupByTag(vpcId, secgroupId string) (*SSecurityGroup, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	if len(vpcId) > 0 {
		params["VpcId"] = vpcId
	}
	params["Tag.1.Key"] = "id"
	params["Tag.1.Value"] = secgroupId

	secgrps := make([]SSecurityGroup, 0)
	if body, err := self.ecsRequest("DescribeSecurityGroups", params); err != nil {
		return nil, err
	} else if err := body.Unmarshal(&secgrps, "SecurityGroups", "SecurityGroup"); err != nil {
		return nil, err
	} else if len(secgrps) != 1 {
		return nil, httperrors.NewNotFoundError("failed to find SecurityGroup %s", secgroupId)
	}
	return &secgrps[0], nil
}

func (self *SPermission) toString() string {
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
	protocol := strings.ToLower(self.IpProtocol)
	if protocol == "all" {
		protocol = "any"
	}
	port, ports := "", strings.Split(self.PortRange, "/")
	if ports[0] == ports[1] && (ports[0] != "-1") {
		port = ports[0]
	} else {
		port = fmt.Sprintf("%s-%s", ports[0], ports[1])
	}
	return fmt.Sprintf("%s:%s %s %s %s", direction, string(action), cidr, protocol, port)
}

func (self *SRegion) addTagToSecurityGroup(secgroupId, key, value string, index int) error {
	if index > 5 || index < 1 {
		index = 1
	}
	params := map[string]string{"ResourceType": "securitygroup", "ResourceId": secgroupId}
	params[fmt.Sprintf("Tag.%d.Key", index)] = key
	params[fmt.Sprintf("Tag.%d.Value", index)] = value
	_, err := self.ecsRequest("AddTags", params)
	return err
}

func (self *SRegion) revokeSecurityGroup(secgroupId, instanceId string) error {
	if secgroup, err := self.GetSecurityGroupDetails(secgroupId); err != nil {
		return err
	} else {
		for _, r := range secgroup.Permissions.Permission {
			if rule, err := secrules.ParseSecurityRule(r.toString()); err != nil {
				return err
			} else {
				rule.Priority = r.Priority
				if err := self.delSecurityGroupRule(secgroup.SecurityGroupId, rule); err != nil {
					return err
				}
			}
		}
		if rule, err := secrules.ParseSecurityRule("in:allow any"); err != nil {
			rule.Priority = 100
			if err := self.addSecurityGroupRule(secgroup.SecurityGroupId, rule); err != nil {
				return err
			}
		}
		if rule, err := secrules.ParseSecurityRule("out:allow any"); err != nil {
			rule.Priority = 100
			if err := self.addSecurityGroupRule(secgroup.SecurityGroupId, rule); err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *SRegion) syncSecgroupRules(secgroupId string, rules []*secrules.SecurityRule) error {
	if secgroup, err := self.GetSecurityGroupDetails(secgroupId); err != nil {
		return err
	} else {
		newRules, newStr := make(map[string]*secrules.SecurityRule), mapset.NewSet()
		for _, rule := range rules {
			rule.Priority = 101 - rule.Priority
			if len(rule.Ports) > 0 {
				for _, port := range rule.Ports {
					rule.PortStart, rule.PortEnd = port, port
					if jsonStr := jsonutils.Marshal(rule).String(); newStr.Add(jsonStr) {
						newRules[jsonStr] = rule
					}
				}
				continue
			} else if rule.PortStart == 0 || rule.PortEnd == 0 {
				rule.PortStart, rule.PortEnd = -1, -1
			}
			if jsonStr := jsonutils.Marshal(rule).String(); newStr.Add(jsonStr) {
				newRules[jsonStr] = rule
			}
		}

		oldRules, oldStr := make(map[string]*secrules.SecurityRule), mapset.NewSet()
		for _, r := range secgroup.Permissions.Permission {
			if rule, err := secrules.ParseSecurityRule(r.toString()); err != nil {
				return err
			} else {
				rule.Priority = r.Priority
				if jsonStr := jsonutils.Marshal(rule).String(); oldStr.Add(jsonStr) {
					oldRules[jsonStr] = rule
				}
			}
		}

		newStr.Each(func(r interface{}) bool {
			rStr := r.(string)
			if !oldStr.Contains(r) {
				rule := newRules[rStr]
				log.Debugf("add secgroup rule: %v Priority: %d", rule, rule.Priority)
				self.addSecurityGroupRule(secgroupId, rule)
			}
			return false
		})
		oldStr.Each(func(r interface{}) bool {
			rStr := r.(string)
			if !newStr.Contains(r) {
				rule := oldRules[rStr]
				log.Debugf("del secgroup rule: %v Priority: %d", rule, rule.Priority)
				self.delSecurityGroupRule(secgroupId, rule)
			}
			return false
		})
	}
	return nil
}

func (self *SRegion) assignSecurityGroup(secgroupId, instanceId string) error {
	params := map[string]string{"InstanceId": instanceId, "SecurityGroupId": secgroupId}
	_, err := self.ecsRequest("JoinSecurityGroup", params)
	return err
}
