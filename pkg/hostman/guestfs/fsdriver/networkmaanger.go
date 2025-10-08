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
	"strings"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func nicDescToNetworkManager(nicDesc *types.SServerNic, mainIp string, mainIp6 string, nicCnt int) string {
	var profile strings.Builder

	profile.WriteString("[connection]\n")
	profile.WriteString(fmt.Sprintf("id=%s\n", nicDesc.Name))
	profile.WriteString(fmt.Sprintf("uuid=%s\n", stringutils2.GenUuid(nicDesc.Name, nicDesc.Mac)))
	profile.WriteString(fmt.Sprintf("interface-name=%s\n", nicDesc.Name))
	if len(nicDesc.TeamingSlaves) > 0 {
		// bonding master
		profile.WriteString("type=bond\n")
		profile.WriteString("autoconnect=true\n")
	} else {
		profile.WriteString("type=ethernet\n")
		if nicDesc.TeamingMaster != nil {
			// bonding slave
			profile.WriteString(fmt.Sprintf("master=%s\n", nicDesc.TeamingMaster.Name))
			profile.WriteString("slave-type=bond\n")
		} else {
			// normal interface
			profile.WriteString("autoconnect=true\n")
		}
	}
	profile.WriteString("\n")

	if len(nicDesc.TeamingSlaves) > 0 {
		profile.WriteString("[bond]\n")
		profile.WriteString("mode=802.3ad\n")
		profile.WriteString("miimon=100\n")
		profile.WriteString("\n")
	}

	if len(nicDesc.Mac) > 0 && nicDesc.NicType != api.NIC_TYPE_INFINIBAND {
		profile.WriteString("[ethernet]\n")
		if len(nicDesc.TeamingSlaves) == 0 {
			// only real physical nic can set HWADDR
			// cmds.WriteString("HWADDR=")
			// cmds.WriteString(nicDesc.Mac)
			// cmds.WriteString("\n")
		}
		profile.WriteString(fmt.Sprintf("mac-address=%s\n", nicDesc.Mac))
		if nicDesc.Mtu > 0 {
			profile.WriteString(fmt.Sprintf("mtu=%d\n", nicDesc.Mtu))
		}
		profile.WriteString("\n")
	}

	if nicDesc.TeamingMaster != nil {
		// slave interface
		profile.WriteString("[ipv4]\n")
		profile.WriteString("method=disabled\n\n")
		profile.WriteString("[ipv6]\n")
		profile.WriteString("method=disabled\n\n")
	} else if nicDesc.Virtual {
		// virtual interface
		profile.WriteString("[ipv4]\n")
		profile.WriteString("method=manual\n")
		profile.WriteString(fmt.Sprintf("address1=%s/32\n", netutils2.PSEUDO_VIP))
		profile.WriteString("\n")
	} else if nicDesc.Manual {
		// manual interface
		if len(nicDesc.Ip) > 0 {
			profile.WriteString("[ipv4]\n")
			profile.WriteString("method=manual\n")
			profile.WriteString(fmt.Sprintf("address1=%s/%d\n", nicDesc.Ip, nicDesc.Masklen))
			if len(nicDesc.Gateway) > 0 && nicDesc.Ip == mainIp {
				profile.WriteString(fmt.Sprintf("gateway=%s\n", nicDesc.Gateway))
			}
			dnslist, _ := netutils2.GetNicDns(nicDesc)
			if len(dnslist) > 0 {
				profile.WriteString(fmt.Sprintf("dns=%s\n", strings.Join(dnslist, ",")))
			}
			var routes = make([][]string, 0)
			routes = netutils2.AddNicRoutes(routes, nicDesc, mainIp, nicCnt)
			for i := range routes {
				if len(routes[i]) >= 2 && routes[i][1] == "0.0.0.0" {
					profile.WriteString(fmt.Sprintf("route%d=%s\n", i+1, routes[i][0]))
				} else {
					profile.WriteString(fmt.Sprintf("route%d=%s\n", i+1, strings.Join(routes[i], ",")))
				}
			}
			profile.WriteString("\n")
		}
		if len(nicDesc.Ip6) > 0 {
			profile.WriteString("[ipv6]\n")
			profile.WriteString("method=manual\n")
			profile.WriteString(fmt.Sprintf("address1=%s/%d\n", nicDesc.Ip6, nicDesc.Masklen6))
			if len(nicDesc.Gateway6) > 0 && nicDesc.Ip6 == mainIp6 {
				profile.WriteString(fmt.Sprintf("gateway=%s\n", nicDesc.Gateway6))
			}
			_, dns6list := netutils2.GetNicDns(nicDesc)
			if len(dns6list) > 0 {
				profile.WriteString(fmt.Sprintf("dns=%s\n", strings.Join(dns6list, ",")))
			}
			profile.WriteString("\n")
		}
	} else {
		// dhcp interface
		if len(nicDesc.Ip) > 0 {
			profile.WriteString("[ipv4]\n")
			profile.WriteString("method=auto\n\n")
		}
		if len(nicDesc.Ip6) > 0 {
			profile.WriteString("[ipv6]\n")
			profile.WriteString("method=auto\n\n")
		}
	}

	return profile.String()
}
