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
	"fmt"
	"io"
	"net"
	"syscall"
	"time"

	"golang.org/x/net/bpf"

	"yunion.io/x/pkg/errors"
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
	Send(b []byte, addr *net.UDPAddr, destMac net.HardwareAddr) error
	Recv(b []byte) ([]byte, *net.UDPAddr, net.HardwareAddr, int, error)
	Send6(b []byte, addr *net.UDPAddr, destMac net.HardwareAddr) error
	Recv6(b []byte) ([]byte, *net.UDPAddr, net.HardwareAddr, int, error)
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

func NewRawSocketConn(iface string, filter []bpf.RawInstruction, serverPort uint16) (*Conn, error) {
	conn, err := newRawSocketConn(iface, filter, serverPort)
	if err != nil {
		return nil, err
	}
	return &Conn{conn, 0}, nil
}

func NewSocketConn(addr string, port int) (*Conn, error) {
	conn, err := newSocketConn(net.ParseIP(addr), port, false)
	if err != nil {
		return nil, err
	}
	return &Conn{conn, 0}, nil
}

// NewConn creates a Conn bound to the given UDP ip:port.
// func NewConn(addr string, disableBroadcast bool) (*Conn, error) {
//	return newConn(addr, disableBroadcast, newPortableConn)
// }

/*func newConn(addr string, disableBroadcast bool, n func(net.IP, int, bool) (conn, error)) (*Conn, error) {
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
}*/

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
func (c *Conn) RecvDHCP() (Packet, *net.UDPAddr, net.HardwareAddr, error) {
	var buf [1500]byte
	b, addr, mac, _, err := c.conn.Recv(buf[:])
	if err != nil {
		return nil, nil, nil, err
	}
	pkt := Unmarshal(b)
	return pkt, addr, mac, nil
}

// SendDHCP sends pkt. The precise transmission mechanism depends
// on pkt.txType(). intf should be the net.Interface returned by
// RecvDHCP if responding to a DHCP client, or the interface for
// which configuration is desired if acting as a client.
func (c *Conn) SendDHCP(pkt Packet, addr *net.UDPAddr, mac net.HardwareAddr) error {
	b := pkt.Marshal()

	switch len(addr.IP) {
	case net.IPv4len:
		if addr.IP.Equal(net.IPv4zero) || pkt.txType() == txBroadcast {
			addr = &net.UDPAddr{IP: net.IPv4bcast, Port: addr.Port}
		}
	case net.IPv6len:
	default:
		return errors.Wrapf(errors.ErrNotSupported, "unsupported length of IP address length %d", len(addr.IP))

	}
	return c.conn.Send(b, addr, mac)
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
	return nil, errors.Wrapf(errors.ErrNotFound, "no such network interface %s", ifi.Name)
}

type socketConn struct {
	sock int
}

func newSocketConn(addr net.IP, port int, disableBroadcast bool) (conn, error) {
	var broadcastOpt = 1
	if disableBroadcast {
		broadcastOpt = 0
	}
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return nil, err
	}
	err = syscall.SetsockoptInt(sock, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if err != nil {
		return nil, err
	}
	err = syscall.SetsockoptInt(sock, syscall.SOL_SOCKET, syscall.SO_BROADCAST, broadcastOpt)
	if err != nil {
		return nil, err
	}
	byteAddr := [4]byte{}
	copy(byteAddr[:], addr.To4()[:4])
	lsa := &syscall.SockaddrInet4{
		Port: port,
		Addr: byteAddr,
	}
	if err = syscall.Bind(sock, lsa); err != nil {
		return nil, err
	}
	if err = syscall.SetNonblock(sock, false); err != nil {
		return nil, err
	}

	// Its equal syscall.CloseOnExec
	// most file descriptors are getting set to close-on-exec
	// apart from syscall open, socket etc.
	syscall.Syscall(syscall.SYS_FCNTL, uintptr(sock), syscall.F_SETFD, syscall.FD_CLOEXEC)
	return &socketConn{sock}, nil
}

func (s *socketConn) Close() error {
	return syscall.Close(s.sock)
}

func (s *socketConn) Recv(b []byte) ([]byte, *net.UDPAddr, net.HardwareAddr, int, error) {
	n, a, err := syscall.Recvfrom(s.sock, b, 0)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	if addr, ok := a.(*syscall.SockaddrInet4); !ok {
		return nil, nil, nil, 0, errors.Wrap(errors.ErrUnsupportedProtocol, "Recvfrom recevice address is not famliy Inet4")
	} else {
		ip := net.IP{addr.Addr[0], addr.Addr[1], addr.Addr[2], addr.Addr[3]}
		udpAddr := &net.UDPAddr{
			IP:   ip,
			Port: addr.Port,
		}
		// there is no interface index info
		return b[:n], udpAddr, nil, 0, nil
	}
}

func (s *socketConn) Send(b []byte, addr *net.UDPAddr, destMac net.HardwareAddr) error {
	destIp := [4]byte{}
	copy(destIp[:], addr.IP.To4()[:4])
	destAddr := &syscall.SockaddrInet4{
		Addr: destIp,
		Port: addr.Port,
	}
	return syscall.Sendto(s.sock, b, 0, destAddr)
}

func (s *socketConn) SetReadDeadline(t time.Time) error {
	return errors.ErrNotImplemented
}

func (s *socketConn) SetWriteDeadline(t time.Time) error {
	return errors.ErrNotImplemented
}
