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

package rbacutils

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
)

type TPolicySet []TPolicy

func (policies TPolicySet) GetMatchRules(service string, resource string, action string, extra ...string) []SRbacRule {
	matchRules := make([]SRbacRule, 0)
	for i := range policies {
		rule := policies[i].GetMatchRule(service, resource, action, extra...)
		if rule != nil {
			matchRules = append(matchRules, *rule)
		}
	}
	return matchRules
}

func DecodePolicySet(jsonObj jsonutils.JSONObject) (TPolicySet, error) {
	jsonArr, err := jsonObj.GetArray()
	if err != nil {
		return nil, errors.Wrap(httperrors.ErrInvalidFormat, "invalid json: not an array")
	}
	set := TPolicySet{}
	for i := range jsonArr {
		policy, err := DecodePolicy(jsonArr[i])
		if err != nil {
			return nil, errors.Wrapf(err, "decode %d", i)
		}
		set = append(set, policy)
	}
	return set, nil
}

func (policies TPolicySet) Encode() jsonutils.JSONObject {
	obj := make([]jsonutils.JSONObject, len(policies))
	for i := range policies {
		obj[i] = policies[i].Encode()
	}
	return jsonutils.NewArray(obj...)
}

// ViolatedBy: policies中deny的权限，但是assign中却是allow
// if any assign allow, but policies deny
// OR
// assign allow, if any policies deny
func (policies TPolicySet) ViolatedBy(assign TPolicySet) bool {
	if policies.violatedBySet(assign, Allow) {
		return true
	}
	if assign.violatedBySet(policies, Deny) {
		return true
	}
	return false
}

func (policies TPolicySet) violatedBySet(assign TPolicySet, expect TRbacResult) bool {
	for i := range assign {
		if policies.violatedByPolicy(assign[i], expect) {
			return true
		}
	}
	return false
}

func (policies TPolicySet) violatedByPolicy(policy TPolicy, expect TRbacResult) bool {
	for i := range policy {
		rule := policy[i]
		if rule.Result != expect {
			continue
		}
		matchRules := policies.GetMatchRules(rule.Service, rule.Resource, rule.Action, rule.Extra...)
		matchRule := GetMatchRule(matchRules, rule.Service, rule.Resource, rule.Action, rule.Extra...)
		if expect == Allow && (matchRule == nil || matchRule.Result == Deny) {
			return true
		} else if expect == Deny && matchRule != nil && matchRule.Result == Allow {
			return true
		}
	}
	return false
}
