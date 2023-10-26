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
	"yunion.io/x/pkg/util/rbacscope"
)

var sysBasicViewPolicy = `
policy:
  compute:
    zones:
      '*': deny
      list: allow
      get: allow
    cloudregions:
      '*': deny
      list: allow
      get: allow
    cachedimages:
      '*': deny
      list: allow
      get: allow
    dbinstance_skus:
      '*': deny
      list: allow
      get: allow
    elasticcacheskus:
      '*': deny
      list: allow
      get: allow
    serverskus:
      '*': deny
      list: allow
      get: allow
    nat_skus:
      '*': deny
      list: allow
      get: allow
    nas_skus:
      '*': deny
      list: allow
      get: allow
    modelarts_skus:
      '*': deny
      list: allow
      get: allow
    loadbalancerclusters:
      '*': deny
      list: allow
      get: allow
    schedtags:
      '*': deny
      list: allow
      get: allow
    capabilities:
      '*': deny
      list: allow
`

var domainBasicViewPolicy = `
policy:
  compute:
    cloudaccounts:
      '*': deny
      list: allow
      get: allow
    cloudproviders:
      '*': deny
      list: allow
      get: allow
    domain_quotas:
      '*': deny
      list: allow
      get: allow
    infras_quotas:
      '*': deny
      list: allow
      get: allow
    vpcs:
      '*': deny
      list: allow
      get: allow
    wires:
      '*': deny
      list: allow
      get: allow
    proxysettings:
      '*': deny
      list: allow
      get: allow
`

var opsAdminPolicy = `
policy:
  '*':
    events:
      '*': deny
      list:
        '*': deny
        splitable: deny
  compute:
    '*':
      '*': allow
    events:
      '*': deny
      list:
        '*': deny
        splitable: deny
    dynamicschedtags:
      '*': deny
    recyclebins:
      '*': deny
    secgroups:
      '*': deny
      list: allow
      get: allow
    servers:
      '*': allow
      perform:
        '*': allow
        start: deny
        stop: deny
        change-owner: deny
        add-secgroup: deny
        set-secgroup: deny
        revoke-secgroup: deny
        revoke-admin-secgroup: deny
        assign-secgroup: deny
        assign-admin-secgroup: deny
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
    users:
      '*': deny
      create: allow
      list: allow
      get: allow
    events:
      '*': deny
  monitor:
    events:
      '*': deny
    '*':
      '*': allow
  log:
    actions:
      list:
        '*': deny
        splitable: deny
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
      '*': deny
    schedtags:
      '*': deny
    secgroups:
      '*': allow
    secgrouprules:
      '*': allow
    servers:
      '*': deny
      delete: allow
      get:
        '*': allow
        vnc: deny
      list: allow
      perform:
        '*': deny
        change-owner: allow
        add-secgroup: allow
        set-secgroup: allow
        revoke-secgroup: allow
        revoke-admin-secgroup: allow
        assign-secgroup: allow
        assign-admin-secgroup: allow
        purge: allow
  notify:
    events:
      '*': deny
    '*':
      '*': allow
  identity:
    events:
      '*': deny
    users:
      create: deny
      '*': allow
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
    '*':
      '*': deny
      get: allow
      list: allow
      perform:
        '*': deny
        purge-splitable: allow
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
      perform:
        '*': deny
        purge-splitable: allow
  identity:
    '*':
      '*': deny
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
		Scope:         rbacscope.ScopeSystem,
		Policy:        toJson(opsAdminPolicy),
		Description:   "System-wide operation manager",
		DescriptionCN: "全局系统管理员权限",
	},
	{
		Name:          "sys-secadmin",
		Scope:         rbacscope.ScopeSystem,
		Policy:        toJson(secAdminPolicy),
		Description:   "System-wide security manager",
		DescriptionCN: "全局安全管理员权限",
	},
	{
		Name:          "sys-adtadmin",
		Scope:         rbacscope.ScopeSystem,
		Policy:        toJson(adtAdminPolicy),
		Description:   "System-wide audit manager",
		DescriptionCN: "全局审计管理员权限",
	},
	{
		Name:          "sys-basic-viewer",
		Scope:         rbacscope.ScopeSystem,
		Policy:        toJson(sysBasicViewPolicy),
		Description:   "System-wide basic viewer",
		DescriptionCN: "全局基础资源只读权限",
	},
	{
		Name:          "domain-opsadmin",
		Scope:         rbacscope.ScopeDomain,
		Policy:        toJson(opsAdminPolicy),
		Description:   "Domain-wide operation manager",
		DescriptionCN: "组织系统管理员权限",
	},
	{
		Name:          "domain-secadmin",
		Scope:         rbacscope.ScopeDomain,
		Policy:        toJson(secAdminPolicy),
		Description:   "Domain-wide security manager",
		DescriptionCN: "组织安全管理员权限",
	},
	{
		Name:          "domain-adtadmin",
		Scope:         rbacscope.ScopeDomain,
		Policy:        toJson(adtAdminPolicy),
		Description:   "Domain-wide audit manager",
		DescriptionCN: "组织审计管理员权限",
	},
	{
		Name:          "domain-basic-viewer",
		Scope:         rbacscope.ScopeDomain,
		Policy:        toJson(domainBasicViewPolicy),
		Description:   "Domain-wide basic viewer",
		DescriptionCN: "组织基础资源只读权限",
	},
	{
		Name:          "normal-user",
		Scope:         rbacscope.ScopeProject,
		Policy:        toJson(normalUserPolicy),
		Description:   "Default policy for normal user",
		DescriptionCN: "普通用户默认权限",
	},
}
