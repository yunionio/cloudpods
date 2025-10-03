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
	"net"
	"path"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

type sSuseLikeRootFs struct {
	*sLinuxRootFs
}

func newSuseLikeRootFs(part IDiskPartition) *sSuseLikeRootFs {
	return &sSuseLikeRootFs{
		sLinuxRootFs: newLinuxRootFs(part),
	}
}

func (r *sSuseLikeRootFs) PrepareFsForTemplate(rootFs IDiskPartition) error {
	if err := r.sLinuxRootFs.PrepareFsForTemplate(rootFs); err != nil {
		return err
	}
	return r.CleanNetworkScripts(rootFs)
}

func (r *sSuseLikeRootFs) CleanNetworkScripts(rootFs IDiskPartition) error {
	networkPath := "/etc/sysconfig/network"
	files := rootFs.ListDir(networkPath, false)
	for i := 0; i < len(files); i++ {
		if strings.HasPrefix(files[i], "ifcfg-") && files[i] != "ifcfg-lo" {
			rootFs.Remove(path.Join(networkPath, files[i]), false)
			continue
		}
		if strings.HasPrefix(files[i], "ifroute-") {
			rootFs.Remove(path.Join(networkPath, files[i]), false)
		}
	}
	return nil
}

func (r *sSuseLikeRootFs) RootSignatures() []string {
	sig := r.sLinuxRootFs.RootSignatures()
	return append([]string{"etc/SUSE-brand", "/etc/os-release"}, sig...)
}

func (r *sSuseLikeRootFs) DeployHostname(rootFs IDiskPartition, hn, domain string) error {
	if rootFs.Exists("/etc/HOSTNAME", false) {
		return rootFs.FilePutContents("/etc/HOSTNAME", getHostname(hn, domain), false, false)
	}
	return nil
}

func (r *sSuseLikeRootFs) enableBondingModule(rootFs IDiskPartition, bondNics []*types.SServerNic) error {
	var content strings.Builder
	for i := range bondNics {
		content.WriteString("alias ")
		content.WriteString(bondNics[i].Name)
		content.WriteString(" bonding\n options ")
		content.WriteString(bondNics[i].Name)
		content.WriteString(" miimon=100 mode=4 lacp_rate=1 xmit_hash_policy=1\n")
	}
	return rootFs.FilePutContents("/etc/modprobe.d/bonding.conf", content.String(), false, false)
}

