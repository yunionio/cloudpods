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

package types

import (
	"net"

	"yunion.io/x/pkg/util/netutils"
)

type SNic struct {
	Type    string   `json:"nic_type"`
	Domain  string   `json:"domain"`
	Wire    string   `json:"wire"`
	IpAddr  string   `json:"ip_addr"`
	WireId  string   `json:"wire_id"`
	NetId   string   `json:"net_id"`
	Rate    int64    `json:"rate"`
	Mtu     int64    `json:"mtu"`
	Mac     string   `json:"mac"`
	Dns     string   `json:"dns"`
	Ntp     string   `json:"ntp"`
	MaskLen int8     `json:"masklen"`
	Net     string   `json:"net"`
	Gateway string   `json:"gateway"`
	LinkUp  bool     `json:"link_up"`
	Routes  []SRoute `json:"routes,omitempty"`
}

type SRoute []string

func (n SNic) GetNetMask() string {
	return netutils.Masklen2Mask(n.MaskLen).String()
}

func (n SNic) GetMac() net.HardwareAddr {
	return getMac(n.Mac)
}

type SServerNic struct {
	Name      string   `json:"name"`
	Index     int      `json:"index"`
	Bridge    string   `json:"bridge"`
	Domain    string   `json:"domain"`
	Ip        string   `json:"ip"`
	Vlan      int      `json:"vlan"`
	Driver    string   `json:"driver"`
	Masklen   int      `json:"masklen"`
	Virtual   bool     `json:"virtual"`
	Manual    bool     `json:"manual"`
	WireId    string   `json:"wire_id"`
	NetId     string   `json:"net_id"`
	Mac       string   `json:"mac"`
	BandWidth int      `json:"bw"`
	Mtu       int      `json:"mtu,omitempty"`
	Dns       string   `json:"dns"`
	Ntp       string   `json:"ntp"`
	Net       string   `json:"net"`
	Interface string   `json:"interface"`
	Gateway   string   `json:"gateway"`
	Ifname    string   `json:"ifname"`
	Routes    []SRoute `json:"routes,omitempty"`
	NicType   string   `json:"nic_type,omitempty"`
	LinkUp    bool     `json:"link_up,omitempty"`
	TeamWith  string   `json:"team_with,omitempty"`

	TeamingMaster *SServerNic   `json:"-"`
	TeamingSlaves []*SServerNic `json:"-"`
}

func (n SServerNic) GetNetMask() string {
	return netutils.Masklen2Mask(int8(n.Masklen)).String()
}

func (n SServerNic) GetMac() net.HardwareAddr {
	return getMac(n.Mac)
}

func (n SServerNic) ToNic() SNic {
	return SNic{
		Type:    n.NicType,
		Domain:  n.Domain,
		IpAddr:  n.Ip,
		WireId:  n.WireId,
		NetId:   n.NetId,
		Mac:     n.Mac,
		Dns:     n.Dns,
		Ntp:     n.Ntp,
		MaskLen: int8(n.Masklen),
		Net:     n.Net,
		Gateway: n.Gateway,
		Routes:  n.Routes,
		LinkUp:  n.LinkUp,
		Mtu:     int64(n.Mtu),
	}
}
