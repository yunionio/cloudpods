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

package policy

import (
	"context"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/hashcache"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

const (
	PolicyDelegation = "delegate"

	PolicyActionList    = rbacutils.ActionList
	PolicyActionGet     = rbacutils.ActionGet
	PolicyActionUpdate  = rbacutils.ActionUpdate
	PolicyActionPatch   = rbacutils.ActionPatch
	PolicyActionCreate  = rbacutils.ActionCreate
	PolicyActionDelete  = rbacutils.ActionDelete
	PolicyActionPerform = rbacutils.ActionPerform
)

type PolicyFetchFunc func(ctx context.Context) (map[rbacutils.TRbacScope][]rbacutils.SPolicyInfo, error)

var (
	PolicyManager        *SPolicyManager
	DefaultPolicyFetcher PolicyFetchFunc

	syncWorkerManager *appsrv.SWorkerManager
)

func init() {
	PolicyManager = &SPolicyManager{
		lock: &sync.Mutex{},
	}
	DefaultPolicyFetcher = remotePolicyFetcher

	// no need to queue many sync tasks
	syncWorkerManager = appsrv.NewWorkerManagerIgnoreOverflow("sync_policy_worker", 1, 2, true, true)
}

type SPolicyManager struct {
	// policies        map[rbacutils.TRbacScope]map[string]*rbacutils.SRbacPolicy
	policies        map[rbacutils.TRbacScope][]rbacutils.SPolicyInfo
	defaultPolicies map[rbacutils.TRbacScope][]*rbacutils.SRbacPolicy
	lastSync        time.Time

	failedRetryInterval time.Duration
	refreshInterval     time.Duration

	cache *hashcache.Cache // policy cache

	lock *sync.Mutex
}

type sPolicyData struct {
	Id            string               `json:"id"`
	Type          string               `json:"type"`
	Enabled       bool                 `json:"enabled"`
	DomainId      string               `json:"domain_id"`
	IsPublic      bool                 `json:"is_public"`
	PublicScope   string               `json:"public_scope"`
	SharedDomains []apis.SharedDomain  `json:"shared_domain"`
	Policy        jsonutils.JSONObject `json:"policy"`
}

func parseJsonPolicy(obj jsonutils.JSONObject) (rbacutils.SPolicyInfo, error) {
	sp := rbacutils.SPolicyInfo{}
	pData := sPolicyData{}
	err := obj.Unmarshal(&pData)
	if err != nil {
		return sp, errors.Wrap(err, "Unmarshal")
	}
	if !pData.Enabled {
		return sp, errors.Wrap(httperrors.ErrInvalidFormat, "not enabled")
	}
	if len(pData.Type) == 0 {
		return sp, errors.Wrap(httperrors.ErrInvalidFormat, "missing type")
	}

	if pData.Policy == nil {
		return sp, errors.Wrap(httperrors.ErrInvalidFormat, "missing policy")
	}

	policy := rbacutils.SRbacPolicy{}
	err = policy.Decode(pData.Policy)
	if err != nil {
		log.Errorf("policy decode error %s", err)
		return sp, errors.Wrap(err, "policy.Decode")
	}

	policy.DomainId = pData.DomainId
	policy.IsPublic = pData.IsPublic
	policy.PublicScope = rbacutils.String2ScopeDefault(pData.PublicScope, rbacutils.ScopeSystem)
	policy.SharedDomainIds = make([]string, len(pData.SharedDomains))
	for i := range pData.SharedDomains {
		policy.SharedDomainIds[i] = pData.SharedDomains[i].Id
	}

	sp.Id = pData.Id
	sp.Name = pData.Type
	sp.Policy = &policy
	return sp, nil
}

func remotePolicyFetcher(ctx context.Context) (map[rbacutils.TRbacScope][]rbacutils.SPolicyInfo, error) {
	s := auth.GetAdminSession(ctx, consts.GetRegion(), "v1")

	policies := make(map[rbacutils.TRbacScope][]rbacutils.SPolicyInfo)

	offset := 0
	for {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewInt(2048), "limit")
		params.Add(jsonutils.NewInt(int64(offset)), "offset")
		params.Add(jsonutils.NewString("system"), "scope")
		params.Add(jsonutils.JSONTrue, "enabled")
		result, err := modules.Policies.List(s, params)
		if err != nil {
			return nil, errors.Wrap(err, "modules.Policies.List")
		}

		for i := 0; i < len(result.Data); i += 1 {
			sp, err := parseJsonPolicy(result.Data[i])
			if err != nil {
				log.Errorf("error parse policty %s", err)
				continue
			}

			if _, ok := policies[sp.Policy.Scope]; !ok {
				policies[sp.Policy.Scope] = make([]rbacutils.SPolicyInfo, 0)
			}
			policies[sp.Policy.Scope] = append(policies[sp.Policy.Scope], sp)
		}

		offset += len(result.Data)
		if offset >= result.Total {
			break
		}
	}
	return policies, nil
}

