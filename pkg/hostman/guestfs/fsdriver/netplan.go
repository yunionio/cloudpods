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

func NewNetplanConfig(allNics []*types.SServerNic, bondNics []*types.SServerNic, mainIp string) *netplan.Configuration {
	network := newNetplanNetwork(allNics, bondNics, mainIp)
	return netplan.NewConfiguration(network)
}

func newNetplanNetwork(allNics []*types.SServerNic, bondNics []*types.SServerNic, mainIp string) *netplan.Network {
	network := netplan.NewNetwork()

	nicCnt := len(allNics) - len(bondNics)
	for _, nic := range allNics {
		nicConf := getNetplanEthernetConfig(nic, false, mainIp, nicCnt)

		if nicConf == nil {
			continue
		}

		network.AddEthernet(nic.Name, nicConf)
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

		netConf := getNetplanEthernetConfig(bondNic, true, mainIp, nicCnt)

		if netConf.Mtu == 0 {
			netConf.Mtu = defaultMtu
		}

		// TODO: implement kinds of bond mode config
		bondConf := netplan.NewBondMode4(netConf, interfaces)

		network.AddBond(bondNic.Name, bondConf)
	}

	return network
}

func getNetplanEthernetConfig(nic *types.SServerNic, isBond bool, mainIp string, nicCnt int) *netplan.EthernetConfig {
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
			if nic.Ip == mainIp && len(mainIp) > 0 {
				gateway6 = nic.Gateway6
			}
		}

		var routeArrs = make([][]string, 0)
		routeArrs = netutils2.AddNicRoutes(routeArrs, nic, mainIp, nicCnt)

		var routes []*netplan.Route

		for _, route := range routeArrs {
			routes = append(routes, &netplan.Route{
				To:  route[0],
				Via: route[1],
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
		nicConf = netplan.NewDHCP4EthernetConfig()
		if len(nic.Ip6) > 0 {
			nicConf.EnableDHCP6()
		}
	}

	return nicConf
}
