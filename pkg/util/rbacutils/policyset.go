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

type SPolicyInfo struct {
	Id     string
	Name   string
	Policy *SRbacPolicy
}

type TPolicySet []*SRbacPolicy

func GetMatchedPolicies(policies []SPolicyInfo, userCred IRbacIdentity) (TPolicySet, []string) {
	matchedPolicies := make([]*SRbacPolicy, 0)
	matchedNames := make([]string, 0)
	maxMatchWeight := 0
	for i := range policies {
		isMatched, matchWeight := policies[i].Policy.Match(userCred)
		if !isMatched || matchWeight < maxMatchWeight {
			continue
		}
		if maxMatchWeight <= matchWeight {
			if maxMatchWeight < matchWeight {
				maxMatchWeight = matchWeight
				matchedPolicies = matchedPolicies[:0]
				matchedNames = matchedNames[:0]
			}
			matchedPolicies = append(matchedPolicies, policies[i].Policy)
			matchedNames = append(matchedNames, policies[i].Name)
		}
	}
	return matchedPolicies, matchedNames
}

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

func (policies TPolicySet) violatedByPolicy(policy *SRbacPolicy, expect TRbacResult) bool {
	for i := range policy.Rules {
		rule := policy.Rules[i]
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
