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
	NETWORK_TYPE_VPC     = "vpc"
	NETWORK_TYPE_CLASSIC = "classic"
)

type WireResourceInput struct {
	// 二层网络(ID或Name)的资源
	WireId string `json:"wire_id"`
	// swagger:ignore
	// Deprecated
	// fitler by wire id
	Wire string `json:"wire" yunion-deprecated-by:"wire_id"`
}

type WireFilterListBase struct {
	WireResourceInput

	// 以二层网络名称排序
	OrderByWire string `json:"order_by_wire"`
}

type WireFilterListInput struct {
	VpcFilterListInput
	ZonalFilterListBase

	WireFilterListBase
}

type NetworkResourceInput struct {
	// IP子网（ID或Name）
	NetworkId string `json:"network_id"`
	// swagger:ignore
	// Deprecated
	// filter by networkId
	Network string `json:"network" yunion-deprecated-by:"network_id"`
}

type NetworkFilterListBase struct {
	NetworkResourceInput

	// 以IP子网的名称排序
	OrderByNetwork string `json:"order_by_network"`
}

type NetworkFilterListInput struct {
	WireFilterListInput
	NetworkFilterListBase
}

type NetworkListInput struct {
	apis.SharableVirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	SchedtagResourceInput
	WireFilterListInput

	HostResourceInput
	StorageResourceInput

	UsableResourceListInput

	// description: Exact matching ip address in network.
	// example: 10.168.222.1
	Ip string `json:"ip"`

	// description: Fuzzy matching ip address in network.
	// example: 10.168.222.1
	IpMatch string `json:"ip_match"`

	IfnameHint []string `json:"ifname_hint"`
	// 起始IP地址
	GuestIpStart []string `json:"guest_ip_start"`
	// 接收IP地址
	GuestIpEnd []string `json:"guest_ip_end"`
	// 掩码
	GuestIpMask []int8 `json:"guest_ip_mask"`
	// 网关地址
	GuestGateway string `json:"guest_gateway"`
	// DNS
	GuestDns []string `json:"guest_dns"`
	// allow multiple dhcp, seperated by ","
	GuestDhcp []string `json:"guest_dhcp"`

	GuestDomain []string `json:"guest_domain"`

	GuestIp6Start []string `json:"guest_ip6_start"`
	GuestIp6End   []string `json:"guest_ip6_end"`
	GuestIp6Mask  []int8   `json:"guest_ip6_mask"`
	GuestGateway6 []string `json:"guest_gateway6"`
	GuestDns6     []string `json:"guest_dns6"`

	GuestDomain6 []string `json:"guest_domain6"`
	// vlanId 1~4096
	VlanId []int `json:"vlan_id"`
	// 服务器类型
	// example: server
	ServerType []string `json:"server_type"`
	// 分配策略
	AllocPolicy []string `json:"alloc_policy"`
	// 是否加入自动分配地址池
	IsAutoAlloc *bool `json:"is_auto_alloc"`
	// 是否为基础网络（underlay）
	IsClassic *bool `json:"is_classic"`

	// filter by Host schedtag
	HostSchedtagId string `json:"host_schedtag_id"`

	// filter by BGP types
	BgpType []string `json:"bgp_type"`
}

type NetworkResourceInfoBase struct {
	// IP子网名称
	Network string `json:"network"`
}

type NetworkResourceInfo struct {
	NetworkResourceInfoBase

	// 二层网络ID
	WireId string `json:"wire_id"`

	WireResourceInfo
}

type NetworkCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	// description: ip range of guest, if not set, you shoud set guest_ip_start,guest_ip_end and guest_ip_mask params
	// example: 10.168.222.1/24
	GuestIpPrefix string `json:"guest_ip_prefix"`

	// description: ip range of guest ip start, if set guest_ip_prefix, this parameter will be useless
	// example: 10.168.222.1
	GuestIpStart string `json:"guest_ip_start"`

	// description: ip range of guest ip end, if set guest_ip_prefix, this parameter will be useless
	// example: 10.168.222.100
	GuestIpEnd string `json:"guest_ip_end"`

	// description: ip range of guest ip mask, if set guest_ip_prefix, this parameter will be useless
	// example: 24
	// maximum: 30
	// minimum: 12
	GuestIpMask int64 `json:"guest_ip_mask"`

	IfnameHint string `json:"ifname_hint"`

	// description: guest gateway
	// example: 192.168.222.1
	GuestGateway string `json:"guest_gateway"`

	// description: guest dns
	// example: 114.114.114.114
	GuestDns string `json:"guest_dns"`

	// description: guest dhcp
	// example: 192.168.222.1,192.168.222.4
	GuestDHCP string `json:"guest_dhcp"`

	// swagger:ignore
	WireId string `json:"wire_id"`

	// description: wire id or name
	Wire string `json:"wire"`

	// description: zone id or name
	Zone string `json:"zone"`

	// description: vpc id or name
	Vpc string `json:"vpc"`

	// description: server type
	// enum: guest,baremetal,pxe,ipmi
	// default: guest
	ServerType string `json:"server_type"`

	// 是否加入自动分配地址池
	IsAutoAlloc *bool `json:"is_auto_alloc"`

	// VlanId
	VlanId *int `json:"vlan_id"`

	// deprecated
	Vlan *int `json:"vlan" yunion-deprecated-by:"vlan_id"`

	// 线路类型
	BgpType string `json:"bgp_type"`
}

