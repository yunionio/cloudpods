package hostdhcp

import (
	"net"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/dhcp"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

var DEFAULT_DHCP_LISTEN_ADDR = "0.0.0.0"

const DEFAULT_DHCP_SERVER_PORT = 67

type SGuestDHCPServer struct {
	server *dhcp.DHCPServer
	relay  *SDHCPRelay

	iface string
}

func NewGuestDHCPServer(iface string, relay []string) *SGuestDHCPServer {
	guestdhcp := new(SGuestDHCPServer)
	guestdhcp.relay = NewDHCPRelay(relay)
	guestdhcp.server = dhcp.NewDHCPServer(DEFAULT_DHCP_LISTEN_ADDR, options.HostOptions.DhcpServerPort)
	guestdhcp.iface = iface
	return guestdhcp
}

func (s *SGuestDHCPServer) Start() {
	log.Infof("SGuestDHCPServer starting ...")
	if s.relay != nil {
		s.relay.Start()
	}
	s.server.ListenAndServe(s)
}

func (s *SGuestDHCPServer) RelaySetup(addr string) {
	s.relay.Setup(addr)
}

func (s *SGuestDHCPServer) getGuestConfig(guestDesc, guestNic jsonutils.JSONObject) *dhcp.ResponseConfig {
	var nicdesc = new(types.ServerNic)
	if err := guestNic.Unmarshal(nicdesc); err != nil {
		log.Errorln(err)
		return nil
	}

	var conf = new(dhcp.ResponseConfig)
	nicIp := nicdesc.Ip
	v4Ip, _ := netutils.NewIPV4Addr(nicIp)
	conf.ClientIP = v4Ip.ToBytes()

	masklen := nicdesc.Masklen

	conf.ServerIP = v4Ip.NetAddr(int8(masklen)).ToBytes()
	conf.SubnetMask = net.ParseIP(netutils2.Netlen2Mask(masklen))
	conf.BroadcastAddr = v4Ip.BroadcastAddr(int8(masklen)).ToBytes()
	conf.Hostname, _ = guestDesc.GetString("name")
	conf.Domain = nicdesc.Domain

	// get main ip
	guestNics, _ := guestDesc.GetArray("nics")
	manNic, err := netutils2.GetMainNic(guestNics)
	if err != nil {
		log.Errorln(err)
		return nil
	}
	mainIp, _ := manNic.GetString("ip")

	var route = [][]string{}
	if len(nicdesc.Gateway) > 0 && mainIp == nicIp {
		conf.Gateway = net.ParseIP(nicdesc.Gateway)

		osName, _ := guestDesc.GetString("os_name")
		if len(osName) == 0 {
			osName = "Linux"
		}
		if !strings.HasPrefix(strings.ToLower(osName), "win") {
			route = append(route, []string{"0.0.0.0/0", nicdesc.Gateway})
		}
		route = append(route, []string{"169.254.169.254/32", nicdesc.Gateway})
	}
	netutils2.AddNicRoutes(
		&route, nicdesc, mainIp, len(guestNics), options.HostOptions.PrivatePrefixes)
	// 有问题？
	conf.Routes = route

	if len(nicdesc.Dns) > 0 {
		conf.DNSServer = net.ParseIP(nicdesc.Dns)
	}
	conf.OsName, _ = guestDesc.GetString("os_name")
	return conf
}

func (s *SGuestDHCPServer) getConfig(pkt *dhcp.Packet) *dhcp.ResponseConfig {
	var (
		guestmananger = guestman.GetGuestManager()
		mac           = pkt.HardwareAddr.String()
		ip, port      = "", ""
		isCandidate   = false
	)
	guestDesc, guestNic := guestmananger.GetGuestNicDesc(mac, ip, port, s.iface, isCandidate)
	if guestNic == nil {
		guestDesc, guestNic = guestmananger.GetGuestNicDesc(mac, ip, port, s.iface, !isCandidate)
	}
	if guestNic != nil && jsonutils.QueryBoolean(guestNic, "virtual", false) {
		return s.getGuestConfig(guestDesc, guestNic)
	}
	return nil
}

func (s *SGuestDHCPServer) ServeDHCP(pkt *dhcp.Packet, intf *net.Interface) (*dhcp.Packet, error) {
	var conf = s.getConfig(pkt)
	if conf != nil {
		return dhcp.MakeReplyPacket(pkt, conf)
	} else if s.relay != nil {
		return s.relay.Relay(pkt, intf)
	}
	return nil, nil
}
