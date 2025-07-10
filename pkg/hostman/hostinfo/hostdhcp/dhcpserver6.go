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

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	guestman "yunion.io/x/onecloud/pkg/hostman/guestman/types"
	"yunion.io/x/onecloud/pkg/util/dhcp"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

const (
	DEFAULT_DHCP6_SERVER_PORT = 547
	// DEFAULT_DHCP_CLIENT_PORT = 68
	DEFAULT_DHCP6_RELAY_PORT = 546
)

type SGuestDHCP6Server struct {
	server *dhcp.DHCP6Server
	relay  *SDHCP6Relay
	conn   *dhcp.Conn

	ifaceDev *netutils2.SNetInterface
}

func NewGuestDHCP6Server(iface string, port int, relay *SDHCPRelayUpstream) (*SGuestDHCP6Server, error) {
	var (
		err       error
		guestdhcp = new(SGuestDHCP6Server)
	)

	dev := netutils2.NewNetInterface(iface)
	if dev.GetHardwareAddr() == nil {
		return nil, errors.Wrapf(errors.ErrInvalidStatus, "iface %s no mac", iface)
	}

	guestdhcp.ifaceDev = dev

	guestdhcp.server, guestdhcp.conn, err = dhcp.NewDHCP6Server2(iface, DEFAULT_DHCP6_SERVER_PORT)
	if err != nil {
		return nil, errors.Wrap(err, "dhcp.NewDHCP6Server2")
	}

	if relay != nil {
		guestdhcp.relay, err = NewDHCP6Relay(guestdhcp.conn, relay)
		if err != nil {
			return nil, errors.Wrap(err, "NewDHCP6Relay")
		}
	}

	return guestdhcp, nil
}

func (s *SGuestDHCP6Server) Start(blocking bool) {
	log.Infof("SGuestDHCP6Server starting ...")
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

func (s *SGuestDHCP6Server) RelaySetup(addr string) error {
	if s.relay != nil {
		return s.relay.Setup(addr)
	}
	return nil
}

func (s *SGuestDHCP6Server) getConfig(cliMac net.HardwareAddr, _ dhcp.Packet) *dhcp.ResponseConfig {
	if guestman.GuestDescGetter == nil {
		return nil
	}

	var (
		ip, port    = "", ""
		isCandidate = false
	)
	guestDesc, guestNic := guestman.GuestDescGetter.GetGuestNicDesc(cliMac.String(), ip, port, s.ifaceDev.String(), isCandidate)
	if guestNic == nil {
		guestDesc, guestNic = guestman.GuestDescGetter.GetGuestNicDesc(cliMac.String(), ip, port, s.ifaceDev.String(), !isCandidate)
	}
	if guestNic != nil && !guestNic.Virtual && len(guestNic.Ip6) > 0 {
		return getGuestConfig(guestDesc, guestNic, s.ifaceDev.GetHardwareAddr())
	}
	return nil
}

func (s *SGuestDHCP6Server) ServeDHCP(pkt dhcp.Packet, cliMac net.HardwareAddr, addr *net.UDPAddr) (dhcp.Packet, []string, error) {
	pkg, err := s.serveDHCPInternal(pkt, cliMac, addr)
	return pkg, nil, err
}

func (s *SGuestDHCP6Server) OnRecvICMP6(pkt dhcp.Packet, cliMac net.HardwareAddr, addr *net.UDPAddr) error {
	log.Infof("SGuestDHCP6Server ServeRA from %s", cliMac.String())
	return nil
}

func (s *SGuestDHCP6Server) serveDHCPInternal(pkt dhcp.Packet, cliMac net.HardwareAddr, addr *net.UDPAddr) (dhcp.Packet, error) {
	var conf = s.getConfig(cliMac, pkt)
	if conf != nil {
		log.Infof("Make DHCPv6 Reply %s TO %s %s", conf.ClientIP6, cliMac.String(), addr.String())
		// Guest request ip
		return dhcp.MakeDHCP6Reply(pkt, conf)
	} else if s.relay != nil && s.relay.server != nil {
		// Host agent as dhcp relay, relay to baremetal
		return s.relay.Relay(pkt, addr)
	}
	return nil, nil
}
