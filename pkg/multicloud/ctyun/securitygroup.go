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
	"fmt"
	"net"
	"sort"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
)

type SSecurityGroup struct {
	region *SRegion

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

// 将安全组规则全部转换为等价的allow规则
func SecurityRuleSetToAllowSet(srs secrules.SecurityRuleSet) secrules.SecurityRuleSet {
	inRuleSet := secrules.SecurityRuleSet{}
	outRuleSet := secrules.SecurityRuleSet{}

	for _, rule := range srs {
		if rule.Direction == secrules.SecurityRuleIngress {
			inRuleSet = append(inRuleSet, rule)
		}

		if rule.Direction == secrules.SecurityRuleEgress {
			outRuleSet = append(outRuleSet, rule)
		}
	}

	sort.Sort(inRuleSet)
	sort.Sort(outRuleSet)

	inRuleSet = inRuleSet.AllowList()
	// out方向空规则默认全部放行
	if outRuleSet.Len() == 0 {
		_, ipNet, _ := net.ParseCIDR("0.0.0.0/0")
		outRuleSet = append(outRuleSet, secrules.SecurityRule{
			Priority:  0,
			Action:    secrules.SecurityRuleAllow,
			IPNet:     ipNet,
			Protocol:  secrules.PROTO_ANY,
			Direction: secrules.SecurityRuleEgress,
			PortStart: -1,
			PortEnd:   -1,
		})
	}
	outRuleSet = outRuleSet.AllowList()

	ret := secrules.SecurityRuleSet{}
	ret = append(ret, inRuleSet...)
	ret = append(ret, outRuleSet...)
	return ret
}

func (self *SSecurityGroup) GetRulesWithExtId() ([]secrules.SecurityRule, error) {
	_rules, err := self.region.GetSecurityGroupRules(self.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "SSecurityGroup.GetRulesWithExtId.GetSecurityGroupRules")
	}

	rules := make([]secrules.SecurityRule, 0)
	for _, r := range _rules {
		if !compatibleSecurityGroupRule(r) {
			continue
		}

		rule, err := self.GetSecurityRule(r, true)
		if err != nil {
			return rules, err
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

func (self *SRegion) syncSecgroupRules(secgroupId string, rules []secrules.SecurityRule) error {
	var DeleteRules []secrules.SecurityRule
	var AddRules []secrules.SecurityRule

	if secgroup, err := self.GetSecurityGroupDetails(secgroupId); err != nil {
		return errors.Wrapf(err, "syncSecgroupRules.GetSecurityGroupDetails(%s)", secgroupId)
	} else {
		remoteRules, err := secgroup.GetRulesWithExtId()
		if err != nil {
			return errors.Wrap(err, "secgroup.GetRulesWithExtId")
		}

		sort.Sort(secrules.SecurityRuleSet(rules))
		sort.Sort(secrules.SecurityRuleSet(remoteRules))

		i, j := 0, 0
		for i < len(rules) || j < len(remoteRules) {
			if i < len(rules) && j < len(remoteRules) {
				permissionStr := remoteRules[j].String()
				ruleStr := rules[i].String()
				cmp := strings.Compare(permissionStr, ruleStr)
				if cmp == 0 {
					// DeleteRules = append(DeleteRules, remoteRules[j])
					// AddRules = append(AddRules, rules[i])
					i += 1
					j += 1
				} else if cmp > 0 {
					DeleteRules = append(DeleteRules, remoteRules[j])
					j += 1
				} else {
					AddRules = append(AddRules, rules[i])
					i += 1
				}
			} else if i >= len(rules) {
				DeleteRules = append(DeleteRules, remoteRules[j])
				j += 1
			} else if j >= len(remoteRules) {
				AddRules = append(AddRules, rules[i])
				i += 1
			}
		}
	}

	for _, r := range DeleteRules {
		// r.Description 实际存储的是ruleId
		if err := self.delSecurityGroupRule(r.Description); err != nil {
			log.Errorf("delSecurityGroupRule %v error: %s", r, err.Error())
			return err
		}
	}

	for _, r := range AddRules {
		if err := self.addSecurityGroupRules(secgroupId, &r); err != nil {
			log.Errorf("addSecurityGroupRule %v error: %s", r, err.Error())
			return err
		}
	}

	return nil
}

func (self *SRegion) delSecurityGroupRule(secGrpRuleId string) error {
	return self.DeleteSecurityGroupRule(secGrpRuleId)
}

func (self *SRegion) addSecurityGroupRules(secGrpId string, rule *secrules.SecurityRule) error {
	direction := ""
	if rule.Direction == secrules.SecurityRuleIngress {
		direction = "ingress"
	} else {
		direction = "egress"
	}

	protocal := rule.Protocol
	if rule.Protocol == secrules.PROTO_ANY {
		protocal = ""
	}

	// imcp协议默认为any
	if rule.Protocol == secrules.PROTO_ICMP {
		return self.addSecurityGroupRule(secGrpId, direction, "-1", "-1", protocal, rule.IPNet.String())
	}

	if len(rule.Ports) > 0 {
		for _, port := range rule.Ports {
			portStr := fmt.Sprintf("%d", port)
			err := self.addSecurityGroupRule(secGrpId, direction, portStr, portStr, protocal, rule.IPNet.String())
			if err != nil {
				return err
			}
		}
	} else {
		portStart := fmt.Sprintf("%d", rule.PortStart)
		portEnd := fmt.Sprintf("%d", rule.PortEnd)
		err := self.addSecurityGroupRule(secGrpId, direction, portStart, portEnd, protocal, rule.IPNet.String())
		if err != nil {
			return err
		}
	}

	return nil
}

func (self *SSecurityGroup) SyncRules(rules []secrules.SecurityRule) error {
	rules = SecurityRuleSetToAllowSet(rules)
	return self.region.syncSecgroupRules(self.ResSecurityGroupID, rules)
}

func (self *SSecurityGroup) Delete() error {
	return self.region.DeleteSecurityGroup(self.GetId())
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
		return "classic"
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
	}

	if len(vpcId) > 0 {
		params["vpcId"] = vpcId
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
		"name":     jsonutils.NewString(name),
	}

	if len(vpcId) > 0 && (vpcId != "classic" && vpcId != "normal") {
		params["vpcId"] = jsonutils.NewString(vpcId)
	}

	resp, err := self.client.DoPost("/apiproxy/v3/createSecurityGroup", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.CreateSecurityGroup.DoPost")
	}

	secgroupId, err := resp.GetString("returnObj", "id")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.CreateSecurityGroup.GetSecgroupId")
	}

	secgroup, err := self.GetSecurityGroupDetails(secgroupId)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.CreateSecurityGroup.GetISecurityGroupById")
	}

	secgroup.region = self
	return secgroup, nil
}

