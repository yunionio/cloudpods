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
	"fmt"
	"sort"
	"sync"

	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
)

var (
	SIMPLE_HOST_PROPS    = []string{"name", "config.network", "vm"}
	SIMPLE_VM_PROPS      = []string{"name", "guest.net", "config.template", "config.hardware.device"}
	SIMPLE_DVPG_PROPS    = []string{"key", "config.defaultPortConfig", "config.distributedVirtualSwitch"}
	SIMPLE_DVS_PROPS     = []string{"name", "config"}
	SIMPLE_NETWORK_PROPS = []string{"name"}
)

type SIPVlan struct {
	IP     netutils.IPV4Addr
	VlanId int32
}

type SSimpleVM struct {
	Name    string
	IPVlans []SIPVlan
}

type SSimpleHostDev struct {
	Name string
	Id   string
	Mac  string
}

func (cli *SESXiClient) scanAllDvPortgroups() ([]*SDistributedVirtualPortgroup, error) {
	var modvpgs []mo.DistributedVirtualPortgroup
	err := cli.scanAllMObjects(SIMPLE_DVPG_PROPS, &modvpgs)
	if err != nil {
		return nil, errors.Wrap(err, "scanAllMObjects")
	}
	dvpgs := make([]*SDistributedVirtualPortgroup, 0, len(modvpgs))
	for i := range modvpgs {
		dvpgs = append(dvpgs, NewDistributedVirtualPortgroup(cli, &modvpgs[i], nil))
	}
	return dvpgs, nil
}

func (cli *SESXiClient) scanAllNetworks() ([]mo.Network, error) {
	var monets []mo.Network
	err := cli.scanAllMObjects(SIMPLE_NETWORK_PROPS, &monets)
	if err != nil {
		return nil, errors.Wrap(err, "scanAllMObjects")
	}
	return monets, nil
}

/*func (cli *SESXiClient) networkName(refV string) (string, error) {
	if cli.networkQueryMap == nil {
		nets, err := cli.scanAllNetworks()
		if err != nil {
			return "", err
		}
		cli.networkQueryMap = &sync.Map{}
		for i := range nets {
			cli.networkQueryMap.Store(nets[i].Reference().Value, nets[i].Name)
		}
	}
	iter, ok := cli.networkQueryMap.Load(refV)
	if !ok {
		return "", nil
	}
	return iter.(string), nil
}*/

func (cli *SESXiClient) hostVMIPsPro(ctx context.Context, hosts []mo.HostSystem) (SNetworkInfoPro, error) {
	ret := SNetworkInfoPro{
		SNetworkInfoBase: SNetworkInfoBase{
			// VMs:    []SSimpleVM{},
			IPPool: SIPPool{},
		},
		HostIps: make(map[string][]netutils.IPV4Addr),
	}
	vsMap, err := cli.getVirtualSwitchs(hosts)
	if err != nil {
		return ret, errors.Wrap(err, "unable to getVirtualSwitchs")
	}
	ret.VsMap = vsMap
	/* dvpgMap, err := cli.getDVPGMap()
	if err != nil {
		return ret, errors.Wrap(err, "unable to get dvpgKeyVlanMap")
	}
	group, ctx := errgroup.WithContext(ctx)
	collection := make([]*SNetworkInfoPro, len(hosts))
	for i := range hosts {
		j := i
		group.Go(func() error {
			nInfo, err := cli.vmIPsPro(&hosts[j], dvpgMap)
			if err != nil {
				return err
			}
			collection[j] = nInfo.(*SNetworkInfoPro)
			return nil
		})
	}

	err = group.Wait()
	if err != nil {
		return ret, err
	}
	cli.mergeNetworInfoPro(&ret, collection)
	*/
	ret.IPPool = NewIPPool(0)
	vsList := vsMap.List()
	for i := range vsList {
		vs := vsList[i]
		for hName, ips := range vs.HostIps {
			sort.Slice(ips, func(i, j int) bool {
				return ips[i] < ips[j]
			})
			for _, ip := range ips {
				ret.IPPool.Insert(ip, SIPProc{
					VlanId: 0,
					VSId:   vs.Id,
					IsHost: true,
				})
			}
			ret.HostIps[hName] = append(ret.HostIps[hName], ips...)
		}
	}
	return ret, nil
}

