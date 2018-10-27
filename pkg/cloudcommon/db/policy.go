package db

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"time"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
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

	blobStr, err := obj.GetString("blob")
	if err != nil {
		log.Errorf("get blob error %s", err)
		return "", policy, err
	}
	blob, err := jsonutils.ParseString(blobStr)
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
	s := auth.GetAdminSession(GetGlobalRegion(), "v1")

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

func (manager *SPolicyManager) Allow(isAdmin bool, userCred mcclient.TokenCredential, service string, resource string, action string, extra ...string) bool {
	var policies map[string]rbacutils.SRbacPolicy
	if isAdmin {
		policies = manager.adminPolicies
	} else {
		policies = manager.policies
	}
	if policies == nil {
		log.Warningf("no policies fetched")
		return false
	}
	userCredJson := userCred.ToJson()
	log.Debugf("%s", userCredJson)
	for _, p := range policies {
		if p.Allow(userCredJson, service, resource, action, extra...) {
			return true
		}
	}
	return false
}
