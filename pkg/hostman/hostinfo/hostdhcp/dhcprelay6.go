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
	"sync"
	"time"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/dhcp"
)

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

func (r *SDHCP6Relay) Setup(ctx context.Context, addr string) error {
	r.ipv6srcAddr = net.ParseIP(addr)
	if len(r.ipv6srcAddr) == 0 {
		return fmt.Errorf("Wrong ip address %s", addr)
	}
	log.Infof("DHCP6 Relay Setup on %s %d", addr, DEFAULT_DHCP6_RELAY_PORT)
	var err error
	r.server, err = dhcp.NewDHCP6Server3(addr, DEFAULT_DHCP6_RELAY_PORT)
	if err != nil {
		return err
	}
	go r.server.ListenAndServe(ctx, r)
	return nil
}

func (r *SDHCP6Relay) ServeDHCP(ctx context.Context, pkt dhcp.Packet, cliMac net.HardwareAddr, addr *net.UDPAddr) (dhcp.Packet, []string, error) {
	pkg, err := r.serveDHCPInternal(pkt, addr)
	return pkg, nil, err
}

func (r *SDHCP6Relay) ServeRA(ctx context.Context, pkt dhcp.Packet, cliMac net.HardwareAddr, addr *net.UDPAddr) (dhcp.Packet, error) {
	pkg, err := r.serveDHCPInternal(pkt, addr)
	return pkg, err
}

func (r *SDHCP6Relay) serveDHCPInternal(pkt dhcp.Packet, _ *net.UDPAddr) (dhcp.Packet, error) {
	log.Infof("DHCP Relay Reply TO %s", pkt.CHAddr())
	v, ok := r.cache.Load(pkt.TransactionID())
	if ok {
		r.cache.Delete(pkt.TransactionID())
		val := v.(*SRelayCache)
		udpAddr := &net.UDPAddr{
			IP:   pkt.CIAddr(),
			Port: val.srcPort,
		}
		if err := r.guestDHCPConn.SendDHCP(pkt, udpAddr, pkt.CHAddr()); err != nil {
			log.Errorln(err)
		}
	}
	return nil, nil
}

func (r *SDHCP6Relay) Relay(pkt dhcp.Packet, addr *net.UDPAddr) (dhcp.Packet, error) {
	if addr.IP.Equal(r.ipv6srcAddr) {
		return nil, nil
	}

	log.Infof("Receive DHCP Relay Rquest FROM %s %s", addr.IP, pkt.CHAddr())
	// clean cache first
	var now = time.Now().Add(time.Second * -30)
	r.cache.Range(func(key, value interface{}) bool {
		v := value.(*SRelayCache)
		if v.timer.Before(now) {
			r.cache.Delete(key)
		}
		return true
	})

	// cache pkt info
	r.cache.Store(pkt.TransactionID(), &SRelayCache{
		mac:     pkt.CHAddr(),
		srcPort: addr.Port,
		timer:   time.Now(),
	})

	pkt.SetGIAddr(r.ipv6srcAddr)

	err := r.server.GetConn().SendDHCP(pkt, &net.UDPAddr{IP: r.destaddr, Port: r.destport}, nil)
	return nil, err
}
