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
	"fmt"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/util/netplan"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

func NewNetplanConfig(allNics []*types.SServerNic, bondNics []*types.SServerNic, mainIp, mainIp6 string) *netplan.Configuration {
	network := newNetplanNetwork(allNics, bondNics, mainIp, mainIp6)
	return netplan.NewConfiguration(network)
}

func newNetplanNetwork(allNics []*types.SServerNic, bondNics []*types.SServerNic, mainIp, mainIp6 string) *netplan.Network {
	network := netplan.NewNetwork()

	nicCnt := len(allNics) - len(bondNics)
	for _, nic := range allNics {
		nicConf := getNetplanEthernetConfig(nic, false, mainIp, mainIp6, nicCnt)
		if nicConf == nil {
			continue
		}

		if nic.VlanInterface {
			ifname := fmt.Sprintf("%s.%d", nic.Name, nic.Vlan)
			vlanConfig := &netplan.VlanConfig{
				EthernetConfig: *nicConf,
				Link:           nic.Name,
				Id:             nic.Vlan,
			}
			network.AddVlan(ifname, vlanConfig)

			ethConfig := &netplan.EthernetConfig{
				DHCP4:      false,
				DHCP6:      false,
				MacAddress: nic.Mac,
				Match:      netplan.NewEthernetConfigMatchMac(nic.Mac),
			}
			network.AddEthernet(nic.Name, ethConfig)
		} else {
			network.AddEthernet(nic.Name, nicConf)
		}
	}

	for _, bondNic := range bondNics {
		if len(bondNic.TeamingSlaves) < 2 {
			log.Warningf("BondNic %s slaves nic %#v less than 2", bondNic.Name, bondNic.TeamingSlaves)
			continue
		}

		var defaultMtu = int16(1442)

		interfaces := make([]string, len(bondNic.TeamingSlaves))
		for i, sn := range bondNic.TeamingSlaves {
			interfaces[i] = sn.Name

			nicConf := &netplan.EthernetConfig{
				DHCP4:      false,
				MacAddress: sn.Mac,
				Match:      netplan.NewEthernetConfigMatchMac(sn.Mac),
			}

			if sn.Mtu > 0 {
				nicConf.Mtu = sn.Mtu
			} else {
				nicConf.Mtu = defaultMtu
			}

			network.AddEthernet(sn.Name, nicConf)
		}

		netConf := getNetplanEthernetConfig(bondNic, true, mainIp, mainIp6, nicCnt)

		if netConf.Mtu == 0 {
			netConf.Mtu = defaultMtu
		}

		// TODO: implement kinds of bond mode config
		bondConf := netplan.NewBondMode4(netConf, interfaces)

		network.AddBond(bondNic.Name, bondConf)
	}

	return network
}

func getNetplanEthernetConfig(nic *types.SServerNic, isBond bool, mainIp, mainIp6 string, nicCnt int) *netplan.EthernetConfig {
	var nicConf *netplan.EthernetConfig

	if !isBond && (nic.TeamingMaster != nil || nic.TeamingSlaves != nil) {
		return nil
	} else if nic.Virtual {
		addr := fmt.Sprintf("%s/32", netutils2.PSEUDO_VIP)
		nicConf = netplan.NewStaticEthernetConfig(addr, "", "", "", nil, nil, nil)
	} else if nic.Manual {
		addr := fmt.Sprintf("%s/%d", nic.Ip, nic.Masklen)
		gateway := ""
		if nic.Ip == mainIp && len(mainIp) > 0 {
			gateway = nic.Gateway
		}
		addr6 := ""
		gateway6 := ""
		if len(nic.Ip6) > 0 {
			addr6 = fmt.Sprintf("%s/%d", nic.Ip6, nic.Masklen6)
			if nic.Ip6 == mainIp6 && len(mainIp6) > 0 {
				gateway6 = nic.Gateway6
			}
		}

		routeArrs4 := make([]netutils2.SRouteInfo, 0)
		routeArrs6 := make([]netutils2.SRouteInfo, 0)
		routeArrs4, routeArrs6 = netutils2.AddNicRoutes(routeArrs4, routeArrs6, nic, mainIp, mainIp6, nicCnt)

		var routes []*netplan.Route

		for _, route := range routeArrs4 {
			routes = append(routes, &netplan.Route{
				To:  fmt.Sprintf("%s/%d", route.Prefix, route.PrefixLen),
				Via: route.Gateway.String(),
			})
		}
		for _, route := range routeArrs6 {
			routes = append(routes, &netplan.Route{
				To:  fmt.Sprintf("%s/%d", route.Prefix, route.PrefixLen),
				Via: route.Gateway.String(),
			})
		}

		nicConf = netplan.NewStaticEthernetConfig(
			addr, addr6, gateway, gateway6,
			[]string{nic.Domain},
			netutils2.GetNicDns(nic),
			routes,
		)
		nicConf.MacAddress = nic.Mac
		if nic.Mtu > 0 {
			nicConf.Mtu = nic.Mtu
		}
	} else {
		// dhcp
		nicConf = netplan.NewDHCPEthernetConfig()
		if len(nic.Ip) > 0 {
			nicConf.EnableDHCP4()
		}
		if len(nic.Ip6) > 0 {
			nicConf.EnableDHCP6()
		}
	}

	return nicConf
}