/*func (cli *SESXiClient) mergeNetworInfoPro(ret *SNetworkInfoPro, nInfos []*SNetworkInfoPro) {
	log.Infof("nInfos before mergeNetworInfoPro: %s", jsonutils.Marshal(nInfos))
	var vmsLen, ipPoolLen int
	for i := range nInfos {
		vmsLen += len(nInfos[i].VMs)
		ipPoolLen += nInfos[i].IPPool.Len()
	}
	ret.VMs = make([]SSimpleVM, 0, vmsLen)
	ret.IPPool = NewIPPool(ipPoolLen)
	for i := range nInfos {
		ret.VMs = append(ret.VMs, nInfos[i].VMs...)
		ret.IPPool.Merge(&nInfos[i].IPPool)
		ret.VsMap.Merge(&nInfos[i].VsMap)
	}
}*/

func (cli *SESXiClient) HostVmIPsPro(ctx context.Context) (SNetworkInfoPro, error) {
	var hosts []mo.HostSystem
	err := cli.scanAllMObjects(SIMPLE_HOST_PROPS, &hosts)
	if err != nil {
		return SNetworkInfoPro{}, errors.Wrap(err, "scanAllMObjects")
	}
	filtedHosts := make([]mo.HostSystem, 0, len(hosts))
	for i := range hosts {
		if hosts[i].Config == nil || hosts[i].Config.Network == nil {
			continue
		}
		filtedHosts = append(filtedHosts, hosts[i])
	}
	return cli.hostVMIPsPro(ctx, filtedHosts)
}

func vpgMapKey(prefix, key string) string {
	if len(prefix) == 0 {
		return key
	}
	return fmt.Sprintf("%s-%s", prefix, key)
}

/*func (cli *SESXiClient) macVlanMap(mohost *mo.HostSystem, movm *mo.VirtualMachine, dvpgMap sVPGMap) (map[string]SIPProc, error) {
	vpgMap := cli.getVPGMap(mohost)
	ret := make(map[string]SIPProc, 2)
	for _, device := range movm.Config.Hardware.Device {
		bcard, ok := device.(types.BaseVirtualEthernetCard)
		if !ok {
			continue
		}
		card := bcard.GetVirtualEthernetCard()
		mac := card.MacAddress
		switch bk := card.Backing.(type) {
		case *types.VirtualEthernetCardDistributedVirtualPortBackingInfo:
			key := vpgMapKey("", bk.Port.PortgroupKey)
			proc, ok := dvpgMap.Get(key)
			if !ok {
				log.Errorf("dvpg %s not found in key-vlanid map", key)
				continue
			}
			ret[mac] = proc
		case *types.VirtualEthernetCardNetworkBackingInfo:
			netName, err := cli.networkName(bk.Network.Value)
			if err != nil {
				return nil, errors.Wrapf(err, "cli.networkName of %q", bk.Network.Value)
			}
			key := vpgMapKey(mohost.Reference().Value, netName)
			proc, ok := vpgMap.Get(key)
			if !ok {
				log.Errorf("vpg %s not found in key-vlanid map", key)
				continue
			}
			ret[mac] = proc
		}
	}
	return ret, nil
}*/

type SNetworkInfoPro struct {
	SNetworkInfoBase

	HostIps map[string][]netutils.IPV4Addr
	VsMap   SVirtualSwitchMap
}

/*func (s *SNetworkInfoPro) Insert(proc SIPProc, ip netutils.IPV4Addr) {
	s.SNetworkInfoBase.Insert(proc, ip)
	s.VsMap.AddVlanIp(proc.VSId, proc.VlanId, ip)
}*/

type SNetworkInfoBase struct {
	// VMs    []SSimpleVM
	IPPool SIPPool
}

func (s *SNetworkInfoBase) Insert(proc SIPProc, ip netutils.IPV4Addr) {
	s.IPPool.Insert(ip, proc)
}

/* func (s *SNetworkInfoBase) AppendVMs(name string, IpVlans []SIPVlan) {
	s.VMs = append(s.VMs, SSimpleVM{name, IpVlans})
}*/

type INetworkInfo interface {
	Insert(SIPProc, netutils.IPV4Addr)
	// AppendVMs(name string, IpVlans []SIPVlan)
}