func (manager *SPolicyManager) start(refreshInterval time.Duration, retryInterval time.Duration) {
	log.Infof("PolicyManager start to fetch policies ...")
	manager.refreshInterval = refreshInterval
	manager.failedRetryInterval = retryInterval
	if len(predefinedDefaultPolicies) > 0 {
		policiesMap := make(map[rbacutils.TRbacScope][]*rbacutils.SRbacPolicy)
		for i := range predefinedDefaultPolicies {
			policy := predefinedDefaultPolicies[i]
			if _, ok := policiesMap[policy.Scope]; !ok {
				policiesMap[policy.Scope] = make([]*rbacutils.SRbacPolicy, 0)
			}
			policies := policiesMap[policy.Scope]
			policies = append(policies, &policy)
			policiesMap[policy.Scope] = policies
		}
		manager.defaultPolicies = policiesMap
		log.Debugf("%#v", manager.defaultPolicies)
	}

	manager.cache = hashcache.NewCache(2048, manager.refreshInterval/2)

	manager.SyncOnce()
}

func (manager *SPolicyManager) SyncOnce() {
	syncWorkerManager.Run(manager.sync, nil, nil)
}

func (manager *SPolicyManager) doSync() error {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("policyManager doSync error %s", r)
			debug.PrintStack()
		}
	}()

	policies, err := DefaultPolicyFetcher(context.Background())
	if err != nil {
		log.Errorf("sync rbac policy failed: %s", err)
		return errors.Wrap(err, "DefaultPolicyFetcher")
	}

	manager.lock.Lock()
	defer manager.lock.Unlock()

	manager.policies = policies

	manager.lastSync = time.Now()
	manager.cache.Invalidate()

	return nil
}

func (manager *SPolicyManager) sync() {
	err := manager.doSync()
	var interval time.Duration
	if err != nil {
		interval = manager.failedRetryInterval
	} else {
		interval = manager.refreshInterval
	}
	time.AfterFunc(interval, manager.SyncOnce)
}

func queryKey(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, service string, resource string, action string, extra ...string) string {
	queryKeys := []string{string(scope)}
	queryKeys = append(queryKeys, userCred.GetProjectId(), userCred.GetDomainId(), userCred.GetUserId())
	roles := userCred.GetRoles()
	if len(roles) > 0 {
		sort.Strings(roles)
	}
	queryKeys = append(queryKeys, strings.Join(roles, ":"))
	if rbacutils.WILD_MATCH == service || len(service) == 0 {
		service = rbacutils.WILD_MATCH
	}
	queryKeys = append(queryKeys, service)
	if rbacutils.WILD_MATCH == resource || len(resource) == 0 {
		resource = rbacutils.WILD_MATCH
	}
	queryKeys = append(queryKeys, resource)
	if rbacutils.WILD_MATCH == action || len(action) == 0 {
		action = rbacutils.WILD_MATCH
	}
	queryKeys = append(queryKeys, action)
	if len(extra) > 0 {
		queryKeys = append(queryKeys, extra...)
	}
	return strings.Join(queryKeys, "-")
}

