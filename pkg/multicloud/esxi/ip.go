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

package esxi

import (
	"github.com/vmware/govmomi/vim25/mo"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
)

type IPV4Range struct {
	iprange *netutils.IPV4AddrRange
}

func (i IPV4Range) Contains(ip string) bool {
	ipaddr, err := netutils.NewIPV4Addr(ip)
	if err != nil {
		log.Errorf("unable to parse ip %q: %v", ip, err)
		return false
	}
	if i.iprange == nil {
		return true
	}
	return i.iprange.Contains(ipaddr)
}

var vmIPV4Filter IPV4Range

func initVMIPV4Filter(cidr string) error {
	if len(cidr) == 0 {
		return nil
	}
	prefix, err := netutils.NewIPV4Prefix(cidr)
	if err != nil {
		return errors.Wrapf(err, "parse cidr %q", cidr)
	}
	irange := prefix.ToIPRange()
	vmIPV4Filter.iprange = &irange
	return nil
}

var HOST_PROPS = []string{"name", "config.network", "vm"}

var VM_PROPS = []string{"name", "guest.net", "config.template", "summary.config.uuid", "summary.runtime.powerState"}

func (cli *SESXiClient) AllHostIP() (map[string]string, []mo.HostSystem, error) {
	var hosts []mo.HostSystem
	err := cli.scanAllMObjects(HOST_PROPS, &hosts)
	if err != nil {
		return nil, nil, errors.Wrap(err, "scanAllMObjects")
	}
	ret := make(map[string]string, len(hosts))
	for i := range hosts {
		// find ip
		host := &SHost{SManagedObject: newManagedObject(cli, &hosts[i], nil)}
		ip := host.GetAccessIp()
		ret[host.GetName()] = ip
	}
	return ret, hosts, nil
}

func (cli *SESXiClient) VMIP(host mo.HostSystem) (map[string][]string, error) {
	var vms []mo.VirtualMachine
	err := cli.references2Objects(host.Vm, VM_PROPS, &vms)
	if err != nil {
		return nil, errors.Wrap(err, "references2Objects")
	}
	ret := make(map[string][]string, len(vms))
	for i := range vms {
		vm := vms[i]
		if vm.Config == nil || vm.Config.Template {
			continue
		}
		if vm.Guest == nil {
			continue
		}
		guestIps := make([]string, 0)
		for _, net := range vm.Guest.Net {
			for _, ip := range net.IpAddress {
				if regutils.MatchIP4Addr(ip) {
					guestIps = append(guestIps, ip)
				}
			}
		}
		ret[vm.Name] = guestIps
	}
	return ret, nil
}

type SVMIPInfo struct {
	Moid       string
	PowerState string
	Name       string
	Uuid       string
	MacIps     []SMacIps
}

type SMacIps struct {
	Mac string
	IPs []string
}

func (cli *SESXiClient) VMIP2() ([]SVMIPInfo, error) {
	var vms []mo.VirtualMachine
	err := cli.scanAllMObjects(VM_PROPS, &vms)
	if err != nil {
		return nil, errors.Wrap(err, "scanAllMObjects")
	}
	ret := make([]SVMIPInfo, 0, len(vms))
	for i := range vms {
		vm := vms[i]
		if vm.Config == nil || vm.Config.Template {
			continue
		}
		if vm.Guest == nil {
			continue
		}
		info := SVMIPInfo{
			Moid:       vm.Reference().Value,
			PowerState: string(vm.Summary.Runtime.PowerState),
			Uuid:       vm.Summary.Config.Uuid,
			Name:       vm.Name,
		}
		macIps := make([]SMacIps, 0, len(vm.Guest.Net))
		for _, net := range vm.Guest.Net {
			if len(net.Network) == 0 {
				continue
			}
			macip := SMacIps{
				Mac: net.MacAddress,
			}
			for _, ip := range net.IpAddress {
				if regutils.MatchIP4Addr(ip) {
					macip.IPs = append(macip.IPs, ip)
				}
			}
			macIps = append(macIps, macip)
		}
		info.MacIps = macIps
		ret = append(ret, info)
	}
	return ret, nil
}
