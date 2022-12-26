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
	"fmt"
	"math/bits"
	"math/rand"
	"net"
	"strconv"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
)

const macChars = "0123456789abcdef"
const macCapChars = "ABCDEF"

func FormatMacAddr(macAddr string) string {
	buf := make([]byte, 12)
	bufIdx := 0
	for i := 0; i < len(macAddr) && bufIdx < len(buf); i += 1 {
		c := macAddr[i]
		if strings.IndexByte(macChars, c) >= 0 {
			buf[bufIdx] = c
			bufIdx += 1
		} else if strings.IndexByte(macCapChars, c) >= 0 {
			buf[bufIdx] = c - 'A' + 'a'
			bufIdx += 1
		}
	}
	if len(buf) == bufIdx {
		return fmt.Sprintf("%c%c:%c%c:%c%c:%c%c:%c%c:%c%c", buf[0], buf[1], buf[2], buf[3], buf[4], buf[5],
			buf[6], buf[7], buf[8], buf[9], buf[10], buf[11])
	} else {
		return ""
	}
}

func IP2Number(ipstr string) (uint32, error) {
	parts := strings.Split(ipstr, ".")
	if len(parts) == 4 {
		var num uint32
		for i := 0; i < 4; i += 1 {
			n, e := strconv.Atoi(parts[i])
			if e != nil {
				return 0, ErrInvalidNumber // fmt.Errorf("invalid number %s", parts[i])
			}
			if n < 0 || n > 255 {
				return 0, ErrOutOfRange
			}
			num = num | (uint32(n) << uint32(24-i*8))
		}
		return num, nil
	}
	return 0, ErrInvalidIPAddr // fmt.Errorf("invalid ip address %s", ipstr)
}

/*func IP2Bytes(ipstr string) ([]byte, error) {
	parts := strings.Split(ipstr, ".")
	if len(parts) == 4 {
		bytes := make([]byte, 4)
		for i := 0; i < 4; i += 1 {
			n, e := strconv.Atoi(parts[i])
			if e != nil {
				return nil, fmt.Errorf("invalid number %s", parts[i])
			}
			bytes[i] = byte(n)
		}
		return bytes, nil
	}
	return nil, fmt.Errorf("invalid ip address %s", ipstr)
}*/

func Number2Bytes(num uint32) []byte {
	a := num >> 24
	num -= a << 24
	b := num >> 16
	num -= b << 16
	c := num >> 8
	num -= c << 8
	return []byte{byte(a), byte(b), byte(c), byte(num)}
}

func Number2IP(num uint32) string {
	bytes := Number2Bytes(num)
	return fmt.Sprintf("%d.%d.%d.%d", bytes[0], bytes[1], bytes[2], bytes[3])
}

type IPV4Addr uint32

func NewIPV4Addr(ipstr string) (IPV4Addr, error) {
	var addr IPV4Addr
	if len(ipstr) > 0 {
		num, err := IP2Number(ipstr)
		if err != nil {
			return addr, errors.Wrap(err, "IP2Number")
		}
		addr = IPV4Addr(num)
	}
	return addr, nil
}

func (addr IPV4Addr) StepDown() IPV4Addr {
	naddr := addr - 1
	return naddr
}

func (addr IPV4Addr) StepUp() IPV4Addr {
	naddr := addr + 1
	return naddr
}

func (addr IPV4Addr) NetAddr(maskLen int8) IPV4Addr {
	mask := Masklen2Mask(maskLen)
	return IPV4Addr(mask & addr)
}

func (addr IPV4Addr) BroadcastAddr(maskLen int8) IPV4Addr {
	mask := Masklen2Mask(maskLen)
	return IPV4Addr(addr | (0xffffffff - mask))
}

func (addr IPV4Addr) CliAddr(maskLen int8) IPV4Addr {
	mask := Masklen2Mask(maskLen)
	return IPV4Addr(addr - (addr & mask))
}

func (addr IPV4Addr) String() string {
	return Number2IP(uint32(addr))
}

