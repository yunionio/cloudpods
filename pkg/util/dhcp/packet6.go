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
	"fmt"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

/*
https://datatracker.ietf.org/doc/html/rfc4861#section-4.6.4
0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     Type      |    Length     |           Reserved            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                              MTU                              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/
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
https://datatracker.ietf.org/doc/html/rfc4861#section-4.6.2
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
	Prefix            []byte
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
	_, ipnet, _ := net.ParseCIDR(fmt.Sprintf("%s/%d", gw.String(), preflen))
	return icmpv6OptPrefixInfo{
		PrefixLen:         preflen,
		Flag:              0x80,
		ValidLifetime:     0xffffffff,
		PreferredLifetime: 0xffffffff,
		Reserved2:         0,
		Prefix:            ipnet.IP,
	}
}

/*
https://datatracker.ietf.org/doc/html/rfc4861#section-4.6.1
0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     Type      |    Length     |    Link-Layer Address ...
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/
type icmpv6OptSourceTargetAddress struct {
	Mac net.HardwareAddr
}

func (opt icmpv6OptSourceTargetAddress) Bytes() []byte {
	return opt.Mac[0:6]
}

func NewIcmpv6OptSourceTargetAddress(macAddr net.HardwareAddr) icmpv6OptSourceTargetAddress {
	return icmpv6OptSourceTargetAddress{
		Mac: macAddr,
	}
}

/*
https://datatracker.ietf.org/doc/html/rfc4191#section-2.3
0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     Type      |    Length     | Prefix Length |Resvd|Prf|Resvd|
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                        Route Lifetime                         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                   Prefix (Variable Length)                    |
.                                                               .
.                                                               .
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/

type icmpv6OptRouteInfo struct {
	PrefixLen     uint8
	Reserved1     uint8
	RouteLifetime uint32
	Prefix        []byte
}

func (opt icmpv6OptRouteInfo) Bytes() []byte {
	buf := make([]byte, 6)
	buf[0] = opt.PrefixLen
	buf[1] = opt.Reserved1
	binary.BigEndian.PutUint32(buf[2:6], opt.RouteLifetime)
	buf = append(buf, opt.Prefix...)
	return buf
}

func NewIcmpv6OptRouteInfo(routeLifetime uint32, prefix []byte, prefixLen uint8) icmpv6OptRouteInfo {
	if prefixLen <= 0 {
		prefix = prefix[0:0]
	} else if prefixLen <= 64 {
		prefix = prefix[0:8]
	} else {
		prefix = prefix[0:16]
	}
	return icmpv6OptRouteInfo{
		PrefixLen:     prefixLen,
		Reserved1:     0,
		RouteLifetime: routeLifetime,
		Prefix:        prefix,
	}
}

func MakeRouterAdverPacket(gwIP net.IP, prefixLen uint8, routes6 []SRouteInfo, mtu uint32) (Packet, net.IP, net.HardwareAddr, error) {
	pkt := layers.ICMPv6RouterAdvertisement{}
	// https://datatracker.ietf.org/doc/html/rfc4861#section-4.2
	// https://datatracker.ietf.org/doc/html/rfc4191
	pkt.Flags = 0xc0 // M=1, O=1, stateful DHCPv6, pref=00
	pkt.HopLimit = 64
	pkt.RouterLifetime = 9000
	pkt.ReachableTime = 0
	pkt.RetransTimer = 0

	// gwMac, _ := net.ParseMAC("14:09:dc:cf:16:d5")

	pkt.Options = layers.ICMPv6Options{
		/*layers.ICMPv6Option{
			Type: layers.ICMPv6OptSourceAddress,
			Data: NewIcmpv6OptSourceTargetAddress(gwMac).Bytes(),
		},*/
		layers.ICMPv6Option{
			Type: layers.ICMPv6OptMTU,
			Data: NewIcmpV6OptMtu(mtu).Bytes(),
		},
		layers.ICMPv6Option{
			Type: layers.ICMPv6OptPrefixInfo,
			Data: NewIcmpv6OptPrefixInfo(gwIP, prefixLen).Bytes(),
		},
	}
	for i := range routes6 {
		rt := routes6[i]
		pkt.Options = append(pkt.Options, layers.ICMPv6Option{
			Type: 24,
			Data: NewIcmpv6OptRouteInfo(9000, rt.Prefix.To16(), rt.PrefixLen).Bytes(),
		})
	}

	sbf := gopacket.NewSerializeBuffer()
	err := pkt.SerializeTo(sbf, gopacket.SerializeOptions{})
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "SerializeTo")
	}
	return sbf.Bytes(), gwIP, nil, nil
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

	// https://www.rfc-editor.org/rfc/rfc3646
	OPTION_DNS_SERVERS OptionCode6 = 23
	OPTION_DOMAIN_LIST OptionCode6 = 24
	// https://www.rfc-editor.org/rfc/rfc4075
	OPTION_SNTP_SERVERS OptionCode6 = 31
	//https://www.rfc-editor.org/rfc/rfc5908
	OPTION_NTP_SERVERS6 OptionCode6 = 56
	// https://www.rfc-editor.org/rfc/rfc4833
	OPTION_NEW_POSIX_TIMEZONE OptionCode6 = 41
	OPTION_NEW_TZDB_TIMEZONE  OptionCode6 = 42
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

