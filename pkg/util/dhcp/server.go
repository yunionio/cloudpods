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
	"net"
	"runtime/debug"

	"golang.org/x/net/bpf"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type DHCPServer struct {
	Address string
	Port    int
	conn    *Conn
}

type DHCP6Server struct {
	DHCPServer
}

// udp unicast socket
func NewDHCPServer3(address string, port int) (*DHCPServer, error) {
	dhcpConn, err := NewSocketConn(address, port)
	if err != nil {
		return nil, errors.Wrap(err, "New DHCP connection")
	}
	return &DHCPServer{
		Address: address,
		Port:    port,
		conn:    dhcpConn,
	}, nil
}

// udp unicast socket
func NewDHCP6Server3(address string, port int) (*DHCP6Server, error) {
	dhcpConn, err := NewSocketConn(address, port)
	if err != nil {
		return nil, errors.Wrap(err, "New DHCP6 connection")
	}
	return &DHCP6Server{
		DHCPServer{
			Address: address,
			Port:    port,
			conn:    dhcpConn,
		},
	}, nil
}

const (
	bpfContinue     uint8 = 0
	bpfProcDhcpResp uint8 = 11

	bpfProcDhcpResp6 uint8 = 8
	bpfProcICMPv6    uint8 = 11
	bpfProcReadPkt6  uint8 = 14
	bpfProcEnd6      uint8 = 15

	bpfProcReadPkt4 uint8 = 14
	bpfProcEnd4     uint8 = 15

	udpProtocol  uint32 = 17
	icmpProtocol uint32 = 58
)

