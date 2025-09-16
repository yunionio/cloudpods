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
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
	"unicode"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var PSEUDO_VIP = "169.254.169.231"

// var MASKS = []string{"0", "128", "192", "224", "240", "248", "252", "254", "255"}

var PRIVATE_PREFIXES = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
}

func GetFreePort() (int, error) {
	return netutils.GetFreePort()
}

func IsTcpPortUsed(addr string, port int) bool {
	server, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return true
	}
	server.Close()
	return false
}

// MyIP returns source ip used to communicate with udp:114.114.114.114
func MyIP() (ip string, err error) {
	return MyIPTo("114.114.114.114")
}

// MyIPTo returns source ip used to communicate with udp:dstIP
func MyIPTo(dstIP string) (ip string, err error) {
	conn, err := net.Dial("udp4", dstIP+":53")
	if err != nil {
		return
	}
	defer conn.Close()
	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		err = fmt.Errorf("not a net.UDPAddr: %#v", conn.LocalAddr())
		return
	}
	ip = addr.IP.String()
	return
}

func MyIPSmart() (ip string, err error) {
	return MyIPSmartTo("114.114.114.114", "2001:4860:4860::8888")
}

func MyIPSmartTo(ipv4Target, ipv6Target string) (ip string, err error) {
	// try IPv4 connect
	if ipv4Target != "" {
		conn, err4 := net.Dial("udp4", ipv4Target+":53")
		if err4 == nil {
			defer conn.Close()
			if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
				return addr.IP.String(), nil
			}
		}
	}

	// try IPv6 connect
	if ipv6Target != "" {
		conn, err6 := net.Dial("udp6", "["+ipv6Target+"]:53")
		if err6 == nil {
			defer conn.Close()
			if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
				return addr.IP.String(), nil
			}
		}
	}

	// try locallink
	return getLocalIP()
}

func getLocalIP() (string, error) {
	// get default route
	if ip := getIPFromDefaultRoute(); ip != "" {
		return ip, nil
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return "", errors.Wrap(err, "get network interfaces")
	}

	var candidateIPs []string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String(), nil
				} else if ipnet.IP.To16() != nil && !ipnet.IP.IsLinkLocalUnicast() {
					candidateIPs = append(candidateIPs, ipnet.IP.String())
				}
			}
		}
	}

	if len(candidateIPs) > 0 {
		return candidateIPs[0], nil
	}

	return "", fmt.Errorf("no suitable IP address found")
}

func getIPFromDefaultRoute() string {
	if ip := getIPFromDefaultRouteV4(); ip != "" {
		return ip
	}

	if ip := getIPFromDefaultRouteV6(); ip != "" {
		return ip
	}

	return ""
}

func getIPFromDefaultRouteV4() string {
	output, err := procutils.NewCommand("ip", "route", "show", "default").Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "default") && strings.Contains(line, "src") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "src" && i+1 < len(fields) {
					return fields[i+1]
				}
			}
		}
		if strings.Contains(line, "default") && strings.Contains(line, "dev") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "dev" && i+1 < len(fields) {
					if ip := getIPFromInterface(fields[i+1]); ip != "" {
						return ip
					}
				}
			}
		}
	}

	return ""
}

func getIPFromDefaultRouteV6() string {
	output, err := procutils.NewCommand("ip", "-6", "route", "show", "default").Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "default") && strings.Contains(line, "src") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "src" && i+1 < len(fields) {
					return fields[i+1]
				}
			}
		}
		if strings.Contains(line, "default") && strings.Contains(line, "dev") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "dev" && i+1 < len(fields) {
					if ip := getIPFromInterfaceV6(fields[i+1]); ip != "" {
						return ip
					}
				}
			}
		}
	}

	return ""
}

func getIPFromInterface(ifaceName string) string {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return ""
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipv4 := ipnet.IP.To4(); ipv4 != nil {
				return ipv4.String()
			}
		}
	}

	return ""
}

