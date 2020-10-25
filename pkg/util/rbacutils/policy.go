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
	"regexp"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
)

type SRbacPolicy struct {
	// condition, when the policy takes effects
	// Deprecated
	Condition string

	DomainId string

	IsPublic bool

	PublicScope TRbacScope

	SharedDomainIds []string

	Projects []string

	Roles []string

	Ips []netutils.IPV4Prefix

	Auth bool // whether needs authentication

	// scope, the scope of the policy, system/domain/project
	Scope TRbacScope
	// Deprecated
	// is_admin=true means scope=system, is_admin=false means scope=project
	IsAdmin bool

	Rules TPolicy
}

var (
	tenantEqualsPattern = regexp.MustCompile(`tenant\s*==\s*['"]?(\w+)['"]?`)
	roleContainsPattern = regexp.MustCompile(`roles.contains\(['"]?(\w+)['"]?\)`)
)

func searchMatchStrings(pattern *regexp.Regexp, condstr string) []string {
	ret := make([]string, 0)
	matches := pattern.FindAllStringSubmatch(condstr, -1)
	for _, match := range matches {
		ret = append(ret, match[1])
	}
	return ret
}

func searchMatchTenants(condstr string) []string {
	return searchMatchStrings(tenantEqualsPattern, condstr)
}

func searchMatchRoles(condstr string) []string {
	return searchMatchStrings(roleContainsPattern, condstr)
}

func (policy *SRbacPolicy) Decode(policyJson jsonutils.JSONObject) error {
	policy.Condition, _ = policyJson.GetString("condition")
	if policyJson.Contains("projects") {
		projectJson, _ := policyJson.GetArray("projects")
		policy.Projects = jsonutils.JSONArray2StringArray(projectJson)
	}
	if policyJson.Contains("roles") {
		roleJson, _ := policyJson.GetArray("roles")
		policy.Roles = jsonutils.JSONArray2StringArray(roleJson)
	}

	if len(policy.Projects) == 0 && len(policy.Roles) == 0 && len(policy.Condition) > 0 {
		// XXX hack
		// for smooth transtion from condition to projects&roles
		policy.Projects = searchMatchTenants(policy.Condition)
		policy.Roles = searchMatchRoles(policy.Condition)
	}
	// empty condition, no longer use this field
	policy.Condition = ""

	if policyJson.Contains("ips") {
		ipsJson, _ := policyJson.GetArray("ips")
		ipStrs := jsonutils.JSONArray2StringArray(ipsJson)
		policy.Ips = make([]netutils.IPV4Prefix, 0)
		for _, ipStr := range ipStrs {
			if len(ipStr) == 0 || ipStr == "0.0.0.0" {
				continue
			}
			prefix, err := netutils.NewIPV4Prefix(ipStr)
			if err != nil {
				continue
			}
			policy.Ips = append(policy.Ips, prefix)
		}
	}

	policy.Auth = jsonutils.QueryBoolean(policyJson, "auth", true)
	if len(policy.Ips) > 0 || len(policy.Roles) > 0 || len(policy.Projects) > 0 {
		policy.Auth = true
	}

	scopeStr, _ := policyJson.GetString("scope")
	if len(scopeStr) > 0 {
		policy.Scope = TRbacScope(scopeStr)
	} else {
		policy.IsAdmin = jsonutils.QueryBoolean(policyJson, "is_admin", false)
		if len(policy.Scope) == 0 {
			if policy.IsAdmin {
				policy.Scope = ScopeSystem
			} else {
				policy.Scope = ScopeProject
			}
		}
	}

	policyBody, err := policyJson.Get("policy")
	if err != nil {
		return errors.Wrap(err, "Get policy")
	}
	policy.Rules, err = DecodePolicy(policyBody)
	if err != nil {
		return errors.Wrap(err, "DecodePolicy")
	}

	return nil
}

func (policy *SRbacPolicy) Encode() jsonutils.JSONObject {
	ret := jsonutils.NewDict()

	if !policy.Auth && len(policy.Projects) == 0 && len(policy.Roles) == 0 && len(policy.Ips) == 0 {
		ret.Add(jsonutils.JSONFalse, "auth")
	} else {
		ret.Add(jsonutils.JSONTrue, "auth")
	}

	if len(policy.Projects) > 0 {
		ret.Add(jsonutils.NewStringArray(policy.Projects), "projects")
	}
	if len(policy.Roles) > 0 {
		ret.Add(jsonutils.NewStringArray(policy.Roles), "roles")
	}
	if len(policy.Ips) > 0 {
		ipStrs := make([]string, len(policy.Ips))
		for i := range policy.Ips {
			ipStrs[i] = policy.Ips[i].String()
		}
		ret.Add(jsonutils.NewStringArray(ipStrs), "ips")
	}

	ret.Add(jsonutils.NewString(string(policy.Scope)), "scope")

	ret.Add(policy.Rules.Encode(), "policy")

	return ret
}

func (policy *SRbacPolicy) IsSystemWidePolicy() bool {
	return (len(policy.DomainId) == 0 || (policy.IsPublic && policy.PublicScope == ScopeSystem)) && len(policy.Roles) == 0 && len(policy.Projects) == 0
}

func (policy *SRbacPolicy) MatchDomain(domainId string) bool {
	if len(policy.DomainId) == 0 || len(domainId) == 0 {
		return true
	}
	if policy.DomainId == domainId {
		return true
	}
	if policy.IsPublic {
		if policy.PublicScope == ScopeSystem {
			return true
		}
		if contains(policy.SharedDomainIds, domainId) {
			return true
		}
	}
	return false
}

func (policy *SRbacPolicy) MatchProject(projectName string) bool {
	if len(policy.Projects) == 0 || len(projectName) == 0 {
		return true
	}
	if contains(policy.Projects, projectName) {
		return true
	}
	return false
}

func (policy *SRbacPolicy) MatchRoles(roleNames []string) bool {
	if len(policy.Roles) == 0 {
		return true
	}
	if intersect(policy.Roles, roleNames) {
		return true
	}
	return false
}

// check whether policy maches a userCred
// return value
// bool isMatched
// int  match weight, the higher the value, the more exact the match
// the more exact match wins
func (policy *SRbacPolicy) Match(userCred IRbacIdentity2) (bool, int) {
	if !policy.Auth && len(policy.Roles) == 0 && len(policy.Projects) == 0 && len(policy.Ips) == 0 {
		return true, 1
	}
	if userCred == nil || len(userCred.GetTokenString()) == 0 {
		return false, 0
	}
	weight := 0
	if policy.MatchDomain(userCred.GetProjectDomainId()) {
		if len(policy.DomainId) > 0 {
			if policy.DomainId == userCred.GetProjectDomainId() {
				weight += 30 // exact domain match
			} else if len(policy.SharedDomainIds) > 0 {
				weight += 20 // shared domain match
			} else {
				weight += 10 // else, system scope match
			}
		}
		if policy.MatchRoles(userCred.GetRoles()) {
			if len(policy.Roles) != 0 {
				weight += 100
			}
			if policy.MatchProject(userCred.GetProjectName()) {
				if len(policy.Projects) > 0 {
					weight += 1000
				}
				if len(policy.Ips) == 0 || containsIp(policy.Ips, userCred.GetLoginIp()) {
					if len(policy.Ips) > 0 {
						weight += 10000
					}
					return true, weight
				}
			}
		}
	}
	return false, 0
}
