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
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"
)

type TRbacResult string

const (
	WILD_MATCH = "*"

	Allow = TRbacResult("allow")
	Deny  = TRbacResult("deny")

	// no more many allow levels, only allow/deny

	AdminAllow = TRbacResult("admin") // deprecated
	OwnerAllow = TRbacResult("owner") // deprecated
	UserAllow  = TRbacResult("user")  // deprecated
	GuestAllow = TRbacResult("guest") // deprecated

	/*ScopeSystem  = rbacscope.ScopeSystem
	ScopeDomain  = rbacscope.ScopeDomain
	ScopeProject = rbacscope.ScopeProject
	// ScopeObject  = "object"
	ScopeUser = rbacscope.ScopeUser
	ScopeNone = rbacscope.ScopeNone*/
)

func (r TRbacResult) IsAllow() bool {
	return r == Allow
}

func (r TRbacResult) IsDeny() bool {
	return r == Deny
}

var (
	strictness = map[TRbacResult]int{
		Deny:       0,
		AdminAllow: 1,
		OwnerAllow: 2,
		UserAllow:  3,
		GuestAllow: 4,
		Allow:      5,
	}
)

func (r TRbacResult) Strictness() int {
	return strictness[r]
}

func (r1 TRbacResult) StricterThan(r2 TRbacResult) bool {
	return r1.Strictness() < r2.Strictness()
}

func (r1 TRbacResult) LooserThan(r2 TRbacResult) bool {
	return r1.Strictness() > r2.Strictness()
}

type SRbacRule struct {
	Service  string
	Resource string
	Action   string
	Extra    []string
	Result   TRbacResult
}

func (r SRbacRule) clone() SRbacRule {
	nr := r
	nr.Extra = make([]string, len(r.Extra))
	if len(r.Extra) > 0 {
		copy(nr.Extra, r.Extra)
	}
	return nr
}

func isWildMatch(str string) bool {
	return len(str) == 0 || str == WILD_MATCH
}

func (rule *SRbacRule) contains(rule2 *SRbacRule) bool {
	if !isWildMatch(rule.Service) && rule.Service != rule2.Service {
		return false
	}
	if !isWildMatch(rule.Resource) && rule.Resource != rule2.Resource {
		return false
	}
	if !isWildMatch(rule.Action) && rule.Action != rule2.Action {
		return false
	}
	if len(rule.Extra) > 0 {
		for i := 0; i < len(rule.Extra); i += 1 {
			if !isWildMatch(rule.Extra[i]) && (rule2.Extra == nil || len(rule2.Extra) < i || rule.Extra[i] != rule2.Extra[i]) {
				return false
			}
		}
	}
	if string(rule.Result) != string(rule2.Result) {
		return false
	}
	return true
}

func (rule *SRbacRule) stricterThan(r2 *SRbacRule) bool {
	return rule.Result.StricterThan(r2.Result)
}

func (rule *SRbacRule) looserThan(r2 *SRbacRule) bool {
	return rule.Result.LooserThan(r2.Result)
}

func (rule *SRbacRule) match(service string, resource string, action string, extra ...string) (bool, int, int) {
	matched := 0
	weight := 0
	if !isWildMatch(rule.Service) {
		if rule.Service != service {
			return false, 0, 0
		}
		matched += 1
		weight += 1
	}
	if !isWildMatch(rule.Resource) {
		if rule.Resource != resource {
			return false, 0, 0
		}
		matched += 1
		weight += 10
	}
	if !isWildMatch(rule.Action) {
		if rule.Action != action {
			return false, 0, 0
		}
		matched += 1
		weight += 100
	}
	for i := 0; i < len(rule.Extra) && i < len(extra); i += 1 {
		if !isWildMatch(rule.Extra[i]) {
			if rule.Extra[i] != extra[i] {
				return false, 0, 0
			}
			matched += 1
			weight += 1000 * (i + 1)
		}
	}
	return true, matched, weight
}

var (
	ShowMatchRuleDebug = false
)