func (p Packet) IsRelayMsg() bool {
	return p.Type6() == DHCPV6_RELAY_FORW || p.Type6() == DHCPV6_RELAY_REPL
}

func (p *Packet) SetType6(hType MessageType) {
	(*p)[0] = byte(hType)
}

func (p *Packet) SetTID(tid uint32) {
	tidBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(tidBytes, tid)
	(*p)[1] = tidBytes[1]
	(*p)[2] = tidBytes[2]
	(*p)[3] = tidBytes[3]
}

func (p *Packet) SetHopCount(hops byte) {
	(*p)[1] = hops
}

func (p *Packet) SetLinkAddr(linkAddr net.IP) {
	copy((*p)[2:18], linkAddr)
}

func (p *Packet) SetPeerAddr(peerAddr net.IP) {
	copy((*p)[18:34], peerAddr)
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
func optionToBytes(o Option6) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint16(buf[0:2], uint16(o.Code))
	binary.BigEndian.PutUint16(buf[2:4], uint16(len(o.Value)))
	buf = append(buf, o.Value...)
	return buf
}

func optionsToBytes(opts []Option6) []byte {
	buf := make([]byte, 0)
	for i := range opts {
		buf = append(buf, optionToBytes(opts[i])...)
	}
	return buf
}

func (p Packet) GetOption6s() []Option6 {
	options := make([]Option6, 0)
	offset := 4
	if p.IsRelayMsg() {
		offset = 18
	}
	i := offset
	for i < len(p) {
		code := binary.BigEndian.Uint16(p[i : i+2])
		length := binary.BigEndian.Uint16(p[i+2 : i+4])
		options = append(options, Option6{
			Code:  OptionCode6(code),
			Value: p[i+4 : i+4+int(length)],
		})
		i += 4 + int(length)
	}
	return options
}

// Creates a request packet that a Client would send to a server.
/*func RequestPacket6(mt MessageType, tid uint32, options []Option6) Packet {
	p := NewPacket6(mt, tid)
	for _, o := range options {
		p.AddOption6(o)
	}
	return p
}*/

func MakeDHCP6Reply(pkt Packet, conf *ResponseConfig) (Packet, error) {
	var msgType MessageType
	pktType := pkt.Type6()
	switch pktType {
	case DHCPV6_SOLICIT:
		msgType = DHCPV6_ADVERTISE
	case DHCPV6_REQUEST:
		msgType = DHCPV6_REPLY
	case DHCPV6_CONFIRM:
		msgType = DHCPV6_REPLY
	case DHCPV6_RENEW:
		msgType = DHCPV6_REPLY
	case DHCPV6_REBIND:
		msgType = DHCPV6_REPLY
	case DHCPV6_INFORMATION_REQUEST:
		msgType = DHCPV6_REPLY
	default:
		return nil, errors.Wrapf(errors.ErrNotSupported, "unsupported message type %d", pktType)
	}

	return makeDHCPReplyPacket6(pkt, conf, msgType), nil
}

