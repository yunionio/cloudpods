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
	"net"
	"sort"

	"yunion.io/x/log"

	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/sortutils"
)

func isWildNet(ipnet *net.IPNet) bool {
	return gotypes.IsNil(ipnet)
}

func compareIPNet(ipnet1, ipnet2 *net.IPNet) sortutils.CompareResult {
	srsIPi := ipnet1.String()
	srsIPj := ipnet2.String()
	if !isWildNet(ipnet1) && !isWildNet(ipnet2) {
		if srsIPi != srsIPj {
			isIPv6i := regutils.MatchCIDR6(srsIPi)
			isIPv6j := regutils.MatchCIDR6(srsIPj)
			if isIPv6i && isIPv6j {
				// compare two ipv6
				v6Rangei := netutils.NewIPV6AddrRangeFromIPNet(ipnet1)
				v6Rangej := netutils.NewIPV6AddrRangeFromIPNet(ipnet2)
				return v6Rangei.Compare(v6Rangej)
			} else if !isIPv6i && !isIPv6j {
				// compare two ipv4
				v4Rangei := netutils.NewIPV4AddrRangeFromIPNet(ipnet1)
				v4Rangej := netutils.NewIPV4AddrRangeFromIPNet(ipnet2)
				return v4Rangei.Compare(v4Rangej)
			} else if isIPv6i && !isIPv6j {
				// v4 first
				return sortutils.More
			} else {
				// if !isIPv6i && isIPv6j {
				// v4 first
				return sortutils.Less
			}
		} else {
			return sortutils.Equal
		}
	} else if isWildNet(ipnet1) && !isWildNet(ipnet2) {
		return sortutils.Less
	} else if isWildNet(ipnet1) && !isWildNet(ipnet2) {
		return sortutils.More
	} else {
		// both wild net, go to next
		return sortutils.Equal
	}
}

func isWildProtocol(protocol string) bool {
	return len(protocol) == 0 || protocol == PROTO_ANY
}

func compareProtocol(protocol1, protocol2 string) sortutils.CompareResult {
	isWild1 := isWildProtocol(protocol1)
	isWild2 := isWildProtocol(protocol1)
	if isWild1 && isWild2 {
		return sortutils.Equal
	} else if isWild1 && !isWild2 {
		return sortutils.Less
	} else if !isWild1 && isWild2 {
		return sortutils.More
	} else {
		return sortutils.CompareString(protocol1, protocol2)
	}
}

type SecurityRuleSet []SecurityRule

func (srs SecurityRuleSet) Len() int {
	return len(srs)
}

func (srs SecurityRuleSet) Swap(i, j int) {
	srs[i], srs[j] = srs[j], srs[i]
}

func (srs SecurityRuleSet) Less(i, j int) bool {
	if srs[i].Priority > srs[j].Priority {
		return true
	} else if srs[i].Priority < srs[j].Priority {
		return false
	}
	// priority equals, compare ipnet
	{
		result := compareIPNet(srs[i].IPNet, srs[j].IPNet)
		switch result {
		case sortutils.Less:
			return true
		case sortutils.More:
			return false
		}
	}
	// compare protocol
	{
		result := compareProtocol(srs[i].Protocol, srs[j].Protocol)
		switch result {
		case sortutils.Less:
			return true
		case sortutils.More:
			return false
		}
	}
	return srs[i].String() < srs[j].String()
}

func (srs SecurityRuleSet) stringList() []string {
	r := make([]string, len(srs))
	for i := range srs {
		r = append(r, srs[i].String())
	}
	return r
}

func (srs SecurityRuleSet) String() string {
	buf := bytes.Buffer{}
	for i := range srs {
		buf.WriteString(srs[i].String())
		buf.WriteString(";")
	}
	return buf.String()
}

func (srs SecurityRuleSet) Equals(srs1 SecurityRuleSet) bool {
	sort.Sort(srs)
	sort.Sort(srs1)
	return srs.equals(srs1)
}

func (srs SecurityRuleSet) equals(srs1 SecurityRuleSet) bool {
	if len(srs) != len(srs1) {
		return false
	}
	for i := range srs {
		if !srs[i].equals(&srs1[i]) {
			return false
		}
	}
	return true
}

