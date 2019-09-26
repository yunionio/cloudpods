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
	"context"
	"database/sql"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func (manager *SGuestManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []db.IModel, fields stringutils2.SSortedStrings) []*jsonutils.JSONDict {
	rows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields)
	guestIds := make([]string, len(objs))
	for i := range objs {
		guestIds[i] = objs[i].GetId()
	}
	if len(fields) == 0 || fields.Contains("disk") {
		gds := fetchGuestDiskSizes(guestIds)
		if gds != nil {
			for i := range rows {
				if gd, ok := gds[objs[i].GetId()]; ok {
					rows[i].Add(jsonutils.NewInt(gd.DiskSizeMb), "disk")
				}
			}
		}
	}
	if len(fields) == 0 || fields.Contains("ips") {
		gips := fetchGuestIPs(guestIds, tristate.False)
		if gips != nil {
			for i := range rows {
				if gip, ok := gips[objs[i].GetId()]; ok {
					rows[i].Add(jsonutils.NewString(strings.Join(gip, ",")), "ips")
				}
			}
		}
	}
	if len(fields) == 0 || fields.Contains("nics") {
		nicsMap := fetchGuestNICs(ctx, guestIds, tristate.False)
		if nicsMap != nil {
			for i := range rows {
				if nics, ok := nicsMap[objs[i].GetId()]; ok {
					rows[i].Add(nics, "nics")
				}
			}
		}
	}
	if len(fields) == 0 || fields.Contains("vpc") || fields.Contains("vpc_id") {
		gvpcs := fetchGuestVpcs(guestIds)
		if gvpcs != nil {
			for i := range rows {
				if gvpc, ok := gvpcs[objs[i].GetId()]; ok {
					if len(fields) == 0 || fields.Contains("vpc") {
						rows[i].Add(jsonutils.NewString(strings.Join(gvpc.Vpc, ",")), "vpc")
					}
					if len(fields) == 0 || fields.Contains("vpc_id") {
						rows[i].Add(jsonutils.NewString(strings.Join(gvpc.VpcId, ",")), "vpc_id")
					}
				}
			}
		}
	}
	if len(fields) == 0 || fields.Contains("secgroups") || fields.Contains("secgroup") {
		gsgs := fetchSecgroups(guestIds)
		if gsgs != nil {
			for i := range rows {
				if gsg, ok := gsgs[objs[i].GetId()]; ok {
					if len(fields) == 0 || fields.Contains("secgroups") {
						rows[i].Add(jsonutils.Marshal(gsg), "secgroups")
					}
					if len(fields) == 0 || fields.Contains("secgroup") {
						rows[i].Add(jsonutils.NewString(gsg[0].Name), "secgroup")
					}
				}
			}
		}
	}
	if len(fields) == 0 || fields.Contains("eip") || fields.Contains("eip_mode") {
		geips := fetchGuestEips(guestIds)
		if geips != nil {
			for i := range rows {
				if eip, ok := geips[objs[i].GetId()]; ok {
					if len(fields) == 0 || fields.Contains("eip") {
						rows[i].Add(jsonutils.NewString(eip.IpAddr), "eip")
					}
					if len(fields) == 0 || fields.Contains("eip_mode") {
						rows[i].Add(jsonutils.NewString(eip.Mode), "eip_mode")
					}
				}
			}
		}
	}
	if len(fields) == 0 || fields.Contains("keypair") {
		gkps := fetchGuestKeypairs(guestIds)
		if gkps != nil {
			for i := range rows {
				if kps, ok := gkps[objs[i].GetId()]; ok {
					rows[i].Add(jsonutils.NewString(kps.Keypair), "keypair")
				}
			}
		}
	}
	if len(fields) == 0 || fields.Contains("isolated_devices") || fields.Contains("is_gpu") {
		gdevs := fetchGuestIsolatedDevices(guestIds)
		if gdevs != nil {
			for i := range rows {
				if gdev, ok := gdevs[objs[i].GetId()]; ok {
					if len(fields) == 0 || fields.Contains("isolated_devices") {
						rows[i].Add(jsonutils.NewString(getIsolatedDeviceDetails(gdev)), "isolated_devices")
					}
					if len(fields) == 0 || fields.Contains("is_gpu") {
						if len(gdev) > 0 {
							rows[i].Add(jsonutils.JSONTrue, "is_gpu")
						} else {
							rows[i].Add(jsonutils.JSONFalse, "is_gpu")
						}
					}
				} else {
					if len(fields) == 0 || fields.Contains("is_gpu") {
						rows[i].Add(jsonutils.JSONFalse, "is_gpu")
					}
				}
			}
		}
	}
	if len(fields) == 0 || fields.Contains("cdrom") {
		gcds := fetchGuestCdroms(guestIds)
		if gcds != nil {
			for i := range rows {
				if gcd, ok := gcds[objs[i].GetId()]; ok {
					rows[i].Add(jsonutils.NewString(gcd.GetDetails()), "cdrom")
				} else {
					rows[i].Add(jsonutils.NewString(""), "cdrom")
				}
			}
		}
	}
	return rows
}

