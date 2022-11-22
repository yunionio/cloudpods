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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SLoadbalancerL7PolicyCreateParams struct {
	Description      string   `json:"description,omitempty"`
	AdminStateUp     bool     `json:"admin_state_up,omitempty"`
	ListenerID       string   `json:"listener_id,omitempty"`
	RedirectPrefix   string   `json:"redirect_prefix,omitempty"`
	RedirectURL      string   `json:"redirect_url,omitempty"`
	RedirectPoolID   string   `json:"redirect_pool_id,omitempty"`
	RedirectHTTPCode *int     `json:"redirect_http_code,omitempty"`
	Name             string   `json:"name,omitempty"`
	Action           string   `json:"action,omitempty"`
	Position         *int     `json:"position,omitempty"`
	Tags             []string `json:"tags,omitempty"`
}

type SLoadbalancerL7Policy struct {
	multicloud.SResourceBase
	OpenStackTags
	region             *SRegion
	l7rules            []SLoadbalancerL7Rule
	ListenerID         string      `json:"listener_id"`
	Description        string      `json:"description"`
	AdminStateUp       bool        `json:"admin_state_up"`
	RuleIds            []SL7RuleID `json:"rules"`
	CreatedAt          string      `json:"created_at"`
	ProvisioningStatus string      `json:"provisioning_status"`
	UpdatedAt          string      `json:"updated_at"`
	RedirectHTTPCode   int         `json:"redirect_http_code"`
	RedirectPoolID     string      `json:"redirect_pool_id"`
	RedirectPrefix     string      `json:"redirect_prefix"`
	RedirectURL        string      `json:"redirect_url"`
	Action             string      `json:"action"`
	Position           int         `json:"position"`
	ProjectID          string      `json:"project_id"`
	ID                 string      `json:"id"`
	OperatingStatus    string      `json:"operating_status"`
	Name               string      `json:"name"`
	Tags               []string    `json:"tags"`
}

func (region *SRegion) GetLoadbalancerL7PolicybyId(policieId string) (*SLoadbalancerL7Policy, error) {
	body, err := region.lbGet(fmt.Sprintf("/v2/lbaas/l7policies/%s", policieId))
	if err != nil {
		return nil, errors.Wrapf(err, `region.lbGet(/v2/lbaas/l7policies/%s)`, policieId)
	}
	l7policy := SLoadbalancerL7Policy{}
	err = body.Unmarshal(&l7policy, "l7policy")
	if err != nil {
		return nil, errors.Wrap(err, "body.Unmarshal(&l7policy, policy)")
	}
	l7policy.region = region
	err = l7policy.fetchLoadbalancerL7Rules()
	if err != nil {
		return nil, errors.Wrap(err, "l7policy.fetchLoadbalancerL7Rules()")
	}
	return &l7policy, nil
}

func (region *SRegion) CreateLoadbalancerL7Policy(listenerId string, rule *cloudprovider.SLoadbalancerListenerRule) (*SLoadbalancerL7Policy, error) {
	type Params struct {
		L7policy SLoadbalancerL7PolicyCreateParams `json:"l7policy"`
	}
	l7policyParams := Params{}

	l7policyParams.L7policy.AdminStateUp = true
	l7policyParams.L7policy.ListenerID = listenerId
	l7policyParams.L7policy.Name = rule.Name

	body, err := region.lbPost("/v2/lbaas/l7policies", jsonutils.Marshal(l7policyParams))
	if err != nil {
		return nil, errors.Wrap(err, `region.lbPost("/v2/lbaas/l7policies", jsonutils.Marshal(l7policyParams))`)
	}
	l7policy := SLoadbalancerL7Policy{}
	l7policy.region = region
	err = body.Unmarshal(&l7policy, "l7policy")
	if err != nil {
		return nil, errors.Wrap(err, "body.Unmarshal(&l7policy, policy)")
	}
	return &l7policy, nil
}

func (region *SRegion) DeleteLoadbalancerListenerL7policy(policyId string) error {
	_, err := region.lbDelete(fmt.Sprintf("/v2/lbaas/l7policies/%s", policyId))
	if err != nil {
		return errors.Wrapf(err, `region.lbDelete(/v2/lbaas/l7policies/%s)`, policyId)
	}
	return nil
}

func (L7policy *SLoadbalancerL7Policy) GetName() string {
	return L7policy.Name
}

func (L7policy *SLoadbalancerL7Policy) GetId() string {
	return L7policy.ID
}

func (L7policy *SLoadbalancerL7Policy) GetGlobalId() string {
	return L7policy.ID
}

func (L7policy *SLoadbalancerL7Policy) GetStatus() string {
	switch L7policy.ProvisioningStatus {
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

func (L7policy *SLoadbalancerL7Policy) Refresh() error {
	newL7policie, err := L7policy.region.GetLoadbalancerL7PolicybyId(L7policy.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(L7policy, newL7policie)
}

func (L7policy *SLoadbalancerL7Policy) IsEmulated() bool {
	return false
}
