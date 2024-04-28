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
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type SNetworkSummary struct {
	Name       string
	Accessible bool
	IpPoolName string
	IpPoolId   int32
}

type IVMNetwork interface {
	GetId() string
	GetName() string
	GetType() string

	GetVlanId() int32
	GetVswitchName() string

	Summary() SNetworkSummary
	SummaryText() string

	GetHosts() []types.ManagedObjectReference

	GetDatacenter() (*SDatacenter, error)

	GetPath() []string
}

const (
	NET_TYPE_NETWORK     = "network"
	NET_TYPE_DVPORTGROUP = "dvportgroup"

	VLAN_MODE_NONE  = "none"
	VLAN_MODE_VLAN  = "vlan"
	VLAN_MODE_PVLAN = "pvlan"
	VLAN_MODE_TRUNK = "trunk"
)

var NETWORK_PROPS = []string{"name", "parent", "host", "summary"}
var DVPORTGROUP_PROPS = []string{"name", "parent", "host", "summary", "config", "key"}

type SNetwork struct {
	SManagedObject
	// HostPortGroup types.HostPortGroup
}

type SDistributedVirtualPortgroup struct {
	SManagedObject
	// HostPortGroup types.HostPortGroup
}

func NewNetwork(manager *SESXiClient, net *mo.Network, dc *SDatacenter) *SNetwork {
	return &SNetwork{SManagedObject: newManagedObject(manager, net, dc)}
}

func NewDistributedVirtualPortgroup(manager *SESXiClient, net *mo.DistributedVirtualPortgroup, dc *SDatacenter) *SDistributedVirtualPortgroup {
	return &SDistributedVirtualPortgroup{SManagedObject: newManagedObject(manager, net, dc)}
}

func (net *SNetwork) getMONetwork() *mo.Network {
	return net.object.(*mo.Network)
}

func (net *SNetwork) GetName() string {
	return net.getMONetwork().Name
}

func (net *SNetwork) GetType() string {
	return NET_TYPE_NETWORK
}

func (net *SNetwork) Summary() SNetworkSummary {
	moNet := net.getMONetwork()
	summary := moNet.Summary.GetNetworkSummary()
	ret := SNetworkSummary{
		Accessible: summary.Accessible,
		Name:       summary.Name,
		IpPoolName: summary.IpPoolName,
	}
	if summary.IpPoolId != nil {
		ret.IpPoolId = *summary.IpPoolId
	}
	return ret
}

func (net *SNetwork) SummaryText() string {
	return jsonutils.Marshal(net.Summary()).String()
}

func (net *SNetwork) GetHosts() []types.ManagedObjectReference {
	return net.getMONetwork().Host
}

func (net *SNetwork) GetVlanId() int32 {
	return 1
}

func (net *SNetwork) GetVswitchName() string {
	return ""
}

func (net *SDistributedVirtualPortgroup) getMODVPortgroup() *mo.DistributedVirtualPortgroup {
	return net.object.(*mo.DistributedVirtualPortgroup)
}

func (net *SDistributedVirtualPortgroup) GetName() string {
	return net.getMODVPortgroup().Name
}

func (net *SDistributedVirtualPortgroup) GetType() string {
	return NET_TYPE_DVPORTGROUP
}

func (net *SDistributedVirtualPortgroup) Summary() SNetworkSummary {
	moNet := net.getMODVPortgroup()
	summary := moNet.Summary.GetNetworkSummary()
	ret := SNetworkSummary{
		Accessible: summary.Accessible,
		Name:       summary.Name,
		IpPoolName: summary.IpPoolName,
	}
	if summary.IpPoolId != nil {
		ret.IpPoolId = *summary.IpPoolId
	}
	return ret
}

func (net *SDistributedVirtualPortgroup) SummaryText() string {
	return jsonutils.Marshal(net.Summary()).String()
}