func (manager *SPolicyManager) AllowScope(userCred mcclient.TokenCredential, service string, resource string, action string, extra ...string) rbacutils.TRbacScope {
	for _, scope := range []rbacutils.TRbacScope{
		rbacutils.ScopeSystem,
		rbacutils.ScopeDomain,
		rbacutils.ScopeProject,
		rbacutils.ScopeUser,
	} {
		result := manager.allow(scope, userCred, service, resource, action, extra...)
		if result == rbacutils.Allow {
			return scope
		}
	}
	return rbacutils.ScopeNone
}

func (manager *SPolicyManager) Allow(targetScope rbacutils.TRbacScope, userCred mcclient.TokenCredential, service string, resource string, action string, extra ...string) rbacutils.TRbacResult {
	var retryScopes []rbacutils.TRbacScope
	switch targetScope {
	case rbacutils.ScopeSystem:
		retryScopes = []rbacutils.TRbacScope{
			rbacutils.ScopeSystem,
		}
	case rbacutils.ScopeDomain:
		retryScopes = []rbacutils.TRbacScope{
			rbacutils.ScopeSystem,
			rbacutils.ScopeDomain,
		}
	case rbacutils.ScopeProject:
		retryScopes = []rbacutils.TRbacScope{
			rbacutils.ScopeSystem,
			rbacutils.ScopeDomain,
			rbacutils.ScopeProject,
		}
	case rbacutils.ScopeUser:
		retryScopes = []rbacutils.TRbacScope{
			rbacutils.ScopeSystem,
			rbacutils.ScopeUser,
		}
	}
	for _, scope := range retryScopes {
		result := manager.allow(scope, userCred, service, resource, action, extra...)
		if result == rbacutils.Allow {
			return rbacutils.Allow
		}
	}
	return rbacutils.Deny
}

func (manager *SPolicyManager) allow(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, service string, resource string, action string, extra ...string) rbacutils.TRbacResult {
	if manager.cache != nil && userCred != nil {
		key := queryKey(scope, userCred, service, resource, action, extra...)
		val := manager.cache.Get(key)
		if val != nil {
			if consts.IsRbacDebug() {
				log.Debugf("query %s:%s:%s:%s from cache %s", service, resource, action, extra, val)
			}
			return val.(rbacutils.TRbacResult)
		}
		result := manager.allowWithoutCache(scope, userCred, service, resource, action, extra...)
		manager.cache.Set(key, result)
		return result
	} else {
		return manager.allowWithoutCache(scope, userCred, service, resource, action, extra...)
	}
}

func (manager *SPolicyManager) findPolicyByName(scope rbacutils.TRbacScope, name string) *rbacutils.SRbacPolicy {
	if policies, ok := manager.policies[scope]; ok {
		for i := range policies {
			if policies[i].Id == name || policies[i].Name == name {
				return policies[i].Policy
			}
		}
	}
	return nil
}

func getMatchedPolicyNames(policies []rbacutils.SPolicyInfo, userCred rbacutils.IRbacIdentity) []string {
	_, matchNames := rbacutils.GetMatchedPolicies(policies, userCred)
	return matchNames
}

func getMatchedPolicyRules(policies []rbacutils.SPolicyInfo, userCred rbacutils.IRbacIdentity, service string, resource string, action string, extra ...string) ([]rbacutils.SRbacRule, bool) {
	matchPolicies, _ := rbacutils.GetMatchedPolicies(policies, userCred)
	if len(matchPolicies) == 0 {
		return nil, false
	}
	return matchPolicies.GetMatchRules(service, resource, action, extra...), true
}

