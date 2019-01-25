package hostdhcp

import (
	"net"
	"strconv"
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/dhcp"
)

const DEFAULT_DHCP_RELAY_PORT = 168

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

	srcaddr string

	destaddr net.IP
	destport int

	cache sync.Map
	// cache map[string]*SRelayCache
}

func NewDHCPRelay(addrs []string) (*SDHCPRelay, error) {
	relay := new(SDHCPRelay)
	addr := addrs[0]
	port, err := strconv.Atoi(addrs[1])
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	relay.destaddr = net.ParseIP(addr)
	relay.destport = port
	relay.cache = sync.Map{}
	// relay.cache = make(map[string]*SRelayCache, 0)

	log.Infof("DHCP Relay Bind addr %s port %d",
		DEFAULT_DHCP_LISTEN_ADDR, DEFAULT_DHCP_RELAY_PORT)
	relay.server, relay.conn, err = dhcp.NewDHCPServer2(DEFAULT_DHCP_LISTEN_ADDR, DEFAULT_DHCP_RELAY_PORT)
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
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

func (r *SDHCPRelay) Setup(addr string) {
	r.srcaddr = addr
}

func (r *SDHCPRelay) ServeDHCP(pkt dhcp.Packet, addr *net.UDPAddr, intf *net.Interface) (dhcp.Packet, error) {
	log.Infof("Receive DHCP Relay Reply TO %s", pkt.CHAddr())
	if v, ok := r.cache.Load(pkt.TransactionID()); ok {
		r.cache.Delete(pkt.TransactionID())
		val := v.(*SRelayCache)
		udpAddr := &net.UDPAddr{
			IP:   pkt.CIAddr(),
			Port: val.srcPort,
		}

		r.conn.SendDHCP(pkt, udpAddr, intf)
	}
	return nil, nil
}

func (r *SDHCPRelay) Relay(pkt dhcp.Packet, addr *net.UDPAddr, intf *net.Interface) (dhcp.Packet, error) {
	log.Infof("Receive DHCP Relay Rquest FROM %s", pkt.CHAddr())

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

	pkt.SetGIAddr(r.destaddr)
	err := r.conn.SendDHCP(pkt, &net.UDPAddr{IP: r.destaddr, Port: r.destport}, intf)
	return nil, err
}