func getIPFromInterfaceV6(ifaceName string) string {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return ""
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipv6 := ipnet.IP.To16(); ipv6 != nil && ipnet.IP.To4() == nil && !ipnet.IP.IsLinkLocalUnicast() {
				return ipv6.String()
			}
		}
	}

	return ""
}

func GetPrivatePrefixes(privatePrefixes []string) []string {
	if privatePrefixes != nil {
		return privatePrefixes
	} else {
		return PRIVATE_PREFIXES
	}
}

func GetMainNicFromDeployApi(nics []*types.SServerNic) (*types.SServerNic, error) {
	var mainIp netutils.IPV4Addr
	var mainNic *types.SServerNic
	for _, n := range nics {
		if len(n.Gateway) > 0 {
			ip := n.Ip
			ipInt, err := netutils.NewIPV4Addr(ip)
			if err != nil {
				return nil, errors.Wrapf(err, "netutils.NewIPV4Addr %s", ip)
			}
			if mainIp == 0 {
				mainIp = ipInt
				mainNic = n
			} else if !netutils.IsPrivate(ipInt) && netutils.IsPrivate(mainIp) {
				mainIp = ipInt
				mainNic = n
			}
		}
	}
	if mainNic != nil {
		return mainNic, nil
	}
	for _, n := range nics {
		ip := n.Ip
		ipInt, err := netutils.NewIPV4Addr(ip)
		if err != nil {
			return nil, errors.Wrap(err, "netutils.NewIPV4Addr")
		}
		if mainIp == 0 {
			mainIp = ipInt
			mainNic = n
		} else if !netutils.IsPrivate(ipInt) && netutils.IsPrivate(mainIp) {
			mainIp = ipInt
			mainNic = n
		}
	}
	if mainNic != nil {
		return mainNic, nil
	}
	return nil, errors.Wrap(errors.ErrInvalidStatus, "no valid nic")
}

func Netlen2Mask(netmasklen int) string {
	return netutils.Netlen2Mask(netmasklen)
}

func addRoute(routes []SRouteInfo, net, gw string) []SRouteInfo {
	route, _ := ParseRouteInfo([]string{net, gw})
	if route != nil {
		for _, rt := range routes {
			if rt.Prefix.String() == route.Prefix.String() && rt.PrefixLen == route.PrefixLen {
				return routes
			}
		}
		// not found
		routes = append(routes, *route)
	}
	return routes
}

func extendRoutes(routes4, routes6 []SRouteInfo, nicRoutes []types.SRoute) ([]SRouteInfo, []SRouteInfo) {
	for i := 0; i < len(nicRoutes); i++ {
		if regutils.MatchCIDR6(nicRoutes[i][0]) {
			routes6 = addRoute(routes6, nicRoutes[i][0], nicRoutes[i][1])
		} else {
			routes4 = addRoute(routes4, nicRoutes[i][0], nicRoutes[i][1])
		}
	}
	return routes4, routes6
}

func isExitAddress(ip string) bool {
	ipv4, err := netutils.NewIPV4Addr(ip)
	if err != nil {
		log.Errorf("NewIPV4Addr %s fail %s", ip, err)
		return false
	}
	return netutils.IsExitAddress(ipv4)
}

var (
	Ip4MetadataServers = []string{
		"169.254.169.254",
	}
	Ip6MetadataServers = []string{
		"fd00:ec2::254",
	}
)

func SetIp4MetadataServers(ip4s []string) {
	if len(ip4s) == 0 {
		ip4s = []string{
			"169.254.169.254",
		}
	}
	Ip4MetadataServers = ip4s
}

func SetIp6MetadataServers(ip6s []string) {
	if len(ip6s) == 0 {
		ip6s = []string{
			"fd00:ec2::254",
		}
	}
	Ip6MetadataServers = ip6s
}

