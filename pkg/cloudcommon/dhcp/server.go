package dhcp

import (
	"fmt"
	"net"

	"yunion.io/x/log"
)

type DHCPServer struct {
	Address string
	Port    int
	conn    *Conn
}

func NewDHCPServer(address string, port int) *DHCPServer {
	return &DHCPServer{
		Address: address,
		Port:    port,
	}
}

func NewDHCPServer2(conn *Conn) *DHCPServer {
	return &DHCPServer{
		conn: conn,
	}
}

type DHCPHandler interface {
	ServeDHCP(pkt *Packet, intf *net.Interface) (*Packet, error)
}

func (s *DHCPServer) ListenAndServe(handler DHCPHandler) error {
	dhcpAddr := fmt.Sprintf("%s:%d", s.Address, s.Port)
	dhcpConn, err := NewConn(dhcpAddr)
	if err != nil {
		return fmt.Errorf("Listen DHCP connection error: %v", err)
	}
	s.conn = dhcpConn
	defer s.conn.Close()
	return s.serveDHCP(handler)
}

func (s *DHCPServer) serveDHCP(handler DHCPHandler) error {
	for {
		pkt, intf, err := s.conn.RecvDHCP()
		if err != nil {
			return fmt.Errorf("Receiving DHCP packet: %s", err)
		}
		if intf == nil {
			return fmt.Errorf("Received DHCP packet with no interface information (this is a violation of dhcp4.Conn's contract)")
		}

		go func() {
			resp, err := handler.ServeDHCP(pkt, intf)
			if err != nil {
				log.Warningf("[DHCP] handler serve error: %v", err)
				return
			}
			if resp == nil {
				log.Warningf("[DHCP] hander response null packet")
				return
			}
			log.Debugf("[DHCP] send response packet: %s to interface: %#v", resp.DebugString(), intf)
			if err = s.conn.SendDHCP(resp, intf); err != nil {
				log.Errorf("[DHCP] failed to response packet for %s: %v", pkt.HardwareAddr, err)
				return
			}
		}()
	}
}
