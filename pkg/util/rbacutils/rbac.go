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
	"strings"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"
)

type TRbacResult string
type TRbacScope string

const (
	WILD_MATCH = "*"

	Allow = TRbacResult("allow")
	Deny  = TRbacResult("deny")

	// no more many allow levels, only allow/deny

	AdminAllow = TRbacResult("admin") // deprecated
	OwnerAllow = TRbacResult("owner") // deprecated
	UserAllow  = TRbacResult("user")  // deprecated
	GuestAllow = TRbacResult("guest") // deprecated

	ScopeSystem  = TRbacScope("system")
	ScopeDomain  = TRbacScope("domain")
	ScopeProject = TRbacScope("project")
	// ScopeObject  = "object"
	ScopeUser = TRbacScope("user")
	ScopeNone = TRbacScope("none")
)

var (
	strictness = map[TRbacResult]int{
		Deny:       0,
		AdminAllow: 1,
		OwnerAllow: 2,
		UserAllow:  3,
		GuestAllow: 4,
		Allow:      5,
	}

	scopeScore = map[TRbacScope]int{
		ScopeNone:    0,
		ScopeUser:    1,
		ScopeProject: 2,
		ScopeDomain:  3,
		ScopeSystem:  4,
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

func (s1 TRbacScope) HigherEqual(s2 TRbacScope) bool {
	return scopeScore[s1] >= scopeScore[s2]
}

func (s1 TRbacScope) HigherThan(s2 TRbacScope) bool {
	return scopeScore[s1] > scopeScore[s2]
}

type SRbacPolicy struct {
	// condition, when the policy takes effects
	Condition string // deprecated

	DomainId string

	IsPublic bool

	Projects []string
	Roles    []string
	Ips      []netutils.IPV4Prefix
	Auth     bool // whether needs authentication

	// scope, the scope of the policy, system/domain/project
	Scope   TRbacScope
	IsAdmin bool // deprecated, is_admin=true means system scope, is_admin=false means project scope

	// rules, the exact rules
	Rules []SRbacRule
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

func (policy *SRbacPolicy) getMatchRule(req []string) *SRbacRule {
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

func (policy *SRbacPolicy) GetMatchRule(service string, resource string, action string, extra ...string) *SRbacRule {
	return GetMatchRule(policy.Rules, service, resource, action, extra...)
}

func GetMatchRule(rules []SRbacRule, service string, resource string, action string, extra ...string) *SRbacRule {
	maxMatchCnt := 0
	minWeight := 1000000
	var matchRule *SRbacRule
	for i := 0; i < len(rules); i += 1 {
		match, matchCnt, weight := rules[i].match(service, resource, action, extra...)
		if match && (maxMatchCnt < matchCnt ||
			(maxMatchCnt == matchCnt && minWeight > weight) ||
			(maxMatchCnt == matchCnt && minWeight == weight && matchRule.looserThan(&rules[i]))) {
			maxMatchCnt = matchCnt
			minWeight = weight
			matchRule = &rules[i]
		}
	}
	return matchRule
}

func CompactRules(rules []SRbacRule) []SRbacRule {
	if len(rules) == 0 {
		return nil
	}
	/*output := make([]SRbacRule, 1)
	output[0] = rules[0]
	for i := 1; i < len(rules); i += 1 {
		isContains := false
		for j := 0; j < len(output); j += 1 {
			if output[j].contains(&rules[i]) {
				isContains = true
				break
			}
			if rules[i].contains(&output[j]) {
				output[j] = rules[i]
				isContains = true
				break
			}
		}
		if !isContains {
			output = append(output, rules[i])
		}
	}*/
	return reduceRules(rules)
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

	ruleJson, err := policyJson.Get("policy")
	if err != nil {
		return err
	}

	/*rules, err := decode(ruleJson, SRbacRule{}, levelService)*/
	rules, err := json2Rules(ruleJson)
	if err != nil {
		return errors.Wrap(err, "json2Rules")
	}

	if len(rules) == 0 {
		return ErrEmptyPolicy
	}

	policy.Rules = rules

	return nil
}

const (
	levelService  = 0
	levelResource = 1
	levelAction   = 2
	levelExtra    = 3
)

func decode(rules jsonutils.JSONObject, decodeRule SRbacRule, level int) ([]SRbacRule, error) {
	switch rules.(type) {
	case *jsonutils.JSONString:
		ruleJsonStr := rules.(*jsonutils.JSONString)
		ruleStr, _ := ruleJsonStr.GetString()
		switch ruleStr {
		case string(Allow), string(AdminAllow), string(OwnerAllow), string(UserAllow), string(GuestAllow):
			decodeRule.Result = Allow
		default:
			decodeRule.Result = Deny
			// default:
			//	return nil, fmt.Errorf("unsupported rule string %s", ruleStr)
		}
		return []SRbacRule{decodeRule}, nil
	case *jsonutils.JSONDict:
		ruleJsonDict, err := rules.GetMap()
		if err != nil {
			return nil, errors.Wrap(err, "get rule map fail")
		}
		rules := make([]SRbacRule, 0)
		for key, ruleJson := range ruleJsonDict {
			rule := decodeRule
			switch {
			case level == levelService:
				rule.Service = key
			case level == levelResource:
				rule.Resource = key
			case level == levelAction:
				rule.Action = key
			case level >= levelExtra:
				if rule.Extra == nil {
					rule.Extra = make([]string, 1)
					rule.Extra[0] = key
				} else {
					rule.Extra = append(rule.Extra, key)
				}
			}
			decoded, err := decode(ruleJson, rule, level+1)
			if err != nil {
				return nil, errors.Wrap(err, "decode")
			}
			rules = append(rules, decoded...)
		}
		return rules, nil
	default:
		return nil, errors.WithMessage(ErrUnsuportRuleData, rules.String())
	}
}

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

func addRule2Json(nodeJson *jsonutils.JSONDict, keys []string, result TRbacResult) error {
	if len(keys) == 1 {
		if nodeJson.Contains(keys[0]) {
			nextJson, _ := nodeJson.Get(keys[0])
			switch nextJson.(type) {
			case *jsonutils.JSONString: // conflict??
				return ErrConflict // fmt.Errorf("conflict?")
			case *jsonutils.JSONDict:
				nextJsonDict := nextJson.(*jsonutils.JSONDict)
				addRule2Json(nextJsonDict, []string{WILD_MATCH}, result)
				return nil
			default:
				return ErrInvalidRules // fmt.Errorf("invalid rules")
			}
		} else {
			nodeJson.Add(jsonutils.NewString(string(result)), keys[0])
			return nil
		}
	}
	// len(keys) > 1
	exist, _ := nodeJson.Get(keys[0])
	if exist != nil {
		switch exist.(type) {
		case *jsonutils.JSONString: // need restruct
			newDict := jsonutils.NewDict()
			newDict.Add(exist, "*")
			nodeJson.Set(keys[0], newDict)
			return addRule2Json(newDict, keys[1:], result)
		case *jsonutils.JSONDict:
			existDict := exist.(*jsonutils.JSONDict)
			return addRule2Json(existDict, keys[1:], result)
		default:
			return ErrInvalidRules // fmt.Errorf("invalid rules")
		}
	} else {
		next := jsonutils.NewDict()
		nodeJson.Add(next, keys[0])
		return addRule2Json(next, keys[1:], result)
	}
}

func (policy *SRbacPolicy) Encode() (jsonutils.JSONObject, error) {
	/*rules := jsonutils.NewDict()
	for i := 0; i < len(policy.Rules); i += 1 {
		keys := policy.Rules[i].toStringArray()
		err := addRule2Json(rules, keys, policy.Rules[i].Result)
		if err != nil {
			return nil, errors.Wrap(err, "addRule2Json")
		}
	}*/

	rules := rules2Json(policy.Rules)

	ret := jsonutils.NewDict()
	// ret.Add(jsonutils.NewString(policy.Condition), "condition")
	// if policy.IsAdmin {
	// 	ret.Add(jsonutils.JSONTrue, "is_admin")
	// } else {
	// 	ret.Add(jsonutils.JSONFalse, "is_admin")
	// }

	ret.Add(jsonutils.NewString(string(policy.Scope)), "scope")

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

	ret.Add(rules, "policy")
	return ret, nil
}

func (policy *SRbacPolicy) Explain(request [][]string) [][]string {
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

type IRbacIdentity interface {
	GetProjectDomainId() string
	GetProjectName() string
	GetRoles() []string
	GetLoginIp() string
}

func (policy *SRbacPolicy) IsSystemWidePolicy() bool {
	return len(policy.Roles) == 0 && len(policy.Projects) == 0
}

// check whether policy maches a userCred
// return value
// bool isMatched
// int  match weight, the higher the value, the more exact the match
// the more exact match wins
func (policy *SRbacPolicy) Match(userCred IRbacIdentity) (bool, int) {
	if !policy.Auth && len(policy.Roles) == 0 && len(policy.Projects) == 0 && len(policy.Ips) == 0 {
		return true, 1
	}
	if userCred == nil {
		return false, 0
	}
	weight := 0
	if policy.IsPublic || len(policy.DomainId) == 0 || policy.DomainId == userCred.GetProjectDomainId() {
		if len(policy.DomainId) > 0 {
			weight += 10
		}
		if !policy.IsPublic {
			weight += 10
		}
		if len(policy.Roles) == 0 || intersect(policy.Roles, userCred.GetRoles()) {
			if len(policy.Roles) != 0 {
				weight += 100
			}
			if len(policy.Projects) == 0 || contains(policy.Projects, userCred.GetProjectName()) {
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

func (policy *SRbacPolicy) MatchRole(roleName string) bool {
	if len(policy.Roles) == 0 || contains(policy.Roles, roleName) {
		return true
	}
	return false
}

func String2Scope(str string) TRbacScope {
	return String2ScopeDefault(str, ScopeProject)
}

func String2ScopeDefault(str string, defScope TRbacScope) TRbacScope {
	switch strings.ToLower(str) {
	case string(ScopeSystem):
		return ScopeSystem
	case string(ScopeDomain):
		return ScopeDomain
	case string(ScopeProject):
		return ScopeProject
	case string(ScopeUser):
		return ScopeUser
	case "true":
		return ScopeSystem
	default:
		return defScope
	}
}
