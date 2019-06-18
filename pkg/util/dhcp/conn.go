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

package dhcp

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/mdlayher/raw"
	"golang.org/x/net/bpf"
	"golang.org/x/net/ipv4"
	"golang.org/x/sys/unix"
)

// defined as a var so tests can override it.
var (
	dhcpClientPort = 68
)

// txType describes how a Packet should be sent on the wire.
type txType int

// The various transmission strategies described in RFC 2131. "MUST",
// "MUST NOT", "SHOULD" and "MAY" are as specified in RFC 2119.
const (
	// Packet MUST be broadcast.
	txBroadcast txType = iota
	// Packet MUST be unicasted to port 67 of RelayAddr
	txRelayAddr
	// Packet MUST be unicasted to port 68 of ClientAddr
	txClientAddr
	// Packet SHOULD be unicasted to port 68 of YourAddr, with the
	// link-layer destination explicitly set to HardwareAddr. You MUST
	// NOT rely on ARP resolution to discover the link-layer
	// destination address.
	//
	// Conn implementations that cannot explicitly set the link-layer
	// destination address MAY instead broadcast the packet.
	txHardwareAddr
)

type conn interface {
	io.Closer
	Send(b []byte, addr *net.UDPAddr, destMac net.HardwareAddr, ifidx int) error
	Recv(b []byte) ([]byte, *net.UDPAddr, net.HardwareAddr, int, error)
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

// Conn is a DHCP-oriented packet socket.
//
// Multiple goroutines may invoke methods on a Conn simultaneously.
type Conn struct {
	conn    conn
	ifIndex int
}

func NewSocketConn(iface string, filter []bpf.RawInstruction) (*Conn, error) {
	conn, err := newSocketConn(iface, filter)
	if err != nil {
		return nil, err
	}
	return &Conn{conn, 0}, nil
}

// NewConn creates a Conn bound to the given UDP ip:port.
func NewConn(addr string, disableBroadcast bool) (*Conn, error) {
	return newConn(addr, disableBroadcast, newPortableConn)
}

func newConn(addr string, disableBroadcast bool, n func(net.IP, int, bool) (conn, error)) (*Conn, error) {
	if addr == "" {
		addr = "0.0.0.0:67"
	}

	ifIndex := 0
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return nil, err
	}
	if !udpAddr.IP.To4().Equal(net.IPv4zero) {
		// Caller wants to listen only on one address. However, DHCP
		// packets are frequently broadcast, so we can't just listen
		// on the given address. Instead, we need to translate it to
		// an interface, and then filter incoming packets based on
		// their received interface.
		ifIndex, err = ipToIfindex(udpAddr.IP)
		if err != nil {
			return nil, err
		}
	}

	c, err := n(udpAddr.IP, udpAddr.Port, disableBroadcast)
	if err != nil {
		return nil, err
	}
	return &Conn{
		conn:    c,
		ifIndex: ifIndex,
	}, nil
}

func ipToIfindex(ip net.IP) (int, error) {
	intfs, err := net.Interfaces()
	if err != nil {
		return 0, err
	}
	for _, intf := range intfs {
		addrs, err := intf.Addrs()
		if err != nil {
			return 0, err
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.Equal(ip) {
					return intf.Index, nil
				}
			}
		}
	}
	return 0, fmt.Errorf("IP %s not found on any local interface", ip)
}

// Close closes the DHCP socket.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// RecvDHCP reads a Packet from the connection. It returns the
// packet and the interface it was received on.
func (c *Conn) RecvDHCP() (Packet, *net.UDPAddr, net.HardwareAddr, *net.Interface, error) {
	var buf [1500]byte
	b, addr, mac, _, err := c.conn.Recv(buf[:])
	if err != nil {
		return nil, nil, nil, nil, err
	}
	/*if c.ifIndex != 0 && ifidx != c.ifIndex {
		log.Errorf("======= ifIndex continue, c.ifIndex: %d, ifidx: %d", c.ifIndex, ifidx)
		continue
	}*/
	pkt := Unmarshal(b)
	// intf, err := net.InterfaceByIndex(ifidx)
	// if err != nil {
	// 	return nil, nil, nil, err
	// }

	// TODO: possibly more validation that the source lines up
	// with what the packet says.
	return pkt, addr, mac, nil, nil
}

