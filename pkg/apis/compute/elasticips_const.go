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
)

const (
	EIP_MODE_INSTANCE_PUBLICIP = "public_ip"
	EIP_MODE_STANDALONE_EIP    = "elastic_ip"

	EIP_ASSOCIATE_TYPE_SERVER       = "server"
	EIP_ASSOCIATE_TYPE_NAT_GATEWAY  = "natgateway"
	EIP_ASSOCIATE_TYPE_LOADBALANCER = "loadbalancer"
	EIP_ASSOCIATE_TYPE_UNKNOWN      = "unknown"

	EIP_STATUS_READY           = "ready"
	EIP_STATUS_UNKNOWN         = "unknown"
	EIP_STATUS_ALLOCATE        = "allocate"
	EIP_STATUS_ALLOCATE_FAIL   = "allocate_fail"
	EIP_STATUS_DEALLOCATE      = "deallocate"
	EIP_STATUS_DEALLOCATE_FAIL = "deallocate_fail"
	EIP_STATUS_ASSOCIATE       = "associate"
	EIP_STATUS_ASSOCIATE_FAIL  = "associate_fail"
	EIP_STATUS_DISSOCIATE      = "dissociate"
	EIP_STATUS_DISSOCIATE_FAIL = "dissociate_fail"

	EIP_STATUS_CHANGE_BANDWIDTH = "change_bandwidth"

	EIP_CHARGE_TYPE_BY_TRAFFIC   = "traffic"
	EIP_CHARGE_TYPE_BY_BANDWIDTH = "bandwidth"

	INSTANCE_ASSOCIATE_EIP         = "associate_eip"
	INSTANCE_ASSOCIATE_EIP_FAILED  = "associate_eip_failed"
	INSTANCE_DISSOCIATE_EIP        = "dissociate_eip"
	INSTANCE_DISSOCIATE_EIP_FAILED = "dissociate_eip_failed"
)

var (
	EIP_ASSOCIATE_VALID_TYPES = []string{EIP_ASSOCIATE_TYPE_SERVER, EIP_ASSOCIATE_TYPE_NAT_GATEWAY}
)

type ElasticipListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput

	ManagedResourceListInput
	RegionalFilterListInput
	UsableResourceListInput

	// filter usable eip for given associate type
	// enmu: server, natgateway
	UsableEipForAssociateType string `json:"usable_eip_for_associate_type"`
	// filter usable eip for given associate id
	UsableEipForAssociateId string `json:"usable_eip_for_associate_id"`

	NetworkFilterListBase

	// 标识弹性或非弹性
	// | Mode       | 说明       |
	// |------------|------------|
	// | public_ip  | 公网IP     |
	// | elastic_ip | 弹性公公网网IP |
	//
	// example: elastic_ip
	Mode string `json:"mode"`

	// IP地址
	IpAddr string `json:"ip_addr"`

	// 绑定资源类型
	AssociateType string `json:"associate_type"`

	// 绑定资源Id
	AssociateId string `json:"associate_id"`

	// 计费类型: 流量、带宽
	// example: bandwidth
	ChargeType []string `json:"charge_type"`

	// 目前只有华为云此字段是必需填写的
	BgpType []string `json:"bgp_type"`

	// 是否跟随主机删除而自动释放
	AutoDellocate *bool `json:"auto_dellocate"`
}
