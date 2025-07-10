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

package icmp6

import (
	"fmt"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"yunion.io/x/pkg/errors"
)

type TPreference int

const (
	PreferenceLow    TPreference = 0
	PreferenceMedium TPreference = 1
	PreferenceHigh   TPreference = 2
)

func (pref TPreference) String() string {
	switch pref {
	case PreferenceLow:
		return "low"
	case PreferenceMedium:
		return "medium"
	case PreferenceHigh:
		return "high"
	}
	return "unknown"
}

type SBaseICMP6Message struct {
	SrcMac net.HardwareAddr
	SrcIP  net.IP
	DstMac net.HardwareAddr
	DstIP  net.IP
}

func (msg SBaseICMP6Message) String() string {
	return fmt.Sprintf("%s[%s]->%s[%s]", msg.SrcIP.String(), msg.SrcMac.String(), msg.DstIP.String(), msg.DstMac.String())
}

type IICMP6Message interface {
	Payload() gopacket.SerializableLayer
	ICMP6TypeCode() layers.ICMPv6TypeCode

	SourceMac() net.HardwareAddr
	SourceIP() net.IP
	DestinationMac() net.HardwareAddr
	DestinationIP() net.IP
}

func (msg SBaseICMP6Message) SourceIP() net.IP {
	return msg.SrcIP
}

func (msg SBaseICMP6Message) SourceMac() net.HardwareAddr {
	return msg.SrcMac
}

func (msg SBaseICMP6Message) DestinationIP() net.IP {
	return msg.DstIP
}

func (msg SBaseICMP6Message) DestinationMac() net.HardwareAddr {
	return msg.DstMac
}

type SRouterSolicitation struct {
	SBaseICMP6Message
}

type SPrefixInfoOption struct {
	IsOnlink          bool
	IsAutoconf        bool
	Prefix            net.IP
	PrefixLen         uint8
	ValidLifetime     uint32
	PreferredLifetime uint32
}

func (opt SPrefixInfoOption) String() string {
	return fmt.Sprintf("%s/%d (IsOnlink: %t, IsAutoconf: %t) ValidLifetime: %d, PreferredLifetime: %d", opt.Prefix.String(), opt.PrefixLen, opt.IsOnlink, opt.IsAutoconf, opt.ValidLifetime, opt.PreferredLifetime)
}

type SRouteInfoOption struct {
	RouteLifetime uint32
	Prefix        net.IP
	PrefixLen     uint8
	Preference    TPreference
}

func (opt SRouteInfoOption) String() string {
	return fmt.Sprintf("%s/%d (Pref: %s) RouteLifetime: %d", opt.Prefix.String(), opt.PrefixLen, opt.Preference.String(), opt.RouteLifetime)
}

type SRouterAdvertisement struct {
	SBaseICMP6Message

	CurHopLimit uint8

	IsManaged   bool
	IsOther     bool
	IsHomeAgent bool

	Preference TPreference

	RouterLifetime uint16
	ReachableTime  uint32
	RetransTimer   uint32

	MTU        uint32
	PrefixInfo []SPrefixInfoOption
	RouteInfo  []SRouteInfoOption
}

type SNeighborSolicitation struct {
	SBaseICMP6Message

	/*
		The IP address of the target of the solicitation.
		It MUST NOT be a multicast address.
	*/
	TargetAddr net.IP
}

type SNeighborAdvertisement struct {
	SBaseICMP6Message

	/*
		For solicited advertisements, the Target Address
		field in the Neighbor Solicitation message that
		prompted this advertisement.  For an unsolicited
		advertisement, the address whose link-layer address
		has changed.  The Target Address MUST NOT be a
		multicast address.
	*/
	TargetAddr net.IP

	IsRouter    bool
	IsSolicited bool
}

func (msg SNeighborSolicitation) ICMP6TypeCode() layers.ICMPv6TypeCode {
	return layers.CreateICMPv6TypeCode(layers.ICMPv6TypeNeighborSolicitation, 0)
}

func (msg SNeighborSolicitation) Payload() gopacket.SerializableLayer {
	pkt := layers.ICMPv6NeighborSolicitation{}
	pkt.TargetAddress = msg.TargetAddr

	pkt.Options = layers.ICMPv6Options{
		layers.ICMPv6Option{
			Type: layers.ICMPv6OptSourceAddress,
			Data: NewIcmpv6OptSourceTargetAddress(msg.SrcMac).Bytes(),
		},
	}

	return &pkt
}

func (msg *SNeighborSolicitation) Unmarshal(data *layers.ICMPv6NeighborSolicitation) error {
	msg.TargetAddr = data.TargetAddress
	return nil
}

func (msg SNeighborAdvertisement) ICMP6TypeCode() layers.ICMPv6TypeCode {
	return layers.CreateICMPv6TypeCode(layers.ICMPv6TypeNeighborAdvertisement, 0)
}

