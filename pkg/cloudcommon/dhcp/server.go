package dhcp

import (
	"fmt"
	"net"
	"runtime/debug"

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

func NewDHCPServer2(address string, port int) (*DHCPServer, *Conn, error) {
	conn, err := NewConn(fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		return nil, nil, err
	}
	return &DHCPServer{
		Address: address,
		Port:    port,
		conn:    conn,
	}, conn, nil
}

type DHCPHandler interface {
	ServeDHCP(pkt Packet, addr *net.UDPAddr, intf *net.Interface) (Packet, error)
}

func (s *DHCPServer) ListenAndServe(handler DHCPHandler) error {
	if s.conn == nil {
		dhcpAddr := fmt.Sprintf("%s:%d", s.Address, s.Port)
		dhcpConn, err := NewConn(dhcpAddr)
		if err != nil {
			return fmt.Errorf("Listen DHCP connection error: %v", err)
		}
		s.conn = dhcpConn
	}
	defer s.conn.Close()
	return s.serveDHCP(handler)
}

func (s *DHCPServer) serveDHCP(handler DHCPHandler) error {
	for {
		pkt, addr, intf, err := s.conn.RecvDHCP()
		if err != nil {
			return fmt.Errorf("Receiving DHCP packet: %s", err)
		}
		if intf == nil {
			return fmt.Errorf("Received DHCP packet with no interface information (this is a violation of dhcp4.Conn's contract)")
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("Serve panic error: %v", r)
					debug.PrintStack()
				}
			}()

			resp, err := handler.ServeDHCP(pkt, addr, intf)
			if err != nil {
				log.Warningf("[DHCP] handler serve error: %v", err)
				return
			}
			if resp == nil {
				// log.Warningf("[DHCP] hander response null packet")
				return
			}
			//log.Debugf("[DHCP] send response packet: %s to interface: %#v", resp.DebugString(), intf)
			if err = s.conn.SendDHCP(resp, addr, intf); err != nil {
				log.Errorf("[DHCP] failed to response packet for %s: %v", pkt.CHAddr(), err)
				return
			}
		}()
	}
}
