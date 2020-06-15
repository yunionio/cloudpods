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

package models

import (
	"reflect"
	"sort"
	"testing"

	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

func TestSCloudaccount_suggestHostNetworks(t *testing.T) {
	cases := []struct {
		in   []string
		want []api.CASimpleNetConf
	}{
		{
			[]string{
				"10.168.13.234",
				"10.168.13.235",
				"10.168.13.233",
				"10.168.13.222",
			},
			[]api.CASimpleNetConf{
				{
					GuestIpStart: "10.168.13.222",
					GuestIpEnd:   "10.168.13.222",
					GuestIpMask:  24,
					GuestGateway: "10.168.13.1",
				},
				{
					GuestIpStart: "10.168.13.233",
					GuestIpEnd:   "10.168.13.235",
					GuestIpMask:  24,
					GuestGateway: "10.168.13.1",
				},
			},
		},
		{
			[]string{
				"10.168.12.254",
				"10.168.43.1",
			},
			[]api.CASimpleNetConf{
				{
					GuestIpStart: "10.168.12.254",
					GuestIpEnd:   "10.168.12.254",
					GuestIpMask:  24,
					GuestGateway: "10.168.12.1",
				},
				{
					GuestIpStart: "10.168.43.1",
					GuestIpEnd:   "10.168.43.1",
					GuestIpMask:  24,
					GuestGateway: "10.168.43.1",
				},
			},
		},
	}
	for _, c := range cases {
		ins := make([]netutils.IPV4Addr, len(c.in))
		for i := range ins {
			ins[i], _ = netutils.NewIPV4Addr(c.in[i])
		}
		out := CloudaccountManager.suggestHostNetworks(ins)
		sort.Slice(out, func(i, j int) bool {
			if out[i].GuestIpStart == out[j].GuestIpStart {
				return out[i].GuestIpEnd < out[j].GuestIpEnd
			}
			return out[i].GuestIpStart < out[j].GuestIpStart
		})
		if !reflect.DeepEqual(out, c.want) {
			t.Fatalf("want: %#v\nreal: %#v", c.want, out)
		}
	}
}

func TestSCloudaccount_suggestVMNetwors(t *testing.T) {
	cases := []struct {
		in1 []string
		in2 []struct {
			startIp string
			endIp   string
		}
		want []api.CASimpleNetConf
	}{
		{
			in1: []string{
				"10.168.222.23",
				"10.168.222.26",
				"10.168.222.145",
				"10.168.222.234",
			},
			in2: []struct {
				startIp string
				endIp   string
			}{
				{"10.168.222.45", "10.168.222.120"},
				{"10.168.222.200", "10.168.222.230"},
			},
			want: []api.CASimpleNetConf{
				{
					GuestIpStart: "10.168.222.1",
					GuestIpEnd:   "10.168.222.44",
					GuestIpMask:  24,
					GuestGateway: "10.168.222.1",
				},
				{
					GuestIpStart: "10.168.222.121",
					GuestIpEnd:   "10.168.222.199",
					GuestIpMask:  24,
					GuestGateway: "10.168.222.1",
				},
				{
					GuestIpStart: "10.168.222.231",
					GuestIpEnd:   "10.168.222.254",
					GuestIpMask:  24,
					GuestGateway: "10.168.222.1",
				},
			},
		},
		{
			in1: []string{
				"10.168.222.23",
				"10.168.224.178",
			},
			in2: []struct {
				startIp string
				endIp   string
			}{
				{"10.168.222.100", "10.168.224.100"},
			},
			want: []api.CASimpleNetConf{
				{
					GuestIpStart: "10.168.222.1",
					GuestIpEnd:   "10.168.222.99",
					GuestIpMask:  24,
					GuestGateway: "10.168.222.1",
				},
				{
					GuestIpStart: "10.168.224.101",
					GuestIpEnd:   "10.168.224.254",
					GuestIpMask:  24,
					GuestGateway: "10.168.224.1",
				},
			},
		},
	}
	for _, c := range cases {
		ins1 := make([]netutils.IPV4Addr, len(c.in1))
		ins2 := make([]netutils.IPV4AddrRange, len(c.in2))
		for i := range ins1 {
			ins1[i], _ = netutils.NewIPV4Addr(c.in1[i])
		}
		for i := range ins2 {
			ip1, _ := netutils.NewIPV4Addr(c.in2[i].startIp)
			ip2, _ := netutils.NewIPV4Addr(c.in2[i].endIp)
			ins2[i] = netutils.NewIPV4AddrRange(ip1, ip2)
		}
		out := CloudaccountManager.suggestVMNetwors(ins1, ins2)
		sort.Slice(out, func(i, j int) bool {
			if out[i].GuestIpStart == out[j].GuestIpStart {
				return out[i].GuestIpEnd < out[j].GuestIpEnd
			}
			return out[i].GuestIpStart < out[j].GuestIpStart
		})
		if !reflect.DeepEqual(out, c.want) {
			t.Fatalf("want: %#v\nreal: %#v", c.want, out)
		}
	}
}
