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

package modules

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type SRolePolicyManager struct {
	modulebase.ResourceManager
}

var RolePolicies SRolePolicyManager

func (manager *SRolePolicyManager) FetchMatchedPolicies(s *mcclient.ClientSession, roleIds []string, projectId string, loginIp string) (map[string][]string, error) {
	input := api.RolePolicyListInput{}
	input.RoleIds = roleIds
	input.ProjectId = projectId
	limit := 2048
	input.Limit = &limit
	details := true
	input.Details = &details
	results, err := manager.List(s, jsonutils.Marshal(input))
	if err != nil {
		return nil, errors.Wrap(err, "List RolePolicyManager")
	}
	ret := make(map[string][]string)
	for i := range results.Data {
		policy, _ := results.Data[i].GetString("policy")
		scope, _ := results.Data[i].GetString("scope")
		if policies, ok := ret[scope]; !ok {
			ret[scope] = []string{policy}
		} else {
			ret[scope] = append(policies, policy)
		}
	}

	return ret, nil
}

func init() {
	RolePolicies = SRolePolicyManager{NewIdentityV3Manager(
		"rolepolicy",
		"rolepolicies",
		[]string{"id", "name", "role", "role_id", "project", "project_id", "policy", "policy_id", "ips", "scope"},
		[]string{},
	)}

	register(&RolePolicies)
}
