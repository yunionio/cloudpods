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
	"yunion.io/x/cloudmux/pkg/apis/compute"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	INTER_VPC_NETWORK_STATUS_AVAILABLE          = compute.INTER_VPC_NETWORK_STATUS_AVAILABLE
	INTER_VPC_NETWORK_STATUS_CREATING           = compute.INTER_VPC_NETWORK_STATUS_CREATING
	INTER_VPC_NETWORK_STATUS_CREATE_FAILED      = "create_failed"
	INTER_VPC_NETWORK_STATUS_DELETE_FAILED      = "delete_failed"
	INTER_VPC_NETWORK_STATUS_DELETING           = compute.INTER_VPC_NETWORK_STATUS_DELETING
	INTER_VPC_NETWORK_STATUS_ACTIVE             = "active"
	INTER_VPC_NETWORK_STATUS_ADDVPC             = "add_vpc"
	INTER_VPC_NETWORK_STATUS_ADDVPC_FAILED      = "add_vpc_failed"
	INTER_VPC_NETWORK_STATUS_REMOVEVPC          = "remove_vpc"
	INTER_VPC_NETWORK_STATUS_REMOVEVPC_FAILED   = "remove_vpc_failed"
	INTER_VPC_NETWORK_STATUS_UPDATEROUTE        = "update_route"
	INTER_VPC_NETWORK_STATUS_UPDATEROUTE_FAILED = "update_route_failed"
	INTER_VPC_NETWORK_STATUS_UNKNOWN            = compute.INTER_VPC_NETWORK_STATUS_UNKNOWN
)

type InterVpcNetworkListInput struct {
	apis.EnabledStatusInfrasResourceBaseListInput
	ManagedResourceListInput
}

type InterVpcNetworkCreateInput struct {
	apis.EnabledStatusInfrasResourceBaseCreateInput
	ManagerId string `json:"manager_id"`
}

type InterVpcNetworkUpdateInput struct {
	apis.EnabledStatusInfrasResourceBaseUpdateInput
}

type InterVpcNetworkDetails struct {
	apis.EnabledStatusInfrasResourceBaseDetails
	ManagedResourceInfo
	VpcCount int `json:"vpc_count"`
}

type InterVpcNetworkSyncstatusInput struct {
}

type InterVpcNetworkAddVpcInput struct {
	// 待加入的vpc id
	// vpc和当前vpc互联所必须是同一平台，且运营平台一致，例如aws中国区不能和aws国际区运营平台不一致
	// 可以通过 /vpcs?usable_for_inter_vpc_network_id=<当前vpc互联id> 过滤可以加入的vpc列表
	// required: true
	VpcId string `json:"vpc_id"`
}

type InterVpcNetworkRemoveVpcInput struct {
	VpcId string
}

type InterVpcNetworkFilterListBase struct {
	InterVpcNetworkId string `json:"inter_vpc_network_id"`
}

type InterVpcNetworkManagerListInput struct {
	apis.EnabledStatusInfrasResourceBaseListInput
	ManagedResourceListInput

	// 按关联的vpc数量排序
	// pattern:asc|desc
	OrderByVpcCount string `json:"order_by_vpc_count"`
}
