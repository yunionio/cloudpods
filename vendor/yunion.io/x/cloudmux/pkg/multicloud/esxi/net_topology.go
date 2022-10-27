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
	"context"
	"sort"

	"github.com/vmware/govmomi/vim25/mo"
	"golang.org/x/sync/errgroup"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
)

func (cli *SESXiClient) HostVmIPsInDc(ctx context.Context, dc *SDatacenter) (SNetworkInfo, error) {
	var hosts []mo.HostSystem
	err := cli.scanMObjects(dc.object.Entity().Self, HOST_SYSTEM_PROPS, &hosts)
	if err != nil {
		return SNetworkInfo{}, errors.Wrap(err, "scanMObjects")
	}
	return cli.hostVMIPs(ctx, hosts)
}

func (cli *SESXiClient) HostVmIPsInCluster(ctx context.Context, cluster *SCluster) (SNetworkInfo, error) {
	var hosts []mo.HostSystem
	err := cli.scanMObjects(cluster.object.Entity().Self, HOST_SYSTEM_PROPS, &hosts)
	if err != nil {
		return SNetworkInfo{}, errors.Wrap(err, "scanMObjects")
	}
	return cli.hostVMIPs(ctx, hosts)
}

type SNetworkInfo struct {
	SNetworkInfoBase

	HostIps map[string]netutils.IPV4Addr
	VlanIps map[int32][]netutils.IPV4Addr
}

func (s *SNetworkInfo) Insert(proc SIPProc, ip netutils.IPV4Addr) {
	s.SNetworkInfoBase.Insert(proc, ip)
	s.VlanIps[proc.VlanId] = append(s.VlanIps[proc.VlanId], ip)
}

func (cli *SESXiClient) hostVMIPs(ctx context.Context, hosts []mo.HostSystem) (SNetworkInfo, error) {
	ret := SNetworkInfo{}
	dvpgMap, err := cli.getDVPGMap()
	if err != nil {
		return ret, errors.Wrap(err, "unable to get dvpgKeyVlanMap")
	}
	group, ctx := errgroup.WithContext(ctx)
	collection := make([]*SNetworkInfo, len(hosts))
	for i := range hosts {
		j := i
		group.Go(func() error {
			nInfo, err := cli.vmIPs(&hosts[j], dvpgMap)
			if err != nil {
				return err
			}
			collection[j] = nInfo.(*SNetworkInfo)
			return nil
		})
	}

	hostIps := make(map[string]netutils.IPV4Addr, len(hosts))
	for i := range hosts {
		// find ip
		host := &SHost{SManagedObject: newManagedObject(cli, &hosts[i], nil)}
		ipStr := host.GetAccessIp()
		ip, err := netutils.NewIPV4Addr(ipStr)
		if err != nil {
			return ret, errors.Wrapf(err, "invalid host ip %q", ipStr)
		}
		hostIps[host.GetName()] = ip
	}
	err = group.Wait()
	if err != nil {
		return ret, err
	}
	// length
	if len(collection) == 0 {
		ret.HostIps = hostIps
		return ret, nil
	}
	ni := cli.mergeNetworInfo(collection)
	ni.HostIps = hostIps
	return ni, nil
}

func (cli *SESXiClient) mergeNetworInfo(nInfos []*SNetworkInfo) SNetworkInfo {
	var vmsLen, vlanIpLen, ipPoolLen int
	for i := range nInfos {
		vmsLen += len(nInfos[i].VMs)
		vlanIpLen += len(nInfos[i].VlanIps)
		ipPoolLen += nInfos[i].IPPool.Len()
	}
	ret := SNetworkInfo{
		SNetworkInfoBase: SNetworkInfoBase{
			IPPool: NewIPPool(ipPoolLen),
			VMs:    make([]SSimpleVM, 0, vmsLen),
		},
		VlanIps: make(map[int32][]netutils.IPV4Addr, vlanIpLen),
	}
	for i := range nInfos {
		ret.VMs = append(ret.VMs, nInfos[i].VMs...)
		for vlan, ips := range nInfos[i].VlanIps {
			ret.VlanIps[vlan] = append(ret.VlanIps[vlan], ips...)
		}
		ret.IPPool.Merge(&nInfos[i].IPPool)
	}
	for _, ips := range ret.VlanIps {
		sort.Slice(ips, func(i, j int) bool {
			return ips[i] < ips[j]
		})
	}
	return ret
}

func (cli *SESXiClient) HostVmIPs(ctx context.Context) (SNetworkInfo, error) {
	var hosts []mo.HostSystem
	err := cli.scanAllMObjects(SIMPLE_HOST_PROPS, &hosts)
	if err != nil {
		return SNetworkInfo{}, errors.Wrap(err, "scanAllMObjects")
	}
	return cli.hostVMIPs(ctx, hosts)
}

func (cli *SESXiClient) vmIPs(host *mo.HostSystem, vpgMap sVPGMap) (INetworkInfo, error) {
	nInfo := NewNetworkInfo(false, len(host.Vm))
	if len(host.Vm) == 0 {
		return nInfo, nil
	}
	var vms []mo.VirtualMachine
	err := cli.references2Objects(host.Vm, SIMPLE_VM_PROPS, &vms)
	if err != nil {
		return nInfo, errors.Wrap(err, "references2Objects")
	}
	for i := range vms {
		vm := vms[i]
		if vm.Config == nil || vm.Config.Template {
			continue
		}
		if vm.Guest == nil {
			continue
		}
		macVlanMap, err := cli.macVlanMap(host, &vm, vpgMap)
		if err != nil {
			return nInfo, errors.Wrap(err, "macVlanMap")
		}
		guestIps := make([]SIPVlan, 0)
		for _, net := range vm.Guest.Net {
			if len(net.Network) == 0 {
				continue
			}
			mac := net.MacAddress
			for _, ip := range net.IpAddress {
				if !regutils.MatchIP4Addr(ip) {
					continue
				}
				if !vmIPV4Filter.Contains(ip) {
					continue
				}
				ipaddr, _ := netutils.NewIPV4Addr(ip)
				if netutils.IsLinkLocal(ipaddr) {
					continue
				}
				proc := macVlanMap[mac]
				guestIps = append(guestIps, SIPVlan{
					IP:     ipaddr,
					VlanId: proc.VlanId,
				})
				nInfo.Insert(proc, ipaddr)
				break
			}
		}
		nInfo.AppendVMs(vm.Name, guestIps)
	}
	return nInfo, nil
}
