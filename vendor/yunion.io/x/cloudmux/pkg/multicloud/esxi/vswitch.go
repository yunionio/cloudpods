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
	"regexp"

	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/pkg/errors"
)

var DVS_PROPS = []string{"name"}

type IVirtualSwitch interface {
	FindNetworkByVlanID(vlanID int32) (IVMNetwork, error)
}

type SVirtualSwitch struct {
	Host              *SHost
	HostVirtualSwitch types.HostVirtualSwitch
}

func (vs *SVirtualSwitch) FindNetworkByVlanID(vlanID int32) (IVMNetwork, error) {
	vsKey := vs.HostVirtualSwitch.Key
	pgs := vs.Host.getHostSystem().Config.Network.Portgroup
	var networkName string
	for i := range pgs {
		if pgs[i].Vswitch != vsKey {
			continue
		}
		if vlanEqual(pgs[i].Spec.VlanId, vlanID) {
			networkName = pgs[i].Spec.Name
			break
		}
	}
	if len(networkName) == 0 {
		return nil, nil
	}
	networks, err := vs.Host.GetNetworks()
	if err != nil {
		return nil, errors.Wrapf(err, "can't get networks of host %q", vs.Host.GetGlobalId())
	}
	for i := range networks {
		if networks[i].GetName() == networkName {
			return networks[i], nil
		}
	}
	return nil, nil
}

type SDistributedVirtualSwitch struct {
	Host                     *SHost
	DistributedVirtualSwitch mo.DistributedVirtualSwitch
}

func (vs *SDistributedVirtualSwitch) FindNetworkByVlanID(vlanID int32) (IVMNetwork, error) {
	var modvpgs []mo.DistributedVirtualPortgroup
	filter := property.Match{}
	filter["config.distributedVirtualSwitch"] = vs.DistributedVirtualSwitch.Self
	err := vs.Host.manager.scanMObjectsWithFilter(vs.Host.datacenter.object.Entity().Self, DVPORTGROUP_PROPS, &modvpgs, filter)
	if err != nil {
		return nil, errors.Wrapf(err, "can't fetch portgroup of DistributedVirtualSwitch %q", vs.DistributedVirtualSwitch.Name)
	}
	dvpgs := make([]*SDistributedVirtualPortgroup, 0, len(modvpgs))
	for i := range modvpgs {
		if modvpgs[i].Config.Uplink != nil && *modvpgs[i].Config.Uplink {
			log.Infof("dvpg %s is uplink, so skip", modvpgs[i].Name)
			continue
		}
		dvpgs = append(dvpgs, NewDistributedVirtualPortgroup(vs.Host.manager, &modvpgs[i], nil))
	}
	for i := range dvpgs {
		if vlanEqual(dvpgs[i].GetVlanId(), vlanID) {
			return dvpgs[i], nil
		}
	}
	return nil, nil
}

func vlanEqual(v1, v2 int32) bool {
	if v1 <= 1 && v2 <= 1 {
		return true
	}
	return v1 == v2
}

var (
	vsBridgeRegex  = regexp.MustCompile(`^(host-\d+)/(.*)`)
	dvsBridgeRegex = regexp.MustCompile(`^dvs-\d+$`)
)

// config.distributedVirtualSwitch

func findVirtualSwitch(host *SHost, bridge string) (IVirtualSwitch, error) {
	group := vsBridgeRegex.FindStringSubmatch(bridge)
	oHost := host.getHostSystem()
	if len(group) > 0 {
		// vswitch
		vsName := group[2]
		for _, vs := range oHost.Config.Network.Vswitch {
			if vs.Name != vsName {
				continue
			}
			return &SVirtualSwitch{
				Host:              host,
				HostVirtualSwitch: vs,
			}, nil
		}
		return nil, nil
	}

	// distributed vswitch
	if !dvsBridgeRegex.MatchString(bridge) {
		return nil, nil
	}
	objRef := types.ManagedObjectReference{
		Type:  "VmwareDistributedVirtualSwitch",
		Value: bridge,
	}
	var dvs mo.DistributedVirtualSwitch
	err := host.manager.reference2Object(objRef, DVS_PROPS, &dvs)
	if err != nil {
		return nil, errors.Wrapf(err, "can't fetch DistributedVirtualSwitch %q", objRef.String())
	}
	return &SDistributedVirtualSwitch{
		Host:                     host,
		DistributedVirtualSwitch: dvs,
	}, nil
}
