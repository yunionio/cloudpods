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

package hcso

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei"
)

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090615.html
type SSecurityGroup struct {
	multicloud.SSecurityGroup
	huawei.HuaweiTags
	region *SRegion

	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	Description         string              `json:"description"`
	EnterpriseProjectID string              `json:"enterprise_project_id "`
	SecurityGroupRules  []SecurityGroupRule `json:"security_group_rules"`
}

func (self *SSecurityGroup) GetId() string {
	return self.ID
}

func (self *SSecurityGroup) GetVpcId() string {
	return ""
}

func (self *SSecurityGroup) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.ID
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.ID
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

func (self *SSecurityGroup) IsEmulated() bool {
	return false
}

func (self *SSecurityGroup) GetDescription() string {
	return self.Description
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	ret := make([]cloudprovider.ISecurityGroupRule, 0)
	rules, err := self.region.GetSecurityGroupRules(self.ID)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		rules[i].secgroup = self
		ret = append(ret, &rules[i])
	}
	return ret, nil

}

func (self *SRegion) GetSecurityGroup(id string) (*SSecurityGroup, error) {
	ret := &SSecurityGroup{region: self}
	resp, err := self.list(SERVICE_VPC, "vpc/security-groups/"+id, nil)
	if err != nil {
		return nil, err
	}
	return ret, resp.Unmarshal(ret, "security_group")
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090617.html
func (self *SRegion) GetSecurityGroups(vpcId string, name string) ([]SSecurityGroup, error) {
	querys := map[string]string{}
	if len(vpcId) > 0 && !utils.IsInStringArray(vpcId, []string{"default", api.NORMAL_VPC_ID}) { // vpc_id = default or normal 时报错 '{"code":"VPC.0601","message":"Query security groups error vpcId is invalid."}'
		querys["vpc_id"] = vpcId
	}

	securitygroups := make([]SSecurityGroup, 0)
	err := doListAllWithMarker(self.ecsClient.SecurityGroups.List, querys, &securitygroups)
	if err != nil {
		return nil, err
	}

	// security 中的vpc字段只是一个标识，实际可以跨vpc使用
	for i := range securitygroups {
		securitygroup := &securitygroups[i]
		securitygroup.region = self
	}

	result := []SSecurityGroup{}
	for _, secgroup := range securitygroups {
		if len(name) == 0 || secgroup.Name == name {
			result = append(result, secgroup)
		}
	}

	return result, nil
}

func (self *SSecurityGroup) GetProjectId() string {
	return ""
}

func (self *SSecurityGroup) Delete() error {
	return self.region.DeleteSecurityGroup(self.ID)
}
