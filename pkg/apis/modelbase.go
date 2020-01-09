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

type ModelBaseDetails struct {
	CanDelete        bool   `json:"can_delete"`
	DeleteFailReason string `json:"delete_fail_reason"`
	CanUpdate        bool   `json:"can_update"`
	UpdateFailReason string `json:"update_fail_reason"`
}

type ModelBaseShortDescDetail struct {
	ResName string `json:"res_name"`
}

type ModelBaseListInput struct {
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
	Order string `json:"order"`
	// Show more details
	Details *bool `json:"details"`
	// Filter results by a simple keyword search
	Search string `json:"search"`
	// Filters
	Filter []string `json:"filter"`
	// Filters with joint table col; joint_tbl.related_key(origin_key).filter_col.filter_cond(filters)
	JointFilter []string `json:"joint_filter"`
	// If true, match if any of the filters matches; otherwise, match if all of the filters match
	FilterAny *bool `json:"filter_any"`
	// Show only specified fields
	Field []string `json:"field"`
	// Export field keys
	ExportKeys string `json:"export_keys"`
}

type IncrementalListInput struct {
	// 增量加载的标记
	// Marker for streamed pagination
	PagingMarker string `json:"paging_marker"`
}
