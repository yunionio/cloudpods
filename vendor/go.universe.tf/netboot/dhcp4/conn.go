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

package dhcp4

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"golang.org/x/net/ipv4"
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
	Recv([]byte) (b []byte, addr *net.UDPAddr, ifidx int, err error)
	Send(b []byte, addr *net.UDPAddr, ifidx int) error
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

// NewConn creates a Conn bound to the given UDP ip:port.
func NewConn(addr string) (*Conn, error) {
	return newConn(addr, newPortableConn)
}

func newConn(addr string, n func(int) (conn, error)) (*Conn, error) {
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

	c, err := n(udpAddr.Port)
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
func (c *Conn) RecvDHCP() (*Packet, *net.Interface, error) {
	var buf [1500]byte
	for {
		b, _, ifidx, err := c.conn.Recv(buf[:])
		if err != nil {
			return nil, nil, err
		}
		if c.ifIndex != 0 && ifidx != c.ifIndex {
			continue
		}
		pkt, err := Unmarshal(b)
		if err != nil {
			continue
		}
		intf, err := net.InterfaceByIndex(ifidx)
		if err != nil {
			return nil, nil, err
		}

		// TODO: possibly more validation that the source lines up
		// with what the packet says.
		return pkt, intf, nil
	}
}

// SendDHCP sends pkt. The precise transmission mechanism depends
// on pkt.txType(). intf should be the net.Interface returned by
// RecvDHCP if responding to a DHCP client, or the interface for
// which configuration is desired if acting as a client.
func (c *Conn) SendDHCP(pkt *Packet, intf *net.Interface) error {
	b, err := pkt.Marshal()
	if err != nil {
		return err
	}

	switch pkt.txType() {
	case txBroadcast, txHardwareAddr:
		addr := net.UDPAddr{
			IP:   net.IPv4bcast,
			Port: dhcpClientPort,
		}
		return c.conn.Send(b, &addr, intf.Index)
	case txRelayAddr:
		addr := net.UDPAddr{
			IP:   pkt.RelayAddr,
			Port: 67,
		}
		return c.conn.Send(b, &addr, 0)
	case txClientAddr:
		addr := net.UDPAddr{
			IP:   pkt.ClientAddr,
			Port: dhcpClientPort,
		}
		return c.conn.Send(b, &addr, 0)
	default:
		return errors.New("unknown TX type for packet")
	}
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

func newPortableConn(port int) (conn, error) {
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

func (c *portableConn) Recv(b []byte) (rb []byte, addr *net.UDPAddr, ifidx int, err error) {
	n, cm, a, err := c.conn.ReadFrom(b)
	if err != nil {
		return nil, nil, 0, err
	}
	return b[:n], a.(*net.UDPAddr), cm.IfIndex, nil
}

func (c *portableConn) Send(b []byte, addr *net.UDPAddr, ifidx int) error {
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