func NewNetworkInfo(dvs bool, size int) INetworkInfo {
	base := SNetworkInfoBase{
		// VMs:    make([]SSimpleVM, 0, size),
		IPPool: NewIPPool(int(size)),
	}
	if dvs {
		return &SNetworkInfoPro{
			SNetworkInfoBase: base,
			VsMap:            NewVirtualSwitchMap(),
		}
	}
	return &SNetworkInfo{
		// SNetworkInfoBase: base,
		HostIps: make(map[string]netutils.IPV4Addr),
		VlanIps: make(map[int32][]netutils.IPV4Addr),
	}
}

/*func (cli *SESXiClient) vmIPsPro(host *mo.HostSystem, vpgMap sVPGMap) (INetworkInfo, error) {
	nInfo := NewNetworkInfo(true, 0)
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
			return nInfo, err
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
}*/

type SVirtualSwitchSpec struct {
	Name        string
	Id          string
	Distributed bool
	Hosts       []SSimpleHostDev
	HostIps     map[string][]netutils.IPV4Addr
	// Vlans       map[int32][]netutils.IPV4Addr
}

func NewVirtualSwitch() *SVirtualSwitchSpec {
	return &SVirtualSwitchSpec{
		HostIps: make(map[string][]netutils.IPV4Addr),
		// Vlans:   make(map[int32][]netutils.IPV4Addr),
	}
}

/*func (vs *SVirtualSwitchSpec) Merge(vsc *SVirtualSwitchSpec) {
	for vlan, ips := range vsc.Vlans {
		vs.Vlans[vlan] = insert(vs.Vlans[vlan], ips)
	}
}*/

func insert(base []netutils.IPV4Addr, add []netutils.IPV4Addr) []netutils.IPV4Addr {
	for _, ip := range add {
		index := sort.Search(len(base), func(n int) bool {
			return base[n] > ip
		})
		base = append(base, 0)
		base = append(base[:index+1], base[index:len(base)-1]...)
		base[index] = ip
	}
	return base
}

type SVirtualSwitchMap struct {
	vsList []SVirtualSwitchSpec
	vsMap  map[string]*SVirtualSwitchSpec
}

func NewVirtualSwitchMap(length ...int) SVirtualSwitchMap {
	initLen := 0
	if len(length) > 0 {
		initLen = length[0]
	}
	return SVirtualSwitchMap{
		vsList: make([]SVirtualSwitchSpec, 0, initLen),
		vsMap:  make(map[string]*SVirtualSwitchSpec, initLen),
	}
}

func (vsm *SVirtualSwitchMap) Insert(id string, vs SVirtualSwitchSpec) {
	_, ok := vsm.vsMap[id]
	if ok {
		// ovs.Merge(&vs)
		return
	}
	vsm.vsList = append(vsm.vsList, vs)
	vsm.vsMap[id] = &vsm.vsList[len(vsm.vsList)-1]
}

func (vms *SVirtualSwitchMap) Merge(ovsm *SVirtualSwitchMap) {
	for i := range ovsm.vsList {
		vs := ovsm.vsList[i]
		vms.Insert(vs.Id, vs)
	}
}

/*func (vsm *SVirtualSwitchMap) AddVlanIp(id string, vlanId int32, ip netutils.IPV4Addr) bool {
	vs, ok := vsm.vsMap[id]
	if !ok {
		vs = NewVirtualSwitch()
		vs.Id = id
		vsm.Insert(id, *vs)
		vs = vsm.vsMap[id]
	}
	vs.Vlans[vlanId] = append(vs.Vlans[vlanId], ip)
	return true
}*/

func (vsm *SVirtualSwitchMap) List() []SVirtualSwitchSpec {
	return vsm.vsList
}

func findIp(hs *mo.HostSystem, nic string) netutils.IPV4Addr {
	for i := range hs.Config.Network.Pnic {
		pnic := hs.Config.Network.Pnic[i]
		if pnic.Device != nic {
			continue
		}
		ip := pnic.Spec.Ip.IpAddress
		if len(ip) == 0 {
			return netutils.IPV4Addr(0)
		}
		if !regutils.MatchIP4Addr(ip) {
			return netutils.IPV4Addr(0)
		}
		ipaddr, _ := netutils.NewIPV4Addr(ip)
		if netutils.IsLinkLocal(ipaddr) {
			return netutils.IPV4Addr(0)
		}
		return ipaddr
	}
	return netutils.IPV4Addr(0)
}

