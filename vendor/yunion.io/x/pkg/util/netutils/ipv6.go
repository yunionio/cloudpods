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

package netutils

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sortutils"
)

type IPV6Addr [8]uint16

func hex2Number(hexStr string) (uint16, error) {
	val, err := strconv.ParseInt(hexStr, 16, 64)
	if err != nil {
		return 0, errors.Wrap(err, "strconv.ParseInt")
	}
	return uint16(val), nil
}

func normalizeIpv6Addr(addrStr string) ([8]uint16, error) {
	var addr [8]uint16
	parts := strings.Split(addrStr, ":")
	expectPartCnt := 8
	if len(parts) > 0 && strings.Contains(parts[len(parts)-1], ".") {
		// 2001:db8:3333:4444:5555:6666:1.2.3.4
		v4part, err := IP2Number(parts[len(parts)-1])
		if err != nil {
			return addr, errors.Wrapf(err, "IP2Number %s", parts[len(parts)-1])
		}
		addr[6] = uint16((v4part >> 16) & 0xffff)
		addr[7] = uint16(v4part & 0xFFFF)
		expectPartCnt = 6
		parts = parts[:len(parts)-1]
	}
	if len(parts) < expectPartCnt {
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		switch len(parts) {
		case 0, 1:
			return addr, errors.Wrap(errors.ErrInvalidFormat, addrStr)
		case 2:
			if len(parts[0]) == 0 && expectPartCnt == 6 {
				// ::192.168.0.1
				return addr, nil
			} else if len(parts[0]) > 0 && expectPartCnt == 6 {
				// 3ffe::192.168.0.1
				var err error
				addr[0], err = hex2Number(parts[0])
				if err != nil {
					return addr, errors.Wrapf(err, "hex2Number %s", addrStr)
				}
				return addr, nil
			} else {
				return addr, errors.Wrap(errors.ErrInvalidFormat, addrStr)
			}
		/*case 3:
		if len(parts[0]) == 0 && len(parts[1]) == 0 && len(parts[2]) == 0 {
			// ::
			return addr, nil
		} else if len(parts[0]) == 0 && len(parts[2]) > 0 {
			// ::1, ::1:192.168.0.1
			for i := 0; i <= 1; i++ {
				if len(parts[2-i]) == 0 {
					continue
				}
				var err error
				addr[expectPartCnt-1-i], err = hex2Number(parts[2-i])
				if err != nil {
					return addr, errors.Wrapf(err, "hex2Number %s", addrStr)
				}
			}
			return addr, nil
		} else if len(parts[0]) > 0 && len(parts[2]) == 0 {
			// 3ffe::
			for i := 0; i <= 1; i++ {
				if len(parts[i]) == 0 {
					continue
				}
				var err error
				addr[i], err = hex2Number(parts[i])
				if err != nil {
					return addr, errors.Wrapf(err, "hex2Number %s", addrStr)
				}
			}
			return addr, nil
		} else {
			return addr, errors.Wrap(errors.ErrInvalidFormat, addrStr)
		}*/
		default:
			if len(parts) == 3 && len(parts[0]) == 0 && len(parts[1]) == 0 && len(parts[2]) == 0 {
				return addr, nil
			} else if len(parts[0]) == 0 {
				if len(parts[1]) == 0 {
					// ::1, ::3ffe:3200:1234:1234, ::1234:123.123.123.123
					for i := 0; i < len(parts)-2; i++ {
						partIdx := len(parts) - 1 - i
						addrIdx := expectPartCnt - 1 - i
						var err error
						addr[addrIdx], err = hex2Number(parts[partIdx])
						if err != nil {
							return addr, errors.Wrapf(err, "hex2Number %s", addrStr)
						}
					}
				} else {
					return addr, errors.Wrap(errors.ErrInvalidFormat, addrStr)
				}
			} else if len(parts[len(parts)-1]) == 0 {
				// 3ffe::, 3ffe:3200::, 3ffe:3200::123.123.123.123
				for i := 0; i < len(parts)-1; i++ {
					if len(parts[i]) == 0 {
						continue
					}
					var err error
					addr[i], err = hex2Number(parts[i])
					if err != nil {
						return addr, errors.Wrapf(err, "hex2Number %s", addrStr)
					}
				}
			} else {
				addrIdx := 0
				for i := 0; i < len(parts); i++ {
					if len(parts[i]) == 0 {
						addrIdx = expectPartCnt + 1 - len(parts) + i
						continue
					}
					var err error
					addr[addrIdx], err = hex2Number(parts[i])
					if err != nil {
						return addr, errors.Wrapf(err, "hex2Number %s", addrStr)
					}
					addrIdx++
				}
			}
		}
	} else {
		for i := 0; i < len(parts); i++ {
			var err error
			addr[i], err = hex2Number(strings.TrimSpace(parts[i]))
			if err != nil {
				return addr, errors.Wrapf(err, "hex2Number %s", addrStr)
			}
		}
	}
	return addr, nil
}

