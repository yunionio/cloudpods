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
	return ""
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

func (self *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	ret := []cloudprovider.ISecurityGroupRule{}
	for i := range self.Rules {
		self.Rules[i].region = self.region
		ret = append(ret, &self.Rules[i])
	}
	return ret, nil
}

func (self *SSecurityGroup) GetName() string {
	return self.Name
}

func (self *SSecurityGroup) GetStatus() string {
	return api.SECGROUP_STATUS_READY
}

func (self *SSecurityGroup) Refresh() error {
	group, err := self.region.GetSecurityGroup(self.UUID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, group)
}

func (self *SSecurityGroup) GetProjectId() string {
	return ""
}

func (self *SSecurityGroup) CreateRule(opts *cloudprovider.SecurityGroupRuleCreateOptions) (cloudprovider.ISecurityGroupRule, error) {
	rule, err := self.region.CreateSecurityGroupRule(self.UUID, opts)
	if err != nil {
		return nil, err
	}
	return rule, nil
}

func (region *SRegion) CreateSecurityGroupRule(secgroupId string, opts *cloudprovider.SecurityGroupRuleCreateOptions) (*SSecurityGroupRule, error) {
	ruleParam := map[string]interface{}{
		"allowedCidr": opts.CIDR,
		"type":        "Ingress",
		"protocol":    "ALL",
	}
	if opts.Direction == secrules.DIR_OUT {
		ruleParam["type"] = "Egress"
	}
	if opts.Protocol != secrules.PROTO_ANY {
		ruleParam["protocol"] = strings.ToUpper(opts.Protocol)
	}
	if opts.Protocol == secrules.PROTO_ICMP || opts.Protocol == secrules.PROTO_ANY {
		opts.Ports = ""
	}
	if opts.Protocol == secrules.PROTO_TCP || opts.Protocol == secrules.PROTO_UDP {
		if len(opts.Ports) == 0 {
			ruleParam["startPort"] = "0"
			ruleParam["endPort"] = "65535"
		} else {
			if strings.Contains(opts.Ports, "-") {
				info := strings.Split(opts.Ports, "-")
				if len(info) == 2 {
					ruleParam["startPort"] = info[0]
					ruleParam["endPort"] = info[1]
				}
			} else {
				ruleParam["startPort"] = opts.Ports
				ruleParam["endPort"] = opts.Ports
			}
		}
	}
	params := map[string]interface{}{
		"params": map[string]interface{}{
			"rules": []map[string]interface{}{ruleParam},
		},
	}
	rule := &SSecurityGroupRule{region: region}
	err := region.client.create(fmt.Sprintf("security-groups/%s/rules", secgroupId), jsonutils.Marshal(params), rule)
	if err != nil {
		return nil, err
	}
	return rule, nil
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
	return secgroup, nil
}

func (self *SSecurityGroup) Delete() error {
	return self.region.client.delete("security-groups", self.UUID, "Permissive")
}
