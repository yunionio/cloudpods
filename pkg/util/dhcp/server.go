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
	"fmt"
	"net"
	"runtime/debug"

	"golang.org/x/net/bpf"
	"yunion.io/x/log"
)

const (
	PORT_67        = 67
	PORT_67_AND_68 = 67 + 68
)

var UDP_BPF_PORT_MAP = map[uint16][]bpf.RawInstruction{
	// ip and udp and dst port 67
	PORT_67: []bpf.RawInstruction{
		{0x28, 0, 0, 0x0000000c},
		{0x15, 0, 8, 0x00000800},
		{0x30, 0, 0, 0x00000017},
		{0x15, 0, 6, 0x00000011},
		{0x28, 0, 0, 0x00000014},
		{0x45, 4, 0, 0x00001fff},
		{0xb1, 0, 0, 0x0000000e},
		{0x48, 0, 0, 0x00000010},
		{0x15, 0, 1, 0x00000043},
		{0x6, 0, 0, 0x00040000},
		{0x6, 0, 0, 0x00000000},
	},
	// ip and udp and dst port 67 and port 68
	PORT_67_AND_68: []bpf.RawInstruction{
		{0x28, 0, 0, 0x0000000c},
		{0x15, 0, 10, 0x00000800},
		{0x30, 0, 0, 0x00000017},
		{0x15, 0, 8, 0x00000011},
		{0x28, 0, 0, 0x00000014},
		{0x45, 6, 0, 0x00001fff},
		{0xb1, 0, 0, 0x0000000e},
		{0x48, 0, 0, 0x00000010},
		{0x15, 0, 3, 0x00000043},
		{0x48, 0, 0, 0x0000000e},
		{0x15, 0, 1, 0x00000044},
		{0x6, 0, 0, 0x00040000},
		{0x6, 0, 0, 0x00000000},
	},
}

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

func NewDHCPServer2(iface string, portDesc uint16) (*DHCPServer, *Conn, error) {
	bpf, ok := UDP_BPF_PORT_MAP[portDesc]
	if !ok {
		return nil, nil, fmt.Errorf("BPF not found %d", portDesc)
	}
	conn, err := NewSocketConn(iface, bpf)
	if err != nil {
		return nil, nil, err
	}
	return &DHCPServer{
		conn: conn,
	}, conn, nil
}

type DHCPHandler interface {
	ServeDHCP(pkt Packet, addr *net.UDPAddr, intf *net.Interface) (Packet, error)
}

func (s *DHCPServer) ListenAndServe(handler DHCPHandler) error {
	if s.conn == nil {
		dhcpAddr := fmt.Sprintf("%s:%d", s.Address, s.Port)
		dhcpConn, err := NewConn(dhcpAddr, false)
		if err != nil {
			return fmt.Errorf("Listen DHCP connection error: %v", err)
		}
		s.conn = dhcpConn
	}
	defer s.conn.Close()
	return s.serveDHCP(handler)
}

func (s *DHCPServer) GetConn() *Conn {
	return s.conn
}

func (s *DHCPServer) serveDHCP(handler DHCPHandler) error {
	for {
		pkt, addr, mac, intf, err := s.conn.RecvDHCP()
		if err != nil {
			return fmt.Errorf("Receiving DHCP packet: %s", err)
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
			if err = s.conn.SendDHCP(resp, addr, mac, intf); err != nil {
				log.Errorf("[DHCP] failed to response packet for %s: %v", pkt.CHAddr(), err)
				return
			}
		}()
	}
}
