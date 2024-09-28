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

package dhcp

import (
	"encoding/binary"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
)

type icmpV6OptMtu struct {
	MTU uint32
}

func (opt icmpV6OptMtu) Bytes() []byte {
	buf := make([]byte, 6)
	binary.BigEndian.PutUint32(buf[2:], opt.MTU)
	return buf
}

func NewIcmpV6OptMtu(mtu uint32) icmpV6OptMtu {
	return icmpV6OptMtu{
		MTU: mtu,
	}
}

/*
0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     Type      |    Length     | Prefix Length |L|A| Reserved1 |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         Valid Lifetime                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                       Preferred Lifetime                      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                           Reserved2                           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+                                                               +
|                                                               |
+                            Prefix                             +
|                                                               |
+                                                               +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/
type icmpv6OptPrefixInfo struct {
	PrefixLen         uint8
	Flag              uint8
	ValidLifetime     uint32
	PreferredLifetime uint32
	Reserved2         uint32
	Prefix            [16]byte
}

func (opt icmpv6OptPrefixInfo) Bytes() []byte {
	buf := make([]byte, 30)
	buf[0] = opt.PrefixLen
	buf[1] = opt.Flag
	binary.BigEndian.PutUint32(buf[2:], opt.ValidLifetime)
	binary.BigEndian.PutUint32(buf[6:], opt.PreferredLifetime)
	binary.BigEndian.PutUint32(buf[10:], opt.Reserved2)
	copy(buf[14:], opt.Prefix[:])
	return buf
}

func NewIcmpv6OptPrefixInfo(gw net.IP, preflen uint8) icmpv6OptPrefixInfo {
	return icmpv6OptPrefixInfo{
		PrefixLen:         preflen,
		Flag:              0x80,
		ValidLifetime:     0xffffffff,
		PreferredLifetime: 0xffffffff,
		Reserved2:         0,
		Prefix:            [16]byte(gw),
	}
}

type icmpv6OptSourceTargetAddress struct {
	Mac netutils.SMacAddr
}

func (opt icmpv6OptSourceTargetAddress) Bytes() []byte {
	return opt.Mac[:]
}

func NewIcmpv6OptSourceTargetAddress(macAddr string) icmpv6OptSourceTargetAddress {
	mac, _ := netutils.ParseMac(macAddr)
	return icmpv6OptSourceTargetAddress{
		Mac: mac,
	}
}

func MakeRouterAdverPacket(gwIP net.IP, preflen uint8, mtu uint32) (Packet, error) {
	pkt := layers.ICMPv6RouterAdvertisement{}
	pkt.Flags = 0x80 // M=1, O=0 stateful DHCPv6

	pkt.Options = layers.ICMPv6Options{
		layers.ICMPv6Option{
			Type: layers.ICMPv6OptMTU,
			Data: NewIcmpV6OptMtu(mtu).Bytes(),
		},
		layers.ICMPv6Option{
			Type: layers.ICMPv6OptPrefixInfo,
			Data: NewIcmpv6OptPrefixInfo(gwIP, preflen).Bytes(),
		},
	}

	sbf := gopacket.NewSerializeBuffer()
	err := pkt.SerializeTo(sbf, gopacket.SerializeOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "SerializeTo")
	}
	return sbf.Bytes(), nil
}

// Creates a request packet that a Client would send to a server.
func RequestPacket6(mt MessageType, chAddr net.HardwareAddr, cIAddr net.IP, xId []byte, broadcast bool, options []Option) Packet {
	p := NewPacket(BootRequest)
	p.SetCHAddr(chAddr)
	p.SetXId(xId)
	if cIAddr != nil {
		p.SetCIAddr(cIAddr)
	}
	p.SetBroadcast(broadcast)
	p.AddOption(OptionDHCPMessageType, []byte{byte(mt)})
	for _, o := range options {
		p.AddOption(o.Code, o.Value)
	}
	p.PadToMinSize()
	return p
}

// ReplyPacket creates a reply packet that a Server would send to a client.
// It uses the req Packet param to copy across common/necessary fields to
// associate the reply the request.
func ReplyPacket6(req Packet, mt MessageType, serverId, yIAddr net.IP, leaseDuration time.Duration, options []Option) Packet {
	p := NewPacket(BootReply)
	p.SetXId(req.XId())
	p.SetFlags(req.Flags())
	p.SetYIAddr(yIAddr)
	p.SetGIAddr(req.GIAddr())
	p.SetCHAddr(req.CHAddr())
	p.AddOption(OptionDHCPMessageType, []byte{byte(mt)})
	p.AddOption(OptionServerIdentifier, []byte(serverId.To4()))
	if leaseDuration > 0 {
		p.AddOption(OptionIPAddressLeaseTime, GetOptTime(leaseDuration))
	}
	for _, o := range options {
		p.AddOption(o.Code, o.Value)
	}
	p.PadToMinSize()
	return p
}

// DHCPv6 https://datatracker.ietf.org/doc/html/rfc8415

// DHCPv6 Message Type
const (
	DHCPV6_SOLICIT             MessageType = 1
	DHCPV6_ADVERTISE           MessageType = 2
	DHCPV6_REQUEST             MessageType = 3
	DHCPV6_CONFIRM             MessageType = 4
	DHCPV6_RENEW               MessageType = 5
	DHCPV6_REBIND              MessageType = 6
	DHCPV6_REPLY               MessageType = 7
	DHCPV6_RELEASE             MessageType = 8
	DHCPV6_DECLINE             MessageType = 9
	DHCPV6_RECONFIGURE         MessageType = 10
	DHCPV6_INFORMATION_REQUEST MessageType = 11
	DHCPV6_RELAY_FORW          MessageType = 12
	DHCPV6_RELAY_REPL          MessageType = 13
)

const (
	DHCPV6_OPTION_CLIENTID                 = 1
	DHCPV6_OPTION_SERVERID                 = 2
	DHCPV6_OPTION_IA_NA                    = 3
	DHCPV6_OPTION_IA_TA                    = 4
	DHCPV6_OPTION_IAADDR                   = 5
	DHCPV6_OPTION_ORO                      = 6
	DHCPV6_OPTION_PREFERENCE               = 7
	DHCPV6_OPTION_ELAPSED_TIME             = 8
	DHCPV6_OPTION_RELAY_MSG                = 9
	DHCPV6_OPTION_AUTH                     = 11
	DHCPV6_OPTION_UNICAST                  = 12
	DHCPV6_OPTION_STATUS_CODE              = 13
	DHCPV6_OPTION_RAPID_COMMIT             = 14
	DHCPV6_OPTION_USER_CLASS               = 15
	DHCPV6_OPTION_VENDOR_CLASS             = 16
	DHCPV6_OPTION_VENDOR_OPTS              = 17
	DHCPV6_OPTION_INTERFACE_ID             = 18
	DHCPV6_OPTION_RECONF_MSG               = 19
	DHCPV6_OPTION_RECONF_ACCEPT            = 20
	DHCPV6_OPTION_IA_PD                    = 25
	DHCPV6_OPTION_IAPREFIX                 = 26
	DHCPV6_OPTION_INFORMATION_REFRESH_TIME = 32
	DHCPV6_OPTION_SOL_MAX_RT               = 82
	DHCPV6_OPTION_INF_MAX_RT               = 83
)

type ResponseConfig6 struct {
	ClientId []byte
	ServerId []byte
}

// ipv6 DHCP message type
func (p Packet) Type6() MessageType {
	return MessageType(p[0])
}

// DHCP transaction ID
func (p Packet) TID() MessageType {
	return MessageType(p[0])
}

func (p Packet) SetType6(hType MessageType) { p[0] = byte(hType) }

func (p Packet) SetCookie(cookie []byte) { copy(p.Cookie(), cookie) }
func (p Packet) SetHops(hops byte)       { p[3] = hops }
func (p Packet) SetXId(xId []byte)       { copy(p.XId(), xId) }

func MakeReplyPacket6(pkt Packet, conf *ResponseConfig6) (Packet, error) {
	msgType := Offer
	if pkt.Type() == Request {
		reqAddr, _ := pkt.ParseOptions().IP(OptionRequestedIPAddress)
		if reqAddr != nil && !conf.ClientIP.Equal(reqAddr) {
			msgType = NAK
		} else {
			msgType = ACK
		}
	}
	return makeDHCPReplyPacket(pkt, conf, msgType), nil
}
