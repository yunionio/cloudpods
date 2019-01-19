package hostdhcp

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/dhcp"
)

const DEFAULT_DHCP_RELAY_PORT = 68

type recvFunc func(pkt *dhcp.Packet)

type SRelayCache struct {
	mac net.HardwareAddr
	// srcPort int
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

	cache map[string]*SRelayCache
}

func NewDHCPRelay(addrs []string) *SDHCPRelay {
	relay := new(SDHCPRelay)
	addr := addrs[0]
	port, err := strconv.Atoi(addrs[1])
	if err != nil {
		log.Errorln(err)
		return nil
	}
	relay.destaddr = net.ParseIP(addr)
	relay.destport = port

	relay.conn, err = dhcp.NewConn(
		fmt.Sprintf("%s:%d", DEFAULT_DHCP_LISTEN_ADDR, DEFAULT_DHCP_RELAY_PORT))
	relay.server = dhcp.NewDHCPServer2(relay.conn)
	return relay
}

func (r *SDHCPRelay) Start() {
	log.Infof("DHCPRelay starting ...")
	r.server.ListenAndServe(r)
}

func (r *SDHCPRelay) Setup(addr string) {
	r.srcaddr = addr
}

/*
   xid = pkt[BOOTP].xid
   if xid in self.relay_packets:
       (mac, sport, dport, tm) = self.relay_packets.pop(xid)
       src_ip = pkt[BOOTP].giaddr
       cli_ip = pkt[BOOTP].ciaddr
       print mac, src_ip, cli_ip, dport, sport
       rep = Ether(dst=mac)
       rep /= IP(src=src_ip, dst=cli_ip)
       rep /= UDP(sport=dport, dport=sport)
       rep /= pkt
       self.send_packet(rep)
*/

func (r *SDHCPRelay) ServeDHCP(pkt dhcp.Packet, intf *net.Interface) (dhcp.Packet, error) {
	if v, ok := r.cache[string(pkt.TransactionID())]; ok {
		delete(r.cache, string(pkt.TransactionID()))
		pkt.SetCHAddr(v.mac)
		// pkt.ClientAddr 不变
		log.Errorln("XXXXX:", pkt.CIAddr())
		pkt.SetGIAddr(nil)
	}
	return pkt, nil
}

/*
   xid = pkt[BOOTP].xid
   mac = pkt[Ether].src
   sport = pkt[UDP].sport   // must be 68
   dport = pkt[UDP].dport   // must be 67
   print 'do_relay', xid, mac, sport, dport
   self.relay_packets[xid] = (mac, sport, dport, time.time())
   msg = pkt.getlayer(BOOTP).copy()
   self.relay.relay(msg)
*/

func (r *SDHCPRelay) Relay(pkt dhcp.Packet, intf *net.Interface) (dhcp.Packet, error) {
	// clean cache first
	var now = time.Now().Add(time.Second * -30)
	for k, v := range r.cache {
		if v.timer.Before(now) {
			delete(r.cache, k)
		}
	}

	// cache pkt
	r.cache[string(pkt.TransactionID())] = &SRelayCache{
		mac:   pkt.CHAddr(),
		timer: time.Now(),
	}

	/*
	   case txRelayAddr:
	       addr := net.UDPAddr{
	           IP:   pkt.RelayAddr,
	           Port: dhcpClientPort,
	       }
	       return c.conn.Send(b, &addr, 0)
	*/
	// euqal sendto addr ??? GIAddr should self address
	pkt.SetGIAddr(r.destaddr)
	// transport to upstream
	err := r.conn.SendDHCP(pkt, &net.UDPAddr{IP: r.destaddr, Port: r.destport}, intf)
	return nil, err
}
