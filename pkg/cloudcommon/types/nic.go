package types

import (
	"net"

	"yunion.io/x/pkg/util/netutils"
)

const (
	NIC_TYPE_IPMI  = "ipmi"
	NIC_TYPE_ADMIN = "admin"
)

var (
	NIC_TYPES = []string{NIC_TYPE_IPMI, NIC_TYPE_ADMIN}
)

type Nic struct {
	Type    string  `json:"nic_type"`
	Domain  string  `json:"domain"`
	Wire    string  `json:"wire"`
	IpAddr  string  `json:"ip_addr"`
	WireId  string  `json:"wire_id"`
	NetId   string  `json:"net_id"`
	Rate    int64   `json:"rate"`
	Mtu     int64   `json:"mtu"`
	Mac     string  `json:"mac"`
	Dns     string  `json:"dns"`
	MaskLen int8    `json:"masklen"`
	Net     string  `json:"net"`
	Gateway string  `json:"gateway"`
	LinkUp  bool    `json:"link_up"`
	Routes  []Route `json:"routes,omitempty"`
}

func (n Nic) GetNetMask() string {
	return netutils.Masklen2Mask(n.MaskLen).String()
}

func (n Nic) GetMac() net.HardwareAddr {
	return getMac(n.Mac)
}

type Route []string

type ServerNic struct {
	Index     int     `json:"index"`
	Bridge    string  `json:"bridge"`
	Domain    string  `json:"domain"`
	Ip        string  `json:"ip"`
	Vlan      int     `json:"vlan"`
	Driver    string  `json:"driver"`
	Masklen   int     `json:"masklen"`
	Virtual   bool    `json:"virtual"`
	Manual    bool    `json:"manual"`
	WireId    string  `json:"wire_id"`
	NetId     string  `json:"net_id"`
	Mac       string  `json:"mac"`
	BandWidth int     `json:"bw"`
	Dns       string  `json:"dns"`
	Net       string  `json:"net"`
	Interface string  `json:"interface"`
	Gateway   string  `json:"gateway"`
	Ifname    string  `json:"ifname"`
	Routes    []Route `json:"routes,omitempty"`
	NicType   string  `json:"nic_type,omitempty"`
	LinkUp    bool    `json:"link_up,omitempty"`
	Mtu       int64   `json:"mtu,omitempty"`
}

func (n ServerNic) GetNetMask() string {
	return netutils.Masklen2Mask(int8(n.Masklen)).String()
}

func (n ServerNic) GetMac() net.HardwareAddr {
	return getMac(n.Mac)
}

func (n ServerNic) ToNic() Nic {
	return Nic{
		Type:    n.NicType,
		Domain:  n.Domain,
		IpAddr:  n.Ip,
		WireId:  n.WireId,
		NetId:   n.NetId,
		Mac:     n.Mac,
		Dns:     n.Dns,
		MaskLen: int8(n.Masklen),
		Net:     n.Net,
		Gateway: n.Gateway,
		Routes:  n.Routes,
		LinkUp:  n.LinkUp,
		Mtu:     n.Mtu,
	}
}
