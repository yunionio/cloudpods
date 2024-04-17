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

package oracle

import (
	"net/url"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
)

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	SOracleTag

	region *SRegion

	Id                   string
	IngressSecurityRules []SSecurityGroupRule `json:"ingress-security-rules"`
	EgressSecurityRules  []SSecurityGroupRule `json:"egress-security-rules"`
	LifecycleState       string               `json:"lifecycle-state"`
	VcnId                string               `json:"vcn-id"`
	DisplayName          string               `json:"display-name"`
}

func (self *SSecurityGroup) GetVpcId() string {
	return self.VcnId
}

func (self *SSecurityGroup) GetId() string {
	return self.Id
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.Id
}

func (self *SSecurityGroup) GetDescription() string {
	return ""
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SSecurityGroup) GetName() string {
	return self.DisplayName
}

func (self *SSecurityGroup) GetStatus() string {
	return api.SECGROUP_STATUS_READY
}

func (self *SSecurityGroup) Refresh() error {
	group, err := self.region.GetSecurityGroup(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, group)
}

func (self *SSecurityGroup) GetReferences() ([]cloudprovider.SecurityGroupReference, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SSecurityGroup) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SSecurityGroup) CreateRule(opts *cloudprovider.SecurityGroupRuleCreateOptions) (cloudprovider.ISecurityGroupRule, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SSecurityGroup) GetProjectId() string {
	return ""
}

func (self *SRegion) GetSecurityGroups(vpcId string) ([]SSecurityGroup, error) {
	params := url.Values{}
	if len(vpcId) > 0 {
		params.Set("vcnId", vpcId)
	}
	resp, err := self.list(SERVICE_IAAS, "securityLists", params)
	if err != nil {
		return nil, err
	}
	ret := []SSecurityGroup{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) GetSecurityGroup(id string) (*SSecurityGroup, error) {
	resp, err := self.get(SERVICE_IAAS, "securityLists", id, nil)
	if err != nil {
		return nil, err
	}
	ret := &SSecurityGroup{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
