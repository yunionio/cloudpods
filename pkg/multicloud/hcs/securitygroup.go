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

package hcs

import (
	"fmt"
	"net"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SecurityGroupRule struct {
	Direction       string `json:"direction"`
	Ethertype       string `json:"ethertype"`
	Id              string `json:"id"`
	Description     string `json:"description"`
	SecurityGroupId string `json:"security_group_id"`
	RemoteGroupId   string `json:"remote_group_id"`
}

type SecurityGroupRuleDetail struct {
	Direction       string `json:"direction"`
	Ethertype       string `json:"ethertype"`
	Id              string `json:"id"`
	Description     string `json:"description"`
	PortRangeMax    int64  `json:"port_range_max"`
	PortRangeMin    int64  `json:"port_range_min"`
	Protocol        string `json:"protocol"`
	RemoteGroupId   string `json:"remote_group_id"`
	RemoteIPPrefix  string `json:"remote_ip_prefix"`
	SecurityGroupId string `json:"security_group_id"`
	TenantId        string `json:"tenant_id"`
}

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	multicloud.HcsTags
	region *SRegion

	Id                  string              `json:"id"`
	Name                string              `json:"name"`
	Description         string              `json:"description"`
	VpcId               string              `json:"vpc_id"`
	EnterpriseProjectId string              `json:"enterprise_project_id "`
	SecurityGroupRules  []SecurityGroupRule `json:"security_group_rules"`
}

// 判断是否兼容云端安全组规则
func compatibleSecurityGroupRule(r SecurityGroupRule) bool {
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

func (self *SSecurityGroup) GetId() string {
	return self.Id
}

func (self *SSecurityGroup) GetVpcId() string {
	return self.VpcId
}

func (self *SSecurityGroup) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.Id
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.Id
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
	if self.Description == self.VpcId {
		return ""
	}
	return self.Description
}

// todo: 这里需要优化查询太多了
func (self *SSecurityGroup) GetRules() ([]cloudprovider.SecurityRule, error) {
	rules := make([]cloudprovider.SecurityRule, 0)
	for _, r := range self.SecurityGroupRules {
		if !compatibleSecurityGroupRule(r) {
			continue
		}

		rule, err := self.GetSecurityRule(r.Id)
		if err != nil {
			return rules, err
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

func (self *SSecurityGroup) GetSecurityRule(ruleId string) (cloudprovider.SecurityRule, error) {
	remoteRule := SecurityGroupRuleDetail{}
	err := self.region.vpcGet("security-group-rules/"+ruleId, &remoteRule)
	if err != nil {
		return cloudprovider.SecurityRule{}, err
	}

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
		ExternalId: ruleId,
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

func (self *SRegion) GetSecurityGroupDetails(id string) (*SSecurityGroup, error) {
	ret := &SSecurityGroup{region: self}
	return ret, self.vpcGet("security-groups/"+id, ret)
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090617.html
func (self *SRegion) GetSecurityGroups(vpcId string) ([]SSecurityGroup, error) {
	query := url.Values{}
	if len(vpcId) > 0 && !utils.IsInStringArray(vpcId, []string{"default", api.NORMAL_VPC_ID}) { // vpc_id = default or normal 时报错 '{"code":"VPC.0601","message":"Query security groups error vpcId is invalid."}'
		query.Set("vpc_id", vpcId)
	}
	ret := []SSecurityGroup{}
	return ret, self.vpcList("security-groups", query, &ret)
}

func (self *SSecurityGroup) GetProjectId() string {
	return ""
}

func (self *SSecurityGroup) Delete() error {
	return self.region.DeleteSecurityGroup(self.Id)
}

func (self *SSecurityGroup) SyncRules(common, inAdds, outAdds, inDels, outDels []cloudprovider.SecurityRule) error {
	for _, r := range append(inDels, outDels...) {
		err := self.region.DeleteSecurityGroupRule(r.ExternalId)
		if err != nil {
			return errors.Wrapf(err, "delSecurityGroupRule(%s %s)", r.ExternalId, r.String())
		}
	}
	for _, r := range append(inAdds, outAdds...) {
		err := self.region.CreateSecurityGroupRule(self.Id, r)
		if err != nil {
			return errors.Wrapf(err, "addSecurityGroupRule(%d %s)", r.Priority, r.String())
		}
	}
	return nil
}

func (self *SRegion) DeleteSecurityGroup(id string) error {
	return self.vpcDelete("security-groups/" + id)
}

func (self *SRegion) DeleteSecurityGroupRule(ruleId string) error {
	return self.vpcDelete("security-group-rules/" + ruleId)
}

func (self *SRegion) CreateSecurityGroupRule(secGrpId string, rule cloudprovider.SecurityRule) error {
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
		return self.AddSecurityGroupRule(secGrpId, direction, "-1", "-1", protocal, rule.IPNet.String())
	}

	if len(rule.Ports) > 0 {
		for _, port := range rule.Ports {
			portStr := fmt.Sprintf("%d", port)
			err := self.AddSecurityGroupRule(secGrpId, direction, portStr, portStr, protocal, rule.IPNet.String())
			if err != nil {
				return err
			}
		}
	} else {
		portStart := fmt.Sprintf("%d", rule.PortStart)
		portEnd := fmt.Sprintf("%d", rule.PortEnd)
		err := self.AddSecurityGroupRule(secGrpId, direction, portStart, portEnd, protocal, rule.IPNet.String())
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SRegion) AddSecurityGroupRule(secGrpId, direction, portStart, portEnd, protocol, ipNet string) error {
	rule := map[string]interface{}{
		"security_group_id": secGrpId,
		"direction":         direction,
		"remote_ip_prefix":  ipNet,
		"ethertype":         "IPV4",
	}
	if len(portStart) > 0 && portStart != "0" && portStart != "-1" {
		rule["port_range_min"] = portStart
	}
	if len(portEnd) > 0 && portEnd != "0" && portEnd != "-1" {
		rule["port_range_max"] = portEnd
	}
	if len(protocol) > 0 {
		rule["protocol"] = protocol
	}
	params := map[string]interface{}{
		"security_group_rule": rule,
	}
	return self.vpcCreate("security-group-rules", params, nil)
}

func (self *SRegion) CreateSecurityGroup(vpcId string, name string, desc string) (*SSecurityGroup, error) {
	params := map[string]interface{}{
		"security_group": map[string]interface{}{
			"name":   name,
			"vpc_id": vpcId,
		},
	}
	ret := &SSecurityGroup{region: self}
	return ret, self.vpcCreate("security-groups", params, ret)
}

func (self *SRegion) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return self.CreateSecurityGroup(conf.VpcId, conf.Name, conf.Desc)
}

func (self *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	return self.GetSecurityGroupDetails(secgroupId)
}

func (self *SRegion) GetISecurityGroupByName(opts *cloudprovider.SecurityGroupFilterOptions) (cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := self.GetSecurityGroups(opts.VpcId)
	if err != nil {
		return nil, err
	}
	for i := range secgroups {
		if secgroups[i].GetName() == opts.Name {
			secgroups[i].region = self
			return &secgroups[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "vpc:%s, name:%s", opts.VpcId, opts.Name)
}
