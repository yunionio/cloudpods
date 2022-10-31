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
	VPC_PEERING_CONNECTION_STATUS_CREATING       = compute.VPC_PEERING_CONNECTION_STATUS_CREATING
	VPC_PEERING_CONNECTION_STATUS_CREATE_FAILED  = "create_failed"
	VPC_PEERING_CONNECTION_STATUS_DELETE_FAILED  = "delete_failed"
	VPC_PEERING_CONNECTION_STATUS_PENDING_ACCEPT = compute.VPC_PEERING_CONNECTION_STATUS_PENDING_ACCEPT
	VPC_PEERING_CONNECTION_STATUS_ACTIVE         = compute.VPC_PEERING_CONNECTION_STATUS_ACTIVE
	VPC_PEERING_CONNECTION_STATUS_DELETING       = compute.VPC_PEERING_CONNECTION_STATUS_DELETING
	VPC_PEERING_CONNECTION_STATUS_UNKNOWN        = compute.VPC_PEERING_CONNECTION_STATUS_UNKNOWN
)

type VpcPeeringConnectionDetails struct {
	apis.EnabledStatusInfrasResourceBaseDetails
	VpcResourceInfo

	PeerVpcName string
}

type VpcPeeringConnectionCreateInput struct {
	apis.EnabledStatusInfrasResourceBaseCreateInput
	SVpcResourceBase
	PeerVpcId string
	//跨区域vpc对等连接带宽，仅对腾讯云有效
	//单位Mbps,可选值 10,20,50,100,200,500,1000
	Bandwidth int
}

type VpcPeeringConnectionListInput struct {
	apis.EnabledStatusInfrasResourceBaseListInput
	apis.ExternalizedResourceBaseListInput
	VpcFilterListInput
	PeerVpcId string
}

type VpcPeeringConnectionUpdateInput struct {
	apis.EnabledStatusInfrasResourceBaseUpdateInput
}