type sGustDiskSize struct {
	GuestId    string
	DiskSizeMb int64
}

func fetchGuestDiskSizes(guestIds []string) map[string]sGustDiskSize {
	disks := DiskManager.Query().SubQuery()
	guestdisks := GuestdiskManager.Query().SubQuery()

	q := disks.Query(guestdisks.Field("guest_id"), sqlchemy.SUM("disk_size_mb", disks.Field("disk_size")))
	q = q.Join(guestdisks, sqlchemy.Equals(guestdisks.Field("disk_id"), disks.Field("id")))
	q = q.Filter(sqlchemy.In(guestdisks.Field("guest_id"), guestIds))
	q = q.GroupBy(guestdisks.Field("guest_id"))

	gds := make([]sGustDiskSize, 0)
	err := q.All(&gds)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("query sGustDiskSize fail: %v", err)
		return nil
	}

	ret := make(map[string]sGustDiskSize)
	for i := range gds {
		ret[gds[i].GuestId] = gds[i]
	}
	return ret
}

func (guest *SGuest) getDiskSize() int {
	result := fetchGuestDiskSizes([]string{guest.Id})
	if result == nil {
		return -1
	}
	if gs, ok := result[guest.Id]; ok {
		return int(gs.DiskSizeMb)
	} else {
		return -1
	}
}

func fetchGuestIPs(guestIds []string, virtual tristate.TriState) map[string][]string {
	guestnetworks := GuestnetworkManager.Query().SubQuery()
	q := guestnetworks.Query(guestnetworks.Field("guest_id"), guestnetworks.Field("ip_addr"))
	q = q.In("guest_id", guestIds)
	if virtual.IsTrue() {
		q = q.IsTrue("virtual")
	} else if virtual.IsFalse() {
		q = q.IsFalse("virtual")
	}
	q = q.IsNotEmpty("ip_addr")
	q = q.Asc("ip_addr")
	type sGuestIdIpAddr struct {
		GuestId string
		IpAddr  string
	}
	gias := make([]sGuestIdIpAddr, 0)
	err := q.All(&gias)
	if err != nil && err != sql.ErrNoRows {
		return nil
	}
	ret := make(map[string][]string)
	for i := range gias {
		if _, ok := ret[gias[i].GuestId]; !ok {
			ret[gias[i].GuestId] = make([]string, 0)
		}
		ret[gias[i].GuestId] = append(ret[gias[i].GuestId], gias[i].IpAddr)
	}
	return ret
}

func fetchGuestNICs(ctx context.Context, guestIds []string, virtual tristate.TriState) map[string]*jsonutils.JSONArray {
	q := GuestnetworkManager.Query().In("guest_id", guestIds)
	nics := make([]SGuestnetwork, 0)
	if err := q.All(&nics); err != nil {
		return nil
	}
	ret := make(map[string]*jsonutils.JSONArray)
	for i := range nics {
		desc := nics[i].GetShortDesc(ctx)
		li := ret[nics[i].GuestId]
		if li == nil {
			li = jsonutils.NewArray(desc)
			ret[nics[i].GuestId] = li
		} else {
			li.Add(desc)
		}
	}
	return ret
}

func (self *SGuest) GetRealIPs() []string {
	result := fetchGuestIPs([]string{self.Id}, tristate.False)
	if result == nil {
		return nil
	}
	if ret, ok := result[self.Id]; ok {
		return ret
	}
	return nil
}

type sGuestVpcsInfo struct {
	GuestId string
	Vpc     []string
	VpcId   []string
}