func GetMatchRule(rules []SRbacRule, service string, resource string, action string, extra ...string) *SRbacRule {
	maxMatchCnt := 0
	minWeight := 1000000
	var matchRule *SRbacRule
	for i := 0; i < len(rules); i += 1 {
		match, matchCnt, weight := rules[i].match(service, resource, action, extra...)
		if match && ShowMatchRuleDebug {
			log.Debugf("rule %s match cnt %d weight %d", rules[i], matchCnt, weight)
		}
		if match && (maxMatchCnt < matchCnt ||
			(maxMatchCnt == matchCnt && minWeight > weight) ||
			(maxMatchCnt == matchCnt && minWeight == weight && matchRule.stricterThan(&rules[i]))) {
			maxMatchCnt = matchCnt
			minWeight = weight
			matchRule = &rules[i]
		}
	}
	return matchRule
}

const (
	levelService  = 0
	levelResource = 1
	levelAction   = 2
	levelExtra    = 3
)

func (rule *SRbacRule) toStringArray() []string {
	strArr := make([]string, 0)
	strArr = append(strArr, rule.Service)
	strArr = append(strArr, rule.Resource)
	strArr = append(strArr, rule.Action)
	if rule.Extra != nil {
		strArr = append(strArr, rule.Extra...)
	}
	i := len(strArr) - 1
	for i > 0 && (len(strArr[i]) == 0 || strArr[i] == WILD_MATCH) {
		i -= 1
	}
	return strArr[0 : i+1]
}

func contains(s1 []string, s string) bool {
	for i := range s1 {
		if s1[i] == s {
			return true
		}
	}
	return false
}

func intersect(s1 []string, s2 []string) bool {
	for i := range s1 {
		for j := range s2 {
			if s1[i] == s2[j] {
				return true
			}
		}
	}
	return false
}

func containsIp(ips []netutils.IPV4Prefix, ipStr string) bool {
	if len(ipStr) == 0 {
		// user comes from unknown ip, assume matches
		return true
	}
	ip, err := netutils.NewIPV4Addr(ipStr)
	if err != nil {
		log.Errorf("user comes from invalid ipv4 addr %s: %s", ipStr, err)
		return false
	}
	for i := range ips {
		if ips[i].Contains(ip) {
			return true
		}
	}
	return false
}

const (
	FAKE_TOKEN = "fake_token"
)

type IBaseIdentity interface {
	GetLoginIp() string
	GetTokenString() string
}
type IRbacIdentity interface {
	GetProjectId() string
	GetRoleIds() []string

	IBaseIdentity
}

type IRbacIdentity2 interface {
	GetProjectDomainId() string
	GetProjectName() string
	GetRoles() []string

	IBaseIdentity
}

type sSimpleRbacIdentity struct {
	projectDomainId string
	projectId       string
	projectName     string
	roleIds         []string
	roles           []string
	ip              string
}

func (id sSimpleRbacIdentity) GetRoles() []string {
	return id.roles
}

func (id sSimpleRbacIdentity) GetRoleIds() []string {
	return id.roleIds
}

func (id sSimpleRbacIdentity) GetProjectName() string {
	return id.projectName
}

func (id sSimpleRbacIdentity) GetProjectId() string {
	return id.projectId
}

func (id sSimpleRbacIdentity) GetProjectDomainId() string {
	return id.projectDomainId
}

func (id sSimpleRbacIdentity) GetLoginIp() string {
	return id.ip
}

func (id sSimpleRbacIdentity) GetTokenString() string {
	return FAKE_TOKEN
}

func newRbacIdentity2(projectDomainId, projectName string, roles []string, ip string) IRbacIdentity2 {
	return sSimpleRbacIdentity{
		projectDomainId: projectDomainId,
		projectName:     projectName,
		roles:           roles,
		ip:              ip,
	}
}

func NewRbacIdentity(projectId string, roleIds []string, ip string) IRbacIdentity {
	return sSimpleRbacIdentity{
		projectId: projectId,
		roleIds:   roleIds,
		ip:        ip,
	}
}
