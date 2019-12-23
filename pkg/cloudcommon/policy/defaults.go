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
	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

var (
	predefinedDefaultPolicies = []rbacutils.SRbacPolicy{
		{
			Auth:  true,
			Scope: rbacutils.ScopeSystem,
			Rules: []rbacutils.SRbacRule{
				{
					Resource: "tasks",
					Action:   PolicyActionPerform,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "hosts",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "zones",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "zones",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "metadatas",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "storages",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "storages",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "vpcs",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "vpcs",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "wires",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "wires",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "cloudregions",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "cloudregions",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "cachedimages",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "cachedimages",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "dbinstance_skus",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "dbinstance_skus",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "serverskus",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "serverskus",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "secgrouprules",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "elasticcacheskus",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "elasticcacheskus",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "secgrouprules",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "yunionagent",
					Resource: "notices",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "yunionagent",
					Resource: "readmarks",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "yunionagent",
					Resource: "readmarks",
					Action:   PolicyActionCreate,
					Result:   rbacutils.Allow,
				},
			},
		},
		{
			Auth:  true,
			Scope: rbacutils.ScopeUser,
			Rules: []rbacutils.SRbacRule{
				{
					Service:  "compute",
					Resource: "keypairs",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "keypairs",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "keypairs",
					Action:   PolicyActionCreate,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "keypairs",
					Action:   PolicyActionUpdate,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "keypairs",
					Action:   PolicyActionDelete,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "identity",
					Resource: "credentials",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "identity",
					Resource: "credentials",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "identity",
					Resource: "credentials",
					Action:   PolicyActionCreate,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "identity",
					Resource: "credentials",
					Action:   PolicyActionUpdate,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "identity",
					Resource: "credentials",
					Action:   PolicyActionDelete,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "yunionconf",
					Resource: "parameters",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "yunionconf",
					Resource: "parameters",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "yunionconf",
					Resource: "parameters",
					Action:   PolicyActionCreate,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "yunionconf",
					Resource: "parameters",
					Action:   PolicyActionUpdate,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "notify",
					Resource: "contacts",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "notify",
					Resource: "contacts",
					Action:   PolicyActionCreate,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "notify",
					Resource: "contacts",
					Action:   PolicyActionUpdate,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "notify",
					Resource: "contacts",
					Action:   PolicyActionDelete,
					Result:   rbacutils.Allow,
				},
			},
		},
		{
			Auth:  true,
			Scope: rbacutils.ScopeProject,
			Rules: []rbacutils.SRbacRule{
				{
					Resource: "tasks",
					Action:   PolicyActionPerform,
					Result:   rbacutils.Allow,
				},
				{
					// quotas for any services
					// Service:  "compute",
					Resource: "quotas",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					// quotas for any services
					// Service:  "compute",
					Resource: "quotas",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					// usages for any services
					// Service:  "compute",
					Resource: "usages",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "networks",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "networks",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "image",
					Resource: "images",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "image",
					Resource: "images",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "image",
					Resource: "guestimages",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "image",
					Resource: "guestimages",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "log",
					Resource: "actions",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "log",
					Resource: "actions",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "cloudproviders",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "cloudproviders",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
			},
		},
		{
			// meta服务 dbinstance_skus列表不需要认证
			Auth:  false,
			Scope: rbacutils.ScopeSystem,
			Rules: []rbacutils.SRbacRule{
				{
					Service:  "yunionmeta",
					Resource: "dbinstance_skus",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "yunionmeta",
					Resource: "dbinstance_skus",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
			},
		},
		{
			// for anonymous update torrent status
			Auth:  false,
			Scope: rbacutils.ScopeSystem,
			Rules: []rbacutils.SRbacRule{
				{
					Service:  "image",
					Resource: "images",
					Action:   PolicyActionPerform,
					Extra:    []string{"update-torrent-status"},
					Result:   rbacutils.Allow,
				},
			},
		},
		{
			// for domain
			Auth:  true,
			Scope: rbacutils.ScopeDomain,
			Rules: []rbacutils.SRbacRule{
				{
					// quotas for any services
					// Service:  "compute",
					Resource: "quotas",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					// quotas for any services
					// Service:  "compute",
					Resource: "quotas",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					// usages for any services
					// Service:  "compute",
					Resource: "usages",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  identityapi.SERVICE_TYPE,
					Resource: "domains",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  identityapi.SERVICE_TYPE,
					Resource: "services",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  identityapi.SERVICE_TYPE,
					Resource: "services",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
			},
		},
		{
			// for policies administration
			Auth:     true,
			Scope:    rbacutils.ScopeSystem,
			DomainId: identityapi.DEFAULT_DOMAIN_ID,
			Projects: []string{identityapi.SystemAdminProject},
			Roles:    []string{identityapi.SystemAdminRole},
			Rules: []rbacutils.SRbacRule{
				{
					Service:  identityapi.SERVICE_TYPE,
					Resource: "policies",
					Action:   PolicyActionCreate,
					Result:   rbacutils.Allow,
				},
				{
					Service:  identityapi.SERVICE_TYPE,
					Resource: "policies",
					Action:   PolicyActionUpdate,
					Result:   rbacutils.Allow,
				},
				{
					Service:  identityapi.SERVICE_TYPE,
					Resource: "policies",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  identityapi.SERVICE_TYPE,
					Resource: "policies",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
			},
		},
	}
)
