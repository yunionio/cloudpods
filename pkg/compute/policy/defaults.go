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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	common_policy "yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

const (
	PolicyActionPerform = common_policy.PolicyActionPerform
	PolicyActionList    = common_policy.PolicyActionList
	PolicyActionGet     = common_policy.PolicyActionGet
	PolicyActionCreate  = common_policy.PolicyActionCreate
	PolicyActionUpdate  = common_policy.PolicyActionUpdate
	PolicyActionDelete  = common_policy.PolicyActionDelete
)

var (
	predefinedDefaultPolicies = []rbacutils.SRbacPolicy{
		{
			Auth:  true,
			Scope: rbacscope.ScopeSystem,
			Rules: []rbacutils.SRbacRule{
				{
					Service:  api.SERVICE_TYPE,
					Resource: "zones",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "zones",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "cloudregions",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "cloudregions",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "cachedimages",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "cachedimages",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "dbinstance_skus",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "dbinstance_skus",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "serverskus",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "serverskus",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "secgrouprules",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "elasticcacheskus",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "elasticcacheskus",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "secgrouprules",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "loadbalancerclusters",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "schedtags",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "dns_recordsets",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "dns_recodsets",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "cloudaccounts",
					Action:   PolicyActionGet,
					Extra:    []string{"saml"},
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "waf_rules",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "waf_rules",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "capabilities",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
			},
		},
		{
			Auth:  true,
			Scope: rbacscope.ScopeUser,
			Rules: []rbacutils.SRbacRule{
				{
					Service:  api.SERVICE_TYPE,
					Resource: "keypairs",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "keypairs",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "keypairs",
					Action:   PolicyActionCreate,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "keypairs",
					Action:   PolicyActionUpdate,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "keypairs",
					Action:   PolicyActionDelete,
					Result:   rbacutils.Allow,
				},
			},
		},
		{
			Auth:  true,
			Scope: rbacscope.ScopeDomain,
			Rules: []rbacutils.SRbacRule{
				{
					Service:  api.SERVICE_TYPE,
					Resource: "cloudaccounts",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "cloudaccounts",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "cloudproviders",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "cloudproviders",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "domain_quotas",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "domain_quotas",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "infras_quotas",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "infras_quotas",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "vpcs",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "vpcs",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "wires",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "wires",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "proxysettings",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "proxysettings",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "nat_skus",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "nat_skus",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "nas_skus",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "nas_skus",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "modelarts_skus",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "modelarts_skus",
					Action:   PolicyActionGet,
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
					Resource: "quotas",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "quotas",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "region_quotas",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "region_quotas",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "zone_quotas",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "zone_quotas",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "project_quotas",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "project_quotas",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "networks",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "networks",
					Action:   PolicyActionList,
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
