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

package ecloud

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

// SSecurityGroup 与 ecloudsdkvpc ListSecGroupResponseContent 字段对应
type SSecurityGroup struct {
	multicloud.SSecurityGroup
	EcloudTags

	region *SRegion

	Id          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Region      string  `json:"region"`
	Type        string  `json:"type"`
	CreatedTime string  `json:"createdTime"`
	Defaulted   *bool   `json:"defaulted,omitempty"`
	Stateful    *bool   `json:"stateful,omitempty"`
	Scale       string  `json:"scale,omitempty"`
	VpoolId     *string `json:"vpoolId,omitempty"`
	Vaz         *string `json:"vaz,omitempty"`
}

func (sg *SSecurityGroup) GetId() string {
	return sg.Id
}

func (sg *SSecurityGroup) GetName() string {
	return sg.Name
}

func (sg *SSecurityGroup) GetGlobalId() string {
	return sg.Id
}

func (sg *SSecurityGroup) GetDescription() string {
	return sg.Description
}

func (sg *SSecurityGroup) GetStatus() string {
	return api.SECGROUP_STATUS_READY
}

func (sg *SSecurityGroup) GetVpcId() string {
	// 移动云安全组为 region 维度，非 VPC 维度
	return ""
}

func (sg *SSecurityGroup) Refresh() error {
	latest, err := sg.region.GetSecurityGroup(sg.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(sg, latest)
}

func (sg *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	rules, err := sg.region.GetSecurityGroupRules(sg.Id)
	if err != nil {
		return nil, err
	}
	ret := make([]cloudprovider.ISecurityGroupRule, len(rules))
	for i := range rules {
		ret[i] = &rules[i]
	}
	return ret, nil
}

func (sg *SSecurityGroup) CreateRule(opts *cloudprovider.SecurityGroupRuleCreateOptions) (cloudprovider.ISecurityGroupRule, error) {
	rule, err := sg.region.CreateSecurityGroupRule(sg.Id, opts)
	if err != nil {
		return nil, err
	}
	return rule, nil
}

func (sg *SSecurityGroup) GetReferences() ([]cloudprovider.SecurityGroupReference, error) {
	return nil, nil
}

func (sg *SSecurityGroup) Delete() error {
	return sg.region.DeleteSecurityGroup(sg.Id)
}
