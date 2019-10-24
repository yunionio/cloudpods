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
	"strconv"
	"sync"
	"time"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/dhcp"
)

const DEFAULT_DHCP_RELAY_PORT = 68

type recvFunc func(pkt *dhcp.Packet)

type SRelayCache struct {
	mac     net.HardwareAddr
	srcPort int
	// dstPort int

	timer time.Time
}

type SDHCPRelay struct {
	server *dhcp.DHCPServer
	OnRecv recvFunc

	guestDHCPConn *dhcp.Conn
	ipv4srcAddr   net.IP

	destaddr net.IP
	destport int

	cache sync.Map
}

func NewDHCPRelay(guestDHCPConn *dhcp.Conn, addrs []string) (*SDHCPRelay, error) {
	relay := new(SDHCPRelay)
	relay.guestDHCPConn = guestDHCPConn
	addr := addrs[0]
	port, err := strconv.Atoi(addrs[1])
	if err != nil {
		return nil, fmt.Errorf("Pares dhcp relay addrs error %s", err)
	}

	log.Infof("Set Relay To Address: %s, %d", addr, port)
	relay.destaddr = net.ParseIP(addr)
	relay.destport = port
	relay.cache = sync.Map{}

	return relay, nil
}

func (r *SDHCPRelay) Setup(addr string) error {
	r.ipv4srcAddr = net.ParseIP(addr)
	if len(r.ipv4srcAddr) == 0 {
		return fmt.Errorf("Wrong ip address %s", addr)
	}
	log.Infof("DHCP Relay Setup on %s %d", addr, DEFAULT_DHCP_RELAY_PORT)
	var err error
	r.server, err = dhcp.NewDHCPServer3(addr, DEFAULT_DHCP_RELAY_PORT)
	if err != nil {
		return err
	}
	go r.server.ListenAndServe(r)
	return nil
}

func (r *SDHCPRelay) ServeDHCP(pkt dhcp.Packet, addr *net.UDPAddr, intf *net.Interface) (dhcp.Packet, []string, error) {
	pkg, err := r.serveDHCPInternal(pkt, addr, intf)
	return pkg, nil, err
}

func (r *SDHCPRelay) serveDHCPInternal(pkt dhcp.Packet, _ *net.UDPAddr, intf *net.Interface) (dhcp.Packet, error) {
	log.Infof("DHCP Relay Reply TO %s", pkt.CHAddr())
	v, ok := r.cache.Load(pkt.TransactionID())
	if ok {
		r.cache.Delete(pkt.TransactionID())
		val := v.(*SRelayCache)
		udpAddr := &net.UDPAddr{
			IP:   pkt.CIAddr(),
			Port: val.srcPort,
		}
		if err := r.guestDHCPConn.SendDHCP(pkt, udpAddr, pkt.CHAddr(), intf); err != nil {
			log.Errorln(err)
		}
	}
	return nil, nil
}

func (r *SDHCPRelay) Relay(pkt dhcp.Packet, addr *net.UDPAddr, intf *net.Interface) (dhcp.Packet, error) {
	if addr.IP.Equal(r.ipv4srcAddr) {
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

	pkt.SetGIAddr(r.ipv4srcAddr)

	err := r.server.GetConn().SendDHCP(pkt, &net.UDPAddr{IP: r.destaddr, Port: r.destport}, nil, intf)
	return nil, err
}
