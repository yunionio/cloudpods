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

package ethernet

// Package raw enables reading and writing data at the device driver level for
// a network interface.

import (
	"errors"
	"net"
	"time"

	"golang.org/x/net/bpf"
)

const (
	// Maximum read timeout per syscall.
	// It is required because read/recvfrom won't be interrupted on closing of the file descriptor.
	readTimeout = 200 * time.Millisecond
)

var (
	// ErrNotImplemented is returned when certain functionality is not yet
	// implemented for the host operating system.
	ErrNotImplemented = errors.New("raw: not implemented")
)

var _ net.Addr = &Addr{}

// Addr is a network address which can be used to contact other machines, using
// their hardware addresses.
type Addr struct {
	HardwareAddr net.HardwareAddr
}

// Network returns the address's network name, "raw".
func (a *Addr) Network() string {
	return "raw"
}

// String returns the address's hardware address.
func (a *Addr) String() string {
	return a.HardwareAddr.String()
}

var _ net.PacketConn = &Conn{}

// Conn is an implementation of the net.PacketConn interface which can send
// and receive data at the network interface device driver level.
type Conn struct {
	// packetConn is the operating system-specific implementation of
	// a raw connection.
	p *packetConn
}

// ReadFrom implements the net.PacketConn ReadFrom method.
func (c *Conn) ReadFrom(b []byte) (int, net.Addr, error) {
	return c.p.ReadFrom(b)
}

// WriteTo implements the net.PacketConn WriteTo method.
func (c *Conn) WriteTo(b []byte, addr net.Addr) (int, error) {
	return c.p.WriteTo(b, addr)
}

// Close closes the connection.
func (c *Conn) Close() error {
	return c.p.Close()
}

// LocalAddr returns the local network address.
func (c *Conn) LocalAddr() net.Addr {
	return c.p.LocalAddr()
}

// SetDeadline implements the net.PacketConn SetDeadline method.
func (c *Conn) SetDeadline(t time.Time) error {
	return c.p.SetDeadline(t)
}

// SetReadDeadline implements the net.PacketConn SetReadDeadline method.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.p.SetReadDeadline(t)
}

// SetWriteDeadline implements the net.PacketConn SetWriteDeadline method.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.p.SetWriteDeadline(t)
}

var _ bpf.Setter = &Conn{}

// SetBPF attaches an assembled BPF program to the connection.
func (c *Conn) SetBPF(filter []bpf.RawInstruction) error {
	return c.p.SetBPF(filter)
}

// SetPromiscuous enables or disables promiscuous mode on the interface, allowing it
// to receive traffic that is not addressed to the interface.
func (c *Conn) SetPromiscuous(b bool) error {
	return c.p.SetPromiscuous(b)
}

// Stats contains statistics about a Conn.
type Stats struct {
	// The total number of packets received.
	Packets uint64

	// The number of packets dropped.
	Drops uint64
}

// Stats retrieves statistics from the Conn.
//
// Only supported on Linux at this time.
func (c *Conn) Stats() (*Stats, error) {
	return c.p.Stats()
}

// ListenPacket creates a net.PacketConn which can be used to send and receive
// data at the network interface device driver level.
//
// ifi specifies the network interface which will be used to send and receive
// data.
//
// proto specifies the protocol (usually the EtherType) which should be
// captured and transmitted.  proto, if needed, is automatically converted to
// network byte order (big endian), akin to the htons() function in C.
//
// cfg specifies optional configuration which may be operating system-specific.
// A nil Config is equivalent to the default configuration: send and receive
// data at the network interface device driver level (usually raw Ethernet frames).
func ListenPacket(ifi *net.Interface, proto uint16, cfg *Config) (*Conn, error) {
	// A nil config is an empty Config.
	if cfg == nil {
		cfg = &Config{}
	}

	p, err := listenPacket(ifi, proto, *cfg)
	if err != nil {
		return nil, err
	}

	return &Conn{
		p: p,
	}, nil
}

// A Config can be used to specify additional options for a Conn.
type Config struct {
	// Linux only: call socket(7) with SOCK_DGRAM instead of SOCK_RAW.
	// Has no effect on other operating systems.
	LinuxSockDGRAM bool

	// Experimental: Linux only (for now, but can be ported to BSD):
	// disables repeated socket reads due to internal timeouts, at the expense
	// of losing the ability to cancel a ReadFrom operation by calling the Close
	// method of the net.PacketConn.
	//
	// Not recommended for programs which may need to open and close multiple
	// sockets during program runs.  This may save some CPU time by avoiding a
	// busy loop for programs which do not need timeouts, or programs which keep
	// a single socket open for the entire duration of the program.
	NoTimeouts bool

	// Linux only: do not accumulate packet socket statistic counters.  Packet
	// socket statistics are reset on each call to retrieve them via getsockopt,
	// but this package's default behavior is to continue accumulating the
	// statistics internally per Conn.  To use the Linux default behavior of
	// resetting statistics on each call to Stats, set this value to true.
	NoCumulativeStats bool
}

// htons converts a short (uint16) from host-to-network byte order.
// Thanks to mikioh for this neat trick:
// https://github.com/mikioh/-stdyng/blob/master/afpacket.go
func htons(i uint16) uint16 {
	return (i<<8)&0xff00 | i>>8
}

// Copyright (c) 2012 The Go Authors. All rights reserved.
// Source code in this file is based on src/net/interface_linux.go,
// from the Go standard library.  The Go license can be found here:
// https://golang.org/LICENSE.

// Taken from:
// https://github.com/golang/go/blob/master/src/net/net.go#L417-L421.
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "i/o timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }
