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

type ModelBaseListInput struct {
	Meta

	// 查询限制量
	// default: 20
	Limit *int `json:"limit"`
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
	// | 操作符         | 参数个数 | 举例                                            | 说明                  ｜
	// |---------------|---------|------------------------------------------------|-----------------------|
	// | in            | > 0     | name.in("test", "good")                        | 在给定数组中            |
	// | notin         | > 0     | name.notin('test')                             | 不在给定数组中          |
	// | between       | 2       | created_at.between('2019-12-10', '2020-01-02') | 在两个值之间            |
	// | ge            | 1       | created_at.ge('2020-01-01')                    | 大于或等于给定值         |
	// | gt            | 1       | created_at.gt('2020-01-01')                    | 严格大于给定值           |
	// | le            | 1       | created_at.le('2020-01-01')                    | 小于或等于给定值         |
	// | lt            | 1       | sync_seconds.lt(900)                           | 严格大于给定值           |
	// | like          | 1       | name.like('%test%')                            | sql字符串匹配            |
	// | contains      | 1       | name.contains('test')                          | 包含给定字符串           |
	// | startswith    | 1       | name.startswith('test')                        | 以给定字符串开头         |
	// | endswith      | 1       | name.endswith('test')                          | 以给定字符串结尾          |
	// | equals        | 1       | name.equals('test')                            | 等于给定值               |
	// | notequals     | 1       | name.notequals('test')                         | 不等于给定值             |
	// | isnull        | 0       | name.isnull()                                  | 值为SQL的NULL           |
	// | isnotnull     | 0       | name.isnotnull()                               | 值不为SQL的NULL         |
	// | isempty       | 0       | name.isempty('test')                           | 值为空字符串             |
	// | isnotempty    | 0       | name.isnotempty('test')                        | 值不是空字符串           |
	// | isnullorempty | 0       | name.isnullorempty('test')                     | 值为SQL的NULL或者空字符串 |
	//
	Filter []string `json:"filter"`
	// 指定关联过滤条件，允许指定多个，后端将根据关联过滤条件和其他表关联查询，支持的查询语法和filter相同，
	// 和其他表关联的语法如下：
	//     joint_tbl.related_key(origin_key).filter_col.filter_ops(values)
	// 其中，joint_tbl为要关联的表，related_key为关联表column，origin_key为当前表column, filter_col为
	// 关联表用于查询匹配的field名称，field_ops为filter支持的操作，values为匹配的值
	// 举例：
	//     guestnetworks_tbl.guest_id(id).ip_addr.equals('10.168.21.222')
	JointFilter []string `json:"joint_filter"`
	// 如果filter_any为true，则查询所有filter的并集，否则为交集
	FilterAny *bool `json:"filter_any"`
	// 返回结果只包含指定的字段
	Field []string `json:"field"`
	// 用于数据导出，指定导出的数据字段
	ExportKeys string `json:"export_keys"`
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

type SharableResourceListInput struct {
	// 根据资源是否共享过滤列表
	IsPublic *bool `json:"is_public"`
}

type SharableVirtualResourceListInput struct {
	VirtualResourceListInput

	// 根据资源的共享范围过滤列表，可能值为：system, domain, project
	PublicScope string `json:"public_scope"`
}

type AdminSharableVirtualResourceListInput struct {
	SharableVirtualResourceListInput
}

type STag struct {
	// 标签key
	Key string
	// 标签Value
	Value string
}

type StandaloneResourceListInput struct {
	ResourceBaseListInput

	// 通过标签过滤
	Tags []STag `json:"tags"`
	// 返回资源的标签不包含特定的用户标签
	WithoutUserMeta bool `json:"without_user_meta"`
	// 返回列表数据中包含资源的标签数据（Metadata）
	WithMeta *bool `json:"with_meta"`
	// 显示所有的资源，包括模拟的资源
	ShowEmulated *bool `json:"show_emulated"`

	// 以资源名称过滤列表
	Names []string `json:"name"`
	// 以资源ID过滤列表
	Ids []string `json:"id"`
}

type StatusStandaloneResourceListInput struct {
	StandaloneResourceListInput

	// 以资源的状态过滤列表
	Status []string `json:"status"`
}

type EnabledStatusStandaloneResourceListInput struct {
	StatusStandaloneResourceListInput

	// 以资源是否启用/禁用过滤列表
	Enabled *bool `json:"enabled"`
}
