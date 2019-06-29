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
	"github.com/vmware/govmomi/vim25/types"
)

type IVMNetwork interface {
	GetId() string
	GetName() string
	GetVlanId() int32
	GetNumPorts() int32
	GetActivePorts() []string
	GetType() string
}

const (
	NET_TYPE_NETWORK     = "network"
	NET_TYPE_DVPORTGROUP = "dvportgroup"

	VLAN_MODE_NONE  = "none"
	VLAN_MODE_VLAN  = "vlan"
	VLAN_MODE_PVLAN = "pvlan"
	VLAN_MODE_TRUNK = "trunk"
)

var NETWORK_PROPS = []string{"name", "parent", "summary", "host", "vm"}
var DVPORTGROUP_PROPS = []string{"name", "parent", "summary", "host", "vm", "config"}

type SNetwork struct {
	SManagedObject
}

type SDistributedVirtualPortgroup struct {
	SManagedObject
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

func (net *SNetwork) GetVlanId() int32 {
	return -1
}

func (net *SNetwork) GetVlanMode() string {
	return VLAN_MODE_NONE
}

func (net *SNetwork) GetNumPorts() int32 {
	return -1
}

func (net *SNetwork) GetActivePorts() []string {
	return nil
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
