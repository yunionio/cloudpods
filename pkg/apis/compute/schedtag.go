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

type SchedtagShortDescDetails struct {
	*apis.StandaloneResourceShortDescDetail
	Default string `json:"default"`
}

type SchedtagCreateInput struct {
	apis.StandaloneResourceCreateInput
	apis.ScopedResourceCreateInput

	// 动态标签策略
	// enum: exclude, prefer, avoid
	DefaultStrategy string `json:"default_strategy"`

	// 资源类型
	// enum: servers, hosts, .....
	// default: hosts
	ResourceType string `json:"resource_type"`
}

type SchedtagResourceInput struct {
	// 以关联的调度标签（ID或Name）过滤列表
	SchedtagId string `json:"schedtag_id"`
	// swagger:ignore
	// Deprecated
	// filter by schedtag_id
	Schedtag string `json:"schedtag" yunion-deprecated-by:"schedtag_id"`
}

type SchedtagFilterListInput struct {
	SchedtagResourceInput

	// 按调度标签名称排序
	// pattern:asc|desc
	OrderBySchedtag string `json:"order_by_schedtag"`

	// 按调度标签资源类型排序
	// pattern:asc|desc
	OrderByResourceType string `json:"order_by_resource_type"`
}

type SchedtagListInput struct {
	apis.StandaloneResourceListInput
	apis.ScopedResourceBaseListInput
	CloudproviderResourceInput

	// fitler by resource_type
	ResourceType []string `json:"resource_type"`
	// swagger:ignore
	// Deprecated
	// filter by type, alias for resource_type
	Type string `json:"type" yunion-deprecated-by:"resource_type"`

	DefaultStrategy []string `json:"default_strategy"`
}

type SchedtagDetails struct {
	apis.StandaloneResourceDetails
	apis.ScopedResourceBaseInfo

	SSchedtag

	DynamicSchedtagCount int    `json:"dynamic_schedtag_count"`
	SchedpolicyCount     int    `json:"schedpolicy_count"`
	HostCount            int    `json:"host_count"`
	ServerCount          int    `json:"server_count"`
	OtherCount           int    `json:"other_count"`
	ResourceCount        int    `json:"resource_count"`
	JoinModelKeyword     string `json:"join_model_keyword"`
}

type SchedtagResourceInfo struct {

	// 调度标签名称
	Schedtag string `json:"schedtag"`

	// 调度标签管理的资源类型
	ResourceType string `json:"resource_type"`
}

type SchedtagJointResourceDetails struct {
	apis.JointResourceBaseDetails

	// 调度标签名称
	Schedtag string `json:"schedtag"`

	// 调度标签管理的资源类型
	ResourceType string `json:"resource_type"`
}

type SchedtagJointsListInput struct {
	apis.JointResourceBaseListInput
	SchedtagFilterListInput
}

type SchedtagSetResourceInput struct {
	ResourceIds []string `json:"resource_ids"`
}
