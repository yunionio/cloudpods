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
	"fmt"
	"net"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
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

func NewGuestDHCPServer(iface string, relay []string) (*SGuestDHCPServer, error) {
	var (
		err       error
		guestdhcp = new(SGuestDHCPServer)
	)

	if len(relay) > 0 && len(relay) != 2 {
		return nil, fmt.Errorf("Wrong dhcp relay address")
	}

	guestdhcp.server, guestdhcp.conn, err = dhcp.NewDHCPServer2(iface, uint16(options.HostOptions.DhcpServerPort), DEFAULT_DHCP_CLIENT_PORT)
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

func (s *SGuestDHCPServer) Start() {
	log.Infof("SGuestDHCPServer starting ...")
	go func() {
		err := s.server.ListenAndServe(s)
		if err != nil {
			log.Errorf("DHCP serve error: %s", err)
		}
	}()
}

func (s *SGuestDHCPServer) RelaySetup(addr string) error {
	if s.relay != nil {
		return s.relay.Setup(addr)
	}
	return nil
}

func (s *SGuestDHCPServer) getGuestConfig(guestDesc, guestNic jsonutils.JSONObject) *dhcp.ResponseConfig {
	var nicdesc = new(types.SServerNic)
	if err := guestNic.Unmarshal(nicdesc); err != nil {
		log.Errorln(err)
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
	conf.Routes = route

	if len(nicdesc.Dns) > 0 {
		conf.DNSServer = net.ParseIP(nicdesc.Dns)
	}
	conf.OsName, _ = guestDesc.GetString("os_name")
	conf.LeaseTime = time.Duration(options.HostOptions.DhcpLeaseTime) * time.Second
	conf.RenewalTime = time.Duration(options.HostOptions.DhcpRenewalTime) * time.Second
	return conf
}

func (s *SGuestDHCPServer) getConfig(pkt dhcp.Packet) *dhcp.ResponseConfig {
	var (
		guestmananger = guestman.GetGuestManager()
		mac           = pkt.CHAddr().String()
		ip, port      = "", ""
		isCandidate   = false
	)
	guestDesc, guestNic := guestmananger.GetGuestNicDesc(mac, ip, port, s.iface, isCandidate)
	if guestNic == nil {
		guestDesc, guestNic = guestmananger.GetGuestNicDesc(mac, ip, port, s.iface, !isCandidate)
	}
	if guestNic != nil && !jsonutils.QueryBoolean(guestNic, "virtual", false) {
		return s.getGuestConfig(guestDesc, guestNic)
	}
	return nil
}

func (s *SGuestDHCPServer) IsDhcpPacket(pkt dhcp.Packet) bool {
	return pkt != nil && (pkt.Type() == dhcp.Request || pkt.Type() == dhcp.Discover)
}

func (s *SGuestDHCPServer) ServeDHCP(pkt dhcp.Packet, addr *net.UDPAddr, intf *net.Interface) (dhcp.Packet, []string, error) {
	pkg, err := s.serveDHCPInternal(pkt, addr, intf)
	return pkg, nil, err
}

func (s *SGuestDHCPServer) serveDHCPInternal(pkt dhcp.Packet, addr *net.UDPAddr, intf *net.Interface) (dhcp.Packet, error) {
	if !s.IsDhcpPacket(pkt) {
		return nil, nil, nil
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
