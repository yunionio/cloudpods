// Copyright 2023 Yunion
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

package volcengine

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"
)

type SCidrList []string

type SSecurityGroupRule struct {
	CreationTime    time.Time
	UpdateTime      time.Time
	Description     string
	Direction       string
	Protocol        string
	Policy          string
	PortStart       int
	PortEnd         int
	CidrIp          string
	PrefixListId    string
	PrefixListCidrs SCidrList
	Priority        int
	SourceGroupId   string
}

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	VolcEngineTags

	region            *SRegion
	Description       string
	Permissions       []SSecurityGroupRule
	SecurityGroupId   string
	SecurityGroupName string
	VpcId             string
	CreationTime      time.Time
	UpdateTime        time.Time
	Type              string
	ProjectName       string
	ServiceManaged    bool
	Status            string
}

func (region *SRegion) CreateSecurityGroup(vpcId string, name string, desc, projectName string) (string, error) {
	params := make(map[string]string)
	if len(vpcId) > 0 {
		params["VpcId"] = vpcId
	}

	if len(projectName) > 0 {
		params["ProjectName"] = projectName
	}

	if len(name) > 0 {
		params["SecurityGroupName"] = name
	}
	if len(desc) > 0 {
		params["Description"] = desc
	}
	params["ClientToken"] = utils.GenRequestId(20)

	body, err := region.vpcRequest("CreateSecurityGroup", params)
	if err != nil {
		return "", errors.Wrap(err, "CreateSecurityGroup")
	}
	return body.GetString("Result", "SecurityGroupId")
}

func (region *SRegion) GetSecurityGroupDetails(secGroupId string) (*SSecurityGroup, error) {
	params := make(map[string]string)
	params["SecurityGroupId"] = secGroupId
	body, err := region.vpcRequest("DescribeSecurityGroupAttributes", params)

	if err != nil {
		return nil, err
	}
	securitygroup := SSecurityGroup{}
	err = body.Unmarshal(&securitygroup, "Result")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal security group details fail")
	}
	securitygroup.region = region
	return &securitygroup, err
}

func (region *SRegion) DeleteSecurityGroupById(secGroupId string) error {
	params := make(map[string]string)
	params["SecurityGroupId"] = secGroupId
	_, err := region.vpcRequest("DeleteSecurityGroupId", params)
	return err
}

func (region *SRegion) AddSecurityGroupRules(secGrpId string, rule cloudprovider.SecurityRule) error {
	if len(rule.Ports) != 0 {
		for _, port := range rule.Ports {
			rule.PortStart, rule.PortEnd = port, port
			err := region.addSecurityGroupRule(secGrpId, rule)
			if err != nil {
				return errors.Wrapf(err, "addSecurityGroupRule %s", rule.String())
			}
		}
		return nil
	}
	return region.addSecurityGroupRule(secGrpId, rule)
}

func (region *SRegion) addSecurityGroupRule(secGrpId string, rule cloudprovider.SecurityRule) error {
	params := make(map[string]string)
	params["RegionId"] = region.RegionId
	params["SecurityGroupId"] = secGrpId
	params["Description"] = rule.Description
	params["PortStart"] = fmt.Sprintf("%d", rule.PortStart)
	params["PortEnd"] = fmt.Sprintf("%d", rule.PortEnd)
	protocol := rule.Protocol
	if len(rule.Protocol) == 0 || rule.Protocol == secrules.PROTO_ANY {
		protocol = "all"
	}
	params["Protocol"] = protocol
	if rule.Action == secrules.SecurityRuleAllow {
		params["Policy"] = "accept"
	} else {
		params["Policy"] = "drop"
	}

	params["Priority"] = fmt.Sprintf("%d", rule.Priority)
	if rule.Direction == secrules.SecurityRuleIngress {
		if rule.IPNet != nil {
			params["CidrIp"] = rule.IPNet.String()
		} else {
			params["CidrIp"] = "0.0.0.0/0"
		}
		_, err := region.vpcRequest("AuthorizeSecurityGroupIngress", params)
		return err
	} else {
		if rule.IPNet != nil {
			params["CidrIp"] = rule.IPNet.String()
		} else {
			params["CidrIp"] = "0.0.0.0/0"
		}
		_, err := region.vpcRequest("AuthorizeSecurityGroupEgress", params)
		return err
	}
}

