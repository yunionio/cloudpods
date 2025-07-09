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

package dhcp

import (
	"fmt"
	"net"
	"testing"
)

func compareBytes(b1, b2 []byte) error {
	if len(b1) != len(b2) {
		return fmt.Errorf("length not match")
	}
	for i := range b1 {
		if b1[i] != b2[i] {
			return fmt.Errorf("%dth not equal", i)
		}
	}
	return nil
}

func TestGetClasslessRoutePack(t *testing.T) {
	cases := []struct {
		net  string
		gw   string
		want []byte
	}{
		{
			net: "10.0.0.0/8",
			gw:  "10.168.120.1",
			want: []byte{
				8, 10, 10, 168, 120, 1,
			},
		},
		{
			net: "192.168.0.0/16",
			gw:  "10.168.120.1",
			want: []byte{
				16, 192, 168, 10, 168, 120, 1,
			},
		},
		{
			net: "172.16.0.0/12",
			gw:  "10.168.222.1",
			want: []byte{
				12, 172, 16, 10, 168, 222, 1,
			},
		},
	}
	for _, c := range cases {
		routeInfo, err := ParseRouteInfo([]string{c.net, c.gw})
		if err != nil {
			t.Errorf("parse route %s error: %v", c.net, err)
			continue
		}
		got := getClasslessRoutePack(*routeInfo)
		if err := compareBytes(c.want, got); err != nil {
			t.Errorf("net: %s gw: %s want: %#v got: %#v", c.net, c.gw, c.want, got)
		}
	}
}

func TestParseRouteInfo(t *testing.T) {
	cases := []struct {
		net  string
		gw   string
		want *SRouteInfo
	}{
		{
			net: "10.0.0.0/8",
			gw:  "10.168.120.1",
			want: &SRouteInfo{
				Prefix:    net.ParseIP("10.0.0.0"),
				PrefixLen: 8,
				Gateway:   net.ParseIP("10.168.120.1"),
			},
		},
		{
			net: "::/0",
			gw:  "3ffe:3200:fe::1",
			want: &SRouteInfo{
				Prefix:    net.ParseIP("::"),
				PrefixLen: 0,
				Gateway:   net.ParseIP("3ffe:3200:fe::1"),
			},
		},
		{
			net: "fd00:ec2::254/128",
			gw:  "::",
			want: &SRouteInfo{
				Prefix:    net.ParseIP("fd00:ec2::254"),
				PrefixLen: 128,
				Gateway:   net.ParseIP("::"),
			},
		},
		{
			net: "fe80::a9fe:a9fe/128",
			gw:  "::",
			want: &SRouteInfo{
				Prefix:    net.ParseIP("fe80::a9fe:a9fe"),
				PrefixLen: 128,
				Gateway:   net.ParseIP("::"),
			},
		},
	}
	for _, c := range cases {
		routeInfo, err := ParseRouteInfo([]string{c.net, c.gw})
		if err != nil {
			t.Errorf("parse route %s error: %v", c.net, err)
			continue
		}
		if routeInfo.String() != c.want.String() {
			t.Errorf("parse route expect %s got: %s", c.want.String(), routeInfo.String())
		}
	}
}
