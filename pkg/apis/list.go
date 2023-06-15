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

	"yunion.io/x/onecloud/pkg/util/tagutils"
)

type ScopedResourceInput struct {
	// 指定查询的权限范围，可能值为project, domain or system
	Scope string `json:"scope"`
}

type DomainizedResourceListInput struct {
	// swagger:ignore
	// Is an admin call? equivalent to scope=system
	// Deprecated
	Admin *bool `json:"admin"`

	ScopedResourceInput

	DomainizedResourceInput

	// 对具有域属性的资源，严格匹配域ID
	ProjectDomainIds []string `json:"project_domain_ids"`
	// Deprecated
	// swagger:ignore
	ProjectDomains []string `json:"project_domains" yunion-deprecated-by:"project_domain_ids"`

	// 按domain名称排序，可能值为asc|desc
	// pattern: asc|desc
	OrderByDomain string `json:"order_by_domain"`

	// filter by domain tags
	DomainTags tagutils.TTagSetList `json:"domain_tags"`
	// filter by domain tags
	NoDomainTags tagutils.TTagSetList `json:"no_domain_tags"`

	// ignore
	// domain tags filters imposed by policy
	// PolicyDomainTags tagutils.TTagSetList `json:"policy_domain_tags"`
}

type ProjectizedResourceListInput struct {
	DomainizedResourceListInput

	ProjectizedResourceInput

	// 对具有项目属性的资源，严格匹配项目ID
	ProjectIds []string `json:"project_ids"`
	// Deprecated
	// swagger:ignore
	Projects []string `json:"projects" yunion-deprecated-by:"project_ids"`

	// 按project名称排序，可能值为asc|desc
	// pattern: asc|desc
	OrderByProject string `json:"order_by_project"`
	// swagger:ignore
	// Deprecated
	OrderByTenant string `json:"order_by_tenant" yunion-deprecated-by:"order_by_project"`

	// filter by project tags
	ProjectTags tagutils.TTagSetList `json:"project_tags"`
	// filter by no project tags
	NoProjectTags tagutils.TTagSetList `json:"no_project_tags"`

	// filter by project organizations
	ProjectOrganizations []string `json:"project_organizations"`

	// ignore
	// project tag fitlers imposed by policy
	PolicyProjectTags tagutils.TTagSetList `json:"policy_project_tags"`
}

type StatusDomainLevelUserResourceListInput struct {
	StatusDomainLevelResourceListInput

	// 查询指定的用户（ID或名称）拥有的资源
	UserId string `json:"user_id"`
	// swagger:ignore
	// Deprecated
	// Filter by userId
	User string `json:"user" yunion-deprecated-by:"user_id"`
}

type UserResourceListInput struct {
	StandaloneResourceListInput
	ScopedResourceInput

	// swagger:ignore
	// Is an admin call? equivalent to scope=system
	// Deprecated
	Admin *bool `json:"admin"`

	// 查询指定的用户（ID或名称）拥有的资源
	UserId string `json:"user_id"`
	// swagger:ignore
	// Deprecated
	// Filter by userId
	User string `json:"user" yunion-deprecated-by:"user_id"`
}

