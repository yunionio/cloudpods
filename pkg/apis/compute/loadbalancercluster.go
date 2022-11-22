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

type LoadbalancerClusterDetails struct {
	apis.StandaloneResourceDetails
	ZoneResourceInfo
	WireResourceInfoBase

	SLoadbalancerCluster
}

type LoadbalancerClusterResourceInfo struct {
	ZoneResourceInfo

	WireResourceInfoBase

	// VPC ID
	VpcId string `json:"vpc_id"`

	// VPC名称
	Vpc string `json:"vpc"`

	// 负载均衡集群名称
	Cluster string `json:"cluster"`
}

type LoadbalancerClusterResourceInput struct {
	// 负载均衡集群ID或名称
	ClusterId string `json:"cluster_id"`

	// swagger:ignore
	// Deprecated
	Cluster string `json:"cluster" yunion-deprecated-by:"cluster_id"`
}

type LoadbalancerClusterFilterListInput struct {
	ZonalFilterListInput
	WireFilterListBase

	LoadbalancerClusterResourceInput

	// 以负载均衡集群排序
	OrderByCluster string `json:"order_by_cluster"`
}

type LoadbalancerClusterListInput struct {
	apis.StandaloneResourceListInput

	ZonalFilterListInput
	WireFilterListBase
}