func (manager *SPolicyManager) allowWithoutCache(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, service string, resource string, action string, extra ...string) rbacutils.TRbacResult {
	matchRules := make([]rbacutils.SRbacRule, 0)
	findMatchPolicy := false
	policies, ok := manager.policies[scope]
	if !ok {
		log.Warningf("no policies fetched for scope %s", scope)
	} else {
		matchRules, findMatchPolicy = getMatchedPolicyRules(policies, userCred, service, resource, action, extra...)
	}

	scopedDeny := false
	switch scope {
	case rbacutils.ScopeUser:
		if !isUserResource(service, resource) {
			scopedDeny = true
		}
	case rbacutils.ScopeProject:
		if !isProjectResource(service, resource) {
			scopedDeny = true
		}
	case rbacutils.ScopeDomain:
		if isSystemResource(service, resource) {
			scopedDeny = true
		}
	case rbacutils.ScopeSystem:
		// no deny at all for system scope
	}
	if scopedDeny {
		rule := rbacutils.SRbacRule{
			Service:  service,
			Resource: resource,
			Result:   rbacutils.Deny,
		}
		matchRules = append(matchRules, rule)
	}

	// try default policies
	defaultPolicies, ok := manager.defaultPolicies[scope]
	if ok {
		for i := range defaultPolicies {
			isMatched, _ := defaultPolicies[i].Match(userCred)
			if !isMatched {
				continue
			}
			rule := defaultPolicies[i].GetMatchRule(service, resource, action, extra...)
			if rule != nil {
				matchRules = append(matchRules, *rule)
			}
		}
	}

	var result rbacutils.TRbacResult
	if len(matchRules) > 0 {
		rule := rbacutils.GetMatchRule(matchRules, service, resource, action, extra...)
		result = rule.Result
	} else if findMatchPolicy {
		// if find matched policy, but no rule matching, allow anyway
		result = rbacutils.Allow
	} else {
		result = rbacutils.Deny
	}

	if consts.IsRbacDebug() {
		log.Debugf("[RBAC: %s] %s %s %s %#v permission %s userCred: %s MatchRules: %d(%s)", scope, service, resource, action, extra, result, userCred, len(matchRules), jsonutils.Marshal(matchRules))
	}
	return result
}

func explainPolicy(ctx context.Context, userCred mcclient.TokenCredential, policyReq jsonutils.JSONObject, name string) ([]string, rbacutils.TRbacResult, error) {
	_, request, result, err := explainPolicyInternal(ctx, userCred, policyReq, name)
	return request, result, err
}

func fetchPolicyByIdOrName(ctx context.Context, id string) (rbacutils.SPolicyInfo, error) {
	s := auth.GetAdminSession(ctx, consts.GetRegion(), "v1")
	data, err := modules.Policies.Get(s, id, nil)
	if err != nil {
		return rbacutils.SPolicyInfo{}, errors.Wrap(err, "modules.Policies.Get")
	}
	return parseJsonPolicy(data)
}

func explainPolicyInternal(ctx context.Context, userCred mcclient.TokenCredential, policyReq jsonutils.JSONObject, name string) (rbacutils.TRbacScope, []string, rbacutils.TRbacResult, error) {
	policySeq, err := policyReq.GetArray()
	if err != nil {
		return rbacutils.ScopeSystem, nil, rbacutils.Deny, httperrors.NewInputParameterError("invalid format")
	}
	service := rbacutils.WILD_MATCH
	resource := rbacutils.WILD_MATCH
	action := rbacutils.WILD_MATCH
	extra := make([]string, 0)
	if len(policySeq) > 1 {
		service, _ = policySeq[1].GetString()
	}
	if len(policySeq) > 2 {
		resource, _ = policySeq[2].GetString()
	}
	if len(policySeq) > 3 {
		action, _ = policySeq[3].GetString()
	}
	if len(policySeq) > 4 {
		for i := 4; i < len(policySeq); i += 1 {
			ev, _ := policySeq[i].GetString()
			extra = append(extra, ev)
		}
	}

	reqStrs := []string{service, resource, action}
	if len(extra) > 0 {
		reqStrs = append(reqStrs, extra...)
	}

	scopeStr, _ := policySeq[0].GetString()
	scope := rbacutils.String2Scope(scopeStr)
	if !consts.IsRbacEnabled() {
		if scope == rbacutils.ScopeProject || (scope == rbacutils.ScopeSystem && userCred.HasSystemAdminPrivilege()) {
			return scope, reqStrs, rbacutils.Allow, nil
		} else {
			return scope, reqStrs, rbacutils.Deny, httperrors.NewForbiddenError("operation not allowed")
		}
	}

	if len(name) == 0 {
		return scope, reqStrs, PolicyManager.Allow(scope, userCred, service, resource, action, extra...), nil
	}

	policy := PolicyManager.findPolicyByName(scope, name)
	if policy == nil {
		// policy not found locally, remote fetch
		sp, err := fetchPolicyByIdOrName(ctx, name)
		if err != nil {
			return scope, reqStrs, rbacutils.Deny, httperrors.NewNotFoundError("policy %s not found", name)
		}
		policy = sp.Policy
	}

	rule := policy.GetMatchRule(service, resource, action, extra...)
	result := rbacutils.Deny
	if rule != nil {
		result = rule.Result
	}
	return scope, reqStrs, result, nil
}

