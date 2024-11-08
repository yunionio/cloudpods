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
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	guestman "yunion.io/x/onecloud/pkg/hostman/guestman/types"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/dhcp"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

const DEFAULT_DHCP_CLIENT_PORT = 68

type SGuestDHCPServer struct {
	server *dhcp.DHCPServer
	relay  *SDHCPRelay
	conn   *dhcp.Conn

	iface string
}

func NewGuestDHCPServer(iface string, port int, relay []string) (*SGuestDHCPServer, error) {
	var (
		err       error
		guestdhcp = new(SGuestDHCPServer)
	)

	if len(relay) > 0 && len(relay) != 2 {
		return nil, fmt.Errorf("Wrong dhcp relay address")
	}

	guestdhcp.server, guestdhcp.conn, err = dhcp.NewDHCPServer2(iface, uint16(port), DEFAULT_DHCP_CLIENT_PORT)
	if err != nil {
		return nil, err
	}

	if len(relay) == 2 {
		guestdhcp.relay, err = NewDHCPRelay(guestdhcp.conn, relay)
		if err != nil {
			return nil, err
		}
	}

	guestdhcp.iface = iface
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
	return nil
}

func (s *SGuestDHCPServer) getGuestConfig(
	guestDesc *desc.SGuestDesc, guestNic *desc.SGuestNetwork,
) *dhcp.ResponseConfig {
	var nicdesc = new(types.SServerNic)
	if err := gusetnetworkJsonDescToServerNic(nicdesc, guestNic); err != nil {
		log.Errorf("failed convert server nic desc")
		return nil
	}

	var conf = new(dhcp.ResponseConfig)
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

	// get main ip
	guestNics := guestDesc.Nics
	manNic := GetMainNic(guestNics)
	var mainIp string
	if manNic != nil {
		mainIp = manNic.Ip
	}

	var route = [][]string{}
	if nicdesc.IsDefault {
		conf.Gateway = net.ParseIP(nicdesc.Gateway)

		osName := guestDesc.OsName
		if len(osName) == 0 {
			osName = "Linux"
		}
		if !strings.HasPrefix(strings.ToLower(osName), "win") {
			route = append(route, []string{"0.0.0.0/0", nicdesc.Gateway})
		}
		route = append(route, []string{"169.254.169.254/32", "0.0.0.0"})
	}
	route = netutils2.AddNicRoutes(route, nicdesc, mainIp, len(guestNics))
	conf.Routes = route

	if len(nicdesc.Dns) > 0 {
		conf.DNSServers = make([]net.IP, 0)
		for _, dns := range strings.Split(nicdesc.Dns, ",") {
			conf.DNSServers = append(conf.DNSServers, net.ParseIP(dns))
		}
	}

	if len(nicdesc.Ntp) > 0 {
		conf.NTPServers = make([]net.IP, 0)
		for _, ntp := range strings.Split(nicdesc.Ntp, ",") {
			conf.NTPServers = append(conf.NTPServers, net.ParseIP(ntp))
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
	guestDesc, guestNic := guestman.GuestDescGetter.GetGuestNicDesc(mac, ip, port, s.iface, isCandidate)
	if guestNic == nil {
		guestDesc, guestNic = guestman.GuestDescGetter.GetGuestNicDesc(mac, ip, port, s.iface, !isCandidate)
	}
	if guestNic != nil && !guestNic.Virtual {
		return s.getGuestConfig(guestDesc, guestNic)
	}
	return nil
}

func (s *SGuestDHCPServer) IsDhcpPacket(pkt dhcp.Packet) bool {
	return pkt != nil && (pkt.Type() == dhcp.Request || pkt.Type() == dhcp.Discover)
}

func (s *SGuestDHCPServer) ServeDHCP(ctx context.Context, pkt dhcp.Packet, addr *net.UDPAddr, intf *net.Interface) (dhcp.Packet, []string, error) {
	pkg, err := s.serveDHCPInternal(pkt, addr, intf)
	return pkg, nil, err
}

func (s *SGuestDHCPServer) serveDHCPInternal(pkt dhcp.Packet, addr *net.UDPAddr, intf *net.Interface) (dhcp.Packet, error) {
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
		return s.relay.Relay(pkt, addr, intf)
	}
	return nil, nil
}