func AddNicRoutes(routes4 []SRouteInfo, routes6 []SRouteInfo, nicDesc *types.SServerNic, mainIp string, mainIp6 string, nicCnt int) ([]SRouteInfo, []SRouteInfo) {
	// always add static routes, even if this is the default NIC
	// if mainIp == nicDesc.Ip {
	// 	return routes
	// }
	if len(nicDesc.Routes) > 0 {
		routes4, routes6 = extendRoutes(routes4, routes6, nicDesc.Routes)
	} else if len(nicDesc.Gateway) > 0 && !isExitAddress(nicDesc.Ip) &&
		nicCnt == 2 && nicDesc.Ip != mainIp && isExitAddress(mainIp) {
		for _, pref := range netutils.GetPrivateIPRanges() {
			prefs := pref.ToPrefixes()
			for _, p := range prefs {
				routes4 = addRoute(routes4, p.String(), nicDesc.Gateway)
			}
		}
	}

	if len(mainIp) > 0 && nicDesc.Ip == mainIp {
		// always add 169.254.169.254 for default NIC
		for _, ip := range Ip4MetadataServers {
			pref := ip + "/32"
			routes4 = addRoute(routes4, pref, "0.0.0.0")
		}
	}
	if len(mainIp6) > 0 && nicDesc.Ip6 == mainIp6 {
		for _, ip6 := range Ip6MetadataServers {
			pref := ip6 + "/128"
			routes6 = addRoute(routes6, pref, "::")
		}
	}

	return routes4, routes6
}

func GetNicDns(nicdesc *types.SServerNic) []string {
	dnslist := []string{}
	if len(nicdesc.Dns) > 0 {
		for _, dns := range strings.Split(nicdesc.Dns, ",") {
			dnslist = append(dnslist, dns)
		}
	}
	return dnslist
}

func NetBytes2Mask(mask []byte) string {
	if len(mask) != 4 {
		return ""
	}

	var res string
	for i := range mask {
		res += strconv.Itoa(int(mask[i])) + "."
	}
	return res[:len(res)-1]
}

type SNetInterface struct {
	name string
	Addr string
	Mask net.IPMask
	mac  string

	Addr6 string
	Mask6 net.IPMask

	Addr4LinkLocal string
	Addr6LinkLocal string

	Mtu int

	VlanId     int
	VlanParent *SNetInterface

	BondingMode   int
	BondingSlaves []*SNetInterface
}

var (
	SECRET_PREFIX        = "169.254"
	SECRET_MASK          = []byte{255, 255, 255, 255}
	SECRET_MASK_LEN      = 32
	secretInterfaceIndex = 254
)

func NewNetInterface(name string) *SNetInterface {
	n := new(SNetInterface)
	n.name = name
	n.FetchConfig()
	return n
}

func NewNetInterfaceWithExpectIp(name string, expectIp string, expectIp6 string) *SNetInterface {
	n := new(SNetInterface)
	n.name = name
	n.FetchConfig2(expectIp, expectIp6)
	return n
}

func (n *SNetInterface) String() string {
	return n.name
}

func (n *SNetInterface) Exist() bool {
	_, err := net.InterfaceByName(n.name)
	return err == nil
}

func (n *SNetInterface) FetchInter() *net.Interface {
	inter, err := net.InterfaceByName(n.name)
	if err != nil {
		log.Errorf("fetch interface %s error %s", n.name, err)
		return nil
	}
	return inter
}

func (n *SNetInterface) FetchConfig() {
	n.FetchConfig2("", "")
}