func ExplainRpc(ctx context.Context, userCred mcclient.TokenCredential, params jsonutils.JSONObject, name string) (jsonutils.JSONObject, error) {
	paramDict, err := params.GetMap()
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid input format")
	}
	ret := jsonutils.NewDict()
	for key, policyReq := range paramDict {
		reqStrs, result, err := explainPolicy(ctx, userCred, policyReq, name)
		if err != nil {
			return nil, err
		}
		reqStrs = append(reqStrs, string(result))
		ret.Add(jsonutils.NewStringArray(reqStrs), key)
	}
	return ret, nil
}

func (manager *SPolicyManager) IsScopeCapable(userCred mcclient.TokenCredential, scope rbacutils.TRbacScope) bool {
	if !consts.IsRbacEnabled() {
		if userCred.HasSystemAdminPrivilege() {
			return true
		}
		if scope == rbacutils.ScopeProject {
			return true
		}
		return false
	}

	if policies, ok := manager.policies[scope]; ok {
		pnames := getMatchedPolicyNames(policies, userCred)
		if len(pnames) > 0 {
			return true
		}
	}
	return false
}

func (manager *SPolicyManager) MatchedPolicyNames(scope rbacutils.TRbacScope, userCred rbacutils.IRbacIdentity) []string {
	ret := make([]string, 0)
	policies, ok := manager.policies[scope]
	if !ok {
		return ret
	}
	return getMatchedPolicyNames(policies, userCred)
}

func (manager *SPolicyManager) AllPolicies() map[string][]string {
	ret := make(map[string][]string)
	for scope, p := range manager.policies {
		k := string(scope)
		ret[k] = make([]string, len(p))
		for i := range p {
			ret[k][i] = p[i].Name
		}
	}
	return ret
}

func (manager *SPolicyManager) RoleMatchPolicies(roleName string) []string {
	ret := make([]string, 0)
	for _, policies := range manager.policies {
		for i := range policies {
			ident := rbacutils.NewRbacIdentity("", "", []string{roleName})
			if matched, _ := policies[i].Policy.Match(ident); matched {
				ret = append(ret, policies[i].Name)
			}
		}
	}
	return ret
}

func (manager *SPolicyManager) GetMatchedPolicySet(userCred rbacutils.IRbacIdentity) (rbacutils.TRbacScope, rbacutils.TPolicySet) {
	for _, scope := range []rbacutils.TRbacScope{
		rbacutils.ScopeSystem,
		rbacutils.ScopeDomain,
		rbacutils.ScopeProject,
	} {
		macthed, _ := rbacutils.GetMatchedPolicies(manager.policies[scope], userCred)
		if len(macthed) > 0 {
			return scope, macthed
		}
	}
	return rbacutils.ScopeNone, nil
}
