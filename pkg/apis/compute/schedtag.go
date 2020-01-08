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

type ScopedResourceCreateInput struct {
	Scope string `json:"scope"`
}

type SchedtagCreateInput struct {
	apis.StandaloneResourceCreateInput
	ScopedResourceCreateInput

	// 动态标签策略
	// enum: exclude, prefer, avoid
	DefaultStrategy string `json:"default_strategy"`

	// 资源类型
	// enum: servers, hosts, .....
	// default: hosts
	ResourceType string `json:"resource_type"`
}

type SchedtagResourceListInput struct {
	// filter by schedtag
	Schedtag string `json:"schedtag"`
	// swagger: ignore
	// Deprecated
	// filter by schedtag_id
	SchedtagId string `json:"schedtag_id"`
}

func (input SchedtagResourceListInput) SchedtagStr() string {
	if len(input.Schedtag) > 0 {
		return input.Schedtag
	}
	if len(input.SchedtagId) > 0 {
		return input.SchedtagId
	}
	return ""
}

type SchedtagListInput struct {
	apis.StandaloneResourceListInput

	// fitler by resource_type
	ResourceType string `json:"resource_type"`
	// swagger: ignore
	// Deprecated
	// filter by type, alias for resource_type
	Type string `json:"type"`
}

func (input SchedtagListInput) ResourceTypeStr() string {
	if len(input.ResourceType) > 0 {
		return input.ResourceType
	}
	if len(input.Type) > 0 {
		return input.Type
	}
	return ""
}
