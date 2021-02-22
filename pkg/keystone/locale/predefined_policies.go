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
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

const (
	RoleAdmin         = "admin"
	RoleFA            = "fa"
	RoleDomainFA      = "domainfa"
	RoleProjectFA     = "projectfa"
	RoleSA            = "sa"
	RoleProjectOwner  = "project_owner"
	RoleDomainAdmin   = "domainadmin"
	RoleProjectEditor = "project_editor"
	RoleProjectViewer = "project_viewer"

	RoleMember = "member"
)

type sPolicyDefinition struct {
	Name     string
	DescCN   string
	Desc     string
	Scope    rbacutils.TRbacScope
	Services map[string][]string
	Extra    map[string]map[string][]string
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
			Scope:  rbacutils.ScopeProject,
			Services: map[string][]string{
				"*": nil,
			},
		},
		{
			Name:   "dashboard",
			DescCN: "控制面板查看相关资源",
			Desc:   "resources for viewing dashboard",
			Scope:  rbacutils.ScopeProject,
			Extra: map[string]map[string][]string{
				"compute": {
					"capabilities": {
						"list",
					},
					"usages": {
						"list",
						"get",
					},
				},
				"image": {
					"usages": {
						"list",
						"get",
					},
				},
				"identity": {
					"usages": {
						"list",
						"get",
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
				},
				"log": {
					"actions": {
						"list",
					},
				},
			},
		},
		{
			Name:   "compute",
			DescCN: "计算服务(云主机与容器)相关资源",
			Desc:   "resources of computing (cloud servers and containers)",
			Scope:  rbacutils.ScopeProject,
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
			Scope:  rbacutils.ScopeProject,
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
			Scope:  rbacutils.ScopeDomain,
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
			Scope:  rbacutils.ScopeDomain,
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
			Scope:  rbacutils.ScopeProject,
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
			Scope:  rbacutils.ScopeProject,
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
			Scope:  rbacutils.ScopeProject,
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
			Scope:  rbacutils.ScopeProject,
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
			Scope:  rbacutils.ScopeDomain,
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
			Name:   "meter",
			DescCN: "计费计量分析服务相关资源",
			Desc:   "resources of metering and billing service",
			Scope:  rbacutils.ScopeProject,
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
			Scope:  rbacutils.ScopeDomain,
			Services: map[string][]string{
				"identity": nil,
			},
		},
		{
			Name:   "image",
			DescCN: "镜像服务相关资源",
			Desc:   "resources of image service",
			Scope:  rbacutils.ScopeProject,
			Services: map[string][]string{
				"image": nil,
			},
		},
		{
			Name:   "monitor",
			DescCN: "监控服务相关资源",
			Desc:   "resources of monitor service",
			Scope:  rbacutils.ScopeProject,
			Services: map[string][]string{
				"monitor": nil,
			},
		},
		{
			Name:   "container",
			DescCN: "容器服务相关资源",
			Desc:   "resources of container service",
			Scope:  rbacutils.ScopeProject,
			Services: map[string][]string{
				"k8s": nil,
			},
		},
		{
			Name:   "cloudid",
			DescCN: "云用户及权限管理相关资源",
			Desc:   "resources of service CloudId and IAM",
			Scope:  rbacutils.ScopeDomain,
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
			Scope:  rbacutils.ScopeDomain,
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
			Scope:  rbacutils.ScopeDomain,
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
			Scope:  rbacutils.ScopeSystem,
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
			Scope:  rbacutils.ScopeSystem,
			Services: map[string][]string{
				"notify": nil,
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
				"domainadmin",
			},
			IsPublic: true,
		},
		{
			Name:          RoleProjectOwner,
			DescriptionCN: "项目主管",
			Description:   "Project owner",
			Policies: []string{
				"projectadmin",
			},
			IsPublic: true,
		},
		{
			Name:          RoleFA,
			DescriptionCN: "系统财务管理员",
			Description:   "System finance administrator",
			Policies: []string{
				"sysmeteradmin",
				"sysdashboard",
			},
			IsPublic: false,
		},
		{
			Name:          RoleDomainFA,
			DescriptionCN: "域财务管理员",
			Description:   "Domain finance administrator",
			Policies: []string{
				"domainmeteradmin",
				"domaindashboard",
			},
			IsPublic: true,
		},
		{
			Name:          RoleProjectFA,
			DescriptionCN: "项目财务管理员",
			Description:   "Project finance administrator",
			Policies: []string{
				"projectmeteradmin",
				"projectdashboard",
			},
			IsPublic: true,
		},
		{
			Name:          RoleProjectEditor,
			DescriptionCN: "项目操作员",
			Description:   "Project operator",
			Policies: []string{
				"projecteditor",
				"projectdashboard",
			},
			IsPublic: true,
		},
		{
			Name:          RoleProjectViewer,
			DescriptionCN: "项目只读成员",
			Description:   "Project read-only member",
			Policies: []string{
				"projectviewer",
				"projectdashboard",
			},
			IsPublic: true,
		},
	}
)