func (r *sSuseLikeRootFs) deployVlanNetworkingScripts(rootFs IDiskPartition, scriptPath, mainIp, mainIp6 string, nicCnt int, nicDesc *types.SServerNic) error {
	var cmds strings.Builder
	var ifname = fmt.Sprintf("%s.%d", nicDesc.Name, nicDesc.Vlan)
	cmds.WriteString("STARTMODE=auto\n")
	if nicDesc.Mtu > 0 {
		cmds.WriteString(fmt.Sprintf("MTU=%d\n", nicDesc.Mtu))
	}
	cmds.WriteString("BOOTPROTO=static\n")
	cmds.WriteString(fmt.Sprintf("IPADDR=%s/%d\n", nicDesc.Ip, nicDesc.Masklen))

	if len(nicDesc.Ip6) > 0 {
		cmds.WriteString("IPV6INIT=yes\n")
		cmds.WriteString("IPV6_AUTOCONF=no\n")
		cmds.WriteString(fmt.Sprintf("IPADDR_V6=%s/%d\n", nicDesc.Ip6, nicDesc.Masklen6))
	}

	cmds.WriteString(fmt.Sprintf("VLAN_ID=%d\n", nicDesc.Vlan))
	cmds.WriteString(fmt.Sprintf("ETHERDEVICE=%s\n", nicDesc.Name))

	var routes4 = make([]netutils2.SRouteInfo, 0)
	var routes6 = make([]netutils2.SRouteInfo, 0)
	var dnsSrv []string
	routes4, routes6 = netutils2.AddNicRoutes(routes4, routes6, nicDesc, mainIp, mainIp6, nicCnt)
	if len(nicDesc.Gateway) > 0 && nicDesc.Ip == mainIp {
		routes4 = append(routes4, netutils2.SRouteInfo{
			SPrefixInfo: netutils2.SPrefixInfo{
				Prefix:    net.ParseIP("0.0.0.0"),
				PrefixLen: 0,
			},
			Gateway: net.ParseIP(nicDesc.Gateway),
		})
	}
	if len(nicDesc.Gateway6) > 0 && nicDesc.Ip6 == mainIp6 {
		routes6 = append(routes6, netutils2.SRouteInfo{
			SPrefixInfo: netutils2.SPrefixInfo{
				Prefix:    net.ParseIP("::"),
				PrefixLen: 0,
			},
			Gateway: net.ParseIP(nicDesc.Gateway6),
		})
	}
	var rtbl strings.Builder
	for _, r := range routes4 {
		rtbl.WriteString(fmt.Sprintf("%s/%d", r.Prefix, r.PrefixLen))
		rtbl.WriteString(" ")
		rtbl.WriteString(r.Gateway.String())
		rtbl.WriteString(" - ")
		rtbl.WriteString(nicDesc.Name)
		rtbl.WriteString("\n")
	}
	for _, r := range routes6 {
		rtbl.WriteString(fmt.Sprintf("%s/%d", r.Prefix, r.PrefixLen))
		rtbl.WriteString(" ")
		rtbl.WriteString(r.Gateway.String())
		rtbl.WriteString(" - ")
		rtbl.WriteString(nicDesc.Name)
		rtbl.WriteString("\n")
	}
	rtblStr := rtbl.String()
	if len(rtblStr) > 0 {
		var fn = fmt.Sprintf("/etc/sysconfig/network/ifroute-%s", ifname)
		if err := rootFs.FilePutContents(fn, rtblStr, false, false); err != nil {
			return err
		}
	}

	dns4list, dns6list := netutils2.GetNicDns(nicDesc)
	for i := 0; i < len(dns4list); i++ {
		if !utils.IsInArray(dns4list[i], dnsSrv) {
			dnsSrv = append(dnsSrv, dns4list[i])
		}
	}
	for i := 0; i < len(dns6list); i++ {
		if !utils.IsInArray(dns6list[i], dnsSrv) {
			dnsSrv = append(dnsSrv, dns6list[i])
		}
	}

	var fn = fmt.Sprintf("/etc/sysconfig/network/ifcfg-%s", ifname)
	log.Debugf("%s: %s", fn, cmds.String())
	if err := rootFs.FilePutContents(fn, cmds.String(), false, false); err != nil {
		return err
	}

	if len(dnsSrv) > 0 {
		cont, err := rootFs.FileGetContents("/etc/sysconfig/network/config", false)
		if err != nil {
			return errors.Wrap(err, "FileGetContents config")
		}

		lines := strings.Split(string(cont), "\n")
		for i := range lines {
			line := strings.TrimSpace(lines[i])
			if strings.HasPrefix(line, "NETCONFIG_DNS_STATIC_SERVERS=") {
				lines[i] = fmt.Sprintf("NETCONFIG_DNS_STATIC_SERVERS=\"%s\"", strings.Join(dnsSrv, " "))
			}
		}
		err = rootFs.FilePutContents("/etc/sysconfig/network/config", strings.Join(lines, "\n"), false, false)
		if err != nil {
			return errors.Wrap(err, "FilePutContents config")
		}
	}

	return nil
}

