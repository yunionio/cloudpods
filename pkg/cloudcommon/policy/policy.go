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
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/apis"
	identity_api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/hashcache"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/tagutils"
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

type PolicyFetchFunc func(ctx context.Context, token mcclient.TokenCredential) (*mcclient.SFetchMatchPoliciesOutput, error)

var (
	PolicyManager        *SPolicyManager
	DefaultPolicyFetcher PolicyFetchFunc
)

func init() {
	PolicyManager = &SPolicyManager{
		lock: &sync.Mutex{},
	}
	DefaultPolicyFetcher = auth.FetchMatchPolicies
}

type SPolicyManager struct {
	defaultPolicies map[rbacscope.TRbacScope][]*rbacutils.SRbacPolicy

	refreshInterval time.Duration

	policyCache     *hashcache.Cache // policy cache
	permissionCache *hashcache.Cache // permission cache

	fetchWorker *appsrv.SWorkerManager

	lock *sync.Mutex
}

type sPolicyData struct {
	Id            string               `json:"id"`
	Name          string               `json:"name"`
	Enabled       bool                 `json:"enabled"`
	DomainId      string               `json:"domain_id"`
	IsPublic      bool                 `json:"is_public"`
	PublicScope   rbacscope.TRbacScope `json:"public_scope"`
	SharedDomains []apis.SharedDomain  `json:"shared_domain"`
	Scope         rbacscope.TRbacScope `json:"scope"`
	Policy        jsonutils.JSONObject `json:"policy"`
	DomainTags    tagutils.TTagSet     `json:"domain_tags"`
	ProjectTags   tagutils.TTagSet     `json:"project_tags"`
	ObjectTags    tagutils.TTagSet     `json:"resource_tags"`

	OrgNodes []identity_api.SOrganizationNodeInfo `json:"org_nodes"`
}

func (data sPolicyData) getPolicy() (*rbacutils.SPolicy, error) {
	var domainTags, projectTags, objectTags tagutils.TTagSetList
	if len(data.DomainTags) > 0 {
		domainTags = domainTags.Append(data.DomainTags)
	}
	if len(data.ProjectTags) > 0 {
		projectTags = projectTags.Append(data.ProjectTags)
	}
	if len(data.ObjectTags) > 0 {
		objectTags = objectTags.Append(data.ObjectTags)
	}
	for i := range data.OrgNodes {
		orgNode := data.OrgNodes[i]
		switch orgNode.Type {
		case identity_api.OrgTypeDomain:
			domainTags = domainTags.Append(orgNode.Tags)
		case identity_api.OrgTypeProject:
			projectTags = projectTags.Append(orgNode.Tags)
		case identity_api.OrgTypeObject:
			objectTags = objectTags.Append(orgNode.Tags)
		}
	}
	return rbacutils.DecodePolicyData(domainTags, projectTags, objectTags, data.Policy)
}

func (manager *SPolicyManager) init(refreshInterval time.Duration, workerCount int) {
	manager.refreshInterval = refreshInterval
	// manager.InitSync(manager)
	if len(predefinedDefaultPolicies) > 0 {
		policiesMap := make(map[rbacscope.TRbacScope][]*rbacutils.SRbacPolicy)
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
	}

	manager.policyCache = hashcache.NewCache(2048, refreshInterval)
	manager.permissionCache = hashcache.NewCache(2048, refreshInterval)

	defaultFetcherFuncAddr := reflect.ValueOf(DefaultPolicyFetcher).Pointer()
	remoteFetcherFuncAddr := reflect.ValueOf(auth.FetchMatchPolicies).Pointer()
	log.Debugf("DefaultPolicyFetcher: %x RemotePolicyFetcher: %x", defaultFetcherFuncAddr, remoteFetcherFuncAddr)
	var isDB bool
	if defaultFetcherFuncAddr == remoteFetcherFuncAddr {
		// remote fetcher, so start watcher
		isDB = false
	} else {
		isDB = true
	}

	if workerCount <= 0 {
		workerCount = 1
	}
	log.Infof("policy fetch worker count %d", workerCount)
	manager.fetchWorker = appsrv.NewWorkerManager("policyFetchWorker", workerCount, 2048, isDB)
}

func getMaskedLoginIp(userCred mcclient.TokenCredential) string {
	loginIp, _ := netutils.NewIPV4Addr(userCred.GetLoginIp())
	return loginIp.NetAddr(16).String()
}

func policyKey(userCred mcclient.TokenCredential) string {
	if userCred == nil || auth.IsGuestToken(userCred) {
		return auth.GUEST_TOKEN
	}
	keys := []string{userCred.GetProjectId()}
	roles := userCred.GetRoleIds()
	if len(roles) > 0 {
		sort.Strings(roles)
	}
	keys = append(keys, strings.Join(roles, ":"))
	keys = append(keys, getMaskedLoginIp(userCred))
	return strings.Join(keys, "-")
}

