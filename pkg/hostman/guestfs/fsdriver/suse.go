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
			cmds.WriteString("STARTMODE=auto\n")
			cmds.WriteString("BOOTPROTO=static\n")
			cmds.WriteString(fmt.Sprintf("IPADDR=%s/%d\n", nicDesc.Ip, nicDesc.Masklen))

			if len(nicDesc.Ip6) > 0 {
				cmds.WriteString("IPV6INIT=yes\n")
				cmds.WriteString("IPV6_AUTOCONF=no\n")
				cmds.WriteString(fmt.Sprintf("IPADDR_V6=%s/%d\n", nicDesc.Ip6, nicDesc.Masklen6))
			}

			var routes = make([][]string, 0)
			routes = netutils2.AddNicRoutes(routes, nicDesc, mainIp, nicCnt)
			if len(nicDesc.Gateway) > 0 && nicDesc.Ip == mainIp {
				routes = append(routes, []string{
					"default",
					nicDesc.Gateway,
				})
			}
			if len(nicDesc.Gateway6) > 0 && nicDesc.Ip == mainIp {
				routes = append(routes, []string{
					"default",
					nicDesc.Gateway6,
				})
			}
			var rtbl strings.Builder
			for _, r := range routes {
				rtbl.WriteString(r[0])
				rtbl.WriteString(" ")
				rtbl.WriteString(r[1])
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

			dnslist := netutils2.GetNicDns(nicDesc)
			for i := 0; i < len(dnslist); i++ {
				if !utils.IsInArray(dnslist[i], dnsSrv) {
					dnsSrv = append(dnsSrv, dnslist[i])
				}
			}
		} else {
			cmds.WriteString("STARTMODE=auto\n")
			cmds.WriteString("BOOTPROTO=dhcp4\n")
			if len(nicDesc.Ip6) > 0 {
				// IPv6 support static temporarily
				// TODO
				cmds.WriteString("IPV6INIT=yes\n")
				cmds.WriteString("IPV6_AUTOCONF=no\n")
				cmds.WriteString(fmt.Sprintf("IPADDR_V6=%s/%d\n", nicDesc.Ip6, nicDesc.Masklen6))
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
