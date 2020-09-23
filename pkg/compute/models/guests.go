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
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/cloudcommon/userdata"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/compute/sshkeys"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=server
// +onecloud:swagger-gen-model-plural=servers
type SGuestManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SDeletePreventableResourceBaseManager

	SHostResourceBaseManager
	SBillingResourceBaseManager
	SNetworkResourceBaseManager
	SDiskResourceBaseManager
	SScalingGroupResourceBaseManager
	db.SMultiArchResourceBaseManager
}

var GuestManager *SGuestManager

func init() {
	GuestManager = &SGuestManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SGuest{},
			"guests_tbl",
			"server",
			"servers",
		),
	}
	GuestManager.SetVirtualObject(GuestManager)
	GuestManager.SetAlias("guest", "guests")
}

type SGuest struct {
	db.SVirtualResourceBase

	db.SExternalizedResourceBase

	SBillingResourceBase
	SDeletePreventableResourceBase
	db.SMultiArchResourceBase

	SHostResourceBase `width:"36" charset:"ascii" nullable:"true" list:"user" get:"user" index:"true"`

	// CPU大小
	VcpuCount int `nullable:"false" default:"1" list:"user" create:"optional"`
	// 内存大小, 单位Mb
	VmemSize int `nullable:"false" list:"user" create:"required"`

	// 启动顺序
	BootOrder string `width:"8" charset:"ascii" nullable:"true" default:"cdn" list:"user" update:"user" create:"optional"`

	// 关机操作类型
	// example: stop
	ShutdownBehavior string `width:"16" charset:"ascii" default:"stop" list:"user" update:"user" create:"optional"`

	// 秘钥对Id
	KeypairId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	// 宿主机Id
	//HostId string `width:"36" charset:"ascii" nullable:"true" list:"admin" get:"admin" index:"true"`

	// 备份机所在宿主机Id
	BackupHostId string `width:"36" charset:"ascii" nullable:"true" list:"user" get:"user"`

	Vga     string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`
	Vdi     string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`
	Machine string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`
	Bios    string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`
	// 操作系统类型
	OsType string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`

	FlavorId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	// 安全组Id
	// example: default
	SecgrpId string `width:"36" charset:"ascii" nullable:"true" list:"user" get:"user" create:"optional"`
	// 管理员可见安全组Id
	AdminSecgrpId string `width:"36" charset:"ascii" nullable:"true" list:"admin" get:"domain"`

	SrcIpCheck  tristate.TriState `nullable:"false" default:"true" create:"optional" list:"user" update:"user"`
	SrcMacCheck tristate.TriState `nullable:"false" default:"true" create:"optional" list:"user" update:"user"`

	// 虚拟化技术
	// example: kvm
	Hypervisor string `width:"16" charset:"ascii" nullable:"false" default:"kvm" list:"user" create:"required"`

	// 套餐名称
	InstanceType string `width:"64" charset:"utf8" nullable:"true" list:"user" create:"optional"`

	SshableLastState tristate.TriState `nullable:"false" default:"false" list:"user"`
}

func (manager *SGuestManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	if query.Contains("host") || query.Contains("wire") || query.Contains("zone") {
		if !db.IsAdminAllowList(userCred, manager) {
			return false
		}
	}
	return manager.SVirtualResourceBaseManager.AllowListItems(ctx, userCred, query)
}

// 云主机实例列表
func (manager *SGuestManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ServerListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SHostResourceBaseManager.ListItemFilter(ctx, q, userCred, query.HostFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SHostResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SDeletePreventableResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DeletePreventableResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDeletePreventableResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SBillingResourceBaseManager.ListItemFilter(ctx, q, userCred, query.BillingResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SBillingResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SMultiArchResourceBaseManager.ListItemFilter(ctx, q, userCred, query.MultiArchResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "MultiArchResourceBaseListInput.ListItemFilter")
	}

	netQ := GuestnetworkManager.Query("guest_id").Snapshot()
	netQ, err = manager.SNetworkResourceBaseManager.ListItemFilter(ctx, netQ, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.ListItemFilter")
	}
	if netQ.IsAltered() {
		q = q.In("id", netQ.SubQuery())
	}

	//diskQ := GuestdiskManager.Query("guest_id").Snapshot()
	//diskQ, err = manager.SDiskResourceBaseManager.ListItemFilter(ctx, diskQ, userCred, query.DiskFilterListInput)
	//if err != nil {
	//	return nil, errors.Wrap(err, "SDiskResourceBaseManager.ListItemFilter")
	//}
	//if diskQ.IsAltered() {
	//	q = q.In("id", diskQ.SubQuery())
	//}

	scalingGroupQ := ScalingGroupGuestManager.Query("guest_id").Snapshot()
	scalingGroupQ, err = manager.SScalingGroupResourceBaseManager.ListItemFilter(ctx, scalingGroupQ, userCred, query.ScalingGroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SScaligGroupResourceBaseManager.ListItemFilter")
	}
	if scalingGroupQ.IsAltered() {
		q = q.In("id", scalingGroupQ.SubQuery())
	}

	hypervisorList := query.Hypervisor
	if len(hypervisorList) > 0 {
		q = q.In("hypervisor", hypervisorList)
	}

	resourceTypeStr := query.ResourceType
	if len(resourceTypeStr) > 0 {
		hosts := HostManager.Query().SubQuery()
		subq := hosts.Query(hosts.Field("id"))
		switch resourceTypeStr {
		case api.HostResourceTypeShared:
			subq = subq.Filter(
				sqlchemy.OR(
					sqlchemy.IsNullOrEmpty(hosts.Field("resource_type")),
					sqlchemy.Equals(hosts.Field("resource_type"), resourceTypeStr),
				),
			)
		default:
			subq = subq.Equals("resource_type", resourceTypeStr)
		}

		q = q.In("host_id", subq.SubQuery())
	}

	hostFilter := query.GetAllGuestsOnHost
	if len(hostFilter) > 0 {
		host, _ := HostManager.FetchByIdOrName(nil, hostFilter)
		if host == nil {
			return nil, httperrors.NewResourceNotFoundError("host %s not found", hostFilter)
		}
		q.Filter(sqlchemy.OR(sqlchemy.Equals(q.Field("host_id"), host.GetId()),
			sqlchemy.Equals(q.Field("backup_host_id"), host.GetId())))
	}

	secgrpFilter := query.SecgroupId
	if len(secgrpFilter) > 0 {
		var notIn = false
		// HACK FOR NOT IN SECGROUP
		if strings.HasPrefix(secgrpFilter, "!") {
			secgrpFilter = secgrpFilter[1:]
			notIn = true
		}
		secgrpIds := []string{}
		secgrps := []SSecurityGroup{}
		sgq := SecurityGroupManager.Query()
		sgq = sgq.Filter(sqlchemy.OR(sqlchemy.Equals(sgq.Field("id"), secgrpFilter), sqlchemy.Equals(sgq.Field("name"), secgrpFilter)))
		if err := db.FetchModelObjects(SecurityGroupManager, sgq, &secgrps); err != nil {
			return nil, err
		}
		if len(secgrps) == 0 {
			return nil, httperrors.NewResourceNotFoundError("secgroup %s not found", secgrpFilter)
		}

		for _, secgrp := range secgrps {
			secgrpIds = append(secgrpIds, secgrp.Id)
		}

		isAdmin := false
		admin := (query.VirtualResourceListInput.Admin != nil && *query.VirtualResourceListInput.Admin)
		if consts.IsRbacEnabled() {
			allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList)
			if allowScope == rbacutils.ScopeSystem || allowScope == rbacutils.ScopeDomain {
				isAdmin = true
			}
		} else if userCred.HasSystemAdminPrivilege() && admin {
			isAdmin = true
		}

		filters := []sqlchemy.ICondition{}
		if notIn {
			filters = append(filters, sqlchemy.NotIn(q.Field("id"),
				GuestsecgroupManager.Query("guest_id").In("secgroup_id", secgrpIds).SubQuery()))
			filters = append(filters, sqlchemy.NotIn(q.Field("secgrp_id"), secgrpIds))
			if isAdmin {
				filters = append(filters, sqlchemy.NotIn(q.Field("admin_secgrp_id"), secgrpIds))
			}
			q = q.Filter(sqlchemy.AND(filters...))
		} else {
			filters = append(filters, sqlchemy.In(q.Field("id"),
				GuestsecgroupManager.Query("guest_id").In("secgroup_id", secgrpIds).SubQuery()))
			filters = append(filters, sqlchemy.In(q.Field("secgrp_id"), secgrpIds))
			if isAdmin {
				filters = append(filters, sqlchemy.In(q.Field("admin_secgrp_id"), secgrpIds))
			}
			q = q.Filter(sqlchemy.OR(filters...))
		}
	}

	usableServerForEipFilter := query.UsableServerForEip
	if len(usableServerForEipFilter) > 0 {
		eipObj, err := ElasticipManager.FetchByIdOrName(userCred, usableServerForEipFilter)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError("eip %s not found", usableServerForEipFilter)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		eip := eipObj.(*SElasticip)

		if len(eip.NetworkId) > 0 {
			sq := GuestnetworkManager.Query("guest_id").Equals("network_id", eip.NetworkId).SubQuery()
			q = q.NotIn("id", sq)
			if cp := eip.GetCloudprovider(); cp == nil || cp.Provider == api.CLOUD_PROVIDER_ONECLOUD {
				gnq := GuestnetworkManager.Query().SubQuery()
				nq := NetworkManager.Query().SubQuery()
				wq := WireManager.Query().SubQuery()
				vq := VpcManager.Query().SubQuery()
				q.Join(gnq, sqlchemy.Equals(gnq.Field("guest_id"), q.Field("id")))
				q.Join(nq, sqlchemy.Equals(nq.Field("id"), gnq.Field("network_id")))
				q.Join(wq, sqlchemy.Equals(wq.Field("id"), nq.Field("wire_id")))
				q.Join(vq, sqlchemy.Equals(vq.Field("id"), wq.Field("vpc_id")))
				q.Filter(sqlchemy.IsNullOrEmpty(gnq.Field("eip_id")))
				q.Filter(sqlchemy.NotEquals(vq.Field("id"), api.DEFAULT_VPC_ID))
				// vpc provider thing will be handled ok below
			}
		}

		hostTable := HostManager.Query().SubQuery()
		zoneTable := ZoneManager.Query().SubQuery()
		hostQ := hostTable.Query(hostTable.Field("id"))
		hostQ = hostQ.Join(zoneTable,
			sqlchemy.Equals(zoneTable.Field("id"), hostTable.Field("zone_id")))
		if eip.ManagerId != "" {
			hostQ = hostQ.Equals("manager_id", eip.ManagerId)
		} else {
			hostQ = hostQ.IsNullOrEmpty("manager_id")
		}
		region, err := eip.GetRegion()
		if err != nil {
			return nil, httperrors.NewGeneralError(errors.Wrapf(err, "eip.GetRegion"))
		}
		regionTable := CloudregionManager.Query().SubQuery()
		sq := hostQ.Join(regionTable, sqlchemy.Equals(zoneTable.Field("cloudregion_id"), regionTable.Field("id"))).
			Filter(sqlchemy.Equals(regionTable.Field("id"), region.GetId())).SubQuery()
		q = q.In("host_id", sq)
	}

	if len(query.IpAddr) > 0 {
		gn := GuestnetworkManager.Query("guest_id").Contains("ip_addr", query.IpAddr).SubQuery()
		guestEip := ElasticipManager.Query("associate_id").Equals("associate_type", api.EIP_ASSOCIATE_TYPE_SERVER).Contains("ip_addr", query.IpAddr).SubQuery()
		q = q.Filter(sqlchemy.OR(
			sqlchemy.In(q.Field("id"), gn),
			sqlchemy.In(q.Field("id"), guestEip),
		))
	}

	diskFilter := query.AttachableServersForDisk
	if len(diskFilter) > 0 {
		diskI, _ := DiskManager.FetchByIdOrName(userCred, diskFilter)
		if diskI == nil {
			return nil, httperrors.NewResourceNotFoundError("disk %s not found", diskFilter)
		}
		disk := diskI.(*SDisk)
		guestdisks := GuestdiskManager.Query().SubQuery()
		count, err := guestdisks.Query().Equals("disk_id", disk.Id).CountWithError()
		if err != nil {
			return nil, httperrors.NewInternalServerError("checkout guestdisk count fail %s", err)
		}
		if count > 0 {
			sgq := guestdisks.Query(guestdisks.Field("guest_id")).Equals("disk_id", disk.Id).SubQuery()
			q = q.Filter(sqlchemy.In(q.Field("id"), sgq))
		} else {
			hosts := HostManager.Query().SubQuery()
			hoststorages := HoststorageManager.Query().SubQuery()
			storages := StorageManager.Query().SubQuery()
			sq := hosts.Query(hosts.Field("id")).
				Join(hoststorages, sqlchemy.Equals(hoststorages.Field("host_id"), hosts.Field("id"))).
				Join(storages, sqlchemy.Equals(storages.Field("id"), hoststorages.Field("storage_id"))).
				Filter(sqlchemy.Equals(storages.Field("id"), disk.StorageId)).SubQuery()
			q = q.In("host_id", sq)
		}
	}

	withEip := (query.WithEip != nil && *query.WithEip)
	withoutEip := (query.WithoutEip != nil && *query.WithoutEip) || (query.EipAssociable != nil && *query.EipAssociable)
	if withEip || withoutEip {
		eips := ElasticipManager.Query().SubQuery()
		sq := eips.Query(eips.Field("associate_id")).Equals("associate_type", api.EIP_ASSOCIATE_TYPE_SERVER)
		sq = sq.IsNotNull("associate_id").IsNotEmpty("associate_id")

		if withEip {
			q = q.In("id", sq)
		} else if withoutEip {
			q = q.NotIn("id", sq)
		}
	}

	if query.EipAssociable != nil {
		sq1 := NetworkManager.Query("id")
		sq2 := WireManager.Query().SubQuery()
		sq3 := VpcManager.Query().SubQuery()
		sq1 = sq1.Join(sq2, sqlchemy.Equals(sq1.Field("wire_id"), sq2.Field("id")))
		sq1 = sq1.Join(sq3, sqlchemy.Equals(sq2.Field("vpc_id"), sq3.Field("id")))
		cond1 := []string{api.VPC_EXTERNAL_ACCESS_MODE_EIP, api.VPC_EXTERNAL_ACCESS_MODE_EIP_DISTGW}
		if *query.EipAssociable {
			sq1 = sq1.Filter(sqlchemy.In(sq3.Field("external_access_mode"), cond1))
		} else {
			sq1 = sq1.Filter(sqlchemy.NotIn(sq3.Field("external_access_mode"), cond1))
		}
		sq := GuestnetworkManager.Query("guest_id").In("network_id", sq1)
		q = q.In("id", sq)
	}

	if len(query.ServerType) > 0 {
		var trueVal, falseVal = true, false
		switch query.ServerType {
		case "normal":
			query.Gpu = &falseVal
			query.Backup = &falseVal
		case "gpu":
			query.Gpu = &trueVal
			query.Backup = &falseVal
		case "backup":
			query.Gpu = &falseVal
			query.Backup = &trueVal
		default:
			return nil, httperrors.NewInputParameterError("unknown server type %s", query.ServerType)
		}
	}

	if query.Backup != nil {
		if *query.Backup {
			q = q.IsNotEmpty("backup_host_id")
		} else {
			q = q.IsEmpty("backup_host_id")
		}
	}

	if query.Gpu != nil {
		isodev := IsolatedDeviceManager.Query().SubQuery()
		sgq := isodev.Query(isodev.Field("guest_id")).
			Filter(sqlchemy.AND(
				sqlchemy.IsNotNull(isodev.Field("guest_id")),
				sqlchemy.Startswith(isodev.Field("dev_type"), "GPU")))
		cond := sqlchemy.NotIn
		if *query.Gpu {
			cond = sqlchemy.In
		}
		q = q.Filter(cond(q.Field("id"), sgq))
	}

	groupFilter := query.GroupId
	if len(groupFilter) != 0 {
		groupObj, err := GroupManager.FetchByIdOrName(userCred, groupFilter)
		if err != nil {
			return nil, httperrors.NewNotFoundError("group %s not found", groupFilter)
		}
		// queryDict.Add(jsonutils.NewString(groupObj.GetId()), "group")
		ggSub := GroupguestManager.Query("guest_id").Equals("group_id", groupObj.GetId()).SubQuery()
		q = q.Join(ggSub, sqlchemy.Equals(ggSub.Field("guest_id"), q.Field("id")))
	}

	/*orderByDisk := query.OrderByDisk
	if orderByDisk == "asc" || orderByDisk == "desc" {
		guestdisks := GuestdiskManager.Query().SubQuery()
		disks := DiskManager.Query().SubQuery()
		guestdiskQ := guestdisks.Query(
			guestdisks.Field("guest_id"),
			sqlchemy.SUM("disks_size", disks.Field("disk_size")),
		)

		guestdiskQ = guestdiskQ.LeftJoin(disks, sqlchemy.Equals(guestdiskQ.Field("disk_id"), disks.Field("id")))
		guestdiskSQ := guestdiskQ.GroupBy(guestdiskQ.Field("guest_id")).SubQuery()

		q = q.LeftJoin(guestdiskSQ, sqlchemy.Equals(q.Field("id"), guestdiskSQ.Field("guest_id")))
		switch orderByDisk {
		case "asc":
			q = q.Asc(guestdiskSQ.Field("disks_size"))
		case "desc":
			q = q.Desc(guestdiskSQ.Field("disks_size"))
		}
	}*/

	if len(query.OsType) > 0 {
		q = q.In("os_type", query.OsType)
	}
	if len(query.VcpuCount) > 0 {
		q = q.In("vcpu_count", query.VcpuCount)
	}
	if len(query.VmemSize) > 0 {
		q = q.In("vmem_size", query.VmemSize)
	}
	if len(query.BootOrder) > 0 {
		q = q.In("boot_order", query.BootOrder)
	}
	if len(query.Vga) > 0 {
		q = q.In("vga", query.Vga)
	}
	if len(query.Vdi) > 0 {
		q = q.In("vdi", query.Vdi)
	}
	if len(query.Machine) > 0 {
		q = q.In("machine", query.Machine)
	}
	if len(query.Bios) > 0 {
		q = q.In("bios", query.Bios)
	}
	if query.SrcIpCheck != nil {
		if *query.SrcIpCheck {
			q = q.IsTrue("src_ip_check")
		} else {
			q = q.IsFalse("src_ip_check")
		}
	}
	if query.SrcMacCheck != nil {
		if *query.SrcMacCheck {
			q = q.IsTrue("src_mac_check")
		} else {
			q = q.IsFalse("src_mac_check")
		}
	}
	if len(query.InstanceType) > 0 {
		q = q.In("instance_type", query.InstanceType)
	}
	if query.WithHost != nil {
		if *query.WithHost {
			q = q.IsNotEmpty("host_id")
		} else {
			q = q.IsNullOrEmpty("host_id")
		}
	}

	return q, nil
}

func (manager *SGuestManager) ExtraSearchConditions(ctx context.Context, q *sqlchemy.SQuery, like string) []sqlchemy.ICondition {
	var sq *sqlchemy.SSubQuery
	if len(like) > 1 {
		switch {
		case strings.Contains(like, "."):
			sq = GuestnetworkManager.Query("guest_id").Contains("ip_addr", like).SubQuery()
		case strings.Contains(like, ":"):
			sq = GuestnetworkManager.Query("guest_id").Contains("mac_addr", like).SubQuery()
		}
	}
	if sq != nil {
		return []sqlchemy.ICondition{sqlchemy.In(q.Field("id"), sq)}
	}
	return nil
}

func (manager *SGuestManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.ServerListInput) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SHostResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.HostFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SHostResourceBaseManager.OrderByExtraFields")
	}
	fields := manager.SNetworkResourceBaseManager.GetOrderByFields(query.NetworkFilterListInput)
	if db.NeedOrderQuery(fields) {
		netQ := GuestnetworkManager.Query("guest_id", "network_id").SubQuery()
		q = q.LeftJoin(netQ, sqlchemy.Equals(q.Field("id"), netQ.Field("guest_id"))).Distinct()
		q, err = manager.SNetworkResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.NetworkFilterListInput)
		if err != nil {
			return nil, errors.Wrap(err, "SNetworkResourceBaseManager.OrderByExtraFields")
		}
	}

	if db.NeedOrderQuery([]string{query.OrderByDisk}) {
		guestdisks := GuestdiskManager.Query().SubQuery()
		disks := DiskManager.Query().SubQuery()
		guestdiskQ := guestdisks.Query(
			guestdisks.Field("guest_id"),
			sqlchemy.SUM("disks_size", disks.Field("disk_size")),
		)
		guestdiskQ = guestdiskQ.LeftJoin(disks, sqlchemy.Equals(guestdiskQ.Field("disk_id"), disks.Field("id")))
		guestdiskSQ := guestdiskQ.GroupBy(guestdiskQ.Field("guest_id")).SubQuery()

		q = q.LeftJoin(guestdiskSQ, sqlchemy.Equals(q.Field("id"), guestdiskSQ.Field("guest_id")))
		db.OrderByFields(q, []string{query.OrderByDisk}, []sqlchemy.IQueryField{guestdiskSQ.Field("disks_size")})
	}

	return q, nil
}

func (manager *SGuestManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SHostResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	guestnets := GuestnetworkManager.Query("guest_id", "network_id").SubQuery()
	q = q.LeftJoin(guestnets, sqlchemy.Equals(q.Field("id"), guestnets.Field("guest_id")))
	q, err = manager.SNetworkResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	guestdisks := GuestdiskManager.Query("guest_id", "disk_id").SubQuery()
	q = q.LeftJoin(guestdisks, sqlchemy.Equals(q.Field("id"), guestdisks.Field("guest_id")))
	q, err = manager.SDiskResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SGuestManager) InitializeData() error {
	guests := make([]SGuest, 0, 10)
	q := manager.Query().Equals("hypervisor", "esxi")
	err := db.FetchModelObjects(manager, q, &guests)
	if err != nil {
		return errors.Wrap(err, "db.FetchModelObjects")
	}
	// remove secgroup for esxi guest
	for i := range guests {
		if len(guests[i].SecgrpId) == 0 {
			continue
		}
		db.Update(&guests[i], func() error {
			guests[i].SecgrpId = ""
			return nil
		})
	}
	return nil
}

func (guest *SGuest) GetHypervisor() string {
	if len(guest.Hypervisor) == 0 {
		return api.HYPERVISOR_DEFAULT
	} else {
		return guest.Hypervisor
	}
}

func (guest *SGuest) GetHostType() string {
	return api.HYPERVISOR_HOSTTYPE[guest.Hypervisor]
}

func (guest *SGuest) GetDriver() IGuestDriver {
	hypervisor := guest.GetHypervisor()
	if !utils.IsInStringArray(hypervisor, api.HYPERVISORS) {
		log.Fatalf("Unsupported hypervisor %s", hypervisor)
	}
	return GetDriver(hypervisor)
}

func (guest *SGuest) validateDeleteCondition(ctx context.Context, isPurge bool) error {
	if guest.DisableDelete.IsTrue() {
		return httperrors.NewInvalidStatusError("Virtual server is locked, cannot delete")
	}
	if !isPurge && guest.IsNotDeletablePrePaid() {
		return httperrors.NewForbiddenError("not allow to delete prepaid server in valid status")
	}
	return guest.SVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (guest *SGuest) ValidatePurgeCondition(ctx context.Context) error {
	return guest.validateDeleteCondition(ctx, true)
}

func (guest *SGuest) ValidateDeleteCondition(ctx context.Context) error {
	host := guest.GetHost()
	if host != nil && guest.GetHypervisor() != api.HYPERVISOR_BAREMETAL {
		if !host.GetEnabled() {
			return httperrors.NewInputParameterError("Cannot delete server on disabled host")
		}
		if host.HostStatus != api.HOST_ONLINE {
			return httperrors.NewInputParameterError("Cannot delete server on offline host")
		}
	}

	if guest.GetHypervisor() == api.HYPERVISOR_HUAWEI {
		disks := guest.GetDisks()
		for _, disk := range disks {
			if snapshots := SnapshotManager.GetDiskSnapshots(disk.DiskId); len(snapshots) > 0 {
				return httperrors.NewResourceBusyError("Cannot delete server disk %s must not have snapshots.", disk.GetName())
			}
		}
	}

	return guest.validateDeleteCondition(ctx, false)
}

func (guest *SGuest) GetDisksQuery() *sqlchemy.SQuery {
	return GuestdiskManager.Query().Equals("guest_id", guest.Id)
}

func (guest *SGuest) DiskCount() (int, error) {
	return guest.GetDisksQuery().CountWithError()
}

func (guest *SGuest) GetSystemDisk() (*SDisk, error) {
	q := DiskManager.Query().Equals("disk_type", api.DISK_TYPE_SYS)
	gs := GuestdiskManager.Query().SubQuery()
	q = q.Join(gs, sqlchemy.Equals(gs.Field("disk_id"), q.Field("id"))).
		Filter(sqlchemy.Equals(gs.Field("guest_id"), guest.Id))

	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	}
	if count == 0 {
		return nil, sql.ErrNoRows
	}
	disk := &SDisk{}
	err = q.First(disk)
	if err != nil {
		return nil, errors.Wrap(err, "q.First(disk)")
	}
	disk.SetModelManager(DiskManager, disk)
	return disk, nil
}

func (guest *SGuest) GetDisks() []SGuestdisk {
	disks := make([]SGuestdisk, 0)
	q := guest.GetDisksQuery().Asc("index")
	err := db.FetchModelObjects(GuestdiskManager, q, &disks)
	if err != nil {
		log.Errorf("Getdisks error: %s", err)
	}
	return disks
}

func (guest *SGuest) GetGuestDisk(diskId string) *SGuestdisk {
	guestdisk, err := db.NewModelObject(GuestdiskManager)
	if err != nil {
		log.Errorf("new guestdisk model failed: %s", err)
		return nil
	}
	q := guest.GetDisksQuery()
	err = q.Equals("disk_id", diskId).First(guestdisk)
	if err != nil {
		log.Errorf("GetGuestDisk error: %s", err)
		return nil
	}
	return guestdisk.(*SGuestdisk)
}

func (guest *SGuest) GetNetworksQuery(netId string) *sqlchemy.SQuery {
	q := GuestnetworkManager.Query().Equals("guest_id", guest.Id)
	if len(netId) > 0 {
		q = q.Equals("network_id", netId)
	}
	return q
}

func (guest *SGuest) NetworkCount() (int, error) {
	return guest.GetNetworksQuery("").CountWithError()
}

func (guest *SGuest) GetVpc() (*SVpc, error) {
	q := guest.GetNetworksQuery("")
	guestnic := &SGuestnetwork{}
	err := q.First(guestnic)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting guest network of %s(%s)", guest.Name, guest.Id)
	}
	guestnic.SetModelManager(GuestnetworkManager, guestnic)
	network := guestnic.GetNetwork()
	if network == nil {
		return nil, errors.Wrapf(err, "failed getting network for guest %s(%s)", guest.Name, guest.Id)
	}
	vpc := network.GetVpc()
	if vpc == nil {
		return nil, errors.Wrapf(err, "failed getting vpc of guest network %s(%s)", network.Name, network.Id)
	}
	return vpc, nil
}

func (guest *SGuest) GetNetworks(netId string) ([]SGuestnetwork, error) {
	guestnics := make([]SGuestnetwork, 0)
	q := guest.GetNetworksQuery(netId).Asc("index")
	err := db.FetchModelObjects(GuestnetworkManager, q, &guestnics)
	if err != nil {
		log.Errorf("GetNetworks error: %s", err)
		return nil, err
	}
	return guestnics, nil
}

func (guest *SGuest) ConvertNetworks(targetGuest *SGuest) error {
	gns, err := guest.GetNetworks("")
	if err != nil {
		return err
	}
	var i int
	for ; i < len(gns); i++ {
		_, err = db.Update(&gns[i], func() error {
			gns[i].GuestId = targetGuest.Id
			return nil
		})
		if err != nil {
			log.Errorf("update guestnetworks failed %s", err)
			break
		}
	}
	if err != nil {
		for j := 0; j < i; j++ {
			_, err = db.Update(&gns[j], func() error {
				gns[j].GuestId = guest.Id
				return nil
			})
			if err != nil {
				log.Errorf("update guestnetworks failed %s", err)
				break
			}
		}
	}
	return err
}

func (guest *SGuest) getGuestnetworkByIpOrMac(ipAddr string, macAddr string) (*SGuestnetwork, error) {
	q := guest.GetNetworksQuery("")
	if len(ipAddr) > 0 {
		q = q.Equals("ip_addr", ipAddr)
	}
	if len(macAddr) > 0 {
		q = q.Equals("mac_addr", macAddr)
	}

	guestnic := SGuestnetwork{}
	err := q.First(&guestnic)
	if err != nil {
		return nil, err
	}
	guestnic.SetModelManager(GuestnetworkManager, &guestnic)
	return &guestnic, nil
}

func (guest *SGuest) GetGuestnetworkByIp(ipAddr string) (*SGuestnetwork, error) {
	return guest.getGuestnetworkByIpOrMac(ipAddr, "")
}

func (guest *SGuest) GetGuestnetworkByMac(macAddr string) (*SGuestnetwork, error) {
	return guest.getGuestnetworkByIpOrMac("", macAddr)
}

func (guest *SGuest) IsNetworkAllocated() bool {
	guestnics, err := guest.GetNetworks("")
	if err != nil {
		return false
	}
	for _, gn := range guestnics {
		if !gn.IsAllocated() {
			return false
		}
	}
	return true
}

func (guest *SGuest) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	guest.HostId = ""
	return guest.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (guest *SGuest) GetCloudproviderId() string {
	host := guest.GetHost()
	if host != nil {
		return host.GetCloudproviderId()
	}
	return ""
}

func (guest *SGuest) GetHost() *SHost {
	if len(guest.HostId) > 0 && regutils.MatchUUID(guest.HostId) {
		host, _ := HostManager.FetchById(guest.HostId)
		if host != nil {
			return host.(*SHost)
		}
	}
	return nil
}

func (guest *SGuest) SetHostId(userCred mcclient.TokenCredential, hostId string) error {
	if guest.HostId != hostId {
		diff, err := db.Update(guest, func() error {
			guest.HostId = hostId
			return nil
		})
		if err != nil {
			return err
		}
		db.OpsLog.LogEvent(guest, db.ACT_UPDATE, diff, userCred)
	}
	return nil
}

func (guest *SGuest) SetHostIdWithBackup(userCred mcclient.TokenCredential, master, slave string) error {
	diff, err := db.Update(guest, func() error {
		guest.HostId = master
		guest.BackupHostId = slave
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(guest, db.ACT_UPDATE, diff, userCred)
	return err
}

func (guest *SGuest) ValidateResizeDisk(disk *SDisk, storage *SStorage) error {
	return guest.GetDriver().ValidateResizeDisk(guest, disk, storage)
}

func ValidateMemData(vmemSize int, driver IGuestDriver) (int, error) {
	if vmemSize > 0 {
		maxVmemGb := driver.GetMaxVMemSizeGB()
		if vmemSize < 8 || vmemSize > maxVmemGb*1024 {
			return 0, httperrors.NewInputParameterError("Memory size must be 8MB ~ %d GB", maxVmemGb)
		}
	}
	return vmemSize, nil
}

func ValidateCpuData(vcpuCount int, driver IGuestDriver) (int, error) {
	maxVcpuCount := driver.GetMaxVCpuCount()
	if vcpuCount < 1 || vcpuCount > maxVcpuCount {
		return 0, httperrors.NewInputParameterError("CPU core count must be 1 ~ %d", maxVcpuCount)
	}
	return vcpuCount, nil
}

func ValidateMemCpuData(vmemSize, vcpuCount int, hypervisor string) (int, int, error) {
	if len(hypervisor) == 0 {
		hypervisor = api.HYPERVISOR_DEFAULT
	}
	driver := GetDriver(hypervisor)

	var err error
	vmemSize, err = ValidateMemData(vmemSize, driver)
	if err != nil {
		return 0, 0, err
	}

	vcpuCount, err = ValidateCpuData(vcpuCount, driver)
	if err != nil {
		return 0, 0, err
	}
	return vmemSize, vcpuCount, nil
}

func (self *SGuest) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	var err error
	var vmemSize int
	var vcpuCount int

	driver := GetDriver(self.Hypervisor)

	if memSize, _ := data.Int("vmem_size"); memSize != 0 {
		vmemSize, err = ValidateMemData(int(memSize), driver)
		if err != nil {
			return nil, err
		}
	}
	if cpuCount, _ := data.Int("vcpu_count"); cpuCount != 0 {
		vcpuCount, err = ValidateCpuData(int(cpuCount), driver)
		if err != nil {
			return nil, err
		}
	}

	if vmemSize > 0 || vcpuCount > 0 {
		if !utils.IsInStringArray(self.Status, []string{api.VM_READY}) && self.GetHypervisor() != api.HYPERVISOR_CONTAINER {
			return nil, httperrors.NewInvalidStatusError("Cannot modify Memory and CPU in status %s", self.Status)
		}
		if self.GetHypervisor() == api.HYPERVISOR_BAREMETAL {
			return nil, httperrors.NewInputParameterError("Cannot modify memory for baremetal")
		}
	}

	if vmemSize > 0 {
		data.Add(jsonutils.NewInt(int64(vmemSize)), "vmem_size")
	}
	if vcpuCount > 0 {
		data.Add(jsonutils.NewInt(int64(vcpuCount)), "vcpu_count")
	}

	data, err = self.GetDriver().ValidateUpdateData(ctx, userCred, data)
	if err != nil {
		return nil, err
	}

	if vcpuCount > 0 || vmemSize > 0 {
		quota, err := self.checkUpdateQuota(ctx, userCred, vcpuCount, vmemSize)
		if err != nil {
			return nil, httperrors.NewOutOfQuotaError("%v", err)
		}
		if !quota.IsEmpty() {
			data.Add(jsonutils.Marshal(quota), "pending_usage")
		}
	}

	if data.Contains("name") {
		if name, _ := data.GetString("name"); len(name) < 2 {
			return nil, httperrors.NewInputParameterError("name is too short")
		}
	}
	input := apis.VirtualResourceBaseUpdateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, errors.Wrap(err, "data.Unmarshal")
	}
	input, err = self.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}
	data.Update(jsonutils.Marshal(input))
	return data, nil
}

func serverCreateInput2ComputeQuotaKeys(input api.ServerCreateInput, ownerId mcclient.IIdentityProvider) SComputeResourceKeys {
	// input.Hypervisor must be set
	brand := guessBrandForHypervisor(input.Hypervisor)
	keys := GetDriver(input.Hypervisor).GetComputeQuotaKeys(
		rbacutils.ScopeProject,
		ownerId,
		brand,
	)
	if len(input.PreferHost) > 0 {
		hostObj, _ := HostManager.FetchById(input.PreferHost)
		host := hostObj.(*SHost)
		zone := host.GetZone()
		keys.ZoneId = zone.Id
		keys.RegionId = zone.CloudregionId
	} else if len(input.PreferZone) > 0 {
		zoneObj, _ := ZoneManager.FetchById(input.PreferZone)
		zone := zoneObj.(*SZone)
		keys.ZoneId = zone.Id
		keys.RegionId = zone.CloudregionId
	} else if len(input.PreferWire) > 0 {
		wireObj, _ := WireManager.FetchById(input.PreferWire)
		wire := wireObj.(*SWire)
		zone := wire.GetZone()
		keys.ZoneId = zone.Id
		keys.RegionId = zone.CloudregionId
	} else if len(input.PreferRegion) > 0 {
		regionObj, _ := CloudregionManager.FetchById(input.PreferRegion)
		keys.RegionId = regionObj.GetId()
	}
	return keys
}

func (manager *SGuestManager) BatchPreValidate(
	ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject, data *jsonutils.JSONDict, count int,
) error {
	input, err := manager.validateCreateData(ctx, userCred, ownerId, query, data)
	if err != nil {
		return errors.Wrap(err, "manager.validateCreateData")
	}
	if input.IsSystem == nil || !(*input.IsSystem) {
		err := manager.checkCreateQuota(ctx, userCred, ownerId, *input, input.Backup, count)
		if err != nil {
			return errors.Wrap(err, "manager.checkCreateQuota")
		}
	}
	return nil
}

func parseInstanceSnapshot(input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	ispi, err := InstanceSnapshotManager.FetchByIdOrName(nil, input.InstanceSnapshotId)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewBadRequestError("can't find instance snapshot %s", input.InstanceSnapshotId)
	}
	if err != nil {
		return nil, httperrors.NewInternalServerError("fetch instance snapshot error %s", err)
	}
	isp := ispi.(*SInstanceSnapshot)
	if isp.Status != api.INSTANCE_SNAPSHOT_READY {
		return nil, httperrors.NewBadRequestError("Instance snapshot not ready")
	}
	return isp.ToInstanceCreateInput(input)
}

func (manager *SGuestManager) validateCreateData(
	ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject, data *jsonutils.JSONDict) (*api.ServerCreateInput, error) {
	// TODO: 定义 api.ServerCreateInput 的 Unmarshal 函数，直接通过 data.Unmarshal(input) 解析参数
	input, err := cmdline.FetchServerCreateInputByJSON(data)
	if err != nil {
		return nil, err
	}

	if len(input.Metadata) > 20 {
		return nil, httperrors.NewInputParameterError("metdata must less then 20")
	}

	if len(input.InstanceSnapshotId) > 0 {
		inputMem := input.VmemSize
		inputCpu := input.VcpuCount
		inputInstaceType := input.InstanceType
		input, err = parseInstanceSnapshot(input)
		if err != nil {
			return nil, err
		}
		// keep input cpu mem flavor
		if inputMem > 0 {
			input.VmemSize = inputMem
		}
		if inputMem > 0 {
			input.VcpuCount = inputCpu
		}
		input.InstanceType = inputInstaceType
	}

	resetPassword := true
	if input.ResetPassword != nil {
		resetPassword = *input.ResetPassword
	}

	passwd := input.Password
	if len(passwd) > 0 {
		err = seclib2.ValidatePassword(passwd)
		if err != nil {
			return nil, err
		}
		resetPassword = true
		input.ResetPassword = &resetPassword
	}

	if resetPassword && len(input.LoginAccount) > 0 {
		if len(input.LoginAccount) > 32 {
			return nil, httperrors.NewInputParameterError("login_account is longer than 32 chars")
		}
		if err := manager.ValidateNameLoginAccount(input.LoginAccount); err != nil {
			return nil, err
		}
	}

	// check group
	if input.InstanceGroupIds != nil && len(input.InstanceGroupIds) != 0 {
		newGroupIds := make([]string, len(input.InstanceGroupIds))
		for index, id := range input.InstanceGroupIds {
			model, err := GroupManager.FetchByIdOrName(userCred, id)
			if err != nil {
				return nil, httperrors.NewResourceNotFoundError("no such group %s", id)
			}
			newGroupIds[index] = model.GetId()
		}
		// list of id or name ==> ids
		input.InstanceGroupIds = newGroupIds
	}

	// check that all image of disk is the part of guest imgae, if use guest image to create guest
	err = manager.checkGuestImage(ctx, input)
	if err != nil {
		return nil, err
	}

	var hypervisor string
	// var rootStorageType string
	var osProf osprofile.SOSProfile
	hypervisor = input.Hypervisor
	if hypervisor != api.HYPERVISOR_CONTAINER {
		if len(input.Disks) == 0 {
			return nil, httperrors.NewInputParameterError("No disk information provided")
		}
		diskConfig := input.Disks[0]
		diskConfig, err = parseDiskInfo(ctx, userCred, diskConfig)
		if err != nil {
			return nil, httperrors.NewInputParameterError("Invalid root image: %s", err)
		}

		// if len(diskConfig.Backend) == 0 {
		// 	diskConfig.Backend = STORAGE_LOCAL
		// }
		// rootStorageType = diskConfig.Backend

		input.Disks[0] = diskConfig

		imgProperties := diskConfig.ImageProperties
		imgSupportUEFI := imgProperties[imageapi.IMAGE_UEFI_SUPPORT] == "true"
		imgIsWindows := imgProperties[imageapi.IMAGE_OS_TYPE] == "Windows"
		if imgSupportUEFI && imgIsWindows && len(input.IsolatedDevices) > 0 {
			input.Bios = "UEFI" // windows gpu passthrough
		}

		if imgProperties[imageapi.IMAGE_DISK_FORMAT] == "iso" {
			return nil, httperrors.NewInputParameterError("System disk does not support iso image, please consider using cdrom parameter")
		}

		if input.Cdrom != "" {
			cdromStr := input.Cdrom
			image, err := parseIsoInfo(ctx, userCred, cdromStr)
			if err != nil {
				return nil, httperrors.NewInputParameterError("parse cdrom device info error %s", err)
			}
			input.Cdrom = image.Id
			if len(imgProperties) == 0 {
				imgProperties = image.Properties
			}
		}

		if arch := imgProperties["os_arch"]; strings.Contains(arch, "aarch") {
			input.OsArch = apis.OS_ARCH_AARCH64
		}

		if len(imgProperties) == 0 {
			imgProperties = map[string]string{"os_type": "Linux"}
		}
		input.DisableUsbKbd = imgProperties[imageapi.IMAGE_DISABLE_USB_KBD] == "true"

		osType := input.OsType
		osProf, err = osprofile.GetOSProfileFromImageProperties(imgProperties, hypervisor)
		if err != nil {
			return nil, httperrors.NewInputParameterError("Invalid root image: %s", err)
		}

		if len(osProf.Hypervisor) > 0 && len(hypervisor) == 0 {
			hypervisor = osProf.Hypervisor
			input.Hypervisor = hypervisor
		}
		if len(osProf.OSType) > 0 && len(osType) == 0 {
			osType = osProf.OSType
			input.OsType = osType
		}
		input.OsProfile = jsonutils.Marshal(osProf)
	}

	input, err = ValidateScheduleCreateData(ctx, userCred, input, hypervisor)
	if err != nil {
		return nil, err
	}

	optionSystemHypervisor := []string{api.HYPERVISOR_KVM, api.HYPERVISOR_ESXI}

	if !utils.IsInStringArray(input.Hypervisor, optionSystemHypervisor) && len(input.Disks[0].ImageId) == 0 && len(input.Disks[0].SnapshotId) == 0 && input.Cdrom == "" {
		return nil, httperrors.NewBadRequestError("Miss operating system???")
	}

	hypervisor = input.Hypervisor
	if hypervisor != api.HYPERVISOR_CONTAINER {
		// support sku here
		var sku *SServerSku
		skuName := input.InstanceType
		if len(skuName) > 0 {
			provider := GetDriver(input.Hypervisor).GetProvider()
			sku, err = ServerSkuManager.FetchSkuByNameAndProvider(skuName, provider, true)
			if err != nil {
				return nil, err
			}

			input.InstanceType = sku.Name
			input.VmemSize = sku.MemorySizeMB
			input.VcpuCount = sku.CpuCoreCount
		} else {
			vmemSize, vcpuCount, err := ValidateMemCpuData(input.VmemSize, input.VcpuCount, input.Hypervisor)
			if err != nil {
				return nil, err
			}

			if vmemSize == 0 {
				return nil, httperrors.NewMissingParameterError("vmem_size")
			}
			if vcpuCount == 0 {
				vcpuCount = 1
			}
			input.VmemSize = vmemSize
			input.VcpuCount = vcpuCount
		}

		dataDiskDefs := []*api.DiskConfig{}
		if sku != nil && sku.AttachedDiskCount > 0 {
			for i := 0; i < sku.AttachedDiskCount; i += 1 {
				dataDisk := &api.DiskConfig{
					SizeMb:  sku.AttachedDiskSizeGB * 1024,
					Backend: strings.ToLower(sku.AttachedDiskType),
				}
				dataDiskDefs = append(dataDiskDefs, dataDisk)
			}
		}

		// start from data disk
		disks := input.Disks
		for idx := 1; idx < len(disks); idx += 1 {
			dataDiskDefs = append(dataDiskDefs, disks[idx])
		}

		rootDiskConfig, err := parseDiskInfo(ctx, userCred, disks[0])
		if err != nil {
			return nil, httperrors.NewGeneralError(err) // should no error
		}
		if input.ResourceType != api.HostResourceTypePrepaidRecycle {
			if len(rootDiskConfig.Backend) == 0 {
				defaultStorageType, _ := data.GetString("default_storage_type")
				if len(defaultStorageType) > 0 {
					rootDiskConfig.Backend = defaultStorageType
				} else {
					rootDiskConfig.Backend = GetDriver(hypervisor).GetDefaultSysDiskBackend()
				}
			}
			sysMinDiskMB := GetDriver(hypervisor).GetMinimalSysDiskSizeGb() * 1024
			if rootDiskConfig.SizeMb != api.DISK_SIZE_AUTOEXTEND && rootDiskConfig.SizeMb < sysMinDiskMB {
				rootDiskConfig.SizeMb = sysMinDiskMB
			}
		}
		if len(rootDiskConfig.Driver) == 0 {
			rootDiskConfig.Driver = osProf.DiskDriver
		}
		log.Debugf("ROOT DISK: %#v", rootDiskConfig)
		input.Disks[0] = rootDiskConfig
		if sku != nil {
			if len(rootDiskConfig.OsArch) > 0 && len(sku.CpuArch) > 0 {
				if !strings.Contains(rootDiskConfig.OsArch, sku.CpuArch) {
					return nil, httperrors.NewConflictError("root disk image(%s) and sku(%s) architecture mismatch", rootDiskConfig.OsArch, sku.CpuArch)
				}
			}
		}
		//data.Set("disk.0", jsonutils.Marshal(rootDiskConfig))

		for i := 0; i < len(dataDiskDefs); i += 1 {
			diskConfig, err := parseDiskInfo(ctx, userCred, dataDiskDefs[i])
			if err != nil {
				return nil, httperrors.NewInputParameterError("parse disk description error %s", err)
			}
			if diskConfig.DiskType == api.DISK_TYPE_SYS {
				return nil, httperrors.NewBadRequestError("Snapshot error: disk index %d > 0 but disk type is %s", i+1, api.DISK_TYPE_SYS)
			}
			if len(diskConfig.Backend) == 0 {
				diskConfig.Backend = rootDiskConfig.Backend
			}
			if len(diskConfig.Driver) == 0 {
				diskConfig.Driver = osProf.DiskDriver
			}
			input.Disks[i+1] = diskConfig
		}

		if len(input.Duration) > 0 {

			/*if !userCred.IsAllow(rbacutils.ScopeSystem, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionPerform, "renew") {
				return nil, httperrors.NewForbiddenError("only admin can create prepaid resource")
			}*/

			if input.ResourceType == api.HostResourceTypePrepaidRecycle {
				return nil, httperrors.NewConflictError("cannot create prepaid server on prepaid resource type")
			}

			billingCycle, err := billing.ParseBillingCycle(input.Duration)
			if err != nil {
				return nil, httperrors.NewInputParameterError("invalid duration %s", input.Duration)
			}

			if input.BillingType == billing_api.BILLING_TYPE_POSTPAID {
				if !GetDriver(hypervisor).IsSupportPostpaidExpire() {
					return nil, httperrors.NewBadRequestError("guest %s unsupport postpaid expire", hypervisor)
				}
			} else {
				if !GetDriver(hypervisor).IsSupportedBillingCycle(billingCycle) {
					return nil, httperrors.NewInputParameterError("unsupported duration %s", input.Duration)
				}
			}

			if len(input.BillingType) == 0 {
				input.BillingType = billing_api.BILLING_TYPE_PREPAID
			}
			input.BillingCycle = billingCycle.String()
			// expired_at will be set later by callback
			// data.Add(jsonutils.NewTimeString(billingCycle.EndAt(time.Time{})), "expired_at")

			input.Duration = billingCycle.String()
		}
	}

	// HACK: if input networks is empty, add one random network config
	if len(input.Networks) == 0 {
		input.Networks = append(input.Networks, &api.NetworkConfig{Exit: false})
	}
	netArray := input.Networks
	for idx := 0; idx < len(netArray); idx += 1 {
		netConfig, err := parseNetworkInfo(userCred, netArray[idx])
		if err != nil {
			return nil, httperrors.NewInputParameterError("parse network description error %s", err)
		}
		err = isValidNetworkInfo(userCred, netConfig)
		if err != nil {
			return nil, err
		}
		if len(netConfig.Driver) == 0 {
			netConfig.Driver = osProf.NetDriver
		}
		netConfig.Project = ownerId.GetProjectId()
		netConfig.Domain = ownerId.GetProjectDomainId()
		input.Networks[idx] = netConfig
	}

	isoDevArray := input.IsolatedDevices
	for idx := 0; idx < len(isoDevArray); idx += 1 { // .Contains(fmt.Sprintf("isolated_device.%d", idx)); idx += 1 {
		if input.Backup {
			return nil, httperrors.NewBadRequestError("Cannot create backup with isolated device")
		}
		devConfig, err := IsolatedDeviceManager.parseDeviceInfo(userCred, isoDevArray[idx])
		if err != nil {
			return nil, httperrors.NewInputParameterError("parse isolated device description error %s", err)
		}
		err = IsolatedDeviceManager.isValidDeviceinfo(devConfig)
		if err != nil {
			return nil, err
		}
		input.IsolatedDevices[idx] = devConfig
	}

	keypairId := input.KeypairId
	if len(keypairId) > 0 {
		keypairObj, err := KeypairManager.FetchByIdOrName(userCred, keypairId)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("Keypair %s not found", keypairId)
		}
		input.KeypairId = keypairObj.GetId()
	}

	secGrpIds := []string{}
	for _, secgroup := range input.Secgroups {
		secGrpObj, err := SecurityGroupManager.FetchByIdOrName(userCred, secgroup)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("Secgroup %s not found", secgroup)
		}
		if !utils.IsInStringArray(secGrpObj.GetId(), secGrpIds) {
			secGrpIds = append(secGrpIds, secGrpObj.GetId())
		}
	}
	if len(secGrpIds) > 0 {
		input.SecgroupId = secGrpIds[0]
		input.Secgroups = secGrpIds[1:]
	} else if input.SecgroupId != "" {
		secGrpId := input.SecgroupId
		secGrpObj, err := SecurityGroupManager.FetchByIdOrName(userCred, secGrpId)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("Secgroup %s not found", secGrpId)
		}
		input.SecgroupId = secGrpObj.GetId()
	} else {
		input.SecgroupId = "default"
	}

	maxSecgrpCount := GetDriver(hypervisor).GetMaxSecurityGroupCount()
	if maxSecgrpCount == 0 { //esxi 不支持安全组
		input.SecgroupId = ""
		input.Secgroups = []string{}
	} else if len(input.Secgroups)+1 > maxSecgrpCount {
		return nil, httperrors.NewInputParameterError("%s shall bind up to %d security groups", hypervisor, maxSecgrpCount)
	}

	preferRegionId, _ := data.GetString("prefer_region_id")
	if err := manager.validateEip(userCred, input, preferRegionId, input.PreferManager); err != nil {
		return nil, err
	}

	/*
		TODO
		group
		for idx := 0; data.Contains(fmt.Sprintf("srvtag.%d", idx)); idx += 1 {

		}*/

	if input.ResourceType != api.HostResourceTypePrepaidRecycle {
		input, err = GetDriver(hypervisor).ValidateCreateData(ctx, userCred, input)
		if err != nil {
			return nil, err
		}
	}

	input.VirtualResourceCreateInput, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return nil, err
	}

	if err := userdata.ValidateUserdata(input.UserData); err != nil {
		return nil, httperrors.NewInputParameterError("Invalid userdata: %v", err)
	}

	err = manager.ValidatePolicyDefinitions(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}

	input.ProjectId = ownerId.GetProjectId()
	input.ProjectDomainId = ownerId.GetProjectDomainId()
	return input, nil
}

func (manager *SGuestManager) ValidatePolicyDefinitions(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.ServerCreateInput) error {
	definitions, err := PolicyDefinitionManager.GetAvailablePolicyDefinitions(ctx, userCred)
	if err != nil {
		return httperrors.NewGeneralError(err)
	}
	for i := range definitions {
		switch definitions[i].Category {
		case api.POLICY_DEFINITION_CATEGORY_CLOUDREGION:
			if len(input.PreferRegion) == 0 {
				return httperrors.NewMissingParameterError(fmt.Sprintf("policy definition %s require prefer_region_id parameter", definitions[i].Name))
			}
			if definitions[i].Parameters == nil {
				return httperrors.NewPolicyDefinitionError("invalid parameters for policy definition %s", definitions[i].Name)
			}
			regionDefinitions := api.SCloudregionPolicyDefinitions{}
			definitions[i].Parameters.Unmarshal(&regionDefinitions)
			regions := []string{}
			for _, region := range regionDefinitions.Cloudregions {
				regions = append(regions, region.Id)
				regions = append(regions, region.Name)
			}
			isIn := utils.IsInStringArray(input.PreferRegion, regions)
			switch definitions[i].Condition {
			case api.POLICY_DEFINITION_CONDITION_IN:
				if !isIn {
					return httperrors.NewPolicyDefinitionError("policy definition %s require cloudregion in %s", definitions[i].Name, definitions[i].Parameters)
				}
			case api.POLICY_DEFINITION_CONDITION_NOT_IN:
				if isIn {
					return httperrors.NewPolicyDefinitionError("policy definition %s require cloudregion not in %s", definitions[i].Name, definitions[i].Parameters)
				}
			default:
				return httperrors.NewPolicyDefinitionError("invalid policy definition %s(%s) condition %s", definitions[i].Name, definitions[i].Id, definitions[i].Condition)
			}
		case api.POLICY_DEFINITION_CATEGORY_TAG:
			tags := []string{}
			if definitions[i].Parameters == nil {
				return httperrors.NewPolicyDefinitionError("invalid parameters for policy definition %s", definitions[i].Name)
			}
			definitions[i].Parameters.Unmarshal(&tags, "tags")
			metadataKeys := []string{}
			for k, _ := range input.Metadata {
				metadataKeys = append(metadataKeys, strings.TrimPrefix(k, db.USER_TAG_PREFIX))
			}
			for _, tag := range tags {
				isIn := utils.IsInStringArray(tag, metadataKeys)
				switch definitions[i].Condition {
				case api.POLICY_DEFINITION_CONDITION_CONTAINS:
					if !isIn {
						return httperrors.NewPolicyDefinitionError("policy definition %s require must contains tag %s", definitions[i].Name, tag)
					}
				case api.POLICY_DEFINITION_CONDITION_EXCEPT:
					if isIn {
						return httperrors.NewPolicyDefinitionError("policy definition %s require except tag %s", definitions[i].Name, tag)
					}
				default:
					return httperrors.NewPolicyDefinitionError("invalid policy definition %s(%s) condition %s", definitions[i].Name, definitions[i].Id, definitions[i].Condition)
				}
			}
		default:
			return httperrors.NewPolicyDefinitionError("invalid category %s for policy definition %s(%s)", definitions[i].Category, definitions[i].Name, definitions[i].Id)
		}
	}
	return nil
}

func (manager *SGuestManager) BatchCreateValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	input, err := manager.validateCreateData(ctx, userCred, ownerId, query, data)
	if err != nil {
		return nil, err
	}
	return input.JSON(input), nil
}

// 创建虚拟机实例
func (manager *SGuestManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, oinput api.ServerCreateInput) (*jsonutils.JSONDict, error) {
	input, err := manager.validateCreateData(ctx, userCred, ownerId, query, oinput.JSON(oinput))
	if err != nil {
		return nil, err
	}
	if input.IsSystem == nil || !(*input.IsSystem) {
		err = manager.checkCreateQuota(ctx, userCred, ownerId, *input, input.Backup, 1)
		if err != nil {
			return nil, err
		}
	}

	return input.JSON(input), nil
}

func (manager *SGuestManager) validateEip(userCred mcclient.TokenCredential, input *api.ServerCreateInput,
	preferRegionId string, preferManagerId string) error {
	if input.PublicIpBw > 0 {
		if !GetDriver(input.Hypervisor).IsSupportPublicIp() {
			return httperrors.NewNotImplementedError("public ip not supported for %s", input.Hypervisor)
		}
		if len(input.PublicIpChargeType) == 0 {
			input.PublicIpChargeType = string(cloudprovider.ElasticipChargeTypeByTraffic)
		}
		if !utils.IsInStringArray(input.PublicIpChargeType, []string{
			string(cloudprovider.ElasticipChargeTypeByTraffic),
			string(cloudprovider.ElasticipChargeTypeByBandwidth),
		}) {
			return httperrors.NewInputParameterError("invalid public_ip_charge_type %s", input.PublicIpChargeType)
		}
		return nil
	}
	eipStr := input.Eip
	eipBw := input.EipBw
	if len(eipStr) > 0 || eipBw > 0 {
		if !GetDriver(input.Hypervisor).IsSupportEip() {
			return httperrors.NewNotImplementedError("eip not supported for %s", input.Hypervisor)
		}
		if len(eipStr) > 0 {
			eipObj, err := ElasticipManager.FetchByIdOrName(userCred, eipStr)
			if err != nil {
				if err == sql.ErrNoRows {
					return httperrors.NewResourceNotFoundError2(ElasticipManager.Keyword(), eipStr)
				} else {
					return httperrors.NewGeneralError(err)
				}
			}

			eip := eipObj.(*SElasticip)
			if eip.Status != api.EIP_STATUS_READY {
				return httperrors.NewInvalidStatusError("eip %s status invalid %s", eipStr, eip.Status)
			}
			if eip.IsAssociated() {
				return httperrors.NewResourceBusyError("eip %s has been associated", eipStr)
			}
			input.Eip = eipObj.GetId()

			eipCloudprovider := eip.GetCloudprovider()
			if len(preferManagerId) > 0 && preferManagerId != eipCloudprovider.Id {
				return httperrors.NewConflictError("cannot assoicate with eip %s: different cloudprovider", eipStr)
			}
			input.PreferManager = eipCloudprovider.Id

			eipRegion, err := eip.GetRegion()
			if err != nil {
				return httperrors.NewGeneralError(errors.Wrapf(err, "eip.GetRegion"))
			}
			// preferRegionId, _ := data.GetString("prefer_region_id")
			if len(preferRegionId) > 0 && preferRegionId != eipRegion.Id {
				return httperrors.NewConflictError("cannot assoicate with eip %s: different region", eipStr)
			}
			input.PreferRegion = eipRegion.Id
		} else {
			// create new eip
		}
	}
	return nil
}

func (self *SGuest) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SVirtualResourceBase.PostUpdate(ctx, userCred, query, data)

	if data.Contains("pending_usage") {
		quota := SQuota{}
		data.Unmarshal(&quota, "pending_usage")
		quotas.CancelPendingUsage(ctx, userCred, &quota, &quota, true)
	}

	self.StartSyncTask(ctx, userCred, true, "")

	if data.Contains("name") || data.Contains("__meta__") {
		err := self.StartRemoteUpdateTask(ctx, userCred, false, "")
		if err != nil {
			log.Errorf("StartRemoteUpdateTask fail: %s", err)
		}
	}
	// notify webhook
	notifyclient.NotifyWebhook(ctx, userCred, self, notifyclient.ActionUpdate)
}

func (manager *SGuestManager) checkCreateQuota(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	input api.ServerCreateInput,
	hasBackup bool,
	count int,
) error {
	req, regionReq := getGuestResourceRequirements(ctx, userCred, input, ownerId, count, hasBackup)
	log.Debugf("computeQuota: %s", jsonutils.Marshal(req))
	log.Debugf("regionQuota: %s", jsonutils.Marshal(regionReq))

	err := quotas.CheckSetPendingQuota(ctx, userCred, &req)
	if err != nil {
		return errors.Wrap(err, "quotas.CheckSetPendingQuota")
	}
	err = quotas.CheckSetPendingQuota(ctx, userCred, &regionReq)
	if err != nil {
		return errors.Wrap(err, "quotas.CheckSetPendingQuota")
	}
	return nil
}

func (self *SGuest) checkUpdateQuota(ctx context.Context, userCred mcclient.TokenCredential, vcpuCount int, vmemSize int) (quotas.IQuota, error) {
	req := SQuota{}

	if vcpuCount > 0 && vcpuCount > int(self.VcpuCount) {
		req.Cpu = vcpuCount - int(self.VcpuCount)
	}

	if vmemSize > 0 && vmemSize > self.VmemSize {
		req.Memory = vmemSize - self.VmemSize
	}

	keys, err := self.GetQuotaKeys()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetQuotaKeys")
	}
	req.SetKeys(keys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, &req)
	if err != nil {
		return nil, errors.Wrap(err, "quotas.CheckSetPendingQuota")
	}

	return &req, nil
}

func getGuestResourceRequirements(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input api.ServerCreateInput,
	ownerId mcclient.IIdentityProvider,
	count int,
	hasBackup bool,
) (SQuota, SRegionQuota) {
	vcpuCount := input.VcpuCount
	if vcpuCount == 0 {
		vcpuCount = 1
	}

	vmemSize := input.VmemSize

	diskSize := 0

	for _, diskConfig := range input.Disks {
		diskSize += diskConfig.SizeMb
	}

	devCount := len(input.IsolatedDevices)

	eNicCnt := 0
	iNicCnt := 0
	eBw := 0
	iBw := 0
	for _, netConfig := range input.Networks {
		if IsExitNetworkInfo(userCred, netConfig) {
			eNicCnt += 1
			eBw += netConfig.BwLimit
		} else {
			iNicCnt += 1
			iBw += netConfig.BwLimit
		}
	}
	if hasBackup {
		vcpuCount = vcpuCount * 2
		vmemSize = vmemSize * 2
		diskSize = diskSize * 2
	}

	eipCnt := 0
	eipBw := input.EipBw
	if eipBw > 0 {
		eipCnt = 1
	}

	req := SQuota{
		Count:          count,
		Cpu:            int(vcpuCount) * count,
		Memory:         int(vmemSize) * count,
		Storage:        diskSize * count,
		IsolatedDevice: devCount * count,
	}
	regionReq := SRegionQuota{
		Port:  iNicCnt * count,
		Eport: eNicCnt * count,
		//Bw:    iBw * count,
		//Ebw:   eBw * count,
		Eip: eipCnt * count,
	}
	keys := serverCreateInput2ComputeQuotaKeys(input, ownerId)
	req.SetKeys(keys)
	regionReq.SetKeys(keys.SRegionalCloudResourceKeys)
	return req, regionReq
}

func (guest *SGuest) getGuestBackupResourceRequirements(ctx context.Context, userCred mcclient.TokenCredential) SQuota {
	guestDisksSize := guest.getDiskSize()
	return SQuota{
		Count:   1,
		Cpu:     int(guest.VcpuCount),
		Memory:  guest.VmemSize,
		Storage: guestDisksSize,
	}
}

func (guest *SGuest) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	guest.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	tags := []string{"cpu_bound", "io_bound", "io_hardlimit"}
	appTags := make([]string, 0)
	for _, tag := range tags {
		if data.Contains(tag) {
			appTags = append(appTags, tag)
		}
	}
	guest.setApptags(ctx, appTags, userCred)
	guest.SetCreateParams(ctx, userCred, data)
	osProfileJson, _ := data.Get("__os_profile__")
	if osProfileJson != nil {
		guest.setOSProfile(ctx, userCred, osProfileJson)
	}
	if jsonutils.QueryBoolean(data, imageapi.IMAGE_DISABLE_USB_KBD, false) {
		guest.SetMetadata(ctx, imageapi.IMAGE_DISABLE_USB_KBD, "true", userCred)
	}

	userData, _ := data.GetString("user_data")
	if len(userData) > 0 {
		guest.setUserData(ctx, userCred, userData)
	}
	secgroups, _ := jsonutils.GetStringArray(data, "secgroups")
	for _, secgroupId := range secgroups {
		if secgroupId != guest.SecgrpId {
			gs := SGuestsecgroup{}
			gs.SecgroupId = secgroupId
			gs.GuestId = guest.Id
			GuestsecgroupManager.TableSpec().Insert(ctx, &gs)
		}
	}
}

func (guest *SGuest) setApptags(ctx context.Context, appTags []string, userCred mcclient.TokenCredential) {
	err := guest.SetMetadata(ctx, api.VM_METADATA_APP_TAGS, strings.Join(appTags, ","), userCred)
	if err != nil {
		log.Errorln(err)
	}
}

func (guest *SGuest) SetCreateParams(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) {
	// delete deploy files info
	createParams := data.(*jsonutils.JSONDict).CopyExcludes("deploy_configs")
	err := guest.SetMetadata(ctx, api.VM_METADATA_CREATE_PARAMS, createParams.String(), userCred)
	if err != nil {
		log.Errorf("Server %s SetCreateParams: %v", guest.Name, err)
	}
}

func (guest *SGuest) GetCreateParams(userCred mcclient.TokenCredential) (*api.ServerCreateInput, error) {
	input := new(api.ServerCreateInput)
	data := guest.GetMetadataJson(api.VM_METADATA_CREATE_PARAMS, userCred)
	if data == nil {
		return nil, fmt.Errorf("Not found %s %s in metadata", guest.Name, api.VM_METADATA_CREATE_PARAMS)
	}
	err := data.Unmarshal(input)
	return input, err
}

func (manager *SGuestManager) SetPropertiesWithInstanceSnapshot(
	ctx context.Context, userCred mcclient.TokenCredential, ispId string, items []db.IModel,
) {
	misp, err := InstanceSnapshotManager.FetchById(ispId)
	if err == nil {
		isp := misp.(*SInstanceSnapshot)
		for i := 0; i < len(items); i++ {
			guest := items[i].(*SGuest)
			if isp.ServerMetadata != nil {
				metadata := make(map[string]interface{}, 0)
				isp.ServerMetadata.Unmarshal(metadata)
				if passwd, ok := metadata["passwd"]; ok {
					delete(metadata, "passwd")
					metadata["login_key"], _ = utils.EncryptAESBase64(guest.Id, passwd.(string))
				}
				metadata[api.BASE_INSTANCE_SNAPSHOT_ID] = isp.Id
				guest.SetAllMetadata(ctx, metadata, userCred)
			}
		}
	}
}

func (manager *SGuestManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	input := api.ServerCreateInput{}
	data.Unmarshal(&input)
	if len(input.InstanceSnapshotId) > 0 {
		manager.SetPropertiesWithInstanceSnapshot(ctx, userCred, input.InstanceSnapshotId, items)
	}
	pendingUsage, pendingRegionUsage := getGuestResourceRequirements(ctx, userCred, input, ownerId, len(items), input.Backup)
	err := RunBatchCreateTask(ctx, items, userCred, data, pendingUsage, pendingRegionUsage, "GuestBatchCreateTask", input.ParentTaskId)
	if err != nil {
		for i := range items {
			guest := items[i].(*SGuest)
			guest.SetStatus(userCred, api.VM_CREATE_FAILED, err.Error())
		}
	}
}

func (guest *SGuest) GetGroups() []SGroupguest {
	guestgroups := make([]SGroupguest, 0)
	q := GroupguestManager.Query().Equals("guest_id", guest.Id)
	err := db.FetchModelObjects(GroupguestManager, q, &guestgroups)
	if err != nil {
		log.Errorf("GetGroups fail %s", err)
		return nil
	}
	return guestgroups
}

func (self *SGuest) getBandwidth(isExit bool) int {
	bw := 0
	networks, err := self.GetNetworks("")
	if err != nil {
		return bw
	}
	if networks != nil && len(networks) > 0 {
		for i := 0; i < len(networks); i += 1 {
			if networks[i].IsExit() == isExit {
				bw += networks[i].getBandwidth()
			}
		}
	}
	return bw
}

func (self *SGuest) getExtBandwidth() int {
	return self.getBandwidth(true)
}

func (self *SGuest) moreExtraInfo(
	out api.ServerDetails,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	fields stringutils2.SSortedStrings,
	isList bool,
) api.ServerDetails {
	// extra.Add(jsonutils.NewInt(int64(self.getExtBandwidth())), "ext_bw")

	if isList {
		if query.Contains("group") {
			groupId, _ := query.GetString("group")
			q := GroupguestManager.Query().Equals("group_id", groupId).Equals("guest_id", self.Id)
			var groupGuest SGroupguest
			err := q.First(&groupGuest)
			if err == nil {
				out.AttachTime = groupGuest.CreatedAt
			}
		}
	} else {
		out.Networks = self.getNetworksDetails()
		out.Disks = self.getDisksDetails()
		out.DisksInfo = self.getDisksInfoDetails()
		out.VirtualIps = strings.Join(self.getVirtualIPs(), ",")
		out.SecurityRules = self.getSecurityGroupsRules()

		osName := self.GetOS()
		if len(osName) > 0 {
			out.OsName = osName
			if len(self.OsType) == 0 {
				out.OsType = osName
			}
		}

		if userCred.HasSystemAdminPrivilege() {
			out.AdminSecurityRules = self.getAdminSecurityRules()
		}

	}

	out.IsPrepaidRecycle = self.IsPrepaidRecycle()

	if len(self.BackupHostId) > 0 && (len(fields) == 0 || fields.Contains("backup_host_name") || fields.Contains("backup_host_status")) {
		backupHost := HostManager.FetchHostById(self.BackupHostId)
		if len(fields) == 0 || fields.Contains("backup_host_name") {
			out.BackupHostName = backupHost.Name
		}
		if len(fields) == 0 || fields.Contains("backup_host_status") {
			out.BackupHostStatus = backupHost.HostStatus
		}
	}

	if len(fields) == 0 || fields.Contains("can_recycle") {
		err := self.CanPerformPrepaidRecycle()
		if err == nil {
			out.CanRecycle = true
		}
	}

	if len(fields) == 0 || fields.Contains("auto_delete_at") {
		if self.PendingDeleted {
			pendingDeletedAt := self.PendingDeletedAt.Add(time.Second * time.Duration(options.Options.PendingDeleteExpireSeconds))
			out.AutoDeleteAt = pendingDeletedAt
		}
	}

	out.CdromSupport, _ = self.GetDriver().IsSupportCdrom(self)

	return out
}

func (self *SGuestManager) GetMetadataHiddenKeys() []string {
	return []string{
		api.VM_METADATA_CREATE_PARAMS,
	}
}

func (self *SGuest) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.ServerDetails, error) {
	return api.ServerDetails{}, nil
}

func (manager *SGuestManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, err
	}

	// exportKeys, _ := query.GetString("export_keys")
	// keys := strings.Split(exportKeys, ",")
	// guest_id as filter key
	if keys.Contains("ips") {
		guestIpsQuery := GuestnetworkManager.Query("guest_id").GroupBy("guest_id")
		guestIpsQuery.AppendField(sqlchemy.GROUP_CONCAT("concat_ip_addr", guestIpsQuery.Field("ip_addr")))
		ipsSubQuery := guestIpsQuery.SubQuery()
		q.LeftJoin(ipsSubQuery, sqlchemy.Equals(q.Field("id"), ipsSubQuery.Field("guest_id")))
		q.AppendField(ipsSubQuery.Field("concat_ip_addr"))
	}

	if keys.Contains("disk") {
		guestDisksQuery := GuestdiskManager.Query("guest_id", "disk_id").GroupBy("guest_id")
		diskQuery := DiskManager.Query("id", "disk_size").SubQuery()
		guestDisksQuery.Join(diskQuery, sqlchemy.Equals(diskQuery.Field("id"), guestDisksQuery.Field("disk_id")))
		guestDisksQuery.AppendField(sqlchemy.SUM("disk_size", diskQuery.Field("disk_size")))
		guestDisksSubQuery := guestDisksQuery.SubQuery()
		q.LeftJoin(guestDisksSubQuery, sqlchemy.Equals(q.Field("id"), guestDisksSubQuery.Field("guest_id")))
		q.AppendField(guestDisksSubQuery.Field("disk_size"))
	}
	if keys.Contains("eip") {
		eipsQuery := ElasticipManager.Query("associate_id", "ip_addr").Equals("associate_type", "server").GroupBy("associate_id")
		eipsSubQuery := eipsQuery.SubQuery()
		q.LeftJoin(eipsSubQuery, sqlchemy.Equals(q.Field("id"), eipsSubQuery.Field("associate_id")))
		q.AppendField(eipsSubQuery.Field("ip_addr", "eip"))
	}

	if keys.ContainsAny(manager.SHostResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SHostResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SHostResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (manager *SGuestManager) GetExportExtraKeys(ctx context.Context, keys stringutils2.SSortedStrings, rowMap map[string]string) *jsonutils.JSONDict {
	res := manager.SVirtualResourceBaseManager.GetExportExtraKeys(ctx, keys, rowMap)
	// exportKeys, _ := query.GetString("export_keys")
	// keys := strings.Split(exportKeys, ",")
	if ips, ok := rowMap["concat_ip_addr"]; ok && len(ips) > 0 {
		res.Set("ips", jsonutils.NewString(ips))
	}
	if eip, ok := rowMap["eip"]; ok && len(eip) > 0 {
		res.Set("eip", jsonutils.NewString(eip))
	}
	if disk, ok := rowMap["disk_size"]; ok {
		res.Set("disk", jsonutils.NewString(disk))
	}
	if host, ok := rowMap["host"]; ok && len(host) > 0 {
		res.Set("host", jsonutils.NewString(host))
	}
	if zone, ok := rowMap["zone"]; ok && len(zone) > 0 {
		res.Set("zone", jsonutils.NewString(zone))
	}
	if region, ok := rowMap["region"]; ok && len(region) > 0 {
		res.Set("region", jsonutils.NewString(region))
	}
	if manager, ok := rowMap["manager"]; ok && len(manager) > 0 {
		res.Set("manager", jsonutils.NewString(manager))
	}
	if keys.Contains("tenant") {
		if projectId, ok := rowMap["tenant_id"]; ok {
			tenant, err := db.TenantCacheManager.FetchTenantById(ctx, projectId)
			if err == nil {
				res.Set("tenant", jsonutils.NewString(tenant.GetName()))
			}
		}
	}
	if keys.Contains("os_distribution") {
		if osType, ok := rowMap["os_type"]; ok {
			res.Set("os_distribution", jsonutils.NewString(osType))
		}
	}
	return res
}

func (self *SGuest) getNetworksDetails() string {
	guestnets, err := self.GetNetworks("")
	if err != nil {
		return ""
	}
	var buf bytes.Buffer
	for _, nic := range guestnets {
		buf.WriteString(nic.GetDetailedString())
		buf.WriteString("\n")
	}
	return buf.String()
}

func (self *SGuest) getDisksDetails() string {
	var buf bytes.Buffer
	for _, disk := range self.GetDisks() {
		if details := disk.GetDetailedString(); len(details) > 0 {
			buf.WriteString(details)
			buf.WriteString("\n")
		}
	}
	return buf.String()
}

func (self *SGuest) getDisksInfoDetails() []api.GuestDiskInfo {
	disks := self.GetDisks()
	details := make([]api.GuestDiskInfo, len(disks))
	for i := range details {
		details[i] = disks[i].GetDetailedInfo()
	}
	return details
}

func (self *SGuest) GetCdrom() *SGuestcdrom {
	return self.getCdrom(false)
}

func (self *SGuest) getCdrom(create bool) *SGuestcdrom {
	cdrom := SGuestcdrom{}
	cdrom.SetModelManager(GuestcdromManager, &cdrom)

	err := GuestcdromManager.Query().Equals("id", self.Id).First(&cdrom)
	if err != nil {
		if err == sql.ErrNoRows {
			if create {
				cdrom.Id = self.Id
				err = GuestcdromManager.TableSpec().Insert(context.TODO(), &cdrom)
				if err != nil {
					log.Errorf("insert cdrom fail %s", err)
					return nil
				}
				return &cdrom
			} else {
				return nil
			}
		} else {
			log.Errorf("getCdrom query fail %s", err)
			return nil
		}
	} else {
		return &cdrom
	}
}

func (self *SGuest) getKeypair() *SKeypair {
	if len(self.KeypairId) > 0 {
		keypair, _ := KeypairManager.FetchById(self.KeypairId)
		if keypair != nil {
			return keypair.(*SKeypair)
		}
	}
	return nil
}

func (self *SGuest) getKeypairName() string {
	keypair := self.getKeypair()
	if keypair != nil {
		return keypair.Name
	}
	return ""
}

func (self *SGuest) getNotifyIps() string {
	ips := self.GetRealIPs()
	vips := self.getVirtualIPs()
	if vips != nil {
		ips = append(ips, vips...)
	}
	return strings.Join(ips, ",")
}

/*
func (self *SGuest) GetRealIPs() []string {
	guestnets, err := self.GetNetworks("")
	if err != nil {
		return nil
	}
	ips := make([]string, 0)
	for _, nic := range guestnets {
		if !nic.Virtual {
			ips = append(ips, nic.IpAddr)
		}
	}
	return ips
}
*/

func (self *SGuest) IsExitOnly() bool {
	for _, ip := range self.GetRealIPs() {
		addr, _ := netutils.NewIPV4Addr(ip)
		if !netutils.IsExitAddress(addr) {
			return false
		}
	}
	return true
}

func (self *SGuest) getVirtualIPs() []string {
	ips := make([]string, 0)
	for _, guestgroup := range self.GetGroups() {
		group := guestgroup.GetGroup()
		groupnets, err := group.GetNetworks()
		if err != nil {
			continue
		}
		for _, groupnetwork := range groupnets {
			ips = append(ips, groupnetwork.IpAddr)
		}
	}
	return ips
}

func (self *SGuest) GetPrivateIPs() []string {
	ips := self.GetRealIPs()
	for i := len(ips) - 1; i >= 0; i-- {
		ipAddr, err := netutils.NewIPV4Addr(ips[i])
		if err != nil {
			log.Errorf("guest %s(%s) has bad ipv4 address (%s): %v", self.Name, self.Id, ips[i], err)
			continue
		}
		if !netutils.IsPrivate(ipAddr) {
			ips = append(ips[:i], ips[i+1:]...)
		}
	}
	return ips
}

func (self *SGuest) getIPs() []string {
	ips := self.GetRealIPs()
	vips := self.getVirtualIPs()
	ips = append(ips, vips...)
	/*eip, _ := self.GetEip()
	if eip != nil {
		ips = append(ips, eip.IpAddr)
	}*/
	return ips
}

func (self *SGuest) getZone() *SZone {
	host := self.GetHost()
	if host != nil {
		return host.GetZone()
	}
	return nil
}

func (self *SGuest) getRegion() *SCloudregion {
	zone := self.getZone()
	if zone != nil {
		return zone.GetRegion()
	}
	return nil
}

func (self *SGuest) GetOS() string {
	if len(self.OsType) > 0 {
		return self.OsType
	}
	return self.GetMetadata("os_name", nil)
}

func (self *SGuest) IsLinux() bool {
	os := self.GetOS()
	if strings.HasPrefix(strings.ToLower(os), "lin") {
		return true
	} else {
		return false
	}
}

func (self *SGuest) IsWindows() bool {
	os := self.GetOS()
	if strings.HasPrefix(strings.ToLower(os), "win") {
		return true
	} else {
		return false
	}
}

func (self *SGuest) getSecgroupJson() ([]jsonutils.JSONObject, error) {
	objs := []jsonutils.JSONObject{}

	secgroups, err := self.GetSecgroups()
	if err != nil {
		return nil, errors.Wrap(err, "GetSecgroups")
	}
	for _, secGrp := range secgroups {
		objs = append(objs, secGrp.getDesc())
	}
	return objs, nil
}

func (self *SGuest) GetSecgroups() ([]SSecurityGroup, error) {
	secgrpQuery := SecurityGroupManager.Query()
	secgrpQuery.Filter(
		sqlchemy.OR(
			sqlchemy.Equals(secgrpQuery.Field("id"), self.SecgrpId),
			sqlchemy.In(secgrpQuery.Field("id"), GuestsecgroupManager.Query("secgroup_id").Equals("guest_id", self.Id).SubQuery()),
		),
	)
	secgroups := []SSecurityGroup{}
	err := db.FetchModelObjects(SecurityGroupManager, secgrpQuery, &secgroups)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return secgroups, nil
}

func (self *SGuest) getAdminSecgroup() *SSecurityGroup {
	secGrp, _ := SecurityGroupManager.FetchSecgroupById(self.AdminSecgrpId)
	return secGrp
}

func (self *SGuest) getAdminSecgroupName() string {
	secgrp := self.getAdminSecgroup()
	if secgrp != nil {
		return secgrp.GetName()
	}
	return ""
}

//获取多个安全组规则，优先级降序排序
func (self *SGuest) getSecurityGroupsRules() string {
	secgroups, _ := self.GetSecgroups()
	secgroupids := []string{}
	for _, secgroup := range secgroups {
		secgroupids = append(secgroupids, secgroup.Id)
	}
	q := SecurityGroupRuleManager.Query()
	q.Filter(sqlchemy.In(q.Field("secgroup_id"), secgroupids)).Desc(q.Field("priority"), q.Field("action"))
	secrules := []SSecurityGroupRule{}
	if err := db.FetchModelObjects(SecurityGroupRuleManager, q, &secrules); err != nil {
		log.Errorf("Get rules error: %v", err)
		return options.Options.DefaultSecurityRules
	}
	rules := []string{}
	for _, rule := range secrules {
		rules = append(rules, rule.String())
	}
	return strings.Join(rules, SECURITY_GROUP_SEPARATOR)
}

func (self *SGuest) getAdminSecurityRules() string {
	secgrp := self.getAdminSecgroup()
	if secgrp != nil {
		ret, _ := secgrp.getSecurityRuleString()
		return ret
	} else {
		return options.Options.DefaultAdminSecurityRules
	}
}

func (self *SGuest) isGpu() bool {
	return len(self.GetIsolatedDevices()) != 0
}

func (self *SGuest) GetIsolatedDevices() []SIsolatedDevice {
	return IsolatedDeviceManager.findAttachedDevicesOfGuest(self)
}

func (self *SGuest) IsFailureStatus() bool {
	return strings.Index(self.Status, "fail") >= 0
}

var (
	lostNamePattern = regexp.MustCompile(`-lost@\d{8}$`)
)

func (self *SGuest) GetIRegion() (cloudprovider.ICloudRegion, error) {
	host := self.GetHost()
	if host == nil {
		return nil, fmt.Errorf("failed to get host by guest %s(%s)", self.Name, self.Id)
	}
	provider, err := host.GetDriver()
	if err != nil {
		return nil, fmt.Errorf("No cloudprovider for host: %s", err)
	}
	if provider.GetFactory().IsOnPremise() {
		return provider.GetOnPremiseIRegion()
	}
	return host.GetIRegion()
}

func (self *SGuest) syncRemoveCloudVM(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	if self.BillingType == billing_api.BILLING_TYPE_PREPAID {
		diff, err := db.Update(self, func() error {
			self.BillingType = billing_api.BILLING_TYPE_POSTPAID
			self.ExpiredAt = time.Time{}
			return nil
		})
		if err != nil {
			return err
		}
		db.OpsLog.LogSyncUpdate(self, diff, userCred)
	}

	if self.IsFailureStatus() {
		return nil
	}

	iregion, err := self.GetIRegion()
	if err != nil {
		return err
	}
	iVM, err := iregion.GetIVMById(self.ExternalId)
	if err == nil { //漂移归位
		if hostId := iVM.GetIHostId(); len(hostId) > 0 {
			host, err := db.FetchByExternalIdAndManagerId(HostManager, hostId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				host := self.GetHost()
				if host != nil {
					return q.Equals("manager_id", host.ManagerId)
				}
				return q
			})
			if err == nil {
				_, err = db.Update(self, func() error {
					self.HostId = host.GetId()
					self.Status = iVM.GetStatus()
					return nil
				})
				return err
			}
		}
	} else if errors.Cause(err) != cloudprovider.ErrNotFound {
		return err
	}

	if options.SyncPurgeRemovedResources.Contains(self.Keyword()) {
		log.Debugf("purge removed resource %s", self.Name)
		return self.purge(ctx, userCred)
	}

	if !lostNamePattern.MatchString(self.Name) {
		db.Update(self, func() error {
			self.Name = fmt.Sprintf("%s-lost@%s", self.Name, timeutils.ShortDate(time.Now()))
			return nil
		})
	}

	if self.Status != api.VM_UNKNOWN {
		self.SetStatus(userCred, api.VM_UNKNOWN, "Sync lost")
	}
	return nil
}

func (guest *SGuest) SyncAllWithCloudVM(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, extVM cloudprovider.ICloudVM) error {
	if host == nil {
		return errors.Error("guest has no host")
	}

	provider := host.GetCloudprovider()
	if provider == nil {
		return errors.Error("host has no provider")
	}

	driver, err := provider.GetProvider()
	if err != nil {
		return errors.Wrap(err, "provider.GetProvider")
	}

	err = guest.syncWithCloudVM(ctx, userCred, driver, host, extVM, provider.GetOwnerId())
	if err != nil {
		return errors.Wrap(err, "guest.syncWithCloudVM")
	}

	syncVMPeripherals(ctx, userCred, guest, extVM, host, provider, driver)

	return nil
}

func (self *SGuest) syncWithCloudVM(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, host *SHost, extVM cloudprovider.ICloudVM, syncOwnerId mcclient.IIdentityProvider) error {
	recycle := false

	if provider.GetFactory().IsSupportPrepaidResources() && self.IsPrepaidRecycle() {
		recycle = true
	}

	// metaData := extVM.GetMetadata()
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		extVM.Refresh()
		if options.NameSyncResources.Contains(self.Keyword()) && !recycle {
			newName, _ := db.GenerateAlterName(self, extVM.GetName())
			if len(newName) > 0 && newName != self.Name {
				self.Name = newName
			}
		}
		if !self.IsFailureStatus() {
			self.Status = extVM.GetStatus()
		}
		self.VcpuCount = extVM.GetVcpuCount()
		self.BootOrder = extVM.GetBootOrder()
		self.Vga = extVM.GetVga()
		self.Vdi = extVM.GetVdi()
		self.OsType = extVM.GetOSType()
		self.Bios = extVM.GetBios()
		self.Machine = extVM.GetMachine()
		if !recycle {
			self.HostId = host.Id
		}

		instanceType := extVM.GetInstanceType()

		if len(instanceType) > 0 {
			self.InstanceType = instanceType
		}

		if extVM.GetHypervisor() == api.HYPERVISOR_AWS {
			sku, err := ServerSkuManager.FetchSkuByNameAndProvider(instanceType, api.CLOUD_PROVIDER_AWS, false)
			if err == nil {
				self.VmemSize = sku.MemorySizeMB
			} else {
				self.VmemSize = extVM.GetVmemSizeMB()
			}
		} else {
			self.VmemSize = extVM.GetVmemSizeMB()
		}

		self.Hypervisor = extVM.GetHypervisor()

		self.IsEmulated = extVM.IsEmulated()

		if provider.GetFactory().IsSupportPrepaidResources() && !recycle &&
			!extVM.GetExpiredAt().IsZero() {

			self.BillingType = extVM.GetBillingType()
			self.ExpiredAt = extVM.GetExpiredAt()
			if self.GetDriver().IsSupportSetAutoRenew() {
				self.AutoRenew = extVM.IsAutoRenew()
			}
		}

		// no need to sync CreatedAt
		// if !recycle {
		//	if createdAt := extVM.GetCreatedAt(); !createdAt.IsZero() {
		//		self.CreatedAt = createdAt
		//	}
		// }

		return nil
	})
	if err != nil {
		log.Errorf("%s", err)
		return err
	}

	db.OpsLog.LogSyncUpdate(self, diff, userCred)

	SyncCloudProject(userCred, self, syncOwnerId, extVM, host.ManagerId)

	if provider.GetFactory().IsSupportPrepaidResources() && recycle {
		vhost := self.GetHost()
		err = vhost.syncWithCloudPrepaidVM(extVM, host)
		if err != nil {
			return err
		}
	}

	return nil
}

func (manager *SGuestManager) newCloudVM(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, host *SHost, extVM cloudprovider.ICloudVM, syncOwnerId mcclient.IIdentityProvider) (*SGuest, error) {

	guest := SGuest{}
	guest.SetModelManager(manager, &guest)

	guest.Status = extVM.GetStatus()
	guest.ExternalId = extVM.GetGlobalId()
	guest.VcpuCount = extVM.GetVcpuCount()
	guest.BootOrder = extVM.GetBootOrder()
	guest.Vga = extVM.GetVga()
	guest.Vdi = extVM.GetVdi()
	guest.OsType = extVM.GetOSType()
	guest.Bios = extVM.GetBios()
	guest.Machine = extVM.GetMachine()
	guest.Hypervisor = extVM.GetHypervisor()

	guest.IsEmulated = extVM.IsEmulated()

	if provider.GetFactory().IsSupportPrepaidResources() {
		guest.BillingType = extVM.GetBillingType()
		if expired := extVM.GetExpiredAt(); !expired.IsZero() {
			guest.ExpiredAt = expired
		}
		if guest.GetDriver().IsSupportSetAutoRenew() {
			guest.AutoRenew = extVM.IsAutoRenew()
		}
	}

	if createdAt := extVM.GetCreatedAt(); !createdAt.IsZero() {
		guest.CreatedAt = createdAt
	}

	guest.HostId = host.Id

	instanceType := extVM.GetInstanceType()

	/*zoneExtId, err := metaData.GetString("zone_ext_id")
	if err != nil {
		log.Errorf("get zone external id fail %s", err)
	}

	isku, err := ServerSkuManager.FetchByZoneExtId(zoneExtId, instanceType)
	if err != nil {
		log.Errorf("get sku zone %s instance type %s fail %s", zoneExtId, instanceType, err)
	} else {
		guest.SkuId = isku.GetId()
	}*/

	if len(instanceType) > 0 {
		guest.InstanceType = instanceType
	}

	if extVM.GetHypervisor() == api.HYPERVISOR_AWS {
		sku, err := ServerSkuManager.FetchSkuByNameAndProvider(instanceType, api.CLOUD_PROVIDER_AWS, false)
		if err == nil {
			guest.VmemSize = sku.MemorySizeMB
		} else {
			guest.VmemSize = extVM.GetVmemSizeMB()
		}
	} else {
		guest.VmemSize = extVM.GetVmemSizeMB()
	}

	var err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		if options.NameSyncResources.Contains(manager.Keyword()) {
			guest.Name = extVM.GetName()
		} else {
			newName, err := db.GenerateName(ctx, manager, syncOwnerId, extVM.GetName())
			if err != nil {
				return errors.Wrapf(err, "db.GenerateName")
			}
			guest.Name = newName
		}

		return manager.TableSpec().Insert(ctx, &guest)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	SyncCloudProject(userCred, &guest, syncOwnerId, extVM, host.ManagerId)

	db.OpsLog.LogEvent(&guest, db.ACT_CREATE, guest.GetShortDesc(ctx), userCred)

	if guest.Status == api.VM_RUNNING {
		db.OpsLog.LogEvent(&guest, db.ACT_START, guest.GetShortDesc(ctx), userCred)
	}

	return &guest, nil
}

func (manager *SGuestManager) TotalCount(
	scope rbacutils.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel,
	status []string, hypervisors []string,
	includeSystem bool, pendingDelete bool,
	hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string,
	since *time.Time,
) SGuestCountStat {
	return usageTotalGuestResouceCount(scope, ownerId, rangeObjs, status, hypervisors, includeSystem, pendingDelete, hostTypes, resourceTypes, providers, brands, cloudEnv, since)
}

func (self *SGuest) detachNetworks(ctx context.Context, userCred mcclient.TokenCredential, gns []SGuestnetwork, reserve bool, deploy bool) error {
	err := GuestnetworkManager.DeleteGuestNics(ctx, userCred, gns, reserve)
	if err != nil {
		return err
	}
	host := self.GetHost()
	if host != nil {
		host.ClearSchedDescCache() // ignore error
	}
	if deploy {
		self.StartGuestDeployTask(ctx, userCred, nil, "deploy", "")
	}
	return nil
}

func (self *SGuest) getAttach2NetworkCount(net *SNetwork) (int, error) {
	q := GuestnetworkManager.Query()
	q = q.Equals("guest_id", self.Id).Equals("network_id", net.Id)
	return q.CountWithError()
}

func (self *SGuest) getUsableNicIndex() int8 {
	nics, err := self.GetNetworks("")
	if err != nil {
		return -1
	}
	maxIndex := int8(len(nics))
	for i := int8(0); i <= maxIndex; i++ {
		found := true
		for j := range nics {
			if nics[j].Index == i {
				found = false
				break
			}
		}
		if found {
			return i
		}
	}
	panic(fmt.Sprintf("cannot find usable nic index for guest %s(%s)",
		self.Name, self.Id))
}

func (self *SGuest) setOSProfile(ctx context.Context, userCred mcclient.TokenCredential, profile jsonutils.JSONObject) error {
	return self.SetMetadata(ctx, "__os_profile__", profile, userCred)
}

func (self *SGuest) GetOSProfile() osprofile.SOSProfile {
	osName := self.GetOS()
	osProf := osprofile.GetOSProfile(osName, self.Hypervisor)
	val := self.GetMetadata("__os_profile__", nil)
	if len(val) > 0 {
		jsonVal, _ := jsonutils.ParseString(val)
		if jsonVal != nil {
			jsonVal.Unmarshal(&osProf)
		}
	}
	return osProf
}

// Summary of network address allocation strategy
//
// IpAddr when specified must be part of the network
//
// Use IpAddr without checking if it's already allocated when UseDesignatedIP
// is true.  See b31bc7fa ("feature: 1. baremetal server reuse host ip...")
//
// Try IpAddr from reserved pool when allowed by TryReserved.  Otherwise
// fallback to usual allocation method (AllocDir).  Error when
// RequireDesignatedIP is true and the allocated address does not match IpAddr
type Attach2NetworkArgs struct {
	Network *SNetwork

	IpAddr              string
	AllocDir            api.IPAllocationDirection
	TryReserved         bool
	RequireDesignatedIP bool
	UseDesignatedIP     bool

	BwLimit   int
	NicDriver string
	NicConfs  []SNicConfig

	Virtual bool

	PendingUsage quotas.IQuota
}

func (args *Attach2NetworkArgs) onceArgs(i int) attach2NetworkOnceArgs {
	if i < 0 || i > len(args.NicConfs)-1 {
		return attach2NetworkOnceArgs{}
	}
	r := attach2NetworkOnceArgs{
		network: args.Network,

		ipAddr:              args.IpAddr,
		allocDir:            args.AllocDir,
		tryReserved:         args.TryReserved,
		requireDesignatedIP: args.RequireDesignatedIP,
		useDesignatedIP:     args.UseDesignatedIP,

		bwLimit:   args.BwLimit,
		nicDriver: args.NicDriver,
		nicConf:   args.NicConfs[i],

		virtual: args.Virtual,

		pendingUsage: args.PendingUsage,
	}
	if i > 0 {
		r.ipAddr = ""
		r.bwLimit = 0
		r.virtual = true
		r.tryReserved = false
		r.requireDesignatedIP = false
		r.useDesignatedIP = false
		r.nicConf = args.NicConfs[i]
		r.nicDriver = ""
	}
	return r
}

type attach2NetworkOnceArgs struct {
	network *SNetwork

	ipAddr              string
	allocDir            api.IPAllocationDirection
	tryReserved         bool
	requireDesignatedIP bool
	useDesignatedIP     bool

	bwLimit     int
	nicDriver   string
	nicConf     SNicConfig
	teamWithMac string

	virtual bool

	pendingUsage quotas.IQuota
}

func (self *SGuest) Attach2Network(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	args Attach2NetworkArgs,
) ([]SGuestnetwork, error) {
	onceArgs := args.onceArgs(0)
	firstNic, err := self.attach2NetworkOnce(ctx, userCred, onceArgs)
	if err != nil {
		return nil, errors.Wrap(err, "self.attach2NetworkOnce")
	}
	retNics := []SGuestnetwork{*firstNic}
	if len(args.NicConfs) > 1 {
		firstMac, _ := netutils2.ParseMac(firstNic.MacAddr)
		for i := 1; i < len(args.NicConfs); i += 1 {
			onceArgs := args.onceArgs(i)
			onceArgs.nicDriver = firstNic.Driver
			onceArgs.teamWithMac = firstNic.MacAddr
			if onceArgs.nicConf.Mac == "" {
				onceArgs.nicConf.Mac = firstMac.Add(i).String()
			}
			gn, err := self.attach2NetworkOnce(ctx, userCred, onceArgs)
			if err != nil {
				return retNics, errors.Wrap(err, "self.attach2NetworkOnce")
			}
			retNics = append(retNics, *gn)
		}
	}
	return retNics, nil
}

func (self *SGuest) attach2NetworkOnce(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	args attach2NetworkOnceArgs,
) (*SGuestnetwork, error) {
	var (
		index     = args.nicConf.Index
		nicDriver = args.nicDriver
	)
	if index < 0 {
		index = self.getUsableNicIndex()
	}
	if nicDriver == "" {
		osProf := self.GetOSProfile()
		nicDriver = osProf.NetDriver
	}
	newArgs := newGuestNetworkArgs{
		guest:   self,
		network: args.network,

		index: index,

		ipAddr:              args.ipAddr,
		allocDir:            args.allocDir,
		tryReserved:         args.tryReserved,
		requireDesignatedIP: args.requireDesignatedIP,
		useDesignatedIP:     args.useDesignatedIP,

		ifname:      args.nicConf.Ifname,
		macAddr:     args.nicConf.Mac,
		bwLimit:     args.bwLimit,
		nicDriver:   nicDriver,
		teamWithMac: args.teamWithMac,

		virtual: args.virtual,
	}
	lockman.LockClass(ctx, QuotaManager, self.ProjectId)
	defer lockman.ReleaseClass(ctx, QuotaManager, self.ProjectId)
	guestnic, err := GuestnetworkManager.newGuestNetwork(ctx, userCred, newArgs)
	if err != nil {
		return nil, errors.Wrap(err, "GuestnetworkManager.newGuestNetwork")
	}
	var (
		network      = args.network
		pendingUsage = args.pendingUsage
		teamWithMac  = args.teamWithMac
	)
	network.updateDnsRecord(guestnic, true)
	network.updateGuestNetmap(guestnic)
	if pendingUsage != nil && len(teamWithMac) == 0 {
		cancelUsage := SRegionQuota{}
		if network.IsExitNetwork() {
			cancelUsage.Eport = 1
		} else {
			cancelUsage.Port = 1
		}
		keys, err := self.GetRegionalQuotaKeys()
		if err != nil {
			log.Warningf("self.GetRegionalQuotaKeys fail %s", err)
		}
		cancelUsage.SetKeys(keys)
		err = quotas.CancelPendingUsage(ctx, userCred, pendingUsage, &cancelUsage, true)
		if err != nil {
			log.Warningf("QuotaManager.CancelPendingUsage fail %s", err)
		}
	}
	db.OpsLog.LogAttachEvent(ctx, self, network, userCred, guestnic.GetShortDesc(ctx))
	return guestnic, nil
}

type sRemoveGuestnic struct {
	nic     *SGuestnetwork
	reserve bool
}

type sAddGuestnic struct {
	index   int
	nic     cloudprovider.ICloudNic
	net     *SNetwork
	reserve bool
}

func getCloudNicNetwork(ctx context.Context, vnic cloudprovider.ICloudNic, host *SHost, ipList []string, index int) (*SNetwork, error) {
	vnet := vnic.GetINetwork()
	if vnet == nil {
		if vnic.InClassicNetwork() {
			region := host.GetRegion()
			cloudprovider := host.GetCloudprovider()
			vpc, err := VpcManager.GetOrCreateVpcForClassicNetwork(ctx, cloudprovider, region)
			if err != nil {
				return nil, errors.Wrap(err, "NewVpcForClassicNetwork")
			}
			zone := host.GetZone()
			wire, err := WireManager.GetOrCreateWireForClassicNetwork(ctx, vpc, zone)
			if err != nil {
				return nil, errors.Wrap(err, "NewWireForClassicNetwork")
			}
			return NetworkManager.GetOrCreateClassicNetwork(ctx, wire)
		}
		ip := vnic.GetIP()
		if len(ip) == 0 {
			if index < len(ipList) {
				ip = ipList[index]
			}
			if len(ip) == 0 {
				return nil, fmt.Errorf("Cannot find inetwork for vnics %s: no ip", vnic.GetMAC())
			}
		}
		// find network by IP
		return host.getNetworkOfIPOnHost(ip)
	}
	localNetObj, err := db.FetchByExternalIdAndManagerId(NetworkManager, vnet.GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		vpc := VpcManager.Query().SubQuery()
		wire := WireManager.Query().SubQuery()
		return q.Join(wire, sqlchemy.Equals(q.Field("wire_id"), wire.Field("id"))).
			Join(vpc, sqlchemy.Equals(wire.Field("vpc_id"), vpc.Field("id"))).
			Filter(sqlchemy.Equals(vpc.Field("manager_id"), host.ManagerId))
	})
	if err != nil {
		return nil, fmt.Errorf("Cannot find network of external_id %s: %v", vnet.GetGlobalId(), err)
	}
	localNet := localNetObj.(*SNetwork)
	return localNet, nil
}

func (self *SGuest) SyncVMNics(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, vnics []cloudprovider.ICloudNic, ipList []string) compare.SyncResult {
	result := compare.SyncResult{}

	guestnics, err := self.GetNetworks("")
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]sRemoveGuestnic, 0)
	adds := make([]sAddGuestnic, 0)

	for i := 0; i < len(guestnics) || i < len(vnics); i += 1 {
		if i < len(guestnics) && i < len(vnics) {
			localNet, err := getCloudNicNetwork(ctx, vnics[i], host, ipList, i)
			if err != nil {
				log.Errorf("%s", err)
				result.Error(err)
				return result
			}
			if guestnics[i].NetworkId == localNet.Id {
				if guestnics[i].MacAddr == vnics[i].GetMAC() {
					if guestnics[i].IpAddr == vnics[i].GetIP() {
						if err := NetworkAddressManager.syncGuestnetworkICloudNic(
							ctx, userCred, &guestnics[i], vnics[i]); err != nil {
							result.AddError(err)
						}
					} else if len(vnics[i].GetIP()) > 0 {
						// ip changed
						removed = append(removed, sRemoveGuestnic{nic: &guestnics[i]})
						adds = append(adds, sAddGuestnic{index: i, nic: vnics[i], net: localNet})
					} else {
						// do nothing
						// vm maybe turned off, ignore the case
					}
				} else {
					reserve := false
					if len(guestnics[i].IpAddr) > 0 && guestnics[i].IpAddr == vnics[i].GetIP() {
						// mac changed
						reserve = true
						removed = append(removed, sRemoveGuestnic{nic: &guestnics[i], reserve: reserve})
						adds = append(adds, sAddGuestnic{index: i, nic: vnics[i], net: localNet, reserve: reserve})
					} else if len(guestnics[i].IpAddr) == 0 {
						db.Update(&guestnics[i], func() error {
							guestnics[i].IpAddr = vnics[i].GetIP()
							guestnics[i].MacAddr = vnics[i].GetMAC()
							return nil
						})
					}
				}
			} else {
				removed = append(removed, sRemoveGuestnic{nic: &guestnics[i]})
				adds = append(adds, sAddGuestnic{index: i, nic: vnics[i], net: localNet})
			}
		} else if i < len(guestnics) {
			removed = append(removed, sRemoveGuestnic{nic: &guestnics[i]})
		} else if i < len(vnics) {
			localNet, err := getCloudNicNetwork(ctx, vnics[i], host, ipList, i)
			if err != nil {
				log.Errorf("%s", err) // ignore this case
			} else {
				adds = append(adds, sAddGuestnic{index: i, nic: vnics[i], net: localNet})
			}
		}
	}

	for _, remove := range removed {
		err := self.detachNetworks(ctx, userCred, []SGuestnetwork{*remove.nic}, remove.reserve, false)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}

	for _, add := range adds {
		if len(add.nic.GetIP()) == 0 && len(ipList) <= add.index {
			continue // cannot determine which network it attached to
		}
		if add.net == nil {
			continue // cannot determine which network it attached to
		}
		ipStr := add.nic.GetIP()
		if len(ipStr) == 0 {
			ipStr = ipList[add.index]
		}
		// check if the IP has been occupied, if yes, release the IP
		gn, err := GuestnetworkManager.getGuestNicByIP(ipStr, add.net.Id)
		if err != nil {
			result.AddError(err)
			continue
		}
		if gn != nil {
			err = gn.Detach(ctx, userCred)
			if err != nil {
				result.AddError(err)
				continue
			}
		}
		nicConf := SNicConfig{
			Mac:    add.nic.GetMAC(),
			Index:  -1,
			Ifname: "",
		}
		// always try allocate from reserved pool
		guestnetworks, err := self.Attach2Network(ctx, userCred, Attach2NetworkArgs{
			Network:             add.net,
			IpAddr:              ipStr,
			NicDriver:           add.nic.GetDriver(),
			TryReserved:         true,
			AllocDir:            api.IPAllocationDefault,
			RequireDesignatedIP: true,
			UseDesignatedIP:     true,
			NicConfs:            []SNicConfig{nicConf},
		})
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
			for i := range guestnetworks {
				guestnetwork := &guestnetworks[i]
				if NetworkAddressManager.syncGuestnetworkICloudNic(
					ctx, userCred, guestnetwork, add.nic); err != nil {
					result.AddError(err)
				}
			}
		}
	}

	return result
}

func (self *SGuest) IsAttach2Disk(disk *SDisk) (bool, error) {
	return self.isAttach2Disk(disk)
}

func (self *SGuest) isAttach2Disk(disk *SDisk) (bool, error) {
	q := GuestdiskManager.Query().Equals("disk_id", disk.Id).Equals("guest_id", self.Id)
	cnt, err := q.CountWithError()
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (self *SGuest) getDiskIndex() int8 {
	guestdisks := self.GetDisks()
	var max uint
	for i := 0; i < len(guestdisks); i++ {
		if uint(guestdisks[i].Index) > max {
			max = uint(guestdisks[i].Index)
		}
	}

	idxs := make([]int, max+1)
	for i := 0; i < len(guestdisks); i++ {
		idxs[guestdisks[i].Index] = 1
	}

	// find first idx not set
	for i := 0; i < len(idxs); i++ {
		if idxs[i] != 1 {
			return int8(i)
		}
	}

	return int8(max + 1)
}

func (self *SGuest) AttachDisk(ctx context.Context, disk *SDisk, userCred mcclient.TokenCredential, driver string, cache string, mountpoint string) error {
	return self.attach2Disk(ctx, disk, userCred, driver, cache, mountpoint)
}

func (self *SGuest) attach2Disk(ctx context.Context, disk *SDisk, userCred mcclient.TokenCredential, driver string, cache string, mountpoint string) error {
	attached, err := self.isAttach2Disk(disk)
	if err != nil {
		return err
	}
	if attached {
		return fmt.Errorf("Guest has been attached to disk")
	}

	if len(driver) == 0 {
		osProf := self.GetOSProfile()
		driver = osProf.DiskDriver
	}
	guestdisk := SGuestdisk{}
	guestdisk.SetModelManager(GuestdiskManager, &guestdisk)

	guestdisk.DiskId = disk.Id
	guestdisk.GuestId = self.Id

	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	guestdisk.Index = self.getDiskIndex()
	err = guestdisk.DoSave(ctx, driver, cache, mountpoint)
	if err == nil {
		db.OpsLog.LogAttachEvent(ctx, self, disk, userCred, nil)
	}
	return err
}

type sSyncDiskPair struct {
	disk  *SDisk
	vdisk cloudprovider.ICloudDisk
}

func (self *SGuest) SyncVMDisks(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, host *SHost, vdisks []cloudprovider.ICloudDisk, syncOwnerId mcclient.IIdentityProvider) compare.SyncResult {
	result := compare.SyncResult{}

	newdisks := make([]sSyncDiskPair, 0)
	for i := 0; i < len(vdisks); i += 1 {
		if len(vdisks[i].GetGlobalId()) == 0 {
			continue
		}
		disk, err := DiskManager.syncCloudDisk(ctx, userCred, provider, vdisks[i], i, syncOwnerId, host.ManagerId)
		if err != nil {
			log.Errorf("syncCloudDisk error: %v", err)
			result.Error(err)
			return result
		}
		if disk.PendingDeleted != self.PendingDeleted { //避免主机正常,磁盘在回收站的情况
			db.Update(disk, func() error {
				disk.PendingDeleted = self.PendingDeleted
				return nil
			})
		}
		newdisks = append(newdisks, sSyncDiskPair{disk: disk, vdisk: vdisks[i]})
	}

	needRemoves := make([]SGuestdisk, 0)

	guestdisks := self.GetDisks()
	for i := 0; i < len(guestdisks); i += 1 {
		find := false
		for j := 0; j < len(newdisks); j += 1 {
			if newdisks[j].disk.Id == guestdisks[i].DiskId {
				find = true
				break
			}
		}
		if !find {
			needRemoves = append(needRemoves, guestdisks[i])
		}
	}

	needAdds := make([]sSyncDiskPair, 0)

	for i := 0; i < len(newdisks); i += 1 {
		find := false
		for j := 0; j < len(guestdisks); j += 1 {
			if newdisks[i].disk.Id == guestdisks[j].DiskId {
				find = true
				break
			}
		}
		if !find {
			needAdds = append(needAdds, newdisks[i])
		}
	}

	for i := 0; i < len(needRemoves); i += 1 {
		err := needRemoves[i].Detach(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}
	for i := 0; i < len(needAdds); i += 1 {
		vdisk := needAdds[i].vdisk
		err := self.attach2Disk(ctx, needAdds[i].disk, userCred, vdisk.GetDriver(), vdisk.GetCacheMode(), vdisk.GetMountpoint())
		if err != nil {
			log.Errorf("attach2Disk error: %v", err)
			result.AddError(err)
		} else {
			result.Add()
		}
	}
	return result
}

func filterGuestByRange(q *sqlchemy.SQuery, rangeObjs []db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string) *sqlchemy.SQuery {
	hosts := HostManager.Query().SubQuery()
	subq := hosts.Query(hosts.Field("id"))
	subq = AttachUsageQuery(subq, hosts, hostTypes, resourceTypes, providers, brands, cloudEnv, rangeObjs)

	q = q.In("host_id", subq.SubQuery())

	return q
}

type SGuestCountStat struct {
	TotalGuestCount       int
	TotalCpuCount         int
	TotalMemSize          int
	TotalDiskSize         int
	TotalIsolatedCount    int
	TotalBackupGuestCount int
	TotalBackupCpuCount   int
	TotalBackupMemSize    int
	TotalBackupDiskSize   int
}

func usageTotalGuestResouceCount(
	scope rbacutils.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel,
	status []string,
	hypervisors []string,
	includeSystem bool,
	pendingDelete bool,
	hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	since *time.Time,
) SGuestCountStat {
	q, guests := _guestResourceCountQuery(scope, ownerId, rangeObjs, status, hypervisors,
		pendingDelete, hostTypes, resourceTypes, providers, brands, cloudEnv, since)
	if !includeSystem {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.IsNull(guests.Field("is_system")), sqlchemy.IsFalse(guests.Field("is_system"))))
	}

	stat := SGuestCountStat{}
	row := q.Row()
	err := q.Row2Struct(row, &stat)
	if err != nil {
		log.Errorf("%s", err)
	}
	stat.TotalCpuCount += stat.TotalBackupCpuCount
	stat.TotalMemSize += stat.TotalBackupMemSize
	stat.TotalDiskSize += stat.TotalBackupDiskSize
	return stat
}

func _guestResourceCountQuery(
	scope rbacutils.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel,
	status []string,
	hypervisors []string,
	pendingDelete bool,
	hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	since *time.Time,
) (*sqlchemy.SQuery, *sqlchemy.SSubQuery) {

	guestdisks := GuestdiskManager.Query().SubQuery()
	disks := DiskManager.Query().SubQuery()

	diskQuery := guestdisks.Query(guestdisks.Field("guest_id"), sqlchemy.SUM("guest_disk_size", disks.Field("disk_size")))
	diskQuery = diskQuery.Join(disks, sqlchemy.Equals(guestdisks.Field("disk_id"), disks.Field("id")))
	diskQuery = diskQuery.GroupBy(guestdisks.Field("guest_id"))
	diskSubQuery := diskQuery.SubQuery()

	backupDiskQuery := guestdisks.Query(guestdisks.Field("guest_id"), sqlchemy.SUM("guest_disk_size", disks.Field("disk_size")))
	backupDiskQuery = backupDiskQuery.LeftJoin(disks, sqlchemy.Equals(guestdisks.Field("disk_id"), disks.Field("id")))
	backupDiskQuery = backupDiskQuery.Filter(sqlchemy.IsNotEmpty(disks.Field("backup_storage_id")))
	backupDiskQuery = backupDiskQuery.GroupBy(guestdisks.Field("guest_id"))

	diskBackupSubQuery := backupDiskQuery.SubQuery()
	// diskBackupSubQuery := diskQuery.IsNotEmpty("backup_storage_id").SubQuery()

	isolated := IsolatedDeviceManager.Query().SubQuery()

	isoDevQuery := isolated.Query(isolated.Field("guest_id"), sqlchemy.COUNT("device_sum"))
	isoDevQuery = isoDevQuery.Filter(sqlchemy.IsNotNull(isolated.Field("guest_id")))
	isoDevQuery = isoDevQuery.GroupBy(isolated.Field("guest_id"))

	isoDevSubQuery := isoDevQuery.SubQuery()

	var guests *sqlchemy.SSubQuery
	if since != nil && !since.IsZero() {
		guests = GuestManager.RawQuery().SubQuery()
	} else {
		guests = GuestManager.Query().SubQuery()
	}
	guestBackupSubQuery := GuestManager.Query(
		"id",
		"vcpu_count",
		"vmem_size",
	).IsNotEmpty("backup_host_id").SubQuery()

	q := guests.Query(sqlchemy.COUNT("total_guest_count"),
		sqlchemy.SUM("total_cpu_count", guests.Field("vcpu_count")),
		sqlchemy.SUM("total_mem_size", guests.Field("vmem_size")),
		sqlchemy.SUM("total_disk_size", diskSubQuery.Field("guest_disk_size")),
		sqlchemy.SUM("total_isolated_count", isoDevSubQuery.Field("device_sum")),
		sqlchemy.SUM("total_backup_disk_size", diskBackupSubQuery.Field("guest_disk_size")),
		sqlchemy.SUM("total_backup_cpu_count", guestBackupSubQuery.Field("vcpu_count")),
		sqlchemy.SUM("total_backup_mem_size", guestBackupSubQuery.Field("vmem_size")),
		sqlchemy.COUNT("total_backup_guest_count", guestBackupSubQuery.Field("id")),
	)

	q = q.LeftJoin(guestBackupSubQuery, sqlchemy.Equals(guestBackupSubQuery.Field("id"), guests.Field("id")))

	q = q.LeftJoin(diskSubQuery, sqlchemy.Equals(diskSubQuery.Field("guest_id"), guests.Field("id")))
	q = q.LeftJoin(diskBackupSubQuery, sqlchemy.Equals(diskBackupSubQuery.Field("guest_id"), guests.Field("id")))

	q = q.LeftJoin(isoDevSubQuery, sqlchemy.Equals(isoDevSubQuery.Field("guest_id"), guests.Field("id")))

	if len(rangeObjs) > 0 || len(hostTypes) > 0 || len(resourceTypes) > 0 || len(providers) > 0 || len(brands) > 0 || len(cloudEnv) > 0 {
		q = filterGuestByRange(q, rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv)
	}

	switch scope {
	case rbacutils.ScopeSystem:
	case rbacutils.ScopeDomain:
		q = q.Filter(sqlchemy.Equals(guests.Field("domain_id"), ownerId.GetProjectDomainId()))
	case rbacutils.ScopeProject:
		q = q.Filter(sqlchemy.Equals(guests.Field("tenant_id"), ownerId.GetProjectId()))
	}

	if len(status) > 0 {
		q = q.Filter(sqlchemy.In(guests.Field("status"), status))
	}
	if len(hypervisors) > 0 {
		q = q.Filter(sqlchemy.In(guests.Field("hypervisor"), hypervisors))
	}

	if pendingDelete {
		q = q.Filter(sqlchemy.IsTrue(guests.Field("pending_deleted")))
	} else {
		q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(guests.Field("pending_deleted")), sqlchemy.IsFalse(guests.Field("pending_deleted"))))
	}

	if since != nil && !since.IsZero() {
		q = q.Filter(sqlchemy.GT(guests.Field("created_at"), *since))
	}

	return q, guests
}

func (self *SGuest) getDefaultNetworkConfig() *api.NetworkConfig {
	netConf := api.NetworkConfig{}
	netConf.BwLimit = options.Options.DefaultBandwidth
	osProf := self.GetOSProfile()
	netConf.Driver = osProf.NetDriver
	return &netConf
}

func (self *SGuest) CreateNetworksOnHost(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	host *SHost,
	netArray []*api.NetworkConfig,
	pendingUsage quotas.IQuota,
	candidateNets []*schedapi.CandidateNet,
) error {
	if len(netArray) == 0 {
		netConfig := self.getDefaultNetworkConfig()
		_, err := self.attach2RandomNetwork(ctx, userCred, host, netConfig, pendingUsage)
		return errors.Wrap(err, "self.attach2RandomNetwork")
	}
	for idx := range netArray {
		netConfig, err := parseNetworkInfo(userCred, netArray[idx])
		if err != nil {
			return errors.Wrapf(err, "parseNetworkInfo at %d", idx)
		}
		var candidateNet *schedapi.CandidateNet
		if len(candidateNets) > idx {
			candidateNet = candidateNets[idx]
		}
		networkIds := []string{}
		if candidateNet != nil {
			networkIds = candidateNet.NetworkIds
		}
		_, err = self.attach2NetworkDesc(ctx, userCred, host, netConfig, pendingUsage, networkIds)
		if err != nil {
			return errors.Wrap(err, "self.attach2NetworkDesc")
		}
	}
	return nil
}

func (self *SGuest) attach2NetworkDesc(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	host *SHost,
	netConfig *api.NetworkConfig,
	pendingUsage quotas.IQuota,
	candiateNetIds []string,
) ([]SGuestnetwork, error) {
	var gns []SGuestnetwork
	var errs []error

	tryNetworkIds := []string{}
	if len(netConfig.Network) > 0 {
		tryNetworkIds = append(tryNetworkIds, netConfig.Network)
	}
	if len(candiateNetIds) > 0 {
		// suggestion by scheduler
		tryNetworkIds = append(tryNetworkIds, candiateNetIds...)
	}

	if len(tryNetworkIds) > 0 {
		for _, tryNetwork := range tryNetworkIds {
			var err error
			netConfig.Network = tryNetwork
			gns, err = self.attach2NamedNetworkDesc(ctx, userCred, host, netConfig, pendingUsage)
			if err == nil {
				return gns, nil
			}
			errs = append(errs, err)
		}
		return nil, errors.NewAggregate(errs)
	} else {
		netConfig.Network = ""
		return self.attach2RandomNetwork(ctx, userCred, host, netConfig, pendingUsage)
	}
}

func (self *SGuest) attach2NamedNetworkDesc(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, netConfig *api.NetworkConfig, pendingUsage quotas.IQuota) ([]SGuestnetwork, error) {
	driver := self.GetDriver()
	net, nicConfs, allocDir, reuseAddr := driver.GetNamedNetworkConfiguration(self, ctx, userCred, host, netConfig)
	if net != nil {
		if len(nicConfs) == 0 {
			return nil, fmt.Errorf("no avaialble network interface?")
		}
		gn, err := self.Attach2Network(ctx, userCred, Attach2NetworkArgs{
			Network:             net,
			PendingUsage:        pendingUsage,
			IpAddr:              netConfig.Address,
			NicDriver:           netConfig.Driver,
			BwLimit:             netConfig.BwLimit,
			Virtual:             netConfig.Vip,
			TryReserved:         netConfig.Reserved,
			AllocDir:            allocDir,
			RequireDesignatedIP: netConfig.RequireDesignatedIP,
			UseDesignatedIP:     reuseAddr,
			NicConfs:            nicConfs,
		})
		if err != nil {
			log.Errorf("Attach2Network fail %s", err)
			return nil, err
		} else {
			return gn, nil
		}
	} else {
		return nil, fmt.Errorf("Network %s not available", netConfig.Network)
	}
}

func (self *SGuest) attach2RandomNetwork(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, netConfig *api.NetworkConfig, pendingUsage quotas.IQuota) ([]SGuestnetwork, error) {
	driver := self.GetDriver()
	return driver.Attach2RandomNetwork(self, ctx, userCred, host, netConfig, pendingUsage)
}

func (self *SGuest) CreateDisksOnHost(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	host *SHost,
	disks []*api.DiskConfig,
	pendingUsage quotas.IQuota,
	inheritBilling bool,
	isWithServerCreate bool,
	candidateDisks []*schedapi.CandidateDisk,
	backupCandidateDisks []*schedapi.CandidateDisk,
	autoAttach bool,
) error {
	for idx := 0; idx < len(disks); idx += 1 {
		diskConfig, err := parseDiskInfo(ctx, userCred, disks[idx])
		if err != nil {
			return err
		}
		var candidateDisk *schedapi.CandidateDisk
		var backupCandidateDisk *schedapi.CandidateDisk
		if len(candidateDisks) > idx {
			candidateDisk = candidateDisks[idx]
		}
		if len(backupCandidateDisks) != 0 && len(backupCandidateDisks) > idx {
			backupCandidateDisk = backupCandidateDisks[idx]
		}
		disk, err := self.createDiskOnHost(ctx, userCred, host, diskConfig, pendingUsage, inheritBilling, isWithServerCreate, candidateDisk, backupCandidateDisk, autoAttach)
		if err != nil {
			return err
		}
		diskConfig.DiskId = disk.Id
		disks[idx] = diskConfig
	}
	return nil
}

func (self *SGuest) createDiskOnStorage(ctx context.Context, userCred mcclient.TokenCredential, storage *SStorage,
	diskConfig *api.DiskConfig, pendingUsage quotas.IQuota, inheritBilling bool, isWithServerCreate bool) (*SDisk, error) {
	lockman.LockObject(ctx, storage)
	defer lockman.ReleaseObject(ctx, storage)

	lockman.LockClass(ctx, QuotaManager, self.ProjectId)
	defer lockman.ReleaseClass(ctx, QuotaManager, self.ProjectId)

	diskName := fmt.Sprintf("vdisk-%s-%d", self.Name, time.Now().UnixNano())

	billingType := billing_api.BILLING_TYPE_POSTPAID
	billingCycle := ""
	if inheritBilling {
		billingType = self.BillingType
		billingCycle = self.BillingCycle
	}

	autoDelete := false
	if storage.IsLocal() || billingType == billing_api.BILLING_TYPE_PREPAID || isWithServerCreate {
		autoDelete = true
	}
	disk, err := storage.createDisk(ctx, diskName, diskConfig, userCred, self.GetOwnerId(), autoDelete, self.IsSystem,
		billingType, billingCycle)

	if err != nil {
		return nil, err
	}

	if pendingUsage != nil {
		cancelUsage := SQuota{}
		cancelUsage.Storage = disk.DiskSize
		keys, err := self.GetQuotaKeys()
		if err != nil {
			return nil, err
		}
		cancelUsage.SetKeys(keys)
		err = quotas.CancelPendingUsage(ctx, userCred, pendingUsage, &cancelUsage, true)
		if err != nil {
			return nil, err
		}
	}

	return disk, nil
}

func (self *SGuest) ChooseHostStorage(host *SHost, diskConfig *api.DiskConfig, candidate *schedapi.CandidateDisk) (*SStorage, error) {
	if candidate == nil || len(candidate.StorageIds) == 0 {
		return self.GetDriver().ChooseHostStorage(host, self, diskConfig, nil)
	}
	return self.GetDriver().ChooseHostStorage(host, self, diskConfig, candidate.StorageIds)
}

func (self *SGuest) createDiskOnHost(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	host *SHost,
	diskConfig *api.DiskConfig,
	pendingUsage quotas.IQuota,
	inheritBilling bool,
	isWithServerCreate bool,
	candidate *schedapi.CandidateDisk,
	backupCandidate *schedapi.CandidateDisk,
	autoAttach bool,
) (*SDisk, error) {
	var (
		storage *SStorage
		err     error
	)
	if len(diskConfig.Storage) > 0 {
		_storage, err := StorageManager.FetchByIdOrName(userCred, diskConfig.Storage)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("storage", diskConfig.Storage)
			}
			return nil, fmt.Errorf("get storage(%s) error: %v", diskConfig.Storage, err)
		}
		storage = _storage.(*SStorage)
	} else {
		storage, err = self.ChooseHostStorage(host, diskConfig, candidate)
		if err != nil {
			return nil, errors.Wrap(err, "ChooseHostStorage")
		}
	}
	if storage == nil {
		return nil, fmt.Errorf("No storage on %s to create disk for %s", host.GetName(), diskConfig.Backend)
	}
	log.Debugf("Choose storage %s:%s for disk %#v", storage.Name, storage.Id, diskConfig)
	disk, err := self.createDiskOnStorage(ctx, userCred, storage, diskConfig, pendingUsage, inheritBilling, isWithServerCreate)
	if err != nil {
		return nil, err
	}
	if len(self.BackupHostId) > 0 {
		backupHost := HostManager.FetchHostById(self.BackupHostId)
		backupStorage, err := self.ChooseHostStorage(backupHost, diskConfig, backupCandidate)
		if err != nil {
			return nil, errors.Wrap(err, "ChooseHostStorage")
		}
		diff, err := db.Update(disk, func() error {
			disk.BackupStorageId = backupStorage.Id
			return nil
		})
		if err != nil {
			log.Errorf("Disk save backup storage error")
			return disk, err
		}
		db.OpsLog.LogEvent(disk, db.ACT_UPDATE, diff, userCred)
	}
	if autoAttach {
		err = self.attach2Disk(ctx, disk, userCred, diskConfig.Driver, diskConfig.Cache, diskConfig.Mountpoint)
	}
	return disk, err
}

func (self *SGuest) CreateIsolatedDeviceOnHost(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, devs []*api.IsolatedDeviceConfig, pendingUsage quotas.IQuota) error {
	for _, devConfig := range devs {
		err := self.createIsolatedDeviceOnHost(ctx, userCred, host, devConfig, pendingUsage)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SGuest) createIsolatedDeviceOnHost(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, devConfig *api.IsolatedDeviceConfig, pendingUsage quotas.IQuota) error {
	lockman.LockClass(ctx, QuotaManager, self.ProjectId)
	defer lockman.ReleaseClass(ctx, QuotaManager, self.ProjectId)

	err := IsolatedDeviceManager.attachHostDeviceToGuestByDesc(ctx, self, host, devConfig, userCred)
	if err != nil {
		return err
	}

	cancelUsage := SQuota{IsolatedDevice: 1}
	keys, err := self.GetQuotaKeys()
	if err != nil {
		return err
	}
	cancelUsage.SetKeys(keys)
	err = quotas.CancelPendingUsage(ctx, userCred, pendingUsage, &cancelUsage, true) // success
	return err
}

func (self *SGuest) JoinGroups(ctx context.Context, userCred mcclient.TokenCredential, groupIds []string) error {
	for _, id := range groupIds {
		_, err := GroupguestManager.Attach(ctx, id, self.Id)
		if err != nil {
			return err
		}
	}
	return nil
}

type SGuestDiskCategory struct {
	Root *SDisk
	Swap []*SDisk
	Data []*SDisk
}

func (self *SGuest) CategorizeDisks() SGuestDiskCategory {
	diskCat := SGuestDiskCategory{}
	guestdisks := self.GetDisks()
	if guestdisks == nil {
		log.Errorf("no disk for this server!!!")
		return diskCat
	}
	for _, gd := range guestdisks {
		if diskCat.Root == nil {
			diskCat.Root = gd.GetDisk()
		} else {
			disk := gd.GetDisk()
			if disk.FsFormat == "swap" {
				diskCat.Swap = append(diskCat.Swap, disk)
			} else {
				diskCat.Data = append(diskCat.Data, disk)
			}
		}
	}
	return diskCat
}

type SGuestNicCategory struct {
	InternalNics []SGuestnetwork
	ExternalNics []SGuestnetwork
}

func (self *SGuest) CategorizeNics() SGuestNicCategory {
	netCat := SGuestNicCategory{}

	guestnics, err := self.GetNetworks("")
	if err != nil {
		log.Errorf("no nics for this server!!! %s", err)
		return netCat
	}

	for _, gn := range guestnics {
		if gn.IsExit() {
			netCat.ExternalNics = append(netCat.ExternalNics, gn)
		} else {
			netCat.InternalNics = append(netCat.InternalNics, gn)
		}
	}
	return netCat
}

func (self *SGuest) LeaveAllGroups(ctx context.Context, userCred mcclient.TokenCredential) {
	groupGuests := make([]SGroupguest, 0)
	q := GroupguestManager.Query()
	err := q.Filter(sqlchemy.Equals(q.Field("guest_id"), self.Id)).All(&groupGuests)
	if err != nil {
		log.Errorf("query by guest_id %s: %v", self.Id, err)
		return
	}
	for _, gg := range groupGuests {
		gg.SetModelManager(GroupguestManager, &gg)
		gg.Delete(context.Background(), userCred)
		var group SGroup
		gq := GroupManager.Query()
		err := gq.Filter(sqlchemy.Equals(gq.Field("id"), gg.GroupId)).First(&group)
		if err != nil {
			log.Errorf("get by group id %s: %v", gg.GroupId, err)
			return
		}
		group.SetModelManager(GroupManager, &group)
		db.OpsLog.LogDetachEvent(ctx, self, &group, userCred, nil)
	}
}

func (self *SGuest) DetachAllNetworks(ctx context.Context, userCred mcclient.TokenCredential) error {
	// from clouds.models.portmaps import Portmaps
	// Portmaps.delete_guest_network_portmaps(self, user_cred)
	gns, err := self.GetNetworks("")
	if err != nil {
		return err
	}
	return GuestnetworkManager.DeleteGuestNics(ctx, userCred, gns, false)
}

func (self *SGuest) EjectIso(userCred mcclient.TokenCredential) bool {
	cdrom := self.getCdrom(false)
	if cdrom != nil && len(cdrom.ImageId) > 0 {
		imageId := cdrom.ImageId
		if cdrom.ejectIso() {
			db.OpsLog.LogEvent(self, db.ACT_ISO_DETACH, imageId, userCred)
			return true
		}
	}
	return false
}

func (self *SGuest) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	// self.SVirtualResourceBase.Delete(ctx, userCred)
	// override
	log.Infof("guest delete do nothing")
	return nil
}

func (self *SGuest) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SGuest) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	overridePendingDelete := false
	purge := false
	if query != nil {
		overridePendingDelete = jsonutils.QueryBoolean(query, "override_pending_delete", false)
		purge = jsonutils.QueryBoolean(query, "purge", false)
	}
	if (overridePendingDelete || purge) && !db.IsAdminAllowDelete(userCred, self) {
		return false
	}
	return self.IsOwner(userCred) || db.IsAdminAllowDelete(userCred, self)
}

// 删除虚拟机
func (self *SGuest) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query api.ServerDeleteInput, data jsonutils.JSONObject) error {
	return self.StartDeleteGuestTask(ctx, userCred, "", query)
}

func (self *SGuest) DeleteAllDisksInDB(ctx context.Context, userCred mcclient.TokenCredential) error {
	for _, guestdisk := range self.GetDisks() {
		disk := guestdisk.GetDisk()
		err := guestdisk.Detach(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "guestdisk.Detach guest_id: %s disk_id: %s", guestdisk.GuestId, guestdisk.DiskId)
		}

		if disk != nil {
			cnt, err := disk.GetGuestDiskCount()
			if err != nil {
				return errors.Wrap(err, "disk.GetGuestDiskCount")
			}
			if cnt == 0 {
				db.OpsLog.LogEvent(disk, db.ACT_DELETE, nil, userCred)
				db.OpsLog.LogEvent(disk, db.ACT_DELOCATE, nil, userCred)
				err = disk.RealDelete(ctx, userCred)
				if err != nil {
					return errors.Wrap(err, "disk.RealDelete")
				}
			}
		}
	}
	return nil
}

func (self *SGuest) DeleteAllInstanceSnapshotInDB(ctx context.Context, userCred mcclient.TokenCredential) error {
	isps, err := self.GetInstanceSnapshots()
	if err != nil {
		return errors.Wrap(err, "unable to GetInstanceSnapshots")
	}
	for i := range isps {
		err = isps[i].RealDelete(ctx, userCred)
		return errors.Wrapf(err, "unable to real delete instance snapshot %q for guest %q", isps[i].GetName(), self.GetId())
	}
	return nil
}

func (self *SGuest) isNeedDoResetPasswd() bool {
	guestdisks := self.GetDisks()
	disk := guestdisks[0].GetDisk()
	if len(disk.SnapshotId) > 0 {
		return false
	}
	return true
}

func (self *SGuest) GetDeployConfigOnHost(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, params *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	config := jsonutils.NewDict()

	desc, err := self.GetDriver().GetJsonDescAtHost(ctx, userCred, self, host, params)
	if err != nil {
		return nil, errors.Wrapf(err, "GetJsonDescAtHost")
	}
	config.Add(desc, "desc")

	deploys, err := cmdline.FetchDeployConfigsByJSON(params)
	if err != nil {
		return nil, err
	}

	if len(deploys) > 0 {
		config.Add(jsonutils.Marshal(deploys), "deploys")
	}

	deployAction, _ := params.GetString("deploy_action")
	if len(deployAction) == 0 {
		deployAction = "deploy"
	}

	config.Add(jsonutils.NewBool(jsonutils.QueryBoolean(params, "enable_cloud_init", false)), "enable_cloud_init")

	if account, _ := params.GetString("login_account"); len(account) > 0 {
		config.Set("login_account", jsonutils.NewString(account))
	}

	resetPasswd := jsonutils.QueryBoolean(params, "reset_password", true)
	if deployAction == "create" && resetPasswd {
		resetPasswd = self.isNeedDoResetPasswd()
	}

	if resetPasswd {
		config.Add(jsonutils.JSONTrue, "reset_password")
		passwd, _ := params.GetString("password")
		if len(passwd) > 0 {
			config.Add(jsonutils.NewString(passwd), "password")
		}
		keypair := self.getKeypair()
		if keypair != nil {
			config.Add(jsonutils.NewString(keypair.PublicKey), "public_key")
		}
		deletePubKey, _ := params.GetString("delete_public_key")
		if len(deletePubKey) > 0 {
			config.Add(jsonutils.NewString(deletePubKey), "delete_public_key")
		}
	} else {
		config.Add(jsonutils.JSONFalse, "reset_password")
	}

	// add default public keys
	_, adminPubKey, err := sshkeys.GetSshAdminKeypair(ctx)
	if err != nil {
		log.Errorf("fail to get ssh admin public key %s", err)
	}

	_, projPubKey, err := sshkeys.GetSshProjectKeypair(ctx, self.ProjectId)

	if err != nil {
		log.Errorf("fail to get ssh project public key %s", err)
	}

	config.Add(jsonutils.NewString(adminPubKey), "admin_public_key")
	config.Add(jsonutils.NewString(projPubKey), "project_public_key")

	config.Add(jsonutils.NewString(deployAction), "action")

	onFinish := "shutdown"
	if jsonutils.QueryBoolean(params, "auto_start", false) || jsonutils.QueryBoolean(params, "restart", false) {
		onFinish = "none"
	} else if utils.IsInStringArray(self.Status, []string{api.VM_ADMIN}) {
		onFinish = "none"
	}

	config.Add(jsonutils.NewString(onFinish), "on_finish")

	return config, nil
}

func (self *SGuest) getVga() string {
	if utils.IsInStringArray(self.Vga, []string{"cirrus", "vmware", "qxl"}) {
		return self.Vga
	}
	return "std"
}

func (self *SGuest) GetVdi() string {
	if utils.IsInStringArray(self.Vdi, []string{"vnc", "spice"}) {
		return self.Vdi
	}
	return "vnc"
}

func (self *SGuest) getMachine() string {
	if utils.IsInStringArray(self.Machine, []string{"pc", "q35"}) {
		return self.Machine
	}
	return "pc"
}

func (self *SGuest) getBios() string {
	if utils.IsInStringArray(self.Bios, []string{"BIOS", "UEFI"}) {
		return self.Bios
	}
	return "BIOS"
}

func (self *SGuest) getKvmOptions() string {
	return self.GetMetadata("kvm", nil)
}

func (self *SGuest) getExtraOptions() jsonutils.JSONObject {
	return self.GetMetadataJson("extra_options", nil)
}

func (self *SGuest) GetJsonDescAtHypervisor(ctx context.Context, host *SHost) *jsonutils.JSONDict {
	desc := jsonutils.NewDict()

	desc.Add(jsonutils.NewString(self.Name), "name")
	if len(self.Description) > 0 {
		desc.Add(jsonutils.NewString(self.Description), "description")
	}
	desc.Add(jsonutils.NewString(self.Id), "uuid")
	desc.Add(jsonutils.NewInt(int64(self.VmemSize)), "mem")
	desc.Add(jsonutils.NewInt(int64(self.VcpuCount)), "cpu")
	desc.Add(jsonutils.NewString(self.getVga()), "vga")
	desc.Add(jsonutils.NewString(self.GetVdi()), "vdi")
	desc.Add(jsonutils.NewString(self.getMachine()), "machine")
	desc.Add(jsonutils.NewString(self.getBios()), "bios")
	desc.Add(jsonutils.NewString(self.BootOrder), "boot_order")

	desc.Add(jsonutils.NewBool(self.SrcIpCheck.Bool()), "src_ip_check")
	desc.Add(jsonutils.NewBool(self.SrcMacCheck.Bool()), "src_mac_check")

	if len(self.BackupHostId) > 0 {
		if self.HostId == host.Id {
			desc.Set("is_master", jsonutils.JSONTrue)
		} else if self.BackupHostId == host.Id {
			desc.Set("is_slave", jsonutils.JSONTrue)
		}
	}
	desc.Set("host_id", jsonutils.NewString(host.Id))

	// isolated devices
	isolatedDevs := IsolatedDeviceManager.generateJsonDescForGuest(self)
	desc.Add(jsonutils.NewArray(isolatedDevs...), "isolated_devices")

	// nics, domain
	jsonNics := make([]jsonutils.JSONObject, 0)
	nics, _ := self.GetNetworks("")
	domain := options.Options.DNSDomain
	for _, nic := range nics {
		nicDesc := nic.getJsonDescAtHost(ctx, host)
		jsonNics = append(jsonNics, nicDesc)
		nicDomain, _ := nicDesc.GetString("domain")
		if len(nicDomain) > 0 && len(domain) == 0 {
			domain = nicDomain
		}
	}
	desc.Add(jsonutils.NewArray(jsonNics...), "nics")
	desc.Add(jsonutils.NewString(domain), "domain")

	// disks
	jsonDisks := make([]jsonutils.JSONObject, 0)
	disks := self.GetDisks()
	if disks != nil && len(disks) > 0 {
		for _, disk := range disks {
			diskDesc := disk.GetJsonDescAtHost(host)
			jsonDisks = append(jsonDisks, diskDesc)
		}
	}
	desc.Add(jsonutils.NewArray(jsonDisks...), "disks")

	// cdrom
	cdrom := self.getCdrom(false)
	if cdrom != nil {
		cdDesc := cdrom.getJsonDesc()
		if cdDesc != nil {
			desc.Add(cdDesc, "cdrom")
		}
	}

	// tenant
	tc, _ := self.GetTenantCache(ctx)
	if tc != nil {
		desc.Add(jsonutils.NewString(tc.GetName()), "tenant")
		desc.Add(jsonutils.NewString(tc.DomainId), "domain_id")
		desc.Add(jsonutils.NewString(tc.Domain), "project_domain")
	}
	desc.Add(jsonutils.NewString(self.ProjectId), "tenant_id")

	// flavor
	// desc.Add(jsonuitls.NewString(self.getFlavorName()), "flavor")

	keypair := self.getKeypair()
	if keypair != nil {
		desc.Add(jsonutils.NewString(keypair.Name), "keypair")
		desc.Add(jsonutils.NewString(keypair.PublicKey), "pubkey")
	}

	netRoles := self.getNetworkRoles()
	if netRoles != nil && len(netRoles) > 0 {
		desc.Add(jsonutils.NewStringArray(netRoles), "network_roles")
	}

	/*
		secGrp := self.getSecgroup()
		if secGrp != nil {
			desc.Add(jsonutils.NewString(secGrp.Name), "secgroup")
		}
	*/

	secgroups, _ := self.getSecgroupJson()
	if secgroups != nil {
		desc.Add(jsonutils.NewArray(secgroups...), "secgroups")
	}

	/*
		TODO
		srs := self.getSecurityRuleSet()
		if srs.estimatedSinglePortRuleCount() <= options.FirewallFlowCountLimit {
	*/

	rules := self.getSecurityGroupsRules()
	if len(rules) > 0 {
		desc.Add(jsonutils.NewString(rules), "security_rules")
	}
	rules = self.getAdminSecurityRules()
	if len(rules) > 0 {
		desc.Add(jsonutils.NewString(rules), "admin_security_rules")
	}

	extraOptions := self.getExtraOptions()
	if extraOptions != nil {
		desc.Add(extraOptions, "extra_options")
	}

	kvmOptions := self.getKvmOptions()
	if len(kvmOptions) > 0 {
		desc.Add(jsonutils.NewString(kvmOptions), "kvm")
	}

	zone := self.getZone()
	if zone != nil {
		desc.Add(jsonutils.NewString(zone.Id), "zone_id")
		desc.Add(jsonutils.NewString(zone.Name), "zone")
	}

	os := self.GetOS()
	if len(os) > 0 {
		desc.Add(jsonutils.NewString(os), "os_name")
	}

	meta, _ := self.GetAllMetadata(nil)
	desc.Add(jsonutils.Marshal(meta), "metadata")

	userData := meta["user_data"]
	if len(userData) > 0 {
		decodeData, _ := userdata.Decode(userData)
		if len(decodeData) > 0 {
			userData = decodeData
		}
		desc.Add(jsonutils.NewString(userData), "user_data")
	}

	if self.PendingDeleted {
		desc.Add(jsonutils.JSONTrue, "pending_deleted")
	} else {
		desc.Add(jsonutils.JSONFalse, "pending_deleted")
	}

	// add scaling group
	sggs, err := ScalingGroupGuestManager.Fetch("", self.Id)
	if err == nil && len(sggs) > 0 {
		desc.Add(jsonutils.NewString(sggs[0].ScalingGroupId), "scaling_group_id")
	}

	return desc
}

func (self *SGuest) GetJsonDescAtBaremetal(ctx context.Context, host *SHost) *jsonutils.JSONDict {
	desc := jsonutils.NewDict()

	desc.Add(jsonutils.NewString(self.Name), "name")
	if len(self.Description) > 0 {
		desc.Add(jsonutils.NewString(self.Description), "description")
	}
	desc.Add(jsonutils.NewString(self.Id), "uuid")
	desc.Add(jsonutils.NewInt(int64(self.VmemSize)), "mem")
	desc.Add(jsonutils.NewInt(int64(self.VcpuCount)), "cpu")
	diskConf := host.getDiskConfig()
	if !gotypes.IsNil(diskConf) {
		desc.Add(diskConf, "disk_config")
	}

	jsonNics := make([]jsonutils.JSONObject, 0)
	jsonStandbyNics := make([]jsonutils.JSONObject, 0)

	netifs := host.GetNetInterfaces()
	domain := options.Options.DNSDomain

	if netifs != nil && len(netifs) > 0 {
		for _, nic := range netifs {
			nicDesc := nic.getServerJsonDesc()
			if nicDesc.Contains("ip") {
				jsonNics = append(jsonNics, nicDesc)
				nicDomain, _ := nicDesc.GetString("domain")
				if len(nicDomain) > 0 && len(domain) == 0 {
					domain = nicDomain
				}
			} else {
				jsonStandbyNics = append(jsonStandbyNics, nicDesc)
			}
		}
	}
	desc.Add(jsonutils.NewArray(jsonNics...), "nics")
	desc.Add(jsonutils.NewArray(jsonStandbyNics...), "nics_standby")
	desc.Add(jsonutils.NewString(domain), "domain")

	jsonDisks := make([]jsonutils.JSONObject, 0)
	disks := self.GetDisks()
	if disks != nil && len(disks) > 0 {
		for _, disk := range disks {
			diskDesc := disk.GetJsonDescAtHost(host)
			jsonDisks = append(jsonDisks, diskDesc)
		}
	}
	desc.Add(jsonutils.NewArray(jsonDisks...), "disks")

	tc, _ := self.GetTenantCache(ctx)
	if tc != nil {
		desc.Add(jsonutils.NewString(tc.GetName()), "tenant")
		desc.Add(jsonutils.NewString(tc.DomainId), "domain_id")
		desc.Add(jsonutils.NewString(tc.Domain), "project_domain")
	}

	desc.Add(jsonutils.NewString(self.ProjectId), "tenant_id")

	keypair := self.getKeypair()
	if keypair != nil {
		desc.Add(jsonutils.NewString(keypair.Name), "keypair")
		desc.Add(jsonutils.NewString(keypair.PublicKey), "pubkey")
	}

	netRoles := self.getNetworkRoles()
	if netRoles != nil && len(netRoles) > 0 {
		desc.Add(jsonutils.NewStringArray(netRoles), "network_roles")
	}

	rules := self.getSecurityGroupsRules()
	if len(rules) > 0 {
		desc.Add(jsonutils.NewString(rules), "security_rules")
	}
	rules = self.getAdminSecurityRules()
	if len(rules) > 0 {
		desc.Add(jsonutils.NewString(rules), "admin_security_rules")
	}

	zone := self.getZone()
	if zone != nil {
		desc.Add(jsonutils.NewString(zone.Id), "zone_id")
		desc.Add(jsonutils.NewString(zone.Name), "zone")
	}

	os := self.GetOS()
	if len(os) > 0 {
		desc.Add(jsonutils.NewString(os), "os_name")
	}

	meta, _ := self.GetAllMetadata(nil)
	desc.Add(jsonutils.Marshal(meta), "metadata")

	userData := meta["user_data"]
	if len(userData) > 0 {
		desc.Add(jsonutils.NewString(userData), "user_data")
	}

	if self.PendingDeleted {
		desc.Add(jsonutils.JSONTrue, "pending_deleted")
	} else {
		desc.Add(jsonutils.JSONFalse, "pending_deleted")
	}

	return desc
}

func (self *SGuest) getNetworkRoles() []string {
	key := db.Metadata.GetSysadminKey("network_role")
	roleStr := self.GetMetadata(key, auth.AdminCredential())
	if len(roleStr) > 0 {
		return strings.Split(roleStr, ",")
	}
	return nil
}

func (manager *SGuestManager) FetchGuestById(guestId string) *SGuest {
	guest, err := manager.FetchById(guestId)
	if err != nil {
		log.Errorf("FetchById fail %s", err)
		return nil
	}
	return guest.(*SGuest)
}

func (manager *SGuestManager) GetSpecShouldCheckStatus(query *jsonutils.JSONDict) (bool, error) {
	return true, nil
}

func (self *SGuest) GetSpec(checkStatus bool) *jsonutils.JSONDict {
	if checkStatus {
		if utils.IsInStringArray(self.Status, []string{api.VM_SCHEDULE_FAILED}) {
			return nil
		}
	}
	spec := jsonutils.NewDict()
	spec.Set("cpu", jsonutils.NewInt(int64(self.VcpuCount)))
	spec.Set("mem", jsonutils.NewInt(int64(self.VmemSize)))

	// get disk spec
	guestdisks := self.GetDisks()
	diskSpecs := jsonutils.NewArray()
	for _, guestdisk := range guestdisks {
		info := guestdisk.ToDiskInfo()
		diskSpec := jsonutils.NewDict()
		diskSpec.Set("size", jsonutils.NewInt(info.Size))
		diskSpec.Set("backend", jsonutils.NewString(info.Backend))
		diskSpec.Set("medium_type", jsonutils.NewString(info.MediumType))
		diskSpec.Set("disk_type", jsonutils.NewString(info.DiskType))
		diskSpecs.Add(diskSpec)
	}
	spec.Set("disk", diskSpecs)

	// get nic spec
	guestnics, _ := self.GetNetworks("")

	nicSpecs := jsonutils.NewArray()
	for _, guestnic := range guestnics {
		nicSpec := jsonutils.NewDict()
		nicSpec.Set("bandwidth", jsonutils.NewInt(int64(guestnic.getBandwidth())))
		t := "int"
		if guestnic.IsExit() {
			t = "ext"
		}
		nicSpec.Set("type", jsonutils.NewString(t))
		nicSpecs.Add(nicSpec)
	}
	spec.Set("nic", nicSpecs)

	// get isolate device spec
	guestgpus := self.GetIsolatedDevices()
	gpuSpecs := jsonutils.NewArray()
	for _, guestgpu := range guestgpus {
		if strings.HasPrefix(guestgpu.DevType, "GPU") {
			gs := guestgpu.GetSpec(false)
			if gs != nil {
				gpuSpecs.Add(gs)
			}
		}
	}
	spec.Set("gpu", gpuSpecs)
	return spec
}

func (manager *SGuestManager) GetSpecIdent(spec *jsonutils.JSONDict) []string {
	cpuCount, _ := spec.Int("cpu")
	memSize, _ := spec.Int("mem")
	memSizeMB, _ := utils.GetSizeMB(fmt.Sprintf("%d", memSize), "M")
	specKeys := []string{
		fmt.Sprintf("cpu:%d", cpuCount),
		fmt.Sprintf("mem:%dM", memSizeMB),
	}

	countKey := func(kf func(*jsonutils.JSONDict) string, dataArray jsonutils.JSONObject) map[string]int64 {
		countMap := make(map[string]int64)
		datas, _ := dataArray.GetArray()
		for _, data := range datas {
			key := kf(data.(*jsonutils.JSONDict))
			if count, ok := countMap[key]; !ok {
				countMap[key] = 1
			} else {
				count++
				countMap[key] = count
			}
		}
		return countMap
	}

	kfuncs := map[string]func(*jsonutils.JSONDict) string{
		"disk": func(data *jsonutils.JSONDict) string {
			backend, _ := data.GetString("backend")
			mediumType, _ := data.GetString("medium_type")
			size, _ := data.Int("size")
			sizeGB, _ := utils.GetSizeGB(fmt.Sprintf("%d", size), "M")
			return fmt.Sprintf("disk:%s_%s_%dG", backend, mediumType, sizeGB)
		},
		"nic": func(data *jsonutils.JSONDict) string {
			typ, _ := data.GetString("type")
			bw, _ := data.Int("bandwidth")
			return fmt.Sprintf("nic:%s_%dM", typ, bw)
		},
		"gpu": func(data *jsonutils.JSONDict) string {
			vendor, _ := data.GetString("vendor")
			model, _ := data.GetString("model")
			return fmt.Sprintf("gpu:%s_%s", vendor, model)
		},
	}

	for sKey, kf := range kfuncs {
		sArrary, err := spec.Get(sKey)
		if err != nil {
			log.Errorf("Get key %s array error: %v", sKey, err)
			continue
		}
		for key, count := range countKey(kf, sArrary) {
			specKeys = append(specKeys, fmt.Sprintf("%sx%d", key, count))
		}
	}
	return specKeys
}

func (self *SGuest) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := self.SVirtualResourceBase.GetShortDesc(ctx)
	desc.Set("mem", jsonutils.NewInt(int64(self.VmemSize)))
	desc.Set("cpu", jsonutils.NewInt(int64(self.VcpuCount)))

	address := jsonutils.NewString(strings.Join(self.GetRealIPs(), ","))
	desc.Set("ip_addr", address)

	if len(self.OsType) > 0 {
		desc.Add(jsonutils.NewString(self.OsType), "os_type")
	}
	if osDist := self.GetMetadata("os_distribution", nil); len(osDist) > 0 {
		desc.Add(jsonutils.NewString(osDist), "os_distribution")
	}
	if osVer := self.GetMetadata("os_version", nil); len(osVer) > 0 {
		desc.Add(jsonutils.NewString(osVer), "os_version")
	}

	templateId := self.GetTemplateId()
	if len(templateId) > 0 {
		desc.Set("template_id", jsonutils.NewString(templateId))
	}
	extBw := self.getBandwidth(true)
	intBw := self.getBandwidth(false)
	if extBw > 0 {
		desc.Set("ext_bandwidth", jsonutils.NewInt(int64(extBw)))
	}
	if intBw > 0 {
		desc.Set("int_bandwidth", jsonutils.NewInt(int64(intBw)))
	}

	if len(self.OsType) > 0 {
		desc.Add(jsonutils.NewString(self.OsType), "os_type")
	}

	if len(self.ExternalId) > 0 {
		desc.Add(jsonutils.NewString(self.ExternalId), "externalId")
	}

	desc.Set("hypervisor", jsonutils.NewString(self.GetHypervisor()))

	host := self.GetHost()

	spec := self.GetSpec(false)
	if self.GetHypervisor() == api.HYPERVISOR_BAREMETAL {
		if host != nil {
			hostSpec := host.GetSpec(false)
			hostSpecIdent := HostManager.GetSpecIdent(hostSpec)
			spec.Set("host_spec", jsonutils.NewString(strings.Join(hostSpecIdent, "/")))
		}
	}
	if spec != nil {
		desc.Update(spec)
	}

	var billingInfo SCloudBillingInfo

	if host != nil {
		desc.Set("host", jsonutils.NewString(host.Name))
		desc.Set("host_id", jsonutils.NewString(host.Id))
		billingInfo.SCloudProviderInfo = host.getCloudProviderInfo()
	}

	if len(self.BackupHostId) > 0 {
		backupHost := HostManager.FetchHostById(self.BackupHostId)
		if backupHost != nil {
			desc.Set("backup_host", jsonutils.NewString(backupHost.Name))
			desc.Set("backup_host_id", jsonutils.NewString(backupHost.Id))
		}
	}

	if priceKey := self.GetMetadata("ext:price_key", nil); len(priceKey) > 0 {
		billingInfo.PriceKey = priceKey
	}

	billingInfo.SBillingBaseInfo = self.getBillingBaseInfo()

	desc.Update(jsonutils.Marshal(billingInfo))

	return desc
}

func (self *SGuest) saveOsType(userCred mcclient.TokenCredential, osType string) error {
	diff, err := db.Update(self, func() error {
		self.OsType = osType
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)
	return err
}

type sDeployInfo struct {
	Os       string
	Account  string
	Key      string
	Distro   string
	Version  string
	Arch     string
	Language string
}

func (self *SGuest) SaveDeployInfo(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) {
	deployInfo := sDeployInfo{}
	data.Unmarshal(&deployInfo)
	info := make(map[string]interface{})
	if len(deployInfo.Os) > 0 {
		self.saveOsType(userCred, deployInfo.Os)
		info["os_name"] = deployInfo.Os
	}
	driver := self.GetDriver()
	if len(deployInfo.Account) > 0 {
		info["login_account"] = deployInfo.Account
		if len(deployInfo.Key) > 0 {
			info["login_key"] = deployInfo.Key
			if len(self.KeypairId) > 0 && !driver.IsSupportdDcryptPasswordFromSecretKey() { // Tencent Cloud does not support simultaneous setting of secret keys and passwords
				info["login_key"], _ = seclib2.EncryptBase64(self.GetKeypairPublicKey(), "")
			}
			info["login_key_timestamp"] = timeutils.UtcNow()
		} else {
			info["login_key"] = "none"
			info["login_key_timestamp"] = "none"
		}
	}
	if len(deployInfo.Distro) > 0 {
		info["os_distribution"] = deployInfo.Distro
	}
	if len(deployInfo.Version) > 0 {
		info["os_version"] = deployInfo.Version
	}
	if len(deployInfo.Arch) > 0 {
		info["os_arch"] = deployInfo.Arch
	}
	if len(deployInfo.Language) > 0 {
		info["os_language"] = deployInfo.Language
	}
	self.SetAllMetadata(ctx, info, userCred)
	self.saveOldPassword(ctx, userCred)
}

func (self *SGuest) isAllDisksReady() bool {
	ready := true
	disks := self.GetDisks()
	if disks == nil || len(disks) == 0 {
		return true
	}
	for i := 0; i < len(disks); i += 1 {
		disk := disks[i].GetDisk()
		if !(disk.isReady() || disk.Status == api.DISK_START_MIGRATE) {
			ready = false
			break
		}
	}
	return ready
}

func (self *SGuest) GetKeypairPublicKey() string {
	keypair := self.getKeypair()
	if keypair != nil {
		return keypair.PublicKey
	}
	return ""
}

func (manager *SGuestManager) GetIpInProjectWithName(projectId, name string, isExitOnly bool) []string {
	guestnics := GuestnetworkManager.Query().SubQuery()
	guests := manager.Query().SubQuery()
	networks := NetworkManager.Query().SubQuery()
	q := guestnics.Query(guestnics.Field("ip_addr")).Join(guests,
		sqlchemy.AND(
			sqlchemy.Equals(guests.Field("id"), guestnics.Field("guest_id")),
			sqlchemy.OR(sqlchemy.IsNull(guests.Field("pending_deleted")),
				sqlchemy.IsFalse(guests.Field("pending_deleted"))))).
		Join(networks, sqlchemy.Equals(networks.Field("id"), guestnics.Field("network_id"))).
		Filter(sqlchemy.Equals(guests.Field("name"), name)).
		Filter(sqlchemy.NotEquals(guestnics.Field("ip_addr"), "")).
		Filter(sqlchemy.IsNotNull(guestnics.Field("ip_addr"))).
		Filter(sqlchemy.IsNotNull(networks.Field("guest_gateway")))
	ips := make([]string, 0)
	rows, err := q.Rows()
	if err != nil {
		log.Errorf("Get guest ip with name query err: %v", err)
		return ips
	}
	defer rows.Close()
	for rows.Next() {
		var ip string
		err = rows.Scan(&ip)
		if err != nil {
			log.Errorf("Get guest ip with name scan err: %v", err)
			return ips
		}
		ips = append(ips, ip)
	}
	return manager.getIpsByExit(ips, isExitOnly)
}

func (manager *SGuestManager) getIpsByExit(ips []string, isExitOnly bool) []string {
	intRet := make([]string, 0)
	extRet := make([]string, 0)
	for _, ip := range ips {
		addr, _ := netutils.NewIPV4Addr(ip)
		if netutils.IsExitAddress(addr) {
			extRet = append(extRet, ip)
			continue
		}
		intRet = append(intRet, ip)
	}
	if isExitOnly {
		return extRet
	} else if len(intRet) > 0 {
		return intRet
	}
	return extRet
}

func (manager *SGuestManager) getExpiredPendingDeleteGuests() []SGuest {
	deadline := time.Now().Add(time.Duration(options.Options.PendingDeleteExpireSeconds*-1) * time.Second)

	q := manager.Query()
	q = q.IsTrue("pending_deleted").LT("pending_deleted_at", deadline).Limit(options.Options.PendingDeleteMaxCleanBatchSize)

	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("fetch guests error %s", err)
		return nil
	}

	return guests
}

func (manager *SGuestManager) CleanPendingDeleteServers(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	guests := manager.getExpiredPendingDeleteGuests()
	if guests == nil {
		return
	}
	for i := 0; i < len(guests); i += 1 {
		opts := api.ServerDeleteInput{
			OverridePendingDelete: true,
		}
		guests[i].StartDeleteGuestTask(ctx, userCred, "", opts)
	}
}

func (manager *SGuestManager) getExpiredPrepaidGuests() []SGuest {
	deadline := time.Now().Add(time.Duration(options.Options.PrepaidExpireCheckSeconds*-1) * time.Second)

	q := manager.Query()
	q = q.Equals("billing_type", billing_api.BILLING_TYPE_PREPAID).LT("expired_at", deadline).
		IsFalse("pending_deleted").Limit(options.Options.ExpiredPrepaidMaxCleanBatchSize)

	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("fetch guests error %s", err)
		return nil
	}

	return guests
}

func (manager *SGuestManager) getNeedRenewPrepaidGuests() ([]SGuest, error) {
	deadline := time.Now().Add(time.Duration(options.Options.PrepaidAutoRenewHours)*time.Hour + 20*time.Minute)

	q := manager.Query()
	q = q.Equals("billing_type", billing_api.BILLING_TYPE_PREPAID).LT("expired_at", deadline).
		IsFalse("pending_deleted").In("hypervisor", GetNotSupportAutoRenewHypervisors()).IsTrue("auto_renew")

	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}

	return guests, nil
}

func (manager *SGuestManager) getExpiredPostpaidGuests() []SGuest {
	q := ListExpiredPostpaidResources(manager.Query(), options.Options.ExpiredPrepaidMaxCleanBatchSize)
	q = q.IsFalse("pending_deleted")
	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("fetch guests error %s", err)
		return nil
	}

	return guests
}

func (self *SGuest) doExternalSync(ctx context.Context, userCred mcclient.TokenCredential) error {
	host := self.GetHost()
	if host == nil {
		return fmt.Errorf("no host???")
	}
	ihost, iprovider, err := host.GetIHostAndProvider()
	if err != nil {
		return err
	}
	iVM, err := ihost.GetIVMById(self.ExternalId)
	if err != nil {
		return err
	}
	return self.syncWithCloudVM(ctx, userCred, iprovider, host, iVM, nil)
}

func (manager *SGuestManager) DeleteExpiredPrepaidServers(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	guests := manager.getExpiredPrepaidGuests()
	if guests == nil {
		return
	}
	deleteSnapshot := options.Options.DeleteSnapshotExpiredRelease
	for i := 0; i < len(guests); i += 1 {
		// fake delete expired prepaid servers
		if len(guests[i].ExternalId) > 0 {
			err := guests[i].doExternalSync(ctx, userCred)
			if err == nil && guests[i].IsValidPrePaid() {
				continue
			}
		}
		guests[i].SetDisableDelete(userCred, false)
		opts := api.ServerDeleteInput{
			DeleteSnapshots: deleteSnapshot,
		}
		guests[i].StartDeleteGuestTask(ctx, userCred, "", opts)
	}
}

func (manager *SGuestManager) AutoRenewPrepaidServer(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	guests, err := manager.getNeedRenewPrepaidGuests()
	if err != nil {
		log.Errorf("failed to get need renew prepaid guests error: %v", err)
		return
	}
	for i := 0; i < len(guests); i += 1 {
		if len(guests[i].ExternalId) > 0 {
			err := guests[i].doExternalSync(ctx, userCred)
			if err == nil && guests[i].IsValidPrePaid() {
				continue
			}
		}
		guests[i].startGuestRenewTask(ctx, userCred, "1M", "")
	}
}

func (manager *SGuestManager) DeleteExpiredPostpaidServers(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	guests := manager.getExpiredPostpaidGuests()
	if len(guests) == 0 {
		log.Infof("No expired postpaid guest")
		return
	}
	deleteSnapshot := options.Options.DeleteSnapshotExpiredRelease
	for i := 0; i < len(guests); i++ {
		if len(guests[i].ExternalId) > 0 {
			err := guests[i].doExternalSync(ctx, userCred)
			if err == nil && guests[i].IsValidPostPaid() {
				continue
			}
		}
		guests[i].SetDisableDelete(userCred, false)
		opts := api.ServerDeleteInput{DeleteSnapshots: deleteSnapshot}
		guests[i].StartDeleteGuestTask(ctx, userCred, "", opts)
	}
}

func (self *SGuestManager) ReconcileBackupGuests(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	self.switchBackupGuests(ctx, userCred)
	self.createBackupGuests(ctx, userCred)
}

func (self *SGuestManager) switchBackupGuests(ctx context.Context, userCred mcclient.TokenCredential) {
	q := self.Query()
	metaDataQuery := db.Metadata.Query().Startswith("id", "server::").
		Equals("key", "switch_backup").IsNotEmpty("value").GroupBy("id")
	metaDataQuery.AppendField(sqlchemy.SubStr("guest_id", metaDataQuery.Field("id"), len("server::")+1, 0))
	subQ := metaDataQuery.SubQuery()
	q = q.Join(subQ, sqlchemy.Equals(q.Field("id"), subQ.Field("guest_id")))

	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("ReconcileBackupGuests failed fetch guests %s", err)
		return
	}
	log.Debugf("Guests count %d need reconcile with switch backup", len(guests))
	for i := 0; i < len(guests); i++ {
		val := guests[i].GetMetadataJson("switch_backup", userCred)
		t, err := val.GetTime()
		if err != nil {
			log.Errorf("failed get time from metadata switch_backup %s", err)
			continue
		}
		if time.Now().After(t) {
			data := jsonutils.NewDict()
			data.Set("purge_backup", jsonutils.JSONTrue)
			_, err := guests[i].PerformSwitchToBackup(ctx, userCred, nil, data)
			if err != nil {
				db.OpsLog.LogEvent(
					&guests[i], db.ACT_SWITCH_FAILED, fmt.Sprintf("switchBackupGuests on reconcile_backup: %s", err), userCred,
				)
				logclient.AddSimpleActionLog(
					&guests[i], logclient.ACT_SWITCH_TO_BACKUP,
					fmt.Sprintf("switchBackupGuests on reconcile_backup: %s", err), userCred, false,
				)
			}
		}
	}
}

func (self *SGuestManager) createBackupGuests(ctx context.Context, userCred mcclient.TokenCredential) {
	q := self.Query()
	metaDataQuery := db.Metadata.Query().Startswith("id", "server::").
		Equals("key", "create_backup").IsNotEmpty("value").GroupBy("id")
	metaDataQuery.AppendField(sqlchemy.SubStr("guest_id", metaDataQuery.Field("id"), len("server::")+1, 0))
	subQ := metaDataQuery.SubQuery()
	q = q.Join(subQ, sqlchemy.Equals(q.Field("id"), subQ.Field("guest_id")))

	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("ReconcileBackupGuests failed fetch guests %s", err)
		return
	}
	log.Infof("Guests count %d need reconcile with create bakcup", len(guests))
	for i := 0; i < len(guests); i++ {
		val := guests[i].GetMetadataJson("create_backup", userCred)
		t, err := val.GetTime()
		if err != nil {
			log.Errorf("failed get time from metadata create_backup %s", err)
			continue
		}
		if time.Now().After(t) {
			if len(guests[i].BackupHostId) > 0 {
				data := jsonutils.NewDict()
				data.Set("purge", jsonutils.JSONTrue)
				data.Set("create", jsonutils.JSONTrue)
				_, err := guests[i].PerformDeleteBackup(ctx, userCred, nil, data)
				if err != nil {
					db.OpsLog.LogEvent(
						&guests[i], db.ACT_DELETE_BACKUP_FAILED,
						fmt.Sprintf("PerformDeleteBackup on ReconcileBackupGuests: %s", err), userCred,
					)
					logclient.AddSimpleActionLog(
						&guests[i], logclient.ACT_DELETE_BACKUP,
						fmt.Sprintf("PerformDeleteBackup on ReconcileBackupGuests: %s", err), userCred, false,
					)
				}
			} else {
				params := jsonutils.NewDict()
				params.Set("reconcile_backup", jsonutils.JSONTrue)
				if _, err := guests[i].StartGuestCreateBackupTask(ctx, userCred, "", params); err != nil {
					db.OpsLog.LogEvent(
						&guests[i], db.ACT_CREATE_BACKUP_FAILED,
						fmt.Sprintf("StartGuestCreateBackupTask on ReconcileBackupGuests: %s", err), userCred,
					)
					logclient.AddSimpleActionLog(
						&guests[i], logclient.ACT_CREATE_BACKUP,
						fmt.Sprintf("StartGuestCreateBackupTask on ReconcileBackupGuests: %s", err), userCred, false,
					)
				}
			}
		}
	}
}

func (self *SGuest) isInReconcile(userCred mcclient.TokenCredential) bool {
	switchBackup := self.GetMetadata("switch_backup", userCred)
	createBackup := self.GetMetadata("create_backup", userCred)
	if len(switchBackup) > 0 || len(createBackup) > 0 {
		return true
	}
	return false
}

func (self *SGuest) IsEipAssociable() error {
	var eip *SElasticip
	var err error
	switch self.Hypervisor {
	case api.HYPERVISOR_AWS:
		eip, err = self.GetElasticIp()
	default:
		eip, err = self.GetEipOrPublicIp()
	}

	if err != nil {
		log.Errorf("Fail to get Eip %s", err)
		return errors.Wrap(err, "IsEipAssociable")
	}

	if eip != nil {
		return httperrors.NewInvalidStatusError("already associate with eip")
	}

	return nil
}

func (self *SGuest) GetEipOrPublicIp() (*SElasticip, error) {
	return ElasticipManager.getEip(api.EIP_ASSOCIATE_TYPE_SERVER, self.Id, "")
}

func (self *SGuest) GetElasticIp() (*SElasticip, error) {
	return ElasticipManager.getEip(api.EIP_ASSOCIATE_TYPE_SERVER, self.Id, api.EIP_MODE_STANDALONE_EIP)
}

func (self *SGuest) GetPublicIp() (*SElasticip, error) {
	return ElasticipManager.getEip(api.EIP_ASSOCIATE_TYPE_SERVER, self.Id, api.EIP_MODE_INSTANCE_PUBLICIP)
}

func (self *SGuest) SyncVMEip(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extEip cloudprovider.ICloudEIP, syncOwnerId mcclient.IIdentityProvider) compare.SyncResult {
	result := compare.SyncResult{}

	eip, err := self.GetPublicIp()
	if err != nil {
		result.Error(fmt.Errorf("getPublicIp error %s", err))
		return result
	} else if eip == nil {
		eip, err = self.GetElasticIp()
		if err != nil {
			result.Error(fmt.Errorf("getEip error %s", err))
			return result
		}
	}

	if eip == nil && extEip == nil {
		// do nothing
	} else if eip == nil && extEip != nil {
		// add
		neip, err := ElasticipManager.getEipByExtEip(ctx, userCred, extEip, provider, self.getRegion(), syncOwnerId)
		if err != nil {
			log.Errorf("getEipByExtEip error %v", err)
			result.AddError(err)
		} else {
			err = neip.AssociateInstance(ctx, userCred, api.EIP_ASSOCIATE_TYPE_SERVER, self)
			if err != nil {
				log.Errorf("AssociateVM error %v", err)
				result.AddError(err)
			} else {
				result.Add()
			}
		}
	} else if eip != nil && extEip == nil {
		// remove
		err = eip.Dissociate(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	} else {
		// sync
		if eip.IpAddr != extEip.GetIpAddr() {
			// remove then add
			err = eip.Dissociate(ctx, userCred)
			if err != nil {
				// fail to remove
				result.DeleteError(err)
			} else {
				result.Delete()
				neip, err := ElasticipManager.getEipByExtEip(ctx, userCred, extEip, provider, self.getRegion(), syncOwnerId)
				if err != nil {
					result.AddError(err)
				} else {
					err = neip.AssociateInstance(ctx, userCred, api.EIP_ASSOCIATE_TYPE_SERVER, self)
					if err != nil {
						result.AddError(err)
					} else {
						result.Add()
					}
				}
			}
		} else {
			// do nothing
			err := eip.SyncWithCloudEip(ctx, userCred, provider, extEip, syncOwnerId)
			if err != nil {
				result.UpdateError(err)
			} else {
				result.Update()
			}
		}
	}

	return result
}

func (self *SGuest) getSecgroupsBySecgroupExternalIds(externalIds []string) ([]SSecurityGroup, error) {
	host := self.GetHost()
	if host == nil {
		return nil, errors.Error("not found host for guest")
	}

	return getSecgroupsBySecgroupExternalIds(host.ManagerId, externalIds)
}

func getSecgroupsBySecgroupExternalIds(managerId string, externalIds []string) ([]SSecurityGroup, error) {
	sq := SecurityGroupCacheManager.Query("secgroup_id").In("external_id", externalIds).Equals("manager_id", managerId)
	q := SecurityGroupManager.Query().In("id", sq.SubQuery())
	secgroups := []SSecurityGroup{}
	err := db.FetchModelObjects(SecurityGroupManager, q, &secgroups)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return secgroups, nil
}

func (self *SGuest) SyncVMSecgroups(ctx context.Context, userCred mcclient.TokenCredential, externalIds []string) error {
	secgroups, err := self.getSecgroupsBySecgroupExternalIds(externalIds)
	if err != nil {
		return errors.Wrap(err, "getSecgroupsBySecgroupExternalIds")
	}
	secgroupIds := []string{}
	for _, secgroup := range secgroups {
		secgroupIds = append(secgroupIds, secgroup.Id)
	}

	return self.saveSecgroups(ctx, userCred, secgroupIds)
}

func (self *SGuest) GetIVM() (cloudprovider.ICloudVM, error) {
	if len(self.ExternalId) == 0 {
		msg := fmt.Sprintf("GetIVM: not managed by a provider")
		log.Errorf(msg)
		return nil, fmt.Errorf(msg)
	}
	host := self.GetHost()
	if host == nil {
		msg := fmt.Sprintf("GetIVM: No valid host")
		log.Errorf(msg)
		return nil, fmt.Errorf(msg)
	}
	ihost, err := host.GetIHost()
	if err != nil {
		msg := fmt.Sprintf("GetIVM: getihost fail %s", err)
		log.Errorf(msg)
		return nil, fmt.Errorf(msg)
	}
	return ihost.GetIVMById(self.ExternalId)
}

func (self *SGuest) PendingDetachScalingGroup() error {
	sggs, err := ScalingGroupGuestManager.Fetch("", self.GetId())
	if err != nil {
		return err
	}
	for i := range sggs {
		sggs[i].SetGuestStatus(api.SG_GUEST_STATUS_PENDING_REMOVE)
	}
	return nil
}

func (self *SGuest) DetachScheduledTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	q := ScheduledTaskLabelManager.Query().Equals("label", self.Id)
	stls := make([]SScheduledTaskLabel, 0, 1)
	err := db.FetchModelObjects(ScheduledTaskLabelManager, q, &stls)
	if err != nil {
		return err
	}
	for i := range stls {
		err := stls[i].Detach(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SGuest) DeleteEip(ctx context.Context, userCred mcclient.TokenCredential) error {
	eip, err := self.GetEipOrPublicIp()
	if err != nil {
		log.Errorf("Delete eip fail for get Eip %s", err)
		return err
	}
	if eip == nil {
		return nil
	}
	if eip.Mode == api.EIP_MODE_INSTANCE_PUBLICIP {
		err = eip.RealDelete(ctx, userCred)
		if err != nil {
			log.Errorf("Delete eip on delete server fail %s", err)
			return err
		}
	} else {
		err = eip.Dissociate(ctx, userCred)
		if err != nil {
			log.Errorf("Dissociate eip on delete server fail %s", err)
			return err
		}
	}
	return nil
}

func (self *SGuest) SetDisableDelete(userCred mcclient.TokenCredential, val bool) error {
	diff, err := db.Update(self, func() error {
		if val {
			self.DisableDelete = tristate.True
		} else {
			self.DisableDelete = tristate.False
		}
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)
	logclient.AddSimpleActionLog(self, logclient.ACT_UPDATE, diff, userCred, true)
	return err
}

func (self *SGuest) getDefaultStorageType() string {
	diskCat := self.CategorizeDisks()
	if diskCat.Root != nil {
		rootStorage := diskCat.Root.GetStorage()
		if rootStorage != nil {
			return rootStorage.StorageType
		}
	}
	return api.STORAGE_LOCAL
}

func (self *SGuest) GetApptags() []string {
	tagsStr := self.GetMetadata(api.VM_METADATA_APP_TAGS, nil)
	if len(tagsStr) > 0 {
		return strings.Split(tagsStr, ",")
	}
	return nil
}

func (self *SGuest) ToSchedDesc() *schedapi.ScheduleInput {
	desc := new(schedapi.ScheduleInput)
	config := &schedapi.ServerConfig{
		Name:          self.Name,
		Memory:        self.VmemSize,
		Ncpu:          int(self.VcpuCount),
		ServerConfigs: new(api.ServerConfigs),
	}
	desc.Id = self.Id
	self.FillGroupSchedDesc(config.ServerConfigs)
	self.FillDiskSchedDesc(config.ServerConfigs)
	self.FillNetSchedDesc(config.ServerConfigs)
	if len(self.HostId) > 0 && regutils.MatchUUID(self.HostId) {
		desc.HostId = self.HostId
	}
	config.Project = self.ProjectId
	config.Domain = self.DomainId
	/*tags := self.GetApptags()
	for i := 0; i < len(tags); i++ {
		desc.Set(tags[i], jsonutils.JSONTrue)
	}*/

	config.Hypervisor = self.GetHypervisor()
	desc.ServerConfig = *config
	desc.OsArch = self.OsArch
	return desc
}

func (self *SGuest) FillGroupSchedDesc(desc *api.ServerConfigs) {
	groups := make([]SGroupguest, 0)
	err := GroupguestManager.Query().Equals("guest_id", self.Id).All(&groups)
	if err != nil {
		log.Errorln(err)
		return
	}
	groupids := make([]string, len(groups))
	for i := range groups {
		groupids[i] = groups[i].GroupId
	}
	desc.InstanceGroupIds = groupids
}

func (self *SGuest) FillDiskSchedDesc(desc *api.ServerConfigs) {
	guestDisks := make([]SGuestdisk, 0)
	err := GuestdiskManager.Query().Equals("guest_id", self.Id).All(&guestDisks)
	if err != nil {
		log.Errorf("FillDiskSchedDesc: %v", err)
		return
	}
	for i := 0; i < len(guestDisks); i++ {
		diskConf := guestDisks[i].ToDiskConfig()
		// HACK: storage used by self, so earse it
		if diskConf.Backend == api.STORAGE_LOCAL {
			diskConf.Storage = ""
		}
		desc.Disks = append(desc.Disks, diskConf)
	}
}

func (self *SGuest) FillNetSchedDesc(desc *api.ServerConfigs) {
	guestNetworks := make([]SGuestnetwork, 0)
	err := GuestnetworkManager.Query().Equals("guest_id", self.Id).All(&guestNetworks)
	if err != nil {
		log.Errorf("FillNetSchedDesc: %v", err)
		return
	}
	if desc.Networks == nil {
		desc.Networks = make([]*api.NetworkConfig, 0)
	}
	for i := 0; i < len(guestNetworks); i++ {
		desc.Networks = append(desc.Networks, guestNetworks[i].ToNetworkConfig())
	}
}

func (self *SGuest) GuestDisksHasSnapshot() (bool, error) {
	guestDisks := self.GetDisks()
	for i := 0; i < len(guestDisks); i++ {
		cnt, err := SnapshotManager.GetDiskSnapshotCount(guestDisks[i].DiskId)
		if err != nil {
			return false, err
		}
		if cnt > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (self *SGuest) OnScheduleToHost(ctx context.Context, userCred mcclient.TokenCredential, hostId string) error {
	err := self.SetHostId(userCred, hostId)
	if err != nil {
		return err
	}

	notes := jsonutils.NewDict()
	notes.Add(jsonutils.NewString(hostId), "host_id")
	db.OpsLog.LogEvent(self, db.ACT_SCHEDULE, notes, userCred)

	return self.GetHost().ClearSchedDescCache()
}

func (guest *SGuest) AllowGetDetailsTasks(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGetSpec(userCred, guest, "tasks")
}

func (guest *SGuest) GetDetailsTasks(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	since := time.Time{}
	if query.Contains("since") {
		since, _ = query.GetTime("since")
	}
	var isOpen *bool = nil
	if query.Contains("is_open") {
		isOpenVal, _ := query.Bool("is_open")
		isOpen = &isOpenVal
	}
	q := taskman.TaskManager.QueryTasksOfObject(guest, since, isOpen)
	objs, err := db.Query2List(taskman.TaskManager, ctx, userCred, q, query, false)
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewArray(objs...), "tasks")
	return ret, nil
}

func (guest *SGuest) GetDynamicConditionInput() *jsonutils.JSONDict {
	return guest.ToSchedDesc().ToConditionInput()
}

func (self *SGuest) ToCreateInput(userCred mcclient.TokenCredential) *api.ServerCreateInput {
	genInput := self.toCreateInput()
	userInput, err := self.GetCreateParams(userCred)
	if err != nil {
		return genInput
	}
	if self.GetHypervisor() != api.HYPERVISOR_BAREMETAL {
		// fill missing create params like schedtags
		disks := []*api.DiskConfig{}
		for idx, disk := range genInput.Disks {
			tmpD := disk
			if idx < len(userInput.Disks) {
				inputDisk := userInput.Disks[idx]
				tmpD.Schedtags = inputDisk.Schedtags
				tmpD.Storage = inputDisk.Storage
			}
			disks = append(disks, tmpD)
		}
		userInput.Disks = disks
	}
	nets := []*api.NetworkConfig{}
	for idx, net := range genInput.Networks {
		tmpN := net
		if idx < len(userInput.Networks) {
			inputNet := userInput.Networks[idx]
			tmpN.Schedtags = inputNet.Schedtags
			tmpN.Network = inputNet.Network
		}
		nets = append(nets, tmpN)
	}
	userInput.Networks = nets
	userInput.IsolatedDevices = genInput.IsolatedDevices
	userInput.Count = 1
	// override some old userInput properties via genInput because of change config behavior
	userInput.VmemSize = genInput.VmemSize
	userInput.VcpuCount = genInput.VcpuCount
	userInput.Vga = genInput.Vga
	userInput.Vdi = genInput.Vdi
	userInput.Bios = genInput.Bios
	userInput.Cdrom = genInput.Cdrom
	userInput.Description = genInput.Description
	userInput.BootOrder = genInput.BootOrder
	userInput.DisableDelete = genInput.DisableDelete
	userInput.ShutdownBehavior = genInput.ShutdownBehavior
	userInput.IsSystem = genInput.IsSystem
	userInput.SecgroupId = genInput.SecgroupId
	userInput.KeypairId = genInput.KeypairId
	userInput.EipBw = genInput.EipBw
	userInput.EipChargeType = genInput.EipChargeType
	provider := self.GetDriver()
	if provider.IsSupportPublicIp() {
		userInput.PublicIpBw = genInput.PublicIpBw
		userInput.PublicIpChargeType = genInput.PublicIpChargeType
	}
	userInput.AutoRenew = genInput.AutoRenew
	// cloned server should belongs to the project creating it
	userInput.ProjectId = userCred.GetProjectId()
	userInput.ProjectDomainId = userCred.GetProjectDomainId()
	userInput.Secgroups = []string{}
	secgroups, _ := self.GetSecgroups()
	for _, secgroup := range secgroups {
		userInput.Secgroups = append(userInput.Secgroups, secgroup.Id)
	}
	if genInput.ResourceType != "" {
		userInput.ResourceType = genInput.ResourceType
	}
	if genInput.InstanceType != "" {
		userInput.InstanceType = genInput.InstanceType
	}
	if genInput.PreferRegion != "" {
		userInput.PreferRegion = genInput.PreferRegion
	}
	if genInput.PreferZone != "" {
		userInput.PreferZone = genInput.PreferZone
	}
	// clean some of user input
	userInput.GenerateName = ""
	userInput.Description = ""
	return userInput
}

func (self *SGuest) toCreateInput() *api.ServerCreateInput {
	r := new(api.ServerCreateInput)
	r.VmemSize = self.VmemSize
	r.VcpuCount = int(self.VcpuCount)
	if guestCdrom := self.getCdrom(false); guestCdrom != nil {
		r.Cdrom = guestCdrom.ImageId
	}
	r.Vga = self.Vga
	r.Vdi = self.Vdi
	r.Bios = self.Bios
	r.Description = self.Description
	r.BootOrder = self.BootOrder
	r.DisableDelete = new(bool)
	*r.DisableDelete = self.DisableDelete.Bool()
	r.ShutdownBehavior = self.ShutdownBehavior
	// ignore r.DeployConfigs
	r.IsSystem = &self.IsSystem
	r.SecgroupId = self.SecgrpId

	r.ServerConfigs = new(api.ServerConfigs)
	r.Hypervisor = self.Hypervisor
	r.InstanceType = self.InstanceType
	r.ProjectId = self.ProjectId
	r.ProjectDomainId = self.DomainId
	r.Count = 1
	r.Disks = self.ToDisksConfig()
	r.Networks = self.ToNetworksConfig()
	r.IsolatedDevices = self.ToIsolatedDevicesConfig()
	r.AutoRenew = self.AutoRenew

	if keypair := self.getKeypair(); keypair != nil {
		r.KeypairId = keypair.Id
	}
	if host := self.GetHost(); host != nil {
		r.ResourceType = host.ResourceType
	}
	if eip, _ := self.GetEipOrPublicIp(); eip != nil {
		switch eip.Mode {
		case api.EIP_MODE_STANDALONE_EIP:
			r.EipBw = eip.Bandwidth
			r.EipChargeType = eip.ChargeType
		case api.EIP_MODE_INSTANCE_PUBLICIP:
			if driver := self.GetDriver(); driver.IsSupportPublicIp() {
				r.PublicIpBw = eip.Bandwidth
				r.PublicIpChargeType = eip.ChargeType
			}
		}
	}
	if zone := self.getZone(); zone != nil {
		r.PreferRegion = zone.GetRegion().GetId()
		r.PreferZone = zone.GetId()
	}
	return r
}

func (self *SGuest) ToDisksConfig() []*api.DiskConfig {
	guestDisks := self.GetDisks()
	if len(guestDisks) == 0 {
		return nil
	}
	ret := make([]*api.DiskConfig, len(guestDisks))
	for idx, guestDisk := range guestDisks {
		diskConf := new(api.DiskConfig)
		disk := guestDisk.GetDisk()
		diskConf.Index = int(guestDisk.Index)
		diskConf.ImageId = disk.GetTemplateId()
		diskConf.SnapshotId = disk.SnapshotId
		diskConf.DiskType = disk.DiskType
		diskConf.SizeMb = disk.DiskSize
		diskConf.Fs = disk.FsFormat
		diskConf.Format = disk.DiskFormat
		diskConf.Driver = guestDisk.Driver
		diskConf.Cache = guestDisk.CacheMode
		diskConf.Mountpoint = guestDisk.Mountpoint
		storage := disk.GetStorage()
		diskConf.Backend = storage.StorageType
		diskConf.Medium = storage.MediumType
		ret[idx] = diskConf
	}
	return ret
}

func (self *SGuest) ToNetworksConfig() []*api.NetworkConfig {
	guestNetworks, _ := self.GetNetworks("")
	if len(guestNetworks) == 0 {
		return nil
	}
	ret := make([]*api.NetworkConfig, 0)
	teamMacs := []string{}
	for _, gn := range guestNetworks {
		if tg, _ := gn.GetTeamGuestnetwork(); tg != nil {
			teamMacs = append(teamMacs, gn.TeamWith)
		}
	}
	for _, guestNetwork := range guestNetworks {
		netConf := new(api.NetworkConfig)
		network := guestNetwork.GetNetwork()
		requireTeaming := false
		if tg, _ := guestNetwork.GetTeamGuestnetwork(); tg != nil {
			requireTeaming = true
		}
		if utils.IsInStringArray(guestNetwork.MacAddr, teamMacs) {
			continue
		}

		// XXX: same wire
		netConf.Wire = network.WireId
		netConf.Network = network.Id
		netConf.Exit = guestNetwork.IsExit()
		// netConf.Private
		// netConf.Reserved
		netConf.Driver = guestNetwork.Driver
		netConf.BwLimit = guestNetwork.BwLimit
		netConf.RequireTeaming = requireTeaming
		// netConf.NetType
		ret = append(ret, netConf)
	}
	return ret
}

func (self *SGuest) ToIsolatedDevicesConfig() []*api.IsolatedDeviceConfig {
	guestIsolatedDevices := self.GetIsolatedDevices()
	if len(guestIsolatedDevices) == 0 {
		return nil
	}
	ret := make([]*api.IsolatedDeviceConfig, len(guestIsolatedDevices))
	for idx, guestIsolatedDevice := range guestIsolatedDevices {
		devConf := new(api.IsolatedDeviceConfig)
		devConf.Model = guestIsolatedDevice.Model
		devConf.Vendor = guestIsolatedDevice.getVendor()
		devConf.DevType = guestIsolatedDevice.DevType
		ret[idx] = devConf
	}
	return ret
}

func (self *SGuest) IsImport(userCred mcclient.TokenCredential) bool {
	return self.GetMetadata("__is_import", userCred) == "true"
}

func (guest *SGuest) AllowGetDetailsRemoteNics(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGetSpec(userCred, guest, "remote-nics")
}

func (guest *SGuest) GetDetailsRemoteNics(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	iVM, err := guest.GetIVM()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	iNics, err := iVM.GetINics()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	type SVNic struct {
		Index  int
		Ip     string
		Mac    string
		Driver string
	}
	nics := make([]SVNic, len(iNics))
	for i := range iNics {
		nics[i] = SVNic{
			Index:  i,
			Ip:     iNics[i].GetIP(),
			Mac:    iNics[i].GetMAC(),
			Driver: iNics[i].GetDriver(),
		}
	}
	// ret := jsonutils.NewDict()
	// ret.Set("vnics", jsonutils.Marshal(nics))
	return jsonutils.Marshal(nics), nil
}

func (self *SGuest) GetInstanceSnapshots() ([]SInstanceSnapshot, error) {
	instanceSnapshots := make([]SInstanceSnapshot, 0)
	q := InstanceSnapshotManager.Query().Equals("guest_id", self.Id)
	err := db.FetchModelObjects(InstanceSnapshotManager, q, &instanceSnapshots)
	if err != nil {
		return nil, err
	}
	return instanceSnapshots, nil
}

func (self *SGuest) GetInstanceSnapshotCount() (int, error) {
	q := InstanceSnapshotManager.Query().Equals("guest_id", self.Id)
	return q.CountWithError()
}

func (self *SGuest) GetDiskSnapshotsNotInInstanceSnapshots() ([]SSnapshot, error) {
	guestDisks := self.GetDisks()
	diskIds := make([]string, len(guestDisks))
	for i := 0; i < len(guestDisks); i++ {
		diskIds[i] = guestDisks[i].DiskId
	}
	snapshots := make([]SSnapshot, 0)
	q := SnapshotManager.Query().IsFalse("fake_deleted").In("disk_id", diskIds)
	sq := InstanceSnapshotJointManager.Query("snapshot_id").SubQuery()
	q = q.LeftJoin(sq, sqlchemy.Equals(q.Field("id"), sq.Field("snapshot_id"))).
		Filter(sqlchemy.IsNull(sq.Field("snapshot_id")))
	err := db.FetchModelObjects(SnapshotManager, q, &snapshots)
	if err != nil {
		log.Errorf("fetch db snapshots failed %s", err)
		return nil, err
	}
	return snapshots, nil
}

func (self *SGuest) getGuestUsage(guestCount int) (SQuota, SRegionQuota, error) {
	usage := SQuota{}
	regionUsage := SRegionQuota{}
	usage.Count = guestCount
	usage.Cpu = int(self.VcpuCount) * guestCount
	usage.Memory = int(self.VmemSize * guestCount)
	diskSize := self.getDiskSize()
	if diskSize < 0 {
		return usage, regionUsage, httperrors.NewInternalServerError("fetch disk size failed")
	}
	usage.Storage = self.getDiskSize() * guestCount
	netCount, err := self.NetworkCount()
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return usage, regionUsage, err
	}
	regionUsage.Port = netCount
	// regionUsage.Bw = self.getBandwidth(false)
	eip, err := self.GetEipOrPublicIp()
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return usage, regionUsage, err
	}
	if eip != nil {
		regionUsage.Eip = 1
	}
	return usage, regionUsage, nil
}

func (self *SGuestManager) checkGuestImage(ctx context.Context, input *api.ServerCreateInput) error {
	// There is no need to check the availability of guest imag if input.Disks is empty
	if len(input.Disks) == 0 {
		return nil
	}
	// That data disks has image id show that these image is part of guest image
	for _, config := range input.Disks[1:] {
		if len(config.ImageId) != 0 && len(input.GuestImageID) == 0 {
			return httperrors.NewMissingParameterError("guest_image_id")
		}
	}

	if len(input.GuestImageID) == 0 {
		return nil
	}

	guestImageId := input.GuestImageID
	params := jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "details")

	s := auth.GetAdminSession(ctx, options.Options.Region, "")
	ret, err := modules.GuestImages.Get(s, guestImageId, params)
	if err != nil {
		return errors.Wrap(err, "get guest image from glance error")
	}

	// input.GuestImageID maybe name of guestimage
	if ret.Contains("id") {
		id, _ := ret.GetString("id")
		input.GuestImageID = id
	}

	images := &api.SImagesInGuest{}
	err = ret.Unmarshal(images)
	if err != nil {
		return errors.Wrap(err, "get guest image from glance error")
	}
	imageIdMap := make(map[string]struct{})
	for _, pair := range images.DataImages {
		imageIdMap[pair.ID] = struct{}{}
	}
	imageIdMap[images.RootImage.ID] = struct{}{}

	// check
	for _, diskConfig := range input.Disks {
		if len(diskConfig.ImageId) != 0 {
			if _, ok := imageIdMap[diskConfig.ImageId]; !ok {
				return httperrors.NewBadRequestError("image %s do not belong to guest image %s", diskConfig.ImageId, guestImageId)
			}
			delete(imageIdMap, diskConfig.ImageId)
		}
	}
	if len(imageIdMap) != 0 {
		return httperrors.NewBadRequestError("miss some subimage of guest image")
	}
	return nil
}

func (self *SGuest) GetDiskIndex(diskId string) int8 {
	for _, gd := range self.GetDisks() {
		if gd.DiskId == diskId {
			return gd.Index
		}
	}
	return -1
}

func (guest *SGuest) GetRegionalQuotaKeys() (quotas.IQuotaKeys, error) {
	host := guest.GetHost()
	if host == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid host")
	}
	provider := host.GetCloudprovider()
	if provider == nil && len(host.ManagerId) > 0 {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid manager")
	}
	region := host.GetRegion()
	if region == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid region")
	}
	return fetchRegionalQuotaKeys(rbacutils.ScopeProject, guest.GetOwnerId(), region, provider), nil
}

func (guest *SGuest) GetQuotaKeys() (quotas.IQuotaKeys, error) {
	host := guest.GetHost()
	if host == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid host")
	}
	provider := host.GetCloudprovider()
	if provider == nil && len(host.ManagerId) > 0 {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid manager")
	}
	zone := host.GetZone()
	if zone == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid zone")
	}
	hypervisor := guest.Hypervisor
	if !utils.IsInStringArray(hypervisor, api.ONECLOUD_HYPERVISORS) {
		hypervisor = ""
	}
	return fetchComputeQuotaKeys(
		rbacutils.ScopeProject,
		guest.GetOwnerId(),
		zone,
		provider,
		hypervisor,
	), nil
}

func (guest *SGuest) GetUsages() []db.IUsage {
	if guest.PendingDeleted || guest.Deleted {
		return nil
	}
	usage, regionUsage, err := guest.getGuestUsage(1)
	if err != nil {
		log.Errorf("guest.getGuestUsage fail %s", err)
		return nil
	}
	keys, err := guest.GetQuotaKeys()
	if err != nil {
		log.Errorf("guest.GetQuotaKeys fail %s", err)
		return nil
	}
	usage.SetKeys(keys)
	regionUsage.SetKeys(keys.(SComputeResourceKeys).SRegionalCloudResourceKeys)
	return []db.IUsage{
		&usage,
		&regionUsage,
	}
}

var (
	// `^[a-zA-Z][a-zA-Z0-9._@-]*$`)
	serverNameREG = regexp.MustCompile(`^[a-zA-Z$][a-zA-Z0-9-${}.]*$`)
	hostnameREG   = regexp.MustCompile(`^[a-z$][a-z0-9-${}.]*$`)
)

func (manager *SGuestManager) ValidateName(name string) error {
	if serverNameREG.MatchString(name) {
		return nil
	}
	return httperrors.NewInputParameterError("name starts with letter, and contains letter, number and - only")
}

func (manager *SGuestManager) ValidateNameLoginAccount(name string) error {
	if hostnameREG.MatchString(name) {
		return nil
	}
	return httperrors.NewInputParameterError("name starts with letter, and contains letter, number and - only")
}

func (guest *SGuest) StartRemoteUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, replaceTags bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	if replaceTags {
		data.Add(jsonutils.JSONTrue, "replace_tags")
	}
	if task, err := taskman.TaskManager.NewTask(ctx, "GuestRemoteUpdateTask", guest, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return errors.Wrap(err, "Start GuestRemoteUpdateTask")
	} else {
		guest.SetStatus(userCred, api.VM_UPDATE_TAGS, "StartRemoteUpdateTask")
		task.ScheduleRun(nil)
	}
	return nil
}

func (guest *SGuest) OnMetadataUpdated(ctx context.Context, userCred mcclient.TokenCredential) {
	if len(guest.ExternalId) == 0 {
		return
	}
	err := guest.StartRemoteUpdateTask(ctx, userCred, true, "")
	if err != nil {
		log.Errorf("StartRemoteUpdateTask fail: %s", err)
	}
}