func (addr IPV4Addr) ToBytes() []byte {
	a := byte((addr & 0xff000000) >> 24)
	b := byte((addr & 0x00ff0000) >> 16)
	c := byte((addr & 0x0000ff00) >> 8)
	d := byte(addr & 0x000000ff)
	return []byte{a, b, c, d}
}

func (addr IPV4Addr) ToMac(prefix string) string {
	bytes := addr.ToBytes()
	return fmt.Sprintf("%s%02x:%02x:%02x:%02x", prefix, bytes[0], bytes[1], bytes[2], bytes[3])
}

type IPV4AddrRange struct {
	start IPV4Addr
	end   IPV4Addr
}

func NewIPV4AddrRange(ip1 IPV4Addr, ip2 IPV4Addr) IPV4AddrRange {
	if ip1 < ip2 {
		return IPV4AddrRange{start: ip1, end: ip2}
	} else {
		return IPV4AddrRange{start: ip2, end: ip1}
	}
}

// n.IP and n.Mask must be ipv4 type.  n.Mask must be canonical
func NewIPV4AddrRangeFromIPNet(n *net.IPNet) IPV4AddrRange {
	pref, err := NewIPV4Prefix(n.String())
	if err != nil {
		panic("unexpected IPNet: " + n.String())
	}
	return pref.ToIPRange()
}

func (ar IPV4AddrRange) Contains(ip IPV4Addr) bool {
	return (ip >= ar.start) && (ip <= ar.end)
}

func (ar IPV4AddrRange) ContainsRange(ar2 IPV4AddrRange) bool {
	return ar.start <= ar2.start && ar.end >= ar2.end
}

func (ar IPV4AddrRange) Random() IPV4Addr {
	return IPV4Addr(uint32(ar.start) + uint32(rand.Intn(int(uint32(ar.end)-uint32(ar.start)))))
}

func (ar IPV4AddrRange) AddressCount() int {
	return int(uint32(ar.end) - uint32(ar.start) + 1)
}

func (ar IPV4AddrRange) String() string {
	return fmt.Sprintf("%s-%s", ar.start, ar.end)
}

func (ar IPV4AddrRange) StartIp() IPV4Addr {
	return ar.start
}

func (ar IPV4AddrRange) EndIp() IPV4Addr {
	return ar.end
}

func (ar IPV4AddrRange) Merge(ar2 IPV4AddrRange) (*IPV4AddrRange, bool) {
	if ar.IsOverlap(ar2) || ar.end+1 == ar2.start || ar2.end+1 == ar.start {
		if ar2.start < ar.start {
			ar.start = ar2.start
		}
		if ar2.end > ar.end {
			ar.end = ar2.end
		}
		return &ar, true
	}
	return nil, false
}

func (ar IPV4AddrRange) IsOverlap(ar2 IPV4AddrRange) bool {
	if ar.start > ar2.end || ar.end < ar2.start {
		return false
	} else {
		return true
	}
}

func (ar IPV4AddrRange) ToIPNets() []*net.IPNet {
	r := []*net.IPNet{}
	mms := ar.ToMaskMatches()
	for _, mm := range mms {
		a := mm[0]
		m := mm[1]
		addr := net.IPv4(byte((a>>24)&0xff), byte((a>>16)&0xff), byte((a>>8)&0xff), byte(a&0xff))
		mask := net.IPv4Mask(byte((m>>24)&0xff), byte((m>>16)&0xff), byte((m>>8)&0xff), byte(m&0xff))
		r = append(r, &net.IPNet{
			IP:   addr,
			Mask: mask,
		})
	}
	return r
}

func (ar IPV4AddrRange) ToMaskMatches() [][2]uint32 {
	r := [][2]uint32{}
	s := uint32(ar.start)
	e := uint32(ar.end)
	if s == e {
		r = append(r, [2]uint32{s, ^uint32(0)})
		return r
	}
	sp, ep := uint64(s), uint64(e)
	ep = ep + 1
	for sp < ep {
		b := uint64(1)
		for (sp+b) <= ep && (sp&(b-1)) == 0 {
			b <<= 1
		}
		b >>= 1
		r = append(r, [2]uint32{uint32(sp), uint32(^(b - 1))})
		sp = sp + b
	}
	return r
}