func getNicDevice(vs *types.HostVirtualSwitch) []string {
	bridge := vs.Spec.Bridge
	switch b := bridge.(type) {
	case *types.HostVirtualSwitchAutoBridge:
		return b.ExcludedNicDevice
	case *types.HostVirtualSwitchBondBridge:
		return b.NicDevice
	case *types.HostVirtualSwitchSimpleBridge:
		return []string{b.NicDevice}
	default:
		return nil
	}
}

func (cli *SESXiClient) getVirtualSwitchs(hss []mo.HostSystem) (SVirtualSwitchMap, error) {
	nic2Ip := make(map[string]netutils.IPV4Addr, len(hss))
	nic2Mac := make(map[string]string, len(hss))
	for j := range hss {
		hs := hss[j]
		macToIp := make(map[string]string, len(hs.Config.Network.Vnic))
		for i := range hs.Config.Network.Vnic {
			vnic := hs.Config.Network.Vnic[i]
			if len(vnic.Spec.Ip.IpAddress) == 0 {
				continue
			}
			macToIp[vnic.Spec.Mac] = vnic.Spec.Ip.IpAddress
		}
		for i := range hs.Config.Network.ConsoleVnic {
			vnic := hs.Config.Network.Vnic[i]
			if len(vnic.Spec.Ip.IpAddress) == 0 {
				continue
			}
			macToIp[vnic.Spec.Mac] = vnic.Spec.Ip.IpAddress
		}
		for i := range hs.Config.Network.Pnic {
			pnic := hs.Config.Network.Pnic[i]
			ip := pnic.Spec.Ip.IpAddress
			nicKey := fmt.Sprintf("%s-%s", hs.Self.Value, pnic.Device)
			nic2Mac[nicKey] = pnic.Mac
			if len(ip) == 0 {
				ip = macToIp[pnic.Mac]
			}
			if len(ip) == 0 {
				log.Warningf("host %s pnic %s has no ipaddress", hs.Name, pnic.Device)
				continue
			}
			if !regutils.MatchIP4Addr(ip) {
				log.Warningf("host %s pnic %s ipaddress %q is not IPV4Addr", hs.Name, pnic.Device, ip)
				continue
			}
			ipaddr, _ := netutils.NewIPV4Addr(ip)
			if netutils.IsLinkLocal(ipaddr) {
				log.Warningf("host %s pnic %s ipaddress %q is link local addr", hs.Name, pnic.Device, ip)
				continue
			}
			nic2Ip[nicKey] = ipaddr
		}
	}
	value2name := make(map[string]string, len(hss))
	for i := range hss {
		value2name[hss[i].Self.Value] = hss[i].Name
	}
	// dvs
	var dvss []mo.DistributedVirtualSwitch
	err := cli.scanAllMObjects(SIMPLE_DVS_PROPS, &dvss)
	if err != nil {
		return SVirtualSwitchMap{}, err
	}
	ret := NewVirtualSwitchMap(len(dvss) + len(hss))
	for i := range dvss {
		dvs := dvss[i]
		vsId := dvs.Self.Value
		vsName := fmt.Sprintf("%s", dvs.Name)
		hosts := dvs.Config.GetDVSConfigInfo().Host
		vs := SVirtualSwitchSpec{
			Name:        vsName,
			Id:          vsId,
			Distributed: true,
			HostIps:     make(map[string][]netutils.IPV4Addr, len(hosts)),
			// Vlans:       make(map[int32][]netutils.IPV4Addr),
		}
		for _, host := range hosts {
			hostName := value2name[host.Config.Host.Value]
			vs.HostIps[hostName] = []netutils.IPV4Addr{}
			hostDev := SSimpleHostDev{
				Id:   host.Config.Host.Value,
				Name: hostName,
			}
			switch back := host.Config.Backing.(type) {
			case *types.DistributedVirtualSwitchHostMemberPnicBacking:
				for i := range back.PnicSpec {
					nicDevice := back.PnicSpec[i].PnicDevice
					nicKey := fmt.Sprintf("%s-%s", host.Config.Host.Value, nicDevice)
					hostDev.Mac = nic2Mac[nicKey]
					ip, ok := nic2Ip[nicKey]
					if ok {
						vs.HostIps[hostName] = append(vs.HostIps[hostName], ip)
					}
				}
			case *types.DistributedVirtualSwitchHostMemberBacking:
			}
			vs.Hosts = append(vs.Hosts, hostDev)
		}
		ret.Insert(vsId, vs)
	}
	// vs
	for i := range hss {
		hs := hss[i]
		if hs.Config == nil || hs.Config.Network == nil {
			continue
		}
		for i := range hs.Config.Network.Vswitch {
			ivs := hs.Config.Network.Vswitch[i]
			vsId := fmt.Sprintf("%s/%s", hs.Self.Value, ivs.Name)
			vsName := fmt.Sprintf("%s/%s", hs.Name, ivs.Name)
			vs := SVirtualSwitchSpec{
				Name:        vsName,
				Id:          vsId,
				Distributed: false,
				HostIps:     make(map[string][]netutils.IPV4Addr),
				// Vlans:       make(map[int32][]netutils.IPV4Addr),
			}
			hostDev := SSimpleHostDev{
				Id:   hs.Self.Value,
				Name: hs.Name,
			}
			for _, nic := range getNicDevice(&ivs) {
				nicKey := fmt.Sprintf("%s-%s", hs.Self.Value, nic)
				log.Infof("nic: %s, nicKey: %s", nic, nicKey)
				hostDev.Mac = nic2Mac[nicKey]
				ip, ok := nic2Ip[nicKey]
				if !ok {
					continue
				}
				vs.HostIps[hs.Name] = append(vs.HostIps[hs.Name], ip)
			}
			vs.Hosts = append(vs.Hosts, hostDev)
			ret.Insert(vsId, vs)
		}
	}
	return ret, nil
}

