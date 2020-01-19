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
type Guestnetworks map[string]*Guestnetwork // guestId as key

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
	correct := true
	for _, subEntry := range subEntries {
		id := subEntry.NetworkId
		m, ok := ms[id]
		if !ok {
			log.Warningf("network id %s not found", id)
			correct = false
			continue
		}
		subId := subEntry.GuestId
		if _, ok := m.Guestnetworks[subId]; ok {
			log.Warningf("guestnetwork id %s/%s already joined", id, subId)
			continue
		}
		subEntry.Network = m
		m.Guestnetworks[subId] = subEntry
	}
	return correct
}

func (set Guestnetworks) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.Servernetworks
}

func (set Guestnetworks) NewModel() db.IModel {
	return &Guestnetwork{}
}

func (set Guestnetworks) AddModel(i db.IModel) {
	m := i.(*Guestnetwork)
	set[m.GuestId] = m
}

func (set Guestnetworks) Copy() apihelper.IModelSet {
	setCopy := Guestnetworks{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}