// convert to pure allow list
//
// requirements on srs
//
//  - ordered by priority
//  - same direction
//
/*func (srs SecurityRuleSet) AllowList() SecurityRuleSet {
	allowList := SecurityRuleSet{}
	denyList := SecurityRuleSet{}

	for i := range srs {
		if srs[i].Action == SecurityRuleAllow {
			allowList = append(allowList, srs[i])
		} else {
			denyList = append(denyList, srs[i])
		}
	}

	sort.Sort(allowList)
	allowList.uniq()

	if len(denyList) > 0 {
		sort.Sort(denyList)
		denyList.uniq()

		for i := range denyList {
			allowList = allowList.cutOut(denyList[i])
		}
	}

	allowList = allowList.collapse()
	return allowList
}

func (srs SecurityRuleSet) cutOut(r SecurityRule) SecurityRuleSet {
	cutRes := SecurityRuleSet{}
	for i := range srs {
		cutout := srs[i].cutOut(r)
		cutRes = append(cutRes, cutout...)
	}
	return cutRes
}

func (srs SecurityRuleSet) cutOutFirst() SecurityRuleSet {
	r := SecurityRuleSet{}
	if len(srs) == 0 {
		return r
	}
	sr := srs[0]
	srs_ := srs[1:]

	for _, sr_ := range srs_ {
		if sr.Action == sr_.Action {
			r = append(r, sr_)
			continue
		}
		cut := sr_.cutOut(sr)
		r = append(r, cut...)
	}
	return r
}*/

// remove duplicate rules
func (srs SecurityRuleSet) uniq() SecurityRuleSet {
	for i := len(srs) - 1; i > 0; i-- {
		sr0 := &srs[i-1]
		sr1 := &srs[i]
		if sr0.String() != sr1.String() {
			continue
		}
		srs = append(srs[:i], srs[i+1:]...)
	}
	return srs
}

// collapse result of AllowList
//
//  - same direction
//  - same action
//
//  As they share the same action, priority's influence on order of rules can be ignored
//
func (srs SecurityRuleSet) collapse() SecurityRuleSet {
	srs1 := make(SecurityRuleSet, len(srs))
	copy(srs1, srs)
	for i := range srs1 {
		sr := &srs1[i]
		if len(sr.Ports) > 0 {
			sort.Slice(sr.Ports, func(i, j int) bool {
				return sr.Ports[i] < sr.Ports[j]
			})
		}
	}
	sort.Slice(srs1, func(i, j int) bool {
		sr0 := &srs1[i]
		sr1 := &srs1[j]
		{
			result := compareProtocol(sr0.Protocol, sr1.Protocol)
			switch result {
			case sortutils.Less:
				return true
			case sortutils.More:
				return false
			}
		}

		{
			result := compareIPNet(sr0.IPNet, sr1.IPNet)
			switch result {
			case sortutils.Less:
				return true
			case sortutils.More:
				return false
			}
		}

		if sr0.PortStart > 0 && sr0.PortEnd > 0 {
			if sr1.PortStart > 0 && sr1.PortEnd > 0 {
				return sr0.PortStart < sr1.PortStart
			}
			// port range comes first
			return true
		} else if len(sr0.Ports) > 0 {
			if sr1.PortStart > 0 && sr1.PortEnd > 0 {
				return false
			} else if len(sr1.Ports) > 0 {
				sr0l := len(sr0.Ports)
				sr1l := len(sr1.Ports)
				for i := 0; i < sr0l && i < sr1l; i++ {
					if sr0.Ports[i] != sr1.Ports[i] {
						return sr0.Ports[i] < sr1.Ports[i]
					}
				}
				return sr0l < sr1l
			}
		}
		return sr0.Priority < sr1.Priority
	})
	// merge ports
	for i := len(srs1) - 1; i > 0; i-- {
		sr0 := &srs1[i-1]
		sr1 := &srs1[i]
		if sr0.Protocol != sr1.Protocol {
			continue
		}
		if !sr0.netEquals(sr1) {
			continue
		}
		if (len(sr0.Ports) > 0 || (sr0.PortStart == sr0.PortEnd && sr0.PortStart > 0)) && (len(sr1.Ports) > 0 || (sr1.PortStart == sr1.PortEnd && sr1.PortStart > 0)) {
			ps := newPortsFromInts(sr0.Ports...)
			ps = append(ps, newPortsFromInts(sr1.Ports...)...)
			if sr0.PortStart == sr0.PortEnd && sr0.PortStart > 0 {
				ps = append(ps, uint16(sr0.PortStart))
			}
			if sr1.PortStart == sr1.PortEnd && sr1.PortStart > 0 {
				ps = append(ps, uint16(sr1.PortStart))
			}
			ps = ps.dedup()
			sr0.Ports = ps.IntSlice()
			sr0.PortStart, sr0.PortEnd = -1, -1
			srs1 = append(srs1[:i], srs1[i+1:]...)
		} else if sr0.PortStart > 0 && sr1.PortStart > 0 && sr0.PortEnd > 0 && sr1.PortEnd > 0 {
			if sr0.PortEnd == sr1.PortStart-1 {
				sr0.PortEnd = sr1.PortEnd
				srs1 = append(srs1[:i], srs1[i+1:]...)
			} else if sr0.PortStart-1 == sr1.PortEnd {
				sr0.PortStart = sr1.PortStart
				srs1 = append(srs1[:i], srs1[i+1:]...)
			} else if sr0.PortStart == sr1.PortStart && sr0.PortEnd == sr1.PortEnd {
				srs1 = append(srs1[:i], srs1[i+1:]...)
			}
			// save that contains, intersects
		}
	}

	for i := range srs1 {
		sr := &srs1[i]
		if sr.PortStart <= 1 && sr.PortEnd >= 65535 {
			sr.PortStart = -1
			sr.PortEnd = -1
		}
	}

	//merge cidr
	sort.Slice(srs1, func(i, j int) bool {
		sr0 := &srs1[i]
		sr1 := &srs1[j]
		{
			result := compareProtocol(sr0.Protocol, sr1.Protocol)
			switch result {
			case sortutils.Less:
				return true
			case sortutils.More:
				return false
			}
		}

		if sr0.GetPortsString() != sr1.GetPortsString() {
			return sr0.GetPortsString() < sr1.GetPortsString()
		}

		{
			result := compareIPNet(sr0.IPNet, sr1.IPNet)
			switch result {
			case sortutils.Less:
				return true
			case sortutils.More:
				return false
			}
		}

		return sr0.Priority < sr1.Priority
	})

	// 将端口和协议相同的规则归类
	needMerged := []SecurityRuleSet{}
	for i, j := 0, 0; i < len(srs1); i++ {
		if i == 0 {
			needMerged = append(needMerged, SecurityRuleSet{srs1[i]})
			continue
		}
		last := needMerged[j][len(needMerged[j])-1]
		if last.Protocol == srs1[i].Protocol && last.GetPortsString() == srs1[i].GetPortsString() {
			needMerged[j] = append(needMerged[j], srs1[i])
			continue
		}
		needMerged = append(needMerged, SecurityRuleSet{srs1[i]})
		j++
	}

	result := SecurityRuleSet{}
	for _, srs := range needMerged {
		result = append(result, srs.mergeNet()...)
	}

	result = result.uniq()
	for i := range result {
		sr := &result[i]
		sr.Priority = 1
	}
	return result
}

