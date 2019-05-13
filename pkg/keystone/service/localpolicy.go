package service

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/pkg/utils"
)

func localPolicyFetcher() (map[string]rbacutils.SRbacPolicy, map[string]rbacutils.SRbacPolicy, error) {
	policyList, err := models.PolicyManager.FetchEnabledPolicies()
	if err != nil {
		return nil, nil, err
	}

	policies := make(map[string]rbacutils.SRbacPolicy)
	adminPolicies := make(map[string]rbacutils.SRbacPolicy)

	for i := range policyList {
		log.Debugf("BLOB: %#v", policyList[i].Blob)
		typeStr := policyList[i].Name
		policy := rbacutils.SRbacPolicy{}
		policyJson, err := jsonutils.ParseString(utils.Unquote(policyList[i].Blob))
		if err != nil {
			log.Errorf("fail to deocde policy blob into JSON %s", err)
			continue
		}
		err = policy.Decode(policyJson)
		if err != nil {
			log.Errorf("fail to decode policy %s %s", typeStr, policyList[i].Blob, err)
			continue
		}

		if policy.IsAdmin {
			adminPolicies[typeStr] = policy
		} else {
			policies[typeStr] = policy
		}
	}

	return policies, adminPolicies, nil
}
