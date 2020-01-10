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

type StandaloneResourceShortDescDetail struct {
	ModelBaseShortDescDetail

	Id   string `json:"id"`
	Name string `json:"name"`
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
