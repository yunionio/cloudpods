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
	"testing"
)

func TestOptionByte(t *testing.T) {
	o := Options{
		1: []byte{3},
		2: []byte{1, 2, 3},
	}

	b, err := o.Byte(1)
	if err != nil {
		t.Fatal(err)
	}
	if b != 3 {
		t.Fatalf("wanted value 3, got %d", b)
	}

	b, err = o.Byte(2)
	if err == nil {
		t.Fatalf("option shouldn't be a valid byte")
	}
}

func TestOptionUint16(t *testing.T) {
	o := Options{
		1: []byte{1, 2},
		2: []byte{1, 2, 3},
	}

	u, err := o.Uint16(1)
	if err != nil {
		t.Fatal(err)
	}
	if u != 258 {
		t.Fatalf("wanted value 258, got %d", u)
	}

	u, err = o.Uint16(2)
	if err == nil {
		t.Fatal("option shouldn't be a valid uint16")
	}
}

func TestOptionUint32(t *testing.T) {
	o := Options{
		1: []byte{1, 2, 3, 4},
		2: []byte{1, 2, 3},
	}

	u, err := o.Uint32(1)
	if err != nil {
		t.Fatal(err)
	}
	if u != 16909060 {
		t.Fatalf("wanted value 16909060, got %d", u)
	}

	u, err = o.Uint32(2)
	if err == nil {
		t.Fatal("option shouldn't be a valid uint32")
	}
}

func TestOptionInt32(t *testing.T) {
	o := Options{
		1: []byte{0xff, 0xff, 0xff, 0xff},
		2: []byte{1, 2, 3},
	}

	u, err := o.Int32(1)
	if err != nil {
		t.Fatal(err)
	}
	if u != -1 {
		t.Fatalf("wanted value -1, got %d", u)
	}

	u, err = o.Int32(2)
	if err == nil {
		t.Fatal("option shouldn't be a valid int32")
	}
}

func TestOptionIPs(t *testing.T) {
	o := Options{
		1: []byte{1, 2, 3, 4, 5, 6, 7, 8},
		2: []byte{1, 2, 3, 4, 5, 6},
		3: []byte{1, 2, 3},
	}

	ips, err := o.IPs(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) != 2 {
		t.Fatal("wrong number of IPs")
	}
	if !ips[0].Equal(net.IPv4(1, 2, 3, 4)) {
		t.Fatalf("wrong first IP, got %s", ips[0])
	}
	if !ips[1].Equal(net.IPv4(5, 6, 7, 8)) {
		t.Fatalf("wrong second IP, got %s", ips[0])
	}

	ips, err = o.IPs(2)
	if err == nil {
		t.Fatal("option shouldn't be a valid IPs")
	}

	ips, err = o.IPs(3)
	if err == nil {
		t.Fatal("option shouldn't be a valid IPs")
	}
}

func TestOptionIP(t *testing.T) {
	o := Options{
		1: []byte{1, 2, 3, 4},
		2: []byte{1, 2, 3, 4, 5, 6},
		3: []byte{1, 2, 3},
	}

	ip, err := o.IP(1)
	if err != nil {
		t.Fatal(err)
	}
	if !ip.Equal(net.IPv4(1, 2, 3, 4)) {
		t.Fatalf("wrong first IP, got %s", ip)
	}

	ip, err = o.IP(2)
	if err == nil {
		t.Fatal("option shouldn't be a valid IPs")
	}

	ip, err = o.IP(3)
	if err == nil {
		t.Fatal("option shouldn't be a valid IPs")
	}
}

func TestOptionIPMask(t *testing.T) {
	o := Options{
		1: []byte{1, 2, 3, 4},
		2: []byte{1, 2, 3, 4, 5, 6},
		3: []byte{1, 2, 3},
	}

	ipmask, err := o.IPMask(1)
	if err != nil {
		t.Fatal(err)
	}
	if !net.IP(ipmask).Equal(net.IP(net.IPv4Mask(1, 2, 3, 4))) {
		t.Fatalf("wrong first IP, got %s", ipmask)
	}

	ipmask, err = o.IPMask(2)
	if err == nil {
		t.Fatal("option shouldn't be a valid IPs")
	}

	ipmask, err = o.IPMask(3)
	if err == nil {
		t.Fatal("option shouldn't be a valid IPs")
	}
}

func TestCopy(t *testing.T) {
	o := Options{
		1: []byte{2},
		2: []byte{3, 4},
	}

	o2 := o.Copy()
	delete(o2, 2)

	if len(o) != 2 {
		t.Fatalf("Mutating Option copy mutated the original")
	}
}
