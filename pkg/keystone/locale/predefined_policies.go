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
	"yunion.io/x/pkg/util/rbacscope"
)

const (
	RoleAdmin         = "admin"
	RoleFA            = "fa"
	RoleDomainFA      = "domainfa"
	RoleProjectFA     = "projectfa"
	RoleSA            = "sa"
	RoleProjectOwner  = "project_owner"
	RoleDomainAdmin   = "domainadmin"
	RoleDomainEditor  = "domain_editor"
	RoleDomainViewer  = "domain_viewer"
	RoleProjectEditor = "project_editor"
	RoleProjectViewer = "project_viewer"

	RoleMember = "member"
)

type sPolicyDefinition struct {
	Name     string
	DescCN   string
	Desc     string
	Scope    rbacscope.TRbacScope
	Services map[string][]string
	Extra    map[string]map[string][]string

	AvailableRoles []string
}

type SRoleDefiniton struct {
	Name        string
	Description string
	Policies    []string
	Project     string
	IsPublic    bool

	DescriptionCN string
}

var (
	policyDefinitons = []sPolicyDefinition{
		{
			Name:   "",
			DescCN: "任意资源",
			Desc:   "any resources",
			Scope:  rbacscope.ScopeProject,
			Services: map[string][]string{
				"*": nil,
			},
		},
		{
			Name:   "dashboard",
			DescCN: "控制面板查看相关资源",
			Desc:   "resources for viewing dashboard",
			Scope:  rbacscope.ScopeProject,
			Extra: map[string]map[string][]string{
				"compute": {
					"dashboard": {
						"get",
					},
					"capabilities": {
						"list",
					},
					"usages": {
						"list",
						"get",
					},
					"quotas": {
						"list",
						"get",
					},
					"zone_quotas": {
						"list",
						"get",
					},
					"region_quotas": {
						"list",
						"get",
					},
					"project_quotas": {
						"list",
						"get",
					},
					"domain_quotas": {
						"list",
						"get",
					},
					"infras_quotas": {
						"list",
						"get",
					},
				},
				"image": {
					"usages": {
						"list",
						"get",
					},
					"image_quotas": {
						"list",
						"get",
					},
				},
				"identity": {
					"usages": {
						"list",
						"get",
					},
					"identity_quotas": {
						"list",
						"get",
					},
					"projects": {
						"list",
					},
				},
				"meter": {
					"bill_conditions": {
						"list",
					},
				},
				"monitor": {
					"alertrecords": {
						"list",
					},
					"alertresources": {
						"list",
					},
					"unifiedmonitors": {
						"perform",
					},
					"monitorresourcealerts": {
						"list",
						"get",
					},
					"nodealerts": {
						"list",
					},
				},
				"notify": {
					"notifications": {
						"list",
						"get",
					},
					"robots": {
						"list",
						"get",
					},
					"receivers": {
						"list",
						"get",
					},
				},
				"devtool": {
					"scriptapplyrecords": {
						"list",
						"get",
					},
				},
				"yunionconf": {
					"scopedpolicybindings": {
						"list",
						"get",
					},
				},
				"suggestion": {
					"suggestsysalerts": {
						"list",
						"get",
					},
				},
			},
		},
		{
			Name:   "compute",
			DescCN: "计算服务(云主机与容器)相关资源",
			Desc:   "resources of computing (cloud servers and containers)",
			Scope:  rbacscope.ScopeProject,
			Services: map[string][]string{
				"compute": nil,
				"image":   nil,
				"k8s":     nil,
			},
		},
		{
			Name:   "server",
			DescCN: "云主机相关资源",
			Desc:   "resources of cloud servers",
			Scope:  rbacscope.ScopeProject,
			Services: map[string][]string{
				"compute": {
					"servers",
					"servertemplates",
					"instancegroups",
					"scalinggroups",
					"scalingactivities",
					"scalingpolicies",
					"disks",
					"networks",
					"eips",
					"snapshotpolicies",
					"snapshotpolicycaches",
					"snapshotpolicydisks",
					"snapshots",
					"instance_snapshots",
					"snapshotpolicies",
					"secgroupcaches",
					"secgrouprules",
					"secgroups",
				},
				"image": nil,
			},
			Extra: map[string]map[string][]string{
				"compute": {
					"isolated_devices": {
						"get",
						"list",
					},
				},
			},
		},
		{
			Name:   "host",
			DescCN: "宿主机和物理机相关资源",
			Desc:   "resources of hosts and baremetals",
			Scope:  rbacscope.ScopeDomain,
			Services: map[string][]string{
				"compute": {
					"hosts",
					"isolated_devices",
					"hostwires",
					"hoststorages",
					"baremetalagents",
					"baremetalnetworks",
					"baremetalevents",
				},
			},
		},
		{
			Name:   "storage",
			DescCN: "云硬盘存储相关资源",
			Desc:   "resources of cloud disk storages",
			Scope:  rbacscope.ScopeDomain,
			Services: map[string][]string{
				"compute": {
					"storages",
				},
			},
		},
		{
			Name:   "loadbalancer",
			DescCN: "负载均衡相关资源",
			Desc:   "resources of load balancers",
			Scope:  rbacscope.ScopeProject,
			Services: map[string][]string{
				"compute": {
					"loadbalanceracls",
					"loadbalanceragents",
					"loadbalancerbackendgroups",
					"loadbalancerbackends",
					"loadbalancercertificates",
					"loadbalancerclusters",
					"loadbalancerlistenerrules",
					"loadbalancerlisteners",
					"loadbalancernetworks",
					"loadbalancers",
				},
			},
			Extra: map[string]map[string][]string{
				"compute": {
					"networks": {
						"get",
						"list",
					},
				},
			},
		},
		{
			Name:   "oss",
			DescCN: "对象存储相关资源",
			Desc:   "resources of object storages",
			Scope:  rbacscope.ScopeProject,
			Services: map[string][]string{
				"compute": {
					"buckets",
				},
			},
		},
		{
			Name:   "dbinstance",
			DescCN: "关系型数据库(MySQL等)相关资源",
			Desc:   "resources of RDS",
			Scope:  rbacscope.ScopeProject,
			Services: map[string][]string{
				"compute": {
					"dbinstance_skus",
					"dbinstanceaccounts",
					"dbinstancebackups",
					"dbinstancedatabases",
					"dbinstancenetworks",
					"dbinstanceparameters",
					"dbinstanceprivileges",
					"dbinstances",
				},
			},
		},
		{
			Name:   "elasticcache",
			DescCN: "弹性缓存(Redis等)相关资源",
			Desc:   "resources of elastic caches",
			Scope:  rbacscope.ScopeProject,
			Services: map[string][]string{
				"compute": {
					"elasticcacheaccounts",
					"elasticcacheacls",
					"elasticcachebackups",
					"elasticcacheparameters",
					"elasticcaches",
					"elasticcacheskus",
				},
			},
		},
		{
			Name:   "network",
			DescCN: "网络相关资源",
			Desc:   "resources of networking",
			Scope:  rbacscope.ScopeDomain,
			Services: map[string][]string{
				"compute": {
					"vpcs",
					"wires",
					"natdentries",
					"natgateways",
					"natsentries",
					"networkinterfacenetworks",
					"networkinterfaces",
					"networks",
					"reservedips",
					"route_tables",
					"globalvpcs",
					"vpc_peering_connections",
					"eips",
					"dns_recordsets",
					"dns_trafficpolicies",
					"dns_zonecaches",
					"dns_zones",
					"dnsrecords",
				},
			},
		},
		{
			Name:   "snapshotpolicy",
			DescCN: "快照策略",
			Desc:   "snapshot policy",
			Scope:  rbacscope.ScopeDomain,
			Services: map[string][]string{
				"compute": {
					"snapshotpolicies",
					"snapshotpolicydisks",
				},
			},
		},
		{
			Name:   "secgroup",
			DescCN: "安全组",
			Desc:   "security group",
			Scope:  rbacscope.ScopeDomain,
			Services: map[string][]string{
				"compute": {
					"secgroups",
					"secgrouprules",
				},
			},
		},
		{
			Name:   "meter",
			DescCN: "计费计量分析服务相关资源",
			Desc:   "resources of metering and billing service",
			Scope:  rbacscope.ScopeProject,
			Services: map[string][]string{
				"meter":      nil,
				"suggestion": nil,
				"notify": {
					"receivers",
				},
			},
		},
		{
			Name:   "identity",
			DescCN: "身份认证(IAM)服务相关资源",
			Desc:   "resources of identity service",
			Scope:  rbacscope.ScopeDomain,
			Services: map[string][]string{
				"identity": nil,
			},
		},
		{
			Name:   "image",
			DescCN: "镜像服务相关资源",
			Desc:   "resources of image service",
			Scope:  rbacscope.ScopeProject,
			Services: map[string][]string{
				"image": nil,
			},
		},
		{
			Name:   "monitor",
			DescCN: "监控服务相关资源",
			Desc:   "resources of monitor service",
			Scope:  rbacscope.ScopeProject,
			Services: map[string][]string{
				"monitor": nil,
			},
		},
		{
			Name:   "container",
			DescCN: "容器服务相关资源",
			Desc:   "resources of container service",
			Scope:  rbacscope.ScopeProject,
			Services: map[string][]string{
				"k8s": nil,
			},
		},
		{
			Name:   "cloudid",
			DescCN: "云用户及权限管理相关资源",
			Desc:   "resources of service CloudId and IAM",
			Scope:  rbacscope.ScopeDomain,
			Services: map[string][]string{
				"compute": {
					"cloudaccounts",
					"cloudproviders",
				},
				"identity": {
					"users",
					"projects",
					"roles",
				},
				"cloudid": nil,
			},
		},
		{
			Name:   "cloudaccount",
			DescCN: "云账号管理相关资源",
			Desc:   "resources for cloud account administration",
			Scope:  rbacscope.ScopeDomain,
			Services: map[string][]string{
				"compute": {
					"cloudaccounts",
					"cloudproviderquotas",
					"cloudproviderregions",
					"cloudproviders",
				},
			},
		},
		{
			Name:   "projectresource",
			DescCN: "项目管理相关资源",
			Desc:   "resources for project administration",
			Scope:  rbacscope.ScopeDomain,
			Services: map[string][]string{
				"compute": {
					"project_quotas",
					"quotas",
					"region_quotas",
					"zone_quotas",
				},
				"image": {
					"image_quotas",
				},
				"identity": {
					"projects",
					"roles",
					"policies",
				},
			},
		},
		{
			Name:   "domainresource",
			DescCN: "域管理相关资源",
			Desc:   "resources for domain administration",
			Scope:  rbacscope.ScopeSystem,
			Services: map[string][]string{
				"compute": {
					"domain_quotas",
					"infras_quotas",
				},
				"identity": {
					"domains",
					"identity_quotas",
					"projects",
					"roles",
					"policies",
					"users",
					"groups",
				},
			},
		},
		{
			Name:   "notify",
			DescCN: "通知服务相关资源",
			Desc:   "resources of notify service",
			Scope:  rbacscope.ScopeSystem,
			Services: map[string][]string{
				"notify": nil,
			},
		},
		{
			Name:   "log",
			DescCN: "日志服务相关资源",
			Desc:   "resources of logger service",
			Scope:  rbacscope.ScopeSystem,
			Services: map[string][]string{
				"log": nil,
			},
			AvailableRoles: []string{
				"viewer",
			},
		},
	}

	adminPerformActions = map[string]map[string][]string{
		"compute": map[string][]string{
			"servers": []string{
				"snapshot-and-clone",
				"createdisk",
				"create-eip",
				"create-backup",
				"save-image",
				"delete-disk",
				"delete-eip",
				"delete-backup",
			},
			"buckets": []string{
				"upload",
				"delete",
			},
		},
		"k8s": map[string][]string{
			"kubeclusters": []string{
				"add-machines",
				"delete-machines",
			},
		},
	}

	RoleDefinitions = []SRoleDefiniton{
		{
			Name:          RoleAdmin,
			DescriptionCN: "系统管理员",
			Description:   "System administrator",
			Policies: []string{
				"sysadmin",
			},
			Project:  "system",
			IsPublic: false,
		},
		{
			Name:          RoleDomainAdmin,
			DescriptionCN: "域管理员",
			Description:   "Domain administrator",
			Policies: []string{
				"domain-admin",
			},
			IsPublic: true,
		},
		{
			Name:          RoleProjectOwner,
			DescriptionCN: "项目主管",
			Description:   "Project owner",
			Policies: []string{
				"project-admin",
			},
			IsPublic: true,
		},
		{
			Name:          RoleFA,
			DescriptionCN: "系统财务管理员",
			Description:   "System finance administrator",
			Policies: []string{
				"sys-meter-admin",
				"sys-dashboard",
			},
			IsPublic: false,
		},
		{
			Name:          RoleDomainFA,
			DescriptionCN: "域财务管理员",
			Description:   "Domain finance administrator",
			Policies: []string{
				"domain-meter-admin",
				"domain-dashboard",
			},
			IsPublic: true,
		},
		{
			Name:          RoleProjectFA,
			DescriptionCN: "项目财务管理员",
			Description:   "Project finance administrator",
			Policies: []string{
				"project-meter-admin",
				"project-dashboard",
			},
			IsPublic: true,
		},
		{
			Name:          RoleDomainEditor,
			DescriptionCN: "域操作员",
			Description:   "Domain operation administrator",
			Policies: []string{
				"domain-editor",
				"domain-dashboard",
			},
			IsPublic: true,
		},
		{
			Name:          RoleProjectEditor,
			DescriptionCN: "项目操作员",
			Description:   "Project operator",
			Policies: []string{
				"project-editor",
				"project-dashboard",
			},
			IsPublic: true,
		},
		{
			Name:          RoleDomainViewer,
			DescriptionCN: "域只读管理员",
			Description:   "Domain read-only administrator",
			Policies: []string{
				"domain-viewer",
				"domain-dashboard",
			},
			IsPublic: true,
		},
		{
			Name:          RoleProjectViewer,
			DescriptionCN: "项目只读成员",
			Description:   "Project read-only member",
			Policies: []string{
				"project-viewer",
				"project-dashboard",
			},
			IsPublic: true,
		},
		{
			Name:          "sys_opsadmin",
			DescriptionCN: "全局系统管理员",
			Description:   "System-wide operation manager",
			Policies: []string{
				"sys-opsadmin",
			},
			IsPublic: true,
		},
		{
			Name:          "sys_secadmin",
			DescriptionCN: "全局安全管理员",
			Description:   "System-wide security manager",
			Policies: []string{
				"sys-secadmin",
			},
			IsPublic: true,
		},
		{
			Name:          "sys_adtadmin",
			DescriptionCN: "全局审计管理员",
			Description:   "System-wide audit manager",
			Policies: []string{
				"sys-adtadmin",
			},
			IsPublic: true,
		},
		{
			Name:          "domain_opsadmin",
			DescriptionCN: "组织系统管理员",
			Description:   "Domain-wide operation manager",
			Policies: []string{
				"domain-opsadmin",
			},
			IsPublic: true,
		},
		{
			Name:          "domain_secadmin",
			DescriptionCN: "组织安全管理员",
			Description:   "Domain-wide security manager",
			Policies: []string{
				"domain-secadmin",
			},
			IsPublic: true,
		},
		{
			Name:          "domain_adtadmin",
			DescriptionCN: "组织审计管理员",
			Description:   "Domain-wide audit manager",
			Policies: []string{
				"domain-adtadmin",
			},
			IsPublic: true,
		},
		{
			Name:          "normal_user",
			DescriptionCN: "缺省普通用户角色",
			Description:   "Default normal user role",
			Policies: []string{
				"normal-user",
			},
			IsPublic: true,
		},
	}
)