func permissionKey(scope rbacscope.TRbacScope, userCred mcclient.TokenCredential, service string, resource string, action string, extra ...string) string {
	queryKeys := []string{string(scope)}
	queryKeys = append(queryKeys, userCred.GetProjectId())
	roles := userCred.GetRoleIds()
	if len(roles) > 0 {
		sort.Strings(roles)
	}
	queryKeys = append(queryKeys, strings.Join(roles, ":"))
	queryKeys = append(queryKeys, getMaskedLoginIp(userCred))
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

func (manager *SPolicyManager) AllowScope(userCred mcclient.TokenCredential, service string, resource string, action string, extra ...string) (rbacscope.TRbacScope, rbacutils.SPolicyResult) {
	for _, scope := range []rbacscope.TRbacScope{
		rbacscope.ScopeSystem,
		rbacscope.ScopeDomain,
		rbacscope.ScopeProject,
		rbacscope.ScopeUser,
	} {
		result := manager.allow(scope, userCred, service, resource, action, extra...)
		if result.Result == rbacutils.Allow {
			return scope, result
		}
	}
	return rbacscope.ScopeNone, rbacutils.PolicyDeny
}

func (manager *SPolicyManager) Allow(targetScope rbacscope.TRbacScope, userCred mcclient.TokenCredential, service string, resource string, action string, extra ...string) rbacutils.SPolicyResult {
	var retryScopes []rbacscope.TRbacScope
	switch targetScope {
	case rbacscope.ScopeSystem:
		retryScopes = []rbacscope.TRbacScope{
			rbacscope.ScopeSystem,
		}
	case rbacscope.ScopeDomain:
		retryScopes = []rbacscope.TRbacScope{
			rbacscope.ScopeSystem,
			rbacscope.ScopeDomain,
		}
	case rbacscope.ScopeProject:
		retryScopes = []rbacscope.TRbacScope{
			rbacscope.ScopeSystem,
			rbacscope.ScopeDomain,
			rbacscope.ScopeProject,
		}
	case rbacscope.ScopeUser:
		retryScopes = []rbacscope.TRbacScope{
			rbacscope.ScopeSystem,
			rbacscope.ScopeUser,
		}
	}
	for _, scope := range retryScopes {
		result := manager.allow(scope, userCred, service, resource, action, extra...)
		if result.Result == rbacutils.Allow {
			return result
		}
	}
	return rbacutils.PolicyDeny
}

type fetchResult struct {
	output *mcclient.SFetchMatchPoliciesOutput
	err    error
}

type policyTask struct {
	manager  *SPolicyManager
	key      string
	userCred mcclient.TokenCredential
	resChan  chan fetchResult
}

func (t *policyTask) Run() {
	val := t.manager.policyCache.AtomicGet(t.key)
	result := fetchResult{}
	if gotypes.IsNil(val) {
		pg, err := DefaultPolicyFetcher(context.Background(), t.userCred)
		if err != nil {
			result.err = errors.Wrap(err, "DefaultPolicyFetcher")
		} else {
			t.manager.policyCache.AtomicSet(t.key, pg)
			result.output = pg
		}
	} else {
		result.output = val.(*mcclient.SFetchMatchPoliciesOutput)
	}
	t.resChan <- result
}

func (t *policyTask) Dump() string {
	return ""
}

func (manager *SPolicyManager) fetchMatchedPolicies(userCred mcclient.TokenCredential) (*mcclient.SFetchMatchPoliciesOutput, error) {
	key := policyKey(userCred)

	task := policyTask{
		manager:  manager,
		key:      key,
		userCred: userCred,
	}
	task.resChan = make(chan fetchResult)

	manager.fetchWorker.Run(&task, nil, nil)

	res := <-task.resChan
	return res.output, res.err
}

func (manager *SPolicyManager) allow(scope rbacscope.TRbacScope, userCred mcclient.TokenCredential, service string, resource string, action string, extra ...string) rbacutils.SPolicyResult {
	// first download userCred policy
	policies, err := manager.fetchMatchedPolicies(userCred)
	if err != nil {
		log.Errorf("fetchMatchedPolicyGroup fail %s", err)
		return rbacutils.PolicyDeny
	}
	// check permission
	key := permissionKey(scope, userCred, service, resource, action, extra...)
	val := manager.permissionCache.AtomicGet(key)
	if !gotypes.IsNil(val) {
		if consts.IsRbacDebug() {
			log.Debugf("query %s:%s:%s:%s from cache %s", service, resource, action, extra, val)
		}
		return val.(rbacutils.SPolicyResult)
	}

	policySet, ok := policies.Policies[scope]
	if !ok {
		policySet = rbacutils.TPolicySet{}
	}
	result := manager.allowWithoutCache(policySet, scope, userCred, service, resource, action, extra...)
	manager.permissionCache.AtomicSet(key, result)
	return result
}

/*
func (manager *SPolicyManager) findPolicyByName(scope rbacscope.TRbacScope, name string) *rbacscope.SRbacPolicyCore {
	if policies, ok := manager.policies[scope]; ok {
		for i := range policies {
			if policies[i].Id == name || policies[i].Name == name {
				return policies[i].Policy
			}
		}
	}
	return nil
}

func getMatchedPolicyNames(policies []rbacscope.SPolicyInfo, userCred rbacscope.IRbacIdentity) []string {
	_, matchNames := rbacscope.GetMatchedPolicies(policies, userCred)
	return matchNames
}

func getMatchedPolicyRules(policies []rbacscope.SPolicyInfo, userCred rbacscope.IRbacIdentity, service string, resource string, action string, extra ...string) ([]rbacscope.SRbacRule, bool) {
	matchPolicies, _ := rbacscope.GetMatchedPolicies(policies, userCred)
	if len(matchPolicies) == 0 {
		return nil, false
	}
	return matchPolicies.GetMatchRules(service, resource, action, extra...), true
}
*/

func (manager *SPolicyManager) allowWithoutCache(policies rbacutils.TPolicySet, scope rbacscope.TRbacScope, userCred mcclient.TokenCredential, service string, resource string, action string, extra ...string) rbacutils.SPolicyResult {
	matchRules := rbacutils.TPolicyMatches{}

	if len(policies) == 0 {
		log.Warningf("no policies fetched for scope %s", scope)
	} else {
		matchRules = policies.GetMatchRules(service, resource, action, extra...)
		if consts.IsRbacDebug() {
			log.Debugf("service %s resource %s action %s extra %s matchRules: %s", service, resource, action, jsonutils.Marshal(extra), jsonutils.Marshal(matchRules))
		}
	}

	scopedDeny := false
	switch scope {
	case rbacscope.ScopeUser:
		if !isUserResource(service, resource) {
			scopedDeny = true
		}
	case rbacscope.ScopeProject:
		if !isProjectResource(service, resource) {
			scopedDeny = true
		}
	case rbacscope.ScopeDomain:
		if isSystemResource(service, resource) {
			scopedDeny = true
		}
	case rbacscope.ScopeSystem:
		// no deny at all for system scope
	}
	if scopedDeny {
		rule := rbacutils.SPolicyMatch{
			Rule: rbacutils.SRbacRule{
				Service:  service,
				Resource: resource,
				Result:   rbacutils.Deny,
			},
		}
		matchRules = append(matchRules, rule)
	}

	result := matchRules.GetResult()
	if result.Result.IsDeny() {
		// denied, try default policies
		defaultPolicies, ok := manager.defaultPolicies[scope]
		if ok {
			for i := range defaultPolicies {
				isMatched, _ := defaultPolicies[i].Match(userCred)
				if !isMatched {
					continue
				}
				rule := defaultPolicies[i].Rules.GetMatchRule(service, resource, action, extra...)
				if rule != nil {
					if consts.IsRbacDebug() {
						log.Debugf("service: %s resource: %s action: %s extra: %s match default policy: %s match rule: %s", service, resource, action, jsonutils.Marshal(extra), jsonutils.Marshal(defaultPolicies[i]), jsonutils.Marshal(rule))
					}
					matchRules = append(matchRules,
						rbacutils.SPolicyMatch{
							Rule: *rule,
						},
					)
				}
			}
		}
		result = matchRules.GetResult()
	}
	if consts.IsRbacDebug() {
		log.Debugf("[RBAC: %s] %s %s %s %s permission %s userCred: %s MatchRules: %d(%s)", scope, service, resource, action, jsonutils.Marshal(extra), result, userCred, len(matchRules), jsonutils.Marshal(matchRules))
	}
	return result
}

// result: allow/deny for the named policy
// userResult: allow/deny for the matched policies of userCred
func explainPolicy(userCred mcclient.TokenCredential, policyReq jsonutils.JSONObject, policyData *sPolicyData) ([]string, rbacutils.SPolicyResult, rbacutils.SPolicyResult, error) {
	_, request, result, userResult, err := explainPolicyInternal(userCred, policyReq, policyData)
	return request, result, userResult, err
}

func fetchPolicyDataByIdOrName(ctx context.Context, id string) (*sPolicyData, error) {
	s := auth.GetAdminSession(ctx, consts.GetRegion())
	data, err := identity.Policies.Get(s, id, nil)
	if err != nil {
		return nil, errors.Wrap(err, "modules.Policies.Get")
	}
	pdata := &sPolicyData{}
	err = data.Unmarshal(&pdata)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal Policy Data")
	}
	return pdata, nil
}