func (net *SDistributedVirtualPortgroup) GetVlanId() int32 {
	dvpg := net.getMODVPortgroup()
	switch conf := dvpg.Config.DefaultPortConfig.(type) {
	case *types.VMwareDVSPortSetting:
		switch vlanConf := conf.Vlan.(type) {
		case *types.VmwareDistributedVirtualSwitchTrunkVlanSpec:
			return -1
		case *types.VmwareDistributedVirtualSwitchPvlanSpec:
			return vlanConf.PvlanId
		case *types.VmwareDistributedVirtualSwitchVlanIdSpec:
			return vlanConf.VlanId
		}
	}
	return -1
}

func (net *SDistributedVirtualPortgroup) GetVlanMode() string {
	dvpg := net.getMODVPortgroup()
	switch conf := dvpg.Config.DefaultPortConfig.(type) {
	case *types.VMwareDVSPortSetting:
		switch conf.Vlan.(type) {
		case *types.VmwareDistributedVirtualSwitchTrunkVlanSpec:
			return VLAN_MODE_TRUNK
		case *types.VmwareDistributedVirtualSwitchPvlanSpec:
			return VLAN_MODE_PVLAN
		case *types.VmwareDistributedVirtualSwitchVlanIdSpec:
			return VLAN_MODE_VLAN
		}
	}
	return VLAN_MODE_NONE
}

func (net *SDistributedVirtualPortgroup) GetNumPorts() int32 {
	dvpg := net.getMODVPortgroup()
	return dvpg.Config.NumPorts
}

func (net *SDistributedVirtualPortgroup) GetActivePorts() []string {
	dvpg := net.getMODVPortgroup()
	switch conf := dvpg.Config.DefaultPortConfig.(type) {
	case *types.VMwareDVSPortSetting:
		return conf.UplinkTeamingPolicy.UplinkPortOrder.ActiveUplinkPort
	}
	return nil
}

func (net *SDistributedVirtualPortgroup) GetHosts() []types.ManagedObjectReference {
	return net.getMODVPortgroup().Host
}

func (net *SDistributedVirtualPortgroup) Uplink() bool {
	dvpg := net.getMODVPortgroup()
	return *dvpg.Config.Uplink
}

func (net *SDistributedVirtualPortgroup) GetDVSUuid() (string, error) {
	dvgp := net.getMODVPortgroup()
	var dvs mo.DistributedVirtualSwitch
	err := net.manager.reference2Object(*dvgp.Config.DistributedVirtualSwitch, []string{"uuid"}, &dvs)
	if err != nil {
		return "", errors.Wrap(err, "reference2Object")
	}
	return dvs.Uuid, nil
}

func (net *SDistributedVirtualPortgroup) GetVswitchName() string {
	dvgp := net.getMODVPortgroup()
	var dvs mo.DistributedVirtualSwitch
	err := net.manager.reference2Object(*dvgp.Config.DistributedVirtualSwitch, []string{"name"}, &dvs)
	if err != nil {
		log.Errorf("retrieve DistributedVirtualSwitch fail %s", err)
		return ""
	}
	return dvs.Name
}

func (net *SDistributedVirtualPortgroup) FindPort() (*types.DistributedVirtualPort, error) {
	dvgp := net.getMODVPortgroup()
	odvs := object.NewDistributedVirtualSwitch(net.manager.client.Client, *dvgp.Config.DistributedVirtualSwitch)
	var (
		False = false
		True  = true
	)
	criteria := types.DistributedVirtualSwitchPortCriteria{
		Connected:    &False,
		Inside:       &True,
		PortgroupKey: []string{dvgp.Key},
	}
	ports, err := odvs.FetchDVPorts(net.manager.context, &criteria)
	if err != nil {
		return nil, errors.Wrap(err, "object.DVS.FetchDVPorts")
	}
	if len(ports) > 0 {
		// release extra space timely
		return &ports[:1][0], nil
	}
	return nil, nil
}

