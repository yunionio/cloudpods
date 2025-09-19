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

package fsdriver

import (
	"testing"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
)

func TestNicDescToNetworkManager(t *testing.T) {
	cases := []struct {
		nicDesc *types.SServerNic
		mainIp  string
		mainIp6 string
		want    string
	}{
		{
			nicDesc: &types.SServerNic{
				Name:     "eth0",
				Ip:       "192.168.1.100",
				Masklen:  24,
				Gateway:  "192.168.1.1",
				Ip6:      "2001:db8::200",
				Masklen6: 64,
				Gateway6: "2001:db8::1",
				Mac:      "00:22:0a:0b:0c:0d",
				Manual:   false,
				Mtu:      1440,
			},
			mainIp:  "192.168.1.100",
			mainIp6: "2001:db8::1",
			want: `[connection]
id=eth0
uuid=de05e375-af25-9477-66ca-d4f41fe5e750
interface-name=eth0
type=ethernet
autoconnect=true

[ethernet]
mac-address=00:22:0a:0b:0c:0d
mtu=1440

[ipv4]
method=auto

[ipv6]
method=auto

`,
		},
		{
			nicDesc: &types.SServerNic{
				Name:     "eth0",
				Ip:       "192.168.1.100",
				Masklen:  24,
				Gateway:  "192.168.1.1",
				Ip6:      "2001:db8::200",
				Masklen6: 64,
				Gateway6: "2001:db8::1",
				Mac:      "00:22:0a:0b:0c:0d",
				Manual:   true,
			},
			mainIp:  "192.168.1.100",
			mainIp6: "2001:db8::200",
			want: `[connection]
id=eth0
uuid=de05e375-af25-9477-66ca-d4f41fe5e750
interface-name=eth0
type=ethernet
autoconnect=true

[ethernet]
mac-address=00:22:0a:0b:0c:0d

[ipv4]
method=manual
address1=192.168.1.100/24
gateway=192.168.1.1

[ipv6]
method=manual
address1=2001:db8::200/64
gateway=2001:db8::1

`,
		},
		{
			nicDesc: &types.SServerNic{
				Name:     "eth0",
				Ip:       "192.168.1.100",
				Masklen:  24,
				Gateway:  "192.168.1.1",
				Ip6:      "2001:db8::200",
				Masklen6: 64,
				Gateway6: "2001:db8::1",
				Mac:      "00:22:0a:0b:0c:0d",
				Domain:   "onecloud.io",
				Dns:      "192.168.1.1,192.168.1.2,fc00::3fe:1",
				Manual:   true,
			},
			mainIp:  "192.168.2.100",
			mainIp6: "2001:db7::100",
			want: `[connection]
id=eth0
uuid=de05e375-af25-9477-66ca-d4f41fe5e750
interface-name=eth0
type=ethernet
autoconnect=true

[ethernet]
mac-address=00:22:0a:0b:0c:0d

[ipv4]
method=manual
address1=192.168.1.100/24
dns=192.168.1.1,192.168.1.2

[ipv6]
method=manual
address1=2001:db8::200/64
dns=fc00::3fe:1

`,
		},
		{
			nicDesc: &types.SServerNic{
				Name:     "bond0",
				Ip:       "192.168.1.100",
				Masklen:  24,
				Gateway:  "192.168.1.1",
				Ip6:      "2001:db8::200",
				Masklen6: 64,
				Gateway6: "2001:db8::1",
				Mac:      "00:22:0a:0b:0c:0d",
				Domain:   "onecloud.io",
				Dns:      "192.168.1.1,192.168.1.2,fc00::3fe:1",
				TeamingSlaves: []*types.SServerNic{
					{
						Name: "eth0",
					},
					{
						Name: "eth1",
					},
				},
			},
			mainIp:  "192.168.2.100",
			mainIp6: "2001:db7::100",
			want: `[connection]
id=bond0
uuid=01bf7157-f43f-58f4-15b0-cfb6d6dfdb5b
interface-name=bond0
type=bond
autoconnect=true

[bond]
mode=802.3ad
miimon=100

[ethernet]
mac-address=00:22:0a:0b:0c:0d

[ipv4]
method=auto

[ipv6]
method=auto

`,
		},
		{
			nicDesc: &types.SServerNic{
				Name:     "eth0",
				Ip:       "192.168.1.100",
				Masklen:  24,
				Gateway:  "192.168.1.1",
				Ip6:      "2001:db8::200",
				Masklen6: 64,
				Gateway6: "2001:db8::1",
				Mac:      "00:22:0a:0b:0c:0d",
				Domain:   "onecloud.io",
				Dns:      "192.168.1.1,192.168.1.2,fc00::3fe:1",
				TeamingMaster: &types.SServerNic{
					Name: "bond0",
				},
			},
			mainIp:  "192.168.2.100",
			mainIp6: "2001:db7::100",
			want: `[connection]
id=eth0
uuid=de05e375-af25-9477-66ca-d4f41fe5e750
interface-name=eth0
type=ethernet
master=bond0
slave-type=bond

[ethernet]
mac-address=00:22:0a:0b:0c:0d

[ipv4]
method=disabled

[ipv6]
method=disabled

`,
		},
	}

	for _, c := range cases {
		got := nicDescToNetworkManager(c.nicDesc, c.mainIp, c.mainIp6)
		if got != c.want {
			t.Errorf("[[got]]\n%s\n[[want]]\n%s\n[[end]]", got, c.want)
		}
	}
}