func NewIPV6Addr(ipstr string) (IPV6Addr, error) {
	var addr IPV6Addr
	if len(ipstr) > 0 {
		addr8, err := normalizeIpv6Addr(ipstr)
		if err != nil {
			return addr, errors.Wrapf(errors.ErrInvalidFormat, "%s %s", ipstr, err.Error())
		}
		addr = addr8
	}
	return addr, nil
}

const MaxUint16 = uint16(0xffff)

func (addr IPV6Addr) StepDown() IPV6Addr {
	naddr := IPV6Addr{}
	borrow := true
	for i := 7; i >= 0; i-- {
		if borrow {
			if addr[i] > 0 {
				naddr[i] = addr[i] - 1
				borrow = false
			} else {
				naddr[i] = MaxUint16
				borrow = true
			}
		} else {
			naddr[i] = addr[i]
		}
	}
	return naddr
}

func (addr IPV6Addr) StepUp() IPV6Addr {
	naddr := IPV6Addr{}
	upshift := true
	for i := 7; i >= 0; i-- {
		if upshift {
			if addr[i] == MaxUint16 {
				naddr[i] = 0
				upshift = true
			} else {
				naddr[i] = addr[i] + 1
				upshift = false
			}
		} else {
			naddr[i] = addr[i]
		}
	}
	return naddr
}

func masklen2Uint16(maskLen uint8) uint16 {
	return ^(uint16(1<<(16-maskLen)) - 1)
}

func (addr IPV6Addr) NetAddr(maskLen uint8) IPV6Addr {
	var naddr IPV6Addr
	for i := 0; i < 8; i++ {
		if maskLen >= 16 {
			naddr[i] = addr[i]
			maskLen -= 16
		} else if maskLen > 0 && maskLen < 16 {
			naddr[i] = addr[i] & masklen2Uint16(maskLen)
			maskLen = 0
		} else {
			naddr[i] = 0
		}
	}
	return naddr
}

func (addr IPV6Addr) BroadcastAddr(maskLen uint8) IPV6Addr {
	var naddr IPV6Addr
	for i := 0; i < 8; i++ {
		if maskLen >= 16 {
			naddr[i] = addr[i]
			maskLen -= 16
		} else if maskLen > 0 && maskLen < 16 {
			mask := masklen2Uint16(maskLen)
			naddr[i] = addr[i] | (MaxUint16 - mask)
			maskLen = 0
		} else {
			naddr[i] = MaxUint16
		}
	}
	return naddr
}

func (addr IPV6Addr) HostAddr(hostAddr IPV6Addr, maskLen uint8) IPV6Addr {
	var naddr IPV6Addr
	for i := 0; i < 8; i++ {
		if maskLen >= 16 {
			naddr[i] = addr[i]
			maskLen -= 16
		} else if maskLen > 0 && maskLen < 16 {
			mask := masklen2Uint16(maskLen)
			naddr[i] = (addr[i] & mask) | (hostAddr[i] & (MaxUint16 - mask))
			maskLen = 0
		} else {
			naddr[i] = hostAddr[i]
		}
	}
	return naddr
}

