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
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func (manager *SGuestManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ServerDetails {
	rows := make([]api.ServerDetails, len(objs))

	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	hostRows := manager.SHostResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	encRows := manager.SEncryptedResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	guestIds := make([]string, len(objs))
	guests := make([]SGuest, len(objs))
	backupHostIds := make([]string, len(objs))
	for i := range objs {
		rows[i] = api.ServerDetails{
			VirtualResourceDetails: virtRows[i],
			HostResourceInfo:       hostRows[i],

			EncryptedResourceDetails: encRows[i],
		}
		guest := objs[i].(*SGuest)
		guestIds[i] = guest.GetId()
		backupHostIds[i] = guest.BackupHostId
		guests[i] = *guest
	}

	if len(fields) == 0 || fields.Contains("disk") {
		gds := fetchGuestDisksInfo(guestIds)
		if gds != nil {
			for i := range rows {
				rows[i].DisksInfo, _ = gds[guestIds[i]]
				rows[i].DiskCount = len(rows[i].DisksInfo)
				shortDescs := []string{}
				for _, info := range rows[i].DisksInfo {
					rows[i].DiskSizeMb += int64(info.SizeMb)
					shortDescs = append(shortDescs, info.ShortDesc())
				}
				rows[i].Disks = strings.Join(shortDescs, "\n")
			}
		}
	}
	/*if len(fields) == 0 || fields.Contains("ips") {
		gips := fetchGuestIPs(guestIds, tristate.False)
		if gips != nil {
			for i := range rows {
				if gip, ok := gips[guestIds[i]]; ok {
					rows[i].IPs = strings.Join(gip, ",")
				}
			}
		}
	}*/
	if len(fields) == 0 || fields.Contains("vip") {
		gvips := fetchGuestVips(guestIds)
		if gvips != nil {
			for i := range rows {
				if vips, ok := gvips[guestIds[i]]; ok {
					rows[i].Vip = strings.Join(vips, ",")
				}
			}
		}
	}
	if len(fields) == 0 || fields.Contains("vip_eip") {
		gvips := fetchGuestVipEips(guestIds)
		if gvips != nil {
			for i := range rows {
				if vips, ok := gvips[guestIds[i]]; ok {
					rows[i].VipEip = strings.Join(vips, ",")
				}
			}
		}
	}

	if len(fields) == 0 || fields.Contains("ips") || fields.Contains("macs") || fields.Contains("nics") || fields.Contains("subips") {
		nicsMap := fetchGuestNICs(ctx, guestIds, tristate.False)
		if nicsMap != nil {
			for i := range rows {
				if nics, ok := nicsMap[guestIds[i]]; ok {
					if len(fields) == 0 || fields.Contains("nics") {
						rows[i].Nics = nics
					}
					if len(fields) == 0 || fields.Contains("macs") {
						macs := make([]string, 0, len(nics))
						for _, nic := range nics {
							macs = append(macs, nic.Mac)
						}
						rows[i].Macs = strings.Join(macs, ",")
					}
					if len(fields) == 0 || fields.Contains("ips") {
						ips := make([]string, 0, len(nics))
						for _, nic := range nics {
							if len(nic.IpAddr) > 0 {
								ips = append(ips, nic.IpAddr)
							}
							if len(nic.Ip6Addr) > 0 {
								ips = append(ips, nic.Ip6Addr)
							}
						}
						rows[i].IPs = strings.Join(ips, ",")
					}
					if len(fields) == 0 || fields.Contains("subips") {
						subips := make([]string, 0)
						for _, nic := range nics {
							if len(nic.SubIps) > 0 {
								ips := strings.Split(nic.SubIps, ",")
								subips = append(subips, ips...)
							}
						}
						rows[i].SubIPs = subips
					}
				}
			}
		}
	}

	if len(fields) == 0 || fields.Contains("vpc") || fields.Contains("vpc_id") {
		gvpcs := fetchGuestVpcs(guestIds)
		if gvpcs != nil {
			for i := range rows {
				if gvpc, ok := gvpcs[guestIds[i]]; ok {
					if len(fields) == 0 || fields.Contains("vpc") {
						rows[i].Vpc = strings.Join(gvpc.Vpc, ",")
					}
					if len(fields) == 0 || fields.Contains("vpc_id") {
						rows[i].VpcId = strings.Join(gvpc.VpcId, ",")
					}

					if len(fields) == 0 || fields.Contains("external_access_mode") {
						rows[i].VpcExternalAccessMode = strings.Join(gvpc.ExternalAccessMode, ",")
					}
				}
			}
		}
	}
	if len(fields) == 0 || fields.Contains("secgroups") || fields.Contains("secgroup") {
		gsgs := fetchSecgroups(guestIds)
		if gsgs != nil {
			for i := range rows {
				if gsg, ok := gsgs[guestIds[i]]; ok {
					if len(fields) == 0 || fields.Contains("secgroups") {
						rows[i].Secgroups = gsg
					}
					if len(fields) == 0 || fields.Contains("secgroup") {
						rows[i].Secgroup = gsg[0].Name
					}
				}
			}
		}
	}
	if len(fields) == 0 || fields.Contains("eip") || fields.Contains("eip_mode") {
		geips := fetchGuestEips(guestIds)
		if geips != nil {
			for i := range rows {
				if eip, ok := geips[guestIds[i]]; ok {
					if len(fields) == 0 || fields.Contains("eip") {
						rows[i].Eip = eip.IpAddr
					}
					if len(fields) == 0 || fields.Contains("eip_mode") {
						rows[i].EipMode = eip.Mode
					}
				}
			}
		}
	}
	if len(fields) == 0 || fields.Contains("keypair") {
		gkps := fetchGuestKeypairs(guestIds)
		if gkps != nil {
			for i := range rows {
				if kps, ok := gkps[guestIds[i]]; ok {
					rows[i].Keypair = kps.Keypair
				}
			}
		}
	}
	if len(fields) == 0 || fields.Contains("isolated_devices") || fields.Contains("is_gpu") {
		gdevs := fetchGuestIsolatedDevices(guestIds)
		if gdevs != nil {
			for i := range rows {
				if gdev, ok := gdevs[guestIds[i]]; ok {
					if len(fields) == 0 || fields.Contains("isolated_devices") {
						rows[i].IsolatedDevices = gdev
					}
					if len(fields) == 0 || fields.Contains("is_gpu") {
						if len(gdev) > 0 {
							rows[i].IsGpu = true
						} else {
							rows[i].IsGpu = false
						}
					}
				} else {
					if len(fields) == 0 || fields.Contains("is_gpu") {
						rows[i].IsGpu = false
					}
				}
			}
		}
		info, _ := fetchGuestGpuInstanceTypes(guestIds)
		if len(info) > 0 {
			for i := range rows {
				gpu, ok := info[guests[i].InstanceType]
				if ok {
					rows[i].IsGpu = true
					rows[i].GpuModel = gpu.Model
					rows[i].GpuCount = gpu.Amount
				}
			}
		}
	}
	if len(fields) == 0 || fields.Contains("cdrom") {
		gcds := fetchGuestCdroms(guestIds)
		if gcds != nil {
			for i := range rows {
				for _, gcd := range gcds[guestIds[i]] {
					if details := gcd.GetDetails(); len(details) > 0 {
						t := api.Cdrom{Ordinal: gcd.Ordinal, Detail: details, BootIndex: gcd.BootIndex, Name: gcd.Name}
						rows[i].Cdrom = append(rows[i].Cdrom, t)
					}
				}
			}
		}
	}
	if len(fields) == 0 || fields.Contains("floppy") {
		gfloppys := fetchGuestFloppys(guestIds)
		if gfloppys != nil {
			for i := range rows {
				for _, gfl := range gfloppys[guestIds[i]] {
					if details := gfl.GetDetails(); len(details) > 0 {
						t := api.Floppy{Ordinal: gfl.Ordinal, Detail: details}
						rows[i].Floppy = append(rows[i].Floppy, t)
					}
				}
			}
		}
	}

	if len(fields) == 0 || fields.Contains("scaling_group") {
		sggs := fetchScalingGroupGuest(guestIds...)
		if sggs != nil && len(sggs) != 0 {
			for i := range rows {
				if sgg, ok := sggs[guestIds[i]]; ok {
					rows[i].ScalingStatus = sgg.GuestStatus
					rows[i].ScalingGroupId = sgg.ScalingGroupId
				}
			}
		}
	}

	if len(fields) == 0 || fields.Contains("backup_host_name") || fields.Contains("backup_host_status") && len(backupHostIds) > 0 {
		backups, _ := fetchGuestBackupInfo(backupHostIds)
		meta := []db.SMetadata{}
		db.Metadata.Query().In("obj_id", guestIds).Equals("obj_type", manager.Keyword()).Equals("key", api.MIRROR_JOB).All(&meta)
		syncStatus := map[string]string{}
		for i := range meta {
			v := meta[i]
			syncStatus[v.ObjId] = v.Value
		}
		if len(backups) > 0 || len(syncStatus) > 0 {
			for i := range rows {
				rows[i].BackupInfo, _ = backups[backupHostIds[i]]
				rows[i].BackupGuestSyncStatus, _ = syncStatus[guestIds[i]]
			}
		}
	}

	if len(fields) == 0 || fields.Contains("container") {
		containers, _ := fetchContainers(guestIds)
		if len(containers) > 0 {
			for i := range rows {
				rows[i].Containers, _ = containers[guestIds[i]]
			}
		}
	}

	for i := range rows {
		if len(fields) == 0 || fields.Contains("auto_delete_at") {
			if guests[i].PendingDeleted {
				pendingDeletedAt := guests[i].PendingDeletedAt.Add(time.Second * time.Duration(options.Options.PendingDeleteExpireSeconds))
				rows[i].AutoDeleteAt = pendingDeletedAt
			}
		}
		if len(fields) == 0 || fields.Contains("can_recycle") {
			if guests[i].BillingType == billing_api.BILLING_TYPE_PREPAID && !guests[i].ExpiredAt.Before(time.Now()) && len(rows[i].ManagerId) > 0 {
				rows[i].CanRecycle = true
			}
		}

		rows[i].IsPrepaidRecycle = (rows[i].HostResourceType == api.HostResourceTypePrepaidRecycle && rows[i].HostBillingType == billing_api.BILLING_TYPE_PREPAID)

		drv, _ := GetDriver(guests[i].Hypervisor, rows[i].Provider)
		if drv != nil {
			rows[i].CdromSupport, _ = drv.IsSupportCdrom(&guests[i])
			rows[i].FloppySupport, _ = drv.IsSupportFloppy(&guests[i])
			rows[i].MonitorUrl = drv.FetchMonitorUrl(ctx, &guests[i])
		}

		if len(guests[i].HostId) == 0 && guests[i].Status == api.VM_SCHEDULE_FAILED {
			rows[i].Brand = "Unknown"
			rows[i].Provider = "Unknown"
		}

		if !isList {
			rows[i].Networks = guests[i].getNetworksDetails()
			rows[i].VirtualIps = strings.Join(guests[i].getVirtualIPs(), ",")
			rows[i].SecurityRules = guests[i].getSecurityGroupsRules()

			osName := guests[i].GetOS()
			if len(osName) > 0 {
				rows[i].OsName = osName
				if len(guests[i].OsType) == 0 {
					rows[i].OsType = osName
				}
			}

			if userCred.HasSystemAdminPrivilege() {
				rows[i].AdminSecurityRules = guests[i].getAdminSecurityRules()
			}
		}
	}

	return rows
}

type sGustDiskSize struct {
	GuestId    string
	DiskSizeMb int64
	DiskCount  int
}

type sGuestDiskInfo struct {
	api.GuestDiskInfo
	GuestId string
}

func fetchGuestDisksInfo(guestIds []string) map[string][]api.GuestDiskInfo {
	disks := DiskManager.Query().SubQuery()
	guestdisks := GuestdiskManager.Query().SubQuery()
	storages := StorageManager.Query().SubQuery()
	q := disks.Query(
		disks.Field("id"),
		disks.Field("name"),
		disks.Field("fs_format"),
		disks.Field("disk_type"),
		guestdisks.Field("index"),
		disks.Field("disk_size").Label("size"),
		disks.Field("disk_format"),
		guestdisks.Field("driver"),
		guestdisks.Field("cache_mode"),
		guestdisks.Field("aio_mode"),
		storages.Field("medium_type"),
		storages.Field("storage_type"),
		guestdisks.Field("iops"),
		guestdisks.Field("bps"),
		disks.Field("template_id").Label("image_id"),
		guestdisks.Field("guest_id"),
		guestdisks.Field("boot_index"),
		disks.Field("storage_id"),
		disks.Field("preallocation"),
	)
	q = q.Join(guestdisks, sqlchemy.Equals(guestdisks.Field("disk_id"), disks.Field("id")))
	q = q.Join(storages, sqlchemy.Equals(disks.Field("storage_id"), storages.Field("id")))
	q = q.Filter(sqlchemy.In(guestdisks.Field("guest_id"), guestIds)).Asc(guestdisks.Field("index"))
	gds := []sGuestDiskInfo{}
	err := q.All(&gds)
	if err != nil {
		return nil
	}
	imageIds := []string{}
	ret := map[string][]api.GuestDiskInfo{}
	for i := range gds {
		if len(gds[i].ImageId) > 0 {
			imageIds = append(imageIds, gds[i].ImageId)
		}
		_, ok := ret[gds[i].GuestId]
		if !ok {
			ret[gds[i].GuestId] = []api.GuestDiskInfo{}
		}
		ret[gds[i].GuestId] = append(ret[gds[i].GuestId], gds[i].GuestDiskInfo)
	}
	imageNames, err := db.FetchIdNameMap2(CachedimageManager, imageIds)
	if err != nil {
		return ret
	}
	for guestId, infos := range ret {
		for i := range infos {
			if len(infos[i].ImageId) > 0 {
				ret[guestId][i].Image, _ = imageNames[infos[i].ImageId]
			}
		}
	}
	return ret
}

func (guest *SGuest) GetDisksSize() int {
	return guest.getDiskSize()
}

func (guest *SGuest) getDiskSize() int {
	result := fetchGuestDisksInfo([]string{guest.Id})
	if result == nil {
		return -1
	}
	gds, ok := result[guest.Id]
	if !ok {
		return -1
	}
	size := 0
	for _, gd := range gds {
		size += gd.SizeMb
	}
	return size
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
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
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

func fetchGuestVips(guestIds []string) map[string][]string {
	groupguests := GroupguestManager.Query().SubQuery()
	groupnetworks := GroupnetworkManager.Query().SubQuery()
	q := groupnetworks.Query(groupnetworks.Field("ip_addr"), groupguests.Field("guest_id"))
	q = q.Join(groupguests, sqlchemy.Equals(q.Field("group_id"), groupguests.Field("group_id")))
	q = q.In("guest_id", guestIds)
	type sGuestVip struct {
		IpAddr  string
		GuestId string
	}
	gvips := make([]sGuestVip, 0)
	err := q.All(&gvips)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil
	}
	ret := make(map[string][]string)
	for i := range gvips {
		if _, ok := ret[gvips[i].GuestId]; !ok {
			ret[gvips[i].GuestId] = make([]string, 0)
		}
		ret[gvips[i].GuestId] = append(ret[gvips[i].GuestId], gvips[i].IpAddr)
	}
	return ret
}

func fetchGuestVipEips(guestIds []string) map[string][]string {
	groupguests := GroupguestManager.Query().SubQuery()
	eips := ElasticipManager.Query().Equals("associate_type", api.EIP_ASSOCIATE_TYPE_INSTANCE_GROUP).SubQuery()

	q := eips.Query(eips.Field("ip_addr"), groupguests.Field("guest_id"))
	q = q.Join(groupguests, sqlchemy.Equals(eips.Field("associate_id"), groupguests.Field("group_id")))
	q = q.In("guest_id", guestIds)
	type sGuestVip struct {
		IpAddr  string
		GuestId string
	}
	gvips := make([]sGuestVip, 0)
	err := q.All(&gvips)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil
	}
	ret := make(map[string][]string)
	for i := range gvips {
		if _, ok := ret[gvips[i].GuestId]; !ok {
			ret[gvips[i].GuestId] = make([]string, 0)
		}
		ret[gvips[i].GuestId] = append(ret[gvips[i].GuestId], gvips[i].IpAddr)
	}
	return ret
}

