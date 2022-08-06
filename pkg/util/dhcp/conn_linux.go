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

// Copyright 2019 Yunion
// Copyright 2016 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux
// +build linux

package dhcp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/mdlayher/raw"
	"golang.org/x/net/bpf"
	"golang.org/x/net/ipv4"
	"golang.org/x/sys/unix"
)

type rawSocketConn struct {
	conn *raw.Conn

	iface          *net.Interface
	ip             net.IP
	dhcpServerPort uint16
}

func newRawSocketConn(iface string, filter []bpf.RawInstruction, dhcpServerPort uint16) (conn, error) {
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, fmt.Errorf("interface by name: %v", err)
	}

	ip, err := interfaceToIPv4Addr(ifi)
	if err != nil {
		return nil, err
	}

	// unix.ETH_P_ALL
	conn, err := raw.ListenPacket(ifi, unix.ETH_P_ALL, &raw.Config{
		NoCumulativeStats: true,
		Filter:            filter,
	})
	return &rawSocketConn{conn, ifi, ip, dhcpServerPort}, nil
}

func (s *rawSocketConn) Close() error {
	return s.conn.Close()
}

func (s *rawSocketConn) Recv(b []byte) ([]byte, *net.UDPAddr, net.HardwareAddr, int, error) {
	// read packet
	n, addr, err := s.conn.ReadFrom(b)
	if err != nil {
		return nil, nil, nil, 0, fmt.Errorf("Read from errror: %s", err)
	}
	b = b[:n]

	srcMac, err := net.ParseMAC(addr.String())
	if err != nil {
		return nil, nil, nil, 0, fmt.Errorf("Parse mac error: %s", err)
	}

	p := gopacket.NewPacket(b, layers.LayerTypeEthernet, gopacket.Default)
	if p.ErrorLayer() != nil {
		return nil, nil, nil, 0, fmt.Errorf("Failed to decode packet: %v", p.ErrorLayer().Error())
	}

	var srcIp net.IP
	ipLayer := p.Layer(layers.LayerTypeIPv4)
	if ipLayer != nil {
		ip4 := ipLayer.(*layers.IPv4)
		srcIp = ip4.SrcIP
	} else {
		return nil, nil, nil, 0, fmt.Errorf("Fetch ip layer failed")
	}

	var srcPort uint16
	udpLayer := p.Layer(layers.LayerTypeUDP)
	if udpLayer != nil {
		udpInfo := udpLayer.(*layers.UDP)
		srcPort = uint16(udpInfo.SrcPort)
	} else {
		return nil, nil, nil, 0, fmt.Errorf("Fetch upd layer failed")
	}

	dhcpLayer := p.Layer(layers.LayerTypeDHCPv4)
	if dhcpLayer != nil {
		dhcp4 := dhcpLayer.(*layers.DHCPv4)
		sbf := gopacket.NewSerializeBuffer()
		if err := dhcp4.SerializeTo(sbf, gopacket.SerializeOptions{}); err != nil {
			return nil, nil, nil, 0, fmt.Errorf("Serialize dhcp packet error %s", err)
		}
		return sbf.Bytes(), &net.UDPAddr{IP: srcIp, Port: int(srcPort)}, srcMac, 0, nil
	} else {
		return nil, nil, nil, 0, fmt.Errorf("Fetch dhcp layer failed")
	}
}

func (s *rawSocketConn) Send(b []byte, addr *net.UDPAddr, destMac net.HardwareAddr, ifidx int) error {
	var dhcp = new(layers.DHCPv4)
	if err := dhcp.DecodeFromBytes(b, gopacket.NilDecodeFeedback); err != nil {
		return fmt.Errorf("Decode dhcp bytes error %s", err)
	}

	var eth = &layers.Ethernet{
		EthernetType: layers.EthernetTypeIPv4,
		SrcMAC:       s.iface.HardwareAddr,
		DstMAC:       destMac,
	}

	var ip = &layers.IPv4{
		Version:  4,
		TTL:      64,
		SrcIP:    s.ip,
		DstIP:    addr.IP,
		Protocol: layers.IPProtocolUDP,
	}

	var (
		srcPort = layers.UDPPort(s.dhcpServerPort)
		dstPort = layers.UDPPort(addr.Port)
	)

	var udp = &layers.UDP{
		SrcPort: srcPort,
		DstPort: dstPort,
	}
	udp.SetNetworkLayerForChecksum(ip)

	var (
		buf  = gopacket.NewSerializeBuffer()
		opts = gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}
	)
	if err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, dhcp); err != nil {
		return fmt.Errorf("SerializeLayers error: %s", err)
	}

	// s.conn.SetWriteDeadline(time.Now().Add(DefaultWriteTimeout)) // 2 second
	if _, err := s.conn.WriteTo(buf.Bytes(), &raw.Addr{HardwareAddr: destMac}); err != nil {
		return fmt.Errorf("Send dhcp packet error %s", err)
	}
	return nil
}

