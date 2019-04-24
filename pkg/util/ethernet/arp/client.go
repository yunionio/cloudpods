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

// MIT License
// ===========

// Copyright (C) 2015 Matt Layher

// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package arp

import (
	"errors"
	"net"
	"time"

	"yunion.io/x/onecloud/pkg/util/ethernet"
)

var (
	// errNoIPv4Addr is returned when an interface does not have an IPv4
	// address.
	errNoIPv4Addr = errors.New("no IPv4 address available for interface")
)

// protocolARP is the uint16 EtherType representation of ARP (Address
// Resolution Protocol, RFC 826).
const protocolARP = 0x0806

// A Client is an ARP client, which can be used to send and receive
// ARP packets.
type Client struct {
	ifi *net.Interface
	ip  net.IP
	p   net.PacketConn
}

// Dial creates a new Client using the specified network interface.
// Dial retrieves the IPv4 address of the interface and binds a raw socket
// to send and receive ARP packets.
func Dial(ifi *net.Interface) (*Client, error) {
	// Open raw socket to send and receive ARP packets using ethernet frames
	// we build ourselves.
	p, err := ethernet.ListenPacket(ifi, protocolARP, nil)
	if err != nil {
		return nil, err
	}
	return New(ifi, p)
}

// New creates a new Client using the specified network interface
// and net.PacketConn. This allows the caller to define exactly how they bind to the
// net.PacketConn. This is most useful to define what protocol to pass to socket(7).
//
// In most cases, callers would be better off calling Dial.
func New(ifi *net.Interface, p net.PacketConn) (*Client, error) {
	// Check for usable IPv4 addresses for the Client
	addrs, err := ifi.Addrs()
	if err != nil {
		return nil, err
	}

	return newClient(ifi, p, addrs)
}

// newClient is the internal, generic implementation of newClient.  It is used
// to allow an arbitrary net.PacketConn to be used in a Client, so testing
// is easier to accomplish.
func newClient(ifi *net.Interface, p net.PacketConn, addrs []net.Addr) (*Client, error) {
	ip, err := firstIPv4Addr(addrs)
	if err != nil {
		return nil, err
	}

	return &Client{
		ifi: ifi,
		ip:  ip,
		p:   p,
	}, nil
}

// Close closes the Client's raw socket and stops sending and receiving
// ARP packets.
func (c *Client) Close() error {
	return c.p.Close()
}

// Request sends an ARP request, asking for the hardware address
// associated with an IPv4 address. The response, if any, can be read
// with the Read method.
//
// Unlike Resolve, which provides an easier interface for getting the
// hardware address, Request allows sending many requests in a row,
// retrieving the responses afterwards.
func (c *Client) Request(ip net.IP) error {
	if c.ip == nil {
		return errNoIPv4Addr
	}

	// Create ARP packet for broadcast address to attempt to find the
	// hardware address of the input IP address
	arp, err := NewPacket(OperationRequest, c.ifi.HardwareAddr, c.ip, ethernet.Broadcast, ip)
	if err != nil {
		return err
	}
	return c.WriteTo(arp, ethernet.Broadcast)
}

// Resolve performs an ARP request, attempting to retrieve the
// hardware address of a machine using its IPv4 address. Resolve must not
// be used concurrently with Read. If you're using Read (usually in a
// loop), you need to use Request instead. Resolve may read more than
// one message if it receives messages unrelated to the request.
func (c *Client) Resolve(ip net.IP) (net.HardwareAddr, error) {
	err := c.Request(ip)
	if err != nil {
		return nil, err
	}

	// Loop and wait for replies
	for {
		arp, _, err := c.Read()
		if err != nil {
			return nil, err
		}

		if arp.Operation != OperationReply || !arp.SenderIP.Equal(ip) {
			continue
		}

		return arp.SenderHardwareAddr, nil
	}
}

// Read reads a single ARP packet and returns it, together with its
// ethernet frame.
func (c *Client) Read() (*Packet, *ethernet.Frame, error) {
	buf := make([]byte, 128)
	for {
		n, _, err := c.p.ReadFrom(buf)
		if err != nil {
			return nil, nil, err
		}

		p, eth, err := parsePacket(buf[:n])
		if err != nil {
			if err == errInvalidARPPacket {
				continue
			}
			return nil, nil, err
		}
		return p, eth, nil
	}
}

// WriteTo writes a single ARP packet to addr. Note that addr should,
// but doesn't have to, match the target hardware address of the ARP
// packet.
func (c *Client) WriteTo(p *Packet, addr net.HardwareAddr) error {
	pb, err := p.MarshalBinary()
	if err != nil {
		return err
	}

	f := &ethernet.Frame{
		Destination: p.TargetHardwareAddr,
		Source:      p.SenderHardwareAddr,
		EtherType:   ethernet.EtherTypeARP,
		Payload:     pb,
	}

	fb, err := f.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = c.p.WriteTo(fb, &ethernet.Addr{HardwareAddr: addr})
	return err
}

// Reply constructs and sends a reply to an ARP request. On the ARP
// layer, it will be addressed to the sender address of the packet. On
// the ethernet layer, it will be sent to the actual remote address
// from which the request was received.
//
// For more fine-grained control, use WriteTo to write a custom
// response.
func (c *Client) Reply(req *Packet, hwAddr net.HardwareAddr, ip net.IP) error {
	p, err := NewPacket(OperationReply, hwAddr, ip, req.SenderHardwareAddr, req.SenderIP)
	if err != nil {
		return err
	}
	return c.WriteTo(p, req.SenderHardwareAddr)
}

// Copyright (c) 2012 The Go Authors. All rights reserved.
// Source code in this file is based on src/net/interface_linux.go,
// from the Go standard library.  The Go license can be found here:
// https://golang.org/LICENSE.

// Documentation taken from net.PacketConn interface.  Thanks:
// http://golang.org/pkg/net/#PacketConn.

// SetDeadline sets the read and write deadlines associated with the
// connection.
func (c *Client) SetDeadline(t time.Time) error {
	return c.p.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future raw socket read calls.
// If the deadline is reached, a raw socket read will fail with a timeout
// (see type net.Error) instead of blocking.
// A zero value for t means a raw socket read will not time out.
func (c *Client) SetReadDeadline(t time.Time) error {
	return c.p.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future raw socket write calls.
// If the deadline is reached, a raw socket write will fail with a timeout
// (see type net.Error) instead of blocking.
// A zero value for t means a raw socket write will not time out.
// Even if a write times out, it may return n > 0, indicating that
// some of the data was successfully written.
func (c *Client) SetWriteDeadline(t time.Time) error {
	return c.p.SetWriteDeadline(t)
}

// HardwareAddr fetches the hardware address for the interface associated
// with the connection.
func (c Client) HardwareAddr() net.HardwareAddr {
	return c.ifi.HardwareAddr
}

// firstIPv4Addr attempts to retrieve the first detected IPv4 address from an
// input slice of network addresses.
func firstIPv4Addr(addrs []net.Addr) (net.IP, error) {
	for _, a := range addrs {
		if a.Network() != "ip+net" {
			continue
		}

		ip, _, err := net.ParseCIDR(a.String())
		if err != nil {
			return nil, err
		}

		// "If ip is not an IPv4 address, To4 returns nil."
		// Reference: http://golang.org/pkg/net/#IP.To4
		if ip4 := ip.To4(); ip4 != nil {
			return ip4, nil
		}
	}

	return nil, nil
}
