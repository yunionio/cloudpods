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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/util/pinyinutils"
	"yunion.io/x/pkg/util/rbacscope"
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
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/compute/sshkeys"
	devtool_utils "yunion.io/x/onecloud/pkg/devtool/utils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/util/bitmap"
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
	db.SRecordChecksumResourceBaseManager
	SHostnameResourceBaseManager

	db.SEncryptedResourceManager
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
		SRecordChecksumResourceBaseManager: *db.NewRecordChecksumResourceBaseManager(),
	}
	GuestManager.SetVirtualObject(GuestManager)
	GuestManager.SetAlias("guest", "guests")
	GuestManager.NameRequireAscii = false
	notifyclient.AddNotifyDBHookResources(GuestManager.KeywordPlural(), GuestManager.AliasPlural())
}

type SGuest struct {
	db.SVirtualResourceBase

	db.SExternalizedResourceBase

	SBillingResourceBase
	SDeletePreventableResourceBase
	db.SMultiArchResourceBase
	db.SRecordChecksumResourceBase

	SHostnameResourceBase
	SHostResourceBase `width:"36" charset:"ascii" nullable:"true" list:"user" get:"user" index:"true"`

	db.SEncryptedResource

	// CPU插槽(socket)的数量
	CpuSockets int `nullable:"false" default:"1" list:"user" create:"optional"`
	// CPU核(core)的数量， VcpuCount = CpuSockets * (cores per socket)，例如 2颗CPU，每颗CPU8核，则 VcpuCount=2*8=16
	VcpuCount int `nullable:"false" default:"1" list:"user" create:"optional"`
	// 内存大小, 单位MB
	VmemSize int `nullable:"false" list:"user" create:"required"`

	// 启动顺序
	BootOrder string `width:"8" charset:"ascii" nullable:"true" default:"cdn" list:"user" update:"user" create:"optional"`

	// 关机操作类型
	// example: stop
	ShutdownBehavior string `width:"16" charset:"ascii" default:"stop" list:"user" update:"user" create:"optional"`

	// 关机收费模式
	// example: keep_charging, stop_charging
	ShutdownMode string `width:"16" charset:"ascii" default:"keep_charging" list:"user"`

	// 秘钥对Id
	KeypairId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	// 备份机所在宿主机Id
	BackupHostId      string `width:"36" charset:"ascii" nullable:"true" list:"user" get:"user"`
	BackupGuestStatus string `width:"36" charset:"ascii" nullable:"false" default:"init" list:"user" create:"optional" json:"backup_guest_status"`

	// 迁移或克隆的速度
	ProgressMbps float64 `nullable:"false" default:"0" list:"user" create:"optional" update:"user" log:"skip"`

	Vga     string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`
	Vdi     string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`
	Machine string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`
	Bios    string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`
	// 操作系统类型
	OsType string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	FlavorId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	// 安全组Id
	// example: default
	SecgrpId string `width:"36" charset:"ascii" nullable:"true" list:"user" get:"user" create:"optional"`
	// 管理员可见安全组Id
	AdminSecgrpId string `width:"36" charset:"ascii" nullable:"true" list:"domain" get:"domain"`

	SrcIpCheck  tristate.TriState `default:"true" create:"optional" list:"user" update:"user"`
	SrcMacCheck tristate.TriState `default:"true" create:"optional" list:"user" update:"user"`

	// 虚拟化技术
	// example: kvm
	Hypervisor string `width:"16" charset:"ascii" nullable:"false" default:"kvm" list:"user" create:"required"`

	// 套餐名称
	InstanceType string `width:"64" charset:"utf8" nullable:"true" list:"user" create:"optional"`

	SshableLastState tristate.TriState `default:"false" list:"user"`

	IsDaemon tristate.TriState `default:"false" list:"admin" create:"admin_optional" update:"admin"`

	// 最大内网带宽
	InternetMaxBandwidthOut int `nullable:"true" list:"user" create:"optional"`
	// 磁盘吞吐量
	Throughput int `nullable:"true" list:"user" create:"optional"`

	QgaStatus string `width:"36" charset:"ascii" nullable:"false" default:"unknown" list:"user" create:"optional"`
	// power_states limit in [on, off, unknown]
	PowerStates string `width:"36" charset:"ascii" nullable:"false" default:"unknown" list:"user" create:"optional"`
	// Used for guest rescue
	RescueMode bool `nullable:"false" default:"false" list:"user" create:"optional"`
}

