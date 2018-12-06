package rbacutils

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/conditionparser"
)

type TRbacResult string

const (
	WILD_MATCH = "*"

	Allow = TRbacResult("allow")

	AdminAllow = TRbacResult("admin")
	OwnerAllow = TRbacResult("owner")
	UserAllow  = TRbacResult("user")
	GuestAllow = TRbacResult("guest")
	Deny       = TRbacResult("deny")
)

var (
	strictness = map[TRbacResult]int{
		Deny:       0,
		AdminAllow: 1,
		OwnerAllow: 2,
		UserAllow:  3,
		GuestAllow: 4,
	}
)

func (r TRbacResult) Strictness() int {
	return strictness[r]
}

func (r1 TRbacResult) StricterThan(r2 TRbacResult) bool {
	return r1.Strictness() < r2.Strictness()
}

type SRbacPolicy struct {
	Condition string
	IsAdmin   bool
	Rules     []SRbacRule
}

type SRbacRule struct {
	Service  string
	Resource string
	Action   string
	Extra    []string
	Result   TRbacResult
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
	maxMatchCnt := 0
	minWeight := 1000000
	var matchRule *SRbacRule
	for i := 0; i < len(policy.Rules); i += 1 {
		match, matchCnt, weight := policy.Rules[i].match(service, resource, action, extra...)
		if match && (maxMatchCnt < matchCnt ||
			(maxMatchCnt == matchCnt && minWeight > weight) ||
			(maxMatchCnt == matchCnt && minWeight == weight && matchRule.stricterThan(&policy.Rules[i]))) {
			maxMatchCnt = matchCnt
			minWeight = weight
			matchRule = &policy.Rules[i]
		}
	}
	return matchRule
}

func CompactRules(rules []SRbacRule) []SRbacRule {
	if len(rules) == 0 {
		return nil
	}
	output := make([]SRbacRule, 1)
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
	}
	return output
}

func (policy *SRbacPolicy) Decode(policyJson jsonutils.JSONObject) error {
	policy.Condition, _ = policyJson.GetString("condition")
	policy.IsAdmin = jsonutils.QueryBoolean(policyJson, "is_admin", false)

	ruleJson, err := policyJson.Get("policy")
	if err != nil {
		return err
	}

	rules, err := decode(policy.IsAdmin, ruleJson, SRbacRule{}, levelService)
	if err != nil {
		return err
	}

	if len(rules) == 0 {
		return fmt.Errorf("empty policy")
	}

	policy.Rules = CompactRules(rules)

	return nil
}

const (
	levelService  = 0
	levelResource = 1
	levelAction   = 2
	levelExtra    = 3
)

func decode(isAdmin bool, rules jsonutils.JSONObject, decodeRule SRbacRule, level int) ([]SRbacRule, error) {
	switch rules.(type) {
	case *jsonutils.JSONString:
		ruleJsonStr := rules.(*jsonutils.JSONString)
		ruleStr, _ := ruleJsonStr.GetString()
		switch ruleStr {
		case string(Allow):
			if isAdmin {
				decodeRule.Result = AdminAllow
			} else {
				decodeRule.Result = OwnerAllow
			}
		case string(AdminAllow):
			decodeRule.Result = AdminAllow
		case string(OwnerAllow):
			decodeRule.Result = OwnerAllow
		case string(UserAllow):
			decodeRule.Result = UserAllow
		case string(GuestAllow):
			decodeRule.Result = GuestAllow
		case string(Deny):
			decodeRule.Result = Deny
		default:
			return nil, fmt.Errorf("unsupported rule string %s", ruleStr)
		}
		return []SRbacRule{decodeRule}, nil
	case *jsonutils.JSONDict:
		ruleJsonDict, err := rules.GetMap()
		if err != nil {
			return nil, fmt.Errorf("get rule map fail %s", err)
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
			decoded, err := decode(isAdmin, ruleJson, rule, level+1)
			if err != nil {
				return nil, err
			}
			rules = append(rules, decoded...)
		}
		return rules, nil
	default:
		return nil, fmt.Errorf("unsupport rule data %s", rules.String())
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
	for i >= 0 && (len(strArr[i]) == 0 || strArr[i] == WILD_MATCH) {
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
				return fmt.Errorf("conflict?")
			case *jsonutils.JSONDict:
				nextJsonDict := nextJson.(*jsonutils.JSONDict)
				addRule2Json(nextJsonDict, []string{WILD_MATCH}, result)
				return nil
			default:
				return fmt.Errorf("invalid rules")
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
			return fmt.Errorf("invalid rules")
		}
	} else {
		next := jsonutils.NewDict()
		nodeJson.Add(next, keys[0])
		return addRule2Json(next, keys[1:], result)
	}
}

func (policy *SRbacPolicy) Encode() (jsonutils.JSONObject, error) {
	rules := jsonutils.NewDict()
	for i := 0; i < len(policy.Rules); i += 1 {
		keys := policy.Rules[i].toStringArray()
		err := addRule2Json(rules, keys, policy.Rules[i].Result)
		if err != nil {
			return nil, err
		}
	}

	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(policy.Condition), "condition")
	if policy.IsAdmin {
		ret.Add(jsonutils.JSONTrue, "is_admin")
	} else {
		ret.Add(jsonutils.JSONFalse, "is_admin")
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

func (policy *SRbacPolicy) Allow(userCred jsonutils.JSONObject, service, resource, action string, extra ...string) TRbacResult {
	if len(policy.Condition) > 0 {
		match, err := conditionparser.Eval(policy.Condition, userCred)
		if err != nil {
			log.Errorf("eval condition %s fail %s", policy.Condition, err)
			return Deny
		}
		if !match {
			return Deny
		}
	}
	rule := policy.GetMatchRule(service, resource, action, extra...)
	if rule == nil {
		return Deny
	}
	return rule.Result
}
