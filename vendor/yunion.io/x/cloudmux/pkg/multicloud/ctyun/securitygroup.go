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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	apis "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	CtyunTags
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

func (self *SRegion) delSecurityGroupRule(secGrpRuleId string) error {
	return self.DeleteSecurityGroupRule(secGrpRuleId)
}

func (self *SRegion) AddSecurityGroupRules(secGrpId string, rule cloudprovider.SecurityRule) error {
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

func (self *SSecurityGroup) SyncRules(common, inAdds, outAdds, inDels, outDels []cloudprovider.SecurityRule) error {
	for _, r := range append(inDels, outDels...) {
		err := self.region.delSecurityGroupRule(r.ExternalId)
		if err != nil {
			return errors.Wrapf(err, "delSecurityGroupRule(%s)", r.ExternalId)
		}
	}
	for _, r := range append(inAdds, outAdds...) {
		err := self.region.AddSecurityGroupRules(self.ResSecurityGroupID, r)
		if err != nil {
			return errors.Wrapf(err, "addSecurityGroupRule(%d %s)", r.Priority, r.String())
		}
	}
	return nil
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

func (self *SSecurityGroup) GetRules() ([]cloudprovider.SecurityRule, error) {
	_rules, err := self.region.GetSecurityGroupRules(self.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "SSecurityGroup.GetRules.GetSecurityGroupRules")
	}

	rules := make([]cloudprovider.SecurityRule, 0)
	for _, r := range _rules {
		if !compatibleSecurityGroupRule(r) {
			continue
		}

		rule, err := self.GetSecurityRule(r)
		if err != nil {
			return rules, err
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

func (self *SSecurityGroup) GetSecurityRule(remoteRule SSecurityGroupRule) (cloudprovider.SecurityRule, error) {
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
		return cloudprovider.SecurityRule{}, err
	}

	rule := cloudprovider.SecurityRule{
		ExternalId: remoteRule.ID,
		SecurityRule: secrules.SecurityRule{
			Priority:    1,
			Action:      secrules.SecurityRuleAllow,
			IPNet:       ipNet,
			Protocol:    protocol,
			Direction:   direction,
			PortStart:   portStart,
			PortEnd:     portEnd,
			Ports:       nil,
			Description: remoteRule.Description,
		},
	}

	err = rule.ValidateRule()
	return rule, err
}

func (self *SSecurityGroup) GetVpcId() string {
	return apis.NORMAL_VPC_ID
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

	if len(vpcId) > 0 && vpcId != apis.NORMAL_VPC_ID {
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
