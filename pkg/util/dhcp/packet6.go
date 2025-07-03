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

// DHCPv6 https://datatracker.ietf.org/doc/html/rfc8415
/*
	0                   1                   2                   3
	0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|    msg-type   |               transaction-id                  |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|                                                               |
	.                            options                            .
	.                 (variable number and length)                  .
	|                                                               |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/
/*
    0                   1                   2                   3
    0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |    msg-type   |   hop-count   |                               |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+                               |
   |                                                               |
   |                         link-address                          |
   |                                                               |
   |                               +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-|
   |                               |                               |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+                               |
   |                                                               |
   |                         peer-address                          |
   |                                                               |
   |                               +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-|
   |                               |                               |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+                               |
   .                                                               .
   .            options (variable number and length)   ....        .
   |                                                               |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/

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

type OptionCode6 uint16

const (
	DHCPV6_OPTION_CLIENTID                 OptionCode6 = 1
	DHCPV6_OPTION_SERVERID                 OptionCode6 = 2
	DHCPV6_OPTION_IA_NA                    OptionCode6 = 3
	DHCPV6_OPTION_IA_TA                    OptionCode6 = 4
	DHCPV6_OPTION_IAADDR                   OptionCode6 = 5
	DHCPV6_OPTION_ORO                      OptionCode6 = 6
	DHCPV6_OPTION_PREFERENCE               OptionCode6 = 7
	DHCPV6_OPTION_ELAPSED_TIME             OptionCode6 = 8
	DHCPV6_OPTION_RELAY_MSG                OptionCode6 = 9
	DHCPV6_OPTION_AUTH                     OptionCode6 = 11
	DHCPV6_OPTION_UNICAST                  OptionCode6 = 12
	DHCPV6_OPTION_STATUS_CODE              OptionCode6 = 13
	DHCPV6_OPTION_RAPID_COMMIT             OptionCode6 = 14
	DHCPV6_OPTION_USER_CLASS               OptionCode6 = 15
	DHCPV6_OPTION_VENDOR_CLASS             OptionCode6 = 16
	DHCPV6_OPTION_VENDOR_OPTS              OptionCode6 = 17
	DHCPV6_OPTION_INTERFACE_ID             OptionCode6 = 18
	DHCPV6_OPTION_RECONF_MSG               OptionCode6 = 19
	DHCPV6_OPTION_RECONF_ACCEPT            OptionCode6 = 20
	DHCPV6_OPTION_IA_PD                    OptionCode6 = 25
	DHCPV6_OPTION_IAPREFIX                 OptionCode6 = 26
	DHCPV6_OPTION_INFORMATION_REFRESH_TIME OptionCode6 = 32
	DHCPV6_OPTION_SOL_MAX_RT               OptionCode6 = 82
	DHCPV6_OPTION_INF_MAX_RT               OptionCode6 = 83
)

// DHCPv6 message type
func (p Packet) Type6() MessageType {
	return MessageType(p[0])
}

// DHCPv6 transaction ID
func (p Packet) TID() uint32 {
	return binary.BigEndian.Uint32([]byte{0, p[1], p[2], p[3]})
}

// DHCPv6 hop Count for relay message
func (p Packet) HopCount() byte {
	return p[1]
}

// DHCPv6 link address for relay message
func (p Packet) LinkAddr() net.IP {
	return net.IP(p[2:18])
}

// DHCPv6 peer address for relay message
func (p Packet) PeerAddr() net.IP {
	return net.IP(p[18:34])
}

func (p Packet) SetType6(hType MessageType) { p[0] = byte(hType) }

func (p Packet) SetTID(tid uint32) {
	tidBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(tidBytes, tid)
	p[1] = tidBytes[1]
	p[2] = tidBytes[2]
	p[3] = tidBytes[3]
}

func (p Packet) SetHopCount(hops byte) {
	p[1] = hops
}

func (p Packet) SetLinkAddr(linkAddr net.IP) {
	copy(p[2:18], linkAddr)
}

func (p Packet) SetPeerAddr(peerAddr net.IP) {
	copy(p[18:34], peerAddr)
}

func NewPacket6(opCode MessageType, tid uint32) Packet {
	p := make(Packet, 4)
	p.SetType6(opCode)
	p.SetTID(tid)
	return p
}

type Option6 struct {
	Code  OptionCode6
	Value []byte
}

// Appends a DHCP option to the end of a packet
/*
    0                   1                   2                   3
    0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |          option-code          |           option-len          |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |                          option-data                          |
   |                      (option-len octets)                      |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/
func (p *Packet) AddOption6(o Option6) {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(o.Code))
	*p = append(*p, buf...)
	binary.BigEndian.PutUint16(buf, uint16(len(o.Value)))
	*p = append(*p, buf...)
	*p = append(*p, o.Value...)
}

// Creates a request packet that a Client would send to a server.
func RequestPacket6(mt MessageType, tid uint32, options []Option6) Packet {
	p := NewPacket6(mt, tid)
	for _, o := range options {
		p.AddOption6(o)
	}
	return p
}
