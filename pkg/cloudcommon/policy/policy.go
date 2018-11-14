package policy

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/conditionparser"
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

	PolicyFailedRetryInterval = 15 * time.Second
	PolicyRefreshInterval     = 15 * time.Minute
)

func init() {
	PolicyManager = &SPolicyManager{}
}

type SPolicyManager struct {
	policies      map[string]rbacutils.SRbacPolicy
	adminPolicies map[string]rbacutils.SRbacPolicy
	lastSync      time.Time
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
	PolicyRefreshInterval = refreshInterval
	PolicyFailedRetryInterval = retryInterval
	manager.sync()
}

func (manager *SPolicyManager) sync() {
	log.Debugf("start synchronize RBAC policies ...")
	policies, adminPolicies, err := fetchPolicies()
	if err != nil {
		log.Errorf("sync policy fail %s", err)
		time.AfterFunc(PolicyFailedRetryInterval, manager.sync)
		return
	}
	manager.policies = policies
	manager.adminPolicies = adminPolicies
	manager.lastSync = time.Now()
	time.AfterFunc(PolicyRefreshInterval, manager.sync)
}

func (manager *SPolicyManager) Allow(isAdmin bool, userCred mcclient.TokenCredential, service string, resource string, action string, extra ...string) rbacutils.TRbacResult {
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
		if result.IsHigherPrivilege(currentPriv) {
			currentPriv = result
		}
	}
	if consts.IsRbacDebug() {
		log.Debugf("[RBAC: %v] %s %s %s %#v permission %s", isAdmin, service, resource, action, extra, currentPriv)
	}
	return currentPriv
}

func (manager *SPolicyManager) explainPolicy(userCred mcclient.TokenCredential, policyReq jsonutils.JSONObject) (rbacutils.TRbacResult, error) {
	policySeq, err := policyReq.GetArray()
	if err != nil {
		return rbacutils.Deny, httperrors.NewInputParameterError("invalid format")
	}
	isAdmin, _ := policySeq[0].Bool()
	if !consts.IsRbacEnabled() {
		if !isAdmin || (isAdmin && userCred.IsSystemAdmin()) {
			return rbacutils.Allow, nil
		} else {
			return rbacutils.Deny, httperrors.NewForbiddenError("operation not allowed")
		}
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
			extra[i-4], _ = policySeq[i].GetString()
		}
	}

	return manager.Allow(isAdmin, userCred, service, resource, action, extra...), nil
}

func (manager *SPolicyManager) ExplainRpc(userCred mcclient.TokenCredential, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	paramDict, err := params.GetMap()
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid input format")
	}
	ret := jsonutils.NewDict()
	for key, policyReq := range paramDict {
		result, err := manager.explainPolicy(userCred, policyReq)
		if err != nil {
			return nil, err
		}
		ret.Add(jsonutils.NewString(string(result)), key)
	}
	return ret, nil
}

func (manager *SPolicyManager) IsAdminCapable(userCred mcclient.TokenCredential) bool {
	userCredJson := userCred.ToJson()
	for _, p := range manager.adminPolicies {
		match, _ := conditionparser.Eval(p.Condition, userCredJson)
		if match {
			return true
		}
	}
	return false
}
