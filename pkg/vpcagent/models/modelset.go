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

package models

import (
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	mcclient_modulebase "yunion.io/x/onecloud/pkg/mcclient/modulebase"
	mcclient_modules "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/vpcagent/apihelper"
)

type Vpcs map[string]*Vpc
type Networks map[string]*Network
type Guests map[string]*Guest
type Hosts map[string]*Host
type Guestnetworks map[string]*Guestnetwork // key: guestId/ifname

func (set Vpcs) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.Vpcs
}

func (set Vpcs) NewModel() db.IModel {
	return &Vpc{}
}

func (set Vpcs) AddModel(i db.IModel) {
	m := i.(*Vpc)
	if m.Id == compute.DEFAULT_VPC_ID {
		return
	}
	set[m.Id] = m
}

func (set Vpcs) Copy() apihelper.IModelSet {
	setCopy := Vpcs{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (ms Vpcs) joinNetworks(subEntries Networks) bool {
	for _, m := range ms {
		m.Networks = Networks{}
	}
	correct := true
	for subId, subEntry := range subEntries {
		id := subEntry.VpcId
		if id == compute.DEFAULT_VPC_ID {
			continue
		}
		m, ok := ms[id]
		if !ok {
			log.Warningf("network %s(%s): vpc id %s not found",
				subEntry.Name, subEntry.Id, id)
			correct = false
			continue
		}
		if _, ok := m.Networks[subId]; ok {
			log.Warningf("network %s(%s): already joined",
				subEntry.Name, subEntry.Id)
			continue
		}
		subEntry.Vpc = m
		m.Networks[subId] = subEntry
	}
	return correct
}

func (set Guests) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.Servers
}

func (set Guests) NewModel() db.IModel {
	return &Guest{}
}

func (set Guests) AddModel(i db.IModel) {
	m := i.(*Guest)
	set[m.Id] = m
}

func (set Guests) Copy() apihelper.IModelSet {
	setCopy := Guests{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set Guests) joinHosts(subEntries Hosts) bool {
	correct := true
	for gId, g := range set {
		hId := g.HostId
		if hId == "" {
			// This is possible at guest creation time when host_id
			// is not decided yet
			continue
		}
		if g.ExternalId != "" {
			// It was observed that we may receive pubcloud guests
			// with host_id set from region API
			continue
		}
		h, ok := subEntries[hId]
		if !ok {
			log.Warningf("guest %s(%s): host id %s not found",
				gId, g.Name, hId)
			delete(set, gId)
			continue
		}
		g.Host = h
	}
	return correct
}

func (set Hosts) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.Hosts
}

func (set Hosts) NewModel() db.IModel {
	return &Host{}
}

func (set Hosts) AddModel(i db.IModel) {
	m := i.(*Host)
	set[m.Id] = m
}

func (set Hosts) Copy() apihelper.IModelSet {
	setCopy := Hosts{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set Networks) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.Networks
}

func (set Networks) NewModel() db.IModel {
	return &Network{}
}

func (set Networks) AddModel(i db.IModel) {
	m := i.(*Network)
	set[m.Id] = m
}

func (set Networks) Copy() apihelper.IModelSet {
	setCopy := Networks{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (ms Networks) joinGuestnetworks(subEntries Guestnetworks) bool {
	for _, m := range ms {
		m.Guestnetworks = Guestnetworks{}
	}
	for _, subEntry := range subEntries {
		netId := subEntry.NetworkId
		m, ok := ms[netId]
		if !ok {
			// this can happen when this guestnetwork is just a
			// stub for external/managed guests and "ms" was
			// already filtered out by conditions like
			// external_id.isnullorempty, etc.
			continue
		}
		subId := subEntry.GuestId + "/" + subEntry.Ifname
		if _, ok := m.Guestnetworks[subId]; ok {
			log.Warningf("guestnetwork net/guest/ifname %s/%s already joined", netId, subId)
			continue
		}
		subEntry.Network = m
		m.Guestnetworks[subId] = subEntry
	}
	return true
}

func (set Guestnetworks) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.Servernetworks
}

func (set Guestnetworks) NewModel() db.IModel {
	return &Guestnetwork{}
}

func (set Guestnetworks) AddModel(i db.IModel) {
	m := i.(*Guestnetwork)
	set[m.GuestId+"/"+m.Ifname] = m
}

func (set Guestnetworks) Copy() apihelper.IModelSet {
	setCopy := Guestnetworks{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set Guestnetworks) joinGuests(subEntries Guests) bool {
	for _, gn := range set {
		gId := gn.GuestId
		g, ok := subEntries[gId]
		if !ok {
			if gn.Network != nil && gn.Network.Vpc != nil {
				// Only log info instead of error because the
				// guest could be in pending_deleted state
				log.Infof("guestnetwork (net:%s,ip:%s) guest id %s not found",
					gn.NetworkId, gn.IpAddr, gId)
			}
			continue
		}
		gn.Guest = g
	}
	return true
}