var (
	v4bpf = []bpf.Instruction{
		/* 0 load ethernet type, 2bytes */
		&bpf.LoadAbsolute{Off: 12, Size: 2},

		/* 1 if ether_type != 0x0800(IPv4) then jump to [end] else continue */
		&bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: 0x800, SkipTrue: bpfGoto(2, bpfProcEnd4), SkipFalse: bpfContinue},
		/* 2 load ipv4 protocol, offset 23, 1 byte */
		&bpf.LoadAbsolute{Off: 23, Size: 1},
		/* 3 if ip_proto != UDP then jump to [end] else continue */
		&bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: udpProtocol, SkipTrue: bpfGoto(4, bpfProcEnd4), SkipFalse: bpfContinue},
		/* 4 if load ip flags & fragment_offset */
		&bpf.LoadAbsolute{Off: 20, Size: 2},
		/* 5 if ip_fragment_offset != 0 then jump to end/25(6+19) else continue */
		bpf.JumpIf{Cond: bpf.JumpBitsSet, Val: 0x1fff, SkipTrue: bpfGoto(6, bpfProcEnd4), SkipFalse: bpfContinue},
		/* 6 store ip_header_length*4 in register X */
		bpf.LoadMemShift{Off: 14},
		/* 7 load 14 + ip_header_len + 0 (UDP src port), 2 bytes */
		bpf.LoadIndirect{Off: 14, Size: 2},
		/* 8 if udp_src_port != 67 then jump to dhcp_resp/11(9+2) else continue */
		bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: 67, SkipTrue: bpfGoto(9, bpfProcDhcpResp), SkipFalse: bpfContinue},
		/* 9 load 14 + ip_header_len + 2 (udp dst port), 2 bytes */
		bpf.LoadIndirect{Off: 16, Size: 2},
		/* 10 if udp_dst_port == 68 then jump to read/24(11+13) else jump to end/25(11+14) */
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: 68, SkipTrue: bpfGoto(11, bpfProcReadPkt4), SkipFalse: bpfGoto(11, bpfProcEnd4)},
		/* 11 [dhcp_resp] if udp_src_port != 68 then jump to end/25(12+13) else continue, which means a UDP response */
		bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: 68, SkipTrue: bpfGoto(12, bpfProcEnd4), SkipFalse: bpfContinue},
		/* 12 load 14 + ip_header_len + 2 (udp dst port), 2 bytes */
		bpf.LoadIndirect{Off: 16, Size: 2},
		/* 13 if udp_dst_port != 67 then jump to end/25(14+11) else continue */
		bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: 67, SkipTrue: bpfGoto(14, bpfProcEnd4), SkipFalse: bpfContinue},

		/* 14 [read] return the whole packet */
		bpf.RetConstant{Val: 0x40000},
		/* 15 [end] return none */
		bpf.RetConstant{Val: 0x0},
	}

	v6bpf = []bpf.Instruction{
		/* 0 load ethernet type, 2bytes */
		&bpf.LoadAbsolute{Off: 12, Size: 2},

		/* 1 [ipv6] if ether_type != 0x86dd(IPv6) then jump to end/25(15 + 10) else continue */
		&bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: 0x86dd, SkipTrue: bpfGoto(2, bpfProcEnd6), SkipFalse: bpfContinue},
		/* 2 load ipv6 next header, offset 20, 1 byte */
		&bpf.LoadAbsolute{Off: 20, Size: 1},
		/* 3 [udp6] if ip_proto != UDP then jump to [icmp6]/24(17 + 7) else continue */
		&bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: udpProtocol, SkipTrue: bpfGoto(4, bpfProcICMPv6), SkipFalse: bpfContinue},
		/* 4 load 14 + 40 + 0 (UDP src port), 2 bytes */
		bpf.LoadAbsolute{Off: 54, Size: 2},
		/* 5 if udp_src_port != 547 then jump to ipv6_resp/21(19+2) else continue */
		bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: 547, SkipTrue: bpfGoto(6, bpfProcDhcpResp), SkipFalse: bpfContinue},
		/* 6 load 14 + 40 + 2 (udp dst port), 2 bytes */
		bpf.LoadAbsolute{Off: 56, Size: 2},
		/* 7 if udp_dst_port == 546 then jump to read/24(21+3) else jump to end/25(21+4) */
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: 546, SkipTrue: bpfGoto(8, bpfProcReadPkt6), SkipFalse: bpfGoto(8, bpfProcEnd6)},
		/* 8 [dhcpv6_resp] if udp_src_port != 546 then jump to end/25(22+3) else continue, which means a UDP response */
		bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: 546, SkipTrue: bpfGoto(9, bpfProcEnd6), SkipFalse: bpfContinue},
		/* 9 load 14 + 40 + 2 (udp dst port), 2 bytes */
		bpf.LoadAbsolute{Off: 56, Size: 2},
		/* 10 if udp_dst_port != 547 then jump to end/25(24+1) else jump to read/ */
		bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: 547, SkipTrue: bpfGoto(11, bpfProcEnd6), SkipFalse: bpfGoto(11, bpfProcReadPkt6)},

		/* 11 [icmp6] if ip_proto != ICMP then jump to [end]/24(17 + 7) else continue */
		&bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: icmpProtocol, SkipTrue: bpfGoto(12, bpfProcEnd6), SkipFalse: bpfContinue},
		/* 12 [icmp6 type] load 14 + 40 + 0 (ICMPv6 type), 1 bytes */
		bpf.LoadAbsolute{Off: 54, Size: 1},
		/* 13 [Router Solicitation] if icmp6_type == 133 then jump to [read] else [end] */
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: 133, SkipTrue: bpfGoto(14, bpfProcReadPkt6), SkipFalse: bpfGoto(14, bpfProcEnd6)},

		/* 14 [read] return the whole packet */
		bpf.RetConstant{Val: 0x40000},
		/* 15 [end] return none */
		bpf.RetConstant{Val: 0x0},
	}
)

func bpfGoto(current uint8, jumpTo uint8) uint8 {
	return jumpTo - current
}

func getRawInstructions(obpf []bpf.Instruction) ([]bpf.RawInstruction, error) {
	bpf := make([]bpf.RawInstruction, 0, len(obpf))
	for i := range obpf {
		inst, err := obpf[i].Assemble()
		if err != nil {
			return nil, errors.Wrap(err, "invalid eBPF instruction")
		}
		bpf = append(bpf, inst)
	}
	return bpf, nil
}

// raw socket
func NewDHCPServer2(iface string, port uint16) (*DHCPServer, *Conn, error) {
	bpf, err := getRawInstructions(v4bpf)
	if err != nil {
		return nil, nil, errors.Wrap(err, "getRawInstructions")
	}
	conn, err := NewRawSocketConn(iface, bpf, port)
	if err != nil {
		return nil, nil, err
	}
	return &DHCPServer{
		conn: conn,
	}, conn, nil
}

