package netutils

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
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
				return 0, fmt.Errorf("invalid number %s", parts[i])
			}
			num = num | (uint32(n) << uint32(24-i*8))
		}
		return num, nil
	}
	return 0, fmt.Errorf("invalid ip address %s", ipstr)
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
			return addr, err
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

func (ar IPV4AddrRange) IsOverlap(ar2 IPV4AddrRange) bool {
	if ar.start > ar2.end || ar.end < ar2.start {
		return false
	} else {
		return true
	}
}

func Masklen2Mask(maskLen int8) IPV4Addr {
	var mask uint32 = 0
	for i := 0; i < int(maskLen); i += 1 {
		mask |= 1 << uint(31-i)
	}
	return IPV4Addr(mask)
}

type IPV4Prefix struct {
	Address IPV4Addr
	MaskLen int8
}

func (pref *IPV4Prefix) String() string {
	return fmt.Sprintf("%s/%d", pref.Address.NetAddr(pref.MaskLen).String(), pref.MaskLen)
}

func Mask2Len(mask IPV4Addr) int8 {
	var maskLen int8 = 0
	for {
		if (mask & (1 << uint(31-maskLen))) == 0 {
			break
		}
		maskLen += 1
	}
	return maskLen
}

func ParsePrefix(prefix string) (IPV4Addr, int8, error) {
	slash := strings.IndexByte(prefix, '/')
	if slash > 0 {
		addr, err := NewIPV4Addr(prefix[:slash])
		if err != nil {
			return 0, 0, err
		}
		if regutils.MatchIP4Addr(prefix[slash+1:]) {
			mask, err := NewIPV4Addr(prefix[slash+1:])
			if err != nil {
				return 0, 0, err
			}
			maskLen := Mask2Len(mask)
			return addr.NetAddr(maskLen), maskLen, nil
		} else {
			maskLen, err := strconv.Atoi(prefix[slash+1:])
			if err != nil {
				return 0, 0, fmt.Errorf("invalid masklen %s", err)
			}
			if maskLen < 0 || maskLen > 32 {
				return 0, 0, fmt.Errorf("out of range masklen")
			}
			return addr.NetAddr(int8(maskLen)), int8(maskLen), nil
		}
	} else {
		addr, err := NewIPV4Addr(prefix)
		if err != nil {
			return 0, 0, err
		}
		return addr, 32, nil
	}
}

func NewIPV4Prefix(prefix string) (IPV4Prefix, error) {
	addr, maskLen, err := ParsePrefix(prefix)
	if err != nil {
		return IPV4Prefix{}, err
	}
	return IPV4Prefix{Address: addr, MaskLen: maskLen}, nil
}

func (prefix IPV4Prefix) ToIPRange() IPV4AddrRange {
	start := prefix.Address.NetAddr(prefix.MaskLen)
	end := prefix.Address.BroadcastAddr(prefix.MaskLen)
	return IPV4AddrRange{start: start, end: end}
}

const (
	hostlocalPrefix = "127.0.0.0/8"

	linklocalPrefix = "169.254.0.0/16"

	multicastPrefix = "224.0.0.0/4"
)

var privatePrefixes = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
}

var privateIPRanges []IPV4AddrRange
var hostLocalIPRange IPV4AddrRange
var linkLocalIPRange IPV4AddrRange
var multicastIPRange IPV4AddrRange

func init() {
	privateIPRanges = make([]IPV4AddrRange, len(privatePrefixes))
	for i, prefix := range privatePrefixes {
		prefix, _ := NewIPV4Prefix(prefix)
		privateIPRanges[i] = prefix.ToIPRange()
	}
	prefix, _ := NewIPV4Prefix(hostlocalPrefix)
	hostLocalIPRange = prefix.ToIPRange()
	prefix, _ = NewIPV4Prefix(linklocalPrefix)
	linkLocalIPRange = prefix.ToIPRange()
	prefix, _ = NewIPV4Prefix(multicastPrefix)
	multicastIPRange = prefix.ToIPRange()
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
