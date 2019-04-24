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
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/regutils2"
)

var PSEUDO_VIP = "169.254.169.231"
var MASKS = []string{"0", "128", "192", "224", "240", "248", "252", "254", "255"}

var PRIVATE_PREFIXES = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
}

func IsTcpPortUsed(addr string, port int) bool {
	conn, _ := net.Dial("tcp", fmt.Sprintf("%s:%d", addr, port))
	if conn != nil {
		conn.Close()
		log.Infof("Tcp port in use: %s %d", addr, port)
		return true
	} else {
		return false
	}
}

// MyIP returns source ip used to communicate with udp:114.114.114.114:53
func MyIP() (ip string, err error) {
	conn, err := net.Dial("udp4", "114.114.114.114:53")
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

func GetPrivatePrefixes(privatePrefixes []string) []string {
	if privatePrefixes != nil {
		return privatePrefixes
	} else {
		return PRIVATE_PREFIXES
	}
}

func GetMainNic(nics []jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var mainIp netutils.IPV4Addr
	var mainNic jsonutils.JSONObject
	for _, n := range nics {
		if n.Contains("gateway") {
			ip, _ := n.GetString("ip")
			ipInt, err := netutils.NewIPV4Addr(ip)
			if err != nil {
				return nil, err
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
	return mainNic, nil
}

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

func addRoute(routes *[][]string, net, gw string) {
	for _, rt := range *routes {
		if rt[0] == net {
			return
		}
	}
	*routes = append(*routes, []string{net, gw})
}

func extendRoutes(routes *[][]string, nicRoutes []types.SRoute) error {
	for i := 0; i < len(nicRoutes); i++ {
		addRoute(routes, nicRoutes[i][0], nicRoutes[i][1])
	}
	return nil
}

func isExitAddress(ip string) bool {
	ipv4, err := netutils.NewIPV4Addr(ip)
	if err != nil {
		return false
	}
	return !netutils.IsPrivate(ipv4) || netutils.IsHostLocal(ipv4) || netutils.IsLinkLocal(ipv4)
}

func AddNicRoutes(routes *[][]string, nicDesc *types.SServerNic, mainIp string, nicCnt int, privatePrefixes []string) {
	if mainIp == nicDesc.Ip {
		return
	}
	if len(nicDesc.Routes) > 0 {
		extendRoutes(routes, nicDesc.Routes)
	} else if len(nicDesc.Gateway) > 0 && isExitAddress(nicDesc.Ip) &&
		nicCnt == 2 && nicDesc.Ip != mainIp && isExitAddress(mainIp) {
		for _, pref := range GetPrivatePrefixes(privatePrefixes) {
			addRoute(routes, pref, nicDesc.Gateway)
		}
	}
}

func GetNicDns(nicdesc *types.SServerNic) []string {
	dnslist := []string{}
	if len(nicdesc.Dns) > 0 {
		dnslist = append(dnslist, nicdesc.Dns)
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
	Mac  string

	// Mtu int
}

var SECRET_PREFIX = "169.254"
var SECRET_MASK = []byte{255, 255, 255, 255}
var secretInterfaceIndex = 254

func NewNetInterface(name string) *SNetInterface {
	n := new(SNetInterface)
	n.name = name
	n.FetchConfig()
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
		log.Errorln(err)
		return nil
	}
	return inter
}

func (n *SNetInterface) FetchConfig() {
	n.Addr = ""
	n.Mask = nil
	n.Mac = ""
	// n.Mtu = 0
	inter := n.FetchInter()
	if inter == nil {
		return
	}

	n.Mac = inter.HardwareAddr.String()
	addrs, err := inter.Addrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					n.Addr = ipnet.IP.String()
					n.Mask = ipnet.Mask
					break
				}
			}
		}
	}

	// mtuStr, err := fileutils2.FileGetContents(fmt.Sprintf("/sys/class/net/%s/mtu", n.name))
	// if err != nil {
	// 	log.Errorln("Fail to read MTU for %s: %s", n.name, err)
	// }

	// n.Mtu, err = strconv.Atoi(mtuStr)
	// if err != nil {
	// 	log.Errorln("Fail to read MTU for %s: %s", n.name, err)
	// }
}

func (n *SNetInterface) DisableGso() {
	procutils.NewCommand(
		"ethtool", "-K", n.name,
		"tso", "off", "gso", "off",
		"gro", "off", "tx", "off",
		"rx", "off", "sg", "off").Run()
}

func (n *SNetInterface) IsSecretInterface() bool {
	return n.IsSecretAddress(n.Addr, n.Mask)
}

func (n *SNetInterface) IsSecretAddress(addr string, mask []byte) bool {
	log.Infof("MASK --- %s", mask)
	if reflect.DeepEqual(mask, SECRET_MASK) && strings.HasPrefix(addr, SECRET_PREFIX) {
		return true
	} else {
		return false
	}
}

func GetSecretInterfaceAddress() (string, []byte) {
	addr := fmt.Sprintf("%s.%d.1", SECRET_PREFIX, secretInterfaceIndex)
	secretInterfaceIndex -= 1
	return addr, SECRET_MASK
}

func (n *SNetInterface) GetRoutes(gwOnly bool) [][]string {
	output, err := procutils.NewCommand("route", "-n").Run()
	if err != nil {
		return nil
	}
	return n.getRoutes(gwOnly, strings.Split(string(output), "\n"))
}

func (n *SNetInterface) getRoutes(gwOnly bool, outputs []string) [][]string {
	re := regexp.MustCompile(`(?P<dest>[0-9.]+)\s+(?P<gw>[0-9.]+)\s+(?P<mask>[0-9.]+)` +
		`\s+[A-Z!]+\s+[0-9]+\s+[0-9]+\s+[0-9]+\s+` + n.name)

	var res [][]string = make([][]string, 0)
	for _, line := range outputs {
		m := regutils2.GetParams(re, line)
		if len(m) > 0 && (!gwOnly || m["gw"] != "0.0.0.0") {
			res = append(res, []string{m["dest"], m["gw"], m["mask"]})
		}
	}
	return res
}

func (n *SNetInterface) getAddresses(output []string) [][]string {
	var addrs = make([][]string, 0)
	re := regexp.MustCompile(`inet (?P<addr>[0-9.]+)/(?P<mask>[0-9]+) `)
	for _, line := range output {
		m := regutils2.GetParams(re, line)
		if len(m) > 0 {
			addrs = append(addrs, []string{m["addr"], m["mask"]})
		}
	}
	return addrs
}

func (n *SNetInterface) GetAddresses() [][]string {
	output, err := procutils.NewCommand("ip", "address", "show", "dev", n.name).Run()
	if err != nil {
		log.Errorln(err)
		return nil
	}
	return n.getAddresses(strings.Split(string(output), "\n"))
}

func (n *SNetInterface) GetSlaveAddresses() [][]string {
	addrs := n.GetAddresses()
	var slaves = make([][]string, 0)
	for _, addr := range addrs {
		if addr[0] != n.Addr {
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
