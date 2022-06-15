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

package apis

import (
	"time"

	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	EXTERNAL_RESOURCE_SOURCE_LOCAL = "local"
	EXTERNAL_RESOURCE_SOURCE_CLOUD = "cloud"
)

type ModelBaseDetails struct {
	Meta

	// 资源是否可以删除, 若为flase, delete_fail_reason会返回不能删除的原因
	// example: true
	CanDelete bool `json:"can_delete"`

	// 资源不能删除的原因
	DeleteFailReason httperrors.Error `json:"delete_fail_reason"`

	// 资源是否可以更新, 若为false,update_fail_reason会返回资源不能删除的原因
	// example: true
	CanUpdate bool `json:"can_update"`

	// 资源不能更新的原因
	UpdateFailReason httperrors.Error `json:"update_fail_reason"`
}

type ModelBaseShortDescDetail struct {
	ResName string `json:"res_name"`
}

type SharedProject struct {
	Id   string `json:"id"`
	Name string `json:"name"`

	DomainId string `json:"domain_id"`
	Domain   string `json:"domain"`
}

type SharedDomain struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type SharableResourceBaseInfo struct {
	// 共享的项目列表
	SharedProjects []SharedProject `json:"shared_projects"`
	// 共享的域列表
	SharedDomains []SharedDomain `json:"shared_domains"`
}

type SharableVirtualResourceDetails struct {
	VirtualResourceDetails
	SharableResourceBaseInfo
}

type AdminSharableVirtualResourceDetails struct {
	SharableVirtualResourceDetails
}

type StandaloneAnonResourceShortDescDetail struct {
	ModelBaseShortDescDetail

	Id string `json:"id"`
}

type StandaloneResourceShortDescDetail struct {
	StandaloneAnonResourceShortDescDetail

	Name string `json:"name"`
}

type EnabledStatusDomainLevelResourceDetails struct {
	StatusDomainLevelResourceDetails
}

type StatusDomainLevelResourceDetails struct {
	DomainLevelResourceDetails
}

type DomainLevelResourceDetails struct {
	StandaloneResourceDetails

	DomainizedResourceInfo
}

type VirtualResourceDetails struct {
	StatusStandaloneResourceDetails

	ProjectizedResourceInfo
}

type VirtualJointResourceBaseDetails struct {
	JointResourceBaseDetails
}

type JointResourceBaseDetails struct {
	ResourceBaseDetails
}

type ResourceBaseDetails struct {
	ModelBaseDetails
}

type EnabledStatusStandaloneResourceDetails struct {
	StatusStandaloneResourceDetails
}

type StatusStandaloneResourceDetails struct {
	StandaloneResourceDetails
}

type MetadataResourceInfo struct {
	// 标签
	Metadata map[string]string `json:"metadata"`
}

type StatusDomainLevelUserResourceDetails struct {
	StatusDomainLevelResourceDetails

	// 用户名称
	OwnerName string `json:"owner_name"`
}

type UserResourceDetails struct {
	StandaloneResourceDetails

	// 用户名称
	OwnerName string `json:"owner_name"`
}

type StandaloneAnonResourceDetails struct {
	ResourceBaseDetails

	MetadataResourceInfo
}

type StandaloneResourceDetails struct {
	StandaloneAnonResourceDetails
}

type DomainizedResourceInfo struct {
	// 资源归属项目的域名称
	ProjectDomain string `json:"project_domain"`
}

type ProjectizedResourceInfo struct {
	DomainizedResourceInfo

	// 资源归属项目的名称
	// alias:project
	Project string `json:"tenant"`

	// 资源归属项目的ID(向后兼容别名）
	// Deprecated
	TenantId string `json:"project_id" yunion-deprecated-by:"tenant_id"`

	// 资源归属项目的名称（向后兼容别名）
	// Deprecated
	Tenant string `json:"project" yunion-deprecated-by:"tenant"`
}

type ScopedResourceBaseInfo struct {
	ProjectizedResourceInfo
	Scope string `json:"scope"`
}

type InfrasResourceBaseDetails struct {
	DomainLevelResourceDetails
	SharableResourceBaseInfo
}

type StatusInfrasResourceBaseDetails struct {
	InfrasResourceBaseDetails
}

type EnabledStatusInfrasResourceBaseDetails struct {
	StatusInfrasResourceBaseDetails
}

type ChangeOwnerCandidateDomainsOutput struct {
	Candidates []SharedDomain `json:"candidates"`
}

type OpsLogDetails struct {
	ModelBaseDetails

	Id      int64  `json:"id"`
	ObjType string `json:"obj_type"`
	ObjId   string `json:"obj_id"`
	ObjName string `json:"obj_name"`
	Action  string `json:"action"`
	Notes   string `json:"notes"`

	ProjectId string `json:"tenant_id"`
	Project   string `json:"tenant"`

	ProjectDomainId string `json:"project_domain_id"`
	ProjectDomain   string `json:"project_domain"`

	UserId   string `json:"user_id"`
	User     string `json:"user"`
	DomainId string `json:"domain_id"`
	Domain   string `json:"domain"`
	Roles    string `json:"roles"`

	OpsTime time.Time `json:"ops_time"`

	OwnerDomainId  string `json:"owner_domain_id"`
	OwnerProjectId string `json:"owner_project_id"`

	OwnerDomain  string `json:"owner_domain"`
	OwnerProject string `json:"owner_tenant"`
}

type StatusStatisticStatusInfo struct {
	Status string
	// 资源总数
	TotalCount int64 `json:"total_count"`
	// 回收站数量
	// 需要指定pending_delete=all
	PendingDeletedCount int64 `json:"pending_deleted_count"`
}

type StatusStatistic struct {
	StatusInfo []StatusStatisticStatusInfo `json:"status_info,allowempty"`
	// CPU总量
	TotalCpuCount int
	// 内存总量
	TotalMemSizeMb int
	// 存储总量
	TotalDiskSizeMb int
}

type ProjectStatistic struct {
	Count int
	Id    string
	Name  string
}
