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

package secrules

import (
	"bytes"
	"fmt"
	"net"
	"sort"

	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
)

type securityRuleCut struct {
	r SecurityRule

	protocolCut bool
	netCut      bool
	portCut     bool

	v4ranges []netutils.IPV4AddrRange
	v6ranges []netutils.IPV6AddrRange
}

func (src *securityRuleCut) String() string {
	s := fmt.Sprintf("[%s;v4=%s;v6=%s;protocolCut=%v;netCut=%v;portCut=%v]",
		src.r.String(),
		netutils.IPV4AddrRangeList(src.v4ranges).String(),
		netutils.IPV6AddrRangeList(src.v6ranges).String(),
		src.protocolCut, src.netCut, src.portCut)
	return s
}

func (src *securityRuleCut) isCut() bool {
	return src.protocolCut && src.netCut && src.portCut
}

func (src securityRuleCut) genRules() []SecurityRule {
	src.v4ranges = netutils.IPV4AddrRangeList(src.v4ranges).Merge()
	src.v6ranges = netutils.IPV6AddrRangeList(src.v6ranges).Merge()

	rs := make([]SecurityRule, 0)

	if len(src.v4ranges) == 1 && src.v4ranges[0].IsAll() && len(src.v6ranges) == 1 && src.v6ranges[0].IsAll() {
		rule := src.r
		rule.IPNet = nil
		rs = append(rs, rule)
		return rs
	}
	for i := range src.v4ranges {
		nets := src.v4ranges[i].ToIPNets()
		for _, net := range nets {
			rule := src.r
			rule.IPNet = net
			rs = append(rs, rule)
		}
	}
	for i := range src.v6ranges {
		nets := src.v6ranges[i].ToIPNets()
		for _, net := range nets {
			rule := src.r
			rule.IPNet = net
			rs = append(rs, rule)
		}
	}
	return rs
}

func newSecurityRuleSetCuts(r SecurityRule) securityRuleCuts {
	var v4ranges []netutils.IPV4AddrRange
	var v6ranges []netutils.IPV6AddrRange
	if r.IPNet == nil {
		// expand
		v4ranges = append(v4ranges, netutils.AllIPV4AddrRange)
		v6ranges = append(v6ranges, netutils.AllIPV6AddrRange)
	} else {
		if regutils.MatchCIDR(r.IPNet.String()) {
			v4ranges = append(v4ranges, netutils.NewIPV4AddrRangeFromIPNet(r.IPNet))
		} else {
			v6ranges = append(v6ranges, netutils.NewIPV6AddrRangeFromIPNet(r.IPNet))
		}
	}
	return []securityRuleCut{
		{
			r:        r,
			v4ranges: v4ranges,
			v6ranges: v6ranges,
		},
	}
}

type securityRuleCuts []securityRuleCut

func (srcs securityRuleCuts) String() string {
	buf := bytes.Buffer{}
	for i := range srcs {
		s := srcs[i].String()
		buf.WriteString(s)
		buf.WriteString("\n")
	}
	return buf.String()
}

func (srcs securityRuleCuts) securityRuleSet() SecurityRuleSet {
	srs := SecurityRuleSet{}
	for i := range srcs {
		src := &srcs[i]
		if src.isCut() {
			continue
		}
		srs = append(srs, src.genRules()...)
	}
	return srs
}

func (srcs securityRuleCuts) cutOutProtocol(protocol string) securityRuleCuts {
	r := securityRuleCuts{}
	for _, src := range srcs {
		sr := src.r
		if sr.Protocol == protocol {
			// cut
			src.protocolCut = true
			r = append(r, src)
		} else if sr.Protocol == PROTO_ANY {
			for _, p := range protocolsSupported {
				src_ := src
				src_.r.Protocol = p
				if p == protocol {
					src_.protocolCut = true
				}
				r = append(r, src_)
			}
		} else if protocol == PROTO_ANY {
			// cut
			src.protocolCut = true
			r = append(r, src)
		} else {
			// retain
			r = append(r, src)
		}
	}
	return r
}

func isV6(n *net.IPNet) bool {
	return regutils.MatchCIDR6(n.String())
}

func (srcs securityRuleCuts) cutOutIPNet(protocol string, n *net.IPNet) securityRuleCuts {
	r := securityRuleCuts{}
	isWildMatch := isWildNet(n)
	isV6 := false
	var v4n netutils.IPV4AddrRange
	var v6n netutils.IPV6AddrRange
	if !isWildMatch {
		if regutils.MatchCIDR6(n.String()) {
			isV6 = true
			v6n = netutils.NewIPV6AddrRangeFromIPNet(n)
		} else {
			v4n = netutils.NewIPV4AddrRangeFromIPNet(n)
		}
	}
	for i := range srcs {
		src := srcs[i]
		if src.netCut {
			r = append(r, src)
			continue
		}
		if src.r.Protocol != protocol && protocol != PROTO_ANY {
			r = append(r, src)
			continue
		}
		if isWildMatch {
			src.netCut = true
			r = append(r, src)
			continue
		}
		if isV6 {
			src.v6ranges = netutils.IPV6AddrRangeList(src.v6ranges).Substract(v6n)
		} else {
			src.v4ranges = netutils.IPV4AddrRangeList(src.v4ranges).Substract(v4n)
		}
		r = append(r, src)
	}
	return r
}

