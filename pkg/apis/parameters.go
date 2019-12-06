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

type BaseListInput struct {
	Meta

	// 查询限制量
	// default: 20
	Limit *int `json:"limit"`
	// 查询偏移量
	// default: 0
	Offset *int `json:"offset"`
	// Name of the field to be ordered by
	OrderBy []string `json:"order_by"`
	// List Order
	// example: desc|asc
	Order string
	// Show more details
	Details *bool `json:"details"`
	// Filter results by a simple keyword search
	Search string `json:"search"`
	// Piggyback metadata information
	WithMeta *bool `json:"with_meta"`
	// Filters
	Filter []string `json:"filters"`
	// Filters with joint table col; joint_tbl.related_key(origin_key).filter_col.filter_cond(filters)
	JointFilter []string `json:"joint_filter"`
	// If true, match if any of the filters matches; otherwise, match if all of the filters match
	FilterAny *bool `json:"filter_any"`
	// Is an admin call?
	Admin *bool `json:"admin"`
	// Tenant ID or Name
	Tenant string `json:"tenant"`
	// Project domain filter
	ProjectDomain string `json:"project_domain"`
	// User ID or Name
	User string `json:"user"`
	// Show only specified fields
	Field []string `json:"field"`
	// Specify query scope, either project, domain or system
	Scope string `json:"scope"`
	// Show system resource
	System *bool `json:"system"`
	// Show only pending deleted resource
	PendingDelete *bool `json:"pending_delete"`
	// Show all resources including pending deleted
	// TODO: fix this???
	PendingDeleteAll *bool `json:"-"`
	// Show all resources including the emulated resources
	ShowEmulated *bool `json:"show_emulated"`
	// Export field keys
	ExportKeys string `json:"export_keys"`

	// Marker for pagination
	PagingMarker string `json:"paging_marker"`
}