func (r *sSuseLikeRootFs) deployNetworkingScripts(rootFs IDiskPartition, nics []*types.SServerNic) error {
	if err := r.sLinuxRootFs.DeployNetworkingScripts(rootFs, nics); err != nil {
		return err
	}

	// ToServerNics(nics)
	allNics, bondNics := convertNicConfigs(nics)
	if len(bondNics) > 0 {
		err := r.enableBondingModule(rootFs, bondNics)
		if err != nil {
			return err
		}
	}

	nicCnt := len(allNics) - len(bondNics)

	var dnsSrv []string

	mainNic := getMainNic(allNics)
	var mainIp string
	if mainNic != nil {
		mainIp = mainNic.Ip
	}
	mainNic6 := getMainNic6(allNics)
	var mainIp6 string
	if mainNic6 != nil {
		mainIp6 = mainNic6.Ip6
	}

	for i := range allNics {
		nicDesc := allNics[i]
		var cmds strings.Builder
		cmds.WriteString("STARTMODE=auto\n")
		if nicDesc.Mtu > 0 {
			cmds.WriteString(fmt.Sprintf("MTU=%d\n", nicDesc.Mtu))
		}
		if len(nicDesc.Mac) > 0 {
			cmds.WriteString("LLADDR=")
			cmds.WriteString(nicDesc.Mac)
			cmds.WriteString("\n")
		}
		if len(nicDesc.TeamingSlaves) != 0 {
			cmds.WriteString(`BONDING_OPTS="mode=4 miimon=100"\n`)
		}
		if nicDesc.TeamingMaster != nil {
			cmds.WriteString("BOOTPROTO=none\n")
			cmds.WriteString("MASTER=")
			cmds.WriteString(nicDesc.TeamingMaster.Name)
			cmds.WriteString("\n")
			cmds.WriteString("SLAVE=yes\n")
		} else if nicDesc.Virtual {
			cmds.WriteString("BOOTPROTO=none\n")
			cmds.WriteString("NETMASK=255.255.255.255\n")
			cmds.WriteString("IPADDR=")
			cmds.WriteString(netutils2.PSEUDO_VIP)
			cmds.WriteString("\n")
		} else if nicDesc.Manual {
			if nicDesc.VlanInterface {
				if err := r.deployVlanNetworkingScripts(rootFs, "", mainIp, mainIp6, nicCnt, nicDesc); err != nil {
					return err
				}
			} else {

				cmds.WriteString("STARTMODE=auto\n")
				cmds.WriteString("BOOTPROTO=static\n")
				if len(nicDesc.Ip) > 0 {
					cmds.WriteString(fmt.Sprintf("IPADDR=%s/%d\n", nicDesc.Ip, nicDesc.Masklen))
				}
				if len(nicDesc.Ip6) > 0 {
					cmds.WriteString("IPV6INIT=yes\n")
					cmds.WriteString("IPV6_AUTOCONF=no\n")
					cmds.WriteString(fmt.Sprintf("IPADDR_V6=%s/%d\n", nicDesc.Ip6, nicDesc.Masklen6))
				}
			}
			var routes4 = make([]netutils2.SRouteInfo, 0)
			var routes6 = make([]netutils2.SRouteInfo, 0)
			routes4, routes6 = netutils2.AddNicRoutes(routes4, routes6, nicDesc, mainIp, mainIp6, nicCnt)
			if len(nicDesc.Gateway) > 0 && nicDesc.Ip == mainIp {
				routes4 = append(routes4, netutils2.SRouteInfo{
					SPrefixInfo: netutils2.SPrefixInfo{
						Prefix:    net.ParseIP("0.0.0.0"),
						PrefixLen: 0,
					},
					Gateway: net.ParseIP(nicDesc.Gateway),
				})
			}
			if len(nicDesc.Gateway6) > 0 && nicDesc.Ip6 == mainIp6 {
				routes6 = append(routes6, netutils2.SRouteInfo{
					SPrefixInfo: netutils2.SPrefixInfo{
						Prefix:    net.ParseIP("::"),
						PrefixLen: 0,
					},
					Gateway: net.ParseIP(nicDesc.Gateway6),
				})
			}
			var rtbl strings.Builder
			for _, r := range routes4 {
				rtbl.WriteString(fmt.Sprintf("%s/%d", r.Prefix, r.PrefixLen))
				rtbl.WriteString(" ")
				rtbl.WriteString(r.Gateway.String())
				rtbl.WriteString(" - ")
				rtbl.WriteString(nicDesc.Name)
				rtbl.WriteString("\n")
			}
			for _, r := range routes6 {
				rtbl.WriteString(fmt.Sprintf("%s/%d", r.Prefix, r.PrefixLen))
				rtbl.WriteString(" ")
				rtbl.WriteString(r.Gateway.String())
				rtbl.WriteString(" - ")
				rtbl.WriteString(nicDesc.Name)
				rtbl.WriteString("\n")
			}
			rtblStr := rtbl.String()
			if len(rtblStr) > 0 {
				var fn = fmt.Sprintf("/etc/sysconfig/network/ifroute-%s", nicDesc.Name)
				if err := rootFs.FilePutContents(fn, rtblStr, false, false); err != nil {
					return err
				}
			}

			dns4list, dns6list := netutils2.GetNicDns(nicDesc)
			for i := 0; i < len(dns4list); i++ {
				if !utils.IsInArray(dns4list[i], dnsSrv) {
					dnsSrv = append(dnsSrv, dns4list[i])
				}
			}
			for i := 0; i < len(dns6list); i++ {
				if !utils.IsInArray(dns6list[i], dnsSrv) {
					dnsSrv = append(dnsSrv, dns6list[i])
				}
			}

		} else {
			cmds.WriteString("STARTMODE=auto\n")
			if len(nicDesc.Ip) > 0 && len(nicDesc.Ip6) == 0 {
				cmds.WriteString("BOOTPROTO=dhcp4\n")
			} else if len(nicDesc.Ip) == 0 && len(nicDesc.Ip6) > 0 {
				cmds.WriteString("BOOTPROTO=dhcp6\n")
			} else if len(nicDesc.Ip) > 0 && len(nicDesc.Ip6) > 0 {
				cmds.WriteString("BOOTPROTO=dhcp\n")
			} else {
				cmds.WriteString("BOOTPROTO=none\n")
			}
		}
		var fn = fmt.Sprintf("/etc/sysconfig/network/ifcfg-%s", nicDesc.Name)
		log.Debugf("%s: %s", fn, cmds.String())
		if err := rootFs.FilePutContents(fn, cmds.String(), false, false); err != nil {
			return err
		}
	}
	if len(dnsSrv) > 0 {
		cont, err := rootFs.FileGetContents("/etc/sysconfig/network/config", false)
		if err != nil {
			return errors.Wrap(err, "FileGetContents config")
		}

		lines := strings.Split(string(cont), "\n")
		for i := range lines {
			line := strings.TrimSpace(lines[i])
			if strings.HasPrefix(line, "NETCONFIG_DNS_STATIC_SERVERS=") {
				lines[i] = fmt.Sprintf("NETCONFIG_DNS_STATIC_SERVERS=\"%s\"", strings.Join(dnsSrv, " "))
			}
		}
		err = rootFs.FilePutContents("/etc/sysconfig/network/config", strings.Join(lines, "\n"), false, false)
		if err != nil {
			return errors.Wrap(err, "FilePutContents config")
		}
	}
	return nil
}

