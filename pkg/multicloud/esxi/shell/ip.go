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

package shell

import (
	"sort"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/multicloud/esxi"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type IPListOption struct {
		Host bool   `help:"List only host ips"`
		Vm   bool   `help:"List only vm ips"`
		Cidr string `help:"Filter ips in this cidr, such as 192.168.10.0/24"`
	}
	shellutils.R(&IPListOption{}, "ip-list", "List all ip", func(cli *esxi.SESXiClient, args *IPListOption) error {
		host, vm := args.Host, args.Vm
		if !host && !vm {
			host, vm = true, true
		}
		var cidr netutils.IPV4Prefix
		var err error
		if len(args.Cidr) > 0 {
			cidr, err = netutils.NewIPV4Prefix(args.Cidr)
			if err != nil {
				return errors.Wrap(err, "unable to parse cidr")
			}
		}
		list := make([]IPDetails, 0, 5)
		if host {
			hostIps, _, err := cli.AllHostIP()
			if err != nil {
				return errors.Wrap(err, "AllHostIP")
			}
			for n, ip := range hostIps {
				ipaddr, err := netutils.NewIPV4Addr(ip)
				if err != nil {
					return errors.Wrapf(err, "NewIPV4Addr for ip %s", ip)
				}
				if len(args.Cidr) > 0 && !cidr.Contains(ipaddr) {
					continue
				}
				list = append(list, IPDetails{
					Name:   n,
					Type:   "host",
					Ip:     ip,
					Ipaddr: ipaddr,
				})
			}
		}
		if vm {
			vmipinfos, err := cli.VMIP2()
			if err != nil {
				return errors.Wrap(err, "VMIP2")
			}
			//split
			for i := range vmipinfos {
				for _, macip := range vmipinfos[i].MacIps {
					mac := macip.Mac
					if len(macip.IPs) == 0 {
						list = append(list, IPDetails{
							Name:       vmipinfos[i].Name,
							Mac:        mac,
							Uuid:       vmipinfos[i].Uuid,
							Moid:       vmipinfos[i].Moid,
							PowerState: vmipinfos[i].PowerState,
							Type:       "vm",
							Ip:         "",
							Ipaddr:     0,
						})
						continue
					}
					ip := macip.IPs[0]
					ipaddr, err := netutils.NewIPV4Addr(ip)
					if err != nil {
						return errors.Wrapf(err, "NewIPV4Addr for ip %s", ip)
					}
					if len(args.Cidr) > 0 && !cidr.Contains(ipaddr) {
						continue
					}

					list = append(list, IPDetails{
						Name:       vmipinfos[i].Name,
						Mac:        mac,
						Uuid:       vmipinfos[i].Uuid,
						Moid:       vmipinfos[i].Moid,
						PowerState: vmipinfos[i].PowerState,
						Type:       "vm",
						Ip:         ip,
						Ipaddr:     ipaddr,
					})
				}
			}

		}
		sort.Slice(list, func(i, j int) bool {
			return list[i].Ipaddr < list[j].Ipaddr
		})
		printList(list, []string{"name", "type", "ip", "mac", "Power_State", "uuid", "moid"})
		return nil
	})
}

type IPDetails struct {
	Moid       string
	PowerState string
	Uuid       string
	Name       string
	Type       string
	Ip         string
	Mac        string
	Ipaddr     netutils.IPV4Addr
}

func (i IPDetails) GetMoid() string {
	return i.Moid
}
func (i IPDetails) GetPowerState() string {
	return i.PowerState
}
func (i IPDetails) GetUuid() string {
	return i.Uuid
}
func (i IPDetails) GetMac() string {
	return i.Mac
}
func (i IPDetails) GetName() string {
	return i.Name
}

func (i IPDetails) GetType() string {
	return i.Type
}

func (i IPDetails) GetIp() string {
	return i.Ip
}
