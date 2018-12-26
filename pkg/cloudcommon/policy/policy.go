package policy

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/conditionparser"
	"yunion.io/x/onecloud/pkg/util/hashcache"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

const (
	PolicyDelegation = "delegate"

	PolicyActionList    = "list"
	PolicyActionGet     = "get"
	PolicyActionUpdate  = "update"
	PolicyActionPatch   = "patch"
	PolicyActionCreate  = "create"
	PolicyActionDelete  = "delete"
	PolicyActionPerform = "perform"
)

var (
	PolicyManager *SPolicyManager
)

func init() {
	PolicyManager = &SPolicyManager{
		lock: &sync.Mutex{},
	}
}

type SPolicyManager struct {
	policies      map[string]rbacutils.SRbacPolicy
	adminPolicies map[string]rbacutils.SRbacPolicy
	defaultPolicy *rbacutils.SRbacPolicy
	lastSync      time.Time

	failedRetryInterval time.Duration
	refreshInterval     time.Duration

	cache *hashcache.Cache // policy cache

	lock *sync.Mutex
}

func parseJsonPolicy(obj jsonutils.JSONObject) (string, rbacutils.SRbacPolicy, error) {
	policy := rbacutils.SRbacPolicy{}
	typeStr, err := obj.GetString("type")
	if err != nil {
		log.Errorf("get type error %s", err)
		return "", policy, err
	}

	blob, err := obj.Get("policy")
	if err != nil {
		log.Errorf("get blob error %s", err)
		return "", policy, err
	}
	err = policy.Decode(blob)
	if err != nil {
		log.Errorf("policy decode error %s", err)
		return "", policy, err
	}

	return typeStr, policy, nil
}

func fetchPolicies() (map[string]rbacutils.SRbacPolicy, map[string]rbacutils.SRbacPolicy, error) {
	s := auth.GetAdminSession(context.Background(), consts.GetRegion(), "v1")

	policies := make(map[string]rbacutils.SRbacPolicy)
	adminPolicies := make(map[string]rbacutils.SRbacPolicy)

	offset := 0
	for {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewInt(2048), "limit")
		params.Add(jsonutils.NewInt(int64(offset)), "offset")
		result, err := modules.Policies.List(s, params)

		if err != nil {
			log.Errorf("fetch policy failed")
			return nil, nil, err
		}

		for i := 0; i < len(result.Data); i += 1 {
			typeStr, policy, err := parseJsonPolicy(result.Data[i])
			if err != nil {
				log.Errorf("error parse policty %s", err)
				continue
			}

			if policy.IsAdmin {
				adminPolicies[typeStr] = policy
			} else {
				policies[typeStr] = policy
			}
		}

		offset += len(result.Data)
		if offset >= result.Total {
			break
		}
	}

	return policies, adminPolicies, nil
}

func (manager *SPolicyManager) start(refreshInterval time.Duration, retryInterval time.Duration) {
	log.Infof("PolicyManager start to fetch policies ...")
	manager.refreshInterval = refreshInterval
	manager.failedRetryInterval = retryInterval
	if len(defaultRules) > 0 {
		manager.defaultPolicy = &rbacutils.SRbacPolicy{
			Rules: rbacutils.CompactRules(defaultRules),
		}
	}

	manager.cache = hashcache.NewCache(2048, manager.refreshInterval/2)
	manager.sync()
}

func (manager *SPolicyManager) SyncOnce() error {
	policies, adminPolicies, err := fetchPolicies()
	if err != nil {
		log.Errorf("sync rbac policy failed: %s", err)
		return err
	}

	manager.lock.Lock()
	defer manager.lock.Unlock()

	manager.policies = policies
	manager.adminPolicies = adminPolicies

	manager.lastSync = time.Now()

	return nil
}

func (manager *SPolicyManager) sync() {
	err := manager.SyncOnce()
	var interval time.Duration
	if err != nil {
		interval = manager.failedRetryInterval
	} else {
		interval = manager.refreshInterval
	}
	time.AfterFunc(interval, manager.sync)
}

