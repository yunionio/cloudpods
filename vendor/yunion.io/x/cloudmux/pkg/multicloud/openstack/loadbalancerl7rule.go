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
	"context"
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SLoadbalancerL7RuleCreateParams struct {
	CompareType  string   `json:"compare_type"`
	Invert       *bool    `json:"invert"`
	Type         string   `json:"type"`
	Value        string   `json:"value"`
	AdminStateUp bool     `json:"admin_state_up"`
	Tags         []string `json:"tags"`
}

type SLoadbalancerL7Rule struct {
	multicloud.SResourceBase
	multicloud.SLoadbalancerRedirectBase
	OpenStackTags
	policy             *SLoadbalancerL7Policy
	CreatedAt          string   `json:"created_at"`
	CompareType        string   `json:"compare_type"`
	ProvisioningStatus string   `json:"provisioning_status"`
	Invert             bool     `json:"invert"`
	AdminStateUp       bool     `json:"admin_state_up"`
	UpdatedAt          string   `json:"updated_at"`
	Value              string   `json:"value"`
	Key                string   `json:"key"`
	ProjectID          string   `json:"project_id"`
	Type               string   `json:"type"`
	ID                 string   `json:"id"`
	OperatingStatus    string   `json:"operating_status"`
	Tags               []string `json:"tags"`
}

func (region *SRegion) CreateLoadbalancerL7Rule(l7policyId string, rule *cloudprovider.SLoadbalancerListenerRule) (*SLoadbalancerL7Rule, error) {
	type Params struct {
		L7Rule SLoadbalancerL7RuleCreateParams `json:"rule"`
	}
	l7ruleParams := Params{}
	l7ruleParams.L7Rule.AdminStateUp = true
	l7ruleParams.L7Rule.Type = "PATH"
	l7ruleParams.L7Rule.Value = rule.Path
	l7ruleParams.L7Rule.CompareType = "REGEX"
	body, err := region.lbPost(fmt.Sprintf("/v2/lbaas/l7policies/%s/rules", l7policyId), jsonutils.Marshal(l7ruleParams))
	if err != nil {
		return nil, errors.Wrapf(err, `region.lbPost(/v2/lbaas/l7policies/%s/rules), jsonutils.Marshal(l7ruleParams))`, l7policyId)
	}
	l7rule := SLoadbalancerL7Rule{}
	err = body.Unmarshal(&l7rule, "rule")
	if err != nil {
		return nil, errors.Wrap(err, `body.Unmarshal(&l7rule, "rule")`)
	}
	return &l7rule, nil
}

func (policy *SLoadbalancerL7Policy) fetchLoadbalancerL7Rules() error {
	l7rules, err := policy.region.GetLoadbalancerL7Rules(policy.ID)
	if err != nil {
		return errors.Wrapf(err, "policy.region.GetLoadbalancerL7Rules(%s)", policy.ID)
	}
	for i := 0; i < len(l7rules); i++ {
		l7rules[i].policy = policy
	}
	policy.l7rules = l7rules
	return nil
}

func (region *SRegion) GetLoadbalancerL7Rules(policyId string) ([]SLoadbalancerL7Rule, error) {
	l7rules := []SLoadbalancerL7Rule{}
	resource := fmt.Sprintf("/v2/lbaas/l7policies/%s/rules", policyId)
	query := url.Values{}
	for {
		resp, err := region.lbList(resource, query)
		if err != nil {
			return nil, errors.Wrap(err, "lbList")
		}
		part := struct {
			Rules      []SLoadbalancerL7Rule
			RulesLinks SNextLinks
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		l7rules = append(l7rules, part.Rules...)
		marker := part.RulesLinks.GetNextMark()
		if len(marker) == 0 {
			break
		}
		query.Set("marker", marker)
	}

	return l7rules, nil
}

func (region *SRegion) GetLoadbalancerL7RulebyId(policyId string, l7ruleId string) (*SLoadbalancerL7Rule, error) {
	body, err := region.lbGet(fmt.Sprintf("/v2/lbaas/l7policies/%s/rules/%s", policyId, l7ruleId))
	if err != nil {
		return nil, errors.Wrapf(err, `region.lbGet(/v2/lbaas/l7policies/%s/rules/%s )`, policyId, l7ruleId)
	}
	l7rule := SLoadbalancerL7Rule{}
	return &l7rule, body.Unmarshal(&l7rule, "rule")
}

func (l7r *SLoadbalancerL7Rule) GetName() string {
	return l7r.policy.Name + l7r.ID
}

func (l7r *SLoadbalancerL7Rule) GetId() string {
	return l7r.ID
}

func (l7r *SLoadbalancerL7Rule) GetGlobalId() string {
	return l7r.ID
}

func (l7r *SLoadbalancerL7Rule) GetStatus() string {
	switch l7r.ProvisioningStatus {
	case "ACTIVE":
		return api.LB_STATUS_ENABLED
	case "PENDING_CREATE":
		return api.LB_CREATING
	case "PENDING_UPDATE":
		return api.LB_SYNC_CONF
	case "PENDING_DELETE":
		return api.LB_STATUS_DELETING
	case "DELETED":
		return api.LB_STATUS_DELETED
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (self *SLoadbalancerL7Rule) IsDefault() bool {
	return false
}

func (l7r *SLoadbalancerL7Rule) IsEmulated() bool {
	return false
}

func (l7r *SLoadbalancerL7Rule) Refresh() error {
	newL7r, err := l7r.policy.region.GetLoadbalancerL7RulebyId(l7r.policy.ID, l7r.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(l7r, newL7r)
}

func (l7r *SLoadbalancerL7Rule) GetCondition() string {
	return ""
}

func (l7r *SLoadbalancerL7Rule) GetDomain() string {
	return ""
}

func (l7r *SLoadbalancerL7Rule) GetPath() string {
	return l7r.Value
}

func (l7r *SLoadbalancerL7Rule) GetProjectId() string {
	return ""
}

func (l7r *SLoadbalancerL7Rule) GetBackendGroupId() string {
	return l7r.policy.RedirectPoolID
}

func (l7r *SLoadbalancerL7Rule) Delete(ctx context.Context) error {
	return l7r.policy.region.DeleteLoadbalancerListenerL7policy(l7r.policy.ID)
}