func (self *SRegion) DeleteSecurityGroupRule(securityGroupRuleId string) error {
	params := map[string]jsonutils.JSONObject{
		"regionId":            jsonutils.NewString(self.GetId()),
		"securityGroupRuleId": jsonutils.NewString(securityGroupRuleId),
	}

	_, err := self.client.DoPost("/apiproxy/v3/deleteSecurityGroupRule", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.DeleteSecurityGroupRule.DoPost")
	}

	return err
}

func (self *SRegion) addSecurityGroupRule(secGrpId, direction, portStart, portEnd, protocol, ipNet string) error {
	secgroupObj := jsonutils.NewDict()
	secgroupObj.Add(jsonutils.NewString(self.GetId()), "regionId")
	secgroupObj.Add(jsonutils.NewString(secGrpId), "securityGroupId")
	secgroupObj.Add(jsonutils.NewString(direction), "direction")
	secgroupObj.Add(jsonutils.NewString(ipNet), "remoteIpPrefix")
	secgroupObj.Add(jsonutils.NewString("IPv4"), "ethertype")
	// 端口为空或者1-65535
	if len(portStart) > 0 && portStart != "0" && portStart != "-1" {
		secgroupObj.Add(jsonutils.NewString(portStart), "portRangeMin")
	}
	if len(portEnd) > 0 && portEnd != "0" && portEnd != "-1" {
		secgroupObj.Add(jsonutils.NewString(portEnd), "portRangeMax")
	}
	if len(protocol) > 0 {
		secgroupObj.Add(jsonutils.NewString(protocol), "protocol")
	}

	params := map[string]jsonutils.JSONObject{
		"jsonStr": secgroupObj,
	}

	resp, err := self.client.DoPost("/apiproxy/v3/createSecurityGroupRule", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.DoPost")
	}

	rule := SSecurityGroupRule{}
	err = resp.Unmarshal(&rule, "returnObj")
	if err != nil {
		return errors.Wrap(err, "SRegion.Unmarshal")
	}

	return nil
}