type NetworkDetails struct {
	apis.SharableVirtualResourceDetails
	WireResourceInfo

	SNetwork

	// 是否是内网
	Exit bool `json:"exit"`
	// 端口数量
	Ports int `json:"ports"`
	// 已使用端口数量
	PortsUsed int `json:"ports_used"`
	// 网卡数量
	Vnics int `json:"vnics"`
	// 裸金属网卡数量
	BmVnics int `json:"bm_nics"`
	// 负载均衡网卡数量
	LbVnics int `json:"lb_vnics"`
	// 浮动Ip网卡数量
	EipVnics   int `json:"eip_vnics"`
	GroupVnics int `json:"group_vnics"`
	// 预留IP数量
	ReserveVnics int `json:"reserve_vnics"`

	// 路由信息
	Routes    [][]string                 `json:"routes"`
	Schedtags []SchedtagShortDescDetails `json:"schedtags"`
}

type NetworkReserveIpInput struct {
	apis.Meta

	// description: reserved ip list
	// required: true
	// example: [10.168.222.131, 10.168.222.134]
	Ips []string `json:"ips"`

	// description: the comment
	// example: reserve ip for test
	Notes  string `json:"notes"`
	Status string `json:"status"`
	// description: The reserved cycle
	// required: false
	Duration string `json:"duration"`
}

type NetworkReleaseReservedIpInput struct {
	apis.Meta

	// description: IP to be released
	// required: true
	// example: 10.168.222.121
	Ip string `json:"ip"`
}

type NetworkPurgeInput struct {
	apis.Meta
}

type NetworkMergeInput struct {
	apis.Meta

	// description: network id or name to be merged
	// required: true
	// example: test-network
	Target string `json:"target"`
}

type NetworkSplitInput struct {
	apis.Meta

	// description: The middle - separated IP must belong to the network
	// required: true
	// example: 10.168.222.181
	SplitIp string `json:"split_ip"`

	// description: another network name after split
	// required: false
	Name string `json:"name"`
}

type NetworkTryCreateNetworkInput struct {
	apis.Meta

	Ip          string `json:"ip"`
	Mask        int    `json:"mask"`
	ServerType  string `json:"server_type"`
	IsOnPremise bool   `json:"is_on_premise"`
}

type NetworkSyncInput struct {
	apis.Meta
}

type NetworkUpdateInput struct {
	apis.SharableVirtualResourceBaseUpdateInput

	// 起始IP地址
	GuestIpStart string `json:"guest_ip_start"`
	// 接收IP地址
	GuestIpEnd string `json:"guest_ip_end"`
	// 掩码
	GuestIpMask *int8 `json:"guest_ip_mask"`
	// 网关地址
	GuestGateway string `json:"guest_gateway"`
	// DNS
	GuestDns string `json:"guest_dns"`
	// allow multiple dhcp, seperated by ","
	GuestDhcp string `json:"guest_dhcp"`

	GuestDomain string `json:"guest_domain"`

	VlanId *int `json:"vlan_id"`

	// 分配策略
	AllocPolicy string `json:"alloc_policy"`

	// 是否加入自动分配地址池
	IsAutoAlloc *bool `json:"is_auto_alloc"`
}

type GetNetworkAddressesInput struct {
	// 获取资源的范围，例如 project|domain|system
	Scope string `json:"scope"`
}

type GetNetworkAddressesOutput struct {
	// IP子网地址记录
	Addresses []SNetworkUsedAddress `json:"addresses"`
}

type NetworkSetBgpTypeInput struct {
	apis.Meta

	// description: new BgpType name
	// required: true
	// example: ChinaTelecom, BGP, etc.
	BgpType string `json:"bgp_type"`
}