func NewDHCP6Server2(iface string, port uint16) (*DHCP6Server, *Conn, error) {
	bpf, err := getRawInstructions(v6bpf)
	if err != nil {
		return nil, nil, errors.Wrap(err, "getRawInstructions")
	}
	conn, err := NewRawSocketConn6(iface, bpf, port)
	if err != nil {
		return nil, nil, err
	}
	return &DHCP6Server{
		DHCPServer{
			conn: conn,
		},
	}, conn, nil
}

func (s *DHCPServer) ListenAndServe(ctx context.Context, handler DHCPHandler) error {
	defer s.conn.Close()
	return s.serveDHCP(ctx, handler)
}

func (s *DHCP6Server) ListenAndServe(ctx context.Context, handler DHCP6Handler) error {
	defer s.conn.Close()
	return s.serveDHCP(ctx, handler)
}

func (s *DHCPServer) GetConn() *Conn {
	return s.conn
}

func (s *DHCPServer) serveDHCP(ctx context.Context, handler DHCPHandler) error {
	for {
		pkt, addr, mac, err := s.conn.RecvDHCP()
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

			resp, targets, err := handler.ServeDHCP(ctx, pkt, mac, addr)
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
					if err = s.conn.SendDHCP(resp, &targetUdpAddr, mac); err != nil {
						log.Errorf("[DHCP] failed to response packet for %s: %v", pkt.CHAddr(), err)
					}
				}
				return
			}
			//log.Debugf("[DHCP] send response packet: %s to interface: %#v", resp.DebugString(), intf)
			if err = s.conn.SendDHCP(resp, addr, mac); err != nil {
				log.Errorf("[DHCP] failed to response packet for %s: %v", pkt.CHAddr(), err)
				return
			}
		}()
	}
}

func (s *DHCP6Server) serveDHCP(ctx context.Context, handler DHCP6Handler) error {
	for {
		pkt, addr, mac, err := s.conn.RecvDHCP()
		if err != nil {
			log.Errorf("Receiving DHCP packet: %s", err)
			continue
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("Serve panic error: %v", r)
					debug.PrintStack()
				}
			}()

			if addr.Port == icmpRAFakePort {
				// receive a RA solication
				resp, err := handler.ServeRA(ctx, pkt, mac, addr)
				if err != nil {
					log.Warningf("[DHCP6] handler ServeRA error: %s", err)
					return
				}
				if resp == nil {
					log.Warningf("[DHCP6] hander ServeRA response null packet")
					return
				}

				//log.Debugf("[DHCP] send response packet: %s to interface: %#v", resp.DebugString(), intf)
				err = s.conn.SendDHCP(resp, addr, mac)
				if err != nil {
					log.Errorf("[DHCP] failed to response packet for %s: %v", pkt.CHAddr(), err)
				}
				return
			}
			resp, targets, err := handler.ServeDHCP(ctx, pkt, mac, addr)
			if err != nil {
				log.Warningf("[DHCP] handler serve error: %s", err)
				return
			}
			if resp == nil {
				// log.Warningf("[DHCP] hander response null packet")
				return
			}
			if len(targets) > 0 {
				targetUdpAddr := *addr
				for _, target := range targets {
					log.Debugf("[DHCP6] Send packet back to %s", target)
					targetUdpAddr.IP = net.ParseIP(target)
					resp.SetGIAddr(targetUdpAddr.IP)
					if err = s.conn.SendDHCP(resp, &targetUdpAddr, mac); err != nil {
						log.Errorf("[DHCP6] failed to response packet for %s: %v", pkt.CHAddr(), err)
					}
				}
				return
			}
			//log.Debugf("[DHCP] send response packet: %s to interface: %#v", resp.DebugString(), intf)
			if err = s.conn.SendDHCP(resp, addr, mac); err != nil {
				log.Errorf("[DHCP6] failed to response packet for %s: %v", pkt.CHAddr(), err)
				return
			}
		}()
	}
}
