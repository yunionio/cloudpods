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
	"strconv"
	"strings"
	"unicode"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/util/procutils"
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

func addRoute(routes [][]string, net, gw string) [][]string {
	for _, rt := range routes {
		if rt[0] == net {
			return routes
		}
	}
	return append(routes, []string{net, gw})
}

func extendRoutes(routes [][]string, nicRoutes []types.SRoute) [][]string {
	for i := 0; i < len(nicRoutes); i++ {
		routes = addRoute(routes, nicRoutes[i][0], nicRoutes[i][1])
	}
	return routes
}

func isExitAddress(ip string) bool {
	ipv4, err := netutils.NewIPV4Addr(ip)
	if err != nil {
		log.Errorf("NewIPV4Addr %s fail %s", ip, err)
		return false
	}
	return netutils.IsExitAddress(ipv4)
}

func AddNicRoutes(routes [][]string, nicDesc *types.SServerNic, mainIp string, nicCnt int) [][]string {
	// always add static routes, even if this is the default NIC
	// if mainIp == nicDesc.Ip {
	// 	return routes
	// }
	if len(nicDesc.Routes) > 0 {
		routes = extendRoutes(routes, nicDesc.Routes)
	} else if len(nicDesc.Gateway) > 0 && !isExitAddress(nicDesc.Ip) &&
		nicCnt == 2 && nicDesc.Ip != mainIp && isExitAddress(mainIp) {
		for _, pref := range netutils.GetPrivateIPRanges() {
			routes = addRoute(routes, pref.String(), nicDesc.Gateway)
		}
	}
	return routes
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
	mac  string

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

func NewNetInterfaceWithExpectIp(name string, expectIp string) *SNetInterface {
	n := new(SNetInterface)
	n.name = name
	n.fetchConfig(expectIp)
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
	n.fetchConfig("")
}

func (n *SNetInterface) fetchConfig(expectIp string) {
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
					n.Addr = ipnet.IP.String()
					n.Mask = ipnet.Mask
					if len(expectIp) > 0 && n.Addr != expectIp {
						continue
					} else {
						break
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

func GetSecretInterfaceAddress() (string, int) {
	addr := fmt.Sprintf("%s.%d.1", SECRET_PREFIX, secretInterfaceIndex)
	secretInterfaceIndex -= 1
	return addr, SECRET_MASK_LEN
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