// FetchConfig2 is used to fetch config with expectIp and expectIp6
func (n *SNetInterface) FetchConfig2(expectIp string, expectIp6 string) {
	n.Addr = ""
	n.Mask = nil
	n.mac = ""
	// n.Mtu = 0
	inter := n.FetchInter()
	if inter == nil {
		return
	}

	n.Mtu = inter.MTU

	n.mac = inter.HardwareAddr.String()
	addrs, err := inter.Addrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					if strings.HasPrefix(ipnet.IP.To4().String(), SECRET_PREFIX) {
						n.Addr4LinkLocal = ipnet.IP.String()
					} else if (len(expectIp) > 0 && ipnet.IP.String() == expectIp) || (len(expectIp) == 0 && n.Addr == "") {
						n.Addr = ipnet.IP.String()
						n.Mask = ipnet.Mask
					}
				} else if ipnet.IP.To16() != nil {
					if ipnet.IP.IsLinkLocalUnicast() {
						n.Addr6LinkLocal = ipnet.IP.String()
					} else if (len(expectIp6) > 0 && ipnet.IP.String() == expectIp6) || (len(expectIp6) == 0 && n.Addr6 == "") {
						n.Addr6 = ipnet.IP.String()
						n.Mask6 = ipnet.Mask
					}
				}
			}
		}
	}

	// check vlanId
	vlanConf := getVlanConfig(n.name)
	if vlanConf != nil {
		n.VlanId = vlanConf.VlanId
		n.VlanParent = NewNetInterface(vlanConf.Parent)
	} else {
		n.VlanId = 1
		n.VlanParent = nil
	}

	// check bonding
	bondingConf := getBondingConfig(n.name)
	if bondingConf != nil {
		n.BondingMode = bondingConf.Mode
		for _, slave := range bondingConf.Slaves {
			n.BondingSlaves = append(n.BondingSlaves, NewNetInterface(slave))
		}
	}
}

func (n *SNetInterface) GetMac() string {
	return n.mac
}

func (n *SNetInterface) GetHardwareAddr() net.HardwareAddr {
	mac, err := net.ParseMAC(n.mac)
	if err != nil {
		return nil
	}
	return mac
}

func (n *SNetInterface) GetAllMacs() []string {
	macs := make([]string, 0, len(n.BondingSlaves)+1)
	find := false
	for _, inf := range n.BondingSlaves {
		macs = append(macs, inf.GetMac())
		if n.mac == inf.GetMac() {
			find = true
		}
	}
	if !find {
		macs = append(macs, n.mac)
	}
	return macs
}

// https://kris.io/2015/10/01/kvm-network-performance-tso-and-gso-turn-it-off/
// General speaking, it is recommended to turn of GSO
// however, this will degrade host network performance
func (n *SNetInterface) SetupGso(on bool) {
	onoff := "off"
	if on {
		onoff = "on"
	}
	procutils.NewCommand(
		"ethtool", "-K", n.name,
		"tso", onoff, "gso", onoff,
		"ufo", onoff, "lro", onoff,
		"gro", onoff, "tx", onoff,
		"rx", onoff, "sg", onoff).Run()
}

func (n *SNetInterface) IsSecretInterface() bool {
	return n.Addr4LinkLocal != "" && n.Addr == ""
}

func (n *SNetInterface) IsSecretInterface6() bool {
	return n.Addr6LinkLocal != "" && n.Addr6 == ""
}

/*func (n *SNetInterface) IsSecretAddress(addr string, mask []byte) bool {
	log.Infof("MASK --- %s", mask)
	if reflect.DeepEqual(mask, SECRET_MASK) && strings.HasPrefix(addr, SECRET_PREFIX) {
		return true
	} else {
		return false
	}
}*/

func GetSecretInterfaceAddress() (string, int) {
	addr := fmt.Sprintf("%s.%d.1", SECRET_PREFIX, secretInterfaceIndex)
	secretInterfaceIndex -= 1
	return addr, SECRET_MASK_LEN
}

func (n *SNetInterface) GetSlaveAddresses() []SNicAddress {
	addrs := n.GetAddresses()
	var slaves = make([]SNicAddress, 0)
	for _, addr := range addrs {
		if addr.Addr != n.Addr && addr.Addr != n.Addr6 {
			slaves = append(slaves, addr)
		}
	}
	return slaves
}

func FormatMac(macStr string) string {
	var ret = []byte{}
	for i := 0; i < len(macStr); i++ {
		if bytes.IndexByte([]byte("0123456789abcdef"), macStr[i]) >= 0 {
			ret = append(ret, macStr[i])
		} else if bytes.IndexByte([]byte("ABCDEF"), macStr[i]) >= 0 {
			ret = append(ret, byte(unicode.ToLower(rune(macStr[i]))))
		}
	}
	if len(ret) == 12 {
		var res string
		for i := 0; i < 12; i += 2 {
			res += string(ret[i:i+2]) + ":"
		}
		return res[:len(res)-1]
	}
	return ""
}