func fetchGuestVpcs(guestIds []string) map[string]sGuestVpcsInfo {
	vpcs := VpcManager.Query().SubQuery()
	wires := WireManager.Query().SubQuery()
	networks := NetworkManager.Query().SubQuery()
	guestnetworks := GuestnetworkManager.Query().SubQuery()

	q := vpcs.Query(guestnetworks.Field("guest_id"), vpcs.Field("id"), vpcs.Field("name"))
	q = q.Join(wires, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
	q = q.Join(networks, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
	q = q.Join(guestnetworks, sqlchemy.Equals(networks.Field("id"), guestnetworks.Field("network_id")))
	q = q.Filter(sqlchemy.In(guestnetworks.Field("guest_id"), guestIds))
	q = q.Distinct()

	type sGuestVpcInfo struct {
		GuestId string
		Id      string
		Name    string
	}
	gvpcs := make([]sGuestVpcInfo, 0)
	err := q.All(&gvpcs)
	if err != nil {
		return nil
	}

	ret := make(map[string]sGuestVpcsInfo)
	for i := range gvpcs {
		gvpc, ok := ret[gvpcs[i].GuestId]
		if !ok {
			gvpc = sGuestVpcsInfo{
				GuestId: gvpcs[i].GuestId,
				Vpc:     make([]string, 0),
				VpcId:   make([]string, 0),
			}
		}
		gvpc.VpcId = append(gvpc.VpcId, gvpcs[i].Id)
		gvpc.Vpc = append(gvpc.Vpc, gvpcs[i].Name)
		ret[gvpcs[i].GuestId] = gvpc
	}

	return ret
}

type sSecgroupInfo struct {
	Id   string
	Name string
}

func fetchSecgroups(guestIds []string) map[string][]sSecgroupInfo {
	secgroups := SecurityGroupManager.Query().SubQuery()
	guestsecgroups := GuestsecgroupManager.Query().SubQuery()
	guests := GuestManager.Query().SubQuery()

	q1 := guests.Query(guests.Field("id").Label("guest_id"),
		guests.Field("secgrp_id").Label("secgroup_id"))
	q1 = q1.Filter(sqlchemy.In(guests.Field("id"), guestIds))
	q2 := guestsecgroups.Query(guestsecgroups.Field("guest_id"), guestsecgroups.Field("secgroup_id"))
	q2 = q2.Filter(sqlchemy.In(guestsecgroups.Field("guest_id"), guestIds))
	uq := sqlchemy.Union(q1, q2)
	q := uq.Query(uq.Field("guest_id"), uq.Field("secgroup_id"), secgroups.Field("name").Label("secgroup_name"))
	q = q.Join(secgroups, sqlchemy.Equals(uq.Field("secgroup_id"), secgroups.Field("id")))

	type sGuestSecgroupInfo struct {
		SecgroupId   string
		SecgroupName string
		GuestId      string
	}

	gsgs := make([]sGuestSecgroupInfo, 0)
	err := q.All(&gsgs)
	if err != nil {
		return nil
	}

	ret := make(map[string][]sSecgroupInfo)
	for i := range gsgs {
		gsg, ok := ret[gsgs[i].GuestId]
		if !ok {
			gsg = make([]sSecgroupInfo, 0)
		}
		gsg = append(gsg, sSecgroupInfo{
			Id:   gsgs[i].SecgroupId,
			Name: gsgs[i].SecgroupName,
		})
		ret[gsgs[i].GuestId] = gsg
	}

	return ret
}

type sEipInfo struct {
	IpAddr  string
	Mode    string
	GuestId string
}

func fetchGuestEips(guestIds []string) map[string]sEipInfo {
	eips := ElasticipManager.Query().SubQuery()

	q := eips.Query(eips.Field("ip_addr"), eips.Field("mode"), eips.Field("associate_id").Label("guest_id"))
	q = q.Equals("associate_type", "server")
	q = q.In("associate_id", guestIds)

	geips := make([]sEipInfo, 0)
	err := q.All(&geips)
	if err != nil {
		return nil
	}
	ret := make(map[string]sEipInfo)
	for i := range geips {
		ret[geips[i].GuestId] = geips[i]
	}
	return ret
}

type sGuestKeypair struct {
	GuestId string
	Keypair string
}

func fetchGuestKeypairs(guestIds []string) map[string]sGuestKeypair {
	keypairs := KeypairManager.Query().SubQuery()
	guests := GuestManager.Query().SubQuery()

	q := guests.Query(guests.Field("id").Label("guest_id"), keypairs.Field("name").Label("keypair"))
	q = q.Join(keypairs, sqlchemy.Equals(guests.Field("keypair_id"), keypairs.Field("id")))
	q = q.Filter(sqlchemy.In(guests.Field("id"), guestIds))

	gkps := make([]sGuestKeypair, 0)
	err := q.All(&gkps)
	if err != nil {
		return nil
	}

	ret := make(map[string]sGuestKeypair)
	for i := range gkps {
		ret[gkps[i].GuestId] = gkps[i]
	}
	return ret
}

func fetchGuestIsolatedDevices(guestIds []string) map[string][]SIsolatedDevice {
	q := IsolatedDeviceManager.Query().In("guest_id", guestIds)
	devs := make([]SIsolatedDevice, 0)
	err := q.All(&devs)
	if err != nil {
		return nil
	}
	ret := make(map[string][]SIsolatedDevice)
	for i := range devs {
		gdevs, ok := ret[devs[i].GuestId]
		if !ok {
			gdevs = make([]SIsolatedDevice, 0)
		}
		gdevs = append(gdevs, devs[i])
		ret[devs[i].GuestId] = gdevs
	}
	return ret
}

func getIsolatedDeviceDetails(devs []SIsolatedDevice) string {
	var buf strings.Builder
	for _, dev := range devs {
		buf.WriteString(dev.getDetailedString())
		buf.WriteString("\n")
	}
	return buf.String()
}

func fetchGuestCdroms(guestIds []string) map[string]SGuestcdrom {
	q := GuestcdromManager.Query().In("id", guestIds)
	gcds := make([]SGuestcdrom, 0)
	err := q.All(&gcds)
	if err != nil {
		return nil
	}
	ret := make(map[string]SGuestcdrom)
	for i := range gcds {
		ret[gcds[i].Id] = gcds[i]
	}
	return ret
}
