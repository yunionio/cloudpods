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

import "yunion.io/x/onecloud/pkg/util/rbacutils"

var (
	defaultRules = []rbacutils.SRbacRule{
		{
			Resource: "tasks",
			Action:   PolicyActionPerform,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "hosts",
			Action:   PolicyActionList,
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
			Resource: "metadatas",
			Action:   PolicyActionList,
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
			Resource: "cachedimages",
			Action:   PolicyActionList,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "cachedimages",
			Action:   PolicyActionGet,
			Result:   rbacutils.UserAllow,
		},
		{
			// quotas for any services
			// Service:  "compute",
			Resource: "quotas",
			Action:   PolicyActionGet,
			Result:   rbacutils.OwnerAllow,
		},
		{
			// usages for any services
			// Service:  "compute",
			Resource: "usages",
			Action:   PolicyActionGet,
			Result:   rbacutils.OwnerAllow,
		},
		{
			Service:  "compute",
			Resource: "serverskus",
			Action:   PolicyActionList,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "serverskus",
			Action:   PolicyActionGet,
			Result:   rbacutils.UserAllow,
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
		{
			Service:  "image",
			Resource: "images",
			Action:   PolicyActionList,
			Result:   rbacutils.OwnerAllow,
		},
		{
			Service:  "image",
			Resource: "images",
			Action:   PolicyActionGet,
			Result:   rbacutils.OwnerAllow,
		},
		{
			Service:  "image",
			Resource: "images",
			Action:   PolicyActionPerform,
			Extra:    []string{"update-torrent-status"},
			Result:   rbacutils.GuestAllow,
		},
		{
			Service:  "log",
			Resource: "actions",
			Action:   PolicyActionList,
			Result:   rbacutils.OwnerAllow,
		},
		{
			Service:  "log",
			Resource: "actions",
			Action:   PolicyActionGet,
			Result:   rbacutils.OwnerAllow,
		},
	}
)