func queryKey(isAdmin bool, userCred mcclient.TokenCredential, service string, resource string, action string, extra ...string) string {
	queryKeys := []string{fmt.Sprintf("%v", isAdmin)}
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

func (manager *SPolicyManager) Allow(isAdmin bool, userCred mcclient.TokenCredential, service string, resource string, action string, extra ...string) rbacutils.TRbacResult {
	if manager.cache != nil {
		key := queryKey(isAdmin, userCred, service, resource, action, extra...)
		val := manager.cache.Get(key)
		if val != nil {
			return val.(rbacutils.TRbacResult)
		}
		result := manager.allowWithoutCache(isAdmin, userCred, service, resource, action, extra...)
		manager.cache.Set(key, result)
		return result
	} else {
		return manager.allowWithoutCache(isAdmin, userCred, service, resource, action, extra...)
	}
}

func (manager *SPolicyManager) findPolicyByName(isAdmin bool, name string) *rbacutils.SRbacPolicy {
	var policies map[string]rbacutils.SRbacPolicy
	if isAdmin {
		policies = manager.adminPolicies
	} else {
		policies = manager.policies
	}
	if policies == nil {
		return nil
	}
	if p, ok := policies[name]; ok {
		return &p
	}
	return nil
}

func (manager *SPolicyManager) allowWithoutCache(isAdmin bool, userCred mcclient.TokenCredential, service string, resource string, action string, extra ...string) rbacutils.TRbacResult {
	var policies map[string]rbacutils.SRbacPolicy
	if isAdmin {
		policies = manager.adminPolicies
	} else {
		policies = manager.policies
	}
	if policies == nil {
		log.Warningf("no policies fetched")
		return rbacutils.Deny
	}
	userCredJson := userCred.ToJson()
	currentPriv := rbacutils.Deny
	for _, p := range policies {
		result := p.Allow(userCredJson, service, resource, action, extra...)
		if currentPriv.StricterThan(result) {
			currentPriv = result
		}
	}
	if !isAdmin && manager.defaultPolicy != nil {
		result := manager.defaultPolicy.Allow(userCredJson, service, resource, action, extra...)
		if currentPriv.StricterThan(result) {
			currentPriv = result
		}
	}
	if consts.IsRbacDebug() {
		log.Debugf("[RBAC: %v] %s %s %s %#v permission %s", isAdmin, service, resource, action, extra, currentPriv)
	}
	return unifyRbacResult(isAdmin, currentPriv)
}

func unifyRbacResult(isAdmin bool, currentPriv rbacutils.TRbacResult) rbacutils.TRbacResult {
	if isAdmin {
		switch currentPriv {
		case rbacutils.OwnerAllow, rbacutils.UserAllow, rbacutils.GuestAllow:
			currentPriv = rbacutils.AdminAllow
		}
	} else {
		switch currentPriv {
		case rbacutils.AdminAllow:
			currentPriv = rbacutils.UserAllow
		}
	}
	return currentPriv
}

func exportRbacResult(currentPriv rbacutils.TRbacResult) rbacutils.TRbacResult {
	if currentPriv == rbacutils.AdminAllow || currentPriv == rbacutils.OwnerAllow {
		// if currentPriv == rbacutils.AdminAllow {
		return rbacutils.Allow
	}
	return currentPriv
}

func (manager *SPolicyManager) explainPolicy(userCred mcclient.TokenCredential, policyReq jsonutils.JSONObject, name string) ([]string, rbacutils.TRbacResult, error) {
	isAdmin, request, result, err := manager.explainPolicyInternal(userCred, policyReq, name)
	if !isAdmin && isAdminResource(request[0], request[1]) && result == rbacutils.OwnerAllow {
		result = rbacutils.Deny
	}
	result = exportRbacResult(result)
	return request, result, err
}

func (manager *SPolicyManager) explainPolicyInternal(userCred mcclient.TokenCredential, policyReq jsonutils.JSONObject, name string) (bool, []string, rbacutils.TRbacResult, error) {
	policySeq, err := policyReq.GetArray()
	if err != nil {
		return false, nil, rbacutils.Deny, httperrors.NewInputParameterError("invalid format")
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

	isAdmin, _ := policySeq[0].Bool()
	if !consts.IsRbacEnabled() {
		if !isAdmin {
			return isAdmin, reqStrs, rbacutils.OwnerAllow, nil
		} else if isAdmin && userCred.HasSystemAdminPrivelege() {
			return isAdmin, reqStrs, rbacutils.AdminAllow, nil
		} else {
			return isAdmin, reqStrs, rbacutils.Deny, httperrors.NewForbiddenError("operation not allowed")
		}
	}
	if len(name) == 0 {
		return isAdmin, reqStrs, manager.allowWithoutCache(isAdmin, userCred, service, resource, action, extra...), nil
	}
	policy := manager.findPolicyByName(isAdmin, name)
	if policy == nil {
		return isAdmin, reqStrs, rbacutils.Deny, httperrors.NewNotFoundError("policy %s not found", name)
	}
	rule := policy.GetMatchRule(service, resource, action, extra...)
	result := rbacutils.Deny
	if rule != nil {
		result = rule.Result
	}
	return isAdmin, reqStrs, unifyRbacResult(isAdmin, result), nil
}

func (manager *SPolicyManager) ExplainRpc(userCred mcclient.TokenCredential, params jsonutils.JSONObject, name string) (jsonutils.JSONObject, error) {
	paramDict, err := params.GetMap()
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid input format")
	}
	ret := jsonutils.NewDict()
	for key, policyReq := range paramDict {
		reqStrs, result, err := manager.explainPolicy(userCred, policyReq, name)
		if err != nil {
			return nil, err
		}
		reqStrs = append(reqStrs, string(result))
		ret.Add(jsonutils.NewStringArray(reqStrs), key)
	}
	return ret, nil
}

func (manager *SPolicyManager) IsAdminCapable(userCred mcclient.TokenCredential) bool {
	if !consts.IsRbacEnabled() && userCred.HasSystemAdminPrivelege() {
		return true
	}

	userCredJson := userCred.ToJson()
	for _, p := range manager.adminPolicies {
		match, _ := conditionparser.Eval(p.Condition, userCredJson)
		if match {
			return true
		}
	}
	return false
}
