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
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

const (
	SECGROUP_NOT_SUPPORT = "openstack_skip_security_group"
)

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
	return ""
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

func (secgroup *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	ret := []cloudprovider.ISecurityGroupRule{}
	for i := range secgroup.SecurityGroupRules {
		secgroup.SecurityGroupRules[i].region = secgroup.region
		ret = append(ret, &secgroup.SecurityGroupRules[i])
	}
	return ret, nil
}

func (secgroup *SSecurityGroup) GetStatus() string {
	return api.SECGROUP_STATUS_READY
}

func (secgroup *SSecurityGroup) Refresh() error {
	new, err := secgroup.region.GetSecurityGroup(secgroup.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(secgroup, new)
}

func (region *SRegion) DeleteSecurityGroupRule(ruleId string) error {
	resource := "/v2.0/security-group-rules/" + ruleId
	_, err := region.vpcDelete(resource)
	return err
}

func (self *SSecurityGroup) CreateRule(opts *cloudprovider.SecurityGroupRuleCreateOptions) (cloudprovider.ISecurityGroupRule, error) {
	rule, err := self.region.CreateSecurityGroupRule(self.Id, opts)
	if err != nil {
		return nil, err
	}
	return rule, nil
}

func (region *SRegion) CreateSecurityGroupRule(secgroupId string, opts *cloudprovider.SecurityGroupRuleCreateOptions) (*SSecurityGroupRule, error) {
	direction := "ingress"
	if opts.Direction == secrules.SecurityRuleEgress {
		direction = "egress"
	}

	if opts.Protocol == secrules.PROTO_ANY {
		opts.Protocol = ""
	}

	ruleInfo := map[string]interface{}{
		"direction":         direction,
		"security_group_id": secgroupId,
		"remote_ip_prefix":  opts.CIDR,
	}
	if len(opts.Protocol) > 0 {
		ruleInfo["protocol"] = opts.Protocol
	}

	params := map[string]map[string]interface{}{
		"security_group_rule": ruleInfo,
	}
	if len(opts.Ports) > 0 {
		if !strings.Contains(opts.Ports, "-") {
			params["security_group_rule"]["port_range_max"] = opts.Ports
			params["security_group_rule"]["port_range_min"] = opts.Ports
		} else {
			info := strings.Split(opts.Ports, "-")
			if len(info) == 2 {
				params["security_group_rule"]["port_range_min"] = info[0]
				params["security_group_rule"]["port_range_max"] = info[1]
			}
		}
	}
	resp, err := region.vpcPost("/v2.0/security-group-rules", params)
	if err != nil {
		return nil, err
	}
	rule := &SSecurityGroupRule{region: region}
	return rule, resp.Unmarshal(rule, "security_group_rule")
}

func (region *SRegion) DeleteSecurityGroup(secGroupId string) error {
	resource := "/v2.0/security-groups/" + secGroupId
	_, err := region.vpcDelete(resource)
	return err
}

func (secgroup *SSecurityGroup) Delete() error {
	return secgroup.region.DeleteSecurityGroup(secgroup.Id)
}

func (region *SRegion) CreateSecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (*SSecurityGroup, error) {
	params := map[string]map[string]interface{}{
		"security_group": {
			"name":        opts.Name,
			"description": opts.Desc,
		},
	}
	if len(opts.ProjectId) > 0 {
		params["security_group"]["project_id"] = opts.ProjectId
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