func (manager *SGuestManager) GetPropertyStatistics(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*apis.StatusStatistic, error) {
	ret, err := manager.SVirtualResourceBaseManager.GetPropertyStatistics(ctx, userCred, query)
	if err != nil {
		return nil, err
	}

	q := manager.Query()
	q, err = db.ListItemQueryFilters(manager, ctx, q, userCred, query, policy.PolicyActionList)
	if err != nil {
		return nil, err
	}

	sq := q.SubQuery()
	statQ := sq.Query(sqlchemy.SUM("total_cpu_count", sq.Field("vcpu_count")), sqlchemy.SUM("total_mem_size_mb", sq.Field("vmem_size")))
	err = statQ.First(ret)
	if err != nil {
		return ret, err
	}
	diskQ := DiskManager.Query()
	gdsSQ := GuestdiskManager.Query().SubQuery()
	diskQ = diskQ.Join(gdsSQ, sqlchemy.Equals(diskQ.Field("id"), gdsSQ.Field("disk_id"))).
		Join(sq, sqlchemy.Equals(gdsSQ.Field("guest_id"), sq.Field("id")))
	diskSQ := diskQ.SubQuery()
	return ret, diskSQ.Query(sqlchemy.SUM("total_disk_size_mb", diskSQ.Field("disk_size"))).First(ret)
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
		// admin := (query.VirtualResourceListInput.Admin != nil && *query.VirtualResourceListInput.Admin)
		allowScope, _ := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList)
		if allowScope == rbacscope.ScopeSystem || allowScope == rbacscope.ScopeDomain {
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

	var eipMode string
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

		if eip.GetProviderName() == api.CLOUD_PROVIDER_AWS {
			eipMode = api.EIP_MODE_STANDALONE_EIP
		}

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

	if len(query.IpAddrs) > 0 {
		grpnets := GroupnetworkManager.Query().SubQuery()
		vipq := GroupguestManager.Query("guest_id")
		conditions := []sqlchemy.ICondition{}
		for _, ipAddr := range query.IpAddrs {
			conditions = append(conditions, sqlchemy.Regexp(grpnets.Field("ip_addr"), ipAddr))
		}
		vipq = vipq.Join(grpnets, sqlchemy.Equals(grpnets.Field("group_id"), vipq.Field("group_id"))).Filter(
			sqlchemy.OR(conditions...),
		)

		grpeips := ElasticipManager.Query().Equals("associate_type", api.EIP_ASSOCIATE_TYPE_INSTANCE_GROUP).SubQuery()
		conditions = []sqlchemy.ICondition{}
		for _, ipAddr := range query.IpAddrs {
			conditions = append(conditions, sqlchemy.Regexp(grpeips.Field("ip_addr"), ipAddr))
		}
		vipeipq := GroupguestManager.Query("guest_id")
		vipeipq = vipeipq.Join(grpeips, sqlchemy.Equals(grpeips.Field("associate_id"), vipeipq.Field("group_id"))).Filter(
			sqlchemy.OR(conditions...),
		)

		gnQ := GuestnetworkManager.Query("guest_id")
		conditions = []sqlchemy.ICondition{}
		for _, ipAddr := range query.IpAddrs {
			conditions = append(conditions, sqlchemy.Regexp(gnQ.Field("ip_addr"), ipAddr))
		}
		gn := gnQ.Filter(sqlchemy.OR(conditions...))

		guestEipQ := ElasticipManager.Query("associate_id").Equals("associate_type", api.EIP_ASSOCIATE_TYPE_SERVER)
		conditions = []sqlchemy.ICondition{}
		for _, ipAddr := range query.IpAddrs {
			conditions = append(conditions, sqlchemy.Regexp(guestEipQ.Field("ip_addr"), ipAddr))
		}
		guestEip := guestEipQ.Filter(sqlchemy.OR(conditions...))

		metadataQ := db.Metadata.Query("obj_id")
		conditions = []sqlchemy.ICondition{}
		for _, ipAddr := range query.IpAddrs {
			conditions = append(conditions, sqlchemy.AND(
				sqlchemy.Regexp(metadataQ.Field("value"), ipAddr),
				sqlchemy.Equals(metadataQ.Field("key"), "sync_ips"),
				sqlchemy.Equals(metadataQ.Field("obj_type"), "server"),
			))
		}
		metadataQ = metadataQ.Filter(sqlchemy.OR(conditions...))

		q = q.Filter(sqlchemy.OR(
			sqlchemy.In(q.Field("id"), gn.SubQuery()),
			sqlchemy.In(q.Field("id"), guestEip.SubQuery()),
			sqlchemy.In(q.Field("id"), vipq.SubQuery()),
			sqlchemy.In(q.Field("id"), vipeipq.SubQuery()),
			sqlchemy.In(q.Field("id"), metadataQ.SubQuery()),
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
		if len(eipMode) > 0 {
			sq = sq.Equals("mode", eipMode)
		}

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

	devTypeQ := func(q *sqlchemy.SQuery, checkType, backup *bool, dType string, conditions []sqlchemy.ICondition) []sqlchemy.ICondition {
		if checkType != nil {
			isodev := IsolatedDeviceManager.Query().SubQuery()
			isodevCons := []sqlchemy.ICondition{sqlchemy.IsNotNull(isodev.Field("guest_id"))}
			if len(dType) > 0 {
				isodevCons = append(isodevCons, sqlchemy.Startswith(isodev.Field("dev_type"), dType))
			}
			sgq := isodev.Query(isodev.Field("guest_id")).Filter(sqlchemy.AND(isodevCons...))
			cond := sqlchemy.NotIn
			if *checkType {
				cond = sqlchemy.In
			}
			if dType == "GPU" {
				sq := ServerSkuManager.Query("name").GT("gpu_count", 0).Distinct().SubQuery()
				if backup != nil {
					afterAnd := sqlchemy.AND
					if *backup {
						backupCond := sqlchemy.IsNotEmpty(q.Field("backup_host_id"))
						conditions = append(conditions, afterAnd(cond(q.Field("instance_type"), sq), backupCond))
					} else {
						backupCond := sqlchemy.IsEmpty(q.Field("backup_host_id"))
						conditions = append(conditions, afterAnd(cond(q.Field("instance_type"), sq), backupCond))
					}
				} else {
					conditions = append(conditions, cond(q.Field("instance_type"), sq))
				}
			} else {
				if backup != nil {
					afterAnd := sqlchemy.AND
					if *backup {
						backupCond := sqlchemy.IsNotEmpty(q.Field("backup_host_id"))
						conditions = append(conditions, afterAnd(cond(q.Field("id"), sgq), backupCond))
					} else {
						backupCond := sqlchemy.IsEmpty(q.Field("backup_host_id"))
						conditions = append(conditions, afterAnd(cond(q.Field("id"), sgq), backupCond))
					}
				} else {
					conditions = append(conditions, cond(q.Field("id"), sgq))
				}
			}
			return conditions
		}
		return conditions
	}

	conditions := []sqlchemy.ICondition{}
	if len(query.ServerType) > 0 {
		var trueVal, falseVal = true, false
		for _, serverType := range query.ServerType {
			switch serverType {
			case "normal":
				query.Normal = &falseVal
				query.Backup = &falseVal
			case "gpu":
				query.Gpu = &trueVal
				query.Backup = &falseVal
				sq := ServerSkuManager.Query("name").IsNotEmpty("gpu_spec").Distinct()
				conditions = append(conditions, sqlchemy.In(q.Field("instance_type"), sq.SubQuery()))
			case "backup":
				query.Gpu = &falseVal
				query.Backup = &trueVal
			case "usb":
				query.Usb = &trueVal
				query.Backup = &falseVal
			default:
				query.CustomDevType = serverType
				query.Backup = &falseVal
			}

			conditions = devTypeQ(q, query.Normal, query.Backup, "", conditions)
			conditions = devTypeQ(q, query.Gpu, query.Backup, "GPU", conditions)
			conditions = devTypeQ(q, query.Usb, query.Backup, api.USB_TYPE, conditions)
			if len(query.CustomDevType) > 0 {
				ct := true
				conditions = devTypeQ(q, &ct, query.Backup, query.CustomDevType, conditions)
			}
			query.Normal = nil
			query.Gpu = nil
			query.Backup = nil
		}
	}

	if len(conditions) > 0 {
		q = q.Filter(sqlchemy.OR(conditions...))
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
	if len(query.OsDist) > 0 {
		metaSQ := db.Metadata.Query().Equals("key", "os_distribution").In("value", query.OsDist).SubQuery()
		q = q.Join(metaSQ, sqlchemy.Equals(q.Field("id"), metaSQ.Field("obj_id")))
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

	if db.NeedOrderQuery([]string{query.OrderByOsDist}) {
		meta := db.Metadata.Query().Equals("key", "os_distribution").SubQuery()
		q = q.LeftJoin(meta, sqlchemy.Equals(q.Field("id"), meta.Field("obj_id")))
		db.OrderByFields(q, []string{query.OrderByOsDist}, []sqlchemy.IQueryField{meta.Field("value")})
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
	if db.NeedOrderQuery([]string{query.OrderByIp}) {
		guestnet := GuestnetworkManager.Query("guest_id", "ip_addr").SubQuery()
		q.AppendField(q.QueryFields()...)
		q.AppendField(guestnet.Field("ip_addr"))
		q = q.LeftJoin(guestnet, sqlchemy.Equals(q.Field("id"), guestnet.Field("guest_id")))
		db.OrderByFields(q, []string{query.OrderByIp}, []sqlchemy.IQueryField{sqlchemy.INET_ATON(q.Field("ip_addr"))})
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
	if field == "os_dist" {
		metaQuery := db.Metadata.Query("obj_id", "value").Equals("key", "os_distribution").SubQuery()
		q = q.AppendField(metaQuery.Field("value", field)).Distinct()
		q = q.Join(metaQuery, sqlchemy.Equals(q.Field("id"), metaQuery.Field("obj_id")))
		q.GroupBy(metaQuery.Field("value"))
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

func (manager *SGuestManager) initHostname() error {
	guests := []SGuest{}
	q := manager.Query().IsNullOrEmpty("hostname")
	err := db.FetchModelObjects(manager, q, &guests)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range guests {
		db.Update(&guests[i], func() error {
			hostname, _ := manager.SHostnameResourceBaseManager.ValidateHostname(
				guests[i].Hostname,
				guests[i].OsType,
				api.HostnameInput{
					Hostname: guests[i].Name,
				},
			)
			guests[i].Hostname = hostname.Hostname
			return nil
		})
	}
	return nil
}

func (manager *SGuestManager) clearSecgroups() error {
	guests := make([]SGuest, 0, 10)
	q := manager.Query()
	q = q.In("hypervisor", []string{api.HYPERVISOR_ESXI, api.HYPERVISOR_NUTANIX}).Filter(
		sqlchemy.OR(
			sqlchemy.IsNotEmpty(q.Field("secgrp_id")),
			sqlchemy.IsNotEmpty(q.Field("admin_secgrp_id")),
		),
	)
	err := db.FetchModelObjects(manager, q, &guests)
	if err != nil {
		return errors.Wrap(err, "db.FetchModelObjects")
	}
	// remove secgroup for esxi nutanix guest
	for i := range guests {
		db.Update(&guests[i], func() error {
			guests[i].SecgrpId = ""
			guests[i].AdminSecgrpId = ""
			return nil
		})
	}
	return nil
}

func (manager *SGuestManager) initAdminSecgroupId() error {
	if len(options.Options.DefaultAdminSecurityGroupId) == 0 {
		return nil
	}
	adminSec, _ := SecurityGroupManager.FetchSecgroupById(options.Options.DefaultAdminSecurityGroupId)
	if adminSec == nil {
		return nil
	}
	adminSecId := adminSec.Id
	guests := make([]SGuest, 0, 10)
	q := manager.Query()
	q = q.In("hypervisor", []string{api.HYPERVISOR_KVM}).IsNullOrEmpty("admin_secgrp_id")
	err := db.FetchModelObjects(manager, q, &guests)
	if err != nil {
		return errors.Wrap(err, "db.FetchModelObjects")
	}
	// remove secgroup for esxi nutanix guest
	for i := range guests {
		db.Update(&guests[i], func() error {
			guests[i].AdminSecgrpId = adminSecId
			return nil
		})
	}
	return nil
}

func (manager *SGuestManager) InitializeData() error {
	if err := manager.initHostname(); err != nil {
		return errors.Wrap(err, "initHostname")
	}
	if err := manager.clearSecgroups(); err != nil {
		return errors.Wrap(err, "cleanSecgroups")
	}
	if err := manager.initAdminSecgroupId(); err != nil {
		return errors.Wrap(err, "initAdminSecgroupId")
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
	return guest.SVirtualResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (guest *SGuest) ValidatePurgeCondition(ctx context.Context) error {
	return guest.validateDeleteCondition(ctx, true)
}

func (guest *SGuest) ValidateDeleteCondition(ctx context.Context, info *api.ServerDetails) error {
	if gotypes.IsNil(info) {
		info = &api.ServerDetails{}
		host, err := guest.GetHost()
		if err != nil {
			return err
		}
		info.HostType = host.HostType
		info.HostEnabled = host.Enabled.Bool()
		info.HostStatus = host.HostStatus
	}
	if len(info.HostType) > 0 && guest.GetHypervisor() != api.HYPERVISOR_BAREMETAL {
		if !info.HostEnabled {
			return httperrors.NewInputParameterError("Cannot delete server on disabled host")
		}
		if info.HostStatus != api.HOST_ONLINE {
			return httperrors.NewInputParameterError("Cannot delete server on offline host")
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

func (self *SGuest) GetDisks() ([]SDisk, error) {
	gds := GuestdiskManager.Query().SubQuery()
	sq := DiskManager.Query()
	q := sq.Join(gds, sqlchemy.Equals(gds.Field("disk_id"), sq.Field("id"))).Filter(
		sqlchemy.Equals(gds.Field("guest_id"), self.Id),
	).Asc(gds.Field("index"))
	disks := []SDisk{}
	err := db.FetchModelObjects(DiskManager, q, &disks)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return disks, nil
}

func (guest *SGuest) GetGuestDisks() ([]SGuestdisk, error) {
	disks := make([]SGuestdisk, 0)
	q := guest.GetDisksQuery().Asc("index")
	err := db.FetchModelObjects(GuestdiskManager, q, &disks)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return disks, nil
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
	vpc, _ := network.GetVpc()
	if vpc == nil {
		return nil, errors.Wrapf(err, "failed getting vpc of guest network %s(%s)", network.Name, network.Id)
	}
	return vpc, nil
}

func (guest *SGuest) IsOneCloudVpcNetwork() (bool, error) {
	gns, err := guest.GetNetworks("")
	if err != nil {
		return false, errors.Wrap(err, "GetNetworks")
	}
	for _, gn := range gns {
		n := gn.GetNetwork()
		if n != nil && n.isOneCloudVpcNetwork() {
			return true, nil
		}
	}
	return false, nil
}

func (guest *SGuest) GetNetworks(netId string) ([]SGuestnetwork, error) {
	guestnics := make([]SGuestnetwork, 0)
	q := guest.GetNetworksQuery(netId).Asc("index")
	err := db.FetchModelObjects(GuestnetworkManager, q, &guestnics)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return guestnics, nil
}

func (guest *SGuest) ConvertEsxiNetworks(targetGuest *SGuest) error {
	gns, err := guest.GetNetworks("")
	if err != nil {
		return err
	}
	var i int
	for ; i < len(gns); i++ {
		_, err = db.Update(&gns[i], func() error {
			gns[i].GuestId = targetGuest.Id
			if gns[i].Driver != "e1000" && gns[i].Driver != "vmxnet3" {
				gns[i].Driver = "e1000"
			}
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

func (guest *SGuest) getGuestnetworkByIpOrMac(ipAddr string, ip6Addr string, macAddr string) (*SGuestnetwork, error) {
	q := guest.GetNetworksQuery("")
	if len(ipAddr) > 0 {
		q = q.Equals("ip_addr", ipAddr)
	}
	if len(ip6Addr) > 0 {
		addr, err := netutils.NewIPV6Addr(ip6Addr)
		if err == nil {
			q = q.Equals("ip6_addr", addr.String())
		}
	}
	if len(macAddr) > 0 {
		macAddr = netutils2.FormatMac(macAddr)
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
	return guest.getGuestnetworkByIpOrMac(ipAddr, "", "")
}

func (guest *SGuest) GetGuestnetworkByIp6(ip6Addr string) (*SGuestnetwork, error) {
	return guest.getGuestnetworkByIpOrMac("", ip6Addr, "")
}

func (guest *SGuest) GetGuestnetworkByMac(macAddr string) (*SGuestnetwork, error) {
	return guest.getGuestnetworkByIpOrMac("", "", macAddr)
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
	if len(guest.SecgrpId) > 0 && len(options.Options.DefaultAdminSecurityGroupId) > 0 {
		adminSec, _ := SecurityGroupManager.FetchSecgroupById(options.Options.DefaultAdminSecurityGroupId)
		if adminSec != nil {
			guest.AdminSecgrpId = adminSec.Id
		}
	}
	guest.HostId = ""
	err := guest.SEncryptedResource.CustomizeCreate(ctx, userCred, ownerId, data, "server-"+pinyinutils.Text2Pinyin(guest.Name))
	if err != nil {
		return errors.Wrap(err, "EncryptResourceBase.CustomizeCreate")
	}
	return guest.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (guest *SGuest) GetCloudproviderId() string {
	host, _ := guest.GetHost()
	if host != nil {
		return host.GetCloudproviderId()
	}
	return ""
}

func (guest *SGuest) GetHost() (*SHost, error) {
	if len(guest.HostId) > 0 && regutils.MatchUUID(guest.HostId) {
		host, err := HostManager.FetchById(guest.HostId)
		if err != nil {
			return nil, err
		}
		return host.(*SHost), nil
	}
	return nil, fmt.Errorf("empty host id")
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

func (self *SGuest) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerUpdateInput) (api.ServerUpdateInput, error) {
	if len(input.Name) > 0 && len(input.Name) < 2 {
		return input, httperrors.NewInputParameterError("name is too short")
	}

	// validate Hostname
	if len(input.Hostname) > 0 {
		if !regutils.MatchDomainName(input.Hostname) {
			return input, httperrors.NewInputParameterError("hostname should be a legal domain name")
		}
	}

	var err error
	input, err = self.GetDriver().ValidateUpdateData(ctx, self, userCred, input)
	if err != nil {
		return input, errors.Wrap(err, "GetDriver().ValidateUpdateData")
	}

	input.VirtualResourceBaseUpdateInput, err = self.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.VirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func serverCreateInput2ComputeQuotaKeys(input api.ServerCreateInput, ownerId mcclient.IIdentityProvider) SComputeResourceKeys {
	// input.Hypervisor must be set
	brand := guessBrandForHypervisor(input.Hypervisor)
	keys := GetDriver(input.Hypervisor).GetComputeQuotaKeys(
		rbacscope.ScopeProject,
		ownerId,
		brand,
	)
	if len(input.PreferHost) > 0 {
		hostObj, _ := HostManager.FetchById(input.PreferHost)
		host := hostObj.(*SHost)
		zone, _ := host.GetZone()
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
		zone, _ := wire.GetZone()
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
	input, err = isp.ToInstanceCreateInput(input)
	if len(input.Disks) == 0 {
		return nil, httperrors.NewInputParameterError("there are no disks in this instance snapshot, try another one")
	}
	return input, nil
}

func parseInstanceBackup(input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	ispi, err := InstanceBackupManager.FetchByIdOrName(nil, input.InstanceBackupId)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewBadRequestError("can't find instance backup %s", input.InstanceBackupId)
	}
	if err != nil {
		return nil, httperrors.NewInternalServerError("fetch instance backup error %s", err)
	}
	isp := ispi.(*SInstanceBackup)
	if isp.Status != api.INSTANCE_BACKUP_STATUS_READY && isp.Status != api.INSTANCE_BACKUP_STATUS_RECOVERY {
		return nil, httperrors.NewBadRequestError("Instance backup not ready")
	}
	input, err = isp.ToInstanceCreateInput(input)
	if len(input.Disks) == 0 {
		return nil, httperrors.NewInputParameterError("there are no disks in this instance backup, try another one")
	}
	return input, nil
}

func (manager *SGuestManager) ExpandBatchCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data *jsonutils.JSONDict,
	index int,
) (*api.ServerCreateInput, error) {
	input, err := cmdline.FetchServerCreateInputByJSON(data)
	if err != nil {
		return nil, err
	}
	for i := range input.Networks {
		if index < len(input.Networks[i].Macs) {
			input.Networks[i].Mac = input.Networks[i].Macs[index]
		}
		if index < len(input.Networks[i].Addresses) {
			input.Networks[i].Address = input.Networks[i].Addresses[index]
		}
		if index < len(input.Networks[i].Addresses6) {
			input.Networks[i].Address6 = input.Networks[i].Addresses6[index]
		}
	}
	log.Debugf("ExpandBatchCreateData %s", jsonutils.Marshal(input))
	return input, nil
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
		if len(inputInstaceType) > 0 {
			input.InstanceType = inputInstaceType
		}
	} else if len(input.InstanceBackupId) > 0 {
		inputMem := input.VmemSize
		inputCpu := input.VcpuCount
		inputInstaceType := input.InstanceType
		input, err = parseInstanceBackup(input)
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
		if len(inputInstaceType) > 0 {
			input.InstanceType = inputInstaceType
		}
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
		return nil, errors.Wrap(err, "checkGuestImage")
	}

	var hypervisor string
	// var rootStorageType string
	var osProf osprofile.SOSProfile
	hypervisor = input.Hypervisor
	if hypervisor != api.HYPERVISOR_POD {
		if len(input.Disks) == 0 && input.Cdrom == "" {
			return nil, httperrors.NewInputParameterError("No bootable disk information provided")
		}
		var imgProperties map[string]string
		var imgEncryptKeyId string

		if len(input.Disks) > 0 {
			diskConfig := input.Disks[0]
			diskConfig, err = parseDiskInfo(ctx, userCred, diskConfig)
			if err != nil {
				return nil, httperrors.NewInputParameterError("Invalid root image: %s", err)
			}
			input.Disks[0] = diskConfig
			imgEncryptKeyId = diskConfig.ImageEncryptKeyId
			imgProperties = diskConfig.ImageProperties
			if imgProperties[imageapi.IMAGE_DISK_FORMAT] == "iso" {
				return nil, httperrors.NewInputParameterError("System disk does not support iso image, please consider using cdrom parameter")
			}
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

		// check boot indexes
		bm := bitmap.NewBitMap(128)
		if input.CdromBootIndex != nil && *input.CdromBootIndex >= 0 {
			bm.Set(int64(*input.CdromBootIndex))
		}
		for i := 0; i < len(input.Disks); i++ {
			if input.Disks[i].BootIndex != nil && *input.Disks[i].BootIndex >= 0 {
				if bm.Has(int64(*input.Disks[i].BootIndex)) {
					return nil, httperrors.NewInputParameterError("duplicate boot index %d", *input.Disks[i].BootIndex)
				}
				bm.Set(int64(*input.Disks[i].BootIndex))
			}
		}

		if arch := imgProperties["os_arch"]; strings.Contains(arch, "aarch") || strings.Contains(arch, "arm") {
			input.OsArch = apis.OS_ARCH_AARCH64
		}

		var imgSupportUEFI *bool
		if desc, ok := imgProperties[imageapi.IMAGE_UEFI_SUPPORT]; ok {
			support := desc == "true"
			imgSupportUEFI = &support
		}
		if input.OsArch == apis.OS_ARCH_AARCH64 {
			// arm image supports UEFI by default
			support := true
			imgSupportUEFI = &support
		}

		switch {
		case imgSupportUEFI != nil && *imgSupportUEFI:
			if len(input.Bios) == 0 {
				input.Bios = "UEFI"
			} else if input.Bios != "UEFI" {
				return nil, httperrors.NewInputParameterError("UEFI image requires UEFI boot mode")
			}
		default:
			// not UEFI or not detectable
			if input.Bios == "UEFI" {
				return nil, httperrors.NewInputParameterError("UEFI boot mode requires UEFI image")
			}
		}

		if len(imgProperties) == 0 {
			imgProperties = map[string]string{"os_type": "Linux"}
		}
		input.DisableUsbKbd = imgProperties[imageapi.IMAGE_DISABLE_USB_KBD] == "true"
		imgIsWindows := imgProperties[imageapi.IMAGE_OS_TYPE] == "Windows"

		hasGpuVga := func() bool {
			for i := 0; i < len(input.IsolatedDevices); i++ {
				if input.IsolatedDevices[i].DevType == api.GPU_VGA_TYPE {
					return true
				}
			}
			return false
		}()
		if imgIsWindows && hasGpuVga && input.Bios != "UEFI" {
			return nil, httperrors.NewInputParameterError("Windows use gpu vga requires UEFI image")
		}

		if vdi, ok := imgProperties[imageapi.IMAGE_VDI_PROTOCOL]; ok && len(vdi) > 0 && len(input.Vdi) == 0 {
			input.Vdi = vdi
		}

		if input.EncryptKeyId == nil && len(imgEncryptKeyId) > 0 {
			input.EncryptKeyId = &imgEncryptKeyId
		}
		if input.EncryptKeyId != nil || input.EncryptKeyNew != nil {
			input.EncryptedResourceCreateInput, err = manager.SEncryptedResourceManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EncryptedResourceCreateInput)
			if err != nil {
				return nil, errors.Wrap(err, "SEncryptedResourceManager.ValidateCreateData")
			}
			if len(imgEncryptKeyId) > 0 {
				if imgEncryptKeyId != *input.EncryptKeyId {
					return nil, errors.Wrap(httperrors.ErrConflict, "encryption key inconsist with image")
				}
			}
		}

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

	optionSystemHypervisor := []string{api.HYPERVISOR_KVM, api.HYPERVISOR_ESXI, api.HYPERVISOR_POD}

	if !utils.IsInStringArray(input.Hypervisor, optionSystemHypervisor) && len(input.Disks[0].ImageId) == 0 && len(input.Disks[0].SnapshotId) == 0 && input.Cdrom == "" {
		return nil, httperrors.NewBadRequestError("Miss operating system???")
	}

	if input.Hypervisor == api.HYPERVISOR_KVM {
		if input.IsDaemon == nil && options.Options.SetKVMServerAsDaemonOnCreate {
			setDaemon := true
			input.IsDaemon = &setDaemon
		}
	}

	hypervisor = input.Hypervisor
	if hypervisor != api.HYPERVISOR_POD {
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
			if sku.AttachedDiskSizeGB == 0 {
				return nil, httperrors.NewInputParameterError("sku %s not indicate attached disk size", sku.Name)
			}
			if len(sku.AttachedDiskType) == 0 {
				return nil, httperrors.NewInputParameterError("sku %s not indicate attached disk backend", sku.Name)
			}
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
		// validate root disk config
		{
			if rootDiskConfig.NVMEDevice != nil {
				return nil, httperrors.NewBadRequestError("NVMe device can't assign as root disk")
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
		}

		for i := 0; i < len(dataDiskDefs); i += 1 {
			diskConfig, err := parseDiskInfo(ctx, userCred, dataDiskDefs[i])
			if err != nil {
				return nil, httperrors.NewInputParameterError("parse disk description error %s", err)
			}
			if diskConfig.DiskType == api.DISK_TYPE_SYS {
				log.Warningf("Snapshot error: disk index %d > 0 but disk type is %s", i+1, api.DISK_TYPE_SYS)
				diskConfig.DiskType = api.DISK_TYPE_DATA
			}
			if len(diskConfig.Backend) == 0 {
				diskConfig.Backend = rootDiskConfig.Backend
			}
			if len(diskConfig.Driver) == 0 {
				diskConfig.Driver = osProf.DiskDriver
			}
			if diskConfig.NVMEDevice != nil {
				if input.Backup {
					return nil, httperrors.NewBadRequestError("Cannot create backup with isolated device")
				}
				devConfig, err := IsolatedDeviceManager.parseDeviceInfo(userCred, diskConfig.NVMEDevice)
				if err != nil {
					return nil, httperrors.NewInputParameterError("parse isolated device description error %s", err)
				}
				err = IsolatedDeviceManager.isValidNVMEDeviceInfo(devConfig)
				if err != nil {
					return nil, err
				}
				diskConfig.NVMEDevice = devConfig
				diskConfig.Driver = api.DISK_DRIVER_VFIO
				diskConfig.Backend = api.STORAGE_NVME_PT
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
	defaultGwCnt := 0
	firstExit := -1
	for idx := 0; idx < len(netArray); idx += 1 {
		netConfig, err := parseNetworkInfo(ctx, userCred, netArray[idx])
		if err != nil {
			return nil, httperrors.NewInputParameterError("parse network description error %s", err)
		}
		err = isValidNetworkInfo(ctx, userCred, netConfig, "")
		if err != nil {
			return nil, err
		}
		if len(netConfig.Driver) == 0 {
			netConfig.Driver = osProf.NetDriver
		}
		if netConfig.SriovDevice != nil {
			if input.Backup {
				return nil, httperrors.NewBadRequestError("Cannot create backup with isolated device")
			}
			devConfig, err := IsolatedDeviceManager.parseDeviceInfo(userCred, netConfig.SriovDevice)
			if err != nil {
				return nil, httperrors.NewInputParameterError("parse isolated device description error %s", err)
			}
			err = IsolatedDeviceManager.isValidNicDeviceInfo(devConfig)
			if err != nil {
				return nil, err
			}
			netConfig.SriovDevice = devConfig
			netConfig.Driver = api.NETWORK_DRIVER_VFIO
		}

		netConfig.Project = ownerId.GetProjectId()
		netConfig.Domain = ownerId.GetProjectDomainId()
		if netConfig.IsDefault {
			defaultGwCnt++
		}
		if firstExit < 0 && netConfig.Exit {
			firstExit = idx
		}
		input.Networks[idx] = netConfig
	}
	// check default gateway
	if defaultGwCnt == 0 {
		defIdx := 0
		if firstExit >= 0 {
			// there is a exit network, make it the default
			defIdx = firstExit
		}
		// make the first nic as default
		input.Networks[defIdx].IsDefault = true
	} else if defaultGwCnt > 1 {
		return nil, errors.Wrapf(httperrors.ErrInputParameter, "more than 1 nic(%d) assigned as default gateway", defaultGwCnt)
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
		err = IsolatedDeviceManager.isValidDeviceInfo(devConfig)
		if err != nil {
			return nil, err
		}
		input.IsolatedDevices[idx] = devConfig
	}

	nvidiaVgpuCnt := 0
	gpuCnt := 0
	for i := 0; i < len(input.IsolatedDevices); i++ {
		if input.IsolatedDevices[i].DevType == api.LEGACY_VGPU_TYPE {
			nvidiaVgpuCnt += 1
		} else if utils.IsInStringArray(input.IsolatedDevices[i].DevType, api.VALID_GPU_TYPES) {
			gpuCnt += 1
		}
	}

	if nvidiaVgpuCnt > 1 {
		return nil, httperrors.NewBadRequestError("Nvidia vgpu count exceed > 1")
	}
	if nvidiaVgpuCnt > 0 && gpuCnt > 0 {
		return nil, httperrors.NewBadRequestError("Nvidia vgpu can't passthrough with other gpus")
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
		input.SecgroupId = options.Options.DefaultSecurityGroupId
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
	name := input.Name
	if len(name) == 0 {
		name = input.GenerateName
	}
	input.HostnameInput, err = manager.SHostnameResourceBaseManager.ValidateHostname(name, input.OsType, input.HostnameInput)
	if err != nil {
		return nil, err
	}

	// validate UserData
	if err := userdata.ValidateUserdata(input.UserData, input.OsType); err != nil {
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
			if eipCloudprovider != nil {
				if len(preferManagerId) > 0 && preferManagerId != eipCloudprovider.Id {
					return httperrors.NewConflictError("cannot assoicate with eip %s: different cloudprovider", eipStr)
				}
				input.PreferManager = eipCloudprovider.Id
			}

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
	if len(self.ExternalId) > 0 && (data.Contains("name") || data.Contains("__meta__") || data.Contains("description")) {
		err := self.StartRemoteUpdateTask(ctx, userCred, false, "")
		if err != nil {
			log.Errorf("StartRemoteUpdateTask fail: %s", err)
		}
	}
	if port, err := data.Int("ssh_port"); err != nil {
		err := self.SetSshPort(ctx, userCred, int(port))
		if err != nil {
			log.Errorf("unable to set sshport for guest %s", self.GetId())
		}
	}
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
		if diskConfig.DiskId != "" {
			// disk has been created, ignore resource requirement
			continue
		}
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
		if netConfig.SriovDevice != nil {
			devCount += 1
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

	if jsonutils.QueryBoolean(data, api.VM_METADATA_ENABLE_MEMCLEAN, false) {
		guest.SetMetadata(ctx, api.VM_METADATA_ENABLE_MEMCLEAN, "true", userCred)
	}
	if jsonutils.QueryBoolean(data, imageapi.IMAGE_DISABLE_USB_KBD, false) {
		guest.SetMetadata(ctx, imageapi.IMAGE_DISABLE_USB_KBD, "true", userCred)
	}

	userData, _ := data.GetString("user_data")
	if len(userData) > 0 {
		guest.setUserData(ctx, userCred, userData)
	}

	if guest.GetDriver().GetMaxSecurityGroupCount() > 0 {
		secgroups, _ := jsonutils.GetStringArray(data, "secgroups")
		for _, secgroupId := range secgroups {
			if secgroupId != guest.SecgrpId {
				gs := SGuestsecgroup{}
				gs.SecgroupId = secgroupId
				gs.GuestId = guest.Id
				GuestsecgroupManager.TableSpec().Insert(ctx, &gs)
			}
		}
	} else {
		db.Update(guest, func() error {
			guest.SecgrpId = ""
			return nil
		})
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

func (guest *SGuest) GetCreateParams(ctx context.Context, userCred mcclient.TokenCredential) (*api.ServerCreateInput, error) {
	input := new(api.ServerCreateInput)
	data := guest.GetMetadataJson(ctx, api.VM_METADATA_CREATE_PARAMS, userCred)
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

func (manager *SGuestManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data []jsonutils.JSONObject) {
	input := api.ServerCreateInput{}
	data[0].Unmarshal(&input)
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
	ctx context.Context,
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
		if backupHost != nil {
			if len(fields) == 0 || fields.Contains("backup_host_name") {
				out.BackupHostName = backupHost.Name
			}
			if len(fields) == 0 || fields.Contains("backup_host_status") {
				out.BackupHostStatus = backupHost.HostStatus
			}
			out.BackupGuestSyncStatus = self.GetGuestBackupMirrorJobStatus(ctx, userCred)
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
	out.FloppySupport, _ = self.GetDriver().IsSupportFloppy(self)

	out.MonitorUrl = self.GetDriver().FetchMonitorUrl(ctx, self)

	return out
}

func (self *SGuestManager) GetMetadataHiddenKeys() []string {
	return []string{
		api.VM_METADATA_CREATE_PARAMS,
	}
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

func (self *SGuest) GetCdrom() *SGuestcdrom {
	return self.getCdrom(false, 0)
}

func (self *SGuest) GetCdromByOrdinal(ordinal int64) *SGuestcdrom {
	return self.getCdrom(false, ordinal)
}

func (self *SGuest) getCdrom(create bool, ordinal int64) *SGuestcdrom {
	cdrom := SGuestcdrom{}
	cdrom.SetModelManager(GuestcdromManager, &cdrom)

	err := GuestcdromManager.Query().Equals("id", self.Id).Equals("ordinal", ordinal).First(&cdrom)
	if err != nil {
		if err == sql.ErrNoRows {
			if create {
				cdrom.Id = self.Id
				cdrom.Ordinal = int(ordinal)
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

func (self *SGuest) getCdroms() ([]SGuestcdrom, error) {
	cdroms := make([]SGuestcdrom, 0)
	q := GuestcdromManager.Query().Equals("id", self.Id)
	err := db.FetchModelObjects(GuestcdromManager, q, &cdroms)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return cdroms, nil
}

func (self *SGuest) getFloppys() ([]SGuestfloppy, error) {
	floppys := make([]SGuestfloppy, 0)
	q := GuestFloppyManager.Query().Equals("id", self.Id)
	err := db.FetchModelObjects(GuestFloppyManager, q, &floppys)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return floppys, nil
}

func (self *SGuest) getFloppy(create bool, ordinal int64) *SGuestfloppy {
	floppy := SGuestfloppy{}
	floppy.SetModelManager(GuestFloppyManager, &floppy)

	err := GuestFloppyManager.Query().Equals("id", self.Id).Equals("ordinal", ordinal).First(&floppy)
	if err != nil {
		if err == sql.ErrNoRows {
			if create {
				floppy.Id = self.Id
				floppy.Ordinal = int(ordinal)
				err = GuestFloppyManager.TableSpec().Insert(context.TODO(), &floppy)
				if err != nil {
					log.Errorf("insert cdrom fail %s", err)
					return nil
				}
				return &floppy
			} else {
				return nil
			}
		} else {
			log.Errorf("getFloppy query fail %s", err)
			return nil
		}
	} else {
		return &floppy
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

func (self *SGuest) getZone() (*SZone, error) {
	host, err := self.GetHost()
	if err != nil {
		return nil, err
	}
	return host.GetZone()
}

func (self *SGuest) getRegion() (*SCloudregion, error) {
	zone, err := self.getZone()
	if err != nil {
		return nil, err
	}
	return zone.GetRegion()
}

func (self *SGuest) GetOS() string {
	if len(self.OsType) > 0 {
		return self.OsType
	}
	return self.GetMetadata(context.Background(), "os_name", nil)
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

func (self *SGuest) getSecgroupJson() ([]*api.SecgroupJsonDesc, error) {
	ret := []*api.SecgroupJsonDesc{}
	secgroups, err := self.GetSecgroups()
	if err != nil {
		return nil, errors.Wrap(err, "GetSecgroups")
	}
	for _, secGrp := range secgroups {
		ret = append(ret, secGrp.getDesc())
	}
	return ret, nil
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

// 获取多个安全组规则，优先级降序排序
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
		log.Errorf("Get security group rules error: %v", err)
		return ""
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
	}
	return ""
}

func (self *SGuest) IsFailureStatus() bool {
	return strings.Index(self.Status, "fail") >= 0
}

var (
	lostNamePattern = regexp.MustCompile(`-lost@\d{8}$`)
)

func (self *SGuest) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	host, err := self.GetHost()
	if err != nil {
		return nil, errors.Wrapf(err, "GetHost")
	}
	provider, err := host.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "host.GetDriver")
	}
	if provider.GetFactory().IsOnPremise() {
		return provider.GetOnPremiseIRegion()
	}
	return host.GetIRegion(ctx)
}

func (self *SGuest) SyncRemoveCloudVM(ctx context.Context, userCred mcclient.TokenCredential, check bool) error {
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

	iregion, err := self.GetIRegion(ctx)
	if err != nil {
		return err
	}

	if check {
		iVM, err := iregion.GetIVMById(self.ExternalId)
		if err == nil { //漂移归位
			if hostId := iVM.GetIHostId(); len(hostId) > 0 {
				host, err := db.FetchByExternalIdAndManagerId(HostManager, hostId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
					host, _ := self.GetHost()
					if host != nil {
						return q.Equals("manager_id", host.ManagerId)
					}
					return q
				})
				if err == nil {
					_, err = db.Update(self, func() error {
						self.HostId = host.GetId()
						self.Status = iVM.GetStatus()
						self.PowerStates = iVM.GetPowerStates()
						self.InferPowerStates()
						return nil
					})
					return err
				}
			}
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			return errors.Wrap(err, "GetIVMById")
		}
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

	if options.Options.EnableSyncPurge {
		log.Debugf("purge removed resource %s", self.Name)
		err := self.purge(ctx, userCred)
		if err != nil {
			return err
		}
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    self,
			Action: notifyclient.ActionSyncDelete,
		})
	}

	return nil
}

func (guest *SGuest) SyncAllWithCloudVM(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, extVM cloudprovider.ICloudVM, syncStatus bool) error {
	if host == nil {
		return errors.Error("guest has no host")
	}

	provider := host.GetCloudprovider()
	if provider == nil {
		return errors.Error("host has no provider")
	}

	driver, err := provider.GetProvider(ctx)
	if err != nil {
		return errors.Wrap(err, "provider.GetProvider")
	}

	err = guest.syncWithCloudVM(ctx, userCred, driver, host, extVM, provider.GetOwnerId(), syncStatus)
	if err != nil {
		return errors.Wrap(err, "guest.syncWithCloudVM")
	}

	syncVMPeripherals(ctx, userCred, guest, extVM, host, provider, driver)

	return nil
}

func (g *SGuest) SyncOsInfo(ctx context.Context, userCred mcclient.TokenCredential, extVM cloudprovider.IOSInfo) error {
	return g.GetDriver().SyncOsInfo(ctx, userCred, g, extVM)
}

func (g *SGuest) syncWithCloudVM(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, host *SHost, extVM cloudprovider.ICloudVM, syncOwnerId mcclient.IIdentityProvider, syncStatus bool) error {
	recycle := false

	if provider.GetFactory().IsSupportPrepaidResources() && g.IsPrepaidRecycle() {
		recycle = true
	}

	diff, err := db.UpdateWithLock(ctx, g, func() error {
		if options.Options.EnableSyncName && !recycle {
			newName, _ := db.GenerateAlterName(g, extVM.GetName())
			if len(newName) > 0 && newName != g.Name {
				g.Name = newName
			}
		}
		hostname := pinyinutils.Text2Pinyin(extVM.GetHostname())
		if len(hostname) > 128 {
			hostname = hostname[:128]
		}
		if extVM.GetName() != hostname {
			g.Hostname = hostname
		}
		if !g.IsFailureStatus() && syncStatus {
			g.Status = extVM.GetStatus()
			g.PowerStates = extVM.GetPowerStates()
			g.InferPowerStates()
		}

		g.VcpuCount = extVM.GetVcpuCount()
		g.CpuSockets = extVM.GetCpuSockets()
		g.BootOrder = extVM.GetBootOrder()
		g.Vga = extVM.GetVga()
		g.Vdi = extVM.GetVdi()
		if len(extVM.GetOsArch()) > 0 {
			g.OsArch = extVM.GetOsArch()
		}
		if len(g.OsType) == 0 {
			g.OsType = string(extVM.GetOsType())
		}
		if len(g.Bios) == 0 {
			g.Bios = string(extVM.GetBios())
		}
		g.Machine = extVM.GetMachine()
		if !recycle {
			g.HostId = host.Id
		}
		g.InternetMaxBandwidthOut = extVM.GetInternetMaxBandwidthOut()
		g.Throughput = extVM.GetThroughput()

		instanceType := extVM.GetInstanceType()

		if len(instanceType) > 0 {
			g.InstanceType = instanceType
		}

		memSizeMb := extVM.GetVmemSizeMB()
		if g.VmemSize == 0 || g.VmemSize != memSizeMb {
			if memSizeMb > 0 {
				g.VmemSize = memSizeMb
			} else {
				sku, _ := ServerSkuManager.FetchSkuByNameAndProvider(instanceType, provider.GetFactory().GetName(), false)
				if sku != nil && sku.MemorySizeMB > 0 {
					g.VmemSize = sku.MemorySizeMB
				}
			}
		}

		g.Hypervisor = extVM.GetHypervisor()

		if len(extVM.GetDescription()) > 0 {
			g.Description = extVM.GetDescription()
		}
		g.IsEmulated = extVM.IsEmulated()

		if provider.GetFactory().IsSupportPrepaidResources() && !recycle {
			g.BillingType = extVM.GetBillingType()
			g.ExpiredAt = extVM.GetExpiredAt()
			if g.GetDriver().IsSupportSetAutoRenew() {
				g.AutoRenew = extVM.IsAutoRenew()
			}
		}
		return nil
	})
	if err != nil {
		log.Errorf("%s", err)
		return err
	}

	db.OpsLog.LogSyncUpdate(g, diff, userCred)

	if len(diff) > 0 {
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    g,
			Action: notifyclient.ActionSyncUpdate,
		})
	}

	g.SyncOsInfo(ctx, userCred, extVM)

	if account := host.GetCloudaccount(); account != nil {
		syncVirtualResourceMetadata(ctx, userCred, g, extVM, account.ReadOnly)
	}
	if cloudprovider := host.GetCloudprovider(); cloudprovider != nil {
		SyncCloudProject(ctx, userCred, g, syncOwnerId, extVM, cloudprovider)
	}

	if provider.GetFactory().IsSupportPrepaidResources() && recycle {
		vhost, _ := g.GetHost()
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
	guest.PowerStates = extVM.GetPowerStates()
	guest.InferPowerStates()
	guest.ExternalId = extVM.GetGlobalId()
	guest.VcpuCount = extVM.GetVcpuCount()
	guest.CpuSockets = extVM.GetCpuSockets()
	guest.BootOrder = extVM.GetBootOrder()
	guest.Vga = extVM.GetVga()
	guest.Vdi = extVM.GetVdi()
	guest.OsArch = extVM.GetOsArch()
	guest.OsType = string(extVM.GetOsType())
	guest.Bios = string(extVM.GetBios())
	guest.Machine = extVM.GetMachine()
	guest.Hypervisor = extVM.GetHypervisor()
	guest.Hostname = pinyinutils.Text2Pinyin(extVM.GetHostname())
	guest.InternetMaxBandwidthOut = extVM.GetInternetMaxBandwidthOut()
	guest.Throughput = extVM.GetThroughput()
	guest.Description = extVM.GetDescription()

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

		newName, err := db.GenerateName(ctx, manager, syncOwnerId, extVM.GetName())
		if err != nil {
			return errors.Wrapf(err, "db.GenerateName")
		}
		guest.Name = newName

		return manager.TableSpec().Insert(ctx, &guest)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	guest.SyncOsInfo(ctx, userCred, extVM)

	syncVirtualResourceMetadata(ctx, userCred, &guest, extVM, false)

	if cloudprovider := host.GetCloudprovider(); cloudprovider != nil {
		SyncCloudProject(ctx, userCred, &guest, syncOwnerId, extVM, cloudprovider)
	}

	db.OpsLog.LogEvent(&guest, db.ACT_CREATE, guest.GetShortDesc(ctx), userCred)

	if guest.Status == api.VM_RUNNING {
		db.OpsLog.LogEvent(&guest, db.ACT_START, guest.GetShortDesc(ctx), userCred)
	}

	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    &guest,
		Action: notifyclient.ActionSyncCreate,
	})

	if guest.GetDriver().GetMaxSecurityGroupCount() == 0 {
		db.Update(&guest, func() error {
			guest.SecgrpId = ""
			return nil
		})
	}

	if guest.Status == api.VM_RUNNING {
		db.OpsLog.LogEvent(&guest, db.ACT_START, guest.GetShortDesc(ctx), userCred)
	}

	return &guest, nil
}

func (manager *SGuestManager) TotalCount(
	scope rbacscope.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel,
	status []string, hypervisors []string,
	includeSystem bool, pendingDelete bool,
	hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string,
	since *time.Time,
	policyResult rbacutils.SPolicyResult,
) SGuestCountStat {
	return usageTotalGuestResouceCount(scope, ownerId, rangeObjs, status, hypervisors, includeSystem, pendingDelete, hostTypes, resourceTypes, providers, brands, cloudEnv, since, policyResult)
}

func (self *SGuest) detachNetworks(ctx context.Context, userCred mcclient.TokenCredential, gns []SGuestnetwork, reserve bool) error {
	err := GuestnetworkManager.DeleteGuestNics(ctx, userCred, gns, reserve)
	if err != nil {
		return err
	}
	host, _ := self.GetHost()
	if host != nil {
		host.ClearSchedDescCache() // ignore error
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
	val := self.GetMetadata(context.Background(), "__os_profile__", nil)
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
// # IpAddr when specified must be part of the network
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
	Ip6Addr             string
	AllocDir            api.IPAllocationDirection
	TryReserved         bool
	RequireDesignatedIP bool
	UseDesignatedIP     bool
	RequireIPv6         bool

	BwLimit        int
	NicDriver      string
	NumQueues      int
	RxTrafficLimit int64
	TxTrafficLimit int64
	NicConfs       []SNicConfig

	Virtual bool

	IsDefault bool

	PendingUsage quotas.IQuota
}

func (args *Attach2NetworkArgs) onceArgs(i int) attach2NetworkOnceArgs {
	if i < 0 || i > len(args.NicConfs)-1 {
		return attach2NetworkOnceArgs{}
	}
	r := attach2NetworkOnceArgs{
		network: args.Network,

		ipAddr:              args.IpAddr,
		ip6Addr:             args.Ip6Addr,
		allocDir:            args.AllocDir,
		tryReserved:         args.TryReserved,
		requireDesignatedIP: args.RequireDesignatedIP,
		useDesignatedIP:     args.UseDesignatedIP,
		requireIPv6:         args.RequireIPv6,

		bwLimit:        args.BwLimit,
		nicDriver:      args.NicDriver,
		numQueues:      args.NumQueues,
		txTrafficLimit: args.TxTrafficLimit,
		rxTrafficLimit: args.RxTrafficLimit,
		nicConf:        args.NicConfs[i],

		virtual: args.Virtual,

		isDefault: args.IsDefault,

		pendingUsage: args.PendingUsage,
	}
	if i > 0 {
		r.ipAddr = ""
		r.ip6Addr = ""
		r.bwLimit = 0
		r.virtual = true
		r.tryReserved = false
		r.requireDesignatedIP = false
		r.useDesignatedIP = false
		r.nicConf = args.NicConfs[i]
		r.nicDriver = ""
		r.numQueues = 1
		r.isDefault = false
		r.requireIPv6 = false
	}
	return r
}

type attach2NetworkOnceArgs struct {
	network *SNetwork

	ipAddr              string
	ip6Addr             string
	allocDir            api.IPAllocationDirection
	tryReserved         bool
	requireDesignatedIP bool
	useDesignatedIP     bool
	requireIPv6         bool

	bwLimit        int
	nicDriver      string
	numQueues      int
	nicConf        SNicConfig
	teamWithMac    string
	rxTrafficLimit int64
	txTrafficLimit int64

	virtual bool

	isDefault bool

	pendingUsage quotas.IQuota
}

func (self *SGuest) Attach2Network(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	args Attach2NetworkArgs,
) ([]SGuestnetwork, error) {
	log.Debugf("Attach2Network %s", jsonutils.Marshal(args))

	onceArgs := args.onceArgs(0)
	firstNic, err := self.attach2NetworkOnce(ctx, userCred, onceArgs)
	if err != nil {
		return nil, errors.Wrap(err, "self.attach2NetworkOnce")
	}
	retNics := []SGuestnetwork{*firstNic}
	if len(args.NicConfs) > 1 {
		firstMac, _ := netutils.ParseMac(firstNic.MacAddr)
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
		ip6Addr:             args.ip6Addr,
		allocDir:            args.allocDir,
		tryReserved:         args.tryReserved,
		requireDesignatedIP: args.requireDesignatedIP,
		useDesignatedIP:     args.useDesignatedIP,
		requireIPv6:         args.requireIPv6,

		ifname:         args.nicConf.Ifname,
		macAddr:        args.nicConf.Mac,
		bwLimit:        args.bwLimit,
		nicDriver:      nicDriver,
		numQueues:      args.numQueues,
		teamWithMac:    args.teamWithMac,
		rxTrafficLimit: args.rxTrafficLimit,
		txTrafficLimit: args.txTrafficLimit,

		virtual: args.virtual,

		isDefault: args.isDefault,
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
	vnetId := vnic.GetINetworkId()
	if len(vnetId) == 0 {
		if vnic.InClassicNetwork() {
			region, _ := host.GetRegion()
			cloudprovider := host.GetCloudprovider()
			vpc, err := VpcManager.GetOrCreateVpcForClassicNetwork(ctx, cloudprovider, region)
			if err != nil {
				return nil, errors.Wrap(err, "NewVpcForClassicNetwork")
			}
			zone, _ := host.GetZone()
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
	localNetObj, err := db.FetchByExternalIdAndManagerId(NetworkManager, vnetId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		// vpc := VpcManager.Query().SubQuery()
		wire := WireManager.Query().SubQuery()
		return q.Join(wire, sqlchemy.Equals(q.Field("wire_id"), wire.Field("id"))).
			Filter(sqlchemy.Equals(wire.Field("manager_id"), host.ManagerId))
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot find network of external_id %s", vnetId)
	}
	localNet := localNetObj.(*SNetwork)
	return localNet, nil
}

func (self *SGuest) SyncVMNics(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	host *SHost,
	vnics []cloudprovider.ICloudNic,
	ipList []string,
) compare.SyncResult {
	result := compare.SyncResult{}

	nics, err := self.GetNetworks("")
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SGuestnetwork, 0)
	commondb := make([]SGuestnetwork, 0)
	commonext := make([]cloudprovider.ICloudNic, 0)
	added := make([]cloudprovider.ICloudNic, 0)
	set := compare.SCompareSet{
		DBFunc:  "GetMAC",
		DBSet:   nics,
		ExtFunc: "GetMAC",
		ExtSet:  vnics,
	}
	err = compare.CompareSetsFunc(set, &removed, &commondb, &commonext, &added, nil)
	if err != nil {
		result.Error(errors.Wrapf(err, "compare.CompareSets"))
		return result
	}

	log.Debugf("SyncVMNics: removed: %d common: %d add: %d", len(removed), len(commondb), len(added))

	for i := 0; i < len(removed); i += 1 {
		err = self.detachNetworks(ctx, userCred, []SGuestnetwork{removed[i]}, false)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(commondb); i += 1 {
		err := NetworkAddressManager.syncGuestnetworkICloudNic(ctx, userCred, &commondb[i], commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		_, err = db.Update(&commondb[i], func() error {
			network := commondb[i].GetNetwork()
			ip := commonext[i].GetIP()
			ip6 := commonext[i].GetIP6()
			if len(ip) > 0 {
				if !network.Contains(ip) {
					localNet, err := getCloudNicNetwork(ctx, commonext[i], host, ipList, i)
					if err != nil {
						return errors.Wrapf(err, "getCloudNicNetwork")
					}
					commondb[i].NetworkId = localNet.Id
					commondb[i].IpAddr = ip
				} else {
					commondb[i].IpAddr = ip
					commondb[i].Ip6Addr = ip6
				}
			}
			commondb[i].Driver = commonext[i].GetDriver()
			return nil
		})
		if err != nil {
			result.UpdateError(errors.Wrapf(err, "db.Update"))
			continue
		}
		result.Update()
	}

	syncIps := make([]string, 0)
	for i := 0; i < len(added); i += 1 {
		localNet, err := getCloudNicNetwork(ctx, added[i], host, ipList, i)
		if err != nil {
			log.Errorf("SyncVMNics getCloudNicNetwork add fail: %s", err)
			if ip := added[i].GetIP(); len(ip) > 0 {
				syncIps = append(syncIps, ip)
			}
			result.AddError(err)
			continue
		}

		nicConf := SNicConfig{
			Mac:    added[i].GetMAC(),
			Index:  -1,
			Ifname: "",
		}

		ip := added[i].GetIP()
		// vmware, may be sync fix ip
		if len(ip) == 0 && len(ipList) > 0 {
			ip = ipList[0]
		}

		// always try allocate from reserved pool
		guestnetworks, err := self.Attach2Network(ctx, userCred, Attach2NetworkArgs{
			Network:             localNet,
			IpAddr:              ip,
			Ip6Addr:             added[i].GetIP6(),
			NicDriver:           added[i].GetDriver(),
			TryReserved:         true,
			AllocDir:            api.IPAllocationDefault,
			RequireDesignatedIP: true,
			// UseDesignatedIP:     true,
			NicConfs: []SNicConfig{nicConf},
		})
		if err != nil {
			result.AddError(err)
			continue
		}
		if len(ipList) > 0 {
			// shift
			ipList = ipList[1:]
		}
		result.Add()
		for i := range guestnetworks {
			guestnetwork := &guestnetworks[i]
			if NetworkAddressManager.syncGuestnetworkICloudNic(
				ctx, userCred, guestnetwork, added[i]); err != nil {
				result.AddError(err)
			}
		}
	}

	if len(syncIps) > 0 {
		self.SetMetadata(ctx, "sync_ips", strings.Join(syncIps, ","), userCred)
	} else {
		self.SetMetadata(ctx, "sync_ips", "None", userCred)
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
	guestdisks, _ := self.GetGuestDisks()
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

func (self *SGuest) AttachDisk(ctx context.Context, disk *SDisk, userCred mcclient.TokenCredential, driver string, cache string, mountpoint string, bootIndex *int8) error {
	return self.attach2Disk(ctx, disk, userCred, driver, cache, mountpoint, bootIndex)
}

func (self *SGuest) attach2Disk(ctx context.Context, disk *SDisk, userCred mcclient.TokenCredential, driver string, cache string, mountpoint string, bootIndex *int8) error {
	attached, err := self.isAttach2Disk(disk)
	if err != nil {
		return errors.Wrap(err, "isAttach2Disk")
	}
	if attached {
		return fmt.Errorf("Guest has been attached to disk")
	}

	if len(driver) == 0 {
		// depends the last disk of this guest
		existingDisks, _ := self.GetGuestDisks()
		if len(existingDisks) > 0 {
			prevDisk := existingDisks[len(existingDisks)-1]
			if prevDisk.Driver == api.DISK_DRIVER_IDE {
				driver = api.DISK_DRIVER_VIRTIO
			} else {
				driver = prevDisk.Driver
			}
		} else {
			osProf := self.GetOSProfile()
			driver = osProf.DiskDriver
		}
	}
	guestdisk := SGuestdisk{}
	guestdisk.SetModelManager(GuestdiskManager, &guestdisk)

	guestdisk.DiskId = disk.Id
	guestdisk.GuestId = self.Id

	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	guestdisk.Index = self.getDiskIndex()
	if bootIndex != nil {
		guestdisk.BootIndex = *bootIndex
	} else {
		guestdisk.BootIndex = -1
	}
	err = guestdisk.DoSave(ctx, driver, cache, mountpoint)
	if err == nil {
		db.OpsLog.LogAttachEvent(ctx, self, disk, userCred, nil)
	}
	return err
}

func (self *SGuest) SyncVMDisks(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider cloudprovider.ICloudProvider,
	host *SHost,
	vdisks []cloudprovider.ICloudDisk,
	syncOwnerId mcclient.IIdentityProvider,
) compare.SyncResult {
	lockman.LockRawObject(ctx, self.Id, DiskManager.Keyword())
	defer lockman.ReleaseRawObject(ctx, self.Id, DiskManager.Keyword())

	result := compare.SyncResult{}

	dbDisks, err := self.GetDisks()
	if err != nil {
		result.Error(errors.Wrapf(err, "GetDisks"))
		return result
	}

	removed := make([]SDisk, 0)
	commondb := make([]SDisk, 0)
	commonext := make([]cloudprovider.ICloudDisk, 0)
	added := make([]cloudprovider.ICloudDisk, 0)
	err = compare.CompareSets(dbDisks, vdisks, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrapf(err, "compare.CompareSets"))
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		self.DetachDisk(ctx, &removed[i], userCred)
		result.Delete()
	}

	for i := 0; i < len(commondb); i += 1 {
		if commondb[i].PendingDeleted != self.PendingDeleted { //避免主机正常,磁盘在回收站的情况
			db.Update(&commondb[i], func() error {
				commondb[i].PendingDeleted = self.PendingDeleted
				return nil
			})
		}
		commondb[i].SyncCloudProjectId(userCred, self.GetOwnerId())
		result.Update()
	}

	for i := 0; i < len(added); i += 1 {
		disk, err := DiskManager.findOrCreateDisk(ctx, userCred, provider, added[i], -1, self.GetOwnerId(), host.ManagerId)
		if err != nil {
			result.AddError(errors.Wrapf(err, "findOrCreateDisk(%s)", added[i].GetGlobalId()))
			continue
		}
		disk.SyncCloudProjectId(userCred, self.GetOwnerId())
		err = self.attach2Disk(ctx, disk, userCred, added[i].GetDriver(), added[i].GetCacheMode(), added[i].GetMountpoint(), nil)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	err = self.fixSysDiskIndex()
	if err != nil {
		result.Error(errors.Wrapf(err, "fixSysDiskIndex"))
	}

	return result
}

func (self *SGuest) fixSysDiskIndex() error {
	disks := DiskManager.Query().SubQuery()
	sysQ := GuestdiskManager.Query().Equals("guest_id", self.Id)
	sysQ = sysQ.Join(disks, sqlchemy.Equals(disks.Field("id"), sysQ.Field("disk_id"))).Filter(sqlchemy.Equals(disks.Field("disk_type"), api.DISK_TYPE_SYS))
	sysDisk := &SGuestdisk{}
	sysDisk.SetModelManager(GuestdiskManager, sysDisk)
	err := sysQ.First(sysDisk)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil
		}
		return err
	}
	if sysDisk.Index == 0 {
		return nil
	}
	q := GuestdiskManager.Query().Equals("guest_id", self.Id).Equals("index", 0)

	firstDisk := &SGuestdisk{}
	firstDisk.SetModelManager(GuestdiskManager, firstDisk)
	err = q.First(firstDisk)
	if err != nil {
		return err
	}
	_, err = db.Update(firstDisk, func() error {
		firstDisk.Index = sysDisk.Index
		return nil
	})
	if err != nil {
		return err
	}
	_, err = db.Update(sysDisk, func() error {
		sysDisk.Index = 0
		return nil
	})
	return err
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
	scope rbacscope.TRbacScope,
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
	policyResult rbacutils.SPolicyResult,
) SGuestCountStat {
	q, guests := _guestResourceCountQuery(scope, ownerId, rangeObjs, status, hypervisors,
		pendingDelete, hostTypes, resourceTypes, providers, brands, cloudEnv, since,
		policyResult,
	)
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
	scope rbacscope.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel,
	status []string,
	hypervisors []string,
	pendingDelete bool,
	hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	since *time.Time,
	policyResult rbacutils.SPolicyResult,
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

	var gq *sqlchemy.SQuery
	if since != nil && !since.IsZero() {
		gq = GuestManager.RawQuery()
	} else {
		gq = GuestManager.Query()
	}
	if len(rangeObjs) > 0 || len(hostTypes) > 0 || len(resourceTypes) > 0 || len(providers) > 0 || len(brands) > 0 || len(cloudEnv) > 0 {
		gq = filterGuestByRange(gq, rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv)
	}

	switch scope {
	case rbacscope.ScopeSystem:
	case rbacscope.ScopeDomain:
		gq = gq.Filter(sqlchemy.Equals(gq.Field("domain_id"), ownerId.GetProjectDomainId()))
	case rbacscope.ScopeProject:
		gq = gq.Filter(sqlchemy.Equals(gq.Field("tenant_id"), ownerId.GetProjectId()))
	}

	if len(status) > 0 {
		gq = gq.Filter(sqlchemy.In(gq.Field("status"), status))
	}
	if len(hypervisors) > 0 {
		gq = gq.Filter(sqlchemy.In(gq.Field("hypervisor"), hypervisors))
	}

	if pendingDelete {
		gq = gq.Filter(sqlchemy.IsTrue(gq.Field("pending_deleted")))
	} else {
		gq = gq.Filter(sqlchemy.OR(sqlchemy.IsNull(gq.Field("pending_deleted")), sqlchemy.IsFalse(gq.Field("pending_deleted"))))
	}

	if since != nil && !since.IsZero() {
		gq = gq.Filter(sqlchemy.GT(gq.Field("created_at"), *since))
	}

	gq = db.ObjectIdQueryWithPolicyResult(gq, GuestManager, policyResult)

	guests := gq.SubQuery()

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
	pendingUsage, pendingUsageZone quotas.IQuota,
	candidateNets []*schedapi.CandidateNet,
) error {
	if len(netArray) == 0 {
		netConfig := self.getDefaultNetworkConfig()
		_, err := self.attach2RandomNetwork(ctx, userCred, host, netConfig, pendingUsage)
		return errors.Wrap(err, "self.attach2RandomNetwork")
	}
	for idx := range netArray {
		netConfig, err := parseNetworkInfo(ctx, userCred, netArray[idx])
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
		if idx == 0 && netConfig.NumQueues == 0 {
			numQueues := self.VcpuCount / 2
			if numQueues > 16 {
				numQueues = 16
			}
			netConfig.NumQueues = numQueues
		}
		gns, err := self.attach2NetworkDesc(ctx, userCred, host, netConfig, pendingUsage, networkIds)
		if err != nil {
			return errors.Wrap(err, "self.attach2NetworkDesc")
		}
		if netConfig.SriovDevice != nil {
			err = self.allocSriovNicDevice(ctx, userCred, host, &gns[0], netConfig, pendingUsageZone)
			if err != nil {
				return errors.Wrap(err, "self.allocSriovNicDevice")
			}
		}

	}
	return nil
}

func (self *SGuest) allocSriovNicDevice(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	host *SHost,
	gn *SGuestnetwork, netConfig *api.NetworkConfig,
	pendingUsageZone quotas.IQuota,
) error {
	net := gn.GetNetwork()
	netConfig.SriovDevice.NetworkIndex = &gn.Index
	netConfig.SriovDevice.WireId = net.WireId
	err := self.createIsolatedDeviceOnHost(ctx, userCred, host, netConfig.SriovDevice, pendingUsageZone)
	if err != nil {
		return errors.Wrap(err, "self.createIsolatedDeviceOnHost")
	}
	dev, err := self.GetIsolatedDeviceByNetworkIndex(gn.Index)
	if err != nil {
		return errors.Wrap(err, "self.GetIsolatedDeviceByNetworkIndex")
	}
	if dev.OvsOffloadInterface != "" {
		_, err = db.Update(gn, func() error {
			gn.Ifname = dev.OvsOffloadInterface
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "update sriov network ifname")
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
	net, nicConfs, allocDir, reuseAddr, err := driver.GetNamedNetworkConfiguration(self, ctx, userCred, host, netConfig)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrapf(httperrors.ErrResourceNotReady, "Network not avaiable on host %q", host.GetName())
		} else {
			return nil, errors.Wrapf(err, "GetNamedNetworkConfiguration on host %q", host.GetName())
		}
	}
	if net != nil {
		if len(nicConfs) == 0 {
			return nil, fmt.Errorf("no available network interface?")
		}
		var sriovWires []string
		if netConfig.SriovDevice != nil {
			if netConfig.SriovDevice.Id != "" {
				idev, err := IsolatedDeviceManager.FetchById(netConfig.SriovDevice.Id)
				if err != nil {
					return nil, errors.Wrap(err, "fetch isolated device")
				}
				dev, _ := idev.(*SIsolatedDevice)
				sriovWires = []string{dev.WireId}
			} else {
				wires, err := IsolatedDeviceManager.FindUnusedNicWiresByModel(netConfig.SriovDevice.Model)
				if err != nil {
					return nil, errors.Wrap(err, "FindUnusedNicWiresByModel")
				}
				sriovWires = wires
			}
			vpc, err := net.GetVpc()
			if err != nil {
				return nil, errors.Wrap(err, "attach2NamedNetworkDesc get vpc by network")
			}
			if vpc.Id == api.DEFAULT_VPC_ID && !utils.IsInStringArray(net.WireId, sriovWires) {
				return nil, fmt.Errorf("no available sriov nic for wire %s", net.WireId)
			}
		}

		gn, err := self.Attach2Network(ctx, userCred, Attach2NetworkArgs{
			Network:             net,
			PendingUsage:        pendingUsage,
			IpAddr:              netConfig.Address,
			Ip6Addr:             netConfig.Address6,
			RequireIPv6:         netConfig.RequireIPv6,
			NicDriver:           netConfig.Driver,
			NumQueues:           netConfig.NumQueues,
			BwLimit:             netConfig.BwLimit,
			RxTrafficLimit:      netConfig.RxTrafficLimit,
			TxTrafficLimit:      netConfig.TxTrafficLimit,
			Virtual:             netConfig.Vip,
			TryReserved:         netConfig.Reserved,
			AllocDir:            allocDir,
			RequireDesignatedIP: netConfig.RequireDesignatedIP,
			UseDesignatedIP:     reuseAddr,
			NicConfs:            nicConfs,

			IsDefault: netConfig.IsDefault,
		})
		if err != nil {
			return nil, errors.Wrap(err, "Attach2Network fail")
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
		if len(disks[idx].DiskId) > 0 && len(disks[idx].Storage) > 0 {
			continue
		}
		diskConfig, err := parseDiskInfo(ctx, userCred, disks[idx])
		if err != nil {
			return errors.Wrap(err, "parseDiskInfo")
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
		if diskConfig.NVMEDevice != nil {
			err = self.attachNVMEDevice(ctx, userCred, host, pendingUsage, disk, diskConfig)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *SGuest) attachNVMEDevice(
	ctx context.Context, userCred mcclient.TokenCredential,
	host *SHost, pendingUsage quotas.IQuota,
	disk *SDisk, diskConfig *api.DiskConfig,
) error {
	gd := self.GetGuestDisk(disk.Id)
	diskConfig.NVMEDevice.DiskIndex = &gd.Index
	err := self.createIsolatedDeviceOnHost(ctx, userCred, host, diskConfig.NVMEDevice, pendingUsage)
	if err != nil {
		return errors.Wrap(err, "self.createIsolatedDeviceOnHost")
	}
	dev, err := self.GetIsolatedDeviceByDiskIndex(gd.Index)
	if err != nil {
		return errors.Wrap(err, "self.GetIsolatedDeviceByDiskIndex")
	}
	diskConfig.SizeMb = dev.NvmeSizeMB
	_, err = db.Update(disk, func() error {
		disk.DiskSize = dev.NvmeSizeMB
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "update nvme disk size")
	}
	return nil
}

func (self *SGuest) CreateDiskOnStorage(ctx context.Context, userCred mcclient.TokenCredential, storage *SStorage,
	diskConfig *api.DiskConfig, pendingUsage quotas.IQuota, inheritBilling bool, isWithServerCreate bool) (*SDisk, error) {
	lockman.LockObject(ctx, storage)
	defer lockman.ReleaseObject(ctx, storage)

	lockman.LockClass(ctx, QuotaManager, self.ProjectId)
	defer lockman.ReleaseClass(ctx, QuotaManager, self.ProjectId)

	diskName := fmt.Sprintf("vdisk-%s-%d", pinyinutils.Text2Pinyin(self.Name), time.Now().UnixNano())

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
		billingType, billingCycle, self.EncryptKeyId)

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
	disk, err := self.CreateDiskOnStorage(ctx, userCred, storage, diskConfig, pendingUsage, inheritBilling, isWithServerCreate)
	if err != nil {
		return nil, err
	}
	if diskConfig.ExistingPath != "" {
		disk.SetMetadata(ctx, api.DISK_META_EXISTING_PATH, diskConfig.ExistingPath, userCred)
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
		err = self.attach2Disk(ctx, disk, userCred, diskConfig.Driver, diskConfig.Cache, diskConfig.Mountpoint, diskConfig.BootIndex)
	}
	err = self.InheritTo(ctx, userCred, disk)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to inherit from guest %s to disk %s", self.GetId(), disk.GetId())
	}
	return disk, err
}

func (self *SGuest) CreateIsolatedDeviceOnHost(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, devs []*api.IsolatedDeviceConfig, pendingUsage quotas.IQuota) error {
	for _, devConfig := range devs {
		if devConfig.DevType == api.NIC_TYPE || devConfig.DevType == api.NVME_PT_TYPE {
			continue
		}
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
	guestdisks, err := self.GetGuestDisks()
	if err != nil {
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

func (self *SGuest) EjectIso(cdromOrdinal int64, userCred mcclient.TokenCredential) bool {
	cdrom := self.getCdrom(false, cdromOrdinal)
	if cdrom != nil && len(cdrom.ImageId) > 0 {
		imageId := cdrom.ImageId
		if cdrom.ejectIso() {
			db.OpsLog.LogEvent(self, db.ACT_ISO_DETACH, imageId, userCred)
			return true
		}
	}
	return false
}

func (self *SGuest) EjectAllIso(userCred mcclient.TokenCredential) bool {
	cdroms, _ := self.getCdroms()
	for _, cdrom := range cdroms {
		if len(cdrom.ImageId) > 0 {
			imageId := cdrom.ImageId
			if cdrom.ejectIso() {
				db.OpsLog.LogEvent(self, db.ACT_ISO_DETACH, imageId, userCred)
			} else {
				return false
			}
		}
	}
	return true
}

func (self *SGuest) EjectVfd(floppyOrdinal int64, userCred mcclient.TokenCredential) bool {
	floppy := self.getFloppy(false, floppyOrdinal)
	if floppy != nil && len(floppy.ImageId) > 0 {
		imageId := floppy.ImageId
		if floppy.ejectVfd() {
			db.OpsLog.LogEvent(self, db.ACT_VFD_DETACH, imageId, userCred)
			return true
		}
	}
	return false
}

func (self *SGuest) EjectAllVfd(userCred mcclient.TokenCredential) bool {
	floppys, _ := self.getFloppys()
	for _, floppy := range floppys {
		if len(floppy.ImageId) > 0 {
			imageId := floppy.ImageId
			if floppy.ejectVfd() {
				db.OpsLog.LogEvent(self, db.ACT_ISO_DETACH, imageId, userCred)
			} else {
				return false
			}
		}
	}
	return true
}

func (self *SGuest) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	// self.SVirtualResourceBase.Delete(ctx, userCred)
	// override
	log.Infof("guest delete do nothing")
	return nil
}

func (self *SGuest) CleanTapRecords(ctx context.Context, userCred mcclient.TokenCredential) error {
	// delete tap devices
	if err := NetTapServiceManager.removeTapServicesByGuestId(ctx, userCred, self.Id); err != nil {
		return errors.Wrap(err, "NetTapServiceManager.getTapServicesByGuestId")
	}
	if err := NetTapFlowManager.removeTapFlowsByGuestId(ctx, userCred, self.Id); err != nil {
		return errors.Wrap(err, "NetTapFlowManager.getTapServicesByGuestId")
	}
	return nil
}

func (self *SGuest) GetLoadbalancerBackends() ([]SLoadbalancerBackend, error) {
	q := LoadbalancerBackendManager.Query().Equals("backend_id", self.Id)
	ret := []SLoadbalancerBackend{}
	return ret, db.FetchModelObjects(LoadbalancerBackendManager, q, &ret)
}

func (self *SGuest) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.purge(ctx, userCred)
}

func (self *SGuest) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	overridePendingDelete := false
	purge := false
	if query != nil {
		overridePendingDelete = jsonutils.QueryBoolean(query, "override_pending_delete", false)
		purge = jsonutils.QueryBoolean(query, "purge", false)
	}
	if (overridePendingDelete || purge) && !db.IsAdminAllowDelete(ctx, userCred, self) {
		return false
	}
	return self.IsOwner(userCred) || db.IsAdminAllowDelete(ctx, userCred, self)
}

// 删除虚拟机
func (self *SGuest) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query api.ServerDeleteInput, data jsonutils.JSONObject) error {
	return self.StartDeleteGuestTask(ctx, userCred, "", query)
}

func (self *SGuest) DeleteAllDisksInDB(ctx context.Context, userCred mcclient.TokenCredential) error {
	guestDisks, err := self.GetGuestDisks()
	if err != nil {
		return errors.Wrapf(err, "GetGuestDisks")
	}
	for _, guestdisk := range guestDisks {
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
	guestdisks, _ := self.GetGuestDisks()
	if len(guestdisks) > 0 {
		disk := guestdisks[0].GetDisk()
		if len(disk.SnapshotId) > 0 {
			return false
		}
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

	if jsonutils.QueryBoolean(params, "deploy_telegraf", false) {
		influxdbUrl := self.GetDriver().FetchMonitorUrl(ctx, self)
		config.Add(jsonutils.JSONTrue, "deploy_telegraf")
		serverDetails, err := self.getDetails(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "get details")
		}
		telegrafConf, err := devtool_utils.GenerateTelegrafConf(
			serverDetails, influxdbUrl, self.OsType, self.Hypervisor)
		if err != nil {
			return nil, errors.Wrap(err, "get telegraf conf")
		}
		config.Add(jsonutils.NewString(telegrafConf), "telegraf_conf")
	}

	return config, nil
}

func (self *SGuest) getDetails(ctx context.Context, userCred mcclient.TokenCredential) (*api.ServerDetails, error) {
	res := GuestManager.FetchCustomizeColumns(ctx, userCred, jsonutils.NewDict(), []interface{}{self}, nil, false)
	jsonDict := jsonutils.Marshal(res[0]).(*jsonutils.JSONDict)
	jsonDict.Update(jsonutils.Marshal(self).(*jsonutils.JSONDict))
	serverDetails := new(api.ServerDetails)
	err := jsonDict.Unmarshal(serverDetails)
	if err != nil {
		return nil, err
	}
	return serverDetails, nil
}

func (self *SGuest) isBootIndexDuplicated(bootIndex int8) (bool, error) {
	if bootIndex < 0 {
		return false, nil
	}
	cdroms, err := self.getCdroms()
	if err != nil {
		return true, err
	}
	for i := 0; i < len(cdroms); i++ {
		if cdroms[i].BootIndex == bootIndex {
			return true, nil
		}
	}
	gd, err := self.GetGuestDisks()
	if err != nil {
		return true, err
	}
	for i := 0; i < len(gd); i++ {
		if gd[i].BootIndex == bootIndex {
			return true, nil
		}
	}
	return false, nil
}

func (self *SGuest) getVga() string {
	if utils.IsInStringArray(self.Vga, []string{"cirrus", "vmware", "qxl", "virtio", "std"}) {
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
	return self.GetMetadata(context.Background(), "kvm", nil)
}

func (self *SGuest) getExtraOptions(ctx context.Context) jsonutils.JSONObject {
	return self.GetMetadataJson(ctx, "extra_options", nil)
}

func (self *SGuest) GetIsolatedDevices() ([]SIsolatedDevice, error) {
	q := IsolatedDeviceManager.Query().Equals("guest_id", self.Id)
	devs := []SIsolatedDevice{}
	err := db.FetchModelObjects(IsolatedDeviceManager, q, &devs)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return devs, nil
}

func (self *SGuest) GetIsolatedDeviceByNetworkIndex(index int8) (*SIsolatedDevice, error) {
	dev := SIsolatedDevice{}
	q := IsolatedDeviceManager.Query().Equals("guest_id", self.Id).Equals("network_index", index)
	if cnt, err := q.CountWithError(); err != nil {
		return nil, err
	} else if cnt == 0 {
		return nil, nil
	}
	err := q.First(&dev)
	if err != nil {
		return nil, err
	}
	dev.SetModelManager(IsolatedDeviceManager, &dev)
	return &dev, nil
}

func (self *SGuest) GetIsolatedDeviceByDiskIndex(index int8) (*SIsolatedDevice, error) {
	dev := SIsolatedDevice{}
	q := IsolatedDeviceManager.Query().Equals("guest_id", self.Id).Equals("disk_index", index)
	if cnt, err := q.CountWithError(); err != nil {
		return nil, err
	} else if cnt == 0 {
		return nil, nil
	}
	err := q.First(&dev)
	if err != nil {
		return nil, err
	}
	dev.SetModelManager(IsolatedDeviceManager, &dev)
	return &dev, nil
}

func (self *SGuest) GetJsonDescAtHypervisor(ctx context.Context, host *SHost) *api.GuestJsonDesc {
	desc := &api.GuestJsonDesc{
		Name:        self.Name,
		Hostname:    self.Hostname,
		Description: self.Description,
		UUID:        self.Id,
		Mem:         self.VmemSize,
		Cpu:         self.VcpuCount,
		CpuSockets:  self.CpuSockets,
		Vga:         self.getVga(),
		Vdi:         self.GetVdi(),
		Machine:     self.getMachine(),
		Bios:        self.getBios(),
		BootOrder:   self.BootOrder,
		SrcIpCheck:  self.SrcIpCheck.Bool(),
		SrcMacCheck: self.SrcMacCheck.Bool(),
		HostId:      host.Id,

		EncryptKeyId: self.EncryptKeyId,

		IsDaemon: self.IsDaemon.Bool(),

		LightMode:  self.RescueMode,
		Hypervisor: self.GetHypervisor(),
	}

	if len(self.BackupHostId) > 0 {
		if self.HostId == host.Id {
			isMaster := true
			desc.IsMaster = &isMaster
		} else if self.BackupHostId == host.Id {
			isSlave := true
			desc.IsSlave = &isSlave
		}
	}

	if self.HostId != host.Id {
		desc.IsVolatileHost = true
	}

	// isolated devices
	isolatedDevs, _ := self.GetIsolatedDevices()
	for _, dev := range isolatedDevs {
		desc.IsolatedDevices = append(desc.IsolatedDevices, dev.getDesc())
	}

	// nics, domain
	desc.Domain = options.Options.DNSDomain
	nics, _ := self.GetNetworks("")
	for _, nic := range nics {
		nicDesc := nic.getJsonDescAtHost(ctx, host)
		desc.Nics = append(desc.Nics, nicDesc)
		if len(nicDesc.Domain) > 0 {
			desc.Domain = nicDesc.Domain
		}
	}

	{
		var prevNicDesc *api.GuestnetworkJsonDesc
		if len(desc.Nics) > 0 {
			prevNicDesc = desc.Nics[len(desc.Nics)-1]
		}
		// append tap nic
		tapNicDesc := self.getTapNicJsonDesc(ctx, prevNicDesc)
		if tapNicDesc != nil {
			desc.Nics = append(desc.Nics, tapNicDesc)
		}
	}

	// disks
	disks, _ := self.GetGuestDisks()
	for _, disk := range disks {
		diskDesc := disk.GetJsonDescAtHost(ctx, host)
		desc.Disks = append(desc.Disks, diskDesc)
	}

	cdroms, _ := self.getCdroms()
	for _, cdrom := range cdroms {
		cdromDesc := cdrom.getJsonDesc()
		desc.Cdroms = append(desc.Cdroms, cdromDesc)
	}
	if len(desc.Cdroms) > 0 {
		desc.Cdrom = desc.Cdroms[0]
	}

	//floppy
	floppys, _ := self.getFloppys()
	for _, floppy := range floppys {
		floppyDesc := floppy.getJsonDesc()
		desc.Floppys = append(desc.Floppys, floppyDesc)
	}

	// tenant
	tc, _ := self.GetTenantCache(ctx)
	if tc != nil {
		desc.Tenant = tc.GetName()
		desc.DomainId = tc.DomainId
		desc.ProjectDomain = tc.Domain
	}
	desc.TenantId = self.ProjectId

	keypair := self.getKeypair()
	if keypair != nil {
		desc.Keypair = keypair.Name
		desc.Pubkey = keypair.PublicKey
	}

	desc.NetworkRoles = self.getNetworkRoles()

	desc.Secgroups, _ = self.getSecgroupJson()

	desc.SecurityRules = self.getSecurityGroupsRules()
	desc.AdminSecurityRules = self.getAdminSecurityRules()

	desc.ExtraOptions = self.getExtraOptions(ctx)

	desc.Kvm = self.getKvmOptions()

	zone, _ := self.getZone()
	if zone != nil {
		desc.ZoneId = zone.Id
		desc.Zone = zone.Name
	}

	desc.OsName = self.GetOS()

	desc.Metadata, _ = self.GetAllMetadata(ctx, nil)

	userData, _ := desc.Metadata["user_data"]
	if len(userData) > 0 {
		decodeData, _ := userdata.Decode(userData)
		if len(decodeData) > 0 {
			userData = decodeData
		}
		desc.UserData = userData
	}
	desc.PendingDeleted = self.PendingDeleted

	// add scaling group
	sggs, err := ScalingGroupGuestManager.Fetch("", self.Id)
	if err == nil && len(sggs) > 0 {
		desc.ScalingGroupId = sggs[0].ScalingGroupId
	}

	return desc
}

func (self *SGuest) GetJsonDescAtBaremetal(ctx context.Context, host *SHost) *api.GuestJsonDesc {
	desc := &api.GuestJsonDesc{
		Name:        self.Name,
		Description: self.Description,
		UUID:        self.Id,
		Mem:         self.VmemSize,
		Cpu:         self.VcpuCount,
	}

	desc.DiskConfig = host.getDiskConfig()

	netifs := host.GetAllNetInterfaces()
	desc.Domain = options.Options.DNSDomain

	for _, nic := range netifs {
		nicDesc := nic.getServerJsonDesc()
		if len(nicDesc.Ip) > 0 {
			desc.Nics = append(desc.Nics, nicDesc)
			if len(nicDesc.Domain) > 0 {
				desc.Domain = nicDesc.Domain
			}
		} else {
			desc.NicsStandby = append(desc.NicsStandby, nicDesc)
		}
	}

	disks, _ := self.GetGuestDisks()
	for _, disk := range disks {
		diskDesc := disk.GetJsonDescAtHost(ctx, host)
		desc.Disks = append(desc.Disks, diskDesc)
	}

	tc, _ := self.GetTenantCache(ctx)
	if tc != nil {
		desc.Tenant = tc.GetName()
		desc.DomainId = tc.DomainId
		desc.ProjectDomain = tc.Domain
	}
	desc.TenantId = self.ProjectId

	keypair := self.getKeypair()
	if keypair != nil {
		desc.Keypair = keypair.Name
		desc.Pubkey = keypair.PublicKey
	}

	desc.NetworkRoles = self.getNetworkRoles()

	desc.SecurityRules = self.getSecurityGroupsRules()
	desc.AdminSecurityRules = self.getAdminSecurityRules()

	zone, _ := self.getZone()
	if zone != nil {
		desc.ZoneId = zone.Id
		desc.Zone = zone.Name
	}

	desc.OsName = self.GetOS()
	desc.Metadata, _ = self.GetAllMetadata(ctx, nil)

	desc.UserData, _ = desc.Metadata["user_data"]
	desc.PendingDeleted = self.PendingDeleted

	return desc
}

func (self *SGuest) getNetworkRoles() []string {
	key := db.Metadata.GetSysadminKey("network_role")
	roleStr := self.GetMetadata(context.Background(), key, auth.AdminCredential())
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
	guestdisks, _ := self.GetGuestDisks()
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
	guestgpus, _ := self.GetIsolatedDevices()
	gpuSpecs := []GpuSpec{}
	for _, guestgpu := range guestgpus {
		if strings.HasPrefix(guestgpu.DevType, "GPU") {
			gs := guestgpu.GetGpuSpec()
			gpuSpecs = append(gpuSpecs, *gs)
		}
	}

	spec.Set("gpu", jsonutils.Marshal(gpuSpecs))
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

func (self *SGuest) GetGpuSpec() *GpuSpec {
	if len(self.InstanceType) == 0 {
		return nil
	}
	host, err := self.GetHost()
	if err != nil {
		return nil
	}
	zone, err := host.GetZone()
	if err != nil {
		return nil
	}
	q := ServerSkuManager.Query().Equals("name", self.InstanceType).Equals("cloudregion_id", zone.CloudregionId).IsNotEmpty("gpu_spec")
	sku := &SServerSku{}
	err = q.First(sku)
	if err != nil {
		return nil
	}
	return &GpuSpec{
		Model:  sku.GpuSpec,
		Amount: sku.GpuCount,
	}
}

func (self *SGuest) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := self.SVirtualResourceBase.GetShortDesc(ctx)
	desc.Set("mem", jsonutils.NewInt(int64(self.VmemSize)))
	desc.Set("cpu", jsonutils.NewInt(int64(self.VcpuCount)))

	desc.Set("status", jsonutils.NewString(self.Status))
	desc.Set("shutdown_mode", jsonutils.NewString(self.ShutdownMode))
	if len(self.InstanceType) > 0 {
		desc.Set("instance_type", jsonutils.NewString(self.InstanceType))
	}
	if gp := self.GetGpuSpec(); gp != nil {
		desc.Set("gpu_model", jsonutils.NewString(gp.Model))
		desc.Set("gpu_count", jsonutils.NewString(gp.Amount))
	}

	address := jsonutils.NewString(strings.Join(self.GetRealIPs(), ","))
	desc.Set("ip_addr", address)

	if len(self.OsType) > 0 {
		desc.Add(jsonutils.NewString(self.OsType), "os_type")
	}
	if osDist := self.GetMetadata(ctx, "os_distribution", nil); len(osDist) > 0 {
		desc.Add(jsonutils.NewString(osDist), "os_distribution")
	}
	if osVer := self.GetMetadata(ctx, "os_version", nil); len(osVer) > 0 {
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

	host, _ := self.GetHost()

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

	if priceKey := self.GetMetadata(ctx, "ext:price_key", nil); len(priceKey) > 0 {
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
	Os               string
	Account          string
	Key              string
	Distro           string
	Version          string
	Arch             string
	Language         string
	TelegrafDeployed bool
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
	if deployInfo.TelegrafDeployed {
		info["telegraf_deployed"] = true
	}
	self.SetAllMetadata(ctx, info, userCred)
	self.saveOldPassword(ctx, userCred)
}

func (self *SGuest) isAllDisksReady() bool {
	ready := true
	disks, _ := self.GetGuestDisks()
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

func (manager *SGuestManager) GetIpsInProjectWithName(projectId, name string, isExitOnly bool, addrType api.TAddressType) []string {
	name = strings.TrimSuffix(name, ".")

	ipField := "ip_addr"
	gwField := "guest_gateway"
	if addrType == api.AddressTypeIPv6 {
		ipField = "ip6_addr"
		gwField = "guest_gateway6"
	}

	guestnics := GuestnetworkManager.Query().IsNotEmpty(ipField).SubQuery()
	guestsQ := manager.Query().IsFalse("pending_deleted").Equals("hostname", name)
	if len(projectId) > 0 {
		guestsQ = guestsQ.Equals("tenant_id", projectId)
	}
	guests := guestsQ.SubQuery()
	networks := NetworkManager.Query().IsNotNull(gwField).SubQuery()

	q := guestnics.Query(guestnics.Field(ipField))
	q = q.Join(guests, sqlchemy.Equals(guests.Field("id"), guestnics.Field("guest_id")))
	q = q.Join(networks, sqlchemy.Equals(networks.Field("id"), guestnics.Field("network_id")))

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
	if addrType == api.AddressTypeIPv6 {
		return ips
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
			DeleteSnapshots:       options.Options.DeleteSnapshotExpiredRelease,
			DeleteEip:             options.Options.DeleteEipExpiredRelease,
			DeleteDisks:           options.Options.DeleteDisksExpiredRelease,
		}
		// 跳过单独在云上开机过的虚拟机，避免误清理
		if len(guests[i].GetExternalId()) > 0 {
			iVm, err := guests[i].GetIVM(ctx)
			if err == nil && iVm.GetStatus() == api.VM_RUNNING {
				if guests[i].Status != api.VM_DELETE_FAIL {
					guests[i].SetStatus(userCred, api.VM_DELETE_FAIL, "vm status is running")
				}
				continue
			}
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
	host, _ := self.GetHost()
	if host == nil {
		return fmt.Errorf("no host???")
	}
	ihost, iprovider, err := host.GetIHostAndProvider(ctx)
	if err != nil {
		return err
	}
	iVM, err := ihost.GetIVMById(self.ExternalId)
	if err != nil {
		return err
	}
	return self.syncWithCloudVM(ctx, userCred, iprovider, host, iVM, nil, true)
}

func (manager *SGuestManager) DeleteExpiredPrepaidServers(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	guests := manager.getExpiredPrepaidGuests()
	if guests == nil {
		return
	}
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
			DeleteSnapshots: options.Options.DeleteSnapshotExpiredRelease,
			DeleteEip:       options.Options.DeleteEipExpiredRelease,
			DeleteDisks:     options.Options.DeleteDisksExpiredRelease,
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
		if len(guests[i].ExternalId) > 0 && !guests[i].GetDriver().IsSupportSetAutoRenew() {
			err := guests[i].doExternalSync(ctx, userCred)
			if err == nil && guests[i].IsValidPrePaid() {
				continue
			}
		}
		guests[i].startGuestRenewTask(ctx, userCred, guests[i].BillingCycle, "")
	}
}

func (manager *SGuestManager) DeleteExpiredPostpaidServers(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	guests := manager.getExpiredPostpaidGuests()
	if len(guests) == 0 {
		log.Infof("No expired postpaid guest")
		return
	}
	for i := 0; i < len(guests); i++ {
		if len(guests[i].ExternalId) > 0 {
			err := guests[i].doExternalSync(ctx, userCred)
			if err == nil && guests[i].IsValidPostPaid() {
				continue
			}
		}
		guests[i].SetDisableDelete(userCred, false)
		opts := api.ServerDeleteInput{
			DeleteSnapshots: options.Options.DeleteSnapshotExpiredRelease,
			DeleteEip:       options.Options.DeleteEipExpiredRelease,
			DeleteDisks:     options.Options.DeleteDisksExpiredRelease,
		}
		guests[i].StartDeleteGuestTask(ctx, userCred, "", opts)
	}
}

func (self *SGuest) IsEipAssociable() error {
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return errors.Wrapf(httperrors.ErrInvalidStatus, "cannot associate eip in status %s", self.Status)
	}

	err := ValidateAssociateEip(self)
	if err != nil {
		return errors.Wrap(err, "ValidateAssociateEip")
	}

	var eip *SElasticip
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

	region, _ := self.getRegion()
	if eip == nil && extEip == nil {
		// do nothing
	} else if eip == nil && extEip != nil {
		// add
		neip, err := ElasticipManager.getEipByExtEip(ctx, userCred, extEip, provider, region, syncOwnerId)
		if err != nil {
			result.AddError(errors.Wrapf(err, "getEipByExtEip"))
		} else {
			err = neip.AssociateInstance(ctx, userCred, api.EIP_ASSOCIATE_TYPE_SERVER, self)
			if err != nil {
				result.AddError(errors.Wrapf(err, "neip.AssociateInstance"))
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
				neip, err := ElasticipManager.getEipByExtEip(ctx, userCred, extEip, provider, region, syncOwnerId)
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
	vpc, err := self.GetVpc()
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpc")
	}
	region, err := vpc.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	filter, err := region.GetDriver().GetSecurityGroupFilter(vpc)
	if err != nil {
		return nil, errors.Wrapf(err, "GetSecurityGroupFilter")
	}

	q := SecurityGroupManager.Query().In("external_id", externalIds)
	q = filter(q)
	secgroups := []SSecurityGroup{}
	err = db.FetchModelObjects(SecurityGroupManager, q, &secgroups)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return secgroups, nil
}

func (self *SGuest) SyncVMSecgroups(ctx context.Context, userCred mcclient.TokenCredential, externalIds []string) error {
	// clear secgroup if vm not support security group
	if self.GetDriver().GetMaxSecurityGroupCount() == 0 {
		_, err := db.Update(self, func() error {
			self.SecgrpId = ""
			self.AdminSecgrpId = ""
			return nil
		})
		return err
	}

	secgroups, err := self.getSecgroupsBySecgroupExternalIds(externalIds)
	if err != nil {
		return errors.Wrap(err, "getSecgroupsBySecgroupExternalIds")
	}
	secgroupIds := []string{}
	for _, secgroup := range secgroups {
		secgroupIds = append(secgroupIds, secgroup.Id)
	}

	return self.SaveSecgroups(ctx, userCred, secgroupIds)
}

func (self *SGuest) GetIVM(ctx context.Context) (cloudprovider.ICloudVM, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty externalId")
	}
	host, err := self.GetHost()
	if err != nil {
		return nil, errors.Wrapf(err, "GetHost")
	}
	iregion, err := host.GetIRegion(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIRegion")
	}
	ihost, err := iregion.GetIHostById(host.ExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIHost")
	}
	ivm, err := ihost.GetIVMById(self.ExternalId)
	if err != nil {
		if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, errors.Wrapf(err, "GetIVMById(%s)", self.ExternalId)
		}
		return iregion.GetIVMById(self.ExternalId)
	}
	return ivm, nil
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
		rootStorage, _ := diskCat.Root.GetStorage()
		if rootStorage != nil {
			return rootStorage.StorageType
		}
	}
	return api.STORAGE_LOCAL
}

func (self *SGuest) GetApptags() []string {
	tagsStr := self.GetMetadata(context.Background(), api.VM_METADATA_APP_TAGS, nil)
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
	guestDisks, err := self.GetGuestDisks()
	if err != nil {
		return false, errors.Wrapf(err, "GetGuestDisks")
	}
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

	host, _ := self.GetHost()
	return host.ClearSchedDescCache()
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

func (self *SGuest) ToCreateInput(ctx context.Context, userCred mcclient.TokenCredential) *api.ServerCreateInput {
	genInput := self.toCreateInput()
	userInput, err := self.GetCreateParams(ctx, userCred)
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
	userInput.Hostname = ""
	return userInput
}

func (self *SGuest) toCreateInput() *api.ServerCreateInput {
	r := new(api.ServerCreateInput)
	r.VmemSize = self.VmemSize
	r.VcpuCount = int(self.VcpuCount)
	if guestCdrom := self.getCdrom(false, 0); guestCdrom != nil {
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
	if host, _ := self.GetHost(); host != nil {
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
	if zone, _ := self.getZone(); zone != nil {
		region, _ := zone.GetRegion()
		r.PreferRegion = region.GetId()
		r.PreferZone = zone.GetId()
	}
	return r
}

func (self *SGuest) ToDisksConfig() []*api.DiskConfig {
	guestDisks, err := self.GetGuestDisks()
	if err != nil {
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
		storage, _ := disk.GetStorage()
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
	guestIsolatedDevices, _ := self.GetIsolatedDevices()
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

func (self *SGuest) IsImport(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return self.GetMetadata(ctx, "__is_import", userCred) == "true"
}

func (guest *SGuest) GetDetailsRemoteNics(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	iVM, err := guest.GetIVM(ctx)
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
	guestDisks, err := self.GetGuestDisks()
	if err != nil {
		return nil, errors.Wrapf(err, "GetGuestDisks")
	}
	diskIds := make([]string, len(guestDisks))
	for i := 0; i < len(guestDisks); i++ {
		diskIds[i] = guestDisks[i].DiskId
	}
	snapshots := make([]SSnapshot, 0)
	q := SnapshotManager.Query().IsFalse("fake_deleted").In("disk_id", diskIds)
	sq := InstanceSnapshotJointManager.Query("snapshot_id").SubQuery()
	q = q.LeftJoin(sq, sqlchemy.Equals(q.Field("id"), sq.Field("snapshot_id"))).
		Filter(sqlchemy.IsNull(sq.Field("snapshot_id")))
	err = db.FetchModelObjects(SnapshotManager, q, &snapshots)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
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
	if len(input.GuestImageID) == 0 {
		return nil
	}

	guestImageId := input.GuestImageID
	params := jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "details")

	s := auth.GetAdminSession(ctx, options.Options.Region)
	ret, err := image.GuestImages.Get(s, guestImageId, params)
	if err != nil {
		return errors.Wrap(err, "get guest image from glance error")
	}

	images := &api.SImagesInGuest{}
	err = ret.Unmarshal(images)
	if err != nil {
		return errors.Wrap(err, "unmarshal guest image")
	}
	input.GuestImageID = images.Id

	log.Infof("usage guest image %s(%s)", images.Name, images.Id)

	if len(input.Disks) > 0 {
		input.Disks[0].ImageId = images.RootImage.Id
	} else {
		input.Disks = append(input.Disks,
			&api.DiskConfig{
				ImageId: images.RootImage.Id,
			},
		)
	}
	for i := range images.DataImages {
		if len(input.Disks) > i+1 {
			input.Disks[i+1].ImageId = images.DataImages[i].Id
		} else {
			input.Disks = append(input.Disks,
				&api.DiskConfig{
					ImageId: images.DataImages[i].Id,
				},
			)
		}
	}

	return nil
}

func (self *SGuest) GetDiskIndex(diskId string) int8 {
	guestDisks, _ := self.GetGuestDisks()
	for _, gd := range guestDisks {
		if gd.DiskId == diskId {
			return gd.Index
		}
	}
	return -1
}

func (guest *SGuest) GetRegionalQuotaKeys() (quotas.IQuotaKeys, error) {
	host, _ := guest.GetHost()
	if host == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid host")
	}
	provider := host.GetCloudprovider()
	if provider == nil && len(host.ManagerId) > 0 {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid manager")
	}
	region, _ := host.GetRegion()
	if region == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid region")
	}
	return fetchRegionalQuotaKeys(rbacscope.ScopeProject, guest.GetOwnerId(), region, provider), nil
}

func (guest *SGuest) GetQuotaKeys() (quotas.IQuotaKeys, error) {
	host, _ := guest.GetHost()
	if host == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid host")
	}
	provider := host.GetCloudprovider()
	if provider == nil && len(host.ManagerId) > 0 {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid manager")
	}
	zone, _ := host.GetZone()
	if zone == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid zone")
	}
	hypervisor := guest.Hypervisor
	if !utils.IsInStringArray(hypervisor, api.ONECLOUD_HYPERVISORS) {
		hypervisor = ""
	}
	return fetchComputeQuotaKeys(
		rbacscope.ScopeProject,
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
	if len(guest.ExternalId) == 0 || options.Options.KeepTagLocalization {
		return
	}
	host, err := guest.GetHost()
	if err != nil {
		return
	}
	if account := host.GetCloudaccount(); account != nil && account.ReadOnly {
		return
	}
	err = guest.StartRemoteUpdateTask(ctx, userCred, true, "")
	if err != nil {
		log.Errorf("StartRemoteUpdateTask fail: %s", err)
	}
}

func (self *SGuest) GetAddress() (string, error) {
	gns, err := self.GetNetworks("")
	if err != nil {
		return "", errors.Wrapf(err, "GetNetworks")
	}
	for _, gn := range gns {
		if !gn.IsExit() {
			return gn.IpAddr, nil
		}
	}
	return "", errors.Wrapf(cloudprovider.ErrNotFound, "guest %s address", self.Name)
}

func (guest *SGuest) InferPowerStates() {
	if len(guest.PowerStates) == 0 {
		switch guest.Status {
		case api.VM_READY:
			guest.PowerStates = api.VM_POWER_STATES_OFF
		case api.VM_UNKNOWN:
			guest.PowerStates = api.VM_POWER_STATES_UNKNOWN
		case api.VM_INIT:
			guest.PowerStates = api.VM_POWER_STATES_ON
		case api.VM_SCHEDULE:
			guest.PowerStates = api.VM_POWER_STATES_ON
		case api.VM_SCHEDULE_FAILED:
			guest.PowerStates = api.VM_POWER_STATES_ON
		case api.VM_CREATE_NETWORK:
			guest.PowerStates = api.VM_POWER_STATES_ON
		case api.VM_NETWORK_FAILED:
			guest.PowerStates = api.VM_POWER_STATES_ON
		case api.VM_DEVICE_FAILED:
			guest.PowerStates = api.VM_POWER_STATES_UNKNOWN
		case api.VM_CREATE_FAILED:
			guest.PowerStates = api.VM_POWER_STATES_OFF
		case api.VM_CREATE_DISK:
			guest.PowerStates = api.VM_POWER_STATES_OFF
		case api.VM_DISK_FAILED:
			guest.PowerStates = api.VM_POWER_STATES_OFF
		case api.VM_IMAGE_CACHING:
			guest.PowerStates = api.VM_POWER_STATES_ON
		case api.VM_START_DEPLOY:
			guest.PowerStates = api.VM_POWER_STATES_ON
		case api.VM_DEPLOYING:
			guest.PowerStates = api.VM_POWER_STATES_OFF
		case api.VM_START_START:
			guest.PowerStates = api.VM_POWER_STATES_ON
		case api.VM_STARTING:
			guest.PowerStates = api.VM_POWER_STATES_ON
		case api.VM_START_FAILED:
			guest.PowerStates = api.VM_POWER_STATES_OFF
		case api.VM_RUNNING:
			guest.PowerStates = api.VM_POWER_STATES_ON
		case api.VM_START_STOP:
			guest.PowerStates = api.VM_POWER_STATES_ON
		case api.VM_STOPPING:
			guest.PowerStates = api.VM_POWER_STATES_ON
		case api.VM_STOP_FAILED:
			guest.PowerStates = api.VM_POWER_STATES_ON
		case api.VM_RENEWING:
			guest.PowerStates = api.VM_POWER_STATES_ON
		case api.VM_RENEW_FAILED:
			guest.PowerStates = api.VM_POWER_STATES_ON
		case api.VM_ATTACH_DISK:
			guest.PowerStates = api.VM_POWER_STATES_UNKNOWN
		case api.VM_DETACH_DISK:
			guest.PowerStates = api.VM_POWER_STATES_UNKNOWN
		default:
			guest.PowerStates = api.VM_POWER_STATES_UNKNOWN
		}
	}
}

func (guest *SGuest) HasBackupGuest() bool {
	return guest.BackupHostId != ""
}

func (guest *SGuest) SetGuestBackupMirrorJobInProgress(ctx context.Context, userCred mcclient.TokenCredential) error {
	return guest.SetMetadata(ctx, api.MIRROR_JOB, api.MIRROR_JOB_INPROGRESS, userCred)
}

func (guest *SGuest) SetGuestBackupMirrorJobNotReady(ctx context.Context, userCred mcclient.TokenCredential) error {
	return guest.SetMetadata(ctx, api.MIRROR_JOB, "", userCred)
}

func (guest *SGuest) TrySetGuestBackupMirrorJobReady(ctx context.Context, userCred mcclient.TokenCredential) error {
	if guest.IsGuestBackupMirrorJobFailed(ctx, userCred) {
		// can't update guest backup mirror job status from failed to ready
		return nil
	}
	return guest.SetMetadata(ctx, api.MIRROR_JOB, api.MIRROR_JOB_READY, userCred)
}

func (guest *SGuest) SetGuestBackupMirrorJobFailed(ctx context.Context, userCred mcclient.TokenCredential) error {
	return guest.SetMetadata(ctx, api.MIRROR_JOB, api.MIRROR_JOB_FAILED, userCred)
}

func (guest *SGuest) IsGuestBackupMirrorJobFailed(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return guest.GetMetadata(ctx, api.MIRROR_JOB, userCred) == api.MIRROR_JOB_FAILED
}

func (guest *SGuest) IsGuestBackupMirrorJobReady(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return guest.GetMetadata(ctx, api.MIRROR_JOB, userCred) == api.MIRROR_JOB_READY
}

func (guest *SGuest) GetGuestBackupMirrorJobStatus(ctx context.Context, userCred mcclient.TokenCredential) string {
	return guest.GetMetadata(ctx, api.MIRROR_JOB, userCred)
}

func (guest *SGuest) ResetGuestQuorumChildIndex(ctx context.Context, userCred mcclient.TokenCredential) error {
	return guest.SetMetadata(ctx, api.QUORUM_CHILD_INDEX, "", userCred)
}

type SGuestTotalCount struct {
	apis.TotalCountBase
	CpuCount  int
	MemMb     int
	DiskMb    int64
	DiskCount int
}

func (manager *SGuestManager) CustomizedTotalCount(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, totalQ *sqlchemy.SQuery) (int, jsonutils.JSONObject, error) {
	results := SGuestTotalCount{}

	totalQ = totalQ.AppendField(sqlchemy.SUM("cpu_count", totalQ.Field("vcpu_count")))
	totalQ = totalQ.AppendField(sqlchemy.SUM("mem_mb", totalQ.Field("vmem_size")))

	err := totalQ.First(&results)
	if err != nil {
		return -1, nil, errors.Wrap(err, "SGuestManager query total")
	}

	log.Debugf("CustomizedTotalCount %s", jsonutils.Marshal(results))

	diskQ := DiskManager.Query()
	diskGuestQ := GuestdiskManager.Query().SubQuery()
	diskQ = diskQ.Join(diskGuestQ, sqlchemy.Equals(diskQ.Field("id"), diskGuestQ.Field("disk_id")))
	totalSQ := totalQ.ResetFields().SubQuery()
	diskQ = diskQ.Join(totalSQ, sqlchemy.Equals(diskGuestQ.Field("guest_id"), totalSQ.Field("id")))
	diskQ = diskQ.AppendField(sqlchemy.COUNT("disk_count"))
	diskQ = diskQ.AppendField(sqlchemy.SUM("disk_mb", diskQ.Field("disk_size")))

	err = diskQ.First(&results)
	if err != nil {
		return -1, nil, errors.Wrap(err, "SGuestManager query total_disk")
	}

	// log.Debugf("CustomizedTotalCount %s", jsonutils.Marshal(results))

	return results.Count, jsonutils.Marshal(results), nil
}

func (guest *SGuest) IsSriov() bool {
	nics, err := guest.GetNetworks("")
	if err != nil {
		log.Errorf("guest.GetNetworks fail %s", err)
		return false
	}
	for i := range nics {
		if nics[i].Driver == api.NETWORK_DRIVER_VFIO {
			return true
		}
	}
	return false
}
