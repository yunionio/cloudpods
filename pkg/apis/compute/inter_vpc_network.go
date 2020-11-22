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

const (
	INTER_VPC_NETWORK_STATUS_AVAILABLE          = "available"
	INTER_VPC_NETWORK_STATUS_CREATING           = "creating"
	INTER_VPC_NETWORK_STATUS_CREATE_FAILED      = "create_failed"
	INTER_VPC_NETWORK_STATUS_DELETE_FAILED      = "delete_failed"
	INTER_VPC_NETWORK_STATUS_DELETING           = "deleting"
	INTER_VPC_NETWORK_STATUS_ACTIVE             = "active"
	INTER_VPC_NETWORK_STATUS_ADDVPC             = "add_vpc"
	INTER_VPC_NETWORK_STATUS_ADDVPC_FAILED      = "add_vpc_failed"
	INTER_VPC_NETWORK_STATUS_REMOVEVPC          = "remove_vpc"
	INTER_VPC_NETWORK_STATUS_REMOVEVPC_FAILED   = "remove_vpc_failed"
	INTER_VPC_NETWORK_STATUS_UPDATEROUTE        = "update_route"
	INTER_VPC_NETWORK_STATUS_UPDATEROUTE_FAILED = "update_route_failed"
	INTER_VPC_NETWORK_STATUS_UNKNOWN            = "unknown"
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
	VpcId string
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
}