func (r *sSuseLikeRootFs) DeployStandbyNetworkingScripts(rootFs IDiskPartition, nics, nicsStandby []*types.SServerNic) error {
	if err := r.sLinuxRootFs.DeployStandbyNetworkingScripts(rootFs, nics, nicsStandby); err != nil {
		return err
	}
	for _, nic := range nicsStandby {
		var cmds string
		if len(nic.NicType) == 0 || nic.NicType != "ipmi" {
			cmds += fmt.Sprintf("LLADDR=%s\n", nic.Mac)
			cmds += "STARTMODE=off\n"
			var fn = fmt.Sprintf("/etc/sysconfig/network/ifcfg-%s%d", NetDevPrefix, nic.Index)
			if err := rootFs.FilePutContents(fn, cmds, false, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *sSuseLikeRootFs) enableSerialConsole(drv IRootFsDriver, rootFs IDiskPartition, sysInfo *jsonutils.JSONDict) error {
	return r.enableSerialConsoleSystemd(rootFs)
}

func (r *sSuseLikeRootFs) disableSerialConcole(drv IRootFsDriver, rootFs IDiskPartition) error {
	r.disableSerialConsoleSystemd(rootFs)
	return nil
}

type SOpenSuseRootFs struct {
	*sSuseLikeRootFs
}

func NewOpenSuseRootFs(part IDiskPartition) IRootFsDriver {
	return &SOpenSuseRootFs{sSuseLikeRootFs: newSuseLikeRootFs(part)}
}

func (c *SOpenSuseRootFs) String() string {
	return "OpenSuseRootFs"
}

func (c *SOpenSuseRootFs) GetName() string {
	return "OpenSUSE"
}

func (c *SOpenSuseRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	rel, _ := rootFs.FileGetContents("/etc/os-release", false)
	var version string
	if len(rel) > 0 {
		lines := strings.Split(string(rel), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "VERSION=") {
				version = strings.Trim(line[len("VERSION="):], " \"'")
				break
			}
		}
	}
	return deployapi.NewReleaseInfo(c.GetName(), version, c.GetArch(rootFs))
}

func (c *SOpenSuseRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []*types.SServerNic) error {
	if err := c.sSuseLikeRootFs.deployNetworkingScripts(rootFs, nics); err != nil {
		return err
	}
	return nil
}