// SendDHCP sends pkt. The precise transmission mechanism depends
// on pkt.txType(). intf should be the net.Interface returned by
// RecvDHCP if responding to a DHCP client, or the interface for
// which configuration is desired if acting as a client.
func (c *Conn) SendDHCP(pkt Packet, addr *net.UDPAddr, mac net.HardwareAddr, intf *net.Interface) error {
	b := pkt.Marshal()

	ipStr, portStr, err := net.SplitHostPort(addr.String())
	if err != nil {
		return err
	}

	if net.ParseIP(ipStr).Equal(net.IPv4zero) || pkt.txType() == txBroadcast {
		port, _ := strconv.Atoi(portStr)
		addr = &net.UDPAddr{IP: net.IPv4bcast, Port: port}
	}
	return c.conn.Send(b, addr, mac, 0)

	/*
		switch pkt.txType() {
		case txBroadcast, txHardwareAddr:
			addr := net.UDPAddr{
				IP:   net.IPv4bcast,
				Port: dhcpClientPort,
			}
			return c.conn.Send(b, &addr, intf.Index)
		case txRelayAddr:
			addr := net.UDPAddr{
				IP:   pkt.RelayAddr(),
				Port: dhcpClientPort,
			}
			log.Errorf("===============relay type pkt, addr: %#v", addr)
			return c.conn.Send(b, &addr, 0)
		case txClientAddr:
			addr := net.UDPAddr{
				IP:   pkt.CIAddr(),
				Port: dhcpClientPort,
			}
			return c.conn.Send(b, &addr, 0)
		default:
			return errors.New("unknown TX type for packet")
		}*/
}

// SetReadDeadline sets the deadline for future Read calls.  If the
// deadline is reached, Read will fail with a timeout (see net.Error)
// instead of blocking.  A zero value for t means Read will not time
// out.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls.  If the
// deadline is reached, Write will fail with a timeout (see net.Error)
// instead of blocking.  A zero value for t means Write will not time
// out.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

type portableConn struct {
	conn *ipv4.PacketConn
}

func newPortableConn(_ net.IP, port int, _ bool) (conn, error) {
	c, err := net.ListenPacket("udp4", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	l := ipv4.NewPacketConn(c)
	if err = l.SetControlMessage(ipv4.FlagInterface, true); err != nil {
		l.Close()
		return nil, err
	}
	return &portableConn{l}, nil
}

func (c *portableConn) Close() error {
	return c.conn.Close()
}

func (c *portableConn) Recv(b []byte) (rb []byte, addr *net.UDPAddr, mac net.HardwareAddr, ifidx int, err error) {
	n, cm, a, err := c.conn.ReadFrom(b)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	return b[:n], a.(*net.UDPAddr), nil, cm.IfIndex, nil
}

func (c *portableConn) Send(b []byte, addr *net.UDPAddr, _ net.HardwareAddr, ifidx int) error {
	if ifidx <= 0 {
		_, err := c.conn.WriteTo(b, nil, addr)
		return err
	}
	cm := ipv4.ControlMessage{
		IfIndex: ifidx,
	}
	_, err := c.conn.WriteTo(b, &cm, addr)
	return err
}

func (c *portableConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *portableConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

type socketConn struct {
	conn *raw.Conn

	iface *net.Interface
	ip    net.IP
}

func interfaceToIPv4Addr(ifi *net.Interface) (net.IP, error) {
	if ifi == nil {
		return net.IPv4zero, nil
	}
	ifat, err := ifi.Addrs()
	if err != nil {
		return nil, err
	}
	for _, ifa := range ifat {
		switch v := ifa.(type) {
		case *net.IPAddr:
			if v.IP.To4() != nil {
				return v.IP, nil
			}
		case *net.IPNet:
			if v.IP.To4() != nil {
				return v.IP, nil
			}
		}
	}
	return nil, errors.New("no such network interface")
}

func newSocketConn(iface string, filter []bpf.RawInstruction) (conn, error) {
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
	return &socketConn{conn, ifi, ip}, nil
}

func (s *socketConn) Close() error {
	return s.conn.Close()
}

func (s *socketConn) Recv(b []byte) ([]byte, *net.UDPAddr, net.HardwareAddr, int, error) {
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
		return sbf.Bytes(), &net.UDPAddr{srcIp, int(srcPort), ""}, srcMac, 0, nil
	} else {
		return nil, nil, nil, 0, fmt.Errorf("Fetch dhcp layer failed")
	}
}

func (s *socketConn) Send(b []byte, addr *net.UDPAddr, destMac net.HardwareAddr, ifidx int) error {
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
		srcPort layers.UDPPort
		dstPort = layers.UDPPort(addr.Port)
	)
	if dstPort == 67 {
		srcPort = 68
	} else {
		srcPort = 67
	}
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

func (s *socketConn) SetReadDeadline(t time.Time) error {
	return s.conn.SetReadDeadline(t)
}

func (s *socketConn) SetWriteDeadline(t time.Time) error {
	return s.conn.SetWriteDeadline(t)
}
