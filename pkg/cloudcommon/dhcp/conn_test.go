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
	"net"
	"reflect"
	"testing"
	"time"
)

func testConn(t *testing.T, impl conn, addr string) {
	c := &Conn{impl, 0}

	s, err := net.Dial("udp4", addr)
	if err != nil {
		t.Fatal(err)
	}

	mac, err := net.ParseMAC("ce:e7:7b:ef:45:f7")
	if err != nil {
		t.Fatal(err)
	}

	p := &Packet{
		Type:          MsgDiscover,
		TransactionID: []byte("1234"),
		Broadcast:     true,
		HardwareAddr:  mac,
	}
	bs, err := p.Marshal()
	if err != nil {
		t.Fatalf("marshaling packet: %s", err)
	}
	// Unmarshal the packet again, to smooth out representation
	// differences (e.g. nil IP vs. IP set to 0.0.0.0).
	p, err = Unmarshal(bs)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		s.Write(bs)
	}()
	if err = c.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	rpkt, intf, err := c.RecvDHCP()
	if err != nil {
		t.Fatalf("reading DHCP packet: %s", err)
	}
	if !reflect.DeepEqual(p, rpkt) {
		t.Fatalf("DHCP packet not the same as when it was sent")
	}

	// Test writing
	p.ClientAddr = net.IPv4(127, 0, 0, 1)
	dhcpClientPort = s.LocalAddr().(*net.UDPAddr).Port
	bs2, err := p.Marshal()
	if err != nil {
		t.Fatalf("marshaling packet: %s", err)
	}
	// Unmarshal the packet again, to smooth out representation
	// differences (e.g. nil IP vs. IP set to 0.0.0.0).
	p, err = Unmarshal(bs2)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { dhcpClientPort = 68 }()

	ch := make(chan *Packet, 1)
	go func() {
		s.SetReadDeadline(time.Now().Add(time.Second))
		var buf [1500]byte
		n, err := s.Read(buf[:])
		if err != nil {
			t.Errorf("reading DHCP packet sent by conn_linux: %s", err)
			ch <- nil
			return
		}
		pkt, err := Unmarshal(buf[:n])
		if err != nil {
			t.Errorf("decoding DHCP packet: %s", err)
			ch <- nil
			return
		}
		ch <- pkt
	}()

	if err = c.SendDHCP(p, intf); err != nil {
		t.Fatalf("sending DHCP packet: %s", err)
	}

	rpkt = <-ch
	if rpkt == nil {
		t.FailNow()
	}
	if !reflect.DeepEqual(p, rpkt) {
		t.Fatalf("DHCP packet not the same as when it was sent")
	}
}

func TestPortableConn(t *testing.T) {
	// Use a listener to grab a free port, but we don't use it beyond
	// that.
	l, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	addr := l.LocalAddr().String()
	l.Close()

	c, err := newPortableConn(port)
	if err != nil {
		t.Fatalf("creating the conn: %s", err)
	}

	testConn(t, c, addr)
}
