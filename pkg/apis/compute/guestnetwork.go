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

type GuestnetworkDetails struct {
	GuestJointResourceDetails

	SGuestnetwork

	// IP子网名称
	Network string `json:"network"`
	// 所属Wire
	WireId string `json:"wire_id"`
}

type GuestnetworkShortDesc struct {
	// IP地址
	IpAddr string `json:"ip_addr"`
	// 是否为外网网卡
	// Deprecated
	IsExit bool `json:"is_exit"`
	// IPv6地址
	Ip6Addr string `json:"ip6_addr"`
	// Mac地址
	Mac string `json:"mac"`
	// Bonding的配对网卡MAC
	TeamWith string `json:"team_with"`
	// 所属Vpc
	VpcId string `json:"vpc_id"`
	// 所属Network
	NetworkId string `json:"network_id"`
}

type GuestnetworkListInput struct {
	GuestJointsListInput

	NetworkFilterListInput

	MacAddr []string `json:"mac_addr"`

	IpAddr []string `json:"ip_addr"`

	Ip6Addr []string `json:"ip6_addr"`

	Driver []string `json:"driver"`

	Ifname []string `json:"ifname"`

	TeamWith []string `json:"team_with"`
}

type GuestnetworkUpdateInput struct {
	GuestJointBaseUpdateInput

	Driver string `json:"driver"`

	BwLimit *int `json:"bw_limit"`

	Index *int8 `json:"index"`
}
