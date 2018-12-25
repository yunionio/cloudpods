package netutils2

import (
	"net"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
)

var PSEUDO_VIP = "169.254.169.231"
var MASKS = []string{"0", "128", "192", "224", "240", "248", "252", "254", "255"}

var PRIVATE_PREFIXES = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
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
			if mainIp > 0 {
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

func extendRoutes(routes *[][]string, nicRoutes []types.Route) error {
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

func AddNicRoutes(routes *[][]string, nicDesc *types.ServerNic, mainIp string, nicCnt int, privatePrefixes []string) {
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

func GetNicDns(nicdesc *types.ServerNic) []string {
	dnslist := []string{}
	if len(nicdesc.Dns) > 0 {
		dnslist = append(dnslist, nicdesc.Dns)
	}
	return dnslist
}

type SNetInterface struct {
	name string
	addr string
	mask string
	mac  string
}

func NewNetInterface(name string) *SNetInterface {
	n := new(SNetInterface)
	n.name = name
	inter, err := net.InterfaceByName(name)
	if err != nil {
		log.Errorln(err)
		return n
	}
	n.mac = inter.HardwareAddr.String()
	addrs, err := inter.Addrs()
	if err != nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					n.addr = ipnet.IP.String()
					n.mask = ipnet.Mask.String()
					break
				}
			}
		}
	}
	return n
}
