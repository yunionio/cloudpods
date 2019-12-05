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

package ctyun

import (
	"net"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SSecurityGroup struct {
	region *SRegion
	vpc    *SVpc

	ID                 string `json:"id"`
	ResSecurityGroupID string `json:"resSecurityGroupId"`
	Name               string `json:"name"`
	AccountID          string `json:"accountId"`
	UserID             string `json:"userId"`
	RegionID           string `json:"regionId"`
	ZoneID             string `json:"zoneId"`
	VpcID              string `json:"vpcId"`
	CreateDate         int64  `json:"createDate"`
	Status             int64  `json:"status"`
}

func (self *SSecurityGroup) SyncRules(rules []secrules.SecurityRule) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SSecurityGroup) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SSecurityGroup) GetId() string {
	return self.ResSecurityGroupID
}

func (self *SSecurityGroup) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.ResSecurityGroupID
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.GetId()
}

func (self *SSecurityGroup) GetStatus() string {
	return ""
}

func (self *SSecurityGroup) Refresh() error {
	if new, err := self.region.GetSecurityGroupDetails(self.GetId()); err != nil {
		return err
	} else {
		return jsonutils.Update(self, new)
	}
}

func (self *SSecurityGroup) IsEmulated() bool {
	return false
}

func (self *SSecurityGroup) GetMetadata() *jsonutils.JSONDict {
	return jsonutils.NewDict()
}

func (self *SSecurityGroup) GetDescription() string {
	return ""
}

// 判断是否兼容云端安全组规则
func compatibleSecurityGroupRule(r SSecurityGroupRule) bool {
	// 忽略了源地址是安全组的规则
	if len(r.RemoteGroupId) > 0 {
		return false
	}

	// 忽略IPV6
	if r.Ethertype == "IPv6" {
		return false
	}

	return true
}

func (self *SSecurityGroup) GetRules() ([]secrules.SecurityRule, error) {
	_rules, err := self.region.GetSecurityGroupRules(self.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "SSecurityGroup.GetRules.GetSecurityGroupRules")
	}

	rules := make([]secrules.SecurityRule, 0)
	for _, r := range _rules {
		if !compatibleSecurityGroupRule(r) {
			continue
		}

		rule, err := self.GetSecurityRule(r, false)
		if err != nil {
			return rules, err
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

func (self *SSecurityGroup) GetSecurityRule(remoteRule SSecurityGroupRule, withRuleId bool) (secrules.SecurityRule, error) {
	var err error
	var direction secrules.TSecurityRuleDirection
	if remoteRule.Direction == "ingress" {
		direction = secrules.SecurityRuleIngress
	} else {
		direction = secrules.SecurityRuleEgress
	}

	protocol := secrules.PROTO_ANY
	if remoteRule.Protocol != "" {
		protocol = remoteRule.Protocol
	}

	var portStart int
	var portEnd int
	if protocol == secrules.PROTO_ICMP {
		portStart = -1
		portEnd = -1
	} else {
		portStart = int(remoteRule.PortRangeMin)
		portEnd = int(remoteRule.PortRangeMax)
	}

	ipNet := &net.IPNet{}
	if len(remoteRule.RemoteIPPrefix) > 0 {
		_, ipNet, err = net.ParseCIDR(remoteRule.RemoteIPPrefix)
	} else {
		_, ipNet, err = net.ParseCIDR("0.0.0.0/0")
	}

	if err != nil {
		return secrules.SecurityRule{}, err
	}

	// withRuleId.将ruleId附加到description字段。该hook有特殊目的，仅在同步安全组时使用。
	desc := ""
	if withRuleId {
		desc = remoteRule.ID
	} else {
		desc = remoteRule.Description
	}

	rule := secrules.SecurityRule{
		Priority:    1,
		Action:      secrules.SecurityRuleAllow,
		IPNet:       ipNet,
		Protocol:    protocol,
		Direction:   direction,
		PortStart:   portStart,
		PortEnd:     portEnd,
		Ports:       nil,
		Description: desc,
	}

	err = rule.ValidateRule()
	return rule, err
}

func (self *SSecurityGroup) GetVpcId() string {
	if len(self.VpcID) == 0 {
		return "normal"
	}

	return self.VpcID
}

func (self *SRegion) GetSecurityGroupDetails(groupId string) (*SSecurityGroup, error) {
	params := map[string]string{
		"regionId":        self.GetId(),
		"securityGroupId": groupId,
	}

	resp, err := self.client.DoGet("/apiproxy/v3/querySecurityGroupDetail", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetSecurityGroupDetails")
	}

	ret := &SSecurityGroup{}
	err = resp.Unmarshal(&ret, "returnObj")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetSecurityGroupDetails.Unmarshal")
	}

	ret.region = self
	return ret, nil
}

func (self *SRegion) GetSecurityGroups(vpcId string) ([]SSecurityGroup, error) {
	params := map[string]string{
		"regionId": self.GetId(),
		"vpcId":    vpcId,
	}

	resp, err := self.client.DoGet("/apiproxy/v3/getSecurityGroups", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetSecurityGroups")
	}

	ret := make([]SSecurityGroup, 0)
	err = resp.Unmarshal(&ret, "returnObj")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetSecurityGroups.Unmarshal")
	}

	for i := range ret {
		ret[i].region = self
	}

	return ret, nil
}

func (self *SRegion) CreateSecurityGroup(vpcId, name string) (*SSecurityGroup, error) {
	params := map[string]jsonutils.JSONObject{
		"regionId": jsonutils.NewString(self.GetId()),
		"vpcId":    jsonutils.NewString(vpcId),
		"name":     jsonutils.NewString(name),
	}

	resp, err := self.client.DoPost("/apiproxy/v3/createSecurityGroup", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.CreateSecurityGroup.DoPost")
	}

	ret := &SSecurityGroup{}
	err = resp.Unmarshal(ret, "returnObj")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.CreateSecurityGroup.Unmarshal")
	}

	vpc, err := self.GetVpc(vpcId)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.CreateSecurityGroup.GetVpc")
	}

	ret.vpc = vpc
	ret.region = self
	return ret, nil
}