func explainPolicyInternal(userCred mcclient.TokenCredential, policyReq jsonutils.JSONObject, policyData *sPolicyData) (rbacscope.TRbacScope, []string, rbacutils.SPolicyResult, rbacutils.SPolicyResult, error) {
	policySeq, err := policyReq.GetArray()
	if err != nil {
		return rbacscope.ScopeSystem, nil, rbacutils.PolicyDeny, rbacutils.PolicyDeny, httperrors.NewInputParameterError("invalid format")
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
	scope := rbacscope.String2Scope(scopeStr)

	userResult := PolicyManager.Allow(scope, userCred, service, resource, action, extra...)
	result := userResult

	if policyData != nil {
		if scope.HigherThan(policyData.Scope) {
			result = rbacutils.PolicyDeny
		} else {
			policy, err := policyData.getPolicy()
			if err != nil {
				return scope, reqStrs, rbacutils.PolicyDeny, userResult, errors.Wrap(err, "getPolicy")
			}
			match := policy.GetMatchRule(service, resource, action, extra...)
			result = rbacutils.PolicyDeny
			if match != nil {
				result.Result = match.Rule.Result
				result.DomainTags = match.DomainTags
				result.ProjectTags = match.ProjectTags
				result.ObjectTags = match.ObjectTags
			}
		}
	}

	return scope, reqStrs, result, userResult, nil
}

func ExplainRpc(ctx context.Context, userCred mcclient.TokenCredential, params jsonutils.JSONObject, name string) (jsonutils.JSONObject, error) {
	paramDict, err := params.GetMap()
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid input format")
	}
	var policyData *sPolicyData
	if len(name) > 0 {
		policyData, err = fetchPolicyDataByIdOrName(ctx, name)
		if err != nil {
			return nil, errors.Wrap(err, "fetchPolicyDataByIdOrName")
		}
	}
	ret := jsonutils.NewDict()
	for key, policyReq := range paramDict {
		reqStrs, result, userResult, err := explainPolicy(userCred, policyReq, policyData)
		if err != nil {
			return nil, err
		}
		reqStrs = append(reqStrs, string(result.Result))
		if len(name) > 0 {
			reqStrs = append(reqStrs, string(userResult.Result))
		}
		ret.Add(jsonutils.NewStringArray(reqStrs), key)
	}
	return ret, nil
}

