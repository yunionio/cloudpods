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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apihelper"
	computeapis "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	mcclient_modulebase "yunion.io/x/onecloud/pkg/mcclient/modulebase"
	mcclient_modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
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
	NetworkAddresses   map[string]*NetworkAddress

	Guestnetworks  map[string]*Guestnetwork  // key: rowId
	Guestsecgroups map[string]*Guestsecgroup // key: guestId/secgroupId

	DnsZones   map[string]*DnsZone
	DnsRecords map[string]*DnsRecord

	RouteTables map[string]*RouteTable

	Groupguests   map[string]*Groupguest
	Groupnetworks map[string]*Groupnetwork
	Groups        map[string]*Group

	LoadbalancerNetworks  map[string]*LoadbalancerNetwork // key: networkId/loadbalancerId
	LoadbalancerListeners map[string]*LoadbalancerListener
	LoadbalancerAcls      map[string]*LoadbalancerAcl
)

func (set Vpcs) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.Vpcs
}

func (set Vpcs) DBModelManager() db.IModelManager {
	return models.VpcManager
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

func (set Vpcs) ModelParamFilter() jsonutils.JSONObject {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString("OneCloud"), "provider")
	return params
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

func (ms Vpcs) joinRouteTables(subEntries RouteTables) bool {
	correct := true
	for _, subEntry := range subEntries {
		vpcId := subEntry.VpcId
		m, ok := ms[vpcId]
		if !ok {
			log.Warningf("vpc_id %s of route table %s(%s) is not present", vpcId, subEntry.Name, subEntry.Id)
			correct = false
			continue
		}
		subEntry.Vpc = m
		if m.RouteTable != nil {
			log.Warningf("vpc %s(%s) has more than 1 route table available, skipping %s(%s)",
				m.Name, m.Id, subEntry.Name, subEntry.Id)
			correct = false
			continue
		}
		m.RouteTable = subEntry
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

func (set Wires) DBModelManager() db.IModelManager {
	return models.WireManager
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

func (set Wires) ModelParamFilter() jsonutils.JSONObject {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString("OneCloud"), "provider")
	return params
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

func (set Guests) DBModelManager() db.IModelManager {
	return models.GuestManager
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

func (set Guests) ModelParamFilter() jsonutils.JSONObject {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString("OneCloud"), "provider")
	return params
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
	j := func(guest *Guest, fname, secgroupId string, isAdmin bool) (*SecurityGroup, bool) {
		if secgroupId == "" {
			return nil, true
		}
		secgroup, ok := subEntries[secgroupId]
		if !ok {
			log.Warningf("cannot find %s %s of guest %s(%s)",
				fname, secgroupId, guest.Name, guest.Id)
			return nil, false
		}
		if !isAdmin {
			// do not save admin security group, store it in AdminSecurityGroup instead
			guest.SecurityGroups[secgroupId] = secgroup
		}
		return secgroup, true
	}
	for _, g := range set {
		adminSecgroup, c0 := j(g, "admin_secgrp_id", g.AdminSecgrpId, true)
		_, c1 := j(g, "secgrp_id", g.SecgrpId, false)
		g.AdminSecurityGroup = adminSecgroup
		if !(c0 && c1) {
			correct = false
		}
	}
	return correct
}

func (set Guests) joinGroupguests(groups Groups, groupGuests Groupguests) bool {
	for _, gg := range groupGuests {
		if guest, ok := set[gg.GuestId]; ok {
			if guest.Groups == nil {
				guest.Groups = Groups{}
			}
			if _, ok := groups[gg.GroupId]; !ok {
				grp := &Group{}
				grp.Id = gg.GroupId
				grp.Groupguests = Groupguests{}
				grp.Groupnetworks = Groupnetworks{}
				groups.AddModel(grp)
			}
			grp := groups[gg.GroupId]
			grp.Groupguests.AddModel(gg)
			guest.Groups.AddModel(grp)
			gg.Guest = guest
			gg.Group = grp
		}
	}
	return true
}

func (set Hosts) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.Hosts
}

func (set Hosts) DBModelManager() db.IModelManager {
	return models.HostManager
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

func (set Hosts) ModelParamFilter() jsonutils.JSONObject {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString("OneCloud"), "provider")
	return params
}

func (set Networks) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.Networks
}

func (set Networks) DBModelManager() db.IModelManager {
	return models.NetworkManager
}

func (set Networks) ModelParamFilter() jsonutils.JSONObject {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString("OneCloud"), "provider")
	return params
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
		m.Groupnetworks = Groupnetworks{}
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

func (ms Networks) joinLoadbalancerNetworks(subEntries LoadbalancerNetworks) bool {
	for _, m := range ms {
		m.LoadbalancerNetworks = LoadbalancerNetworks{}
	}
	for subEntryId, subEntry := range subEntries {
		netId := subEntry.NetworkId
		m, ok := ms[netId]
		if !ok {
			// this can happen for external loadbalancers
			log.Warningf("cannot find network %s for loadblancer %s",
				subEntry.NetworkId, subEntry.LoadbalancerId)
			// so that we can ignore these for later stages
			delete(subEntries, subEntryId)
			continue
		}
		subId := subEntry.NetworkId + "/" + subEntry.LoadbalancerId
		if _, ok := m.LoadbalancerNetworks[subId]; ok {
			log.Warningf("loadbalancernetwork net/lb %s already joined", subId)
			continue
		}
		subEntry.Network = m
		m.LoadbalancerNetworks[subId] = subEntry
	}
	return true
}

func (ms Networks) joinNetworkAddresses(subEntries NetworkAddresses) bool {
	correct := true
	for _, subEntry := range subEntries {
		netId := subEntry.NetworkId
		m, ok := ms[netId]
		if !ok {
			log.Errorf("cannot find network id %s of network address %s", netId, subEntry.Id)
			correct = false
			break
		}
		subEntry.Network = m
	}
	return correct
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

func (set Guestnetworks) DBModelManager() db.IModelManager {
	return models.GuestnetworkManager
}

func (set Guestnetworks) NewModel() db.IModel {
	return &Guestnetwork{}
}

func (set Guestnetworks) AddModel(i db.IModel) {
	m := i.(*Guestnetwork)
	k := fmt.Sprintf("%d", m.RowId)
	set[k] = m
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
				// guest could be in pending_deleted state or guest is not a KVM
				// log.Infof("guestnetwork (net:%s,ip:%s) guest id %s not found", gn.NetworkId, gn.IpAddr, gId)
			}
			continue
		}
		gn.Guest = g
		if g.Guestnetworks == nil {
			g.Guestnetworks = Guestnetworks{}
		}
		rowIdStr := fmt.Sprintf("%d", gn.RowId)
		g.Guestnetworks[rowIdStr] = gn
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

func (set Guestnetworks) joinNetworkAddresses(subEntries NetworkAddresses) bool {
	correct := true
	byRowId := map[string]*Guestnetwork{}
	for _, guestnetwork := range set {
		rowIdStr := fmt.Sprintf("%d", guestnetwork.RowId)
		byRowId[rowIdStr] = guestnetwork
	}
	for _, subEntry := range subEntries {
		if subEntry.ParentType != computeapis.NetworkAddressParentTypeGuestnetwork {
			continue
		}
		switch subEntry.Type {
		case computeapis.NetworkAddressTypeSubIP:
			parentId := subEntry.ParentId
			guestnetwork, ok := byRowId[parentId]
			if !ok {
				log.Errorf("cannot find guestnetwork row id %s of network address %s", parentId, subEntry.Id)
				correct = false
			}
			if guestnetwork.SubIPs == nil {
				guestnetwork.SubIPs = NetworkAddresses{}
			}
			guestnetwork.SubIPs[subEntry.Id] = subEntry
			subEntry.Guestnetwork = guestnetwork
		}
	}
	return correct
}

func (set NetworkAddresses) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.NetworkAddresses
}

func (set NetworkAddresses) DBModelManager() db.IModelManager {
	return models.NetworkAddressManager
}

func (set NetworkAddresses) NewModel() db.IModel {
	return &NetworkAddress{}
}

func (set NetworkAddresses) AddModel(i db.IModel) {
	m := i.(*NetworkAddress)
	set[m.Id] = m
}

func (set NetworkAddresses) Copy() apihelper.IModelSet {
	setCopy := NetworkAddresses{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set SecurityGroups) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.SecGroups
}

func (set SecurityGroups) DBModelManager() db.IModelManager {
	return models.SecurityGroupManager
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
			// log.Warningf("secgrouprule %s: secgroup %s not found",
			// 	subEntry.Id, id)
			// secgroup onpremise filter may filter out secgroup without any guest or host
			// https://github.com/yunionio/cloudpods/pull/11781
			// so ignore this error case
			// correct = false
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

func (set SecurityGroupRules) DBModelManager() db.IModelManager {
	return models.SecurityGroupRuleManager
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

func (set Guestsecgroups) DBModelManager() db.IModelManager {
	return models.GuestsecgroupManager
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

func (set Elasticips) DBModelManager() db.IModelManager {
	return models.ElasticipManager
}

func (set Elasticips) NewModel() db.IModel {
	return &Elasticip{}
}

func (set Elasticips) AddModel(i db.IModel) {
	m := i.(*Elasticip)
	if m.IpAddr == "" || m.NetworkId == "" {
		// eip records can exist with network id and ip addr not set
		return
	}
	set[m.Id] = m
}

func (set Elasticips) Copy() apihelper.IModelSet {
	setCopy := Elasticips{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set DnsZones) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.DnsZones
}

func (set DnsZones) DBModelManager() db.IModelManager {
	return models.DnsZoneManager
}

func (set DnsZones) NewModel() db.IModel {
	return &DnsZone{}
}

func (set DnsZones) AddModel(i db.IModel) {
	m := i.(*DnsZone)
	set[m.Id] = m
}

func (set DnsZones) Copy() apihelper.IModelSet {
	setCopy := DnsZones{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (ms DnsZones) joinRecords(subEntries DnsRecords) bool {
	correct := true
	for _, subEntry := range subEntries {
		zoneId := subEntry.DnsZoneId
		m, ok := ms[zoneId]
		if !ok {
			log.Warningf("dns_zone_id %s of record %s(%s) is not present", zoneId, subEntry.Name, subEntry.Id)
			correct = false
			continue
		}
		subEntry.DnsZone = m
	}
	return correct
}

func (set DnsRecords) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.DnsRecords
}

func (set DnsRecords) DBModelManager() db.IModelManager {
	return models.DnsRecordManager
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

func (set RouteTables) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.RouteTables
}

func (set RouteTables) DBModelManager() db.IModelManager {
	return models.RouteTableManager
}

func (set RouteTables) NewModel() db.IModel {
	return &RouteTable{}
}

func (set RouteTables) AddModel(i db.IModel) {
	m := i.(*RouteTable)
	set[m.Id] = m
}

func (set RouteTables) Copy() apihelper.IModelSet {
	setCopy := RouteTables{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set Groupguests) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.InstanceGroupGuests
}

func (set Groupguests) DBModelManager() db.IModelManager {
	return models.GroupguestManager
}

func (set Groupguests) NewModel() db.IModel {
	return &Groupguest{}
}

func (set Groupguests) AddModel(i db.IModel) {
	m := i.(*Groupguest)
	k := fmt.Sprintf("%d", m.RowId)
	set[k] = m
}

func (set Groupguests) Copy() apihelper.IModelSet {
	setCopy := Groupguests{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set LoadbalancerNetworks) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.Loadbalancernetworks
}

func (set LoadbalancerNetworks) DBModelManager() db.IModelManager {
	return models.LoadbalancernetworkManager
}

func (set LoadbalancerNetworks) NewModel() db.IModel {
	return &LoadbalancerNetwork{}
}

func (set LoadbalancerNetworks) AddModel(i db.IModel) {
	m := i.(*LoadbalancerNetwork)
	k := fmt.Sprintf("%s/%s", m.NetworkId, m.LoadbalancerId)
	set[k] = m
}

func (set LoadbalancerNetworks) Copy() apihelper.IModelSet {
	setCopy := LoadbalancerNetworks{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set Groupnetworks) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.InstancegroupNetworks
}

func (set Groupnetworks) DBModelManager() db.IModelManager {
	return models.GroupnetworkManager
}

func (set Groupnetworks) NewModel() db.IModel {
	return &Groupnetwork{}
}

func (set Groupnetworks) AddModel(i db.IModel) {
	m := i.(*Groupnetwork)
	k := fmt.Sprintf("%d", m.RowId)
	set[k] = m
}

func (set Groupnetworks) Copy() apihelper.IModelSet {
	setCopy := Groupnetworks{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set Groupnetworks) joinElasticips(subEntries Elasticips) bool {
	correct := true
	for _, gn := range set {
		eipId := gn.EipId
		if eipId == "" {
			continue
		}
		eip, ok := subEntries[eipId]
		if !ok {
			log.Warningf("groupnetwork %s: eip %s not found", gn.GroupId, eipId)
			correct = false
			continue
		}
		if eip.Groupnetwork != nil {
			if eip.Groupnetwork != gn {
				log.Errorf("eip %s associated to more than 1 groupnetwork: %s(%s), %s(%s)", eipId,
					eip.Groupnetwork.GroupId, eip.Groupnetwork.IpAddr, gn.GroupId, gn.IpAddr)
				correct = false
			}
			continue
		}
		eip.Groupnetwork = gn
		gn.Elasticip = eip
	}
	return correct
}

func (set Groups) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.InstanceGroups
}

func (set Groups) DBModelManager() db.IModelManager {
	return models.GroupManager
}

func (set Groups) NewModel() db.IModel {
	return &Group{}
}

func (set Groups) AddModel(i db.IModel) {
	m := i.(*Group)
	set[m.Id] = m
}

func (set Groups) Copy() apihelper.IModelSet {
	setCopy := Groups{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set Groups) joinGroupnetworks(subEntries Groupnetworks, networks Networks) bool {
	ret := true
	for _, gn := range subEntries {
		if network, ok := networks[gn.NetworkId]; ok {
			gn.Network = network
			gn.Network.Groupnetworks.AddModel(gn)
		} else {
			log.Errorf("Network %s not found for vip %s", gn.NetworkId, gn.IpAddr)
			ret = false
		}
		if _, ok := set[gn.GroupId]; !ok {
			// no group was found, create one
			grp := &Group{}
			grp.Id = gn.GroupId
			grp.Groupguests = Groupguests{}
			grp.Groupnetworks = Groupnetworks{}
			set.AddModel(grp)
		}
		group := set[gn.GroupId]
		gn.Group = group
		group.Groupnetworks.AddModel(gn)
	}
	return ret
}

func (set LoadbalancerNetworks) joinElasticips(subEntries Elasticips) bool {
	correct := true

	lnMap := map[string]*LoadbalancerNetwork{}
	for _, m := range set {
		lbId := m.LoadbalancerId
		if old, ok := lnMap[lbId]; ok {
			log.Errorf("loadbalancer %s is associated with more than 1 networks: %s, %s",
				lbId, old.NetworkId, m.NetworkId)
			correct = false
			continue
		}
		lnMap[lbId] = m
	}
	for _, subEntry := range subEntries {
		if subEntry.AssociateType != computeapis.EIP_ASSOCIATE_TYPE_LOADBALANCER {
			continue
		}
		m, ok := lnMap[subEntry.AssociateId]
		if !ok {
			log.Errorf("elasticip %s(%s) associated with non-existent loadbalancer %s",
				subEntry.Name, subEntry.Id, subEntry.AssociateId)
			correct = false
			continue
		}
		subEntry.LoadbalancerNetwork = m
		m.Elasticip = subEntry
	}
	return correct
}

func (set LoadbalancerNetworks) joinLoadbalancerListeners(subEntries LoadbalancerListeners) bool {
	lbListeners := map[string]LoadbalancerListeners{}
	for subId, subEntry := range subEntries {
		lbId := subEntry.LoadbalancerId
		v, ok := lbListeners[lbId]
		if !ok {
			v = LoadbalancerListeners{}
			lbListeners[lbId] = v
		}
		v[subId] = subEntry
	}

	for _, m := range set {
		lbId := m.LoadbalancerId
		m.LoadbalancerListeners = lbListeners[lbId]
	}
	return true
}

func (set LoadbalancerListeners) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.LoadbalancerListeners
}

func (set LoadbalancerListeners) DBModelManager() db.IModelManager {
	return models.LoadbalancerListenerManager
}

func (set LoadbalancerListeners) NewModel() db.IModel {
	return &LoadbalancerListener{}
}

func (set LoadbalancerListeners) AddModel(i db.IModel) {
	m := i.(*LoadbalancerListener)
	set[m.Id] = m
}

func (set LoadbalancerListeners) Copy() apihelper.IModelSet {
	setCopy := LoadbalancerListeners{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set LoadbalancerListeners) joinLoadbalancerAcls(subEntries LoadbalancerAcls) bool {
	for _, m := range set {
		if m.AclStatus != computeapis.LB_BOOL_ON {
			continue
		}
		switch m.AclType {
		case computeapis.LB_ACL_TYPE_WHITE,
			computeapis.LB_ACL_TYPE_BLACK:
			lbacl := subEntries[m.AclId]
			m.LoadbalancerAcl = lbacl
		}
	}
	return true
}

func (set LoadbalancerAcls) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.LoadbalancerAcls
}

func (set LoadbalancerAcls) DBModelManager() db.IModelManager {
	return models.LoadbalancerAclManager
}

func (set LoadbalancerAcls) NewModel() db.IModel {
	return &LoadbalancerAcl{}
}

func (set LoadbalancerAcls) AddModel(i db.IModel) {
	m := i.(*LoadbalancerAcl)
	set[m.Id] = m
}

func (set LoadbalancerAcls) Copy() apihelper.IModelSet {
	setCopy := LoadbalancerAcls{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}
