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

package hostdhcp

import (
	"net"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	guestman "yunion.io/x/onecloud/pkg/hostman/guestman/types"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/dhcp"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

const (
	DEFAULT_DHCP_SERVER_PORT = 67
	// DEFAULT_DHCP_CLIENT_PORT = 68
	DEFAULT_DHCP_RELAY_PORT = 68
)

type SGuestDHCPServer struct {
	server *dhcp.DHCPServer
	relay  *SDHCPRelay
	conn   *dhcp.Conn

	ifaceDev *netutils2.SNetInterface
}

type SDHCPRelayUpstream struct {
	IP   string
	Port int
}

func NewGuestDHCPServer(iface string, port int, relay *SDHCPRelayUpstream) (*SGuestDHCPServer, error) {
	var (
		err       error
		guestdhcp = new(SGuestDHCPServer)
	)

	dev := netutils2.NewNetInterface(iface)
	if dev.GetHardwareAddr() == nil {
		return nil, errors.Wrapf(errors.ErrInvalidStatus, "iface %s no mac", iface)
	}

	guestdhcp.ifaceDev = dev

	guestdhcp.server, guestdhcp.conn, err = dhcp.NewDHCPServer2(iface, DEFAULT_DHCP_SERVER_PORT)
	if err != nil {
		return nil, errors.Wrap(err, "dhcp.NewDHCPServer2")
	}

	if relay != nil {
		guestdhcp.relay, err = NewDHCPRelay(guestdhcp.conn, relay)
		if err != nil {
			return nil, errors.Wrap(err, "NewDHCPRelay")
		}
	}

	return guestdhcp, nil
}

func (s *SGuestDHCPServer) Start(blocking bool) {
	log.Infof("SGuestDHCPServer starting ...")
	serve := func() {
		err := s.server.ListenAndServe(s)
		if err != nil {
			log.Errorf("DHCP serve error: %s", err)
		}
	}
	if blocking {
		serve()
	} else {
		go serve()
	}
}

func (s *SGuestDHCPServer) RelaySetup(addr string) error {
	if s.relay != nil {
		return s.relay.Setup(addr)
	}
	return nil
}

func gusetnetworkJsonDescToServerNic(nicdesc *types.SServerNic, guestNic *desc.SGuestNetwork) error {
	if guestNic.Routes != nil {
		if err := guestNic.Routes.Unmarshal(&nicdesc.Routes); err != nil {
			return err
		}
	}

	nicdesc.Index = int(guestNic.Index)
	nicdesc.Bridge = guestNic.Bridge
	if !apis.IsIllegalSearchDomain(guestNic.Domain) {
		nicdesc.Domain = guestNic.Domain
	}
	nicdesc.Ip = guestNic.Ip
	nicdesc.Vlan = guestNic.Vlan
	nicdesc.Driver = guestNic.Driver
	nicdesc.Masklen = int(guestNic.Masklen)
	nicdesc.Virtual = guestNic.Virtual
	if guestNic.Manual != nil {
		nicdesc.Manual = *guestNic.Manual
	}
	nicdesc.WireId = guestNic.WireId
	nicdesc.NetId = guestNic.NetId
	nicdesc.Mac = guestNic.Mac
	nicdesc.Mtu = guestNic.Mtu
	nicdesc.Dns = guestNic.Dns
	nicdesc.Ntp = guestNic.Ntp
	nicdesc.Net = guestNic.Net
	nicdesc.Interface = guestNic.Interface
	nicdesc.Gateway = guestNic.Gateway
	nicdesc.Ifname = guestNic.Ifname
	nicdesc.NicType = guestNic.NicType
	nicdesc.LinkUp = guestNic.LinkUp
	nicdesc.TeamWith = guestNic.TeamWith

	nicdesc.IsDefault = guestNic.IsDefault

	nicdesc.Ip6 = guestNic.Ip6
	nicdesc.Masklen6 = int(guestNic.Masklen6)
	nicdesc.Gateway6 = guestNic.Gateway6

	return nil
}

func GetMainNic(nics []*desc.SGuestNetwork) *desc.SGuestNetwork {
	for _, n := range nics {
		if n.IsDefault {
			return n
		}
	}
	for _, n := range nics {
		if n.Ip != "" && n.Gateway != "" {
			return n
		}
	}
	return nil
}

func GetMainNic6(nics []*desc.SGuestNetwork) *desc.SGuestNetwork {
	for _, n := range nics {
		if n.IsDefault {
			return n
		}
	}
	for _, n := range nics {
		if n.Ip6 != "" && n.Gateway6 != "" {
			return n
		}
	}
	return nil
}

func getGuestConfig(
	guestDesc *desc.SGuestDesc, guestNic *desc.SGuestNetwork,
	serverMac net.HardwareAddr,
) *dhcp.ResponseConfig {
	var nicdesc = new(types.SServerNic)
	if err := gusetnetworkJsonDescToServerNic(nicdesc, guestNic); err != nil {
		log.Errorf("failed convert server nic desc")
		return nil
	}

	var conf = new(dhcp.ResponseConfig)

	conf.InterfaceMac = serverMac

	nicIp := nicdesc.Ip
	v4Ip, _ := netutils.NewIPV4Addr(nicIp)
	conf.ClientIP = net.ParseIP(nicdesc.Ip)

	masklen := nicdesc.Masklen
	conf.ServerIP = net.ParseIP(v4Ip.NetAddr(int8(masklen)).String())
	conf.SubnetMask = net.ParseIP(netutils2.Netlen2Mask(int(masklen)))
	conf.BroadcastAddr = v4Ip.BroadcastAddr(int8(masklen)).ToBytes()
	if len(guestDesc.Hostname) > 0 {
		conf.Hostname = guestDesc.Hostname
	} else {
		conf.Hostname = guestDesc.Name
	}
	conf.Hostname = strings.ToLower(conf.Hostname)
	conf.Domain = nicdesc.Domain

	if len(nicdesc.Ip6) > 0 {
		// ipv6
		conf.ClientIP6 = net.ParseIP(nicdesc.Ip6)
	}

	// get main ip
	guestNics := guestDesc.Nics
	mainNic := GetMainNic(guestNics)
	var mainIp string
	if mainNic != nil {
		mainIp = mainNic.Ip
	}
	mainNic6 := GetMainNic6(guestNics)
	var mainIp6 string
	if mainNic6 != nil {
		mainIp6 = mainNic6.Ip6
	}

	route4 := make([]dhcp.SRouteInfo, 0)
	route6 := make([]dhcp.SRouteInfo, 0)
	if nicdesc.IsDefault {
		osName := guestDesc.OsName
		if len(osName) == 0 {
			osName = "Linux"
		}

		if nicdesc.Gateway != "" {
			conf.Gateway = net.ParseIP(nicdesc.Gateway)

			if !strings.HasPrefix(strings.ToLower(osName), "win") {
				route4 = append(route4, dhcp.SRouteInfo{
					Prefix:    net.ParseIP("0.0.0.0"),
					PrefixLen: 0,
					Gateway:   net.ParseIP(nicdesc.Gateway),
				})
			}
		}

		if len(nicdesc.Gateway6) > 0 {
			conf.Gateway6 = net.ParseIP(nicdesc.Gateway6)
			conf.PrefixLen6 = uint8(nicdesc.Masklen6)
			route6 = append(route6, dhcp.SRouteInfo{
				Prefix:    net.ParseIP("::"),
				PrefixLen: 0,
				Gateway:   net.ParseIP(nicdesc.Gateway6),
			})
		}
	}
	route4, route6 = netutils2.AddNicRoutes(route4, route6, nicdesc, mainIp, mainIp6, len(guestNics))

	conf.Routes = route4
	conf.Routes6 = route6

	if len(nicdesc.Dns) > 0 {
		conf.DNSServers = make([]net.IP, 0)
		conf.DNSServers6 = make([]net.IP, 0)
		for _, dns := range strings.Split(nicdesc.Dns, ",") {
			if regutils.MatchIP4Addr(dns) {
				conf.DNSServers = append(conf.DNSServers, net.ParseIP(dns))
			} else if regutils.MatchIP6Addr(dns) {
				conf.DNSServers6 = append(conf.DNSServers6, net.ParseIP(dns))
			}
		}
	}

	if len(nicdesc.Ntp) > 0 {
		conf.NTPServers = make([]net.IP, 0)
		conf.NTPServers6 = make([]net.IP, 0)
		for _, ntp := range strings.Split(nicdesc.Ntp, ",") {
			if regutils.MatchIP4Addr(ntp) {
				conf.NTPServers = append(conf.NTPServers, net.ParseIP(ntp))
			} else if regutils.MatchIP6Addr(ntp) {
				conf.NTPServers6 = append(conf.NTPServers6, net.ParseIP(ntp))
			} else if regutils.MatchDomainName(ntp) {
				ntpAddrs, _ := net.LookupHost(ntp)
				for _, ntpAddr := range ntpAddrs {
					if regutils.MatchIP4Addr(ntpAddr) {
						conf.NTPServers = append(conf.NTPServers, net.ParseIP(ntpAddr))
					} else if regutils.MatchIP6Addr(ntpAddr) {
						conf.NTPServers6 = append(conf.NTPServers6, net.ParseIP(ntpAddr))
					}
				}
			}
		}
	}

	if nicdesc.Mtu > 0 {
		conf.MTU = uint16(nicdesc.Mtu)
	}

	conf.OsName = guestDesc.OsName
	conf.LeaseTime = time.Duration(options.HostOptions.DhcpLeaseTime) * time.Second
	conf.RenewalTime = time.Duration(options.HostOptions.DhcpRenewalTime) * time.Second
	return conf
}

func (s *SGuestDHCPServer) getConfig(pkt dhcp.Packet) *dhcp.ResponseConfig {
	if guestman.GuestDescGetter == nil {
		return nil
	}

	var (
		mac         = pkt.CHAddr().String()
		ip, port    = "", ""
		isCandidate = false
	)
	guestDesc, guestNic := guestman.GuestDescGetter.GetGuestNicDesc(mac, ip, port, s.ifaceDev.String(), isCandidate)
	if guestNic == nil {
		guestDesc, guestNic = guestman.GuestDescGetter.GetGuestNicDesc(mac, ip, port, s.ifaceDev.String(), !isCandidate)
	}
	if guestNic != nil && !guestNic.Virtual && len(guestNic.Ip6) > 0 {
		return getGuestConfig(guestDesc, guestNic, s.ifaceDev.GetHardwareAddr())
	}
	return nil
}

func (s *SGuestDHCPServer) IsDhcpPacket(pkt dhcp.Packet) bool {
	return pkt != nil && (pkt.Type() == dhcp.Request || pkt.Type() == dhcp.Discover)
}

func (s *SGuestDHCPServer) ServeDHCP(pkt dhcp.Packet, cliMac net.HardwareAddr, addr *net.UDPAddr) (dhcp.Packet, []string, error) {
	pkg, err := s.serveDHCPInternal(pkt, addr)
	return pkg, nil, err
}

func (s *SGuestDHCPServer) serveDHCPInternal(pkt dhcp.Packet, addr *net.UDPAddr) (dhcp.Packet, error) {
	if !s.IsDhcpPacket(pkt) {
		return nil, nil
	}
	var conf = s.getConfig(pkt)
	if conf != nil {
		log.Infof("Make DHCP Reply %s TO %s", conf.ClientIP, pkt.CHAddr())
		// Guest request ip
		return dhcp.MakeReplyPacket(pkt, conf)
	} else if s.relay != nil && s.relay.server != nil {
		// Host agent as dhcp relay, relay to baremetal
		return s.relay.Relay(pkt, addr)
	}
	return nil, nil
}