func (manager *SPolicyManager) IsScopeCapable(userCred mcclient.TokenCredential, scope rbacscope.TRbacScope) bool {
	policies, err := manager.fetchMatchedPolicies(userCred)
	if err != nil {
		log.Errorf("fetchMatchedPolicyGroup fail %s", err)
		return false
	}

	if set, ok := policies.Policies[scope]; ok && len(set) > 0 {
		return true
	}

	return false
}

/*
func (manager *SPolicyManager) MatchedPolicyNames(ctx context.Context, scope rbacscope.TRbacScope, ident rbacscope.IRbacIdentity) []string {
	policies, err := manager.fetchMatchedPolicies(ctx, userCred)
	if err != nil {
		log.Errorf("fetchMatchedPolicyGroup fail %s", err)
		return false
	}

	ret := make([]string, 0)
	policies, ok := manager.policies[scope]
	if !ok {
		return ret
	}
	return getMatchedPolicyNames(policies, userCred)
}*/

/*
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
	ident := rbacscope.NewRbacIdentity("", "", []string{roleName})
	ret := make([]string, 0)
	for _, policies := range manager.policies {
		for i := range policies {
			if matched, _ := policies[i].Policy.Match(ident); matched {
				ret = append(ret, policies[i].Name)
			}
		}
	}
	return ret
}

func (manager *SPolicyManager) GetMatchedPolicySet(userCred rbacscope.IRbacIdentity) (rbacscope.TRbacScope, rbacscope.TPolicySet) {
	for _, scope := range []rbacscope.TRbacScope{
		rbacscope.ScopeSystem,
		rbacscope.ScopeDomain,
		rbacscope.ScopeProject,
	} {
		macthed, _ := rbacscope.GetMatchedPolicies(manager.policies[scope], userCred)
		if len(macthed) > 0 {
			return scope, macthed
		}
	}
	return rbacscope.ScopeNone, nil
}
*/