func (s *rawSocketConn) SetReadDeadline(t time.Time) error {
	return s.conn.SetReadDeadline(t)
}

func (s *rawSocketConn) SetWriteDeadline(t time.Time) error {
	return s.conn.SetWriteDeadline(t)
}

type linuxConn struct {
	port uint16
	conn *ipv4.RawConn
}

// NewSnooperConn creates a Conn that listens on the given UDP ip:port.
//
// Unlike NewConn, NewSnooperConn does not bind to the ip:port,
// enabling the Conn to coexist with other services on the machine.
func NewSnooperConn(addr string) (*Conn, error) {
	return newConn(addr, false, newLinuxConn)
}

func newLinuxConn(_ net.IP, port int, disableBroadcast bool) (conn, error) {
	if port == 0 {
		return nil, errors.New("must specify a listen port")
	}

	filter, err := bpf.Assemble([]bpf.Instruction{
		// Load IPv4 packet length
		bpf.LoadMemShift{Off: 0},
		// Get UDP dport
		bpf.LoadIndirect{Off: 2, Size: 2},
		// Correct dport?
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: uint32(port), SkipFalse: 1},
		// Accept
		bpf.RetConstant{Val: 1500},
		// Ignore
		bpf.RetConstant{Val: 0},
	})
	if err != nil {
		return nil, err
	}

	c, err := net.ListenPacket("ip4:17", "0.0.0.0")
	if err != nil {
		return nil, err
	}
	r, err := ipv4.NewRawConn(c)
	if err != nil {
		c.Close()
		return nil, err
	}
	if err = r.SetControlMessage(ipv4.FlagInterface, true); err != nil {
		c.Close()
		return nil, fmt.Errorf("setting packet filter: %s", err)
	}
	if err = r.SetBPF(filter); err != nil {
		c.Close()
		return nil, fmt.Errorf("setting packet filter: %s", err)
	}

	ret := &linuxConn{
		port: uint16(port),
		conn: r,
	}
	return ret, nil
}

func (c *linuxConn) Close() error {
	return c.conn.Close()
}

func (c *linuxConn) Recv(b []byte) (rb []byte, addr *net.UDPAddr, mac net.HardwareAddr, ifidx int, err error) {
	hdr, p, cm, err := c.conn.ReadFrom(b)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	if len(p) < 8 {
		return nil, nil, nil, 0, errors.New("not a UDP packet, too short")
	}
	sport := int(binary.BigEndian.Uint16(p[:2]))
	return p[8:], &net.UDPAddr{IP: hdr.Src, Port: sport}, nil, cm.IfIndex, nil
}

func (c *linuxConn) Send(b []byte, addr *net.UDPAddr, _ net.HardwareAddr, ifidx int) error {
	raw := make([]byte, 8+len(b))
	// src port
	binary.BigEndian.PutUint16(raw[:2], c.port)
	// dst port
	binary.BigEndian.PutUint16(raw[2:4], uint16(addr.Port))
	// length
	binary.BigEndian.PutUint16(raw[4:6], uint16(8+len(b)))
	copy(raw[8:], b)

	hdr := ipv4.Header{
		Version:  4,
		Len:      ipv4.HeaderLen,
		TOS:      0xc0, // DSCP CS6 (Network Control)
		TotalLen: ipv4.HeaderLen + 8 + len(b),
		TTL:      64,
		Protocol: 17,
		Dst:      addr.IP,
	}

	if ifidx > 0 {
		cm := ipv4.ControlMessage{
			IfIndex: ifidx,
		}
		return c.conn.WriteTo(&hdr, raw, &cm)
	}
	return c.conn.WriteTo(&hdr, raw, nil)
}

func (c *linuxConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *linuxConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
