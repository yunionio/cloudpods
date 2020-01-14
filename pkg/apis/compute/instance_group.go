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

package compute

import "yunion.io/x/onecloud/pkg/apis"

type InstanceGroupListInput struct {
	apis.VirtualResourceListInput
	ZonalFilterListInput
	ServerFilterListInput

	// 以service_type过滤列表结果
	ServiceType string `json:"service_type"`
	// 以parent_id过滤列表结果
	ParentId string `json:"parent_id"`
}

type InstanceGroupDetail struct {
	apis.VirtualResourceDetails
	SGroup

	// 云主机数量
	GuestCount int `json:"guest_count"`
}

type GroupFilterListInput struct {
	// 以指定实例组（ID或Name）过滤列表结果
	Group string `json:"group"`
	// swagger:ignore
	// deprecated: true
	// Filter by instance group Id
	GroupId string `json:"group_id" deprecated-by:"group"`
}
