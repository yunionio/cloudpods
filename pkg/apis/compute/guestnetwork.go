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
	"reflect"

	"yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
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

	GuestIpMask int8 `json:"guest_ip_mask"`
	// 网关地址
	GuestGateway string `json:"guest_gateway"`

	GuestIp6Mask uint8 `json:"guest_ip6_mask"`
	// 网关地址
	GuestGateway6 string `json:"guest_gateway6"`
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
	// 端口映射
	PortMappings GuestPortMappings `json:"port_mappings"`

	IsDefault bool `json:"is_default"`
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

	IsDefault    *bool             `json:"is_default"`
	PortMappings GuestPortMappings `json:"port_mappings"`
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

		MappedIp6Addr string `json:"mapped_ip6_addr"`
	} `json:"vpc"`

	Networkaddresses jsonutils.JSONObject `json:"networkaddresses"`

	VirtualIps   []string          `json:"virtual_ips"`
	PortMappings GuestPortMappings `json:"port_mappings"`
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

type GuestPortMappingProtocol string

const (
	GuestPortMappingProtocolTCP GuestPortMappingProtocol = "tcp"
	GuestPortMappingProtocolUDP GuestPortMappingProtocol = "udp"
)

const (
	GUEST_PORT_MAPPING_RANGE_START = 20000
	GUEST_PORT_MAPPING_RANGE_END   = 25000
)

type GuestPortMappingPortRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type GuestPortMappingEnvValueFrom string

const (
	GuestPortMappingEnvValueFromPort     GuestPortMappingEnvValueFrom = "port"
	GuestPortMappingEnvValueFromHostPort GuestPortMappingEnvValueFrom = "host_port"
)

type GuestPortMappingEnv struct {
	Key       string                       `json:"key"`
	ValueFrom GuestPortMappingEnvValueFrom `json:"value_from"`
}

type GuestPortMapping struct {
	Protocol GuestPortMappingProtocol `json:"protocol"`
	// 容器内部 Port 端口范围 1-65535，-1表示由宿主机自动分配和 HostPort 相同的端口
	Port          int                        `json:"port"`
	HostPort      *int                       `json:"host_port,omitempty"`
	HostIp        string                     `json:"host_ip"`
	HostPortRange *GuestPortMappingPortRange `json:"host_port_range,omitempty"`
	// whitelist for remote ips
	RemoteIps []string              `json:"remote_ips"`
	Rule      *GuestPortMappingRule `json:"rule,omitempty"`
	Envs      []GuestPortMappingEnv `json:"envs,omitempty"`
}

type GuestPortMappingRule struct {
	FirstPortOffset *int `json:"first_port_offset"`
}

type GuestPortMappings []*GuestPortMapping

func (g GuestPortMappings) String() string {
	return jsonutils.Marshal(g).String()
}

func (g GuestPortMappings) IsZero() bool {
	return len(g) == 0
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&GuestPortMappings{}), func() gotypes.ISerializable {
		return &GuestPortMappings{}
	})
}