const (
	DUID_TYPE_LINK_LAYER_ADDRESS = 3
	DUID_HARDWARE_TYPE_ETHERNET  = 1
)

func makeServerId(serverMac net.HardwareAddr) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint16(buf[0:2], uint16(DUID_TYPE_LINK_LAYER_ADDRESS))
	binary.BigEndian.PutUint16(buf[2:4], uint16(DUID_HARDWARE_TYPE_ETHERNET))
	buf = append(buf, serverMac[:6]...)
	return buf
}

func makeIAAddr(ip net.IP, preferLT, validLT uint32, opts []Option6) []byte {
	buf := make([]byte, 24)
	copy(buf[0:16], ip.To16())
	binary.BigEndian.PutUint32(buf[16:20], preferLT)
	binary.BigEndian.PutUint32(buf[20:24], validLT)
	buf = append(buf, optionsToBytes(opts)...)
	return buf
}

func responseIANA(buf []byte, opts []Option6) []byte {
	log.Debugf("responseIANA buf %d", len(buf))
	if len(buf) > 12 {
		buf = buf[:12]
	}
	iaID := binary.BigEndian.Uint32(buf[0:4])
	t1 := binary.BigEndian.Uint32(buf[4:8])
	t2 := binary.BigEndian.Uint32(buf[8:12])
	log.Debugf("responseIANA IA_NA IAID %d t1 %d t2 %d", iaID, t1, t2)
	buf = append(buf, optionsToBytes(opts)...)
	return buf
}

func makeIPv6s(ips []net.IP) []byte {
	buf := make([]byte, 0)
	for _, ip := range ips {
		ip6 := ip.To16()
		if ip6 == nil {
			continue
		}
		buf = append(buf, ip6...)
	}
	return buf
}

func makeDHCPReplyPacket6(pkt Packet, conf *ResponseConfig, msgType MessageType) Packet {
	log.Debugf("makeDHCPReplyPacket6 msgType %d tid %x", msgType, pkt.TID())

	resp := NewPacket6(msgType, pkt.TID())
	originOpts := pkt.GetOption6s()
	getOption := func(code OptionCode6) Option6 {
		for _, o := range originOpts {
			if o.Code == code {
				return o
			}
		}
		return Option6{}
	}

	options := make([]Option6, 0)

	// copy clientID
	options = append(options, getOption(DHCPV6_OPTION_CLIENTID))
	// serverID
	options = append(options, Option6{
		Code:  DHCPV6_OPTION_SERVERID,
		Value: makeServerId(conf.InterfaceMac),
	})
	// Identity Association for Non-temporary Addresses Option
	ianaOpt := getOption(DHCPV6_OPTION_IA_NA)
	options = append(options, Option6{
		Code: DHCPV6_OPTION_IA_NA,
		Value: responseIANA(ianaOpt.Value, []Option6{
			{
				Code: DHCPV6_OPTION_IAADDR,
				Value: makeIAAddr(conf.ClientIP6, uint32(conf.LeaseTime.Seconds()), uint32(conf.LeaseTime.Seconds()), []Option6{
					{
						Code:  DHCPV6_OPTION_STATUS_CODE,
						Value: []byte{0, 0, 'S', 'u', 'c', 'c', 'e', 's', 's'},
					},
				}),
			},
		}),
	})

	if len(conf.DNSServers6) > 0 {
		options = append(options, Option6{
			Code:  OPTION_DNS_SERVERS,
			Value: makeIPv6s(conf.DNSServers6),
		})
	}

	if len(conf.NTPServers6) > 0 {
		options = append(options, Option6{
			Code:  OPTION_SNTP_SERVERS,
			Value: makeIPv6s(conf.NTPServers6),
		})
	}

	resp = append(resp, optionsToBytes(options)...)

	return resp
}
