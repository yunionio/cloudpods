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

package netutils2

import (
	"net"
	"strings"

	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
)

func str2ipRange(cidr string) (*netutils.IPV4AddrRange, *netutils.IPV6AddrRange) {
	if regutils.MatchCIDR(cidr) {
		v4prefix, err := netutils.NewIPV4Prefix(cidr)
		if err != nil {
			return nil, nil
		}
		v4range := v4prefix.ToIPRange()
		return &v4range, nil
	} else if regutils.MatchCIDR6(cidr) {
		v6prefix, err := netutils.NewIPV6Prefix(cidr)
		if err != nil {
			return nil, nil
		}
		v6range := v6prefix.ToIPRange()
		return nil, &v6range
	} else if regutils.MatchIP4Addr(cidr) {
		v4addr, err := netutils.NewIPV4Addr(cidr)
		if err != nil {
			return nil, nil
		}
		v4range := netutils.NewIPV4AddrRange(v4addr, v4addr)
		return &v4range, nil
	} else if regutils.MatchIP6Addr(cidr) {
		v6addr, err := netutils.NewIPV6Addr(cidr)
		if err != nil {
			return nil, nil
		}
		v6range := netutils.NewIPV6AddrRange(v6addr, v6addr)
		return nil, &v6range
	}
	return nil, nil
}

func str2ipRangeList(cidr string) ([]netutils.IPV4AddrRange, []netutils.IPV6AddrRange) {
	v4ranges := []netutils.IPV4AddrRange{}
	v6ranges := []netutils.IPV6AddrRange{}
	if strings.Contains(cidr, ",") {
		cidrStrs := strings.Split(cidr, ",")
		for _, cidrStr := range cidrStrs {
			cidrStr = strings.TrimSpace(cidrStr)
			v4range, v6range := str2ipRange(cidrStr)
			if v4range != nil {
				v4ranges = append(v4ranges, *v4range)
			}
			if v6range != nil {
				v6ranges = append(v6ranges, *v6range)
			}
		}
	} else {
		v4range, v6range := str2ipRange(cidr)
		if v4range != nil {
			v4ranges = append(v4ranges, *v4range)
		}
		if v6range != nil {
			v6ranges = append(v6ranges, *v6range)
		}
	}
	if len(v4ranges) > 0 {
		v4ranges = netutils.IPV4AddrRangeList(v4ranges).Merge()
	}
	if len(v6ranges) > 0 {
		v6ranges = netutils.IPV6AddrRangeList(v6ranges).Merge()
	}
	return v4ranges, v6ranges
}

func Str2IPNets(cidr string) []*net.IPNet {
	v4ranges, v6ranges := str2ipRangeList(cidr)
	ipnets := []*net.IPNet{}
	for i := range v4ranges {
		v4nets := v4ranges[i].ToIPNets()
		ipnets = append(ipnets, v4nets...)
	}
	for i := range v6ranges {
		v6nets := v6ranges[i].ToIPNets()
		ipnets = append(ipnets, v6nets...)
	}
	return ipnets
}