func (srs SecurityRuleSet) mergeNet() SecurityRuleSet {
	ranges4 := []netutils.IPV4AddrRange{}
	ranges6 := []netutils.IPV6AddrRange{}

	for i := 0; i < len(srs); i++ {
		if isWildNet(srs[i].IPNet) {
			// wild mark
			ranges4 = append(ranges4, netutils.AllIPV4AddrRange)
			ranges6 = append(ranges6, netutils.AllIPV6AddrRange)
		} else {
			cidr := srs[i].IPNet.String()
			if regutils.MatchCIDR6(cidr) {
				// ipv6
				ranges6 = append(ranges6, netutils.NewIPV6AddrRangeFromIPNet(srs[i].IPNet))
			} else {
				ranges4 = append(ranges4, netutils.NewIPV4AddrRangeFromIPNet(srs[i].IPNet))
			}
		}
	}

	ranges4 = netutils.IPV4AddrRangeList(ranges4).Merge()
	ranges6 = netutils.IPV6AddrRangeList(ranges6).Merge()

	nets := []*net.IPNet{}
	hasWildNet4 := false
	hasWildNet6 := false
	for i := range ranges4 {
		addr := ranges4[i]
		for _, n := range addr.ToIPNets() {
			if n.String() == "0.0.0.0/0" {
				hasWildNet4 = true
			} else {
				nets = append(nets, n)
				log.Debugf("merge v4 %s", n.String())
			}
		}
	}
	for i := range ranges6 {
		addr := ranges6[i]
		for _, n := range addr.ToIPNets() {
			if n.String() == "::/0" {
				hasWildNet6 = true
			} else {
				nets = append(nets, n)
			}
		}
	}

	result := SecurityRuleSet{}
	if hasWildNet4 && hasWildNet6 {
		val := srs[0]
		val.IPNet = nil
		result = append(result, val)
	} else if hasWildNet4 {
		val := srs[0]
		val.IPNet = &net.IPNet{
			IP:   net.IPv4zero,
			Mask: net.CIDRMask(0, 32),
		}
		result = append(result, val)
	} else if hasWildNet6 {
		val := srs[0]
		val.IPNet = &net.IPNet{
			IP:   net.IPv6zero,
			Mask: net.CIDRMask(0, 128),
		}
		result = append(result, val)
	}
	for _, net := range nets {
		val := srs[0]
		val.IPNet = net
		result = append(result, val)
	}
	return result
}
