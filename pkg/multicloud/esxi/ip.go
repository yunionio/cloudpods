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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
)

var HOST_PROPS = []string{"name", "config.network", "vm"}

var VM_PROPS = []string{"name", "guest.net", "config.template"}

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
		if vm.Config.Template {
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

func (cli *SESXiClient) VMIP2() (map[string][]string, error) {
	var vms []mo.VirtualMachine
	err := cli.scanAllMObjects(VM_PROPS, &vms)
	if err != nil {
		return nil, errors.Wrap(err, "scanAllMObjects")
	}
	ret := make(map[string][]string, len(vms))
	for i := range vms {
		vm := vms[i]
		if vm.Config.Template {
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
