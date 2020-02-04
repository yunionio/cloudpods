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
	Meta

	// 资源是否可以删除, 若为flase, delete_fail_reason会返回不能删除的原因
	// example: true
	CanDelete bool `json:"can_delete"`

	// 资源不能删除的原因
	DeleteFailReason string `json:"delete_fail_reason"`

	// 资源是否可以更新, 若为false,update_fail_reason会返回资源不能删除的原因
	// example: true
	CanUpdate bool `json:"can_update"`

	// 资源不能删除的原因
	UpdateFailReason string `json:"update_fail_reason"`
}

type JoinModelBaseDetails struct {
	ModelBaseDetails
}

type ModelBaseShortDescDetail struct {
	ResName string `json:"res_name"`
}

type SharedProject struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type SharableVirtualResourceDetails struct {
	VirtualResourceDetails

	SharedProjects []SharedProject `json:"shared_projects"`
}

type StandaloneResourceShortDescDetail struct {
	ModelBaseShortDescDetail

	Id   string `json:"id"`
	Name string `json:"name"`
}

type VirtualResourceDetails struct {
	StandaloneResourceDetails
}

type StandaloneResourceDetails struct {
	ModelBaseDetails
}