func (addr IPV6Addr) String() string {
	hexStrs := make([]string, 8)
	for i := 0; i < 8; i++ {
		hexStrs[i] = fmt.Sprintf("%x", addr[i])
	}

	maxZeroLen := 0
	maxZeroStart := -1
	maxZeroEnd := -1
	zeroStart := -1
	zeroEnd := -1
	for i := 0; i < len(addr); i++ {
		if addr[i] == 0 {
			if zeroStart >= 0 {
				zeroEnd = i
			} else {
				zeroStart = i
				zeroEnd = i
			}
		} else {
			if zeroStart >= 0 {
				// record the max
				if zeroEnd-zeroStart+1 > maxZeroLen {
					maxZeroStart = zeroStart
					maxZeroEnd = zeroEnd
					maxZeroLen = maxZeroEnd - maxZeroStart + 1
				}
				zeroStart = -1
				zeroEnd = -1
			} else {
				// do nothing
			}
		}
	}
	if zeroStart >= 0 {
		if zeroEnd-zeroStart+1 > maxZeroLen {
			maxZeroStart = zeroStart
			maxZeroEnd = zeroEnd
			maxZeroLen = maxZeroEnd - maxZeroStart + 1
		}
	}
	if maxZeroLen > 0 {
		return strings.Join(hexStrs[:maxZeroStart], ":") + "::" + strings.Join(hexStrs[maxZeroEnd+1:], ":")
	} else {
		return strings.Join(hexStrs, ":")
	}
}

func (addr IPV6Addr) Lt(addr2 IPV6Addr) bool {
	for i := 0; i < 8; i++ {
		if addr[i] < addr2[i] {
			return true
		} else if addr[i] > addr2[i] {
			return false
		}
	}
	return false
}

func (addr IPV6Addr) Le(addr2 IPV6Addr) bool {
	for i := 0; i < 8; i++ {
		if addr[i] < addr2[i] {
			return true
		} else if addr[i] > addr2[i] {
			return false
		}
	}
	return true
}

func (addr IPV6Addr) Equals(addr2 IPV6Addr) bool {
	for i := 0; i < 8; i++ {
		if addr[i] != addr2[i] {
			return false
		}
	}
	return true
}

func (addr IPV6Addr) Gt(addr2 IPV6Addr) bool {
	return !addr.Le(addr2)
}

func (addr IPV6Addr) Ge(addr2 IPV6Addr) bool {
	return !addr.Lt(addr2)
}

type IPV6AddrRange struct {
	start IPV6Addr
	end   IPV6Addr
}

func NewIPV6AddrRange(ip1 IPV6Addr, ip2 IPV6Addr) IPV6AddrRange {
	ar := IPV6AddrRange{}
	ar.Set(ip1, ip2)
	return ar
}

func (ar *IPV6AddrRange) Set(ip1 IPV6Addr, ip2 IPV6Addr) {
	if ip1.Lt(ip2) {
		ar.start = ip1
		ar.end = ip2
	} else {
		ar.start = ip2
		ar.end = ip1
	}
}

// n.IP and n.Mask must be ipv4 type.  n.Mask must be canonical
func NewIPV6AddrRangeFromIPNet(n *net.IPNet) IPV6AddrRange {
	pref, err := NewIPV6Prefix(n.String())
	if err != nil {
		panic("unexpected IPNet: " + n.String())
	}
	return pref.ToIPRange()
}

func (ar IPV6AddrRange) Contains(ip IPV6Addr) bool {
	return ip.Ge(ar.start) && ip.Le(ar.end)
}

func (ar IPV6AddrRange) ContainsRange(ar2 IPV6AddrRange) bool {
	return ar.start.Le(ar2.start) && ar.end.Ge(ar2.end)
}

const (
	borderEqual = 0
	borderLower = 1
	borderUpper = 2
	notBorder   = 3
)

func randomUint16(min, max uint16) uint16 {
	return min + uint16(rand.Intn(int(max)+1-int(min)))
}

func randomNext(min, max uint16, border int) (uint16, int) {
	var newVal uint16
	if min == max {
		newVal = min
	} else {
		newVal = randomUint16(min, max)
		if newVal == min {
			if border == borderUpper {
				border = notBorder
			} else if border == borderEqual {
				border = borderLower
			}
		} else if newVal == max {
			if border == borderLower {
				border = notBorder
			} else if border == borderEqual {
				border = borderUpper
			}
		} else {
			border = notBorder
		}
	}
	return newVal, border
}

