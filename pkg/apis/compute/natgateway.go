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
	"time"

	"yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	NAT_STAUTS_AVAILABLE             = compute.NAT_STAUTS_AVAILABLE     // 可用
	NAT_STATUS_ALLOCATE              = compute.NAT_STATUS_ALLOCATE      // 创建中
	NAT_STATUS_DEPLOYING             = compute.NAT_STATUS_DEPLOYING     // 配置中
	NAT_STATUS_UNKNOWN               = compute.NAT_STATUS_UNKNOWN       // 未知状态
	NAT_STATUS_CREATE_FAILED         = compute.NAT_STATUS_CREATE_FAILED // 创建失败
	NAT_STATUS_DELETED               = "deleted"                        // 删除
	NAT_STATUS_DELETING              = compute.NAT_STATUS_DELETING      // 删除中
	NAT_STATUS_DELETE_FAILED         = "delete_failed"                  // 删除失败
	NAT_STATUS_SET_AUTO_RENEW        = "set_auto_renew"                 // 设置自动续费中
	NAT_STATUS_SET_AUTO_RENEW_FAILED = "set_auto_renew_failed"          // 设置自动续费失败
	NAT_STATUS_RENEWING              = "renewing"                       // 续费中
	NAT_STATUS_RENEW_FAILED          = "renew_failed"                   // 续费失败
)

type NatGetewayListInput struct {
	apis.StatusInfrasResourceBaseListInput
	apis.ExternalizedResourceBaseListInput
	apis.DeletePreventableResourceBaseListInput

	VpcFilterListInput
}

type NatEntryListInput struct {
	apis.StatusInfrasResourceBaseListInput
	apis.ExternalizedResourceBaseListInput
	NatGatewayFilterListInput
}

type NatDEntryListInput struct {
	NatEntryListInput

	ExternalIP   []string `json:"external_ip"`
	ExternalPort []int    `json:"external_port"`

	InternalIP   []string `json:"internal_ip"`
	InternalPort []int    `json:"internal_port"`
	IpProtocol   []string `json:"ip_protocol"`
}

type NatSEntryListInput struct {
	NatEntryListInput
	NetworkFilterListBase

	IP         []string `json:"ip"`
	SourceCIDR []string `json:"source_cidr"`
}

type NatGatewayResourceInfo struct {
	// NAT网关名称
	Natgateway string `json:"natgateway"`

	// 归属VPC ID
	VpcId string `json:"vpc_id"`

	VpcResourceInfo
}

type NatGatewayResourceInput struct {
	// NAT网关ID or Name
	NatgatewayId string `json:"natgateway_id"`

	// swagger:ignore
	// Deprecated
	Natgateway string `json:"natgateway" yunion-deprecated-by:"natgateway_id"`
}

type NatGatewayFilterListInput struct {
	NatGatewayResourceInput

	// 以NAT网关名字排序
	OrderByNatgateway string `json:"order_by_natgateway"`

	VpcFilterListInput
}

type NatEntryDetails struct {
	apis.StatusInfrasResourceBaseDetails
	NatGatewayResourceInfo
}

type NatGatewaySyncstatusInput struct {
}

type NatgatewayCreateInput struct {
	apis.StatusInfrasResourceBaseCreateInput

	// 包年包月时间周期
	Duration string `json:"duration"`

	// 是否自动续费(仅包年包月时生效)
	// default: false
	AutoRenew bool `json:"auto_renew"`

	// 到期释放时间，仅后付费支持
	ExpiredAt time.Time `json:"expired_at"`

	// 计费方式
	// enum: postpaid, prepaid
	BillingType string `json:"billing_type"`
	// swagger:ignore
	BillingCycle string `json:"billing_cycle"`

	NetworkId string `json:"network_id"`

	// swagger:ignore
	VpcId string `json:"vpc_id"`

	// 绑定已有弹性公网IP，要求EIP必须和Vpc在同一区域
	Eip string `json:"eip"`

	// 绑定新建弹性公网IP
	EipBw int `json:"eip_bw,omitzero"`

	// 弹性公网IP计费类型
	// enum: bandwidth, traffic
	// default: traffic
	EipChargeType    string `json:"eip_charge_type,omitempty"`
	EipBgpType       string `json:"eip_bgp_type"`
	EipAutoDellocate bool   `json:"eip_auto_dellocate"`
}

func (opts *NatgatewayCreateInput) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type NatgatewayDeleteInput struct {
	Force bool `json:"force"`
}
