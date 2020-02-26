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

import (
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/util/ansible"
)

type LoadbalancerAgentDeployInput struct {
	apis.Meta

	Host         ansible.Host
	DeployMethod string
}

const (
	DeployMethodYum  = "yum"
	DeployMethodCopy = "copy"
)

type LoadbalancerListenerListInput struct {
	apis.VirtualResourceListInput
	LoadbalancerFilterListInput

	// filter by backend_group
	BackendGroup string `json:"backend_group"`
	// filter by acl
	Acl string `json:"acl"`
}

type LoadbalancerListenerRuleListInput struct {
	apis.VirtualResourceListInput
	LoadbalancerListenerFilterListInput

	// filter by backend_group
	BackendGroup string `json:"backend_group"`
}

type LoadbalancerListInput struct {
	apis.VirtualResourceListInput
	VpcFilterListInput
	ZonalFilterListBase
	NetworkFilterListBase

	// filter by cluster
	Cluster string `json:"cluster"`
}

type LoadbalancerAgentListInput struct {
	apis.StandaloneResourceListInput
	LoadbalancerClusterFilterListInput
}

type LoadbalancerCertificateListInput struct {
	apis.VirtualResourceListInput

	UsableResourceListInput
	RegionalFilterListInput
	ManagedResourceListInput
}

type LoadbalancerBackendListInput struct {
	apis.VirtualResourceListInput

	LoadbalancerBackendGroupFilterListInput

	// filter by backend server
	Backend string `json:"backend"`

	// filter by backend group
	// BackendGroup string `json:"backend_group"`
}

type LoadbalancerBackendGroupListInput struct {
	apis.VirtualResourceListInput

	LoadbalancerFilterListInput

	// filter LoadbalancerBackendGroup with no reference
	NoRef *bool `json:"no_ref"`
}

type LoadbalancerClusterListInput struct {
	apis.StandaloneResourceListInput

	ZonalFilterListInput
	WireFilterListBase
}

type LoadbalancerAclListInput struct {
	apis.SharableVirtualResourceListInput

	ManagedResourceListInput
	RegionalFilterListInput
}

type LoadbalancerDetails struct {
	apis.VirtualResourceDetails
	VpcResourceInfo
	ZoneResourceInfoBase
	NetworkResourceInfoBase

	SLoadbalancer

	// 公网IP地址
	Eip string `json:"eip"`

	// 公网IP地址类型: 弹性、非弹性
	// example: public_ip
	EipMode string `json:"eip_mode"`

	// 后端服务器组名称
	BackendGroup string `json:"backend_group"`
}

type LoadbalancerResourceInfo struct {
	// 负载均衡名称
	Loadbalancer string `json:"loadbalancer"`

	// VPC ID
	VpcId string `json:"vpc_id"`

	VpcResourceInfo

	// 可用区ID
	ZoneId string `json:"zone_id"`

	ZoneResourceInfoBase
}

type LoadbalancerFilterListInput struct {
	VpcFilterListInput

	ZonalFilterListBase

	// 负载均衡名称
	Loadbalancer string `json:"loadbalancer"`

	// swagger:ignore
	// Deprecated
	LoadbalancerId string `json:"loadbalancer_id" deprecated-by:"loadbalancer"`

	// 以负载均衡名称排序
	OrderByLoadbalancer string `json:"order_by_loadbalancer"`
}
