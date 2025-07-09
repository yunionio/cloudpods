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
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"

	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/util/dhcp"
)

func GetNicDHCPConfig(
	n *types.SNic,
	serverIP net.IP,
	serverMac net.HardwareAddr,
	hostName string,
	isPxe bool,
	arch uint16,
	osName string,
) (*dhcp.ResponseConfig, error) {
	if n == nil {
		return nil, errors.Wrap(errors.ErrEmpty, "Nic is nil")
	}
	if n.IpAddr == "" && n.Ip6Addr == "" {
		return nil, errors.Wrap(errors.ErrEmpty, "Nic no ip or ip6 address")
	}

	routes4 := make([]dhcp.SRouteInfo, 0)
	routes6 := make([]dhcp.SRouteInfo, 0)
	for _, route := range n.Routes {
		routeInfo, err := dhcp.ParseRouteInfo([]string{route[0], route[1]})
		if err != nil {
			return nil, errors.Wrapf(err, "Parse route %s error: %q", route, err)
		}
		if regutils.MatchCIDR6(route[0]) {
			routes6 = append(routes6, *routeInfo)
		} else {
			routes4 = append(routes4, *routeInfo)
		}
	}

	if n.IsDefault == nil || *n.IsDefault {
		if n.Gateway != "" && !strings.HasPrefix(strings.ToLower(osName), "win") {
			routes4 = append(routes4, dhcp.SRouteInfo{
				Prefix:    net.ParseIP("0.0.0.0"),
				PrefixLen: 0,
				Gateway:   net.ParseIP(n.Gateway),
			})
		}
		if n.Gateway6 != "" {
			routes6 = append(routes6, dhcp.SRouteInfo{
				Prefix:    net.ParseIP("::"),
				PrefixLen: 0,
				Gateway:   net.ParseIP(n.Gateway6),
			})
		}
	}

	conf := &dhcp.ResponseConfig{
		InterfaceMac: serverMac,

		ServerIP:    serverIP,
		Domain:      n.Domain,
		OsName:      osName,
		Hostname:    strings.ToLower(hostName),
		Routes:      routes4,
		Routes6:     routes6,
		LeaseTime:   time.Duration(o.Options.DhcpLeaseTime) * time.Second,
		RenewalTime: time.Duration(o.Options.DhcpRenewalTime) * time.Second,
	}

	if n.IpAddr != "" {
		ipAddr, err := netutils.NewIPV4Addr(n.IpAddr)
		if err != nil {
			return nil, errors.Wrapf(err, "Parse IP address error: %q", n.IpAddr)
		}

		subnetMask := net.ParseIP(netutils.Masklen2Mask(n.Masklen).String())

		conf.ClientIP = net.ParseIP(ipAddr.String())
		conf.SubnetMask = subnetMask
		conf.BroadcastAddr = net.ParseIP(ipAddr.BroadcastAddr(n.Masklen).String())
		conf.Gateway = net.ParseIP(n.Gateway)
	}
	if n.Ip6Addr != "" {
		ipAddr6, err := netutils.NewIPV6Addr(n.Ip6Addr)
		if err != nil {
			return nil, errors.Wrapf(err, "Parse IPv6 address error: %q", n.Ip6Addr)
		}

		conf.ClientIP6 = net.ParseIP(ipAddr6.String())
		conf.PrefixLen6 = n.Masklen6
		conf.Gateway6 = net.ParseIP(n.Gateway6)
	}

	if len(n.Dns) > 0 {
		conf.DNSServers = make([]net.IP, 0)
		conf.DNSServers6 = make([]net.IP, 0)
		for _, dns := range strings.Split(n.Dns, ",") {
			if regutils.MatchIP4Addr(dns) {
				conf.DNSServers = append(conf.DNSServers, net.ParseIP(dns))
			} else if regutils.MatchIP6Addr(dns) {
				conf.DNSServers6 = append(conf.DNSServers6, net.ParseIP(dns))
			}
		}
	}

	if len(n.Ntp) > 0 {
		conf.NTPServers = make([]net.IP, 0)
		conf.NTPServers6 = make([]net.IP, 0)
		for _, ntp := range strings.Split(n.Ntp, ",") {
			if regutils.MatchIP4Addr(ntp) {
				conf.NTPServers = append(conf.NTPServers, net.ParseIP(ntp))
			} else if regutils.MatchIP6Addr(ntp) {
				conf.NTPServers6 = append(conf.NTPServers6, net.ParseIP(ntp))
			} else if regutils.MatchDomainName(ntp) {
				ntpAddrs, _ := net.LookupHost(ntp)
				for _, ntpAddr := range ntpAddrs {
					if regutils.MatchIP4Addr(ntpAddr) {
						conf.NTPServers = append(conf.NTPServers, net.ParseIP(ntpAddr))
					} else if regutils.MatchIP6Addr(ntpAddr) {
						conf.NTPServers6 = append(conf.NTPServers6, net.ParseIP(ntpAddr))
					}
				}
			}
		}
	}

	if n.Mtu > 0 {
		conf.MTU = uint16(n.Mtu)
	}

	if isPxe {
		conf.BootServer = serverIP.String()
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
