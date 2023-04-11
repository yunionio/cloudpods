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

	apis.EnabledResourceBaseListInput

	ZonalFilterListInput

	ServerFilterListInput

	// 以service_type过滤列表结果
	ServiceType string `json:"service_type"`
	// 以parent_id过滤列表结果
	ParentId string `json:"parent_id"`

	// 调度策略
	SchedStrategy string `json:"sched_strategy"`
	// 按ip地址排序
	// pattern:asc|desc
	OrderByVips string `json:"order_by_vips"`
	// 按关联的虚拟机数量排序
	// pattern:asc|desc
	OrderByGuestCount string `json:"order_by_guest_count"`
}

type InstanceGroupDetail struct {
	apis.VirtualResourceDetails
	ZoneResourceInfo

	SGroup

	// 云主机数量
	GuestCount int `json:"guest_count"`

	// VIP
	Vips []string `json:"vips"`

	// EIP of VIP
	VipEip string `json:"vip_eip"`

	// Attached NetworkId
	NetworkId string `json:"network_id"`

	// Attached Network
	Network string `json:"network"`
}

type GroupResourceInput struct {
	// 实例组（ID或Name）
	GroupId string `json:"group_id"`
	// swagger:ignore
	// Deprecated
	// Filter by instance group Id
	Group string `json:"group" yunion-deprecated-by:"group_id"`
}

type GroupFilterListInput struct {
	GroupResourceInput

	// 按组名排序
	OrderByGroup string `json:"order_by_group"`
}

type GroupResourceInfo struct {
	// 实例组名称
	Group string `json:"group"`
}
