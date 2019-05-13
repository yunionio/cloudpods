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
	defaultPolicies = []rbacutils.SRbacPolicy{
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
					Resource: "schedtags",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "compute",
					Resource: "schedtags",
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
					// usages for any services
					// Service:  "compute",
					Resource: "usages",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  "yunionconf",
					Resource: "parameters",
					Action:   PolicyActionGet,
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
			// for policies
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
					Action:   PolicyActionList,
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
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
			},
		},
		{
			Service:  "identity",
			Resource: "policies",
			Action:   PolicyActionList,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "identity",
			Resource: "policies",
			Action:   PolicyActionGet,
			Result:   rbacutils.UserAllow,
		},
	}
)
