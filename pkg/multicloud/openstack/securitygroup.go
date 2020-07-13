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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

const (
	SECGROUP_NOT_SUPPORT = "openstack_skip_security_group"
)

type SSecurityGroupRule struct {
	Direction       string
	Ethertype       string
	ID              string
	PortRangeMax    int
	PortRangeMin    int
	Protocol        string
	RemoteGroupID   string
	RemoteIpPrefix  string
	SecurityGroupID string
	ProjectID       string
	RevisionNumber  int
	Tags            []string
	TenantID        string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Description     string
}

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	region *SRegion

	Description        string
	ID                 string
	Name               string
	SecurityGroupRules []SSecurityGroupRule
	ProjectID          string
	RevisionNumber     int
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Tags               []string
	TenantID           string
}

func (region *SRegion) GetSecurityGroup(secgroupId string) (*SSecurityGroup, error) {
	_, resp, err := region.Get("network", "/v2.0/security-groups/"+secgroupId, "", nil)
	if err != nil {
		return nil, err
	}
	secgroup := &SSecurityGroup{region: region}
	return secgroup, resp.Unmarshal(secgroup, "security_group")
}

func (region *SRegion) GetSecurityGroups(name string) ([]SSecurityGroup, error) {
	url := "/v2.0/security-groups"
	if len(name) > 0 {
		url = fmt.Sprintf("%s?name=%s", url, name)
	}
	secgroups := []SSecurityGroup{}
	for len(url) > 0 {
		_, resp, err := region.List("network", url, "", nil)
		if err != nil {
			return nil, err
		}
		_secgroups := []SSecurityGroup{}
		err = resp.Unmarshal(&_secgroups, "security_groups")
		if err != nil {
			return nil, errors.Wrap(err, `resp.Unmarshal(&_secgroups, "security_groups")`)
		}
		secgroups = append(secgroups, _secgroups...)
		url = ""
		if resp.Contains("security_groups_links") {
			nextLink := []SNextLink{}
			err = resp.Unmarshal(&nextLink, "security_groups_links")
			if err != nil {
				return nil, errors.Wrap(err, `resp.Unmarshal(&nextLink, "security_groups_links")`)
			}
			for _, next := range nextLink {
				if next.Rel == "next" {
					url = next.Href
					break
				}
			}
		}
	}
	for i := range secgroups {
		secgroups[i].region = region
	}

	return secgroups, nil
}

func (secgroup *SSecurityGroup) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (secgroup *SSecurityGroup) GetVpcId() string {
	return "normal"
}

func (secgroup *SSecurityGroup) GetId() string {
	return secgroup.ID
}

func (secgroup *SSecurityGroup) GetGlobalId() string {
	return secgroup.ID
}

func (secgroup *SSecurityGroup) GetDescription() string {
	return secgroup.Description
}

func (secgroup *SSecurityGroup) GetName() string {
	if len(secgroup.Name) > 0 {
		return secgroup.Name
	}
	return secgroup.ID
}

func (secgrouprule *SSecurityGroupRule) toRules() ([]cloudprovider.SecurityRule, error) {
	rules := []cloudprovider.SecurityRule{}
	// 暂时忽略IPv6安全组规则,忽略远端也是安全组的规则
	if secgrouprule.Ethertype != "IPv4" || len(secgrouprule.RemoteGroupID) > 0 {
		return rules, fmt.Errorf("ethertype: %s remoteGroupId: %s", secgrouprule.Ethertype, secgrouprule.RemoteGroupID)
	}
	rule := cloudprovider.SecurityRule{
		ExternalId: secgrouprule.ID,
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
		return rules, errors.Wrap(httperrors.ErrUnsupportedProtocol, secgrouprule.Protocol)
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
			log.Errorf("failed to convert rule %s for secgroup %s(%s) error: %v", rule.ID, secgroup.Name, secgroup.ID, err)
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
	new, err := secgroup.region.GetSecurityGroup(secgroup.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(secgroup, new)
}

func (region *SRegion) delSecurityGroupRule(ruleId string) error {
	_, err := region.Delete("network", "/v2.0/security-group-rules/"+ruleId, "")
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
			_, _, err := region.Post("network", "/v2.0/security-group-rules", "", jsonutils.Marshal(params))
			if err != nil {
				return err
			}
		}
		return nil
	}
	if rule.PortEnd > 0 && rule.PortStart > 0 {
		params["security_group_rule"]["port_range_min"] = rule.PortStart
		params["security_group_rule"]["port_range_max"] = rule.PortEnd
	}
	_, _, err := region.Post("network", "/v2.0/security-group-rules", "", jsonutils.Marshal(params))
	return err
}

func (region *SRegion) DeleteSecurityGroup(secGroupId string) error {
	_, err := region.Delete("network", "/v2.0/security-groups/"+secGroupId, "")
	return err
}

func (secgroup *SSecurityGroup) Delete() error {
	return secgroup.region.DeleteSecurityGroup(secgroup.ID)
}

func (region *SRegion) CreateSecurityGroup(name, description string) (*SSecurityGroup, error) {
	params := map[string]map[string]interface{}{
		"security_group": {
			"name":        name,
			"description": description,
		},
	}
	_, resp, err := region.Post("network", "/v2.0/security-groups", "", jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	secgroup := &SSecurityGroup{region: region}
	return secgroup, resp.Unmarshal(secgroup, "security_group")
}

func (secgroup *SSecurityGroup) GetProjectId() string {
	return secgroup.TenantID
}

func (secgroup *SSecurityGroup) SyncRules(common, inAdds, outAdds, inDels, outDels []cloudprovider.SecurityRule) error {
	for _, r := range append(inDels, outDels...) {
		err := secgroup.region.delSecurityGroupRule(r.ExternalId)
		if err != nil {
			return errors.Wrapf(err, "delSecurityGroupRule(%s)", r.ExternalId)
		}
	}
	for _, r := range append(inAdds, outAdds...) {
		err := secgroup.region.addSecurityGroupRules(secgroup.ID, r)
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
