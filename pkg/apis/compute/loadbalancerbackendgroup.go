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

type LoadbalancerBackendGroupDetails struct {
	apis.VirtualResourceDetails
	LoadbalancerResourceInfo

	SLoadbalancerBackendGroup
}

type LoadbalancerBackendGroupResourceInfo struct {
	LoadbalancerResourceInfo

	// 负载均衡后端组名称
	BackendGroup string `json:"backend_group"`

	// 负载均衡ID
	LoadbalancerId string `json:"loadbalancer_id"`
}

type LoadbalancerBackendGroupResourceInput struct {
	// 负载均衡后端组ID或名称
	BackendGroup string `json:"backend_group"`

	// swagger:ignore
	// Deprecated
	BackendGroupId string `json:"backend_group_id" "yunion:deprecated-by":"backend_group"`
}

type LoadbalancerBackendGroupFilterListInput struct {
	LoadbalancerFilterListInput

	LoadbalancerBackendGroupResourceInput

	// 以负载均衡后端组名称排序
	OrderByBackendGroup string `json:"order_by_backend_group"`
}
