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
	conn   *dhcp.Conn

	guestDHCPConn *dhcp.Conn

	srcaddr     string
	ipv4srcAddr net.IP

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
		log.Errorln(err)
		return nil, err
	}

	log.Infof("Set Relay To Address: %s, %d", addr, port)
	relay.destaddr = net.ParseIP(addr)
	relay.destport = port
	relay.cache = sync.Map{}

	return relay, nil
}

func (r *SDHCPRelay) Start() {
	log.Infof("DHCPRelay starting ...")
	go func() {
		err := r.server.ListenAndServe(r)
		if err != nil {
			log.Errorf("DHCP Relay error %s", err)
		}
	}()
}

func (r *SDHCPRelay) Setup(addr string) error {
	var err error
	r.srcaddr = addr
	r.ipv4srcAddr = net.ParseIP(addr)
	log.Infof("DHCP Relay Server Bind addr %s port %d", r.srcaddr, DEFAULT_DHCP_RELAY_PORT)
	r.server, r.conn, err = dhcp.NewDHCPServer2(r.srcaddr, DEFAULT_DHCP_RELAY_PORT, false)
	if err != nil {
		log.Errorln(err)
		return err
	}
	r.Start()
	return nil
}

func (r *SDHCPRelay) ServeDHCP(pkt dhcp.Packet, addr *net.UDPAddr, intf *net.Interface) (dhcp.Packet, error) {
	log.Infof("DHCP Relay Reply TO %s", pkt.CHAddr())
	v, ok := r.cache.Load(pkt.TransactionID())
	if ok {
		r.cache.Delete(pkt.TransactionID())
		val := v.(*SRelayCache)
		udpAddr := &net.UDPAddr{
			IP:   pkt.CIAddr(),
			Port: val.srcPort,
		}
		if err := r.guestDHCPConn.SendDHCP(pkt, udpAddr, intf); err != nil {
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
	err := r.conn.SendDHCP(pkt, &net.UDPAddr{IP: r.destaddr, Port: r.destport}, intf)
	return nil, err
}