func (ar IPV6AddrRange) Random() IPV6Addr {
	var naddr IPV6Addr
	border := borderEqual
	for i := 0; i < 8; i++ {
		switch border {
		case borderEqual:
			naddr[i], border = randomNext(ar.start[i], ar.end[i], border)
		case borderLower:
			naddr[i], border = randomNext(ar.start[i], MaxUint16, border)
		case borderUpper:
			naddr[i], border = randomNext(0, ar.end[i], border)
		default:
			// any random
			naddr[i] = randomUint16(0, MaxUint16)
		}
	}
	return naddr
}

func (ar IPV6AddrRange) String() string {
	return fmt.Sprintf("%s-%s", ar.start, ar.end)
}

func (ar IPV6AddrRange) StartIp() IPV6Addr {
	return ar.start
}

func (ar IPV6AddrRange) EndIp() IPV6Addr {
	return ar.end
}

func (ar IPV6AddrRange) Merge(ar2 IPV6AddrRange) (IPV6AddrRange, bool) {
	if ar.IsOverlap(ar2) || ar.end.StepUp().Equals(ar2.start) || ar2.end.StepUp().Equals(ar.start) {
		if ar2.start.Lt(ar.start) {
			ar.start = ar2.start
		}
		if ar2.end.Gt(ar.end) {
			ar.end = ar2.end
		}
		return ar, true
	}
	return ar, false
}

func (ar IPV6AddrRange) IsOverlap(ar2 IPV6AddrRange) bool {
	if ar.start.Gt(ar2.end) || ar.end.Lt(ar2.start) {
		return false
	} else {
		return true
	}
}

func (ar IPV6AddrRange) equals(ar2 IPV6AddrRange) bool {
	return ar.start.Equals(ar2.start) && ar.end.Equals(ar2.end)
}

type IPV6Prefix struct {
	Address IPV6Addr
	MaskLen uint8
	ipRange IPV6AddrRange
}

func (pref *IPV6Prefix) String() string {
	if pref.MaskLen == 128 {
		return pref.Address.String()
	} else {
		return fmt.Sprintf("%s/%d", pref.Address.NetAddr(pref.MaskLen).String(), pref.MaskLen)
	}
}

func (pref *IPV6Prefix) Equals(pref1 *IPV6Prefix) bool {
	if pref1 == nil {
		return false
	}
	return pref.ipRange.equals(pref1.ipRange)
}

func ParsePrefix6(prefix string) (IPV6Addr, uint8, error) {
	slash := strings.IndexByte(prefix, '/')
	if slash > 0 {
		addr, err := NewIPV6Addr(prefix[:slash])
		if err != nil {
			return addr, 0, errors.Wrap(err, "NewIPV6Addr")
		}
		maskLen, err := strconv.Atoi(prefix[slash+1:])
		if err != nil {
			return addr, 0, ErrInvalidMask // fmt.Errorf("invalid masklen %s", err)
		}
		if maskLen < 0 || maskLen > 128 {
			return addr, 0, ErrOutOfRangeMask // fmt.Errorf("out of range masklen")
		}
		return addr.NetAddr(uint8(maskLen)), uint8(maskLen), nil
	} else {
		addr, err := NewIPV6Addr(prefix)
		if err != nil {
			return addr, 0, errors.Wrap(err, "NewIPV4Addr")
		}
		return addr, 128, nil
	}
}

func NewIPV6Prefix(prefix string) (IPV6Prefix, error) {
	addr, maskLen, err := ParsePrefix6(prefix)
	if err != nil {
		return IPV6Prefix{}, errors.Wrap(err, "ParsePrefix6")
	}
	pref := IPV6Prefix{
		Address: addr,
		MaskLen: maskLen,
	}
	pref.ipRange = pref.ToIPRange()
	return pref, nil
}

func NewIPV6PrefixFromAddr(addr IPV6Addr, masklen uint8) IPV6Prefix {
	pref := IPV6Prefix{
		Address: addr.NetAddr(masklen),
		MaskLen: masklen,
	}
	pref.ipRange = pref.ToIPRange()
	return pref
}

func (prefix IPV6Prefix) ToIPRange() IPV6AddrRange {
	start := prefix.Address.NetAddr(prefix.MaskLen)
	end := prefix.Address.BroadcastAddr(prefix.MaskLen)
	return IPV6AddrRange{start: start, end: end}
}

