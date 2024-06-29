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

package dhcp

import (
	"context"
	"fmt"
	"net"
	"runtime/debug"

	"golang.org/x/net/bpf"

	"yunion.io/x/log"
)

type DHCPServer struct {
	Address string
	Port    int
	conn    *Conn
}

// net.ListenPacket
func NewDHCPServer(address string, port int) (*DHCPServer, error) {
	dhcpAddr := fmt.Sprintf("%s:%d", address, port)
	dhcpConn, err := NewConn(dhcpAddr, false)
	if err != nil {
		return nil, fmt.Errorf("New DHCP connection error: %v", err)
	}
	return &DHCPServer{
		Address: address,
		Port:    port,
		conn:    dhcpConn,
	}, nil
}

// udp socket
func NewDHCPServer3(address string, port int) (*DHCPServer, error) {
	dhcpConn, err := NewSocketConn(address, port)
	if err != nil {
		return nil, fmt.Errorf("New DHCP connection error: %v", err)
	}
	return &DHCPServer{
		Address: address,
		Port:    port,
		conn:    dhcpConn,
	}, nil
}

// raw socket
func NewDHCPServer2(iface string, dhcpServerPort, dhcpRelayPort uint16) (*DHCPServer, *Conn, error) {
	// ip and udp and port 67 and port 68
	bpf := []bpf.RawInstruction{
		{Op: 0x28, Jt: 0, Jf: 0, K: 0x0000000c},
		{Op: 0x15, Jt: 0, Jf: 13, K: 0x00000800},
		{Op: 0x30, Jt: 0, Jf: 0, K: 0x00000017},
		{Op: 0x15, Jt: 0, Jf: 11, K: 0x00000011},
		{Op: 0x28, Jt: 0, Jf: 0, K: 0x00000014},
		{Op: 0x45, Jt: 9, Jf: 0, K: 0x00001fff},
		{Op: 0xb1, Jt: 0, Jf: 0, K: 0x0000000e},
		{Op: 0x48, Jt: 0, Jf: 0, K: 0x0000000e},
		{Op: 0x15, Jt: 0, Jf: 2, K: uint32(dhcpServerPort)},
		{Op: 0x48, Jt: 0, Jf: 0, K: 0x00000010},
		{Op: 0x15, Jt: 3, Jf: 4, K: uint32(dhcpRelayPort)},
		{Op: 0x15, Jt: 0, Jf: 3, K: uint32(dhcpRelayPort)},
		{Op: 0x48, Jt: 0, Jf: 0, K: 0x00000010},
		{Op: 0x15, Jt: 0, Jf: 1, K: uint32(dhcpServerPort)},
		{Op: 0x6, Jt: 0, Jf: 0, K: 0x00040000},
		{Op: 0x6, Jt: 0, Jf: 0, K: 0x00000000},
	}
	conn, err := NewRawSocketConn(iface, bpf, dhcpServerPort)
	if err != nil {
		return nil, nil, err
	}
	return &DHCPServer{
		conn: conn,
	}, conn, nil
}

type DHCPHandler interface {
	ServeDHCP(ctx context.Context, pkt Packet, addr *net.UDPAddr, intf *net.Interface) (Packet, []string, error)
}

func (s *DHCPServer) ListenAndServe(handler DHCPHandler) error {
	defer s.conn.Close()
	return s.serveDHCP(handler)
}

func (s *DHCPServer) GetConn() *Conn {
	return s.conn
}

func (s *DHCPServer) serveDHCP(handler DHCPHandler) error {
	ctx := context.WithValue(context.Background(), "dhcp_server", s)
	for {
		pkt, addr, mac, intf, err := s.conn.RecvDHCP()
		if err != nil {
			log.Errorf("Receiving DHCP packet: %s", err)
			continue
		}
		// if intf == nil {
		// 	return fmt.Errorf("Received DHCP packet with no interface information (this is a violation of dhcp4.Conn's contract)")
		// }

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("Serve panic error: %v", r)
					debug.PrintStack()
				}
			}()

			resp, targets, err := handler.ServeDHCP(ctx, pkt, addr, intf)
			if err != nil {
				log.Warningf("[DHCP] handler serve error: %v", err)
				return
			}
			if resp == nil {
				// log.Warningf("[DHCP] hander response null packet")
				return
			}
			if len(targets) > 0 {
				targetUdpAddr := *addr
				for _, target := range targets {
					log.Debugf("[DHCP] Send packet back to %s", target)
					targetUdpAddr.IP = net.ParseIP(target)
					resp.SetGIAddr(targetUdpAddr.IP)
					if err = s.conn.SendDHCP(resp, &targetUdpAddr, mac, intf); err != nil {
						log.Errorf("[DHCP] failed to response packet for %s: %v", pkt.CHAddr(), err)
					}
				}
				return
			}
			//log.Debugf("[DHCP] send response packet: %s to interface: %#v", resp.DebugString(), intf)
			if err = s.conn.SendDHCP(resp, addr, mac, intf); err != nil {
				log.Errorf("[DHCP] failed to response packet for %s: %v", pkt.CHAddr(), err)
				return
			}
		}()
	}
}