type ModelBaseListInput struct {
	Meta

	// 查询限制量
	// default: 20
	Limit *int `json:"limit" default:"20" help:"max items per page"`
	// 查询偏移量
	// default: 0
	Offset *int `json:"offset"`
	// 列表排序时，用于排序的字段的名称，该字段不提供时，则按默认字段排序。一般时按照资源的新建时间逆序排序。
	OrderBy []string `json:"order_by"`
	// 列表排序时的顺序，desc为从高到低，asc为从低到高。默认是按照资源的创建时间desc排序。
	// example: desc|asc
	Order string `json:"order"`
	// 列表返回资源的更多详细信息。默认只显示基本字段，该字段为true则返回扩展字段信息。
	Details *bool `json:"details"`
	// 模糊搜索所有字段
	Search string `json:"search"`
	// 指定过滤条件，允许指定多个，每个条件的格式为"字段名称.操作符(匹配信息)"，例如name字段等于test的过滤器为：name.equals('test')
	// 支持的操作符如下：
	//
	// | 操作符        | 参数个数 | 举例                                           | 说明                        |
	// |---------------|----------|------------------------------------------------|-----------------------------|
	// | in            | > 0      | name.in("test", "good")                        | 在给定数组中                |
	// | notin         | > 0      | name.notin('test')                             | 不在给定数组中              |
	// | between       | 2        | created_at.between('2019-12-10', '2020-01-02') | 在两个值之间                |
	// | ge            | 1        | created_at.ge('2020-01-01')                    | 大于或等于给定值            |
	// | gt            | 1        | created_at.gt('2020-01-01')                    | 严格大于给定值              |
	// | le            | 1        | created_at.le('2020-01-01')                    | 小于或等于给定值            |
	// | lt            | 1        | sync_seconds.lt(900)                           | 严格大于给定值              |
	// | like          | > 0      | name.like('%test%')                            | sql字符串匹配任意一个字符串 |
	// | contains      | > 0      | name.contains('test')                          | 包含任意一个给定字符串      |
	// | startswith    | > 0      | name.startswith('test')                        | 以任意一个给定字符串开头    |
	// | endswith      | > 0      | name.endswith('test')                          | 以任意一个给定字符串结尾    |
	// | equals        | > 0      | name.equals('test')                            | 等于任意一个给定值          |
	// | notequals     | 1        | name.notequals('test')                         | 不等于给定值                |
	// | isnull        | 0        | name.isnull()                                  | 值为SQL的NULL               |
	// | isnotnull     | 0        | name.isnotnull()                               | 值不为SQL的NULL             |
	// | isempty       | 0        | name.isempty('test')                           | 值为空字符串                |
	// | isnotempty    | 0        | name.isnotempty('test')                        | 值不是空字符串              |
	// | isnullorempty | 0        | name.isnullorempty('test')                     | 值为SQL的NULL或者空字符串   |
	//
	Filter []string `json:"filter"`
	// 指定关联过滤条件，允许指定多个，后端将根据关联过滤条件和其他表关联查询，支持的查询语法和filter相同，
	// 和其他表关联的语法如下：
	//     joint_resources.related_key(origin_key).filter_col.filter_ops(values)
	// 其中，joint_resources为要关联的资源名称，related_key为关联表column，origin_key为当前表column, filter_col为
	// 关联表用于查询匹配的field名称，field_ops为filter支持的操作，values为匹配的值
	// 举例：
	//     guestnetworks.guest_id(id).ip_addr.equals('10.168.21.222')
	JointFilter []string `json:"joint_filter"`
	// 如果filter_any为true，则查询所有filter的并集，否则为交集
	FilterAny *bool `json:"filter_any"`
	// 返回结果只包含指定的字段
	Field []string `json:"field"`
	// 用于数据导出，指定导出的数据字段
	ExportKeys string `json:"export_keys" help:"Export field keys"`
	// 返回结果携带delete_fail_reason和update_fail_reason字段
	ShowFailReason *bool `json:"show_fail_reason"`
}

func (o ModelBaseListInput) GetExportKeys() string {
	return o.ExportKeys
}

type IncrementalListInput struct {
	// 用于指定增量加载的标记
	PagingMarker string `json:"paging_marker"`
}

type VirtualResourceListInput struct {
	StatusStandaloneResourceListInput
	ProjectizedResourceListInput

	// 列表中包含标记为"系统资源"的资源
	System *bool `json:"system"`
	// 是否显示回收站内的资源，默认不显示（对实现了回收站的资源有效，例如主机，磁盘，镜像）
	PendingDelete *bool `json:"pending_delete"`
	// 是否显示所有资源，包括回收站和不再回收站的资源
	// TODO: fix this???
	PendingDeleteAll *bool `json:"-"`
}

type ResourceBaseListInput struct {
	ModelBaseListInput
}

type SharableVirtualResourceListInput struct {
	VirtualResourceListInput
	SharableResourceBaseListInput
	// 根据资源的共享范围过滤列表，可能值为：system, domain, project
	PublicScope string `json:"public_scope"`
}

type AdminSharableVirtualResourceListInput struct {
	SharableVirtualResourceListInput
}

type MetadataResourceListInput struct {
	// 通过标签过滤（包含这些标签）
	Tags tagutils.TTagSet `json:"tags"`

	// 通过一组标签过滤（还包含这些标签，OR的关系）
	ObjTags tagutils.TTagSetList `json:"obj_tags"`

	// 通过标签过滤（不包含这些标签）
	NoTags tagutils.TTagSet `json:"no_tags"`

	// 通过一组标签过滤（还不包含这些标签，AND的关系）
	NoObjTags tagutils.TTagSetList `json:"no_obj_tags"`

	// ignore
	// 策略规定的标签过滤器
	// PolicyObjectTags tagutils.TTagSetList `json:"policy_object_tags"`

	// 通过标签排序
	OrderByTag string `json:"order_by_tag"`

	// deprecated
	// 返回资源的标签不包含用户标签
	WithoutUserMeta *bool `json:"without_user_meta"`

	// 返回包含用户标签的资源
	WithUserMeta *bool `json:"with_user_meta"`

	// 返回包含外部标签的资源
	WithCloudMeta *bool `json:"with_cloud_meta"`

	// 返回包含任意标签的资源
	WithAnyMeta *bool `json:"with_any_meta"`

	// 返回列表数据中包含资源的标签数据（Metadata）
	WithMeta *bool `json:"with_meta"`
}

