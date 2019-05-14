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

package service

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

func localPolicyFetcher() (map[string]rbacutils.SRbacPolicy, map[string]rbacutils.SRbacPolicy, error) {
	policyList, err := models.PolicyManager.FetchEnabledPolicies()
	if err != nil {
		return nil, nil, err
	}

	policies := make(map[string]rbacutils.SRbacPolicy)
	adminPolicies := make(map[string]rbacutils.SRbacPolicy)

	for i := range policyList {
		typeStr := policyList[i].Name
		policy := rbacutils.SRbacPolicy{}
		policyStr, err := policyList[i].Blob.GetString()
		if err != nil {
			log.Errorf("fail to get string of blob %s", err)
			continue
		}
		policyJson, err := jsonutils.ParseString(policyStr)
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
