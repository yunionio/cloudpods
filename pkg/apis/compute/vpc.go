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

type VpcUsage struct {
	// 二层网络数量
	// example: 1
	WireCount int `json:"wire_count"`
	// IP子网个数
	// example: 2
	NetworkCount int `json:"network_count"`
	// 路由表个数
	// example: 0
	RoutetableCount int `json:"routetable_count"`
	// NAT网关个数
	// example: 0
	NatgatewayCount int `json:"natgateway_count"`

	// DnsZone个数
	// example: 2
	DnsZoneCount int `json:"dns_zone_count"`

	RequestVpcPeerCount int `json:"request_vpc_peer_count"`
	AcceptVpcPeerCount  int `json:"accpet_vpc_peer_count"`
}

type VpcDetails struct {
	apis.EnabledStatusInfrasResourceBaseDetails
	ManagedResourceInfo
	CloudregionResourceInfo
	GlobalVpcResourceInfo

	SVpc

	VpcUsage
}

type VpcResourceInfoBase struct {
	// Vpc Name
	Vpc string `json:"vpc"`

	// VPC外部Id
	VpcExtId string `json:"vpc_ext_id"`
}

type VpcResourceInfo struct {
	VpcResourceInfoBase

	// VPC归属区域ID
	CloudregionId string `json:"cloudregion_id"`

	CloudregionResourceInfo

	// VPC归属云订阅ID
	ManagerId string `json:"manager_id"`

	ManagedResourceInfo
}

type VpcSyncstatusInput struct {
}

type VpcCreateInput struct {
	apis.EnabledStatusInfrasResourceBaseCreateInput

	CloudregionResourceInput

	CloudproviderResourceInput

	// CIDR_BLOCK
	CidrBlock string `json:"cidr_block"`

	// 仅对谷歌云有用，若谷歌云订阅只有一个全局VPC，此参数可不传
	// 若有多个全局VPC，谷歌云需要指定其中一个全局VPC
	GlobalvpcId string `json:"globalvpc_id"`

	// Vpc外网访问模式
	ExternalAccessMode string `json:"external_access_mode"`
}

type VpcUpdateInput struct {
	apis.EnabledStatusInfrasResourceBaseUpdateInput

	// Vpc外网访问模式
	ExternalAccessMode string `json:"external_access_mode"`
}

type VpcResourceInput struct {
	// 关联VPC(ID或Name)
	VpcId string `json:"vpc_id"`
	// swagger:ignore
	// Deprecated
	// filter by vpc Id
	Vpc string `json:"vpc" yunion-deprecated-by:"vpc_id"`

	// Vpc外网访问模式
	ExternalAccessMode string `json:"external_access_mode"`
}

type VpcFilterListInputBase struct {
	VpcResourceInput

	// 按VPC名称排序
	// pattern:asc|desc
	OrderByVpc string `json:"order_by_vpc"`
}

type VpcFilterListInput struct {
	VpcFilterListInputBase
	RegionalFilterListInput
	ManagedResourceListInput
}

type VpcTopologyInput struct {
}

type NetworkTopologyOutput struct {
	Name         string                `json:"name"`
	Status       string                `json:"status"`
	GuestIpStart string                `json:"guest_ip_start"`
	GuestIpEnd   string                `json:"guest_ip_end"`
	GuestIpMask  int8                  `json:"guest_ip_mask"`
	ServerType   string                `json:"server_type"`
	VlanId       int                   `json:"vlan_id"`
	Address      []SNetworkUsedAddress `json:"address"`
}

type HostnetworkTopologyOutput struct {
	IpAddr  string `json:"ip_addr"`
	MacAddr string `json:"mac_addr"`
}

type StorageShortDesc struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Enabled     bool   `json:"enabled"`
	StorageType string `json:"storage_type"`
	CapacityMb  int64  `json:"capacity_mb"`
}

type HostTopologyOutput struct {
	Name       string                      `json:"name"`
	Id         string                      `json:"id"`
	Status     string                      `json:"status"`
	HostStatus string                      `json:"host_status"`
	HostType   string                      `json:"host_type"`
	Networks   []HostnetworkTopologyOutput `json:"networks"`
	Schedtags  []SchedtagShortDescDetails  `json:"schedtags"`
	Storages   []StorageShortDesc          `json:"storages"`
}

type WireTopologyOutput struct {
	Name      string                  `json:"name"`
	Status    string                  `json:"status"`
	Bandwidth int                     `json:"bandwidth"`
	Zone      string                  `json:"zone"`
	Networks  []NetworkTopologyOutput `json:"networks"`
	Hosts     []HostTopologyOutput    `json:"hosts"`
}

type VpcTopologyOutput struct {
	Name   string               `json:"name"`
	Status string               `json:"status"`
	Wires  []WireTopologyOutput `json:"wires"`
}
