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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/util/netplan"
)

func TestNewNetplanConfig(t *testing.T) {
	cases := []struct {
		mainIp string
		nics   []*types.SServerNic
		want   *netplan.Configuration
	}{
		{
			mainIp: "10.168.222.175",
			nics: []*types.SServerNic{
				&types.SServerNic{
					Name:      "eth0",
					Index:     0,
					Bridge:    "br0",
					Domain:    "cloud.local",
					Ip:        "10.168.222.175",
					Vlan:      1,
					Driver:    "virtio",
					Masklen:   24,
					Virtual:   false,
					Manual:    true,
					WireId:    "399a06f3-7925-46c1-8b9f-d5a8580a74df",
					NetId:     "22c93412-5882-4de0-8357-45ce647ceada",
					Mac:       "00:24:c7:16:80:f2",
					BandWidth: 1000,
					Mtu:       1500,
					Dns:       "8.8.8.8",
					Ntp:       "",
					Net:       "vnet222",
					Interface: "ens5",
					Gateway:   "10.168.222.1",
					Ifname:    "vnet222-175",
					Routes:    nil,

					LinkUp: true,
				},
				&types.SServerNic{
					Name:      "eth1",
					Index:     1,
					Bridge:    "br0",
					Domain:    "cloud.local",
					Ip:        "",
					Vlan:      1,
					Driver:    "virtio",
					Masklen:   0,
					Virtual:   true,
					Manual:    true,
					WireId:    "399a06f3-7925-46c1-8b9f-d5a8580a74df",
					NetId:     "22c93412-5882-4de0-8357-45ce647ceada",
					Mac:       "00:24:c7:16:80:f3",
					BandWidth: 1000,
					Mtu:       1500,
					Dns:       "8.8.8.8",
					Ntp:       "",
					Net:       "vnet222",
					Interface: "ens5",
					Gateway:   "",
					Ifname:    "vnet222-bpg",
					Routes:    nil,

					LinkUp:   true,
					TeamWith: "00:24:c7:16:80:f2",
				},
			},
			want: &netplan.Configuration{
				Network: &netplan.Network{
					Version:  2,
					Renderer: netplan.NetworkRendererNetworkd,
					Ethernets: map[string]*netplan.EthernetConfig{
						"eth0": &netplan.EthernetConfig{
							MacAddress: "00:24:c7:16:80:f2",
							Match: &netplan.EthernetConfigMatch{
								MacAddress: "00:24:c7:16:80:f2",
							},
							Mtu: 1500,
						},
						"eth1": &netplan.EthernetConfig{
							MacAddress: "00:24:c7:16:80:f3",
							Match: &netplan.EthernetConfigMatch{
								MacAddress: "00:24:c7:16:80:f3",
							},
							Mtu: 1500,
						},
					},
					Bonds: map[string]*netplan.Bond{
						"bond0": &netplan.Bond{
							EthernetConfig: netplan.EthernetConfig{
								Addresses: []string{
									"10.168.222.175/24",
								},
								MacAddress: "00:24:c7:16:80:f2",
								Gateway4:   "10.168.222.1",
								Nameservers: &netplan.Nameservers{
									Search: []string{
										"cloud.local",
									},
									Addresses: []string{
										"8.8.8.8",
									},
								},
								Mtu: 1500,
								Routes: []*netplan.Route{
									&netplan.Route{
										To:  "169.254.169.254/32",
										Via: "0.0.0.0",
									},
								},
							},
							Interfaces: []string{
								"eth0",
								"eth1",
							},
							Parameters: &netplan.BondMode4Params{
								BondModeBaseParams: &netplan.BondModeBaseParams{
									Mode:               "802.3ad",
									MiiMonitorInterval: 100,
								},
							},
						},
					},
				},
			},
		},
		{
			mainIp: "10.168.22.175",
			nics: []*types.SServerNic{
				&types.SServerNic{
					Name:      "eth0",
					Index:     0,
					Bridge:    "br0",
					Domain:    "cloud.local",
					Ip:        "10.168.222.175",
					Vlan:      1,
					Driver:    "virtio",
					Masklen:   24,
					Virtual:   false,
					Manual:    true,
					WireId:    "399a06f3-7925-46c1-8b9f-d5a8580a74df",
					NetId:     "22c93412-5882-4de0-8357-45ce647ceada",
					Mac:       "00:24:c7:16:80:f2",
					BandWidth: 1000,
					Mtu:       1500,
					Dns:       "8.8.8.8",
					Ntp:       "",
					Net:       "vnet222",
					Interface: "ens5",
					Gateway:   "10.168.222.1",
					Ifname:    "vnet222-175",
					Routes:    nil,

					LinkUp: true,
				},
				&types.SServerNic{
					Name:      "eth1",
					Index:     1,
					Bridge:    "br0",
					Domain:    "cloud.local",
					Ip:        "",
					Vlan:      1,
					Driver:    "virtio",
					Masklen:   0,
					Virtual:   true,
					Manual:    true,
					WireId:    "399a06f3-7925-46c1-8b9f-d5a8580a74df",
					NetId:     "22c93412-5882-4de0-8357-45ce647ceada",
					Mac:       "00:24:c7:16:80:f3",
					BandWidth: 1000,
					Mtu:       1500,
					Dns:       "8.8.8.8",
					Ntp:       "",
					Net:       "vnet222",
					Interface: "ens5",
					Gateway:   "",
					Ifname:    "vnet222-bpg",
					Routes:    nil,

					LinkUp:   true,
					TeamWith: "00:24:c7:16:80:f2",
				},
			},
			want: &netplan.Configuration{
				Network: &netplan.Network{
					Version:  2,
					Renderer: netplan.NetworkRendererNetworkd,
					Ethernets: map[string]*netplan.EthernetConfig{
						"eth0": &netplan.EthernetConfig{
							MacAddress: "00:24:c7:16:80:f2",
							Match: &netplan.EthernetConfigMatch{
								MacAddress: "00:24:c7:16:80:f2",
							},
							Mtu: 1500,
						},
						"eth1": &netplan.EthernetConfig{
							MacAddress: "00:24:c7:16:80:f3",
							Match: &netplan.EthernetConfigMatch{
								MacAddress: "00:24:c7:16:80:f3",
							},
							Mtu: 1500,
						},
					},
					Bonds: map[string]*netplan.Bond{
						"bond0": &netplan.Bond{
							EthernetConfig: netplan.EthernetConfig{
								Addresses: []string{
									"10.168.222.175/24",
								},
								MacAddress: "00:24:c7:16:80:f2",
								Nameservers: &netplan.Nameservers{
									Search: []string{
										"cloud.local",
									},
									Addresses: []string{
										"8.8.8.8",
									},
								},
								Mtu: 1500,
							},
							Interfaces: []string{
								"eth0",
								"eth1",
							},
							Parameters: &netplan.BondMode4Params{
								BondModeBaseParams: &netplan.BondModeBaseParams{
									Mode:               "802.3ad",
									MiiMonitorInterval: 100,
								},
							},
						},
					},
				},
			},
		},
	}
	for i, c := range cases {
		allNics, bondNics := convertNicConfigs(c.nics)
		netplanConfig := NewNetplanConfig(allNics, bondNics, c.mainIp)
		if jsonutils.Marshal(netplanConfig).String() != jsonutils.Marshal(c.want).String() {
			t.Errorf("nics %d: %s want: %s got: %s", i, jsonutils.Marshal(c.nics), jsonutils.Marshal(c.want).PrettyString(), jsonutils.Marshal(netplanConfig).PrettyString())
		}
	}
}
