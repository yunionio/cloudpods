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
	VPC_STATUS_PENDING       = compute.VPC_STATUS_PENDING
	VPC_STATUS_AVAILABLE     = compute.VPC_STATUS_AVAILABLE
	VPC_STATUS_UNAVAILABLE   = compute.VPC_STATUS_UNAVAILABLE
	VPC_STATUS_FAILED        = compute.VPC_STATUS_FAILED
	VPC_STATUS_START_DELETE  = "start_delete"
	VPC_STATUS_DELETING      = compute.VPC_STATUS_DELETING
	VPC_STATUS_DELETE_FAILED = "delete_failed"
	VPC_STATUS_DELETED       = "deleted"
	VPC_STATUS_UNKNOWN       = compute.VPC_STATUS_UNKNOWN

	MAX_VPC_PER_REGION = 3

	DEFAULT_VPC_ID = compute.DEFAULT_VPC_ID
	NORMAL_VPC_ID  = compute.NORMAL_VPC_ID // 没有关联VPC的安全组，统一使用normal

	CLASSIC_VPC_NAME = "-"
)

type UsableResourceListInput struct {
	// filter by network usability of the resource
	Usable *bool `json:"usable"`
}

type UsableVpcResourceListInput struct {
	// filter by Vpc usability of the resource
	UsableVpc *bool `json:"usable_vpc"`
}

type VpcListInput struct {
	apis.EnabledStatusInfrasResourceBaseListInput
	apis.ExternalizedResourceBaseListInput

	ManagedResourceListInput
	RegionalFilterListInput
	GlobalVpcResourceListInput

	DnsZoneFilterListBase

	InterVpcNetworkFilterListBase
	// 过滤可以加入指定vpc互联的vpc
	UsableForInterVpcNetworkId string `json:"usable_for_inter_vpc_network_id"`

	UsableResourceListInput
	UsableVpcResourceListInput

	// 过滤vpc底下有指定zone的ip子网
	ZoneId string `json:"zone_id"`

	// filter by globalvpc
	Globalvpc string `json:"globalvpc"`

	// 是否是默认VPC
	// example: true
	IsDefault *bool `json:"is_default"`

	// CIDR地址段
	// example: 192.168.222.0/24
	CidrBlock []string `json:"cidr_block"`

	// enmu: eip, none
	ExternalAccessMode string `json:"external_access_mode"`

	// 按子网数量排序
	// pattern:asc|desc
	OrderByNetworkCount string `json:"order_by_network_count"`
	// 按二层网络数量排序
	// pattern:asc|desc
	OrderByWireCount string `json:""order_by_wire_count`
}

const (
	VPC_PROVIDER_OVN = "ovn"
)
