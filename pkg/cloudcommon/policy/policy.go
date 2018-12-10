package policy

import (
	"time"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/conditionparser"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/hashcache"
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

	defaultRules = []rbacutils.SRbacRule{
		{
			Resource: "tasks",
			Action:   PolicyActionPerform,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "zones",
			Action:   PolicyActionList,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "zones",
			Action:   PolicyActionGet,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "storages",
			Action:   PolicyActionList,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "storages",
			Action:   PolicyActionGet,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "schedtags",
			Action:   PolicyActionList,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "schedtags",
			Action:   PolicyActionGet,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "cloudregions",
			Action:   PolicyActionList,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "cloudregions",
			Action:   PolicyActionGet,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "quotas",
			Action:   PolicyActionGet,
			Result:   rbacutils.OwnerAllow,
		},
		{
			Service:  "compute",
			Resource: "usages",
			Action:   PolicyActionGet,
			Result:   rbacutils.OwnerAllow,
		},
		{
			Service:  "yunionagent",
			Resource: "notices",
			Action:   PolicyActionList,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "yunionagent",
			Resource: "readmarks",
			Action:   PolicyActionList,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "yunionagent",
			Resource: "readmarks",
			Action:   PolicyActionCreate,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "yunionconf",
			Resource: "parameters",
			Action:   PolicyActionGet,
			Result:   rbacutils.OwnerAllow,
		},
	}
)

func init() {
	PolicyManager = &SPolicyManager{}
}

type SPolicyManager struct {
	policies      map[string]rbacutils.SRbacPolicy
	adminPolicies map[string]rbacutils.SRbacPolicy
	defaultPolicy *rbacutils.SRbacPolicy
	lastSync      time.Time

	failedRetryInterval time.Duration
	refreshInterval     time.Duration

	cache *hashcache.Cache // policy cache
}

func parseJsonPolicy(obj jsonutils.JSONObject) (string, rbacutils.SRbacPolicy, error) {
	policy := rbacutils.SRbacPolicy{}
	typeStr, err := obj.GetString("type")
	if err != nil {
		log.Errorf("get type error %s", err)
		return "", policy, err
	}

	blobStr, err := obj.GetString("policy")
	if err != nil {
		log.Errorf("get blob error %s", err)
		return "", policy, err
	}
	blob, err := jsonutils.ParseYAML(blobStr)
	if err != nil {
		log.Errorf("parse blob json error %s", err)
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
	s := auth.GetAdminSession(consts.GetRegion(), "v1")

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

	manager.cache = hashcache.NewCache(2048, time.Second*5)
	manager.sync()
}

func (manager *SPolicyManager) sync() {
	policies, adminPolicies, err := fetchPolicies()
	if err != nil {
		log.Errorf("sync rbac policy failed: %s", err)
		time.AfterFunc(manager.failedRetryInterval, manager.sync)
		return
	}
	manager.policies = policies
	manager.adminPolicies = adminPolicies

	manager.lastSync = time.Now()
	time.AfterFunc(manager.refreshInterval, manager.sync)
}

func (manager *SPolicyManager) Allow(isAdmin bool, userCred mcclient.TokenCredential, service string, resource string, action string, extra ...string) rbacutils.TRbacResult {
	if manager.cache != nil {
		isAdminStr := fmt.Sprintf("%v", isAdmin)
		queryKeys := []string{isAdminStr, userCred.String()}
		if rbacutils.WILD_MATCH == service || len(service) == 0 {
			queryKeys = append(queryKeys, rbacutils.WILD_MATCH)
		}
		if rbacutils.WILD_MATCH == resource || len(resource) == 0 {
			queryKeys = append(queryKeys, rbacutils.WILD_MATCH)
		}
		if rbacutils.WILD_MATCH == action || len(action) == 0 {
			queryKeys = append(queryKeys, rbacutils.WILD_MATCH)
		}
		if len(extra) > 0 {
			queryKeys = append(queryKeys, extra...)
		}
		key := strings.Join(queryKeys, "-")
		log.Debugf("%s", key)
		val := manager.cache.Get(key)
		if val != nil {
			log.Debugf("cache hit")
			return val.(rbacutils.TRbacResult)
		}
		log.Debugf("cache miss!!")
		result := manager.allowWithoutCache(isAdmin, userCred, service, resource, action, extra...)
		manager.cache.Set(key, result)
		return result
	} else {
		return manager.allowWithoutCache(isAdmin, userCred, service, resource, action, extra...)
	}
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

func (manager *SPolicyManager) explainPolicy(userCred mcclient.TokenCredential, policyReq jsonutils.JSONObject) ([]string, rbacutils.TRbacResult, error) {
	policySeq, err := policyReq.GetArray()
	if err != nil {
		return nil, rbacutils.Deny, httperrors.NewInputParameterError("invalid format")
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
			return reqStrs, rbacutils.OwnerAllow, nil
		} else if isAdmin && userCred.HasSystemAdminPrivelege() {
			return reqStrs, rbacutils.AdminAllow, nil
		} else {
			return reqStrs, rbacutils.Deny, httperrors.NewForbiddenError("operation not allowed")
		}
	}
	return reqStrs, manager.Allow(isAdmin, userCred, service, resource, action, extra...), nil
}

func (manager *SPolicyManager) ExplainRpc(userCred mcclient.TokenCredential, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	paramDict, err := params.GetMap()
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid input format")
	}
	ret := jsonutils.NewDict()
	for key, policyReq := range paramDict {
		reqStrs, result, err := manager.explainPolicy(userCred, policyReq)
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
