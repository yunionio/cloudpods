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

type BaseListOptions struct {
	// Query page limit
	// default: 20
	Limit int `json:"limit"`

	// Query page offset
	// default: 0
	Offset int `json:"offset"`

	// Name of the field to be ordered by
	OrderBy []string `json:"order_by"`

	// List order
	// enum: desc,asc
	Order string `json:"order"`

	// Show more details
	// default:false
	Details bool `json:"details"`

	// Filter results by a simple keyword search
	Search string `json:"search"`

	// Piggyback metadata information
	Meta bool `json:"meta"`

	// Filters
	Filter []string `json:"filters"`
}

type OverridePendingDelete struct {
	// 从回收站删除资源时需要指定此参数为true, 回收站资源到期会自动释放。到期清理时间由服务参数PendingDeleteExpireSeconds确定。默认是259200秒（3天）
	// in: query
	OverridePendingDelete bool `json:"override_pending_delete"`
}