func (cli *SESXiClient) getVPGMap(mohost *mo.HostSystem) sVPGMap {
	sm := newVPGMap()
	if mohost.Config == nil || mohost.Config.Network == nil {
		return sm
	}
	for i := range mohost.Config.Network.Portgroup {
		ipg := mohost.Config.Network.Portgroup[i]
		key := vpgMapKey(mohost.Reference().Value, ipg.Spec.Name)
		vlan := ipg.Spec.VlanId
		vsId := fmt.Sprintf("%s/%s", mohost.Reference().Value, ipg.Spec.VswitchName)
		sm.Insert(key, SIPProc{
			VlanId: vlan,
			VSId:   vsId,
		})
	}
	return sm
}

func (cli *SESXiClient) getDVPGMap() (sVPGMap, error) {
	sm := newVPGMap()
	dvpgs, err := cli.scanAllDvPortgroups()
	if err != nil {
		return sm, err
	}
	for i := range dvpgs {
		modvpg := dvpgs[i].getMODVPortgroup()
		key := vpgMapKey("", modvpg.Key)
		vlanid := dvpgs[i].GetVlanId()
		vsId := modvpg.Config.DistributedVirtualSwitch.Value
		sm.Insert(key, SIPProc{
			VlanId: vlanid,
			VSId:   vsId,
		})
	}
	return sm, nil
}

func newVPGMap() sVPGMap {
	return sVPGMap{m: &sync.Map{}}
}

type sVPGMap struct {
	m *sync.Map
}

func (vm *sVPGMap) Insert(key string, proc SIPProc) {
	vm.m.Store(key, proc)
}

func (vm *sVPGMap) Get(key string) (SIPProc, bool) {
	v, ok := vm.m.Load(key)
	if !ok {
		return SIPProc{}, ok
	}
	r := v.(SIPProc)
	return r, ok
}

type SIPProc struct {
	VSId   string
	VlanId int32
	IsHost bool
}

type SIPPool struct {
	p map[netutils.IPV4Addr]SIPProc
}

func NewIPPool(size ...int) SIPPool {
	if len(size) > 0 {
		return SIPPool{p: make(map[netutils.IPV4Addr]SIPProc, size[0])}
	}
	return SIPPool{p: make(map[netutils.IPV4Addr]SIPProc)}
}

func (p *SIPPool) Has(ip netutils.IPV4Addr) bool {
	_, ok := p.p[ip]
	return ok
}

func (p *SIPPool) Get(ip netutils.IPV4Addr) (SIPProc, bool) {
	r, ok := p.p[ip]
	return r, ok
}

func (p *SIPPool) Insert(ip netutils.IPV4Addr, proc SIPProc) {
	p.p[ip] = proc
}

func (p *SIPPool) Merge(op *SIPPool) {
	for ip, proc := range op.p {
		if p.Has(ip) {
			continue
		}
		p.Insert(ip, proc)
	}
}

func (p *SIPPool) Len() int {
	return len(p.p)
}