func (msg SNeighborAdvertisement) Payload() gopacket.SerializableLayer {
	pkt := layers.ICMPv6NeighborAdvertisement{}
	pkt.TargetAddress = msg.TargetAddr
	pkt.Flags = 0x00
	if msg.IsRouter {
		pkt.Flags |= 0x80
	}
	if msg.IsSolicited {
		pkt.Flags |= 0x40
	}

	pkt.Options = layers.ICMPv6Options{
		layers.ICMPv6Option{
			Type: layers.ICMPv6OptSourceAddress,
			Data: NewIcmpv6OptSourceTargetAddress(msg.SrcMac).Bytes(),
		},
	}

	return &pkt
}

func (msg *SNeighborAdvertisement) Unmarshal(data *layers.ICMPv6NeighborAdvertisement) error {
	msg.TargetAddr = data.TargetAddress
	msg.IsRouter = data.Flags&0x80 != 0
	msg.IsSolicited = data.Flags&0x40 != 0
	return nil
}

func (msg SRouterSolicitation) ICMP6TypeCode() layers.ICMPv6TypeCode {
	return layers.CreateICMPv6TypeCode(layers.ICMPv6TypeRouterSolicitation, 0)
}

func (msg SRouterSolicitation) Payload() gopacket.SerializableLayer {
	pkt := layers.ICMPv6RouterSolicitation{}

	pkt.Options = layers.ICMPv6Options{
		layers.ICMPv6Option{
			Type: layers.ICMPv6OptSourceAddress,
			Data: NewIcmpv6OptSourceTargetAddress(msg.SrcMac).Bytes(),
		},
	}

	return &pkt
}

func (msg *SRouterSolicitation) Unmarshal(data *layers.ICMPv6RouterSolicitation) error {
	return nil
}

func (msg SRouterAdvertisement) ICMP6TypeCode() layers.ICMPv6TypeCode {
	return layers.CreateICMPv6TypeCode(layers.ICMPv6TypeRouterAdvertisement, 0)
}

func (msg SRouterAdvertisement) Payload() gopacket.SerializableLayer {
	pkt := layers.ICMPv6RouterAdvertisement{}
	// https://datatracker.ietf.org/doc/html/rfc4861#section-4.2
	// https://datatracker.ietf.org/doc/html/rfc4191
	pkt.Flags = 0x00 // M=1, O=1, stateful DHCPv6, pref=00
	if msg.IsManaged {
		pkt.Flags |= 0x80
	}
	if msg.IsOther {
		pkt.Flags |= 0x40
	}
	if msg.IsHomeAgent {
		pkt.Flags |= 0x20
	}

	switch msg.Preference {
	case PreferenceLow:
		pkt.Flags |= 0x18
	case PreferenceMedium:
		// do nothing
	case PreferenceHigh:
		pkt.Flags |= 0x08
	}

	pkt.HopLimit = msg.CurHopLimit
	pkt.RouterLifetime = msg.RouterLifetime
	pkt.ReachableTime = msg.ReachableTime
	pkt.RetransTimer = msg.RetransTimer

	pkt.Options = layers.ICMPv6Options{
		layers.ICMPv6Option{
			Type: layers.ICMPv6OptSourceAddress,
			Data: NewIcmpv6OptSourceTargetAddress(msg.SrcMac).Bytes(),
		},
		layers.ICMPv6Option{
			Type: layers.ICMPv6OptMTU,
			Data: NewIcmpV6OptMtu(msg.MTU).Bytes(),
		},
	}
	for i := range msg.PrefixInfo {
		pref := msg.PrefixInfo[i]
		pkt.Options = append(pkt.Options, layers.ICMPv6Option{
			Type: layers.ICMPv6OptPrefixInfo,
			Data: NewIcmpv6OptPrefixInfo(pref).Bytes(),
		})
	}
	for i := range msg.RouteInfo {
		rt := msg.RouteInfo[i]
		pkt.Options = append(pkt.Options, layers.ICMPv6Option{
			Type: 24,
			Data: NewIcmpv6OptRouteInfo(rt).Bytes(),
		})
	}

	return &pkt
}

func (msg *SRouterAdvertisement) Unmarshal(data *layers.ICMPv6RouterAdvertisement) error {
	msg.CurHopLimit = data.HopLimit
	msg.IsManaged = data.Flags&0x80 != 0
	msg.IsOther = data.Flags&0x40 != 0
	msg.IsHomeAgent = data.Flags&0x20 != 0
	switch data.Flags & 0x18 {
	case 0x18:
		msg.Preference = PreferenceLow
	case 0x08:
		msg.Preference = PreferenceHigh
	default:
		msg.Preference = PreferenceMedium
	}
	msg.RouterLifetime = data.RouterLifetime

	for i := range data.Options {
		opt := data.Options[i]
		switch opt.Type {
		case layers.ICMPv6OptMTU:
			msg.MTU = DecodeIcmpV6OptMtu(opt.Data)
		case layers.ICMPv6OptPrefixInfo:
			prefix := DecodeIcmpv6OptPrefixInfo(opt.Data)
			msg.PrefixInfo = append(msg.PrefixInfo, prefix)
		case 24:
			route := DecodeIcmpv6OptRouteInfo(opt.Data)
			msg.RouteInfo = append(msg.RouteInfo, route)
		}
	}

	return nil
}

