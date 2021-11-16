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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type IpPermission struct {
	FromPort int `xml:"fromPort"`
	Groups   []struct {
		Description            string `xml:"description"`
		GroupId                string `xml:"groupId"`
		GroupName              string `xml:"groupName"`
		PeeringStatus          string `xml:"peeringStatus"`
		UserId                 string `xml:"userId"`
		VpcId                  string `xml:"vpcId"`
		VpcPeeringConnectionId string `xml:"vpcPeeringConnectionId"`
	} `xml:"groups>item"`
	PrefixListIds []struct {
		Description  string `xml:"description"`
		PrefixListId string `xml:"prefixListId"`
	} `xml:"prefixListIds>item"`
	IpProtocol string `xml:"ipProtocol"`
	IpRanges   []struct {
		CidrIp      string `xml:"cidrIp"`
		Description string `xml:"description"`
	} `xml:"ipRanges>item"`
	Ipv6Ranges []struct {
		CidrIpv6    string `xml:"cidrIpv6"`
		Description string `xml:"description"`
	} `xml:"ipv6Ranges>item"`
	ToPort int `xml:"toPort"`
}

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	multicloud.AwsTags
	region *SRegion

	GroupDescription    string         `xml:"groupDescription"`
	GroupId             string         `xml:"groupId"`
	GroupName           string         `xml:"groupName"`
	OwnerId             string         `xml:"ownerId"`
	VpcId               string         `xml:"vpcId"`
	IpPermissions       []IpPermission `xml:"ipPermissions>item"`
	IpPermissionsEgress []IpPermission `xml:"ipPermissionsEgress>item"`
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
	return self.GroupId
}

func (self *SSecurityGroup) GetVpcId() string {
	return self.VpcId
}

func (self *SSecurityGroup) GetName() string {
	if len(self.GroupName) > 0 {
		return self.GroupName
	}
	return self.GroupId
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.GroupId
}

func (self *SSecurityGroup) GetStatus() string {
	return ""
}

func (self *SSecurityGroup) Refresh() error {
	group, err := self.region.getSecurityGroupById(self.VpcId, self.GroupId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, group)
}

func (self *SSecurityGroup) IsEmulated() bool {
	return false
}

func (self *SSecurityGroup) GetDescription() string {
	return self.GroupDescription
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.SecurityRule, error) {
	return getSecRules(self.IpPermissions, self.IpPermissionsEgress), nil
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
	params, err := YunionSecRuleToAws(rule)
	if err != nil {
		return err
	}
	params["GroupId"] = secGrpId
	err = func() error {
		if rule.Direction == secrules.SecurityRuleIngress {
			return self.ec2Request("AuthorizeSecurityGroupIngress", params, nil)
		}

		return self.ec2Request("AuthorizeSecurityGroupEgress", params, nil)
	}()

	if err != nil && strings.Contains(err.Error(), "InvalidPermission.Duplicate") {
		log.Debugf("addSecurityGroupRule %s %s", rule.Direction, err.Error())
		return nil
	}

	return err
}

func (self *SRegion) DelSecurityGroupRule(secGrpId string, rule cloudprovider.SecurityRule) error {
	params, err := YunionSecRuleToAws(rule)
	if err != nil {
		return err
	}

	params["GroupId"] = secGrpId

	if rule.Direction == secrules.SecurityRuleIngress {
		return self.ec2Request("RevokeSecurityGroupIngress", params, nil)
	}
	return self.ec2Request("RevokeSecurityGroupEgress", params, nil)
}

func (self *SRegion) CreateSecurityGroup(vpcId string, name string, secgroupIdTag string, desc string) (string, error) {
	// 这里的描述aws 上层代码拼接的描述。并非用户提交的描述，用户描述放置在Yunion本地数据库中。）
	if len(desc) == 0 {
		desc = "vpc default group"
	}
	if strings.ToLower(name) == "default" {
		name = randomString(fmt.Sprintf("%s-", vpcId), 9)
	}
	params := map[string]string{
		"VpcId":            vpcId,
		"GroupName":        name,
		"GroupDescription": desc,
	}
	ret := struct {
		GroupId string `xml:"groupId"`
	}{}
	return ret.GroupId, self.ec2Request("CreateSecurityGroup", params, &ret)
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

func (self *SRegion) getSecurityGroupById(vpcId, secgroupId string) (*SSecurityGroup, error) {
	secgroups, err := self.GetSecurityGroups(vpcId, "", secgroupId)
	if err != nil {
		return nil, errors.Wrap(err, "GetSecurityGroups")
	}

	for i := range secgroups {
		if secgroups[i].GroupId == secgroupId {
			secgroups[i].region = self
			return &secgroups[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, secgroupId)
}

func getSecRules(ingress []IpPermission, egress []IpPermission) []cloudprovider.SecurityRule {
	rules := []cloudprovider.SecurityRule{}
	for _, p := range ingress {
		ret, err := AwsIpPermissionToYunion(secrules.SecurityRuleIngress, p)
		if err != nil {
			log.Debugln(err)
		}

		for _, rule := range ret {
			rules = append(rules, rule)
		}
	}

	for _, p := range egress {
		ret, err := AwsIpPermissionToYunion(secrules.SecurityRuleEgress, p)
		if err != nil {
			log.Debugln(err)
		}

		for _, rule := range ret {
			rules = append(rules, rule)
		}
	}

	return rules
}

func (self *SRegion) GetSecurityGroups(vpcId string, name string, secgroupId string) ([]SSecurityGroup, error) {
	params := map[string]string{}
	idx := 1
	if len(vpcId) > 0 {
		params[fmt.Sprintf("Filter.%d.vpc-id", idx)] = vpcId
		idx++
	}

	if len(name) > 0 {
		params[fmt.Sprintf("Filter.%d.group-name", idx)] = name
		idx++
	}

	if len(secgroupId) > 0 {
		params["GroupId.1"] = secgroupId
	}

	ret := []SSecurityGroup{}
	for {
		result := struct {
			Secgroups []SSecurityGroup `xml:"securityGroupInfo>item"`
			NextToken string           `xml:"nextToken"`
		}{}

		err := self.ec2Request("DescribeSecurityGroups", params, &result)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeSecurityGroups")
		}
		ret = append(ret, result.Secgroups...)
		if len(result.NextToken) == 0 || len(result.Secgroups) == 0 {
			break
		}
		params["NextToken"] = result.NextToken
	}

	return ret, nil
}

func (self *SSecurityGroup) GetProjectId() string {
	return ""
}

func (self *SSecurityGroup) SyncRules(common, inAdds, outAdds, inDels, outDels []cloudprovider.SecurityRule) error {
	for _, r := range append(inDels, outDels...) {
		err := self.region.DelSecurityGroupRule(self.GroupId, r)
		if err != nil {
			if strings.Contains(err.Error(), "InvalidPermission.NotFound") {
				continue
			}
			return errors.Wrapf(err, "delSecurityGroupRule %s %d %s", r.Name, r.Priority, r.String())
		}
	}

	for _, r := range append(inAdds, outAdds...) {
		err := self.region.addSecurityGroupRules(self.GroupId, r)
		if err != nil {
			return errors.Wrapf(err, "addSecurityGroupRules %d %s", r.Priority, r.String())
		}
	}
	return nil
}

func (self *SSecurityGroup) Delete() error {
	return self.region.DeleteSecurityGroup(self.GroupId)
}
