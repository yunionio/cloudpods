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

package cloudpods

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	CloudpodsTags
	region *SRegion

	api.SecgroupDetails
}

func (self *SSecurityGroup) GetName() string {
	return self.Name
}

func (self *SSecurityGroup) GetDescription() string {
	return self.Description
}

func (self *SSecurityGroup) GetId() string {
	return self.Id
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.Id
}

func (self *SSecurityGroup) GetStatus() string {
	return self.Status
}

func (self *SSecurityGroup) GetProjectId() string {
	return self.TenantId
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	ret := []cloudprovider.ISecurityGroupRule{}
	rules := []SecurityGroupRule{}
	err := self.region.list(&modules.SecGroupRules, map[string]interface{}{"scope": "system", "secgroup_id": self.Id}, &rules)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		rules[i].region = self.region
		ret = append(ret, &rules[i])
	}
	return ret, nil
}

func (self *SSecurityGroup) GetVpcId() string {
	return api.NORMAL_VPC_ID
}

func (self *SSecurityGroup) GetReferences() ([]cloudprovider.SecurityGroupReference, error) {
	params := map[string]interface{}{
		"secgroup_id": self.Id,
	}
	servers := []SInstance{}
	err := self.region.list(&modules.Servers, params, &servers)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.SecurityGroupReference{}
	for i := range servers {
		ret = append(ret, cloudprovider.SecurityGroupReference{
			Id:   servers[i].Id,
			Name: servers[i].Name,
		})
	}
	return ret, nil
}

func (self *SRegion) DeleteSecRule(id string) error {
	return self.cli.delete(&modules.SecGroupRules, id)
}

func (self *SRegion) CreateSecRule(secId string, opts *cloudprovider.SecurityGroupRuleCreateOptions) error {
	input := api.SSecgroupRuleCreateInput{}
	input.SecgroupId = secId
	input.Priority = &opts.Priority
	input.Action = string(opts.Action)
	input.Protocol = string(opts.Protocol)
	input.Direction = string(opts.Direction)
	input.Description = opts.Desc
	input.CIDR = opts.CIDR

	input.Ports = opts.Ports
	ret := struct{}{}
	return self.create(&modules.SecGroupRules, input, &ret)
}

func (self *SSecurityGroup) Delete() error {
	return self.region.cli.delete(&modules.SecGroups, self.Id)
}

func (self *SRegion) GetSecurityGroups() ([]SSecurityGroup, error) {
	params := map[string]interface{}{
		"cloud_env": "onpremise",
	}
	ret := []SSecurityGroup{}
	return ret, self.cli.list(&modules.SecGroups, params, &ret)
}

func (self *SRegion) GetSecurityGroup(id string) (*SSecurityGroup, error) {
	secgroup := &SSecurityGroup{region: self}
	return secgroup, self.cli.get(&modules.SecGroups, id, nil, secgroup)
}

func (self *SRegion) CreateISecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	params := map[string]interface{}{
		"name":        opts.Name,
		"description": opts.Desc,
	}
	if len(opts.ProjectId) > 0 {
		params["project_id"] = opts.ProjectId
	}
	secgroup := &SSecurityGroup{region: self}
	return secgroup, self.create(&modules.SecGroups, params, secgroup)
}

func (self *SRegion) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := self.GetSecurityGroups()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSecurityGroup{}
	for i := range secgroups {
		secgroups[i].region = self
		ret = append(ret, &secgroups[i])
	}
	return ret, nil
}

func (self *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	secgroup, err := self.GetSecurityGroup(secgroupId)
	if err != nil {
		return nil, err
	}
	return secgroup, nil
}
