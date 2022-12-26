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

package openstack

import (
	"fmt"
	"net/url"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

const (
	SECGROUP_NOT_SUPPORT = "openstack_skip_security_group"
)

type SSecurityGroupRule struct {
	Direction       string
	Ethertype       string
	Id              string
	PortRangeMax    int
	PortRangeMin    int
	Protocol        string
	RemoteGroupId   string
	RemoteIpPrefix  string
	SecurityGroupId string
	ProjectId       string
	RevisionNumber  int
	Tags            []string
	TenantId        string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Description     string
}

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	OpenStackTags
	region *SRegion

	Description        string
	Id                 string
	Name               string
	SecurityGroupRules []SSecurityGroupRule
	ProjectId          string
	RevisionNumber     int
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Tags               []string
	TenantId           string
}

func (region *SRegion) GetSecurityGroup(secgroupId string) (*SSecurityGroup, error) {
	resource := "/v2.0/security-groups/" + secgroupId
	resp, err := region.vpcGet(resource)
	if err != nil {
		return nil, errors.Wrap(err, "vpcGet")
	}
	secgroup := &SSecurityGroup{region: region}
	err = resp.Unmarshal(secgroup, "security_group")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return secgroup, nil
}

func (region *SRegion) GetSecurityGroups(projectId, name string) ([]SSecurityGroup, error) {
	secgroups := []SSecurityGroup{}
	resource := "/v2.0/security-groups"
	query := url.Values{}
	if len(name) > 0 {
		query.Set("name", name)
	}
	if len(projectId) > 0 {
		query.Set("project_id", projectId)
	}
	for {
		resp, err := region.vpcList(resource, query)
		if err != nil {
			return nil, errors.Wrap(err, "vpcList")
		}
		part := struct {
			SecurityGroups      []SSecurityGroup
			SecurityGroupsLinks SNextLinks
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		secgroups = append(secgroups, part.SecurityGroups...)
		marker := part.SecurityGroupsLinks.GetNextMark()
		if len(marker) == 0 {
			break
		}
		query.Set("marker", marker)
	}
	return secgroups, nil
}

func (secgroup *SSecurityGroup) GetVpcId() string {
	return "normal"
}

func (secgroup *SSecurityGroup) GetId() string {
	return secgroup.Id
}

func (secgroup *SSecurityGroup) GetGlobalId() string {
	return secgroup.Id
}

func (secgroup *SSecurityGroup) GetDescription() string {
	return secgroup.Description
}

func (secgroup *SSecurityGroup) GetName() string {
	if len(secgroup.Name) > 0 {
		return secgroup.Name
	}
	return secgroup.Id
}

func (secgrouprule *SSecurityGroupRule) toRules() ([]cloudprovider.SecurityRule, error) {
	rules := []cloudprovider.SecurityRule{}
	// 暂时忽略IPv6安全组规则,忽略远端也是安全组的规则
	if secgrouprule.Ethertype != "IPv4" || len(secgrouprule.RemoteGroupId) > 0 {
		return rules, fmt.Errorf("ethertype: %s remoteGroupId: %s", secgrouprule.Ethertype, secgrouprule.RemoteGroupId)
	}
	rule := cloudprovider.SecurityRule{
		ExternalId: secgrouprule.Id,
		SecurityRule: secrules.SecurityRule{
			Direction:   secrules.DIR_IN,
			Action:      secrules.SecurityRuleAllow,
			Description: secgrouprule.Description,
		},
	}
	if utils.IsInStringArray(secgrouprule.Protocol, []string{"any", "-1", ""}) {
		rule.Protocol = secrules.PROTO_ANY
	} else if utils.IsInStringArray(secgrouprule.Protocol, []string{"6", "tcp"}) {
		rule.Protocol = secrules.PROTO_TCP
	} else if utils.IsInStringArray(secgrouprule.Protocol, []string{"17", "udp"}) {
		rule.Protocol = secrules.PROTO_UDP
	} else if utils.IsInStringArray(secgrouprule.Protocol, []string{"1", "icmp"}) {
		rule.Protocol = secrules.PROTO_ICMP
	} else {
		return rules, errors.Wrap(cloudprovider.ErrUnsupportedProtocol, secgrouprule.Protocol)
	}
	if secgrouprule.Direction == "egress" {
		rule.Direction = secrules.DIR_OUT
	}
	if len(secgrouprule.RemoteIpPrefix) == 0 {
		secgrouprule.RemoteIpPrefix = "0.0.0.0/0"
	}

	rule.ParseCIDR(secgrouprule.RemoteIpPrefix)
	if secgrouprule.PortRangeMax > 0 && secgrouprule.PortRangeMin > 0 {
		if secgrouprule.PortRangeMax == secgrouprule.PortRangeMin {
			rule.Ports = []int{secgrouprule.PortRangeMax}
		} else {
			rule.PortStart = secgrouprule.PortRangeMin
			rule.PortEnd = secgrouprule.PortRangeMax
		}
	}
	err := rule.ValidateRule()
	if err != nil && err != secrules.ErrInvalidPriority {
		return rules, errors.Wrap(err, "rule.ValidateRule")
	}
	return []cloudprovider.SecurityRule{rule}, nil
}

func (secgroup *SSecurityGroup) GetRules() ([]cloudprovider.SecurityRule, error) {
	rules := []cloudprovider.SecurityRule{}
	for _, rule := range secgroup.SecurityGroupRules {
		subRules, err := rule.toRules()
		if err != nil {
			log.Errorf("failed to convert rule %s for secgroup %s(%s) error: %v", rule.Id, secgroup.Name, secgroup.Id, err)
			continue
		}
		rules = append(rules, subRules...)
	}
	return rules, nil
}

func (secgroup *SSecurityGroup) GetStatus() string {
	return ""
}

func (secgroup *SSecurityGroup) IsEmulated() bool {
	return false
}

func (secgroup *SSecurityGroup) Refresh() error {
	new, err := secgroup.region.GetSecurityGroup(secgroup.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(secgroup, new)
}

func (region *SRegion) delSecurityGroupRule(ruleId string) error {
	resource := "/v2.0/security-group-rules/" + ruleId
	_, err := region.vpcDelete(resource)
	return err
}

func (region *SRegion) addSecurityGroupRules(secgroupId string, rule cloudprovider.SecurityRule) error {
	direction := "ingress"
	if rule.Direction == secrules.SecurityRuleEgress {
		direction = "egress"
	}

	if rule.Protocol == secrules.PROTO_ANY {
		rule.Protocol = ""
	}

	ruleInfo := map[string]interface{}{
		"direction":         direction,
		"security_group_id": secgroupId,
		"remote_ip_prefix":  rule.IPNet.String(),
	}
	if len(rule.Protocol) > 0 {
		ruleInfo["protocol"] = rule.Protocol
	}

	params := map[string]map[string]interface{}{
		"security_group_rule": ruleInfo,
	}
	if len(rule.Ports) > 0 {
		for _, port := range rule.Ports {
			params["security_group_rule"]["port_range_max"] = port
			params["security_group_rule"]["port_range_min"] = port
			resource := "/v2.0/security-group-rules"
			_, err := region.vpcPost(resource, params)
			if err != nil {
				return errors.Wrap(err, "vpcPost")
			}
		}
		return nil
	}
	if rule.PortEnd > 0 && rule.PortStart > 0 {
		params["security_group_rule"]["port_range_min"] = rule.PortStart
		params["security_group_rule"]["port_range_max"] = rule.PortEnd
	}
	_, err := region.vpcPost("/v2.0/security-group-rules", params)
	return err
}

func (region *SRegion) DeleteSecurityGroup(secGroupId string) error {
	resource := "/v2.0/security-groups/" + secGroupId
	_, err := region.vpcDelete(resource)
	return err
}

func (secgroup *SSecurityGroup) Delete() error {
	return secgroup.region.DeleteSecurityGroup(secgroup.Id)
}

func (region *SRegion) CreateSecurityGroup(projectId, name, description string) (*SSecurityGroup, error) {
	params := map[string]map[string]interface{}{
		"security_group": {
			"name":        name,
			"description": description,
		},
	}
	if len(projectId) > 0 {
		params["security_group"]["project_id"] = projectId
	}
	resp, err := region.vpcPost("/v2.0/security-groups", params)
	if err != nil {
		return nil, errors.Wrap(err, "vpcPost")
	}
	secgroup := &SSecurityGroup{region: region}
	err = resp.Unmarshal(secgroup, "security_group")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return secgroup, nil
}

func (secgroup *SSecurityGroup) GetProjectId() string {
	return secgroup.TenantId
}

func (secgroup *SSecurityGroup) SyncRules(common, inAdds, outAdds, inDels, outDels []cloudprovider.SecurityRule) error {
	for _, r := range append(inDels, outDels...) {
		err := secgroup.region.delSecurityGroupRule(r.ExternalId)
		if err != nil {
			return errors.Wrapf(err, "delSecurityGroupRule(%s)", r.ExternalId)
		}
	}
	for _, r := range append(inAdds, outAdds...) {
		err := secgroup.region.addSecurityGroupRules(secgroup.Id, r)
		if err != nil {
			if jsonError, ok := err.(*httputils.JSONClientError); ok {
				if jsonError.Class == "SecurityGroupRuleExists" {
					continue
				}
			}
			return errors.Wrapf(err, "addSecgroupRules(%s)", r.String())
		}
	}
	return nil
}
