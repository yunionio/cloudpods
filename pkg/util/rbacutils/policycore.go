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

type TPolicy []SRbacRule

func (policy TPolicy) getMatchRule(req []string) *SRbacRule {
	service := WILD_MATCH
	if len(req) > levelService {
		service = req[levelService]
	}
	resource := WILD_MATCH
	if len(req) > levelResource {
		resource = req[levelResource]
	}
	action := WILD_MATCH
	if len(req) > levelAction {
		action = req[levelAction]
	}
	var extra []string
	if len(req) > levelExtra {
		extra = req[levelExtra:]
	} else {
		extra = make([]string, 0)
	}

	return policy.GetMatchRule(service, resource, action, extra...)
}

func (policy TPolicy) GetMatchRule(service string, resource string, action string, extra ...string) *SRbacRule {
	return GetMatchRule(policy, service, resource, action, extra...)
}

func decodePolicy(policyJson jsonutils.JSONObject) (TPolicy, error) {
	rules, err := json2Rules(policyJson)
	if err != nil {
		return nil, errors.Wrap(err, "json2Rules")
	}
	if len(rules) == 0 {
		return nil, ErrEmptyPolicy
	}
	return rules, nil
}

func DecodeRawPolicyData(input jsonutils.JSONObject) (TPolicy, error) {
	policyData, err := input.Get("policy")
	if err != nil || policyData == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidFormat, "invalid policy data")
	}
	return decodePolicy(policyData)
}

func (policy TPolicy) encode() jsonutils.JSONObject {
	return rules2Json(policy)
}

func (policy TPolicy) EncodeRawData() jsonutils.JSONObject {
	ret := jsonutils.NewDict()
	ret.Add(policy.encode(), "policy")
	return ret
}

func (policy TPolicy) Explain(request [][]string) [][]string {
	output := make([][]string, len(request))
	for i := 0; i < len(request); i += 1 {
		rule := policy.getMatchRule(request[i])
		if rule == nil {
			output[i] = append(request[i], string(Deny))
		} else {
			output[i] = append(request[i], string(rule.Result))
		}
	}
	return output
}

// Contains of TPolicy
//
//	TPolicy p1 contains p2 means any action allowed by p2 is also allowed by p1
//	and any action denied by p1 is also denied by p2
func (policy1 TPolicy) Contains(policy2 TPolicy) bool {
	for _, rule := range policy2 {
		if rule.Result == Allow {
			rule := policy1.GetMatchRule(rule.Service, rule.Resource, rule.Action, rule.Extra...)
			if rule == nil {
				return false
			}
			if rule.Result == Deny {
				return false
			}
		}
	}
	for _, rule := range policy1 {
		if rule.Result == Deny {
			rule := policy2.GetMatchRule(rule.Service, rule.Resource, rule.Action, rule.Extra...)
			if rule != nil && rule.Result == Allow {
				return false
			}
		}
	}
	return true
}
