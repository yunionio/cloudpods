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
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	common_policy "yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

const (
	PolicyActionGet     = common_policy.PolicyActionGet
	PolicyActionList    = common_policy.PolicyActionList
	PolicyActionCreate  = common_policy.PolicyActionCreate
	PolicyActionUpdate  = common_policy.PolicyActionUpdate
	PolicyActionDelete  = common_policy.PolicyActionDelete
	PolicyActionPerform = common_policy.PolicyActionPerform
)

var (
	predefinedDefaultPolicies = []rbacutils.SRbacPolicy{
		{
			Auth:  true,
			Scope: rbacscope.ScopeUser,
			Rules: []rbacutils.SRbacRule{
				{
					Service:  api.SERVICE_TYPE,
					Resource: "credentials",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "credentials",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "credentials",
					Action:   PolicyActionCreate,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "credentials",
					Action:   PolicyActionUpdate,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "credentials",
					Action:   PolicyActionDelete,
					Result:   rbacutils.Allow,
				},
			},
		},
		{
			Auth:  true,
			Scope: rbacscope.ScopeProject,
			Rules: []rbacutils.SRbacRule{
				{
					Service:  api.SERVICE_TYPE,
					Resource: "users",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "groups",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
			},
		},
		{
			// for domain
			Auth:  true,
			Scope: rbacscope.ScopeDomain,
			Rules: []rbacutils.SRbacRule{
				{
					Service:  api.SERVICE_TYPE,
					Resource: "domains",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
			},
		},
		{
			// for policies administration
			Auth:     true,
			Scope:    rbacscope.ScopeSystem,
			DomainId: api.DEFAULT_DOMAIN_ID,
			Projects: []string{api.SystemAdminProject},
			Roles:    []string{api.SystemAdminRole},
			Rules: []rbacutils.SRbacRule{
				{
					Service:  api.SERVICE_TYPE,
					Resource: "policies",
					Action:   PolicyActionCreate,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "policies",
					Action:   PolicyActionUpdate,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "policies",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "policies",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "policies",
					Action:   PolicyActionPerform,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "roles",
					Action:   PolicyActionCreate,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "roles",
					Action:   PolicyActionUpdate,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "roles",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "roles",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "roles",
					Action:   PolicyActionPerform,
					Result:   rbacutils.Allow,
				},
			},
		},
	}
)

func Init() {
	if consts.IsEnableDefaultPolicy() {
		common_policy.AppendDefaultPolicies(predefinedDefaultPolicies)
	}
}