func (prefix IPV6Prefix) Contains(ip IPV6Addr) bool {
	return prefix.ipRange.Contains(ip)
}

func DeriveIPv6AddrFromIPv4AddrMac(ipAddr string, macAddr string, startIp6, endIp6 string, maskLen6 uint8) string {
	v4, _ := NewIPV4Addr(ipAddr)
	v4Bytes := v4.ToBytes()
	mac, _ := ParseMac(macAddr)
	hostAddr := IPV6Addr{
		0, 0, 0, 0,
		binary.BigEndian.Uint16([]byte{v4Bytes[0], v4Bytes[1]}),
		binary.BigEndian.Uint16([]byte{v4Bytes[2], v4Bytes[3]}),
		binary.BigEndian.Uint16([]byte{mac[2], mac[3]}),
		binary.BigEndian.Uint16([]byte{mac[4], mac[5]}),
	}
	sAddr6, _ := NewIPV6Addr(startIp6)
	eAddr6, _ := NewIPV6Addr(endIp6)
	netAddr6 := sAddr6.NetAddr(maskLen6)
	i := uint8(64)
	if maskLen6 > i {
		i = maskLen6
	}
	for i < 128 {
		addr6 := netAddr6.HostAddr(hostAddr, i)
		if addr6.Ge(sAddr6) && addr6.Le(eAddr6) {
			return addr6.String()
		}
		i++
	}
	return ""
}

func (ar IPV6AddrRange) Substract(ar2 IPV6AddrRange) ([]IPV6AddrRange, *IPV6AddrRange) {
	lefts, overlap, sub := ar.Substract2(ar2)
	var subp *IPV6AddrRange
	if overlap {
		subp = &sub
	}
	return lefts, subp
}

func (ar IPV6AddrRange) Substract2(ar2 IPV6AddrRange) ([]IPV6AddrRange, bool, IPV6AddrRange) {
	lefts := []IPV6AddrRange{}
	// no intersection, no substract
	if ar.end.Lt(ar2.start) || ar.start.Gt(ar2.end) {
		lefts = append(lefts, ar)
		return lefts, false, IPV6AddrRange{}
	}

	// ar contains ar2
	if ar.ContainsRange(ar2) {
		if ar.start.Equals(ar2.start) && ar.end.Equals(ar2.end) {
			// lefts empty
		} else if ar.start.Lt(ar2.start) && ar.end.Equals(ar2.end) {
			lefts = append(lefts,
				NewIPV6AddrRange(ar.start, ar2.start.StepDown()),
			)
		} else if ar.start.Equals(ar2.start) && ar.end.Gt(ar2.end) {
			lefts = append(lefts,
				NewIPV6AddrRange(ar2.end.StepUp(), ar.end),
			)
		} else {
			lefts = append(lefts,
				NewIPV6AddrRange(ar.start, ar2.start.StepDown()),
				NewIPV6AddrRange(ar2.end.StepUp(), ar.end),
			)
		}
		return lefts, true, ar2
	}

	// ar contained by ar2
	if ar2.ContainsRange(ar) {
		return lefts, true, ar
	}

	// intersect, ar on the left
	if ar.start.Lt(ar2.start) && ar.end.Ge(ar2.start) {
		lefts = append(lefts, NewIPV6AddrRange(ar.start, ar2.start.StepDown()))
		sub_ := NewIPV6AddrRange(ar2.start, ar.end)
		return lefts, true, sub_
	}

	// intersect, ar on the right
	if ar.start.Le(ar2.end) && ar.end.Gt(ar2.end) {
		lefts = append(lefts, NewIPV6AddrRange(ar2.end.StepUp(), ar.end))
		sub_ := NewIPV6AddrRange(ar.start, ar2.end)
		return lefts, true, sub_
	}

	// no intersection
	return lefts, false, IPV6AddrRange{}
}

func (addr IPV6Addr) ToBytes() []byte {
	ret := make([]byte, 16)
	for i := 0; i < len(addr); i++ {
		binary.BigEndian.PutUint16(ret[2*i:], addr[i])
	}
	return ret
}

func (addr IPV6Addr) ToIP() net.IP {
	return net.IP(addr.ToBytes())
}

