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
	apis.StatusStandaloneResourceDetails
	LoadbalancerResourceInfo

	SLoadbalancerBackendGroup

	LbListenerCount int `json:"lb_listener_count"`

	IsDefault bool   `json:"is_default"`
	ProjectId string `json:"tenant_id"`
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
	BackendGroupId string `json:"backend_group_id"`

	// swagger:ignore
	// Deprecated
	BackendGroup string `json:"backend_group" yunion-deprecated-by:"backend_group_id"`
}

type LoadbalancerBackendGroupFilterListInput struct {
	LoadbalancerFilterListInput

	LoadbalancerBackendGroupResourceInput

	// 以负载均衡后端组名称排序
	OrderByBackendGroup string `json:"order_by_backend_group"`
}

type LoadbalancerBackendGroupCreateInput struct {
	apis.StatusStandaloneResourceCreateInput

	//swagger: ignore
	Loadbalancer string `json:"loadbalancer" yunion-deprecated-by:"loadbalancer_id"`
	// 负载均衡ID
	LoadbalancerId string `json:"loadbalancer_id"`

	Type string `json:"type"`

	Backends []struct {
		Index       int
		Weight      int
		Port        int
		Id          string
		Name        string
		ExternalId  string
		BackendType string
		BackendRole string
		Address     string
		ZoneId      string
		HostName    string
	} `json:"backends"`
}

type LoadbalancerBackendGroupListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	LoadbalancerFilterListInput

	// filter LoadbalancerBackendGroup with no reference
	NoRef *bool `json:"no_ref"`

	Type []string `json:"type"`
}