func fetchGuestNICs(ctx context.Context, guestIds []string, virtual tristate.TriState) map[string][]api.GuestnetworkShortDesc {
	netq := NetworkManager.Query().SubQuery()
	wirq := WireManager.Query().SubQuery()

	subIPQ := NetworkAddressManager.Query("parent_id").Equals("parent_type", api.NetworkAddressParentTypeGuestnetwork)
	subIPQ = subIPQ.AppendField(sqlchemy.GROUP_CONCAT("sub_ips", subIPQ.Field("ip_addr")))
	subIPQ = subIPQ.GroupBy(subIPQ.Field("parent_id"))
	subIP := subIPQ.SubQuery()

	gnwq := GuestnetworkManager.Query()
	q := gnwq.AppendField(
		gnwq.Field("guest_id"),

		gnwq.Field("ip_addr"),
		gnwq.Field("ip6_addr"),
		gnwq.Field("mac_addr").Label("mac"),
		gnwq.Field("team_with"),
		gnwq.Field("network_id"), // caution: do not alias netq.id as network_id

		gnwq.Field("port_mappings"),

		wirq.Field("vpc_id"),
		subIP.Field("sub_ips"),
	)
	q = q.Join(netq, sqlchemy.Equals(netq.Field("id"), gnwq.Field("network_id")))
	q = q.Join(wirq, sqlchemy.Equals(wirq.Field("id"), netq.Field("wire_id")))
	q = q.LeftJoin(subIP, sqlchemy.Equals(q.Field("row_id"), subIP.Field("parent_id")))
	q = q.In("guest_id", guestIds)

	var descs []struct {
		GuestId string `json:"guest_id"`
		api.GuestnetworkShortDesc
	}
	if err := q.All(&descs); err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("query guest nics info: %v", err)
		}
		return nil
	}
	ret := map[string][]api.GuestnetworkShortDesc{}
	for i := range descs {
		desc := &descs[i]
		guestId := desc.GuestId
		if _, ok := ret[guestId]; !ok {
			ret[guestId] = []api.GuestnetworkShortDesc{desc.GuestnetworkShortDesc}
		} else {
			ret[guestId] = append(ret[guestId], desc.GuestnetworkShortDesc)
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

func (self *SGuest) fetchNICShortDesc(ctx context.Context) []api.GuestnetworkShortDesc {
	nicsMap := fetchGuestNICs(ctx, []string{self.Id}, tristate.False)
	if nicsMap == nil {
		return nil
	}
	return nicsMap[self.Id]
}

type sGuestVpcsInfo struct {
	GuestId            string
	Vpc                []string
	VpcId              []string
	ExternalAccessMode []string
}

func fetchGuestVpcs(guestIds []string) map[string]sGuestVpcsInfo {
	vpcs := VpcManager.Query().SubQuery()
	wires := WireManager.Query().SubQuery()
	networks := NetworkManager.Query().SubQuery()
	guestnetworks := GuestnetworkManager.Query().SubQuery()

	q := vpcs.Query(guestnetworks.Field("guest_id"), vpcs.Field("id"), vpcs.Field("name"), vpcs.Field("external_access_mode"))
	q = q.Join(wires, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
	q = q.Join(networks, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
	q = q.Join(guestnetworks, sqlchemy.Equals(networks.Field("id"), guestnetworks.Field("network_id")))
	q = q.Filter(sqlchemy.In(guestnetworks.Field("guest_id"), guestIds))
	q = q.Distinct()

	type sGuestVpcInfo struct {
		GuestId            string
		Id                 string
		Name               string
		ExternalAccessMode string
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
				GuestId:            gvpcs[i].GuestId,
				Vpc:                make([]string, 0),
				VpcId:              make([]string, 0),
				ExternalAccessMode: make([]string, 0),
			}
		}
		gvpc.VpcId = append(gvpc.VpcId, gvpcs[i].Id)
		gvpc.Vpc = append(gvpc.Vpc, gvpcs[i].Name)
		gvpc.ExternalAccessMode = append(gvpc.ExternalAccessMode, gvpcs[i].ExternalAccessMode)
		ret[gvpcs[i].GuestId] = gvpc
	}

	return ret
}

func fetchSecgroups(guestIds []string) map[string][]apis.StandaloneShortDesc {
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

	ret := make(map[string][]apis.StandaloneShortDesc)
	for i := range gsgs {
		gsg, ok := ret[gsgs[i].GuestId]
		if !ok {
			gsg = make([]apis.StandaloneShortDesc, 0)
		}
		gsg = append(gsg, apis.StandaloneShortDesc{
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
	q = q.Equals("associate_type", api.EIP_ASSOCIATE_TYPE_SERVER)
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

func fetchGuestGpuInstanceTypes(guestIds []string) (map[string]*GpuSpec, error) {
	ret := map[string]*GpuSpec{}
	sq := GuestManager.Query("instance_type").In("id", guestIds).SubQuery()
	q := ServerSkuManager.Query("name", "gpu_spec", "gpu_count").In("name", sq).IsNotEmpty("gpu_spec").Distinct()
	gpus := []struct {
		Name     string
		GpuSpec  string
		GpuCount string
	}{}
	err := q.All(&gpus)
	if err != nil {
		return ret, err
	}
	for _, gpu := range gpus {
		ret[gpu.Name] = &GpuSpec{Model: gpu.GpuSpec, Amount: gpu.GpuCount}
	}
	return ret, nil
}

func fetchGuestBackupInfo(hostIds []string) (map[string]api.BackupInfo, error) {
	ret := map[string]api.BackupInfo{}
	hosts := []SHost{}
	err := HostManager.Query().In("id", hostIds).All(&hosts)
	if err != nil {
		return nil, err
	}
	for _, host := range hosts {
		ret[host.Id] = api.BackupInfo{BackupHostName: host.Name, BackupHostStatus: host.HostStatus}
	}
	return ret, nil
}

func fetchContainers(guestIds []string) (map[string][]*api.PodContainerDesc, error) {
	ret := map[string][]*api.PodContainerDesc{}
	containers := []SContainer{}
	err := GetContainerManager().Query().In("guest_id", guestIds).All(&containers)
	if err != nil {
		return nil, err
	}
	for i := range containers {
		container := containers[i]
		_, ok := ret[container.GuestId]
		if !ok {
			ret[container.GuestId] = []*api.PodContainerDesc{}
		}
		desc := &api.PodContainerDesc{
			Id:     container.GetId(),
			Name:   container.GetName(),
			Status: container.Status,
		}
		if container.Spec != nil {
			desc.Image = container.Spec.Image
		}
		ret[container.GuestId] = append(ret[container.GuestId], desc)
	}
	return ret, nil
}

func fetchGuestIsolatedDevices(guestIds []string) map[string][]api.SIsolatedDevice {
	q := IsolatedDeviceManager.Query().In("guest_id", guestIds)
	devs := make([]SIsolatedDevice, 0)
	err := q.All(&devs)
	if err != nil {
		return nil
	}
	ret := make(map[string][]api.SIsolatedDevice)
	for i := range devs {
		dev := api.SIsolatedDevice{}
		dev.Id = devs[i].Id
		dev.HostId = devs[i].HostId
		dev.DevType = devs[i].DevType
		dev.Model = devs[i].Model
		dev.GuestId = devs[i].GuestId
		dev.Addr = devs[i].Addr
		dev.VendorDeviceId = devs[i].VendorDeviceId
		dev.NumaNode = byte(devs[i].NumaNode)
		gdevs, ok := ret[devs[i].GuestId]
		if !ok {
			gdevs = make([]api.SIsolatedDevice, 0)
		}
		gdevs = append(gdevs, dev)
		ret[devs[i].GuestId] = gdevs
	}
	return ret
}

func fetchGuestCdroms(guestIds []string) map[string][]SGuestcdrom {
	sq := GuestcdromManager.Query().In("id", guestIds).SubQuery()
	image := CachedimageManager.Query().SubQuery()

	q := sq.Query(
		sq.Field("id"),
		sq.Field("path"),
		sq.Field("boot_index"),
		sq.Field("image_id"),
		image.Field("size"),
		image.Field("name"),
	)
	q = q.LeftJoin(image, sqlchemy.Equals(sq.Field("image_id"), image.Field("id")))

	gcds := make([]SGuestcdrom, 0)
	err := q.All(&gcds)
	if err != nil {
		return nil
	}
	ret := make(map[string][]SGuestcdrom)
	for i := range gcds {
		ret[gcds[i].Id] = append(ret[gcds[i].Id], gcds[i])
	}
	return ret
}

func fetchGuestFloppys(guestIds []string) map[string][]SGuestfloppy {
	q := GuestFloppyManager.Query().In("id", guestIds)
	gfls := make([]SGuestfloppy, 0)
	err := q.All(&gfls)
	if err != nil {
		return nil
	}
	ret := make(map[string][]SGuestfloppy)
	for i := range gfls {
		ret[gfls[i].Id] = append(ret[gfls[i].Id], gfls[i])
	}
	return ret
}

func fetchScalingGroupGuest(guestIds ...string) map[string]SScalingGroupGuest {
	q := ScalingGroupGuestManager.Query().In("guest_id", guestIds)
	sggs := make([]SScalingGroupGuest, 0)
	err := q.All(&sggs)
	if err != nil {
		return nil
	}
	ret := make(map[string]SScalingGroupGuest)
	for i := range sggs {
		ret[sggs[i].GuestId] = sggs[i]
	}
	return ret
}

func (self *SGuest) SyncInstanceSnapshots(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider) compare.SyncResult {
	syncResult := compare.SyncResult{}

	extGuest, err := self.GetIVM(ctx)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	extSnapshots, err := extGuest.GetInstanceSnapshots()
	if errors.Cause(err) == errors.ErrNotImplemented {
		return syncResult
	}
	syncOwnerId := provider.GetOwnerId()
	localSnapshots, err := self.GetInstanceSnapshots()
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	lockman.LockClass(ctx, InstanceSnapshotManager, db.GetLockClassKey(InstanceSnapshotManager, syncOwnerId))
	defer lockman.ReleaseClass(ctx, InstanceSnapshotManager, db.GetLockClassKey(InstanceSnapshotManager, syncOwnerId))

	removed := make([]SInstanceSnapshot, 0)
	commondb := make([]SInstanceSnapshot, 0)
	commonext := make([]cloudprovider.ICloudInstanceSnapshot, 0)
	added := make([]cloudprovider.ICloudInstanceSnapshot, 0)

	err = compare.CompareSets(localSnapshots, extSnapshots, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}
	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudInstanceSnapshot(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].SyncWithCloudInstanceSnapshot(ctx, userCred, commonext[i], self)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		_, err := InstanceSnapshotManager.newFromCloudInstanceSnapshot(ctx, userCred, added[i], self)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}
	return syncResult
}