func (ar IPV4AddrRange) Substract(ar2 IPV4AddrRange) (lefts []IPV4AddrRange, sub *IPV4AddrRange) {
	lefts = []IPV4AddrRange{}
	// no intersection, no substract
	if ar.end < ar2.start || ar.start > ar2.end {
		lefts = append(lefts, ar)
		return
	}

	// ar contains ar2
	if ar.ContainsRange(ar2) {
		nns := [][2]int64{
			[2]int64{int64(ar.start), int64(ar2.start) - 1},
			[2]int64{int64(ar2.end) + 1, int64(ar.end)},
		}
		for _, nn := range nns {
			if nn[0] <= nn[1] {
				lefts = append(lefts, NewIPV4AddrRange(IPV4Addr(nn[0]), IPV4Addr(nn[1])))
			}
		}
		ar2_ := ar2
		sub = &ar2_
		return
	}

	// ar contained by ar2
	if ar2.ContainsRange(ar) {
		ar_ := ar
		sub = &ar_
		return
	}

	// intersect, ar on the left
	if ar.start < ar2.start && ar.end >= ar2.start {
		lefts = append(lefts, NewIPV4AddrRange(ar.start, ar2.start-1))
		sub_ := NewIPV4AddrRange(ar2.start, ar.end)
		sub = &sub_
		return
	}

	// intersect, ar on the right
	if ar.start <= ar2.end && ar.end > ar2.end {
		lefts = append(lefts, NewIPV4AddrRange(ar2.end+1, ar.end))
		sub_ := NewIPV4AddrRange(ar.start, ar2.end)
		sub = &sub_
		return
	}

	// no intersection
	return
}

func (ar IPV4AddrRange) equals(ar2 IPV4AddrRange) bool {
	return ar.start == ar2.start && ar.end == ar2.end
}

func Masklen2Mask(maskLen int8) IPV4Addr {
	if maskLen < 0 {
		panic("negative masklen")
	}
	return IPV4Addr(^(uint32(1<<(32-uint8(maskLen))) - 1))
}

type IPV4Prefix struct {
	Address IPV4Addr
	MaskLen int8
	ipRange IPV4AddrRange
}

func (pref *IPV4Prefix) String() string {
	if pref.MaskLen == 32 {
		return pref.Address.String()
	} else {
		return fmt.Sprintf("%s/%d", pref.Address.NetAddr(pref.MaskLen).String(), pref.MaskLen)
	}
}

func (pref *IPV4Prefix) Equals(pref1 *IPV4Prefix) bool {
	if pref1 == nil {
		return false
	}
	return pref.ipRange.equals(pref1.ipRange)
}

func Mask2Len(mask IPV4Addr) int8 {
	return int8(bits.LeadingZeros32(^uint32(mask)))
}

func ParsePrefix(prefix string) (IPV4Addr, int8, error) {
	slash := strings.IndexByte(prefix, '/')
	if slash > 0 {
		addr, err := NewIPV4Addr(prefix[:slash])
		if err != nil {
			return 0, 0, errors.Wrap(err, "NewIPV4Addr")
		}
		if regutils.MatchIP4Addr(prefix[slash+1:]) {
			mask, err := NewIPV4Addr(prefix[slash+1:])
			if err != nil {
				return 0, 0, errors.Wrap(err, "NewIPV4Addr")
			}
			maskLen := Mask2Len(mask)
			return addr.NetAddr(maskLen), maskLen, nil
		} else {
			maskLen, err := strconv.Atoi(prefix[slash+1:])
			if err != nil {
				return 0, 0, ErrInvalidMask // fmt.Errorf("invalid masklen %s", err)
			}
			if maskLen < 0 || maskLen > 32 {
				return 0, 0, ErrOutOfRangeMask // fmt.Errorf("out of range masklen")
			}
			return addr.NetAddr(int8(maskLen)), int8(maskLen), nil
		}
	} else {
		addr, err := NewIPV4Addr(prefix)
		if err != nil {
			return 0, 0, errors.Wrap(err, "NewIPV4Addr")
		}
		return addr, 32, nil
	}
}

