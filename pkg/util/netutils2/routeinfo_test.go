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

package netutils2

import (
	"net"
	"testing"
)

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
				SPrefixInfo: SPrefixInfo{
					Prefix:    net.ParseIP("10.0.0.0"),
					PrefixLen: 8,
				},
				Gateway: net.ParseIP("10.168.120.1"),
			},
		},
		{
			net: "::/0",
			gw:  "3ffe:3200:fe::1",
			want: &SRouteInfo{
				SPrefixInfo: SPrefixInfo{
					Prefix:    net.ParseIP("::"),
					PrefixLen: 0,
				},
				Gateway: net.ParseIP("3ffe:3200:fe::1"),
			},
		},
		{
			net: "fd00:ec2::254/128",
			gw:  "::",
			want: &SRouteInfo{
				SPrefixInfo: SPrefixInfo{
					Prefix:    net.ParseIP("fd00:ec2::254"),
					PrefixLen: 128,
				},
				Gateway: net.ParseIP("::"),
			},
		},
		{
			net: "fe80::a9fe:a9fe/128",
			gw:  "::",
			want: &SRouteInfo{
				SPrefixInfo: SPrefixInfo{
					Prefix:    net.ParseIP("fe80::a9fe:a9fe"),
					PrefixLen: 128,
				},
				Gateway: net.ParseIP("::"),
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