func (msg SRouterAdvertisement) String() string {
	return fmt.Sprintf(`RouterAdvertisement %s 
CurHopLimit: %d, IsManaged: %t, IsOther: %t, IsHomeAgent: %t, Preference: %s, RouterLifetime: %d, ReachableTime: %d, RetransTimer: %d
MTU: %d
PrefixInfo: %v
RouteInfo: %v)`,
		msg.SBaseICMP6Message.String(),
		msg.CurHopLimit, msg.IsManaged, msg.IsOther, msg.IsHomeAgent, msg.Preference.String(), msg.RouterLifetime, msg.ReachableTime, msg.RetransTimer,
		msg.MTU, msg.PrefixInfo, msg.RouteInfo)
}

func EncodePacket(msg IICMP6Message) ([]byte, error) {
	var eth = &layers.Ethernet{
		EthernetType: layers.EthernetTypeIPv6,
		SrcMAC:       msg.SourceMac(),
		DstMAC:       msg.DestinationMac(),
	}

	var ip = &layers.IPv6{
		Version:    6,
		HopLimit:   64,
		SrcIP:      msg.SourceIP(),
		DstIP:      msg.DestinationIP(),
		NextHeader: layers.IPProtocolICMPv6,
	}

	var icmp6 = &layers.ICMPv6{
		TypeCode: msg.ICMP6TypeCode(),
	}
	icmp6.SetNetworkLayerForChecksum(ip)

	var payload = msg.Payload()

	var (
		buf  = gopacket.NewSerializeBuffer()
		opts = gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}
	)
	if err := gopacket.SerializeLayers(buf, opts, eth, ip, icmp6, payload); err != nil {
		return nil, errors.Wrap(err, "SerializeLayers error")
	}

	return buf.Bytes(), nil
}

func DecodePacket(data []byte) (IICMP6Message, error) {
	packet := gopacket.NewPacket(data, layers.LayerTypeEthernet, gopacket.Default)
	if packet.ErrorLayer() != nil {
		return nil, errors.Wrap(packet.ErrorLayer().Error(), "DecodePacket error")
	}

	var baseMsg SBaseICMP6Message

	{
		ethLayer := packet.Layer(layers.LayerTypeEthernet)
		if ethLayer != nil {
			eth := ethLayer.(*layers.Ethernet)
			baseMsg.SrcMac = eth.SrcMAC
			baseMsg.DstMac = eth.DstMAC
		} else {
			return nil, errors.Wrap(packet.ErrorLayer().Error(), "Expect Ethernet layer")
		}
	}

	{
		ipLayer := packet.Layer(layers.LayerTypeIPv6)
		if ipLayer != nil {
			// ipv6
			ip6 := ipLayer.(*layers.IPv6)
			baseMsg.SrcIP = ip6.SrcIP
			baseMsg.DstIP = ip6.DstIP
		} else {
			return nil, errors.Wrap(packet.ErrorLayer().Error(), "Expect IPv6 packet")
		}
	}

	{
		icmpLayer := packet.Layer(layers.LayerTypeICMPv6)
		if icmpLayer != nil {
			icmp6 := icmpLayer.(*layers.ICMPv6)
			if icmp6 == nil {
				return nil, errors.Wrap(packet.ErrorLayer().Error(), "Expect IPv6 packet")
			}

			switch icmp6.TypeCode.Type() {
			case 133:
				payload := packet.Layer(layers.LayerTypeICMPv6RouterSolicitation)
				if payload != nil {
					msg := &SRouterSolicitation{
						SBaseICMP6Message: baseMsg,
					}
					return msg, nil
				} else {
					return nil, errors.Wrap(packet.ErrorLayer().Error(), "Expect ICMPv6 Router Solicitation packet")
				}
			case 134:
				payload := packet.Layer(layers.LayerTypeICMPv6RouterAdvertisement)
				if payload != nil {
					msg := &SRouterAdvertisement{
						SBaseICMP6Message: baseMsg,
					}
					return msg, nil
				} else {
					return nil, errors.Wrap(packet.ErrorLayer().Error(), "Expect ICMPv6 Router Advertisement packet")
				}
			case 135:
				payload := packet.Layer(layers.LayerTypeICMPv6NeighborSolicitation)
				if payload != nil {
					msg := &SNeighborSolicitation{
						SBaseICMP6Message: baseMsg,
					}
					return msg, nil
				} else {
					return nil, errors.Wrap(packet.ErrorLayer().Error(), "Expect ICMPv6 Neighbor Solicitation packet")
				}
			case 136:
				payload := packet.Layer(layers.LayerTypeICMPv6NeighborAdvertisement)
				if payload != nil {
					msg := &SNeighborAdvertisement{
						SBaseICMP6Message: baseMsg,
					}
					return msg, nil
				} else {
					return nil, errors.Wrap(packet.ErrorLayer().Error(), "Expect ICMPv6 Neighbor Advertisement packet")
				}
			default:
				return nil, errors.Wrap(packet.ErrorLayer().Error(), "Expect ICMPv6 packet")
			}
		} else {
			return nil, errors.Wrap(packet.ErrorLayer().Error(), "Expect IPv6 packet")
		}
	}
}