func (srcs securityRuleCuts) cutOutPortRange(protocol string, portStart, portEnd uint16) securityRuleCuts {
	pr1 := &portRange{
		start: portStart,
		end:   portEnd,
	}
	r := securityRuleCuts{}
	for _, src := range srcs {
		if src.r.Protocol != protocol {
			src_ := src
			r = append(r, src_)
			continue
		}
		sr := src.r
		if len(sr.Ports) > 0 {
			ps := newPortsFromInts(sr.Ports...)
			left, sub := ps.substractPortRange(pr1)
			if len(left) > 0 {
				src_ := src
				src_.r.Ports = left.IntSlice()
				r = append(r, src_)
			}
			if len(sub) > 0 {
				src_ := src
				src_.r.Ports = left.IntSlice()
				src_.portCut = true
				r = append(r, src_)
			}
		} else if sr.PortStart > 0 && sr.PortEnd > 0 {
			pr := newPortRange(uint16(sr.PortStart), uint16(sr.PortEnd))
			left, sub := pr.substractPortRange(pr1)
			for _, l := range left {
				src_ := src
				src_.r.PortStart = int(l.start)
				src_.r.PortEnd = int(l.end)
				r = append(r, src_)
			}
			if sub != nil && sub.count() > 0 {
				src_ := src
				src_.r.PortStart = int(sub.start)
				src_.r.PortEnd = int(sub.end)
				src_.portCut = true
				r = append(r, src_)
			}
		} else {
			{
				nns := [][2]int32{
					[2]int32{1, int32(portStart) - 1},
					[2]int32{int32(portEnd) + 1, 65535},
				}
				for _, nn := range nns {
					if nn[0] <= nn[1] {
						src_ := src
						src_.r.PortStart = int(nn[0])
						src_.r.PortEnd = int(nn[1])
						r = append(r, src_)
					}
				}
			}
			{
				src_ := src
				src_.r.PortStart = int(portStart)
				src_.r.PortEnd = int(portEnd)
				src_.portCut = true
				r = append(r, src_)
			}
		}
	}
	return r
}

func (srcs securityRuleCuts) cutOutPorts(protocol string, ps1 []uint16) securityRuleCuts {
	r := securityRuleCuts{}
	for _, src := range srcs {
		if src.r.Protocol != protocol {
			src_ := src
			r = append(r, src_)
			continue
		}
		sr := src.r
		if len(sr.Ports) > 0 {
			ps0 := newPortsFromInts(sr.Ports...)
			left, sub := ps0.substractPorts(ps1)
			if len(left) > 0 {
				src_ := src
				src_.r.Ports = left.IntSlice()
				r = append(r, src_)
			}
			if len(sub) > 0 {
				src_ := src
				src_.r.Ports = sub.IntSlice()
				src_.portCut = true
				r = append(r, src_)
			}
		} else if sr.PortStart > 0 && sr.PortEnd > 0 {
			pr := newPortRange(uint16(sr.PortStart), uint16(sr.PortEnd))
			ps := ports(ps1)
			left, sub := pr.substractPorts(ps)
			for _, l := range left {
				src_ := src
				src_.r.PortStart = int(l.start)
				src_.r.PortEnd = int(l.end)
				r = append(r, src_)
			}
			if len(sub) > 0 {
				src_ := src
				src_.r.Ports = sub.IntSlice()
				src_.r.PortStart = 0
				src_.r.PortEnd = 0
				src_.portCut = true
				r = append(r, src_)
			}
		} else {
			sort.Slice(ps1, func(i, j int) bool {
				return ps1[i] < ps1[j]
			})
			add := func(s, e uint16) {
				src_ := src
				src_.r.PortStart = int(s)
				src_.r.PortEnd = int(e)
				r = append(r, src_)
			}
			s := uint16(1)
			for _, p := range ps1 {
				if s <= p-1 {
					add(s, p-1)
					s = p + 1
				}
			}
			if s != 0 && s <= 65535 {
				add(s, 65535)
			}
			{
				src_ := src
				src_.r.Ports = ports(ps1).IntSlice()
				src_.portCut = true
				r = append(r, src_)
			}
		}
	}
	return r
}

func (srcs securityRuleCuts) cutOutPortsAll() securityRuleCuts {
	r := securityRuleCuts{}
	for _, src := range srcs {
		src_ := src
		src_.portCut = true
		r = append(r, src_)
	}
	return r
}
