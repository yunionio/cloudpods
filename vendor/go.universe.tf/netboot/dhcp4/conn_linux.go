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

//+build linux

package dhcp4

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/bpf"
	"golang.org/x/net/ipv4"
)

type linuxConn struct {
	port uint16
	conn *ipv4.RawConn
}

// NewSnooperConn creates a Conn that listens on the given UDP ip:port.
//
// Unlike NewConn, NewSnooperConn does not bind to the ip:port,
// enabling the Conn to coexist with other services on the machine.
func NewSnooperConn(addr string) (*Conn, error) {
	return newConn(addr, newLinuxConn)
}

func newLinuxConn(port int) (conn, error) {
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

func (c *linuxConn) Recv(b []byte) (rb []byte, addr *net.UDPAddr, ifidx int, err error) {
	hdr, p, cm, err := c.conn.ReadFrom(b)
	if err != nil {
		return nil, nil, 0, err
	}
	if len(p) < 8 {
		return nil, nil, 0, errors.New("not a UDP packet, too short")
	}
	sport := int(binary.BigEndian.Uint16(p[:2]))
	return p[8:], &net.UDPAddr{IP: hdr.Src, Port: sport}, cm.IfIndex, nil
}

func (c *linuxConn) Send(b []byte, addr *net.UDPAddr, ifidx int) error {
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
