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
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/dhcp"
)

type SRelayCache6 struct {
	peerMac net.HardwareAddr
	peerUdp *net.UDPAddr

	linkAddr net.IP
	peerAddr net.IP

	timer time.Time
}

type SDHCP6Relay struct {
	server *dhcp.DHCP6Server
	OnRecv recvFunc

	guestDHCPConn *dhcp.Conn
	ipv6srcAddr   net.IP

	destaddr net.IP
	destport int

	cache sync.Map
}

func NewDHCP6Relay(guestDHCPConn *dhcp.Conn, config *SDHCPRelayUpstream) (*SDHCP6Relay, error) {
	relay := new(SDHCP6Relay)
	relay.guestDHCPConn = guestDHCPConn

	log.Infof("Set Relay To Address: %s, %d", config.IP, config.Port)
	relay.destaddr = net.ParseIP(config.IP)
	relay.destport = config.Port
	relay.cache = sync.Map{}

	return relay, nil
}

func (r *SDHCP6Relay) Setup(addr string) error {
	r.ipv6srcAddr = net.ParseIP(addr)
	if len(r.ipv6srcAddr) == 0 {
		return fmt.Errorf("wrong ip address %s", addr)
	}
	log.Infof("DHCP6 Relay Setup on %s %d", addr, DEFAULT_DHCP6_RELAY_PORT)
	var err error
	r.server, err = dhcp.NewDHCP6Server3(addr, DEFAULT_DHCP6_RELAY_PORT)
	if err != nil {
		return errors.Wrapf(err, "NewDHCP6Server3")
	}
	go r.server.ListenAndServe(r)
	return nil
}

func (r *SDHCP6Relay) ServeDHCP(pkt dhcp.Packet, cliMac net.HardwareAddr, addr *net.UDPAddr) (dhcp.Packet, []string, error) {
	pkg, err := r.serveDHCPInternal(pkt, addr)
	return pkg, nil, err
}

func (r *SDHCP6Relay) OnRecvICMP6(pkt dhcp.Packet) error {
	// null operation
	return nil
}

func getSessionKey(tid uint32, clientID []byte) string {
	return fmt.Sprintf("%x-%x", tid, clientID)
}

func (r *SDHCP6Relay) serveDHCPInternal(pkt dhcp.Packet, _ *net.UDPAddr) (dhcp.Packet, error) {
	if pkt.Type6() != dhcp.DHCPV6_RELAY_REPL {
		return nil, errors.Wrapf(errors.ErrInvalidFormat, "not a valid relay reply message")
	}

	hopCount := pkt.HopCount()
	decapPkt, err := dhcp.DecapDHCP6RelayMsg(pkt)
	if err != nil {
		return nil, errors.Wrapf(err, "DecapDHCP6RelayMsg")
	}
	tid, err := decapPkt.TID6()
	if err != nil {
		return nil, errors.Wrapf(err, "TID6")
	}
	cliID, err := decapPkt.ClientID()
	if err != nil {
		return nil, errors.Wrapf(err, "ClientID")
	}

	key := getSessionKey(tid, cliID)
	v, ok := r.cache.Load(key)
	if ok {
		r.cache.Delete(key)
		val := v.(*SRelayCache6)

		if hopCount > 1 {
			pkt.SetHopCount(hopCount - 1)
			pkt.SetLinkAddr(val.linkAddr)
			pkt.SetPeerAddr(val.peerAddr)
			if err := r.server.GetConn().SendDHCP(pkt, val.peerUdp, val.peerMac); err != nil {
				log.Errorf("send relay packet to client %s %s failed: %s", val.peerUdp, val.peerMac, err)
			}
		} else {
			pkt = decapPkt
			if err := r.guestDHCPConn.SendDHCP(pkt, val.peerUdp, val.peerMac); err != nil {
				log.Errorf("last hop send dhcp packet to client %s %s failed: %s", val.peerUdp, val.peerMac, err)
			}
		}
	}
	return nil, nil
}

func (r *SDHCP6Relay) Relay(pkt dhcp.Packet, cliMac net.HardwareAddr, cliAddr *net.UDPAddr) (dhcp.Packet, error) {
	if cliAddr.IP.Equal(r.ipv6srcAddr) {
		// come from local? ignore it
		return nil, nil
	}

	log.Infof("Receive IPv6 DHCPRequest FROM %s, relay to upstream %s:%d", cliAddr.IP, r.destaddr, r.destport)

	if pkt.Type6() == dhcp.DHCPV6_RELAY_REPL {
		return nil, errors.Wrapf(errors.ErrInvalidFormat, "cannot relay a reply message")
	}

	session := &SRelayCache6{
		peerMac: cliMac,
		peerUdp: cliAddr,
		timer:   time.Now(),
	}
	hopCount := uint8(0)
	if pkt.Type6() == dhcp.DHCPV6_RELAY_FORW {
		hopCount = pkt.HopCount()
		session.linkAddr = pkt.LinkAddr()
		session.peerAddr = pkt.PeerAddr()
	} else {
		pkt = dhcp.EncapDHCP6RelayMsg(pkt)
	}
	pkt.SetHopCount(hopCount + 1)
	pkt.SetLinkAddr(r.ipv6srcAddr)
	pkt.SetPeerAddr(cliAddr.IP)

	tid, err := pkt.TID6()
	if err != nil {
		return nil, errors.Wrapf(err, "TID6")
	}
	cliID, err := pkt.ClientID()
	if err != nil {
		return nil, errors.Wrapf(err, "ClientID")
	}
	sessionKey := getSessionKey(tid, cliID)
	r.cache.Store(sessionKey, session)

	err = r.server.GetConn().SendDHCP(pkt, &net.UDPAddr{IP: r.destaddr, Port: r.destport}, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "SendDHCP to upstream %s:%d", r.destaddr, r.destport)
	}

	return nil, nil
}
