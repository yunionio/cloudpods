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
	"testing"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
)

func TestNetBytes2Mask(t *testing.T) {
	type args struct {
		mask []byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"test 255.255.255.255",
			args{[]byte{255, 255, 255, 255}},
			"255.255.255.255",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NetBytes2Mask(tt.args.mask); got != tt.want {
				t.Errorf("NetBytes2Mask() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatMac(t *testing.T) {
	type args struct {
		macStr string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test-format-mac-1",
			args: args{"FFFFFFFFFFFF"},
			want: "ff:ff:ff:ff:ff:ff",
		},
		{
			name: "test-format-mac-2",
			args: args{"FFFFFFFFFF"},
			want: "",
		},
		{
			name: "test-format-mac-3",
			args: args{"FFDDEECCBBAA"},
			want: "ff:dd:ee:cc:bb:aa",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatMac(tt.args.macStr); got != tt.want {
				t.Errorf("FormatMac() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewNetInterface(t *testing.T) {
	n := NewNetInterface("eth0")
	t.Logf("NetInterface: %s %s %s %s", n.name, n.Addr, n.Mask.String(), n.mac)
	addrs := n.GetAddresses()
	t.Logf("addrs: %s", addrs)
	slaves := n.GetSlaveAddresses()
	t.Logf("slaves: %s", slaves)
	routes := n.GetRouteSpecs()
	t.Logf("routes: %s", routes)
	for i := range routes {
		t.Logf("route to %s", routes[i].Dst.String())
	}

	m := NewNetInterface("docker0")
	m.ClearAddrs()
}

func TestMyDefault(t *testing.T) {
	myip, err := MyIP()
	if err != nil {
		// Skip if it's no route to host
		t.Fatalf("MyIP: %v", err)
	}

	if myip != "" {
		srcIp, ifname, err := DefaultSrcIpDev()
		if err != nil {
			t.Fatalf("default srcip dev: %v", err)
		}
		if srcIp.String() != myip {
			t.Errorf("myip: %s, srcip: %s", myip, srcIp.String())
		}
		if ifname == "" {
			t.Errorf("empty ifname")
		}
	}
}

func TestGetMainNicFromDeployApi(t *testing.T) {
	nics1 := []*types.SServerNic{
		{
			Ip:      "10.168.222.19",
			Gateway: "10.168.222.1",
		},
		{
			Ip:      "114.114.114.114",
			Gateway: "114.114.114.1",
		},
	}
	nics2 := []*types.SServerNic{
		{
			Ip: "10.168.222.19",
		},
		{
			Ip:      "114.114.114.114",
			Gateway: "114.114.114.1",
		},
	}
	nics3 := []*types.SServerNic{
		{
			Ip:      "10.168.222.19",
			Gateway: "10.168.222.1",
		},
		{
			Ip: "114.114.114.114",
		},
	}
	nics4 := []*types.SServerNic{
		{
			Ip: "10.168.222.19",
		},
		{
			Ip: "114.114.114.114",
		},
	}
	cases := []struct {
		nics []*types.SServerNic
		want *types.SServerNic
	}{
		{
			nics1,
			nics1[1],
		},
		{
			nics2,
			nics2[1],
		},
		{
			nics3,
			nics3[0],
		},
		{
			nics4,
			nics4[1],
		},
	}
	for _, c := range cases {
		got, err := GetMainNicFromDeployApi(c.nics)
		if err != nil {
			t.Errorf("error %s", err)
		} else if got != c.want {
			t.Errorf("error: got %v want %v", got, c.want)
		}
	}
}