func (region *SRegion) GetSecurityGroups(vpcId, name string, securityGroupIds []string, pageSize int, pageNumber int) ([]SSecurityGroup, int, error) {
	if pageSize > 100 || pageSize <= 0 {
		pageSize = 100
	}
	params := make(map[string]string)
	params["PageSize"] = fmt.Sprintf("%d", pageSize)
	params["PageNumber"] = fmt.Sprintf("%d", pageNumber)
	if len(vpcId) > 0 {
		params["VpcId"] = vpcId
	}
	if len(name) > 0 {
		params["SecurityGroupName"] = name
	}

	if len(securityGroupIds) > 0 {
		params["SecurityGroupIds"] = jsonutils.Marshal(securityGroupIds).String()
	}

	body, err := region.vpcRequest("DescribeSecurityGroups", params)
	if err != nil {
		log.Errorf("GetSecurityGroups fail %s", err)
		return nil, 0, err
	}

	secgrps := make([]SSecurityGroup, 0)
	err = body.Unmarshal(&secgrps, "Result", "SecurityGroups")
	if err != nil {
		log.Errorf("Unmarshal security groups fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("Result", "TotalCount")
	return secgrps, int(total), nil
}

func (rule *SSecurityGroupRule) toUniformRule() (cloudprovider.SecurityRule, error) {
	uniformRule := cloudprovider.SecurityRule{
		SecurityRule: secrules.SecurityRule{
			Action:      secrules.SecurityRuleDeny,
			Direction:   secrules.DIR_IN,
			Priority:    101 - rule.Priority,
			Description: rule.Description,
			PortStart:   -1,
			PortEnd:     -1,
		},
	}
	if strings.ToLower(rule.Policy) == "accept" {
		uniformRule.Action = secrules.SecurityRuleAllow
	}
	cidr := rule.CidrIp
	if rule.Direction == "egress" {
		uniformRule.Direction = secrules.DIR_OUT
	}
	uniformRule.ParseCIDR(cidr)
	switch strings.ToLower(rule.Protocol) {
	case "tcp", "udp", "icmp":
		uniformRule.Protocol = strings.ToLower(rule.Protocol)
	case "all":
		uniformRule.Protocol = secrules.PROTO_ANY
	default:
		return uniformRule, fmt.Errorf("unsupported protocal %s", rule.Protocol)
	}
	port := ""
	if rule.PortStart == rule.PortEnd {
		if rule.PortStart != -1 {
			port = fmt.Sprintf("%d", rule.PortStart)
		}
	} else if rule.PortStart != -1 && rule.PortEnd != 65535 {
		port = fmt.Sprintf("%d-%d", rule.PortStart, rule.PortEnd)
	}
	err := uniformRule.ParsePorts(port)
	if err != nil {
		return uniformRule, errors.Wrapf(err, "ParsePorts(%s)", port)
	}
	return uniformRule, nil
}

func (secgroup *SSecurityGroup) GetId() string {
	return secgroup.SecurityGroupId
}

func (secgroup *SSecurityGroup) GetName() string {
	return secgroup.SecurityGroupName
}

func (secgroup *SSecurityGroup) GetGlobalId() string {
	return secgroup.GetId()
}

func (secgroup *SSecurityGroup) GetCreatedAt() time.Time {
	return secgroup.CreationTime
}

func (secgroup *SSecurityGroup) GetDescription() string {
	return secgroup.Description
}

func (secgroup *SSecurityGroup) GetStatus() string {
	return secgroup.Status
}

func (secgroup *SSecurityGroup) Refresh() error {
	if body, err := secgroup.region.GetSecurityGroupDetails(secgroup.GetId()); err != nil {
		return err
	} else {
		return jsonutils.Update(secgroup, body)
	}
}

func (secgroup *SSecurityGroup) GetProjectId() string {
	return secgroup.ProjectName
}

func (secgroup *SSecurityGroup) GetRules() ([]cloudprovider.SecurityRule, error) {
	rules := make([]cloudprovider.SecurityRule, 0)
	updatedSecgroup, err := secgroup.region.GetSecurityGroupDetails(secgroup.SecurityGroupId)
	if err != nil {
		return nil, err
	}
	outAllow := secrules.MustParseSecurityRule("out:allow any")
	rules = append(rules, cloudprovider.SecurityRule{SecurityRule: *outAllow})
	for _, permission := range updatedSecgroup.Permissions {
		if len(permission.SourceGroupId) > 0 {
			continue
		}
		if !utils.IsInStringArray(strings.ToLower(permission.Protocol), []string{"tcp", "udp", "icmp", "all"}) {
			continue
		}
		rule, err := permission.toUniformRule()
		if err != nil {
			log.Errorf("convert rule %s for group %s(%s) error: %v", permission.Description, secgroup.SecurityGroupName, secgroup.SecurityGroupId, err)
			continue
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (secgroup *SSecurityGroup) GetVpcId() string {
	return secgroup.VpcId
}

func (secgroup *SSecurityGroup) GetReferences() ([]cloudprovider.SecurityGroupReference, error) {
	ret := []cloudprovider.SecurityGroupReference{}
	return ret, errors.Wrapf(errors.ErrNotImplemented, "GetReferences not supported")
}

func (region *SRegion) DeleteSecurityGroup(secGrpId string) error {
	params := make(map[string]string)
	params["SecurityGroupId"] = secGrpId

	_, err := region.vpcRequest("DeleteSecurityGroup", params)
	if err != nil {
		return errors.Wrapf(err, "Delete security group fail")
	}
	return nil
}

func (secgroup *SSecurityGroup) Delete() error {
	return secgroup.region.DeleteSecurityGroupById(secgroup.SecurityGroupId)
}
