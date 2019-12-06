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

type NetworkListInput struct {
	apis.BaseListInput

	apis.SharableVirtualResourceListInput

	CloudaccountListInput
	CloudTypeListInput

	Zone  string   `json:"zone"`
	Zones []string `json:"zones"`

	Vpc         string `json:"vpc"`
	Cloudregion string `json:"cloudregion"`
	Usable      bool   `json:"usable"`
	Host        string `json:"host"`
	City        string `json:"city"`
}

type NetworkCreateInput struct {
	apis.Meta

	IsSystem bool `json:"is_system"`

	// description: network name
	// unique: true
	// required: true
	// example: test-network
	Name string `json:"name"`

	// description: network description
	// required: false
	// example: test create network
	Description string `json:"description"`

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
}

type NetworkDetails struct {
	apis.Meta
	apis.SharableVirtualResourceDetails

	CloudproviderDetails
	SNetwork
	Wire         string `json:"wire"`
	Exit         bool   `json:"exit"`
	Ports        int    `json:"ports"`
	PortsUsed    int    `json:"ports_used"`
	Vnics        int    `json:"vnics"`
	BmVnics      int    `json:"bm_nics"`
	LbVnics      int    `json:"lb_vnics"`
	EipVnics     int    `json:"eip_vnics"`
	GroupVnics   int    `json:"group_vnics"`
	ReserveVnics int    `json:"reserve_vnics"`
	Vpc          string `json:"vpc"`
	VpcId        string `json:"vpc_id"`
	VpcExtId     string `json:"vpc_ext_id"`

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

type NetworkStatusInput struct {
	apis.Meta

	// description: network status
	// required: true
	// example: available
	// enum: available,unavailable
	Status string `json:"status"`
}
