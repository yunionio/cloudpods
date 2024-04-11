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
	"yunion.io/x/jsonutils"
)

type GuestnetworkDetails struct {
	GuestJointResourceDetails

	SGuestnetwork

	// IP子网名称
	Network string `json:"network"`
	// 所属Wire
	WireId string `json:"wire_id"`

	// EipAddr associate with this guestnetwork
	EipAddr string `json:"eip_addr"`

	NetworkAddresses []NetworkAddrConf `json:"network_addresses"`
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
	// 附属IP
	SubIps string `json:"sub_ips"`
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

	IsDefault *bool `json:"is_default"`
}

type GuestnetworkBaseDesc struct {
	Net            string               `json:"net"`
	NetId          string               `json:"net_id"`
	Mac            string               `json:"mac"`
	Virtual        bool                 `json:"virtual"`
	Ip             string               `json:"ip"`
	Gateway        string               `json:"gateway"`
	Dns            string               `json:"dns"`
	Domain         string               `json:"domain"`
	Ntp            string               `json:"ntp"`
	Routes         jsonutils.JSONObject `json:"routes"`
	Ifname         string               `json:"ifname"`
	Masklen        int8                 `json:"masklen"`
	Vlan           int                  `json:"vlan"`
	Bw             int                  `json:"bw"`
	Mtu            int16                `json:"mtu"`
	Index          int                  `json:"index"`
	RxTrafficLimit int64                `json:"rx_traffic_limit"`
	TxTrafficLimit int64                `json:"tx_traffic_limit"`
	NicType        compute.TNicType     `json:"nic_type"`

	Ip6      string `json:"ip6"`
	Gateway6 string `json:"gateway6"`
	Masklen6 uint8  `json:"masklen6"`

	// 是否为缺省路由网关
	IsDefault bool `json:"is_default"`

	Bridge    string `json:"bridge"`
	WireId    string `json:"wire_id"`
	Interface string `json:"interface"`

	Vpc struct {
		Id           string `json:"id"`
		Provider     string `json:"provider"`
		MappedIpAddr string `json:"mapped_ip_addr"`
	} `json:"vpc"`

	Networkaddresses jsonutils.JSONObject `json:"networkaddresses"`

	VirtualIps []string `json:"virtual_ips"`
}

type GuestnetworkJsonDesc struct {
	GuestnetworkBaseDesc

	Driver    string `json:"driver"`
	NumQueues int    `json:"num_queues"`
	Vectors   *int   `json:"vectors"`

	ExternalId string `json:"external_id"`
	TeamWith   string `json:"team_with"`
	Manual     *bool  `json:"manual"`

	UpscriptPath   string `json:"upscript_path"`
	DownscriptPath string `json:"downscript_path"`

	// baremetal
	Rate        int    `json:"rate"`
	BaremetalId string `json:"baremetal_id"`

	LinkUp bool `json:"link_up"`
}

type SNicTrafficRecord struct {
	RxTraffic int64
	TxTraffic int64

	HasBeenSetDown bool
}
