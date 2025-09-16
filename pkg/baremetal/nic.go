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

package baremetal

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"yunion.io/x/pkg/util/netutils"

	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/util/dhcp"
)

func GetNicDHCPConfig(
	n *types.SNic,
	serverIP string,
	hostName string,
	isPxe bool,
	arch uint16,
) (*dhcp.ResponseConfig, error) {
	if n == nil {
		return nil, fmt.Errorf("Nic is nil")
	}
	if n.IpAddr == "" {
		return nil, fmt.Errorf("Nic no ip address")
	}
	ipAddr, err := netutils.NewIPV4Addr(n.IpAddr)
	if err != nil {
		return nil, fmt.Errorf("Parse IP address error: %q", n.IpAddr)
	}

	subnetMask := net.ParseIP(netutils.Masklen2Mask(n.Masklen).String())

	routes := make([][]string, 0)
	for _, route := range n.Routes {
		routes = append(routes, []string{route[0], route[1]})
	}

	parseIPs := func(ips string) []net.IP {
		ret := make([]net.IP, 0)
		iplist := strings.Split(ips, ",")
		for _, ip := range iplist {
			ret = append(ret, net.ParseIP(ip))
		}
		return ret
	}

	parseDomains := func(domains string) []net.IP {
		ret := make([]net.IP, 0)
		domainlist := strings.Split(domains, ",")
		for _, domain := range domainlist {
			addrs, _ := net.LookupHost(domain)
			for _, addr := range addrs {
				ret = append(ret, net.ParseIP(addr))
			}
		}
		return ret
	}

	conf := &dhcp.ResponseConfig{
		ServerIP:      net.ParseIP(serverIP),
		ClientIP:      net.ParseIP(ipAddr.String()),
		Gateway:       net.ParseIP(n.Gateway),
		SubnetMask:    subnetMask,
		BroadcastAddr: net.ParseIP(ipAddr.BroadcastAddr(n.Masklen).String()),
		DNSServers:    parseIPs(n.Dns),
		NTPServers:    parseDomains(n.Ntp),
		Domain:        n.Domain,
		OsName:        "Linux",
		Hostname:      hostName,
		Routes:        routes,
		LeaseTime:     time.Duration(o.Options.DhcpLeaseTime) * time.Second,
		RenewalTime:   time.Duration(o.Options.DhcpRenewalTime) * time.Second,
	}

	if isPxe {
		conf.BootServer = serverIP
		if len(o.Options.TftpBootServer) > 0 {
			conf.BootServer = o.Options.TftpBootServer
		}
		switch arch {
		case dhcp.CLIENT_ARCH_EFI_BC, dhcp.CLIENT_ARCH_EFI_X86_64:
			if o.Options.BootLoader == o.BOOT_LOADER_SYSLINUX {
				conf.BootFile = "bootx64.efi"
			} else {
				conf.BootFile = "grub_bootx64.efi"
			}
		case dhcp.CLIENT_ARCH_EFI_IA32:
			conf.BootFile = "bootia32.efi"
		case dhcp.CLIENT_ARCH_EFI_ARM64:
			conf.BootFile = "grub_arm64.efi"
		default:
			//if o.Options.EnableTftpHttpDownload {
			// bootFile = "lpxelinux.0"
			//}else {
			// bootFile := "pxelinux.0"
			//}
			if o.Options.BootLoader == o.BOOT_LOADER_SYSLINUX {
				conf.BootFile = "lpxelinux.0"
			} else {
				conf.BootFile = "grub_booti386"
			}
		}
		if len(o.Options.TftpBootFilename) > 0 {
			conf.BootFile = o.Options.TftpBootFilename
		}
		if len(o.Options.TftpBootServer) > 0 {
			conf.BootBlock = getPxeBlockSize(o.Options.TftpBootFilesize)
		} else {
			pxePath := filepath.Join(o.Options.TftpRoot, conf.BootFile)
			if f, err := os.Open(pxePath); err != nil {
				return nil, err
			} else {
				if info, err := f.Stat(); err != nil {
					return nil, err
				} else {
					conf.BootBlock = getPxeBlockSize(info.Size())
				}
			}
		}
	}
	return conf, nil
}

func getPxeBlockSize(pxeSize int64) uint16 {
	pxeBlk := pxeSize / 512
	if pxeSize > pxeBlk*512 {
		pxeBlk += 1
	}
	return uint16(pxeBlk)
}
