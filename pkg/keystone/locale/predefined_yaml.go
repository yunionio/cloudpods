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

package locale

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

var opsAdminPolicy = `
policy:
  '*':
    events:
      '*': deny
  compute:
    '*':
      '*': allow
    events:
      '*': deny
    dynamicschedtags:
      '*': deny
    recyclebins:
      '*': deny
    schedpolicies:
      '*': deny
    schedtags:
      '*': deny
    secgroups:
      '*': deny
    servers:
      '*': allow
      perform:
        '*': allow
        start: deny
        stop: deny
        change-owner: deny
  image:
    '*':
      '*': allow
    events:
      '*': deny
  identity:
    '*':
      '*': deny
      list: allow
      get: allow
    events:
      '*': deny
  log:
    actions:
      list: deny
      get: deny
`

var secAdminPolicy = `
policy:
  '*':
    events:
      '*': deny
  compute:
    events:
      '*': deny
    '*':
      '*': deny
      get: allow
      list: allow
    disks:
      '*': deny
      delete: allow
      get: allow
      list: allow
      perform:
        '*': deny
        change-owner: allow
        purge: allow
    dynamicschedtags:
      '*': allow
    recyclebins:
      '*': allow
      get: allow
      list: allow
    schedpolicies:
      '*': allow
    schedtags:
      '*': allow
    secgroups:
      '*': allow
    servers:
      '*': deny
      delete: allow
      get: allow
      list: allow
      perform:
        '*': deny
        change-owner: allow
        purge: allow
  monitor:
    events:
      '*': deny
    '*':
      '*': allow
  identity:
    events:
      '*': deny
    '*':
      '*': allow
  image:
    events:
      '*': deny
    '*':
      '*': deny
      delete: allow
      get: allow
      list: allow
      perform:
        '*': deny
        change-owner: allow
        purge: allow
  log:
    actions:
      list: deny
      get: deny
  yunionconf:
    '*':
      '*': allow
`

var adtAdminPolicy = `
policy:
  '*':
    '*':
      '*': deny
    events:
      '*': allow
  log:
    '*':
      '*': deny
      get: allow
      list: allow
  identity:
    '*':
      '*': deny
      get: allow
      list: allow
`

var normalUserPolicy = `
policy:
  compute:
    '*':
      '*': deny
      list: allow
      get: allow
    servers:
      '*': allow
      delete: deny
      perform:
        clone: deny
        snapshot-and-clone: deny
        purge: deny
        change-ipaddr: deny
        change-bandwidth: deny
        change-config: deny
        change-owner: deny
        change-disk-storage: deny
  image:
    images:
      '*': deny
      list: allow
      get: allow
`

func toJson(yamlDef string) jsonutils.JSONObject {
	yaml, err := jsonutils.ParseYAML(yamlDef)
	if err != nil {
		log.Errorf("fail to parse %s: %s", yamlDef, err)
	}
	return yaml
}

var predefinedPolicyData = []SPolicyData{
	{
		Name:          "sys-opsadmin",
		Scope:         rbacutils.ScopeSystem,
		Policy:        toJson(opsAdminPolicy),
		Description:   "System-wide operation manager",
		DescriptionCN: "全局系统管理员权限",
	},
	{
		Name:          "sys-secadmin",
		Scope:         rbacutils.ScopeSystem,
		Policy:        toJson(secAdminPolicy),
		Description:   "System-wide security manager",
		DescriptionCN: "全局安全管理员权限",
	},
	{
		Name:          "sys-adtadmin",
		Scope:         rbacutils.ScopeSystem,
		Policy:        toJson(adtAdminPolicy),
		Description:   "System-wide audit manager",
		DescriptionCN: "全局审计管理员权限",
	},
	{
		Name:          "domain-opsadmin",
		Scope:         rbacutils.ScopeDomain,
		Policy:        toJson(opsAdminPolicy),
		Description:   "Domain-wide operation manager",
		DescriptionCN: "组织系统管理员权限",
	},
	{
		Name:          "domain-secadmin",
		Scope:         rbacutils.ScopeDomain,
		Policy:        toJson(secAdminPolicy),
		Description:   "Domain-wide security manager",
		DescriptionCN: "组织安全管理员权限",
	},
	{
		Name:          "domain-adtadmin",
		Scope:         rbacutils.ScopeDomain,
		Policy:        toJson(adtAdminPolicy),
		Description:   "Domain-wide audit manager",
		DescriptionCN: "组织审计管理员权限",
	},
	{
		Name:          "normal-user",
		Scope:         rbacutils.ScopeProject,
		Policy:        toJson(normalUserPolicy),
		Description:   "Default policy for normal user",
		DescriptionCN: "普通用户默认权限",
	},
}