func (pref IPV6Prefix) ToIPNet() *net.IPNet {
	return &net.IPNet{
		IP:   pref.Address.ToIP(),
		Mask: net.CIDRMask(int(pref.MaskLen), 128),
	}
}

func (ar IPV6AddrRange) ToIPNets() []*net.IPNet {
	r := []*net.IPNet{}
	mms := ar.ToPrefixes()
	for _, mm := range mms {
		r = append(r, mm.ToIPNet())
	}
	return r
}

func (ar IPV6AddrRange) ToPrefixes() []IPV6Prefix {
	prefixes := make([]IPV6Prefix, 0)
	sp := ar.StartIp()
	ep := ar.EndIp()
	for sp.Le(ep) {
		masklen := uint8(128)
		for sp.NetAddr(masklen-1).Equals(sp) && sp.BroadcastAddr(masklen-1).Le(ep) && masklen > 0 {
			masklen--
		}
		if masklen == 0 {
			prefixes = append(prefixes, NewIPV6PrefixFromAddr(sp, 0))
			break
		}
		prefixes = append(prefixes, NewIPV6PrefixFromAddr(sp, masklen))
		sp = sp.BroadcastAddr(masklen).StepUp()
	}
	return prefixes
}

type IPV6AddrRangeList []IPV6AddrRange

func (rl IPV6AddrRangeList) Len() int {
	return len(rl)
}

func (rl IPV6AddrRangeList) Swap(i, j int) {
	rl[i], rl[j] = rl[j], rl[i]
}

func (rl IPV6AddrRangeList) Less(i, j int) bool {
	return rl[i].Compare(rl[j]) == sortutils.Less
}

func (v6range IPV6AddrRange) Compare(r2 IPV6AddrRange) sortutils.CompareResult {
	if v6range.start.Lt(r2.start) {
		return sortutils.Less
	} else if v6range.start.Gt(r2.start) {
		return sortutils.More
	} else {
		// start equals, compare ends
		if v6range.end.Gt(r2.end) {
			return sortutils.Less
		} else if v6range.end.Lt(r2.end) {
			return sortutils.More
		} else {
			return sortutils.Equal
		}
	}
}

func (rl IPV6AddrRangeList) Merge() []IPV6AddrRange {
	sort.Sort(rl)
	ret := make([]IPV6AddrRange, 0, len(rl))
	for i := range rl {
		if i == 0 {
			ret = append(ret, rl[i])
		} else {
			result, isMerged := ret[len(ret)-1].Merge(rl[i])
			if isMerged {
				ret[len(ret)-1] = result
			} else {
				ret = append(ret, rl[i])
			}
		}
	}
	return ret
}

func (rl IPV6AddrRangeList) String() string {
	ret := make([]string, len(rl))
	for i := range rl {
		ret[i] = rl[i].String()
	}
	return strings.Join(ret, ",")
}

var IPV6Zero = IPV6Addr([8]uint16{0, 0, 0, 0, 0, 0, 0, 0})
var IPV6Ones = IPV6Addr([8]uint16{0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff})
var AllIPV6AddrRange = IPV6AddrRange{
	start: IPV6Zero,
	end:   IPV6Ones,
}

func (r IPV6AddrRange) IsAll() bool {
	return r.start.Equals(IPV6Zero) && r.end.Equals(IPV6Ones)
}

func (rl IPV6AddrRangeList) Substract(addrRange IPV6AddrRange) []IPV6AddrRange {
	ret := make([]IPV6AddrRange, 0)
	for i := range rl {
		lefts, _ := rl[i].Substract(addrRange)
		ret = append(ret, lefts...)
	}
	return ret
}

func Mac2LinkLocal(mac string) (IPV6Addr, error) {
	macBytes, err := ParseMac(mac)
	if err != nil {
		return IPV6Addr{}, errors.Wrap(err, "ParseMac")
	}
	return IPV6Addr{
		0xfe80, 0, 0, 0,
		binary.BigEndian.Uint16([]byte{macBytes[0] ^ 0x02, macBytes[1]}),
		binary.BigEndian.Uint16([]byte{macBytes[2], 0xff}),
		binary.BigEndian.Uint16([]byte{0xfe, macBytes[3]}),
		binary.BigEndian.Uint16([]byte{macBytes[4], macBytes[5]}),
	}, nil
}