func MacEqual(mac1, mac2 string) bool {
	mac1 = FormatMac(mac1)
	mac2 = FormatMac(mac2)
	if len(mac1) > 0 && len(mac2) > 0 && mac1 == mac2 {
		return true
	}
	return false
}

func netmask2len(mask string) int {
	masks := []string{"0", "128", "192", "224", "240", "248", "252", "254", "255"}
	for i := 0; i < len(masks); i++ {
		if masks[i] == mask {
			return i
		}
	}
	return -1
}

func Netmask2Len(mask string) int {
	data := strings.Split(mask, ".")
	mlen := 0
	for _, d := range data {
		if d != "0" {
			nle := netmask2len(d)
			log.Errorln(d)
			if nle < 0 {
				return -1
			}
			mlen += nle
		}
	}
	return mlen
}

func PrefixSplit(pref string) (string, int, error) {
	slash := strings.Index(pref, "/")
	var intMask int
	var err error
	if slash > 0 {
		ip := pref[:slash]
		mask := pref[slash+1:]
		if regutils.MatchIPAddr(mask) {
			intMask = Netmask2Len(mask)
		} else {
			intMask, err = strconv.Atoi(mask)
			if err != nil {
				return "", 0, err
			}
		}
		return ip, intMask, nil
	} else {
		return pref, 32, nil
	}
}

func TestTcpPort(ip string, port int, timeoutSecs int, tries int) error {
	if timeoutSecs <= 0 {
		timeoutSecs = 3
	}
	if tries <= 0 {
		tries = 3
	}

	address := net.JoinHostPort(ip, fmt.Sprintf("%d", port))
	// 3 second timeout
	errs := make([]error, 0)
	for i := 0; i < tries; i++ {
		conn, err := net.DialTimeout("tcp", address, time.Duration(timeoutSecs)*time.Second)
		if err != nil {
			errs = append(errs, err)
		} else {
			if conn != nil {
				_ = conn.Close()
				return nil
			} else {
				errs = append(errs, errors.Wrap(errors.ErrEmpty, "nil conn"))
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return errors.NewAggregate(errs)
}

func IP2SolicitMcastIP(ip6 net.IP) net.IP {
	// Solicited-Node Multicast Address, FF02::1:FF00:0/104
	return net.IP{0xff, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0xff, ip6[13], ip6[14], ip6[15]}
}

func IP2SolicitMcastMac(ip6 net.IP) net.HardwareAddr {
	// Solicited-Node Multicast Address, 33:33:ff:00:00:00
	return net.HardwareAddr{0x33, 0x33, 0xff, ip6[13], ip6[14], ip6[15]}
}

func SplitV46Addr(addrsStr string) ([]string, []string) {
	servers4 := stringutils2.NewSortedStrings(nil)
	servers6 := stringutils2.NewSortedStrings(nil)
	for _, ntp := range strings.Split(addrsStr, ",") {
		if regutils.MatchIP4Addr(ntp) {
			servers4 = servers4.Append(ntp)
		} else if regutils.MatchIP6Addr(ntp) {
			servers6 = servers6.Append(ntp)
		} else if regutils.MatchDomainName(ntp) {
			ntpAddrs, _ := net.LookupHost(ntp)
			for _, ntpAddr := range ntpAddrs {
				if regutils.MatchIP4Addr(ntpAddr) {
					servers4 = servers4.Append(ntpAddr)
				} else if regutils.MatchIP6Addr(ntpAddr) {
					servers6 = servers6.Append(ntpAddr)
				}
			}
		}
	}
	return servers4, servers6
}

func SplitV46Addr2IP(addrsStr string) ([]net.IP, []net.IP) {
	addrs4, addrs6 := SplitV46Addr(addrsStr)
	ip4s := make([]net.IP, len(addrs4))
	ip6s := make([]net.IP, len(addrs6))
	for i := range addrs4 {
		ip4s[i] = net.ParseIP(addrs4[i])
	}
	for i := range addrs6 {
		ip6s[i] = net.ParseIP(addrs6[i])
	}
	return ip4s, ip6s
}
