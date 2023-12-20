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

package huawei

import (
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	HuaweiTags
	region *SRegion

	Id                  string              `json:"id"`
	Name                string              `json:"name"`
	Description         string              `json:"description"`
	VpcId               string              `json:"vpc_id"`
	EnterpriseProjectId string              `json:"enterprise_project_id "`
	SecurityGroupRules  []SecurityGroupRule `json:"security_group_rules"`
}

func (self *SSecurityGroup) GetId() string {
	return self.Id
}

func (self *SSecurityGroup) GetVpcId() string {
	return ""
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

func (self *SSecurityGroup) GetTags() (map[string]string, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SSecurityGroup) GetStatus() string {
	return api.SECGROUP_STATUS_READY
}

func (self *SSecurityGroup) Refresh() error {
	group, err := self.region.GetSecurityGroup(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, group)
}

func (self *SSecurityGroup) GetDescription() string {
	return self.Description
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	ret := make([]cloudprovider.ISecurityGroupRule, 0)
	rules, err := self.region.GetSecurityGroupRules(self.Id)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		rules[i].secgroup = self
		ret = append(ret, &rules[i])
	}
	return ret, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v3&api=ShowSecurityGroup
func (self *SRegion) GetSecurityGroup(id string) (*SSecurityGroup, error) {
	ret := &SSecurityGroup{region: self}
	resp, err := self.list(SERVICE_VPC_V3, "vpc/security-groups/"+id, nil)
	if err != nil {
		return nil, err
	}
	return ret, resp.Unmarshal(ret, "security_group")
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v3&api=ListSecurityGroups
func (self *SRegion) GetSecurityGroups(name string) ([]SSecurityGroup, error) {
	ret := []SSecurityGroup{}
	params := url.Values{}
	if len(name) > 0 {
		params.Set("name", name)
	}
	for {
		resp, err := self.list(SERVICE_VPC_V3, "vpc/security-groups", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			SecurityGroups []SSecurityGroup
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrapf(err, "Unmarshal")
		}
		ret = append(ret, part.SecurityGroups...)
		if len(part.SecurityGroups) == 0 {
			break
		}
		params.Set("marker", part.SecurityGroups[len(part.SecurityGroups)-1].Id)
	}
	return ret, nil
}

func (self *SSecurityGroup) CreateRule(opts *cloudprovider.SecurityGroupRuleCreateOptions) (cloudprovider.ISecurityGroupRule, error) {
	rule, err := self.region.CreateSecurityGroupRule(self.Id, opts)
	if err != nil {
		return nil, err
	}
	rule.secgroup = self
	return rule, nil
}

func (self *SSecurityGroup) GetProjectId() string {
	return self.EnterpriseProjectId
}

func (self *SSecurityGroup) Delete() error {
	return self.region.DeleteSecurityGroup(self.Id)
}
