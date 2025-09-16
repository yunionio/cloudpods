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
	"context"

	"yunion.io/x/log"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SElbListenerPolicy struct {
	multicloud.SResourceBase
	multicloud.SLoadbalancerRedirectBase
	HuaweiTags
	region   *SRegion
	lb       *SLoadbalancer
	listener *SElbListener

	RedirectPoolID     string  `json:"redirect_pool_id"`
	RedirectListenerID *string `json:"redirect_listener_id"`
	Description        string  `json:"description"`
	AdminStateUp       bool    `json:"admin_state_up"`
	Rules              []Rule  `json:"rules"`
	TenantID           string  `json:"tenant_id"`
	ProjectID          string  `json:"project_id"`
	ListenerID         string  `json:"listener_id"`
	RedirectURL        *string `json:"redirect_url"`
	ProvisioningStatus string  `json:"provisioning_status"`
	Action             string  `json:"action"`
	Position           int64   `json:"position"`
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
}

type Rule struct {
	ID string `json:"id"`
}

type SElbListenerPolicyRule struct {
	region *SRegion
	policy *SElbListenerPolicy

	CompareType        string      `json:"compare_type"`
	ProvisioningStatus string      `json:"provisioning_status"`
	AdminStateUp       bool        `json:"admin_state_up"`
	TenantID           string      `json:"tenant_id"`
	ProjectID          string      `json:"project_id"`
	Invert             bool        `json:"invert"`
	Value              string      `json:"value"`
	Key                interface{} `json:"key"`
	Type               string      `json:"type"`
	ID                 string      `json:"id"`
}

func (self *SElbListenerPolicy) GetId() string {
	return self.ID
}

func (self *SElbListenerPolicy) GetName() string {
	return self.Name
}

func (self *SElbListenerPolicy) GetGlobalId() string {
	return self.GetId()
}

// 负载均衡没有启用禁用操作
func (self *SElbListenerPolicy) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SElbListenerPolicy) Refresh() error {
	resp, err := self.lb.region.list(SERVICE_ELB, "elb/l7policies/"+self.ID, nil)
	if err != nil {
		return err
	}
	return resp.Unmarshal(self, "l7policy")
}

func (self *SElbListenerPolicy) IsDefault() bool {
	return false
}

func (self *SElbListenerPolicy) IsEmulated() bool {
	return false
}

func (self *SElbListenerPolicy) GetProjectId() string {
	return ""
}

func (self *SElbListenerPolicy) GetRules() ([]SElbListenerPolicyRule, error) {
	ret, err := self.region.GetLoadBalancerPolicyRules(self.GetId())
	if err != nil {
		return nil, err
	}

	for i := range ret {
		ret[i].region = self.lb.region
		ret[i].policy = self
	}

	return ret, nil
}

func (self *SElbListenerPolicy) GetDomain() string {
	rules, err := self.GetRules()
	if err != nil {
		log.Errorf("loadbalancer rule GetDomain %s", err)
	}

	for i := range rules {
		if rules[i].Type == "HOST_NAME" {
			return rules[i].Value
		}
	}

	return ""
}

func (self *SElbListenerPolicy) GetCondition() string {
	return ""
}

func (self *SElbListenerPolicy) GetPath() string {
	rules, err := self.GetRules()
	if err != nil {
		log.Errorf("loadbalancer rule GetPath %s", err)
	}

	for i := range rules {
		if rules[i].Type == "PATH" {
			return rules[i].Value
		}
	}

	return ""
}

func (self *SElbListenerPolicy) GetBackendGroupId() string {
	return self.RedirectPoolID
}

func (self *SElbListenerPolicy) Delete(ctx context.Context) error {
	return self.region.DeleteLoadBalancerPolicy(self.GetId())
}

func (self *SRegion) DeleteLoadBalancerPolicy(policyId string) error {
	_, err := self.delete(SERVICE_ELB, "elb/l7policies/"+policyId)
	return err
}
