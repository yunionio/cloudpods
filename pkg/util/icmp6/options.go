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
	"encoding/binary"
	"net"
)

type ICMP6OptionType int

const (
	ICMP6OptMTU              ICMP6OptionType = 1
	ICMP6OptPrefixInfo       ICMP6OptionType = 2
	ICMP6OptLinkLayerAddress ICMP6OptionType = 3
	ICMP6OptRouteInfo        ICMP6OptionType = 24
)

type ICMP6Option struct {
	Type ICMP6OptionType
	Data []byte
}

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

func DecodeIcmpV6OptMtu(data []byte) uint32 {
	return binary.BigEndian.Uint32(data[2:])
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

func DecodeIcmpv6OptPrefixInfo(data []byte) SPrefixInfoOption {
	rawInfo := icmpv6OptPrefixInfo{
		PrefixLen:         data[0],
		Flag:              data[1],
		ValidLifetime:     binary.BigEndian.Uint32(data[2:6]),
		PreferredLifetime: binary.BigEndian.Uint32(data[6:10]),
		Reserved2:         binary.BigEndian.Uint32(data[10:14]),
		Prefix:            data[14:],
	}
	return SPrefixInfoOption{
		IsOnlink:          rawInfo.Flag&0x80 != 0,
		IsAutoconf:        rawInfo.Flag&0x40 != 0,
		Prefix:            net.IP(rawInfo.Prefix),
		PrefixLen:         rawInfo.PrefixLen,
		ValidLifetime:     rawInfo.ValidLifetime,
		PreferredLifetime: rawInfo.PreferredLifetime,
	}
}

func NewIcmpv6OptPrefixInfo(prefInfo SPrefixInfoOption) icmpv6OptPrefixInfo {
	flag := uint8(0)
	if prefInfo.IsOnlink {
		flag |= 0x80
	}
	if prefInfo.IsAutoconf {
		flag |= 0x40
	}
	return icmpv6OptPrefixInfo{
		PrefixLen:         prefInfo.PrefixLen,
		Flag:              flag,
		ValidLifetime:     prefInfo.ValidLifetime,
		PreferredLifetime: prefInfo.PreferredLifetime,
		Reserved2:         0,
		Prefix:            prefInfo.Prefix.To16(),
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

func DecodeIcmpv6OptSourceTargetAddress(data []byte) net.HardwareAddr {
	return net.HardwareAddr(data[0:6])
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

func DecodeIcmpv6OptRouteInfo(data []byte) SRouteInfoOption {
	rawInfo := icmpv6OptRouteInfo{
		PrefixLen:     data[0],
		Reserved1:     data[1],
		RouteLifetime: binary.BigEndian.Uint32(data[2:6]),
		Prefix:        data[6:],
	}
	for i := len(rawInfo.Prefix); i < 16; i++ {
		rawInfo.Prefix = append(rawInfo.Prefix, 0)
	}
	var preference TPreference
	switch rawInfo.Reserved1 {
	case 0xc0:
		preference = PreferenceLow
	case 0x0:
		preference = PreferenceMedium
	case 0x40:
		preference = PreferenceHigh
	}
	return SRouteInfoOption{
		RouteLifetime: rawInfo.RouteLifetime,
		Prefix:        net.IP(rawInfo.Prefix[:16]),
		PrefixLen:     rawInfo.PrefixLen,
		Preference:    preference,
	}
}

func NewIcmpv6OptRouteInfo(routeInfo SRouteInfoOption) icmpv6OptRouteInfo {
	prefix := routeInfo.Prefix.To16()
	prefixLen := routeInfo.PrefixLen
	routeLifetime := routeInfo.RouteLifetime
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
