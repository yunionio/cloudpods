// MIT License
// ===========

// Copyright (C) 2015 Matt Layher

// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package ethernet

import (
	"errors"
	"net"
	"time"
)

/*
https://tools.ietf.org/html/rfc826

Ethernet transmission layer (not necessarily accessible to the user):
        48.bit: Ethernet address of destination
        48.bit: Ethernet address of sender
        16.bit: Protocol type = ether_type$ADDRESS_RESOLUTION
    Ethernet packet data:
        16.bit: (ar$hrd) Hardware address space (e.g., Ethernet,
                         Packet Radio Net.)
        16.bit: (ar$pro) Protocol address space.  For Ethernet
                         hardware, this is from the set of type
                         fields ether_typ$<protocol>.
         8.bit: (ar$hln) byte length of each hardware address
         8.bit: (ar$pln) byte length of each protocol address
        16.bit: (ar$op)  opcode (ares_op$REQUEST | ares_op$REPLY)
        nbytes: (ar$sha) Hardware address of sender of this
                         packet, n from the ar$hln field.
        mbytes: (ar$spa) Protocol address of sender of this
                         packet, m from the ar$pln field.
        nbytes: (ar$tha) Hardware address of target of this
                         packet (if known).
        mbytes: (ar$tpa) Protocol address of target.


Define the following for referring to the values put in the TYPE
field of the Ethernet packet header:
        ether_type$XEROX_PUP,
        ether_type$DOD_INTERNET,
        ether_type$CHAOS,
and a new one:
        ether_type$ADDRESS_RESOLUTION.
Also define the following values (to be discussed later):
        ares_op$REQUEST (= 1, high byte transmitted first) and
        ares_op$REPLY   (= 2),
and
        ares_hrd$Ethernet (= 1).
*/

// An Operation is an ARP operation, such as request or reply.
type Operation uint16

// Operation constants which indicate an ARP request or reply.
const (
	OperationRequest Operation = 1
	OperationReply   Operation = 2
)

// A Packet is a raw ARP packet, as described in RFC 826.
type Packet struct {
	// HardwareType specifies an IANA-assigned hardware type, as described
	// in RFC 826.
	HardwareType uint16

	// ProtocolType specifies the internetwork protocol for which the ARP
	// request is intended.  Typically, this is the IPv4 EtherType.
	ProtocolType uint16

	// HardwareAddrLength specifies the length of the sender and target
	// hardware addresses included in a Packet.
	HardwareAddrLength uint8

	// IPLength specifies the length of the sender and target IPv4 addresses
	// included in a Packet.
	IPLength uint8

	// Operation specifies the ARP operation being performed, such as request
	// or reply.
	Operation Operation

	// SenderHardwareAddr specifies the hardware address of the sender of this
	// Packet.
	SenderHardwareAddr net.HardwareAddr

	// SenderIP specifies the IPv4 address of the sender of this Packet.
	SenderIP net.IP

	// TargetHardwareAddr specifies the hardware address of the target of this
	// Packet.
	TargetHardwareAddr net.HardwareAddr

	// TargetIP specifies the IPv4 address of the target of this Packet.
	TargetIP net.IP
}

// NewPacket creates a new Packet from an input Operation and hardware/IPv4
// address values for both a sender and target.
//
// If either hardware address is less than 6 bytes in length, or there is a
// length mismatch between the two, ErrInvalidHardwareAddr is returned.
//
// If either IP address is not an IPv4 address, or there is a length mismatch
// between the two, ErrInvalidIP is returned.
func NewPacket(op Operation, srcHW net.HardwareAddr, srcIP net.IP, dstHW net.HardwareAddr, dstIP net.IP) (*Packet, error) {
	// Validate hardware addresses for minimum length, and matching length
	if len(srcHW) < 6 {
		return nil, ErrInvalidHardwareAddr
	}
	if len(dstHW) < 6 {
		return nil, ErrInvalidHardwareAddr
	}
	if len(srcHW) != len(dstHW) {
		return nil, ErrInvalidHardwareAddr
	}

	// Validate IP addresses to ensure they are IPv4 addresses, and
	// correct length
	srcIP = srcIP.To4()
	if srcIP == nil {
		return nil, ErrInvalidIP
	}
	dstIP = dstIP.To4()
	if dstIP == nil {
		return nil, ErrInvalidIP
	}

	return &Packet{
		// There is no Go-native way to detect hardware type of a network
		// interface, so default to 1 (ethernet 10Mb) for now
		HardwareType: 1,

		// Default to EtherType for IPv4
		ProtocolType: uint16(EtherTypeIPv4),

		// Populate other fields using input data
		HardwareAddrLength: uint8(len(srcHW)),
		IPLength:           uint8(len(srcIP)),
		Operation:          op,
		SenderHardwareAddr: srcHW,
		SenderIP:           srcIP,
		TargetHardwareAddr: dstHW,
		TargetIP:           dstIP,
	}, nil
}

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
	p, err := ListenPacket(ifi, protocolARP, nil)
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
func NewArpClient(ifi *net.Interface, p net.PacketConn) (*Client, error) {
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
	arp, err := NewPacket(OperationRequest, c.ifi.HardwareAddr, c.ip, Broadcast, ip)
	if err != nil {
		return err
	}
	return c.WriteTo(arp, Broadcast)
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
func (c *Client) Read() (*Packet, *Frame, error) {
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

	f := &Frame{
		Destination: p.TargetHardwareAddr,
		Source:      p.SenderHardwareAddr,
		EtherType:   EtherTypeARP,
		Payload:     pb,
	}

	fb, err := f.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = c.p.WriteTo(fb, &Addr{HardwareAddr: addr})
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