func (net *SDistributedVirtualPortgroup) AddHostToDVS(host *SHost) (err error) {
	// get dvs
	dvgp := net.getMODVPortgroup()
	var s mo.DistributedVirtualSwitch
	err = net.manager.reference2Object(*dvgp.Config.DistributedVirtualSwitch, []string{"config"}, &s)
	if err != nil {
		return errors.Wrapf(err, "fail to convert reference to object")
	}
	moHost := host.getHostSystem()

	// check firstly
	for _, host := range s.Config.GetDVSConfigInfo().Host {
		if host.Config.Host.Value == moHost.Reference().Value {
			// host is already a member of dvs
			return nil
		}
	}
	config := &types.DVSConfigSpec{ConfigVersion: s.Config.GetDVSConfigInfo().ConfigVersion}
	backing := new(types.DistributedVirtualSwitchHostMemberPnicBacking)
	pnics := moHost.Config.Network.Pnic

	if len(pnics) == 0 {
		return errors.Error("no pnic in this host")
	}

	config.Host = []types.DistributedVirtualSwitchHostMemberConfigSpec{
		{
			Operation: "add",
			Host:      moHost.Reference(),
			Backing:   backing,
		},
	}
	dvs := object.NewDistributedVirtualSwitch(net.manager.client.Client, s.Reference())
	task, err := dvs.Reconfigure(net.manager.context, config)
	if err != nil {
		return errors.Wrapf(err, "dvs.Reconfigure")
	}
	err = task.Wait(net.manager.context)
	if err == nil {
		return nil
	}
	return err
}

func (dc *SDatacenter) resolveNetworks(netMobs []types.ManagedObjectReference) ([]IVMNetwork, error) {
	netPortMobs := make([]types.ManagedObjectReference, 0)
	netNetMobs := make([]types.ManagedObjectReference, 0)

	for i := range netMobs {
		// log.Debugf("type: %s value: %s", netMobs[i].Type, netMobs[i].Value)
		if netMobs[i].Type == "DistributedVirtualPortgroup" {
			netPortMobs = append(netPortMobs, netMobs[i])
		} else {
			netNetMobs = append(netNetMobs, netMobs[i])
		}
	}

	nets := make([]IVMNetwork, 0)

	if len(netPortMobs) > 0 {
		moPorts := make([]mo.DistributedVirtualPortgroup, 0)
		err := dc.manager.references2Objects(netPortMobs, DVPORTGROUP_PROPS, &moPorts)
		if err != nil {
			return nil, errors.Wrap(err, "references2Objects")
		}
		for i := range moPorts {
			port := NewDistributedVirtualPortgroup(dc.manager, &moPorts[i], dc)
			nets = append(nets, port)
		}
	}
	if len(netNetMobs) > 0 {
		moNets := make([]mo.Network, 0)
		err := dc.manager.references2Objects(netNetMobs, NETWORK_PROPS, &moNets)
		if err != nil {
			return nil, errors.Wrap(err, "references2Objects")
		}
		for i := range moNets {
			net := NewNetwork(dc.manager, &moNets[i], dc)
			nets = append(nets, net)
		}
	}

	return nets, nil
}

func (cli *SESXiClient) GetNetworks() ([]IVMNetwork, error) {
	dcs, err := cli.GetDatacenters()
	if err != nil {
		return nil, errors.Wrap(err, "GetDatacenters")
	}
	netMap := make(map[string]IVMNetwork)
	nets := make([]IVMNetwork, 0)
	for _, dc := range dcs {
		dcNets, err := dc.GetNetworks()
		if err != nil {
			return nil, errors.Wrap(err, "Datacenter.GetNetworks")
		}
		for i := range dcNets {
			net := dcNets[i]
			if _, ok := netMap[net.GetId()]; !ok {
				netMap[net.GetId()] = net
				nets = append(nets, net)
			} else {
				log.Errorf("network %s(%s) already exist in other datacenter", net.GetName(), net.GetId())
			}
		}
	}
	return nets, nil
}
