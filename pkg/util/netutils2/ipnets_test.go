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
	"fmt"
	"net"
	"testing"
)

func TestStr2IPNets(t *testing.T) {
	tests := []struct {
		cidr string
		want []*net.IPNet
	}{
		{
			cidr: "192.168.1.0/24",
			want: []*net.IPNet{
				{IP: net.IPv4(192, 168, 1, 0), Mask: net.CIDRMask(24, 32)},
			},
		},
		{
			cidr: "192.168.1.0/24,192.168.2.0/24",
			want: []*net.IPNet{
				{IP: net.IPv4(192, 168, 1, 0), Mask: net.CIDRMask(24, 32)},
				{IP: net.IPv4(192, 168, 2, 0), Mask: net.CIDRMask(24, 32)},
			},
		},
		{
			cidr: "192.168.1.0/24,192.168.2.0/24,192.168.3.0/24",
			want: []*net.IPNet{
				{IP: net.IPv4(192, 168, 1, 0), Mask: net.CIDRMask(24, 32)},
				{IP: net.IPv4(192, 168, 2, 0), Mask: net.CIDRMask(23, 32)},
			},
		},
		{
			cidr: "192.168.2.0/24,192.168.2.0/24,192.168.3.0/24",
			want: []*net.IPNet{
				{IP: net.IPv4(192, 168, 2, 0), Mask: net.CIDRMask(23, 32)},
			},
		},
	}

	for _, test := range tests {
		got := Str2IPNets(test.cidr)
		gotStr := fmt.Sprintf("%v", got)
		wantStr := fmt.Sprintf("%v", test.want)
		if gotStr != wantStr {
			t.Errorf("Str2IPNets(%s) = %s, want %s", test.cidr, gotStr, wantStr)
		}
	}
}
