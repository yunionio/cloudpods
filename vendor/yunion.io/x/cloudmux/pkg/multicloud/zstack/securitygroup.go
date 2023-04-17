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

package zstack

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSecurityGroupRule struct {
	ZStackBasic
	SecurityGroupUUID       string `json:"securityGroupUuid"`
	Type                    string `json:"type"`
	IPVersion               int    `json:"ipVersion"`
	StartPort               int    `json:"startPort"`
	EndPort                 int    `json:"endPort"`
	Protocol                string `json:"protocol"`
	State                   string `json:"state"`
	AllowedCIDR             string `json:"allowedCidr"`
	RemoteSecurityGroupUUID string `json:"remoteSecurityGroupUuid"`
	ZStackTime
}

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	ZStackTags
	region *SRegion

	ZStackBasic
	State     string `json:"state"`
	IPVersion int    `json:"ipVersion"`
	ZStackTime
	InternalID             int                  `json:"internalId"`
	Rules                  []SSecurityGroupRule `json:"rules"`
	AttachedL3NetworkUUIDs []string             `json:"attachedL3NetworkUuids"`
}

func (region *SRegion) GetSecurityGroup(secgroupId string) (*SSecurityGroup, error) {
	secgroup := &SSecurityGroup{region: region}
	return secgroup, region.client.getResource("security-groups", secgroupId, secgroup)
}

func (region *SRegion) GetSecurityGroups(secgroupId string, instanceId string, name string) ([]SSecurityGroup, error) {
	secgroups := []SSecurityGroup{}
	params := url.Values{}
	if len(secgroupId) > 0 {
		params.Add("q", "uuid="+secgroupId)
	}
	if len(instanceId) > 0 {
		params.Add("q", "vmNic.vmInstanceUuid="+instanceId)
	}
	if len(name) > 0 {
		params.Add("q", "name="+name)
	}
	err := region.client.listAll("security-groups", params, &secgroups)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(secgroups); i++ {
		secgroups[i].region = region
	}
	return secgroups, nil
}

func (self *SSecurityGroup) GetVpcId() string {
	return api.NORMAL_VPC_ID
}

func (self *SSecurityGroup) GetId() string {
	return self.UUID
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.UUID
}

func (self *SSecurityGroup) GetDescription() string {
	return self.Description
}

func (rule *SSecurityGroupRule) toRule() (cloudprovider.SecurityRule, error) {
	r := cloudprovider.SecurityRule{
		SecurityRule: secrules.SecurityRule{
			Direction: secrules.DIR_IN,
			Action:    secrules.SecurityRuleAllow,
			Priority:  1,
			Protocol:  secrules.PROTO_ANY,
			PortStart: rule.StartPort,
			PortEnd:   rule.EndPort,
		},
	}
	r.ParseCIDR(rule.AllowedCIDR)
	if rule.Type == "Egress" {
		r.Direction = secrules.DIR_OUT
	}
	if rule.Protocol != "ALL" {
		r.Protocol = strings.ToLower(rule.Protocol)
	}
	return r, nil
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.SecurityRule, error) {
	rules := []cloudprovider.SecurityRule{}
	rules = append(rules, cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("out:allow any")})
	for i := 0; i < len(self.Rules); i++ {
		if self.Rules[i].IPVersion == 4 {
			rule, err := self.Rules[i].toRule()
			if err != nil {
				return nil, err
			}
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

func (self *SSecurityGroup) GetName() string {
	return self.Name
}

func (self *SSecurityGroup) GetStatus() string {
	return ""
}

func (self *SSecurityGroup) IsEmulated() bool {
	return false
}

func (self *SSecurityGroup) Refresh() error {
	new, err := self.region.GetSecurityGroup(self.UUID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SSecurityGroup) GetProjectId() string {
	return ""
}

func (region *SRegion) AddSecurityGroupRule(secgroupId string, rules []secrules.SecurityRule) error {
	ruleParam := []map[string]interface{}{}
	for _, rule := range rules {
		Type := "Ingress"
		if rule.Direction == secrules.DIR_OUT {
			Type = "Egress"
		}
		protocol := "ALL"
		if rule.Protocol != secrules.PROTO_ANY {
			protocol = strings.ToUpper(rule.Protocol)
		}
		if rule.Protocol == secrules.PROTO_ICMP || rule.Protocol == secrules.PROTO_ANY {
			rule.PortStart = -1
			rule.PortEnd = -1
			rule.Ports = []int{}
		}
		if len(rule.Ports) > 0 {
			for _, port := range rule.Ports {
				ruleParam = append(ruleParam, map[string]interface{}{
					"type":        Type,
					"startPort":   port,
					"endPort":     port,
					"protocol":    protocol,
					"allowedCidr": rule.IPNet.String(),
				})
			}
		} else {
			if protocol != "ALL" {
				// TCP UDP端口不能为-1
				if (rule.Protocol == secrules.PROTO_TCP || rule.Protocol == secrules.PROTO_UDP) &&
					(rule.PortStart <= 0 && rule.PortEnd <= 0) {
					rule.PortStart = 0
					rule.PortEnd = 65535
				}
				ruleParam = append(ruleParam, map[string]interface{}{
					"type":        Type,
					"startPort":   rule.PortStart,
					"endPort":     rule.PortEnd,
					"protocol":    protocol,
					"allowedCidr": rule.IPNet.String(),
				})
			} else {
				ruleParam = append(ruleParam, map[string]interface{}{
					"type":        Type,
					"protocol":    protocol,
					"allowedCidr": rule.IPNet.String(),
				})
			}
		}
	}
	if len(ruleParam) > 0 {
		params := map[string]interface{}{
			"params": map[string]interface{}{
				"rules": ruleParam,
			},
		}
		return region.client.create(fmt.Sprintf("security-groups/%s/rules", secgroupId), jsonutils.Marshal(params), nil)
	}
	return nil
}

func (region *SRegion) DeleteSecurityGroupRules(ruleIds []string) error {
	if len(ruleIds) > 0 {
		ids := []string{}
		for _, ruleId := range ruleIds {
			ids = append(ids, fmt.Sprintf("ruleUuids=%s", ruleId))
		}
		resource := fmt.Sprintf("security-groups/rules?%s", strings.Join(ids, "&"))
		return region.client.delete(resource, "", "")
	}
	return nil
}

func (region *SRegion) CreateSecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (*SSecurityGroup, error) {
	secgroup := &SSecurityGroup{region: region}
	params := map[string]map[string]string{
		"params": {
			"name":        opts.Name,
			"description": opts.Desc,
		},
	}
	err := region.client.create("security-groups", jsonutils.Marshal(params), secgroup)
	if err != nil {
		return nil, err
	}
	if opts.OnCreated != nil {
		opts.OnCreated(secgroup.UUID)
	}
	rules := opts.InRules.AllowList()
	rules = append(rules, opts.OutRules.AllowList()...)
	return secgroup, region.AddSecurityGroupRule(secgroup.UUID, rules)
}

func (self *SSecurityGroup) Delete() error {
	return self.region.client.delete("security-groups", self.UUID, "Permissive")
}
