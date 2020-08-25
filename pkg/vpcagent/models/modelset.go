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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	mcclient_modulebase "yunion.io/x/onecloud/pkg/mcclient/modulebase"
	mcclient_modules "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/vpcagent/apihelper"
)

type (
	Vpcs               map[string]*Vpc
	Wires              map[string]*Wire
	Networks           map[string]*Network
	Guests             map[string]*Guest
	Hosts              map[string]*Host
	SecurityGroups     map[string]*SecurityGroup
	SecurityGroupRules map[string]*SecurityGroupRule
	Elasticips         map[string]*Elasticip

	Guestnetworks  map[string]*Guestnetwork  // key: guestId/ifname
	Guestsecgroups map[string]*Guestsecgroup // key: guestId/secgroupId

	DnsRecords map[string]*DnsRecord
)

func (set Vpcs) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.Vpcs
}

func (set Vpcs) NewModel() db.IModel {
	return &Vpc{}
}

func (set Vpcs) AddModel(i db.IModel) {
	m := i.(*Vpc)
	set[m.Id] = m
}

func (set Vpcs) Copy() apihelper.IModelSet {
	setCopy := Vpcs{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (ms Vpcs) joinWires(subEntries Wires) bool {
	correct := true
	for _, subEntry := range subEntries {
		vpcId := subEntry.VpcId
		m, ok := ms[vpcId]
		if !ok {
			log.Warningf("vpc_id %s of wire %s(%s) is not present", vpcId, subEntry.Name, subEntry.Id)
			correct = false
			continue
		}
		subEntry.Vpc = m
		m.Wire = subEntry
	}
	return correct
}

func (ms Vpcs) joinNetworks(subEntries Networks) bool {
	for _, m := range ms {
		m.Networks = Networks{}
	}
	correct := true
	for subId, subEntry := range subEntries {
		wire := subEntry.Wire
		if wire == nil {
			// let it go.  By the time the subnet has externalId or
			// managerId set, we will not receive updates from them
			// anymore
			log.Warningf("network %s(%s) has no wire", subEntry.Name, subEntry.Id)
			delete(subEntries, subId)
			continue
		}
		id := wire.VpcId
		m, ok := ms[id]
		if !ok {
			// let it go.  By the time the subnet has externalId or
			// managerId set, we will not receive updates from them
			// anymore
			log.Warningf("network %s(%s): vpc id %s not found",
				subEntry.Name, subEntry.Id, id)
			delete(subEntries, subId)
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

func (set Wires) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.Wires
}

func (set Wires) NewModel() db.IModel {
	return &Wire{}
}

func (set Wires) AddModel(i db.IModel) {
	m := i.(*Wire)
	set[m.Id] = m
}

func (set Wires) Copy() apihelper.IModelSet {
	setCopy := Wires{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set Wires) IncludeEmulated() bool {
	return true
}

func (ms Wires) joinNetworks(subEntries Networks) bool {
	correct := true
	for _, subEntry := range subEntries {
		wireId := subEntry.WireId
		m, ok := ms[wireId]
		if !ok {
			correct = false
			continue
		}
		subEntry.Wire = m
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

func (set Guests) initJoin() {
	for _, el := range set {
		el.SecurityGroups = SecurityGroups{}
	}
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

func (set Guests) joinSecurityGroups(subEntries SecurityGroups) bool {
	correct := true
	j := func(guest *Guest, fname, secgroupId string) (*SecurityGroup, bool) {
		if secgroupId == "" {
			return nil, true
		}
		secgroup, ok := subEntries[secgroupId]
		if !ok {
			log.Warningf("cannot find %s %s of guest %s(%s)",
				fname, secgroupId, guest.Name, guest.Id)
			return nil, false
		}
		guest.SecurityGroups[secgroupId] = secgroup
		return secgroup, true
	}
	for _, g := range set {
		adminSecgroup, c0 := j(g, "admin_secgrp_id", g.AdminSecgrpId)
		_, c1 := j(g, "secgrp_id", g.SecgrpId)
		g.AdminSecurityGroup = adminSecgroup
		if !(c0 && c1) {
			correct = false
		}
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

func (ms Networks) joinElasticips(subEntries Elasticips) bool {
	for _, m := range ms {
		m.Elasticips = Elasticips{}
	}
	correct := true
	for _, subEntry := range subEntries {
		netId := subEntry.NetworkId
		m, ok := ms[netId]
		if !ok {
			log.Warningf("eip %s(%s): network %s not found", subEntry.Name, subEntry.Id, netId)
			correct = false
			continue
		}
		if _, ok := m.Elasticips[subEntry.Id]; ok {
			log.Warningf("elasticip %s(%s) already joined", subEntry.Name, subEntry.Id)
			correct = false
			continue
		}
		subEntry.Network = m
		m.Elasticips[subEntry.Id] = subEntry
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

func (set Guestnetworks) joinElasticips(subEntries Elasticips) bool {
	correct := true
	for _, gn := range set {
		eipId := gn.EipId
		if eipId == "" {
			continue
		}
		eip, ok := subEntries[eipId]
		if !ok {
			log.Warningf("guestnetwork %s(%s): eip %s not found", gn.GuestId, gn.Ifname, eipId)
			correct = false
			continue
		}
		if eip.Guestnetwork != nil {
			if eip.Guestnetwork != gn {
				log.Errorf("eip %s associated to more than 1 guestnetwork: %s(%s), %s(%s)", eipId,
					eip.Guestnetwork.GuestId, eip.Guestnetwork.Ifname, gn.GuestId, gn.Ifname)
				correct = false
			}
			continue
		}
		eip.Guestnetwork = gn
		gn.Elasticip = eip
	}
	return correct
}

func (set SecurityGroups) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.SecGroups
}

func (set SecurityGroups) NewModel() db.IModel {
	return &SecurityGroup{}
}

func (set SecurityGroups) AddModel(i db.IModel) {
	m := i.(*SecurityGroup)
	set[m.Id] = m
}

func (set SecurityGroups) Copy() apihelper.IModelSet {
	setCopy := SecurityGroups{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (ms SecurityGroups) joinSecurityGroupRules(subEntries SecurityGroupRules) bool {
	for _, m := range ms {
		m.SecurityGroupRules = SecurityGroupRules{}
	}
	correct := true
	for subId, subEntry := range subEntries {
		id := subEntry.SecgroupId
		m, ok := ms[id]
		if !ok {
			log.Warningf("secgrouprule %s: secgroup %s not found",
				subEntry.Id, id)
			correct = false
			continue
		}
		if _, ok := m.SecurityGroupRules[subId]; ok {
			log.Warningf("secgrouprule %s: already joined", subEntry.Id)
			continue
		}
		subEntry.SecurityGroup = m
		m.SecurityGroupRules[subId] = subEntry
	}
	return correct
}

func (set SecurityGroupRules) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.SecGroupRules
}

func (set SecurityGroupRules) NewModel() db.IModel {
	return &SecurityGroupRule{}
}

func (set SecurityGroupRules) AddModel(i db.IModel) {
	m := i.(*SecurityGroupRule)
	set[m.Id] = m
}

func (set SecurityGroupRules) Copy() apihelper.IModelSet {
	setCopy := SecurityGroupRules{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set Guestsecgroups) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.Serversecgroups
}

func (set Guestsecgroups) NewModel() db.IModel {
	return &Guestsecgroup{}
}

func (set Guestsecgroups) AddModel(i db.IModel) {
	m := i.(*Guestsecgroup)
	set[m.ModelSetKey()] = m
}

func (set Guestsecgroups) Copy() apihelper.IModelSet {
	setCopy := Guestsecgroups{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set Guestsecgroups) joinSecurityGroups(subEntries SecurityGroups) bool {
	for _, el := range set {
		secgroupId := el.SecgroupId
		guestId := el.GuestId
		subEntry, ok := subEntries[secgroupId]
		if !ok {
			// This is possible if guestsecgroups is for external resources
			log.Infof("guestsecgroups cannot find secgroup %s for guest %s",
				secgroupId, guestId)
			continue
		}
		el.SecurityGroup = subEntry
	}
	return true
}

func (set Guestsecgroups) joinGuests(subEntries Guests) bool {
	for _, el := range set {
		secgroupId := el.SecgroupId
		guestId := el.GuestId
		subEntry, ok := subEntries[guestId]
		if !ok {
			// This is possible if guestsecgroups is for external resources
			log.Infof("guestsecgroups cannot find guest %s for secgroup %s",
				guestId, secgroupId)
			continue
		}
		el.Guest = subEntry
		subEntry.SecurityGroups[secgroupId] = el.SecurityGroup
	}
	return true
}

func (set Guestsecgroups) join(secgroups SecurityGroups, guests Guests) bool {
	// order matters as joinGuests() will need to refer to secgroup joined
	// by joinSecurityGroups()
	c0 := set.joinSecurityGroups(secgroups)
	c1 := set.joinGuests(guests)
	return c0 && c1
}

func (set Elasticips) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.Elasticips
}

func (set Elasticips) NewModel() db.IModel {
	return &Elasticip{}
}

func (set Elasticips) AddModel(i db.IModel) {
	m := i.(*Elasticip)
	set[m.Id] = m
}

func (set Elasticips) Copy() apihelper.IModelSet {
	setCopy := Elasticips{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set DnsRecords) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.DNSRecords
}

func (set DnsRecords) NewModel() db.IModel {
	return &DnsRecord{}
}

func (set DnsRecords) AddModel(i db.IModel) {
	m := i.(*DnsRecord)
	set[m.Id] = m
}

func (set DnsRecords) Copy() apihelper.IModelSet {
	setCopy := DnsRecords{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}