type StandaloneAnonResourceListInput struct {
	ResourceBaseListInput

	MetadataResourceListInput

	// 显示所有的资源，包括模拟的资源
	ShowEmulated *bool `json:"show_emulated" help:"show emulated resources" negative:"do not show emulated resources"`

	// 以资源ID过滤列表
	Ids []string `json:"id" help:"filter by ids"`
}

type StandaloneResourceListInput struct {
	StandaloneAnonResourceListInput

	// 以资源名称过滤列表
	Names []string `json:"name" help:"filter by names"`
}

type StatusResourceBaseListInput struct {
	// 以资源的状态过滤列表
	Status []string `json:"status"`
}

type EnabledResourceBaseListInput struct {
	// 以资源是否启用/禁用过滤列表
	Enabled *bool `json:"enabled"`
}

type SharableResourceBaseListInput struct {
	// 以资源是否共享过滤列表
	IsPublic *bool `json:"is_public"`
	// 根据资源的共享范围过滤列表，可能值为：system, domain, project
	PublicScope string `json:"public_scope"`
}

type DomainLevelResourceListInput struct {
	StandaloneResourceListInput
	DomainizedResourceListInput
}

type StatusStandaloneResourceListInput struct {
	StandaloneResourceListInput
	StatusResourceBaseListInput
}

type EnabledStatusStandaloneResourceListInput struct {
	StatusStandaloneResourceListInput
	EnabledResourceBaseListInput
}

type StatusDomainLevelResourceListInput struct {
	DomainLevelResourceListInput
	StatusResourceBaseListInput
}

type EnabledStatusDomainLevelResourceListInput struct {
	StatusDomainLevelResourceListInput
	EnabledResourceBaseListInput
}

type JointResourceBaseListInput struct {
	ResourceBaseListInput
}

type VirtualJointResourceBaseListInput struct {
	JointResourceBaseListInput
}

type ExternalizedResourceBaseListInput struct {
	// 以资源外部ID过滤
	ExternalId string `json:"external_id"`
}

type DeletePreventableResourceBaseListInput struct {
	// 是否禁止删除
	DisableDelete *bool `json:"disable_delete"`
}

type ScopedResourceBaseListInput struct {
	ProjectizedResourceListInput
	// 指定匹配的范围，可能值为project, domain or system
	BelongScope string `json:"belong_scope"`
}

type InfrasResourceBaseListInput struct {
	DomainLevelResourceListInput
	SharableResourceBaseListInput
}

type StatusInfrasResourceBaseListInput struct {
	InfrasResourceBaseListInput
	StatusResourceBaseListInput
}

type EnabledStatusInfrasResourceBaseListInput struct {
	StatusInfrasResourceBaseListInput
	EnabledResourceBaseListInput
}

type MultiArchResourceBaseListInput struct {
	// 通过操作系统架构过滤
	// x86会过滤出os_arch为空或os_arch=i386或以x86开头的资源
	// arm会过滤出os_arch=aarch64或os_arch=aarch32或者以arm开头的资源
	// 其他的输入会过滤出以输入字符开头的资源
	// enmu: x86, arm
	OsArch []string `json:"os_arch"`
}

type AutoDeleteResourceBaseListInput struct {
	AutoDelete *bool
}

type OpsLogListInput struct {
	OwnerProjectIds []string `json:"owner_project_ids"`
	OwnerDomainIds  []string `json:"owner_domain_ids"`

	// filter by obj type
	ObjTypes []string `json:"obj_type"`

	// filter by obj name or obj id
	Objs []string `json:"obj"`

	// filter by obj ids
	ObjIds []string `json:"obj_id"`

	// filter by obj name
	ObjNames []string `json:"obj_name"`

	// filter by action
	Actions []string `json:"action"`

	Since time.Time `json:"since"`

	Until time.Time `json:"until"`
}

type IdNameDetails struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type TotalCountBase struct {
	Count int `json:"count"`
}