func NewIPV4Prefix(prefix string) (IPV4Prefix, error) {
	addr, maskLen, err := ParsePrefix(prefix)
	if err != nil {
		return IPV4Prefix{}, errors.Wrap(err, "ParsePrefix")
	}
	pref := IPV4Prefix{
		Address: addr,
		MaskLen: maskLen,
	}
	pref.ipRange = pref.ToIPRange()
	return pref, nil
}

func (prefix IPV4Prefix) ToIPRange() IPV4AddrRange {
	start := prefix.Address.NetAddr(prefix.MaskLen)
	end := prefix.Address.BroadcastAddr(prefix.MaskLen)
	return IPV4AddrRange{start: start, end: end}
}

func (prefix IPV4Prefix) Contains(ip IPV4Addr) bool {
	return prefix.ipRange.Contains(ip)
}

const (
	hostlocalPrefix = "127.0.0.0/8"

	linklocalPrefix = "169.254.0.0/16"

	multicastPrefix = "224.0.0.0/4"
)

var privateIPRanges []IPV4AddrRange
var hostLocalIPRange IPV4AddrRange
var linkLocalIPRange IPV4AddrRange
var multicastIPRange IPV4AddrRange

func init() {
	updatePrivateIPRanges(nil)

	prefix, _ := NewIPV4Prefix(hostlocalPrefix)
	hostLocalIPRange = prefix.ToIPRange()
	prefix, _ = NewIPV4Prefix(linklocalPrefix)
	linkLocalIPRange = prefix.ToIPRange()
	prefix, _ = NewIPV4Prefix(multicastPrefix)
	multicastIPRange = prefix.ToIPRange()
}

func updatePrivateIPRanges(prefs []string) {
	if len(prefs) == 0 {
		prefs = []string{
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
		}
	}
	privateIPRanges = make([]IPV4AddrRange, len(prefs))
	for i, prefix := range prefs {
		prefix, err := NewIPV4Prefix(prefix)
		if err != nil {
			continue
		}
		privateIPRanges[i] = prefix.ToIPRange()
	}
}

func SetPrivatePrefixes(pref []string) {
	updatePrivateIPRanges(pref)
}

func GetPrivateIPRanges() []IPV4AddrRange {
	return privateIPRanges
}

func IsPrivate(addr IPV4Addr) bool {
	for _, ipRange := range privateIPRanges {
		if ipRange.Contains(addr) {
			return true
		}
	}
	return false
}

func IsHostLocal(addr IPV4Addr) bool {
	return hostLocalIPRange.Contains(addr)
}

func IsLinkLocal(addr IPV4Addr) bool {
	return linkLocalIPRange.Contains(addr)
}

func IsMulticast(addr IPV4Addr) bool {
	return multicastIPRange.Contains(addr)
}

func IsExitAddress(addr IPV4Addr) bool {
	return !IsPrivate(addr) && !IsHostLocal(addr) && !IsLinkLocal(addr) && !IsMulticast(addr)
}

func MacUnpackHex(mac string) string {
	if regutils.MatchCompactMacAddr(mac) {
		parts := make([]string, 6)
		for i := 0; i < 12; i += 2 {
			parts[i/2] = strings.ToLower(mac[i : i+2])
		}
		return strings.Join(parts, ":")
	}
	return ""
}

func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

var MASKS = []string{"0", "128", "192", "224", "240", "248", "252", "254", "255"}

func Netlen2Mask(netmasklen int) string {
	var mask = ""
	var segCnt = 0
	for netmasklen > 0 {
		var m string
		if netmasklen > 8 {
			m = MASKS[8]
			netmasklen -= 8
		} else {
			m = MASKS[netmasklen]
			netmasklen = 0
		}
		if mask != "" {
			mask += "."
		}
		mask += m
		segCnt += 1
	}
	for i := 0; i < (4 - segCnt); i++ {
		if mask != "" {
			mask += "."
		}
		mask += "0"
	}
	return mask
}
