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
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	napi "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/scheduler"
	"yunion.io/x/onecloud/pkg/util/cgrouputils/cpuset"
	"yunion.io/x/onecloud/pkg/util/k8s/tokens"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SHostManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SZoneResourceBaseManager
	SManagedResourceBaseManager
	SHostnameResourceBaseManager
}

var HostManager *SHostManager

func init() {
	HostManager = &SHostManager{
		SEnabledStatusInfrasResourceBaseManager: db.NewEnabledStatusInfrasResourceBaseManager(
			SHost{},
			"hosts_tbl",
			"host",
			"hosts",
		),
	}
	HostManager.SetVirtualObject(HostManager)
	HostManager.SetAlias("baremetal", "baremetals")
	notifyclient.AddNotifyDBHookResources(HostManager.KeywordPlural(), HostManager.AliasPlural())
	GuestManager.NameRequireAscii = false
}

type SHost struct {
	db.SEnabledStatusInfrasResourceBase
	db.SExternalizedResourceBase
	SZoneResourceBase `update:""`
	SManagedResourceBase
	SBillingResourceBase
	SHostnameResourceBase

	// 机架
	Rack string `width:"16" charset:"ascii" nullable:"true" get:"domain" update:"domain" create:"domain_optional"`
	// 机位
	Slots string `width:"16" charset:"ascii" nullable:"true" get:"domain" update:"domain" create:"domain_optional"`

	// 管理口MAC
	AccessMac string `width:"32" charset:"ascii" nullable:"true" index:"true" list:"domain" update:"domain"`

	// 管理口Ip地址
	AccessIp string `width:"16" charset:"ascii" nullable:"true" list:"domain" update:"domain"`

	// 管理地址
	ManagerUri string `width:"256" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`

	// 系统信息
	SysInfo jsonutils.JSONObject `nullable:"true" search:"domain" list:"domain" update:"domain" create:"domain_optional"`
	// 物理机序列号信息
	SN string `width:"128" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`

	// CPU核数
	CpuCount int `nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
	// 物理CPU颗数
	NodeCount int8 `nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
	// CPU描述信息
	CpuDesc string `width:"128" charset:"ascii" nullable:"true" get:"domain" update:"domain" create:"domain_optional"`
	// CPU频率
	CpuMhz int `nullable:"true" get:"domain" update:"domain" create:"domain_optional"`
	// CPU缓存大小,单位KB
	CpuCache int `nullable:"true" get:"domain" update:"domain" create:"domain_optional"`
	// 预留CPU大小
	CpuReserved int `nullable:"true" default:"0" list:"domain" update:"domain" create:"domain_optional"`
	// CPU超分比
	CpuCmtbound float32 `nullable:"true" default:"8" list:"domain" update:"domain" create:"domain_optional"`
	// CPUMicrocode
	CpuMicrocode string `width:"64" charset:"ascii" nullable:"true" get:"domain" update:"domain" create:"domain_optional"`
	// CPU架构
	CpuArchitecture string `width:"16" charset:"ascii" nullable:"true" get:"domain" list:"domain" update:"domain" create:"domain_optional"`

	// 内存大小,单位Mb
	MemSize int `nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
	// 预留内存大小
	MemReserved int `nullable:"true" default:"0" list:"domain" update:"domain" create:"domain_optional"`
	// 内存超分比
	MemCmtbound float32 `nullable:"true" default:"1" list:"domain" update:"domain" create:"domain_optional"`
	// 页大小
	PageSizeKB int `nullable:"false" default:"4" list:"domain" update:"domain" create:"domain_optional"`

	// 存储大小,单位Mb
	StorageSize int64 `nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
	// 存储类型
	StorageType string `width:"20" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
	// 存储驱动类型
	StorageDriver string `width:"20" charset:"ascii" nullable:"true" get:"domain" update:"domain" create:"domain_optional"`
	// 存储详情
	StorageInfo jsonutils.JSONObject `nullable:"true" get:"domain" update:"domain" create:"domain_optional"`

	// IPMI地址
	IpmiIp string `width:"16" charset:"ascii" nullable:"true" list:"domain"`

	// IPMI详情
	IpmiInfo jsonutils.JSONObject `nullable:"true" get:"domain" update:"domain" create:"domain_optional"`

	// 宿主机状态
	// example: online
	HostStatus string `width:"16" charset:"ascii" nullable:"false" default:"offline" list:"domain"`

	// 宿主机类型
	HostType string `width:"36" charset:"ascii" nullable:"false" list:"domain" update:"domain" create:"domain_required"`

	// host服务软件版本
	Version string `width:"128" charset:"ascii" list:"domain" update:"domain" create:"domain_optional"`
	// OVN软件版本
	OvnVersion string `width:"64" charset:"ascii" list:"domain" update:"domain" create:"domain_optional"`

	IsBaremetal bool `nullable:"true" default:"false" list:"domain" update:"domain" create:"domain_optional"`

	// 是否处于维护状态
	IsMaintenance bool `nullable:"true" default:"false" list:"domain"`

	LastPingAt time.Time ``
	// health check enabled by host agent online
	EnableHealthCheck bool `nullable:"true" default:"false"`

	ResourceType string `width:"36" charset:"ascii" nullable:"false" list:"domain" update:"domain" create:"domain_optional" default:"shared"`

	RealExternalId string `width:"256" charset:"utf8" get:"domain"`

	// 是否为导入的宿主机
	IsImport bool `nullable:"true" default:"false" list:"domain" create:"domain_optional"`

	// 是否允许PXE启动
	EnablePxeBoot tristate.TriState `default:"true" list:"domain" create:"domain_optional" update:"domain"`

	// 主机UUID
	Uuid string `width:"64" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`

	// 主机启动模式, 可能值为PXE和ISO
	BootMode string `width:"8" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`

	// IPv4地址，作为私有云vpc访问外网时的网关
	OvnMappedIpAddr string `width:"16" charset:"ascii" nullable:"true" list:"user"`

	// UEFI详情
	UefiInfo jsonutils.JSONObject `nullable:"true" get:"domain" update:"domain" create:"domain_optional"`
}

func (manager *SHostManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{ZoneManager},
	}
}

// 宿主机/物理机列表
func (manager *SHostManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.HostListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SZoneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ZonalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemFilter")
	}

	resType := query.ResourceType
	if len(resType) > 0 {
		switch resType {
		case api.HostResourceTypeShared:
			q = q.Filter(
				sqlchemy.OR(
					sqlchemy.IsNullOrEmpty(q.Field("resource_type")),
					sqlchemy.Equals(q.Field("resource_type"), api.HostResourceTypeShared),
				),
			)
		default:
			q = q.Equals("resource_type", resType)
		}
	}

	q, err = manager.SEnabledStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemFilter")
	}

	if len(query.AnyMac) > 0 {
		anyMac := netutils.FormatMacAddr(query.AnyMac)
		if len(anyMac) == 0 {
			return nil, errors.Wrapf(httperrors.ErrInputParameter, "invalid any_mac address %s", query.AnyMac)
		}
		netifs, _ := NetInterfaceManager.FetchByMac(anyMac)
		var bmIds []string
		for i := range netifs {
			if !utils.IsInArray(netifs[i].BaremetalId, bmIds) {
				bmIds = append(bmIds, netifs[i].BaremetalId)
			}
		}
		if len(bmIds) > 0 {
			q = q.In("id", bmIds)
		} else {
			q = q.Equals("access_mac", anyMac)
		}
	}
	if len(query.AnyIp) > 0 {
		hnQ := HostnetworkManager.Query("baremetal_id") //.Contains("ip_addr", query.AnyIp).SubQuery()
		conditions := []sqlchemy.ICondition{}
		for _, ip := range query.AnyIp {
			conditions = append(conditions, sqlchemy.Contains(hnQ.Field("ip_addr"), ip))
		}
		hn := hnQ.Filter(
			sqlchemy.OR(conditions...),
		)
		conditions = []sqlchemy.ICondition{}
		for _, ip := range query.AnyIp {
			conditions = append(conditions, sqlchemy.Contains(q.Field("access_ip"), ip))
			conditions = append(conditions, sqlchemy.Contains(q.Field("ipmi_ip"), ip))
		}
		conditions = append(conditions, sqlchemy.In(q.Field("id"), hn))
		q = q.Filter(sqlchemy.OR(
			conditions...,
		))
	}

	schedTagStr := query.SchedtagId
	if len(schedTagStr) > 0 {
		schedTag, _ := SchedtagManager.FetchByIdOrName(nil, schedTagStr)
		if schedTag == nil {
			return nil, httperrors.NewResourceNotFoundError("Schedtag %s not found", schedTagStr)
		}
		hostschedtags := HostschedtagManager.Query().SubQuery()
		scopeQuery := hostschedtags.Query(hostschedtags.Field("host_id")).Equals("schedtag_id", schedTag.GetId()).SubQuery()
		q = q.In("id", scopeQuery)
	}

	wireStr := query.WireId
	if len(wireStr) > 0 {
		wire, _ := WireManager.FetchByIdOrName(nil, wireStr)
		if wire == nil {
			return nil, httperrors.NewResourceNotFoundError("Wire %s not found", wireStr)
		}
		hostwires := NetInterfaceManager.Query().SubQuery()
		scopeQuery := hostwires.Query(hostwires.Field("baremetal_id")).Equals("wire_id", wire.GetId()).SubQuery()
		q = q.In("id", scopeQuery)
	}

	storageStr := query.StorageId
	if len(storageStr) > 0 {
		storage, _ := StorageManager.FetchByIdOrName(nil, storageStr)
		if storage == nil {
			return nil, httperrors.NewResourceNotFoundError("Storage %s not found", storageStr)
		}
		hoststorages := HoststorageManager.Query().SubQuery()
		scopeQuery := hoststorages.Query(hoststorages.Field("host_id")).Equals("storage_id", storage.GetId()).SubQuery()
		notAttached := (query.StorageNotAttached != nil && *query.StorageNotAttached)
		if !notAttached {
			q = q.In("id", scopeQuery)
		} else {
			q = q.NotIn("id", scopeQuery)
		}
	}

	hostStorageType := query.HostStorageType
	if len(hostStorageType) > 0 {
		hoststorages := HoststorageManager.Query()
		storages := StorageManager.Query().In("storage_type", hostStorageType).SubQuery()
		hq := hoststorages.Join(storages, sqlchemy.Equals(hoststorages.Field("storage_id"), storages.Field("id"))).SubQuery()
		scopeQuery := hq.Query(hq.Field("host_id")).SubQuery()
		q = q.In("id", scopeQuery)
	}

	hypervisorStr := query.Hypervisor
	if len(hypervisorStr) > 0 {
		hostType, ok := api.HYPERVISOR_HOSTTYPE[hypervisorStr]
		if !ok {
			return nil, httperrors.NewInputParameterError("not supported hypervisor %s", hypervisorStr)
		}
		q = q.Filter(sqlchemy.Equals(q.Field("host_type"), hostType))
	}

	usable := (query.Usable != nil && *query.Usable)
	if usable {
		hosts := HostManager.Query().SubQuery()
		netifs := NetInterfaceManager.Query().SubQuery()
		networks := NetworkManager.Query().SubQuery()
		providers := usableCloudProviders().SubQuery()

		hostQ1 := hosts.Query(hosts.Field("id"))
		hostQ1 = hostQ1.Join(providers, sqlchemy.Equals(hosts.Field("manager_id"), providers.Field("id")))
		hostQ1 = hostQ1.Join(netifs, sqlchemy.Equals(hosts.Field("id"), netifs.Field("baremetal_id")))
		hostQ1 = hostQ1.Join(networks, sqlchemy.Equals(netifs.Field("wire_id"), networks.Field("wire_id")))
		hostQ1 = hostQ1.Filter(sqlchemy.Equals(networks.Field("status"), api.NETWORK_STATUS_AVAILABLE))
		hostQ1 = hostQ1.Filter(sqlchemy.IsTrue(hosts.Field("enabled")))

		hostQ2 := hosts.Query(hosts.Field("id"))
		hostQ2 = hostQ2.Join(netifs, sqlchemy.Equals(hosts.Field("id"), netifs.Field("baremetal_id")))
		hostQ2 = hostQ2.Join(networks, sqlchemy.Equals(netifs.Field("wire_id"), networks.Field("wire_id")))
		hostQ2 = hostQ2.Filter(sqlchemy.IsNullOrEmpty(hosts.Field("manager_id")))
		hostQ2 = hostQ2.Filter(sqlchemy.Equals(networks.Field("status"), api.NETWORK_STATUS_AVAILABLE))
		hostQ2 = hostQ2.Filter(sqlchemy.IsTrue(hosts.Field("enabled")))

		q = q.Filter(sqlchemy.OR(
			sqlchemy.In(q.Field("id"), hostQ1.SubQuery()),
			sqlchemy.In(q.Field("id"), hostQ2.SubQuery()),
		))

		zones := ZoneManager.Query().SubQuery()
		q = q.Join(zones, sqlchemy.Equals(q.Field("zone_id"), zones.Field("id"))).
			Filter(sqlchemy.Equals(zones.Field("status"), api.ZONE_ENABLE))

		q = q.In("status", []string{api.HOST_STATUS_RUNNING, api.HOST_STATUS_READY})

		q = q.Filter(
			sqlchemy.OR(
				sqlchemy.AND(
					sqlchemy.NotEquals(q.Field("host_type"), api.HOST_TYPE_BAREMETAL),
					sqlchemy.Equals(q.Field("host_status"), api.HOST_ONLINE),
				),
				sqlchemy.Equals(q.Field("host_type"), api.HOST_TYPE_BAREMETAL),
			),
		)
	}

	if query.IsEmpty != nil {
		isEmpty := *query.IsEmpty
		sq := GuestManager.Query("host_id").IsNotEmpty("host_id").GroupBy("host_id").SubQuery()
		if isEmpty {
			q = q.NotIn("id", sq)
		} else {
			q = q.In("id", sq)
		}
	}

	if query.Baremetal != nil {
		isBaremetal := *query.Baremetal
		if isBaremetal {
			q = q.Equals("host_type", api.HOST_TYPE_BAREMETAL)
		} else {
			q = q.NotEquals("host_type", api.HOST_TYPE_BAREMETAL)
		}
	}

	fieldQueryMap := map[string][]string{
		"rack":             query.Rack,
		"slots":            query.Slots,
		"access_mac":       query.AccessMac,
		"access_ip":        query.AccessIp,
		"sn":               query.SN,
		"storage_type":     query.StorageType,
		"ipmi_ip":          query.IpmiIp,
		"host_status":      query.HostStatus,
		"host_type":        query.HostType,
		"version":          query.Version,
		"ovn_version":      query.OvnVersion,
		"uuid":             query.Uuid,
		"boot_mode":        query.BootMode,
		"cpu_architecture": query.CpuArchitecture,
	}

	for f, vars := range fieldQueryMap {
		vars = stringutils2.FilterEmpty(vars)
		if len(vars) > 0 {
			q = q.In(f, vars)
		}
	}

	if len(query.CpuCount) > 0 {
		q = q.In("cpu_count", query.CpuCount)
	}
	if len(query.MemSize) > 0 {
		q = q.In("mem_size", query.MemSize)
	}
	if query.IsMaintenance != nil {
		if *query.IsMaintenance {
			q = q.IsTrue("is_maintenance")
		} else {
			q = q.IsFalse("is_maintenance")
		}
	}
	if query.IsImport != nil {
		if *query.IsImport {
			q = q.IsTrue("is_import")
		} else {
			q = q.IsFalse("is_import")
		}
	}
	if query.EnablePxeBoot != nil {
		if *query.EnablePxeBoot {
			q = q.IsTrue("enable_pxe_boot")
		} else {
			q = q.IsFalse("enable_pxe_boot")
		}
	}
	if len(query.OsArch) > 0 {
		q = db.ListQueryByArchitecture(q, "cpu_architecture", query.OsArch)
	}

	// for provider onecloud
	if len(query.ServerIdForNetwork) > 0 {
		guest := GuestManager.FetchGuestById(query.ServerIdForNetwork)
		if guest != nil && guest.GetHypervisor() == api.HYPERVISOR_KVM {
			nets, _ := guest.GetNetworks("")
			if len(nets) > 0 {
				wires := []string{}
				for i := 0; i < len(nets); i++ {
					net := nets[i].GetNetwork()
					vpc, _ := net.GetVpc()
					if vpc.Id != api.DEFAULT_VPC_ID {
						q = q.IsNotEmpty("ovn_version")
					} else {
						if !utils.IsInStringArray(net.WireId, wires) {
							wires = append(wires, net.WireId)
							netifs := NetInterfaceManager.Query().SubQuery()
							scopeQuery := netifs.Query(netifs.Field("baremetal_id")).Equals("wire_id", net.WireId).SubQuery()
							q = q.In("id", scopeQuery)
						}
					}
				}
			}
		}
	}

	return q, nil
}

func (manager *SHostManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.HostListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SZoneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ZonalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.OrderByExtraFields")
	}

	if db.NeedOrderQuery([]string{query.OrderByServerCount}) {
		guests := GuestManager.Query().SubQuery()
		guestCounts := guests.Query(
			guests.Field("host_id"),
			sqlchemy.COUNT("id").Label("guest_count"),
		).GroupBy("host_id").SubQuery()
		q = q.LeftJoin(guestCounts, sqlchemy.Equals(q.Field("id"), guestCounts.Field("host_id")))
		db.OrderByFields(q, []string{query.OrderByServerCount}, []sqlchemy.IQueryField{guestCounts.Field("guest_count")})
	}

	if db.NeedOrderQuery([]string{query.OrderByCpuCommitRate}) {
		guestsQ := GuestManager.Query()
		if options.Options.IgnoreNonrunningGuests {
			guestsQ = guestsQ.Equals("status", api.VM_RUNNING)
		}
		guests := guestsQ.SubQuery()
		hosts := HostManager.Query().SubQuery()
		hostQ := hosts.Query(
			hosts.Field("id"),
			hosts.Field("cpu_count"),
			hosts.Field("cpu_reserved"),
			sqlchemy.SUM("guest_vcpu_count", guests.Field("vcpu_count")),
		).LeftJoin(guests, sqlchemy.Equals(hosts.Field("id"), guests.Field("host_id")))

		hostSQ := hostQ.GroupBy(hostQ.Field("host_id")).SubQuery()

		divSQ := hostSQ.Query(
			hostSQ.Field("id"),
			sqlchemy.SUB("vcpu_count", hostSQ.Field("cpu_count"), hostSQ.Field("cpu_reserved")),
			hostSQ.Field("guest_vcpu_count"),
		).SubQuery()

		sq := divSQ.Query(
			divSQ.Field("id").Label("host_id"),
			sqlchemy.DIV("cpu_commit_rate", divSQ.Field("guest_vcpu_count"), divSQ.Field("vcpu_count")),
		).SubQuery()

		q = q.LeftJoin(sq, sqlchemy.Equals(q.Field("id"), sq.Field("host_id")))

		db.OrderByFields(q, []string{query.OrderByCpuCommitRate}, []sqlchemy.IQueryField{sq.Field("cpu_commit_rate")})
	}

	if db.NeedOrderQuery([]string{query.OrderByStorage}) {
		hoststorages := HoststorageManager.Query().SubQuery()
		storages := StorageManager.Query().IsTrue("enabled").In("storage_type", api.HOST_STORAGE_LOCAL_TYPES).SubQuery()
		hoststoragesQ := hoststorages.Query(
			hoststorages.Field("host_id"),
			sqlchemy.SUM("storage_capacity", storages.Field("capacity")),
		)

		hoststoragesQ = hoststoragesQ.LeftJoin(storages, sqlchemy.Equals(hoststoragesQ.Field("storage_id"), storages.Field("id")))
		hoststoragesSQ := hoststoragesQ.GroupBy(hoststoragesQ.Field("host_id")).SubQuery()

		q = q.LeftJoin(hoststoragesSQ, sqlchemy.Equals(q.Field("id"), hoststoragesSQ.Field("host_id")))
		db.OrderByFields(q, []string{query.OrderByStorage}, []sqlchemy.IQueryField{hoststoragesSQ.Field("storage_capacity")})
	}

	if db.NeedOrderQuery([]string{query.OrderByStorageCommitRate}) {
		hoststorages := HoststorageManager.Query().SubQuery()
		disks := DiskManager.Query().Equals("status", api.DISK_READY).SubQuery()
		storages := StorageManager.Query().IsTrue("enabled").In("storage_type", api.HOST_STORAGE_LOCAL_TYPES).SubQuery()

		disksQ := disks.Query(
			disks.Field("storage_id"),
			sqlchemy.SUM("disk_size", disks.Field("disk_size")),
			storages.Field("capacity"),
			storages.Field("reserved"),
		).LeftJoin(storages, sqlchemy.Equals(disks.Field("storage_id"), storages.Field("id")))

		disksSQ := disksQ.GroupBy(disksQ.Field("storage_id")).SubQuery()

		divSQ := disksSQ.Query(
			disksSQ.Field("storage_id"),
			disksSQ.Field("disk_size"),
			sqlchemy.SUB("storage_capacity", disksSQ.Field("capacity"), disksSQ.Field("reserved")),
		).SubQuery()

		hoststoragesQ := hoststorages.Query(
			hoststorages.Field("host_id"),
			sqlchemy.SUM("storage_used", divSQ.Field("disk_size")),
			sqlchemy.SUM("storage_capacity", divSQ.Field("storage_capacity")),
		)
		hoststoragesQ = hoststoragesQ.LeftJoin(divSQ, sqlchemy.Equals(hoststoragesQ.Field("storage_id"), divSQ.Field("storage_id")))
		hoststoragesSQ1 := hoststoragesQ.GroupBy(hoststoragesQ.Field("host_id")).SubQuery()

		hoststoragesSQ := hoststoragesSQ1.Query(
			hoststoragesSQ1.Field("host_id"),
			sqlchemy.DIV("storage_commit_rate",
				hoststoragesSQ1.Field("storage_used"),
				hoststoragesSQ1.Field("storage_capacity"),
			),
		).SubQuery()

		q = q.LeftJoin(hoststoragesSQ, sqlchemy.Equals(q.Field("id"), hoststoragesSQ.Field("host_id")))
		db.OrderByFields(q, []string{query.OrderByStorageCommitRate}, []sqlchemy.IQueryField{hoststoragesSQ.Field("storage_commit_rate")})
	}

	if db.NeedOrderQuery([]string{query.OrderByMemCommitRate}) {
		guestsQ := GuestManager.Query()
		if options.Options.IgnoreNonrunningGuests {
			guestsQ = guestsQ.Equals("status", api.VM_RUNNING)
		}
		guests := guestsQ.SubQuery()
		hosts := HostManager.Query().SubQuery()
		hostQ := hosts.Query(
			hosts.Field("id"),
			hosts.Field("mem_size"),
			hosts.Field("mem_reserved"),
			sqlchemy.SUM("guest_vmem_size", guests.Field("vmem_size")),
		).LeftJoin(guests, sqlchemy.Equals(hosts.Field("id"), guests.Field("host_id")))

		hostSQ := hostQ.GroupBy(hostQ.Field("host_id")).SubQuery()

		divSQ := hostSQ.Query(
			hostSQ.Field("id"),
			sqlchemy.SUB("vmem_size", hostSQ.Field("mem_size"), hostSQ.Field("mem_reserved")),
			hostSQ.Field("guest_vmem_size"),
		).SubQuery()

		sq := divSQ.Query(
			divSQ.Field("id").Label("host_id"),
			sqlchemy.DIV("mem_commit_rate", divSQ.Field("guest_vmem_size"), divSQ.Field("vmem_size")),
		).SubQuery()

		q = q.LeftJoin(sq, sqlchemy.Equals(q.Field("id"), sq.Field("host_id")))

		db.OrderByFields(q, []string{query.OrderByMemCommitRate}, []sqlchemy.IQueryField{sq.Field("mem_commit_rate")})
	}

	return q, nil
}

func (manager *SHostManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SZoneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SHostManager) CustomizeFilterList(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*db.CustomizeListFilters, error) {
	filters := db.NewCustomizeListFilters()

	if query.Contains("cdrom_boot") {
		cdromBoot := jsonutils.QueryBoolean(query, "cdrom_boot", false)
		cdromBootF := func(obj jsonutils.JSONObject) (bool, error) {
			id, err := obj.GetString("id")
			if err != nil {
				return false, err
			}
			host := manager.FetchHostById(id)
			ipmiInfo, err := host.GetIpmiInfo()
			if err != nil {
				return false, err
			}
			if cdromBoot && ipmiInfo.CdromBoot {
				return true, nil
			}
			if !cdromBoot && !ipmiInfo.CdromBoot {
				return true, nil
			}
			return false, nil
		}

		filters.Append(cdromBootF)
	}

	return filters, nil
}

func (hh *SHost) IsArmHost() bool {
	return hh.CpuArchitecture == apis.OS_ARCH_AARCH64
}

func (hh *SHost) GetZone() (*SZone, error) {
	zone, err := ZoneManager.FetchById(hh.ZoneId)
	if err != nil {
		return nil, err
	}
	return zone.(*SZone), nil
}

func (hh *SHost) GetRegion() (*SCloudregion, error) {
	zone, err := hh.GetZone()
	if err != nil {
		return nil, err
	}
	return zone.GetRegion()
}

func (hh *SHost) GetCpuCount() int {
	if hh.CpuReserved > 0 && hh.CpuReserved < hh.CpuCount {
		return int(hh.CpuCount - hh.CpuReserved)
	} else {
		return int(hh.CpuCount)
	}
}

func (hh *SHost) GetMemSize() int {
	if hh.MemReserved > 0 && hh.MemReserved < hh.MemSize {
		return hh.MemSize - hh.MemReserved
	} else {
		return hh.MemSize
	}
}

func (hh *SHost) IsHugePage() bool {
	return hh.PageSizeKB > 4
}

func (hh *SHost) GetMemoryOvercommitBound() float32 {
	if hh.IsHugePage() {
		return 1.0
	}
	if hh.MemCmtbound > 0 {
		return hh.MemCmtbound
	}
	return options.Options.DefaultMemoryOvercommitBound
}

func (hh *SHost) GetVirtualMemorySize() float32 {
	return float32(hh.GetMemSize()) * hh.GetMemoryOvercommitBound()
}

func (hh *SHost) GetCPUOvercommitBound() float32 {
	if hh.CpuCmtbound > 0 {
		return hh.CpuCmtbound
	}
	return options.Options.DefaultCPUOvercommitBound
}

func (hh *SHost) GetVirtualCPUCount() float32 {
	return float32(hh.GetCpuCount()) * hh.GetCPUOvercommitBound()
}

func (hh *SHost) ValidateDeleteCondition(ctx context.Context, info api.HostDetails) error {
	if hh.IsBaremetal && hh.HostType != api.HOST_TYPE_BAREMETAL {
		return httperrors.NewInvalidStatusError("Host is a converted baremetal, should be unconverted before delete")
	}
	if hh.GetEnabled() {
		return httperrors.NewInvalidStatusError("Host is not disabled")
	}
	if info.Guests > 0 || info.BackupGuests > 0 {
		return httperrors.NewNotEmptyError("Not an empty host")
	}
	return hh.SEnabledStatusInfrasResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (hh *SHost) ValidatePurgeCondition(ctx context.Context) error {
	return hh.validateDeleteCondition(ctx, true)
}

func (hh *SHost) validateDeleteCondition(ctx context.Context, purge bool) error {
	if !purge && hh.IsBaremetal && hh.HostType != api.HOST_TYPE_BAREMETAL {
		return httperrors.NewInvalidStatusError("Host is a converted baremetal, should be unconverted before delete")
	}
	if hh.GetEnabled() {
		return httperrors.NewInvalidStatusError("Host is not disabled")
	}
	cnt, err := hh.GetGuestCount()
	if err != nil {
		return httperrors.NewInternalServerError("getGuestCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("Not an empty host")
	}
	cnt, err = hh.GetBackupGuestCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetBackupGuestCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("Not an empty host")
	}

	for _, hoststorage := range hh.GetHoststorages() {
		storage := hoststorage.GetStorage()
		if storage != nil && storage.IsLocal() {
			cnt, err := storage.GetDiskCount()
			if err != nil {
				return httperrors.NewInternalServerError("GetDiskCount fail %s", err)
			}
			if cnt > 0 {
				return httperrors.NewNotEmptyError("Local host storage is not empty???")
			}
		}

	}
	return hh.SEnabledStatusInfrasResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (hh *SHost) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("Host delete do nothing")
	return nil
}

func (hh *SHost) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if hh.IsBaremetal {
		return hh.StartDeleteBaremetalTask(ctx, userCred, "")
	}
	return hh.RealDelete(ctx, userCred)
}

func (hh *SHost) StartDeleteBaremetalTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalDeleteTask", hh, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	return task.ScheduleRun(nil)
}

func (hh *SHost) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return hh.purge(ctx, userCred)
}

func (hh *SHost) GetLoadbalancerBackends() ([]SLoadbalancerBackend, error) {
	q := LoadbalancerBackendManager.Query().Equals("backend_id", hh.Id)
	ret := []SLoadbalancerBackend{}
	return ret, db.FetchModelObjects(LoadbalancerBackendManager, q, &ret)
}

func (hh *SHost) GetHoststoragesQuery() *sqlchemy.SQuery {
	return HoststorageManager.Query().Equals("host_id", hh.Id)
}

func (hh *SHost) GetStorageCount() (int, error) {
	return hh.GetHoststoragesQuery().CountWithError()
}

func (hh *SHost) GetHoststorages() []SHoststorage {
	hoststorages := make([]SHoststorage, 0)
	q := hh.GetHoststoragesQuery()
	err := db.FetchModelObjects(HoststorageManager, q, &hoststorages)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		log.Errorf("GetHoststorages error %s", err)
		return nil
	}
	return hoststorages
}

func (hh *SHost) GetStorages() ([]SStorage, error) {
	sq := HoststorageManager.Query("storage_id").Equals("host_id", hh.Id).SubQuery()
	q := StorageManager.Query().In("id", sq)
	storages := []SStorage{}
	if err := db.FetchModelObjects(StorageManager, q, &storages); err != nil {
		return nil, err
	}
	return storages, nil
}

func (hh *SHost) GetHoststorageOfId(storageId string) *SHoststorage {
	hoststorage := SHoststorage{}
	hoststorage.SetModelManager(HoststorageManager, &hoststorage)
	err := hh.GetHoststoragesQuery().Equals("storage_id", storageId).First(&hoststorage)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("GetHoststorageOfId fail %s", err)
		}
		return nil
	}
	return &hoststorage
}

func (hh *SHost) GetHoststorageByExternalId(extId string) *SHoststorage {
	hoststorage := SHoststorage{}
	hoststorage.SetModelManager(HoststorageManager, &hoststorage)

	hoststorages := HoststorageManager.Query().SubQuery()
	storages := StorageManager.Query().SubQuery()
	q := hoststorages.Query()
	q = q.Join(storages, sqlchemy.Equals(hoststorages.Field("storage_id"), storages.Field("id")))
	q = q.Filter(sqlchemy.Equals(hoststorages.Field("host_id"), hh.Id))
	q = q.Filter(sqlchemy.Equals(storages.Field("external_id"), extId))

	err := q.First(&hoststorage)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("GetHoststorageByExternalId fail %s", err)
		}
		return nil
	}

	return &hoststorage
}

func (hh *SHost) GetStorageByFilePath(path string) *SStorage {
	hoststorages := hh.GetHoststorages()
	if hoststorages == nil {
		return nil
	}
	for i := 0; i < len(hoststorages); i += 1 {
		if len(hoststorages[i].MountPoint) > 0 && strings.HasPrefix(path, hoststorages[i].MountPoint) {
			return hoststorages[i].GetStorage()
		}
	}
	return nil
}

func (hh *SHost) GetBaremetalstorage() *SHoststorage {
	if !hh.IsBaremetal {
		return nil
	}
	hoststorages := HoststorageManager.Query().SubQuery()
	storages := StorageManager.Query().SubQuery()
	q := hoststorages.Query()
	q = q.Join(storages, sqlchemy.AND(sqlchemy.Equals(storages.Field("id"), hoststorages.Field("storage_id")),
		sqlchemy.IsFalse(storages.Field("deleted"))))
	q = q.Filter(sqlchemy.Equals(storages.Field("storage_type"), api.STORAGE_BAREMETAL))
	q = q.Filter(sqlchemy.Equals(hoststorages.Field("host_id"), hh.Id))
	cnt, err := q.CountWithError()
	if err != nil {
		return nil
	}
	if cnt == 1 {
		hs := SHoststorage{}
		hs.SetModelManager(HoststorageManager, &hs)
		err := q.First(&hs)
		if err != nil {
			log.Errorf("error %s", err)
			return nil
		}
		return &hs
	}
	// log.Errorf("Cannot find baremetalstorage??")
	return nil
}

func (hh *SHost) SaveCleanUpdates(doUpdate func() error) (map[string]sqlchemy.SUpdateDiff, error) {
	return hh.saveUpdates(doUpdate, true)
}

func (hh *SHost) SaveUpdates(doUpdate func() error) (map[string]sqlchemy.SUpdateDiff, error) {
	return hh.saveUpdates(doUpdate, false)
}

func (hh *SHost) saveUpdates(doUpdate func() error, doSchedClean bool) (map[string]sqlchemy.SUpdateDiff, error) {
	diff, err := db.Update(hh, doUpdate)
	if err != nil {
		return nil, err
	}
	if doSchedClean {
		hh.ClearSchedDescCache()
	}
	return diff, nil
}

func (hh *SHost) PerformUpdateStorage(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	bs := hh.GetBaremetalstorage()
	capacity, _ := data.Int("capacity")
	zoneId, _ := data.GetString("zone_id")
	storageCacheId, _ := data.GetString("storagecache_id")
	if bs == nil {
		// 1. create storage
		storage := SStorage{}
		storage.Name = fmt.Sprintf("storage%s", hh.GetName())
		storage.Capacity = capacity
		storage.StorageType = api.STORAGE_BAREMETAL
		storage.MediumType = hh.StorageType
		storage.Cmtbound = 1.0
		storage.Status = api.STORAGE_ONLINE
		storage.Enabled = tristate.True
		storage.ZoneId = zoneId
		storage.StoragecacheId = storageCacheId
		storage.DomainId = hh.DomainId
		storage.DomainSrc = string(apis.OWNER_SOURCE_LOCAL)
		err := StorageManager.TableSpec().Insert(ctx, &storage)
		if err != nil {
			return nil, fmt.Errorf("Create baremetal storage error: %v", err)
		}
		storage.SetModelManager(StorageManager, &storage)
		db.OpsLog.LogEvent(&storage, db.ACT_CREATE, storage.GetShortDesc(ctx), userCred)
		// 2. create host storage
		bmStorage := SHoststorage{}
		bmStorage.HostId = hh.Id
		bmStorage.StorageId = storage.Id
		bmStorage.RealCapacity = capacity
		bmStorage.MountPoint = ""
		err = HoststorageManager.TableSpec().Insert(ctx, &bmStorage)
		if err != nil {
			return nil, fmt.Errorf("Create baremetal hostStorage error: %v", err)
		}
		bmStorage.SetModelManager(HoststorageManager, &bmStorage)
		db.OpsLog.LogAttachEvent(ctx, hh, &storage, userCred, bmStorage.GetShortDesc(ctx))
		bmStorage.syncLocalStorageShare(ctx, userCred)
		return nil, nil
	}
	storage := bs.GetStorage()
	//if capacity != int64(storage.Capacity)  {
	diff, err := db.Update(storage, func() error {
		storage.Capacity = capacity
		storage.StoragecacheId = storageCacheId
		storage.Enabled = tristate.True
		storage.DomainId = hh.DomainId
		storage.DomainSrc = string(apis.OWNER_SOURCE_LOCAL)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Update baremetal storage error: %v", err)
	}
	db.OpsLog.LogEvent(storage, db.ACT_UPDATE, diff, userCred)
	bs.syncLocalStorageShare(ctx, userCred)
	//}
	return nil, nil
}

func (hh *SHost) GetFetchUrl(disableHttps bool) string {
	managerUrl, err := url.Parse(hh.ManagerUri)
	if err != nil {
		log.Errorf("GetFetchUrl fail to parse url: %s", err)
	}

	if disableHttps {
		managerUrl.Scheme = "http"
	}

	portStr := managerUrl.Port()
	var port int
	if len(portStr) > 0 {
		port, _ = strconv.Atoi(portStr)
	} else {
		if managerUrl.Scheme == "https" {
			port = 443
		} else if managerUrl.Scheme == "http" {
			port = 80
		}
	}
	return fmt.Sprintf("%s://%s:%d", managerUrl.Scheme, strings.Split(managerUrl.Host, ":")[0], port+40000)
}

func (hh *SHost) GetAttachedEnabledHostStorages(storageType []string) []SStorage {
	return hh._getAttachedStorages(tristate.False, tristate.True, storageType)
}

func (hh *SHost) _getAttachedStorages(isBaremetal tristate.TriState, enabled tristate.TriState, storageType []string) []SStorage {
	storages := StorageManager.Query().SubQuery()
	hoststorages := HoststorageManager.Query().SubQuery()
	q := storages.Query()
	q = q.Join(hoststorages, sqlchemy.Equals(storages.Field("id"), hoststorages.Field("storage_id")))
	if enabled.IsTrue() {
		q = q.IsTrue("enabled")
	} else if enabled.IsFalse() {
		q = q.IsFalse("enabled")
	}
	if isBaremetal.IsTrue() {
		q = q.Equals("storage_type", api.STORAGE_BAREMETAL)
	} else if isBaremetal.IsFalse() {
		q = q.NotEquals("storage_type", api.STORAGE_BAREMETAL)
	}
	if len(storageType) > 0 {
		q = q.In("storage_type", storageType)
	}
	q = q.Filter(sqlchemy.Equals(hoststorages.Field("host_id"), hh.Id))
	ret := make([]SStorage, 0)
	err := db.FetchModelObjects(StorageManager, q, &ret)
	if err != nil {
		log.Errorf("GetAttachedStorages fail %s", err)
		return nil
	}
	return ret
}

func (hh *SHost) SyncAttachedStorageStatus() {
	storages := hh.GetAttachedEnabledHostStorages(nil)
	if storages != nil {
		for _, storage := range storages {
			storage.SyncStatusWithHosts()
		}
		hh.ClearSchedDescCache()
	}
}

func (hh *SHostManager) IsNewNameUnique(name string, userCred mcclient.TokenCredential, kwargs *jsonutils.JSONDict) (bool, error) {
	q := hh.Query().Equals("name", name)
	if kwargs != nil && kwargs.Contains("zone_id") {
		zoneId, _ := kwargs.GetString("zone_id")
		q.Equals("zone_id", zoneId)
	}
	cnt, err := q.CountWithError()
	if err != nil {
		return false, err
	}
	return cnt == 0, nil
}

func (hh *SHostManager) GetPropertyK8sMasterNodeIps(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	cli, err := tokens.GetCoreClient()
	if err != nil {
		return nil, errors.Wrap(err, "get k8s client")
	}
	nodes, err := cli.Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/master",
	})
	if err != nil {
		return nil, errors.Wrap(err, "list master nodes")
	}
	ips := make([]string, 0)
	for i := range nodes.Items {
		for j := range nodes.Items[i].Status.Addresses {
			if nodes.Items[i].Status.Addresses[j].Type == v1.NodeInternalIP {
				ips = append(ips, nodes.Items[i].Status.Addresses[j].Address)
			}
		}
	}
	log.Infof("k8s master nodes ips %v", ips)
	res := jsonutils.NewDict()
	res.Set("ips", jsonutils.Marshal(ips))
	return res, nil
}

func (hh *SHostManager) GetPropertyBmStartRegisterScript(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	regionUri, err := auth.GetPublicServiceURL("compute_v2", options.Options.Region, "")
	if err != nil {
		return nil, err
	}
	var script string
	script += fmt.Sprintf("curl -k -fsSL -H 'X-Auth-Token: %s' %s/misc/bm-prepare-script", userCred.GetTokenString(), regionUri)
	res := jsonutils.NewDict()
	res.Add(jsonutils.NewString(script), "script")
	return res, nil
}

func (hh *SHostManager) GetPropertyNodeCount(ctx context.Context, userCred mcclient.TokenCredential, query api.HostListInput) (jsonutils.JSONObject, error) {
	hosts := hh.Query().SubQuery()
	q := hosts.Query(hosts.Field("host_type"), sqlchemy.SUM("node_count_total", hosts.Field("node_count")))
	return hh.getCount(ctx, userCred, q, query)
}

func (hh *SHostManager) getCount(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery, query api.HostListInput) (jsonutils.JSONObject, error) {
	filterAny := false
	if query.FilterAny != nil {
		filterAny = *query.FilterAny
	}
	q, err := db.ApplyListItemsGeneralFilters(hh, q, userCred, query.Filter, filterAny)
	if err != nil {
		return nil, err
	}

	q, err = hh.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	rows, err := q.GroupBy("host_type").Rows()
	if err != nil {
		return nil, err
	}

	ret := jsonutils.NewDict()
	defer rows.Close()
	for rows.Next() {
		var hostType string
		var count int64
		err = rows.Scan(&hostType, &count)
		if err != nil {
			log.Errorf("getCount scan err: %v", err)
			return ret, nil
		}

		ret.Add(jsonutils.NewInt(count), hostType)
	}

	return ret, nil
}

func (hh *SHostManager) GetPropertyHostTypeCount(ctx context.Context, userCred mcclient.TokenCredential, query api.HostListInput) (jsonutils.JSONObject, error) {
	hosts := hh.Query().SubQuery()
	// select host_type, (case host_type when 'huaweicloudstack' then count(DISTINCT external_id) else count(id) end) as count from hosts_tbl group by host_type;
	cs := sqlchemy.NewCase()
	hcso := sqlchemy.Equals(hosts.Field("host_type"), api.HOST_TYPE_HCSO)
	cs.When(hcso, sqlchemy.COUNT("", sqlchemy.DISTINCT("", hosts.Field("external_id"))))
	cs.Else(sqlchemy.COUNT("", hosts.Field("id")))
	q := hosts.Query(hosts.Field("host_type"), sqlchemy.NewFunction(cs, "count"))
	return hh.getCount(ctx, userCred, q, query)
}

func (hh *SHostManager) ClearAllSchedDescCache() error {
	return hh.ClearSchedDescSessionCache("", "")
}

func (hh *SHostManager) ClearSchedDescCache(hostId string) error {
	return hh.ClearSchedDescSessionCache(hostId, "")
}

func (hh *SHostManager) ClearSchedDescSessionCache(hostId, sessionId string) error {
	s := auth.GetAdminSession(context.Background(), options.Options.Region)
	return scheduler.SchedManager.CleanCache(s, hostId, sessionId, false)
}

func (hh *SHost) ClearSchedDescCache() error {
	return hh.ClearSchedDescSessionCache("")
}

func (hh *SHost) ClearSchedDescSessionCache(sessionId string) error {
	return HostManager.ClearSchedDescSessionCache(hh.Id, sessionId)
}

// sync clear sched desc on scheduler side
func (hh *SHostManager) SyncClearSchedDescSessionCache(hostId, sessionId string) error {
	s := auth.GetAdminSession(context.Background(), options.Options.Region)
	return scheduler.SchedManager.CleanCache(s, hostId, sessionId, true)
}

func (hh *SHost) SyncCleanSchedDescCache() error {
	return hh.SyncClearSchedDescSessionCache("")
}

func (hh *SHost) SyncClearSchedDescSessionCache(sessionId string) error {
	return HostManager.SyncClearSchedDescSessionCache(hh.Id, sessionId)
}

func (hh *SHost) GetDetailsSpec(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return GetModelSpec(HostManager, hh)
}

func (man *SHostManager) GetSpecShouldCheckStatus(query *jsonutils.JSONDict) (bool, error) {
	statusCheck := true
	if query.Contains("is_empty") {
		isEmpty, err := query.Bool("is_empty")
		if err != nil {
			return statusCheck, err
		}
		if !isEmpty {
			statusCheck = false
		}
	}
	return statusCheck, nil
}

func (hh *SHost) GetSpec(statusCheck bool) *jsonutils.JSONDict {
	if statusCheck {
		if !hh.GetEnabled() {
			return nil
		}
		if !utils.IsInStringArray(hh.Status, []string{api.BAREMETAL_RUNNING, api.BAREMETAL_READY}) || hh.GetBaremetalServer() != nil || hh.IsMaintenance {
			return nil
		}
		if hh.MemSize == 0 || hh.CpuCount == 0 {
			return nil
		}
		if hh.ResourceType == api.HostResourceTypePrepaidRecycle {
			cnt, err := hh.GetGuestCount()
			if err != nil {
				return nil
			}
			if cnt > 0 {
				// occupied
				return nil
			}
		}

		if len(hh.ManagerId) > 0 {
			providerObj, _ := CloudproviderManager.FetchById(hh.ManagerId)
			if providerObj == nil {
				return nil
			}
			provider := providerObj.(*SCloudprovider)
			if !provider.IsAvailable() {
				return nil
			}
		}
	}
	spec := hh.GetHardwareSpecification()
	specInfo := new(api.HostSpec)
	if err := spec.Unmarshal(specInfo); err != nil {
		return spec
	}
	nifs := hh.GetAllNetInterfaces()
	var nicCount int
	for _, nif := range nifs {
		if nif.NicType != api.NIC_TYPE_IPMI {
			nicCount++
		}
	}
	specInfo.NicCount = nicCount

	var manufacture string
	var model string
	if hh.SysInfo != nil {
		manufacture, _ = hh.SysInfo.GetString("manufacture")
		model, _ = hh.SysInfo.GetString("model")
	}
	if manufacture == "" {
		manufacture = "Unknown"
	}
	if model == "" {
		model = "Unknown"
	}
	specInfo.Manufacture = strings.ReplaceAll(manufacture, " ", "_")
	specInfo.Model = strings.ReplaceAll(model, " ", "_")
	devices := IsolatedDeviceManager.FindByHost(hh.Id)
	if len(devices) > 0 {
		specInfo.IsolatedDevices = make([]api.IsolatedDeviceSpec, len(devices))
		for i := 0; i < len(devices); i++ {
			specInfo.IsolatedDevices[i].DevType = devices[i].DevType
			specInfo.IsolatedDevices[i].Model = devices[i].Model
			specInfo.IsolatedDevices[i].PciId = devices[i].VendorDeviceId
			specInfo.IsolatedDevices[i].Vendor = devices[i].getVendor()
		}
	}

	return specInfo.JSON(specInfo)
}

func (manager *SHostManager) GetSpecIdent(input *jsonutils.JSONDict) []string {
	spec := new(api.HostSpec)
	input.Unmarshal(spec)
	specKeys := []string{
		fmt.Sprintf("cpu:%d", spec.Cpu),
		fmt.Sprintf("mem:%dM", spec.Mem),
		fmt.Sprintf("nic:%d", spec.NicCount),
		fmt.Sprintf("manufacture:%s", spec.Manufacture),
		fmt.Sprintf("model:%s", spec.Model),
	}
	diskDriverSpec := spec.Disk
	if diskDriverSpec != nil {
		for driver, driverSpec := range diskDriverSpec {
			specKeys = append(specKeys, parseDiskDriverSpec(driver, driverSpec)...)
		}
	}
	sort.Strings(specKeys)
	return specKeys
}

func parseDiskDriverSpec(driver string, adapterSpecs api.DiskAdapterSpec) []string {
	ret := make([]string, 0)
	for adapterKey, adapterSpec := range adapterSpecs {
		for _, diskSpec := range adapterSpec {
			sizeGB, _ := utils.GetSizeGB(fmt.Sprintf("%d", diskSpec.Size), "M")
			diskKey := fmt.Sprintf("disk:%s_%s_%s_%dGx%d", driver, adapterKey, diskSpec.Type, sizeGB, diskSpec.Count)
			ret = append(ret, diskKey)
		}
	}
	return ret
}

func ConvertStorageInfo2BaremetalStorages(storageInfo jsonutils.JSONObject) []*baremetal.BaremetalStorage {
	if storageInfo == nil {
		return nil
	}
	storages := []baremetal.BaremetalStorage{}
	err := storageInfo.Unmarshal(&storages)
	if err != nil {
		log.Errorf("Unmarshal to baremetal storage error: %v", err)
		return nil
	}
	ret := make([]*baremetal.BaremetalStorage, len(storages))
	for i := range storages {
		ret[i] = &storages[i]
	}
	return ret
}

func GetDiskSpecV2(storageInfo jsonutils.JSONObject) api.DiskDriverSpec {
	refStorages := ConvertStorageInfo2BaremetalStorages(storageInfo)
	if refStorages == nil {
		return nil
	}
	return baremetal.GetDiskSpecV2(refStorages)
}

func (hh *SHost) GetHardwareSpecification() *jsonutils.JSONDict {
	spec := &api.HostSpec{
		Cpu: int(hh.CpuCount),
		Mem: hh.MemSize,
	}
	if hh.StorageInfo != nil {
		spec.Disk = GetDiskSpecV2(hh.StorageInfo)
		spec.Driver = hh.StorageDriver
	}
	ret := spec.JSON(spec)
	if hh.StorageInfo != nil {
		ret.Set("storage_info", hh.StorageInfo)
	}
	return ret
}

type SStorageCapacity struct {
	Capacity   int64 `json:"capacity,omitzero"`
	Used       int64 `json:"used_capacity,omitzero"`
	ActualUsed int64 `json:"real_time_used_capacity,omitzero"`
	Wasted     int64 `json:"waste_capacity,omitzero"`
	VCapacity  int64 `json:"virtual_capacity,omitzero"`
}

func (cap *SStorageCapacity) GetFree() int64 {
	return cap.VCapacity - cap.Used - cap.Wasted
}

func (cap *SStorageCapacity) GetCommitRate() float64 {
	if cap.Capacity > 0 {
		return float64(int(float64(cap.Used)*100.0/float64(cap.Capacity)+0.5)) / 100.0
	} else {
		return 0.0
	}
}

func (cap *SStorageCapacity) Add(cap2 SStorageCapacity) {
	cap.Capacity += cap2.Capacity
	cap.Used += cap2.Used
	cap.Wasted += cap2.Wasted
	cap.VCapacity += cap2.VCapacity
	cap.ActualUsed += cap2.ActualUsed
}

func (cap *SStorageCapacity) toCapacityInfo() api.SStorageCapacityInfo {
	info := api.SStorageCapacityInfo{}
	info.UsedCapacity = cap.Used
	info.WasteCapacity = cap.Wasted
	info.VirtualCapacity = cap.VCapacity
	info.CommitRate = cap.GetCommitRate()
	info.FreeCapacity = cap.GetFree()
	return info
}

func (hh *SHost) GetAttachedLocalStorageCapacity() SStorageCapacity {
	ret := SStorageCapacity{}
	storages := hh.GetAttachedEnabledHostStorages(api.HOST_STORAGE_LOCAL_TYPES)
	for _, s := range storages {
		ret.Add(s.getStorageCapacity())
	}
	return ret
}

func (hh *SHost) GetAttachedLocalStorages() []SStorage {
	return hh.GetAttachedEnabledHostStorages(api.HOST_STORAGE_LOCAL_TYPES)
}

func _getLeastUsedStorage(storages []SStorage, backends []string) *SStorage {
	var best *SStorage
	var bestCap int64
	for i := 0; i < len(storages); i++ {
		s := storages[i]
		if len(backends) > 0 {
			in, _ := utils.InStringArray(s.StorageType, backends)
			if !in {
				continue
			}
		}
		capa := s.GetFreeCapacity()
		if best == nil || bestCap < capa {
			bestCap = capa
			best = &s
		}
	}
	return best
}

func ChooseLeastUsedStorage(storages []SStorage, backend string) *SStorage {
	var backends []string
	if backend == api.STORAGE_LOCAL {
		backends = []string{api.STORAGE_NAS, api.STORAGE_LOCAL}
	} else if len(backend) > 0 {
		backends = []string{backend}
	} else {
		backends = []string{}
	}
	return _getLeastUsedStorage(storages, backends)
}

func (hh *SHost) GetLeastUsedStorage(backend string) *SStorage {
	storages := hh.GetAttachedEnabledHostStorages(nil)
	if storages != nil {
		return ChooseLeastUsedStorage(storages, backend)
	}
	return nil
}

/*func (hh *SHost) GetWiresQuery() *sqlchemy.SQuery {
	return Manager.Query().Equals("host_id", hh.Id)
}

func (hh *SHost) GetWireCount() (int, error) {
	return hh.GetWiresQuery().CountWithError()
}

func (hh *SHost) GetHostwires() []SHostwire {
	hw := make([]SHostwire, 0)
	q := hh.GetWiresQuery()
	err := db.FetchModelObjects(HostwireManager, q, &hw)
	if err != nil {
		log.Errorf("GetWires error %s", err)
		return nil
	}
	return hw
}*/

func (hh *SHost) getAttachedWires() []SWire {
	wires := WireManager.Query().SubQuery()
	netifs := NetInterfaceManager.Query().SubQuery()
	q := wires.Query()
	q = q.Join(netifs, sqlchemy.Equals(netifs.Field("wire_id"), wires.Field("id")))
	q = q.Filter(sqlchemy.Equals(netifs.Field("baremetal_id"), hh.Id))
	q = q.Distinct()
	ret := make([]SWire, 0)
	err := db.FetchModelObjects(WireManager, q, &ret)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return ret
}

func (hh *SHostManager) GetEnabledKvmHost() (*SHost, error) {
	hostq := HostManager.Query().IsTrue("enabled").Equals("host_status", api.HOST_ONLINE).In("host_type", []string{api.HOST_TYPE_HYPERVISOR, api.HOST_TYPE_KVM})
	host := SHost{}
	err := hostq.First(&host)
	if err != nil {
		return nil, err
	}
	host.SetModelManager(HostManager, &host)
	return &host, nil
}

func (hh *SHost) GetMasterWire() *SWire {
	wires := WireManager.Query().SubQuery()
	netifs := NetInterfaceManager.Query().SubQuery()
	q := wires.Query()
	q = q.Join(netifs, sqlchemy.Equals(netifs.Field("wire_id"), wires.Field("id")))
	q = q.Filter(sqlchemy.Equals(netifs.Field("baremetal_id"), hh.Id))
	q = q.Filter(sqlchemy.Equals(netifs.Field("nic_type"), api.NIC_TYPE_ADMIN))
	wire := SWire{}
	wire.SetModelManager(WireManager, &wire)

	err := q.First(&wire)
	if err != nil {
		log.Errorf("GetMasterWire fail %s", err)
		return nil
	}
	return &wire
}

/*func (hh *SHost) getHostwires() ([]SHostwire, error) {
	hostwires := make([]SHostwire, 0)
	q := hh.GetWiresQuery()
	err := db.FetchModelObjects(HostwireManager, q, &hostwires)
	if err != nil {
		return nil, err
	}
	return hostwires, nil
}

func (hh *SHost) getHostwiresOfId(wireId string) []SHostwire {
	hostwires := make([]SHostwire, 0)

	q := hh.GetWiresQuery().Equals("wire_id", wireId)
	err := db.FetchModelObjects(HostwireManager, q, &hostwires)
	if err != nil {
		log.Errorf("getHostwiresOfId fail %s", err)
		return nil
	}
	return hostwires
}

func (hh *SHost) getHostwireOfIdAndMac(wireId string, mac string) *SHostwire {
	hostwire := SHostwire{}
	hostwire.SetModelManager(HostwireManager, &hostwire)

	q := hh.GetWiresQuery().Equals("wire_id", wireId)
	q = q.Equals("mac_addr", mac)
	err := q.First(&hostwire)
	if err != nil {
		log.Errorf("getHostwireOfIdAndMac fail %s", err)
		return nil
	}
	return &hostwire
}*/

func (hh *SHost) GetGuestsQuery() *sqlchemy.SQuery {
	return GuestManager.Query().Equals("host_id", hh.Id)
}

func (hh *SHost) GetGuests() ([]SGuest, error) {
	q := hh.GetGuestsQuery()
	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return guests, nil
}

func (hh *SHost) GetKvmGuests() []SGuest {
	q := GuestManager.Query().Equals("host_id", hh.Id).Equals("hypervisor", api.HYPERVISOR_KVM)
	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("GetGuests %s", err)
		return nil
	}
	return guests
}

func (hh *SHost) GetGuestsMasterOnThisHost() []SGuest {
	q := hh.GetGuestsQuery().IsNotEmpty("backup_host_id")
	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("GetGuests %s", err)
		return nil
	}
	return guests
}

func (hh *SHost) GetGuestsBackupOnThisHost() []SGuest {
	q := GuestManager.Query().Equals("backup_host_id", hh.Id)
	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("GetGuests %s", err)
		return nil
	}
	return guests
}

func (hh *SHost) GetBackupGuestCount() (int, error) {
	q := GuestManager.Query().Equals("backup_host_id", hh.Id)
	return q.CountWithError()
}

func (hh *SHost) GetGuestCount() (int, error) {
	q := hh.GetGuestsQuery()
	return q.CountWithError()
}

func (hh *SHost) GetContainerCount(status []string) (int, error) {
	q := hh.GetGuestsQuery()
	q = q.Filter(sqlchemy.Equals(q.Field("hypervisor"), api.HYPERVISOR_POD))
	if len(status) > 0 {
		q = q.In("status", status)
	}
	return q.CountWithError()
}

func (hh *SHost) GetNonsystemGuestCount() (int, error) {
	q := hh.GetGuestsQuery()
	q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("is_system")), sqlchemy.IsFalse(q.Field("is_system"))))
	return q.CountWithError()
}

func (hh *SHost) GetRunningGuestCount() (int, error) {
	q := hh.GetGuestsQuery()
	q = q.In("status", api.VM_RUNNING_STATUS)
	return q.CountWithError()
}

func (hh *SHost) GetNotReadyGuestsStat() (*SHostGuestResourceUsage, error) {
	guests := GuestManager.Query().SubQuery()
	q := guests.Query(sqlchemy.COUNT("guest_count"),
		sqlchemy.SUM("guest_vcpu_count", guests.Field("vcpu_count")),
		sqlchemy.SUM("guest_vmem_size", guests.Field("vmem_size")))
	cond := sqlchemy.OR(sqlchemy.Equals(q.Field("host_id"), hh.Id),
		sqlchemy.Equals(q.Field("backup_host_id"), hh.Id))
	q = q.Filter(cond)
	q = q.NotEquals("status", api.VM_READY)
	stat := SHostGuestResourceUsage{}
	err := q.First(&stat)
	if err != nil {
		return nil, err
	}
	return &stat, nil
}

func (hh *SHost) GetRunningGuestResourceUsage() *SHostGuestResourceUsage {
	return hh.getGuestsResource(api.VM_RUNNING)
}

func (hh *SHost) GetBaremetalnetworksQuery() *sqlchemy.SQuery {
	return HostnetworkManager.Query().Equals("baremetal_id", hh.Id)
}

func (hh *SHost) GetBaremetalnetworks() []SHostnetwork {
	q := hh.GetBaremetalnetworksQuery()
	hns := make([]SHostnetwork, 0)
	err := db.FetchModelObjects(HostnetworkManager, q, &hns)
	if err != nil {
		log.Errorf("GetBaremetalnetworks error: %s", err)
	}
	return hns
}

func (hh *SHost) GetAttach2Network(netId string) *SHostnetwork {
	q := hh.GetBaremetalnetworksQuery()
	netifs := NetInterfaceManager.Query().Equals("baremetal_id", hh.Id)
	netifs = netifs.Filter(sqlchemy.OR(
		sqlchemy.IsNullOrEmpty(netifs.Field("nic_type")),
		sqlchemy.NotEquals(netifs.Field("nic_type"), api.NIC_TYPE_IPMI),
	))
	netifsSub := netifs.SubQuery()
	q = q.Join(netifsSub, sqlchemy.AND(
		sqlchemy.Equals(q.Field("mac_addr"), netifsSub.Field("mac")),
		sqlchemy.Equals(q.Field("vlan_id"), netifsSub.Field("vlan_id")),
	))
	q = q.Equals("network_id", netId)
	hn := SHostnetwork{}
	hn.SetModelManager(HostnetworkManager, &hn)

	err := q.First(&hn)
	if err != nil {
		log.Errorf("GetAttach2Network fail %s", err)
		return nil
	}
	return &hn
}

func (h *SHost) getNetInterfacesInternal(wireId string, nicTypes []compute.TNicType) []SNetInterface {
	q := NetInterfaceManager.Query().Equals("baremetal_id", h.Id)
	if len(wireId) > 0 {
		q = q.Equals("wire_id", wireId)
	}
	if len(nicTypes) > 0 {
		q = q.In("nic_type", nicTypes)
	}
	q = q.Asc("index")
	q = q.Asc("vlan_id")
	q = q.Asc("nic_type")
	netifs := make([]SNetInterface, 0)
	err := db.FetchModelObjects(NetInterfaceManager, q, &netifs)
	if err != nil {
		log.Errorf("GetNetInterfaces fail %s", err)
		return nil
	}
	return netifs
}

func (hh *SHost) GetAllNetInterfaces() []SNetInterface {
	return hh.getNetInterfacesInternal("", nil)
}

func (hh *SHost) GetHostNetInterfaces() []SNetInterface {
	return hh.getNetInterfacesInternal("", api.HOST_NIC_TYPES)
}

func (hh *SHost) GetAdminNetInterfaces() []SNetInterface {
	return hh.getNetInterfacesInternal("", []compute.TNicType{api.NIC_TYPE_ADMIN})
}

func (hh *SHost) GetNetInterface(mac string, vlanId int) *SNetInterface {
	netif, _ := NetInterfaceManager.FetchByMacVlan(mac, vlanId)
	if netif != nil && netif.BaremetalId == hh.Id {
		return netif
	}
	return nil
}

func (hh *SHost) DeleteBaremetalnetwork(ctx context.Context, userCred mcclient.TokenCredential, bn *SHostnetwork, reserve bool) {
	net := bn.GetNetwork()
	bn.Delete(ctx, userCred)
	db.OpsLog.LogDetachEvent(ctx, hh, net, userCred, nil)
	if reserve && net != nil && len(bn.IpAddr) > 0 && regutils.MatchIP4Addr(bn.IpAddr) {
		ReservedipManager.ReserveIP(ctx, userCred, net, bn.IpAddr, "Delete baremetalnetwork to reserve", api.AddressTypeIPv4)
	}
}

func (hh *SHost) GetHostDriver() IHostDriver {
	if !utils.IsInStringArray(hh.HostType, api.HOST_TYPES) {
		log.Fatalf("Unsupported host type %s", hh.HostType)
	}
	return GetHostDriver(hh.HostType)
}

func (manager *SHostManager) getHostsByZoneProvider(zone *SZone, region *SCloudregion, provider *SCloudprovider) ([]SHost, error) {
	hosts := make([]SHost, 0)
	q := manager.Query()
	if zone != nil {
		q = q.Equals("zone_id", zone.Id)
	}
	if region != nil {
		zoneQ := ZoneManager.Query().Equals("cloudregion_id", region.Id).SubQuery()
		q = q.Join(zoneQ, sqlchemy.Equals(q.Field("zone_id"), zoneQ.Field("id")))
	}
	if provider != nil {
		q = q.Equals("manager_id", provider.Id)
	}
	// exclude prepaid_recycle fake hosts
	q = q.NotEquals("resource_type", api.HostResourceTypePrepaidRecycle)

	err := db.FetchModelObjects(manager, q, &hosts)
	if err != nil {
		log.Errorf("%s", err)
		return nil, err
	}
	return hosts, nil
}

func (manager *SHostManager) SyncHosts(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, zone *SZone, region *SCloudregion, hosts []cloudprovider.ICloudHost, xor bool) ([]SHost, []cloudprovider.ICloudHost, compare.SyncResult) {
	key := provider.Id
	if zone != nil {
		key = fmt.Sprintf("%s-%s", zone.Id, provider.Id)
	}
	lockman.LockRawObject(ctx, manager.Keyword(), key)
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), key)

	syncResult := compare.SyncResult{}

	dbHosts, err := manager.getHostsByZoneProvider(zone, region, provider)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	localHosts := make([]SHost, 0)
	remoteHosts := make([]cloudprovider.ICloudHost, 0)

	removed := make([]SHost, 0)
	commondb := make([]SHost, 0)
	commonext := make([]cloudprovider.ICloudHost, 0)
	added := make([]cloudprovider.ICloudHost, 0)

	err = compare.CompareSets(dbHosts, hosts, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		if removed[i].IsPrepaidRecycleResource() {
			continue
		}
		err = removed[i].syncRemoveCloudHost(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		if !xor {
			err = commondb[i].syncWithCloudHost(ctx, userCred, commonext[i], provider)
			if err != nil {
				syncResult.UpdateError(err)
			}
		}
		localHosts = append(localHosts, commondb[i])
		remoteHosts = append(remoteHosts, commonext[i])
		syncResult.Update()
	}
	for i := 0; i < len(added); i += 1 {
		new, err := manager.NewFromCloudHost(ctx, userCred, added[i], provider, zone)
		if err != nil {
			syncResult.AddError(err)
		} else {
			localHosts = append(localHosts, *new)
			remoteHosts = append(remoteHosts, added[i])
			syncResult.Add()
		}
	}

	return localHosts, remoteHosts, syncResult
}

func (hh *SHost) syncRemoveCloudHost(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, hh)
	defer lockman.ReleaseObject(ctx, hh)

	err := hh.ValidatePurgeCondition(ctx)
	if err != nil {
		err = hh.purge(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "purge")
		}
	} else {
		err = hh.RealDelete(ctx, userCred)
	}
	return err
}

func (hh *SHost) syncWithCloudHost(ctx context.Context, userCred mcclient.TokenCredential, extHost cloudprovider.ICloudHost, provider *SCloudprovider) error {
	diff, err := db.UpdateWithLock(ctx, hh, func() error {
		// hh.Name = extHost.GetName()

		hh.Status = extHost.GetStatus()
		hh.HostStatus = extHost.GetHostStatus()
		hh.AccessIp = extHost.GetAccessIp()
		hh.AccessMac = extHost.GetAccessMac()
		hh.SN = extHost.GetSN()
		hh.SysInfo = extHost.GetSysInfo()
		hh.CpuCount = extHost.GetCpuCount()
		hh.NodeCount = extHost.GetNodeCount()
		cpuDesc := extHost.GetCpuDesc()
		if len(cpuDesc) > 128 {
			cpuDesc = cpuDesc[:128]
		}
		hh.CpuDesc = cpuDesc
		hh.CpuMhz = extHost.GetCpuMhz()
		hh.MemSize = extHost.GetMemSizeMB()
		hh.StorageSize = extHost.GetStorageSizeMB()
		hh.StorageType = extHost.GetStorageType()
		hh.HostType = extHost.GetHostType()
		hh.OvnVersion = extHost.GetOvnVersion()

		if cpuCmt := extHost.GetCpuCmtbound(); cpuCmt > 0 {
			hh.CpuCmtbound = cpuCmt
		}

		if memCmt := extHost.GetMemCmtbound(); memCmt > 0 {
			hh.MemCmtbound = memCmt
		}

		if arch := extHost.GetCpuArchitecture(); len(arch) > 0 {
			hh.CpuArchitecture = arch
		}

		if reservedMem := extHost.GetReservedMemoryMb(); reservedMem > 0 {
			hh.MemReserved = reservedMem
		}

		hh.IsEmulated = extHost.IsEmulated()
		hh.SetEnabled(extHost.GetEnabled())

		hh.IsMaintenance = extHost.GetIsMaintenance()
		hh.Version = extHost.GetVersion()

		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "syncWithCloudZone")
	}

	db.OpsLog.LogSyncUpdate(hh, diff, userCred)

	if provider != nil {
		SyncCloudDomain(userCred, hh, provider.GetOwnerId())
		hh.SyncShareState(ctx, userCred, provider.getAccountShareInfo())
	}
	if account := hh.GetCloudaccount(); account != nil {
		syncMetadata(ctx, userCred, hh, extHost, account.ReadOnly)
	}

	if err := hh.syncSchedtags(ctx, userCred, extHost); err != nil {
		log.Errorf("syncSchedtags fail:  %v", err)
		return err
	}

	if len(diff) > 0 {
		if err := HostManager.ClearSchedDescCache(hh.Id); err != nil {
			log.Errorf("ClearSchedDescCache for host %s error %v", hh.Name, err)
		}
	}

	return nil
}

func (hh *SHost) syncWithCloudPrepaidVM(extVM cloudprovider.ICloudVM, host *SHost) error {
	_, err := hh.SaveUpdates(func() error {

		hh.CpuCount = extVM.GetVcpuCount()
		hh.MemSize = extVM.GetVmemSizeMB()

		hh.BillingType = extVM.GetBillingType()
		hh.ExpiredAt = extVM.GetExpiredAt()

		hh.ExternalId = host.ExternalId

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudZone error %s", err)
	}

	if err := HostManager.ClearSchedDescCache(hh.Id); err != nil {
		log.Errorf("ClearSchedDescCache for host %s error %v", hh.Name, err)
	}

	return err
}

var (
	METADATA_EXT_SCHEDTAG_KEY = "ext:schedtag"
)

func (s *SHost) getAllSchedtagsWithExtSchedtagKey(ctx context.Context, userCred mcclient.TokenCredential) (map[string]*SSchedtag, error) {
	q := SchedtagManager.Query().Equals("resource_type", HostManager.KeywordPlural())
	sts := make([]SSchedtag, 0, 5)
	err := db.FetchModelObjects(SchedtagManager, q, &sts)
	if err != nil {
		return nil, err
	}
	stMap := make(map[string]*SSchedtag)
	for i := range sts {
		extTagName := sts[i].GetMetadata(ctx, METADATA_EXT_SCHEDTAG_KEY, userCred)
		if len(extTagName) == 0 {
			continue
		}
		stMap[extTagName] = &sts[i]
	}
	return stMap, nil
}

func (s *SHost) syncSchedtags(ctx context.Context, userCred mcclient.TokenCredential, extHost cloudprovider.ICloudHost) error {
	stq := SchedtagManager.Query()
	subq := HostschedtagManager.Query("schedtag_id").Equals("host_id", s.Id).SubQuery()
	stq = stq.Join(subq, sqlchemy.Equals(stq.Field("id"), subq.Field("schedtag_id")))
	schedtags := make([]SSchedtag, 0)
	err := db.FetchModelObjects(SchedtagManager, stq, &schedtags)
	if err != nil {
		return errors.Wrap(err, "db.FetchModelObjects")
	}
	extSchedtagStrs, err := extHost.GetSchedtags()
	if err != nil {
		return errors.Wrap(err, "extHost.GetSchedtags")
	}
	extStStrSet := sets.NewString(extSchedtagStrs...)
	removed := make([]*SSchedtag, 0)
	removedIds := make([]string, 0)
	for i := range schedtags {
		stag := &schedtags[i]
		extTagName := stag.GetMetadata(ctx, METADATA_EXT_SCHEDTAG_KEY, userCred)
		if len(extTagName) == 0 {
			continue
		}
		if !extStStrSet.Has(extTagName) {
			removed = append(removed, stag)
			removedIds = append(removedIds, stag.GetId())
		} else {
			extStStrSet.Delete(extTagName)
		}
	}
	added := extStStrSet.UnsortedList()

	var stagMap map[string]*SSchedtag
	if len(added) > 0 {
		stagMap, err = s.getAllSchedtagsWithExtSchedtagKey(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "getAllSchedtagsWithExtSchedtagKey")
		}
	}

	for _, stStr := range added {
		st, ok := stagMap[stStr]
		if !ok {
			st = &SSchedtag{
				ResourceType: HostManager.KeywordPlural(),
			}
			st.DomainId = s.DomainId
			st.Name = stStr
			st.Description = "Sync from cloud"
			st.SetModelManager(SchedtagManager, st)
			err := SchedtagManager.TableSpec().Insert(ctx, st)
			if err != nil {
				return errors.Wrapf(err, "unable to create schedtag %q", stStr)
			}
			st.SetMetadata(ctx, METADATA_EXT_SCHEDTAG_KEY, stStr, userCred)
		}
		// attach
		hostschedtag := &SHostschedtag{
			HostId: s.GetId(),
		}
		hostschedtag.SetModelManager(HostschedtagManager, hostschedtag)
		hostschedtag.SchedtagId = st.GetId()
		err = HostschedtagManager.TableSpec().Insert(ctx, hostschedtag)
		if err != nil {
			return errors.Wrapf(err, "unable to create hostschedtag for tag %q host %q", stStr, s.GetId())
		}
	}

	if len(removedIds) == 0 {
		return nil
	}

	q := HostschedtagManager.Query().Equals("host_id", s.GetId()).In("schedtag_id", removedIds)
	hostschedtags := make([]SHostschedtag, 0, len(removedIds))
	err = db.FetchModelObjects(HostschedtagManager, q, &hostschedtags)
	if err != nil {
		return errors.Wrap(err, "db.FetchModelObject")
	}
	for i := range hostschedtags {
		err = hostschedtags[i].Detach(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "unable to detach host %q and schedtag %q", hostschedtags[i].HostId, hostschedtags[i].SchedtagId)
		}
	}

	// try to clean
	for _, tag := range removed {
		cnt, err := tag.GetObjectCount()
		if err != nil {
			log.Errorf("unable to GetObjectCount for schedtag %q: %v", tag.GetName(), err)
			continue
		}
		if cnt > 0 {
			continue
		}
		err = tag.Delete(ctx, userCred)
		if err != nil {
			log.Errorf("unable to delete schedtag %q: %v", tag.GetName(), err)
			continue
		}
	}

	return nil
}

func (manager *SHostManager) NewFromCloudHost(ctx context.Context, userCred mcclient.TokenCredential, extHost cloudprovider.ICloudHost, provider *SCloudprovider, izone *SZone) (*SHost, error) {
	host := SHost{}
	host.SetModelManager(manager, &host)

	if izone == nil {
		// onpremise host
		accessIp := extHost.GetAccessIp()
		if len(accessIp) == 0 {
			msg := fmt.Sprintf("fail to find wire for host %s: empty host access ip", extHost.GetName())
			return nil, fmt.Errorf(msg)
		}
		wire, err := WireManager.GetOnPremiseWireOfIp(accessIp)
		if err != nil {
			return nil, errors.Wrapf(err, "GetOnPremiseWireOfIp for host %s with ip %s", extHost.GetName(), accessIp)
		}
		izone, err = wire.GetZone()
		if err != nil {
			return nil, errors.Wrapf(err, "get zone for wire %s", wire.Name)
		}
	}

	host.ExternalId = extHost.GetGlobalId()
	host.ZoneId = izone.Id

	host.HostType = extHost.GetHostType()
	host.OvnVersion = extHost.GetOvnVersion()

	host.Status = extHost.GetStatus()
	host.HostStatus = extHost.GetHostStatus()
	host.SetEnabled(extHost.GetEnabled())

	host.AccessIp = extHost.GetAccessIp()
	host.AccessMac = extHost.GetAccessMac()
	host.SN = extHost.GetSN()
	host.SysInfo = extHost.GetSysInfo()
	host.CpuCount = extHost.GetCpuCount()
	host.NodeCount = extHost.GetNodeCount()
	cpuDesc := extHost.GetCpuDesc()
	if len(cpuDesc) > 128 {
		cpuDesc = cpuDesc[:128]
	}
	host.CpuDesc = cpuDesc
	host.CpuMhz = extHost.GetCpuMhz()
	host.MemSize = extHost.GetMemSizeMB()
	host.StorageSize = extHost.GetStorageSizeMB()
	host.StorageType = extHost.GetStorageType()
	host.CpuCmtbound = 8.0
	if cpuCmt := extHost.GetCpuCmtbound(); cpuCmt > 0 {
		host.CpuCmtbound = cpuCmt
	}
	host.MemCmtbound = 1.0
	if memCmt := extHost.GetMemCmtbound(); memCmt > 0 {
		host.MemCmtbound = memCmt
	}
	if arch := extHost.GetCpuArchitecture(); len(arch) > 0 {
		host.CpuArchitecture = arch
	}

	if reservedMem := extHost.GetReservedMemoryMb(); reservedMem > 0 {
		host.MemReserved = reservedMem
	}

	host.ManagerId = provider.Id
	host.IsEmulated = extHost.IsEmulated()

	host.IsMaintenance = extHost.GetIsMaintenance()
	host.Version = extHost.GetVersion()

	host.IsPublic = false
	host.PublicScope = string(rbacscope.ScopeNone)

	var err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		//newName, err := db.GenerateName(ctx, manager, userCred, extHost.GetName())
		//if err != nil {
		//	return errors.Wrapf(err, "db.GenerateName")
		//}
		host.Name = extHost.GetName()

		return manager.TableSpec().Insert(ctx, &host)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	db.OpsLog.LogEvent(&host, db.ACT_CREATE, host.GetShortDesc(ctx), userCred)

	SyncCloudDomain(userCred, &host, provider.GetOwnerId())

	if err := host.syncSchedtags(ctx, userCred, extHost); err != nil {
		log.Errorf("newFromCloudHost fail in syncSchedtags %v", err)
		return nil, err
	}

	if provider != nil {
		host.SyncShareState(ctx, userCred, provider.getAccountShareInfo())
	}

	if err := manager.ClearSchedDescCache(host.Id); err != nil {
		log.Errorf("ClearSchedDescCache for host %s error %v", host.Name, err)
	}

	return &host, nil
}

func (hh *SHost) SyncHostStorages(ctx context.Context, userCred mcclient.TokenCredential, storages []cloudprovider.ICloudStorage, provider *SCloudprovider, xor bool) ([]SStorage, []cloudprovider.ICloudStorage, compare.SyncResult) {
	lockman.LockRawObject(ctx, "storages", hh.Id)
	defer lockman.ReleaseRawObject(ctx, "storages", hh.Id)

	localStorages := make([]SStorage, 0)
	remoteStorages := make([]cloudprovider.ICloudStorage, 0)
	syncResult := compare.SyncResult{}

	dbStorages := make([]SStorage, 0)

	hostStorages := hh.GetHoststorages()
	for i := 0; i < len(hostStorages); i += 1 {
		storage := hostStorages[i].GetStorage()
		if storage == nil {
			hostStorages[i].Delete(ctx, userCred)
		} else {
			dbStorages = append(dbStorages, *storage)
		}
	}

	// dbStorages := hh._getAttachedStorages(tristate.None, tristate.None)

	removed := make([]SStorage, 0)
	commondb := make([]SStorage, 0)
	commonext := make([]cloudprovider.ICloudStorage, 0)
	added := make([]cloudprovider.ICloudStorage, 0)

	err := compare.CompareSets(dbStorages, storages, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		log.Infof("host %s not connected with %s any more, to detach...", hh.Id, removed[i].Id)
		err := hh.syncRemoveCloudHostStorage(ctx, userCred, &removed[i])
		if errors.Cause(err) == ErrStorageInUse && removed[i].StorageType == api.STORAGE_LOCAL {
			removed[i].SetStatus(userCred, api.STORAGE_OFFLINE, "the only host used this local storage has detached")
			// prevent generating a delete error for syncResult
			continue
		}
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		if !xor {
			log.Infof("host %s is still connected with %s, to update ...", hh.Id, commondb[i].Id)
			err := hh.syncWithCloudHostStorage(ctx, userCred, &commondb[i], commonext[i], provider)
			if err != nil {
				syncResult.UpdateError(err)
			}
		}
		localStorages = append(localStorages, commondb[i])
		remoteStorages = append(remoteStorages, commonext[i])
		syncResult.Update()
	}

	for i := 0; i < len(added); i += 1 {
		log.Infof("host %s is found connected with %s, to add ...", hh.Id, added[i].GetId())
		local, err := hh.newCloudHostStorage(ctx, userCred, added[i], provider)
		if err != nil {
			syncResult.AddError(err)
		} else {
			localStorages = append(localStorages, *local)
			remoteStorages = append(remoteStorages, added[i])
			syncResult.Add()
		}
	}
	return localStorages, remoteStorages, syncResult
}

func (hh *SHost) syncRemoveCloudHostStorage(ctx context.Context, userCred mcclient.TokenCredential, localStorage *SStorage) error {
	hs := hh.GetHoststorageOfId(localStorage.Id)
	err := hs.ValidateDeleteCondition(ctx, nil)
	if err == nil {
		log.Errorf("sync remove hoststorage fail: %s", err)
		err = hs.Detach(ctx, userCred)
	} else {

	}
	return err
}

func (hh *SHost) syncWithCloudHostStorage(ctx context.Context, userCred mcclient.TokenCredential, localStorage *SStorage, extStorage cloudprovider.ICloudStorage, provider *SCloudprovider) error {
	// do nothing
	hs := hh.GetHoststorageOfId(localStorage.Id)
	err := hs.syncWithCloudHostStorage(userCred, extStorage)
	if err != nil {
		return err
	}
	s := hs.GetStorage()
	err = s.syncWithCloudStorage(ctx, userCred, extStorage, provider)
	return err
}

func (hh *SHost) isAttach2Storage(storage *SStorage) bool {
	hs := hh.GetHoststorageOfId(storage.Id)
	return hs != nil
}

func (hh *SHost) Attach2Storage(ctx context.Context, userCred mcclient.TokenCredential, storage *SStorage, mountPoint string) error {
	if hh.isAttach2Storage(storage) {
		return nil
	}

	hs := SHoststorage{}
	hs.SetModelManager(HoststorageManager, &hs)

	hs.StorageId = storage.Id
	hs.HostId = hh.Id
	hs.MountPoint = mountPoint
	err := HoststorageManager.TableSpec().Insert(ctx, &hs)
	if err != nil {
		return err
	}

	db.OpsLog.LogAttachEvent(ctx, hh, storage, userCred, nil)

	return nil
}

func (hh *SHost) newCloudHostStorage(ctx context.Context, userCred mcclient.TokenCredential, extStorage cloudprovider.ICloudStorage, provider *SCloudprovider) (*SStorage, error) {
	storageObj, err := db.FetchByExternalIdAndManagerId(StorageManager, extStorage.GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("manager_id", provider.Id)
	})
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			// no cloud storage found, this may happen for on-premise host
			// create the storage right now
			zone, _ := hh.GetZone()
			storageObj, err = StorageManager.newFromCloudStorage(ctx, userCred, extStorage, provider, zone)
			if err != nil {
				return nil, errors.Wrapf(err, "StorageManager.newFromCloudStorage")
			}
		} else {
			return nil, errors.Wrapf(err, "FetchByExternalIdAndManagerId(%s)", extStorage.GetGlobalId())
		}
	}
	storage := storageObj.(*SStorage)
	err = hh.Attach2Storage(ctx, userCred, storage, extStorage.GetMountPoint())
	return storage, err
}

func (hh *SHost) SyncHostWires(ctx context.Context, userCred mcclient.TokenCredential, wires []cloudprovider.ICloudWire) compare.SyncResult {
	lockman.LockRawObject(ctx, "wires", hh.Id)
	defer lockman.ReleaseRawObject(ctx, "wires", hh.Id)

	syncResult := compare.SyncResult{}

	dbWires := make([]SWire, 0)

	hostNetifs := hh.GetHostNetInterfaces()
	for i := 0; i < len(hostNetifs); i += 1 {
		wire := hostNetifs[i].GetWire()
		if wire != nil {
			// hostNetifs[i].Remove(ctx, userCred)
		} else {
			dbWires = append(dbWires, *wire)
		}
	}

	// dbWires := hh.getAttachedWires()

	removed := make([]SWire, 0)
	commondb := make([]SWire, 0)
	commonext := make([]cloudprovider.ICloudWire, 0)
	added := make([]cloudprovider.ICloudWire, 0)

	err := compare.CompareSets(dbWires, wires, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		log.Infof("host %s not connected with %s any more, to detach...", hh.Id, removed[i].Id)
		err := hh.syncRemoveCloudHostWire(ctx, userCred, &removed[i])
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		log.Infof("host %s is still connected with %s, to update...", hh.Id, commondb[i].Id)
		err := hh.syncWithCloudHostWire(commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		log.Infof("host %s is found connected with %s, to add...", hh.Id, added[i].GetId())
		err := hh.newCloudHostWire(ctx, userCred, added[i])
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}

	return syncResult
}

func (hh *SHost) syncRemoveCloudHostWire(ctx context.Context, userCred mcclient.TokenCredential, localwire *SWire) error {
	netifs := hh.getNetifsOnWire(localwire.Id)
	for i := range netifs {
		err := netifs[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "delete")
		}
	}
	return nil
}

func (hh *SHost) syncWithCloudHostWire(extWire cloudprovider.ICloudWire) error {
	// do nothing
	return nil
}

func (hh *SHost) Attach2Wire(ctx context.Context, userCred mcclient.TokenCredential, wire *SWire) error {
	netif := SNetInterface{}
	netif.SetModelManager(NetInterfaceManager, &netif)

	netif.Mac = stringutils2.HashIdsMac(hh.Id, wire.Id)
	netif.VlanId = 1
	netif.WireId = wire.Id
	netif.BaremetalId = hh.Id
	err := NetInterfaceManager.TableSpec().InsertOrUpdate(ctx, &netif)
	if err != nil {
		return errors.Wrap(err, "InsertOrUpdate")
	}
	return nil
}

func (hh *SHost) newCloudHostWire(ctx context.Context, userCred mcclient.TokenCredential, extWire cloudprovider.ICloudWire) error {
	wireObj, err := db.FetchByExternalIdAndManagerId(WireManager, extWire.GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		sq := VpcManager.Query().SubQuery()
		return q.Join(sq, sqlchemy.Equals(sq.Field("id"), q.Field("vpc_id"))).Filter(sqlchemy.Equals(sq.Field("manager_id"), hh.ManagerId))
	})
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	wire := wireObj.(*SWire)
	err = hh.Attach2Wire(ctx, userCred, wire)
	return err
}

type SGuestSyncResult struct {
	Local  *SGuest
	Remote cloudprovider.ICloudVM
	IsNew  bool
}

func IsNeedSkipSync(ext cloudprovider.ICloudResource) (bool, string) {
	if len(options.Options.SkipServerBySysTagKeys) == 0 && len(options.Options.SkipServerByUserTagKeys) == 0 {
		return false, ""
	}
	if keys := strings.Split(options.Options.SkipServerBySysTagKeys, ","); len(keys) > 0 {
		for key := range ext.GetSysTags() {
			key = strings.Trim(key, "")
			if len(key) > 0 && utils.IsInStringArray(key, keys) {
				return true, key
			}
		}
	}
	if userKeys := strings.Split(options.Options.SkipServerByUserTagKeys, ","); len(userKeys) > 0 {
		tags, _ := ext.GetTags()
		for key := range tags {
			key = strings.Trim(key, "")
			if len(key) > 0 && utils.IsInStringArray(key, userKeys) {
				return true, key
			}
		}
	}
	return false, ""
}

func (self *SGuest) Purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.purge(ctx, userCred)
}

func (hh *SHost) SyncHostVMs(ctx context.Context, userCred mcclient.TokenCredential, iprovider cloudprovider.ICloudProvider, vms []cloudprovider.ICloudVM, syncOwnerId mcclient.IIdentityProvider, xor bool) ([]SGuestSyncResult, compare.SyncResult) {
	lockman.LockRawObject(ctx, GuestManager.Keyword(), hh.Id)
	defer lockman.ReleaseRawObject(ctx, GuestManager.Keyword(), hh.Id)

	syncVMPairs := make([]SGuestSyncResult, 0)
	syncResult := compare.SyncResult{}

	dbVMs, err := hh.GetGuests()
	if err != nil {
		syncResult.Error(errors.Wrapf(err, "GetGuests"))
		return nil, syncResult
	}

	for i := range dbVMs {
		if taskman.TaskManager.IsInTask(&dbVMs[i]) {
			syncResult.Error(fmt.Errorf("server %s(%s)in task", dbVMs[i].Name, dbVMs[i].Id))
			return nil, syncResult
		}
	}

	removed := make([]SGuest, 0)
	commondb := make([]SGuest, 0)
	commonext := make([]cloudprovider.ICloudVM, 0)
	added := make([]cloudprovider.ICloudVM, 0)
	duplicated := make(map[string][]cloudprovider.ICloudVM)

	err = compare.CompareSets2(dbVMs, vms, &removed, &commondb, &commonext, &added, &duplicated)
	if err != nil {
		syncResult.Error(err)
		return nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err := removed[i].SyncRemoveCloudVM(ctx, userCred, true)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			skip, key := IsNeedSkipSync(commonext[i])
			if skip {
				log.Infof("delete server %s(%s) with system tag key: %s", commonext[i].GetName(), commonext[i].GetGlobalId(), key)
				err := commondb[i].purge(ctx, userCred)
				if err != nil {
					syncResult.DeleteError(err)
					continue
				}
				syncResult.Delete()
				continue
			}
			err := commondb[i].syncWithCloudVM(ctx, userCred, iprovider, hh, commonext[i], syncOwnerId, true)
			if err != nil {
				syncResult.UpdateError(err)
				continue
			}
			syncVMPair := SGuestSyncResult{
				Local:  &commondb[i],
				Remote: commonext[i],
				IsNew:  false,
			}
			syncVMPairs = append(syncVMPairs, syncVMPair)
			syncResult.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		skip, key := IsNeedSkipSync(added[i])
		if skip {
			log.Infof("skip server %s(%s) sync with system tag key: %s", added[i].GetName(), added[i].GetGlobalId(), key)
			continue
		}
		vm, err := db.FetchByExternalIdAndManagerId(GuestManager, added[i].GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			sq := HostManager.Query().SubQuery()
			return q.Join(sq, sqlchemy.Equals(sq.Field("id"), q.Field("host_id"))).Filter(sqlchemy.Equals(sq.Field("manager_id"), hh.ManagerId))
		})
		if err != nil && errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("failed to found guest by externalId %s error: %v", added[i].GetGlobalId(), err)
			continue
		}
		if vm != nil {
			guest := vm.(*SGuest)
			ihost := added[i].GetIHost()
			if ihost == nil {
				log.Errorf("failed to found ihost from vm %s", added[i].GetGlobalId())
				continue
			}
			_host, err := db.FetchByExternalIdAndManagerId(HostManager, ihost.GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", hh.ManagerId)
			})
			if err != nil {
				log.Errorf("failed to found host by externalId %s", ihost.GetGlobalId())
				continue
			}
			host := _host.(*SHost)
			err = guest.syncWithCloudVM(ctx, userCred, iprovider, host, added[i], syncOwnerId, true)
			if err != nil {
				syncResult.UpdateError(err)
			} else {
				syncResult.Update()
			}
			continue
		}
		if added[i].GetBillingType() == billing_api.BILLING_TYPE_PREPAID {
			vhost := HostManager.GetHostByRealExternalId(added[i].GetGlobalId())
			if vhost != nil {
				// this recycle vm is not build yet, skip synchronize
				err = vhost.SyncWithRealPrepaidVM(ctx, userCred, added[i])
				if err != nil {
					syncResult.AddError(err)
				}
				continue
			}
		}
		new, err := GuestManager.newCloudVM(ctx, userCred, iprovider, hh, added[i], syncOwnerId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncVMPair := SGuestSyncResult{
				Local:  new,
				Remote: added[i],
				IsNew:  true,
			}
			syncVMPairs = append(syncVMPairs, syncVMPair)
			syncResult.Add()
		}
	}

	if len(duplicated) > 0 {
		errs := make([]error, 0)
		for k, vms := range duplicated {
			errs = append(errs, errors.Wrapf(errors.ErrDuplicateId, "Duplicate Id %s (%d)", k, len(vms)))
		}
		syncResult.AddError(errors.NewAggregate(errs))
	}

	return syncVMPairs, syncResult
}

func (hh *SHost) getNetworkOfIPOnHost(ipAddr string) (*SNetwork, error) {
	netInterfaces := hh.GetHostNetInterfaces()
	for _, netInterface := range netInterfaces {
		network, err := netInterface.GetCandidateNetworkForIp(nil, nil, rbacscope.ScopeNone, ipAddr)
		if err == nil && network != nil {
			return network, nil
		}
	}

	return nil, fmt.Errorf("IP %s not reachable on this host", ipAddr)
}

func (hh *SHost) GetNetinterfacesWithIdAndCredential(netId string, userCred mcclient.TokenCredential, reserved bool) ([]SNetInterface, *SNetwork, error) {
	netObj, err := NetworkManager.FetchById(netId)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fetch by id %q", netId)
	}
	net := netObj.(*SNetwork)
	used, err := net.getFreeAddressCount()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get network %q free address count", net.GetName())
	}
	if used == 0 && !reserved && !options.Options.BaremetalServerReuseHostIp {
		return nil, nil, errors.Errorf("network %q out of usage", net.GetName())
	}
	matchNetIfs := make([]SNetInterface, 0)
	netifs := hh.GetHostNetInterfaces()
	for i := 0; i < len(netifs); i++ {
		if !netifs[i].IsUsableServernic() {
			continue
		}
		if netifs[i].WireId == net.WireId {
			matchNetIfs = append(matchNetIfs, netifs[i])
			// return &netifs[i], net
		}
	}
	if len(matchNetIfs) > 0 {
		return matchNetIfs, net, nil
	}
	return nil, nil, errors.Errorf("not found matched netinterface by net %q wire %q", net.GetName(), net.WireId)
}

func (hh *SHost) GetNetworkWithId(netId string, reserved bool) (*SNetwork, error) {
	var q1, q2, q3 *sqlchemy.SQuery
	{
		// classic network
		networks := NetworkManager.Query()
		netifs := NetInterfaceManager.Query().SubQuery()
		hosts := HostManager.Query().SubQuery()
		q1 = networks
		q1 = q1.Join(netifs, sqlchemy.Equals(netifs.Field("wire_id"), networks.Field("wire_id")))
		q1 = q1.Join(hosts, sqlchemy.Equals(hosts.Field("id"), netifs.Field("baremetal_id")))
		q1 = q1.Filter(sqlchemy.Equals(networks.Field("id"), netId))
		q1 = q1.Filter(sqlchemy.Equals(hosts.Field("id"), hh.Id))
	}
	{
		// vpc network
		networks := NetworkManager.Query()
		wires := WireManager.Query().SubQuery()
		vpcs := VpcManager.Query().SubQuery()
		regions := CloudregionManager.Query().SubQuery()
		q2 = networks
		q2 = q2.Join(wires, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
		q2 = q2.Join(vpcs, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
		q2 = q2.Join(regions, sqlchemy.Equals(regions.Field("id"), vpcs.Field("cloudregion_id")))
		q2 = q2.Filter(sqlchemy.Equals(networks.Field("id"), netId))
		q2 = q2.Filter(
			sqlchemy.OR(
				sqlchemy.AND(
					sqlchemy.Equals(regions.Field("provider"), api.CLOUD_PROVIDER_ONECLOUD),
					sqlchemy.NOT(sqlchemy.Equals(vpcs.Field("id"), api.DEFAULT_VPC_ID)),
				),
				sqlchemy.AND(
					sqlchemy.Equals(regions.Field("provider"), api.CLOUD_PROVIDER_CLOUDPODS),
					sqlchemy.NOT(sqlchemy.Equals(vpcs.Field("external_id"), api.DEFAULT_VPC_ID)),
				),
			),
		)
	}
	{
		// network additional wires
		networks := NetworkManager.Query()
		networkAdditionalWires := NetworkAdditionalWireManager.Query().SubQuery()
		netifs := NetInterfaceManager.Query().SubQuery()
		hosts := HostManager.Query().SubQuery()
		q3 = networks
		q3 = q3.Join(networkAdditionalWires, sqlchemy.Equals(networks.Field("id"), networkAdditionalWires.Field("network_id")))
		q3 = q3.Join(netifs, sqlchemy.Equals(netifs.Field("wire_id"), networkAdditionalWires.Field("wire_id")))
		q3 = q3.Join(hosts, sqlchemy.Equals(hosts.Field("id"), netifs.Field("baremetal_id")))
		q3 = q3.Filter(sqlchemy.Equals(networks.Field("id"), netId))
		q3 = q3.Filter(sqlchemy.Equals(hosts.Field("id"), hh.Id))
	}

	q := sqlchemy.Union(q1, q2, q3).Query().Distinct()

	net := SNetwork{}
	net.SetModelManager(NetworkManager, &net)
	err := q.First(&net)
	if err != nil {
		return nil, err
	}
	if reserved {
		return &net, nil
	}
	freeCnt, err := net.getFreeAddressCount()
	if err != nil {
		return nil, err
	}
	if freeCnt > 0 {
		return &net, nil
	}
	return nil, fmt.Errorf("No IP address")
}

func (manager *SHostManager) FetchHostById(hostId string) *SHost {
	host := SHost{}
	host.SetModelManager(manager, &host)
	err := manager.Query().Equals("id", hostId).First(&host)
	if err != nil {
		log.Errorf("fetchHostById fail %s", err)
		return nil
	} else {
		return &host
	}
}

func (manager *SHostManager) FetchHostByHostname(hostname string) *SHost {
	host := SHost{}
	host.SetModelManager(manager, &host)
	err := manager.Query().Startswith("name", hostname).First(&host)
	if err != nil {
		log.Errorf("fetch host by hostname %s failed %s", hostname, err)
		return nil
	} else {
		return &host
	}
}

func (manager *SHostManager) totalCountQ(
	userCred mcclient.IIdentityProvider,
	scope rbacscope.TRbacScope,
	rangeObjs []db.IStandaloneModel,
	hostStatus, status string,
	hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	enabled, isBaremetal tristate.TriState,
	policyResult rbacutils.SPolicyResult,
) *sqlchemy.SQuery {
	hosts := manager.Query().SubQuery()
	q := hosts.Query(
		hosts.Field("mem_size"),
		hosts.Field("mem_reserved"),
		hosts.Field("mem_cmtbound"),
		hosts.Field("cpu_count"),
		hosts.Field("cpu_reserved"),
		hosts.Field("cpu_cmtbound"),
		hosts.Field("storage_size"),
	)
	if scope != rbacscope.ScopeSystem && userCred != nil {
		q = q.Filter(sqlchemy.Equals(hosts.Field("domain_id"), userCred.GetProjectDomainId()))
	}
	if len(status) > 0 {
		q = q.Filter(sqlchemy.Equals(hosts.Field("status"), status))
	}
	if len(hostStatus) > 0 {
		q = q.Filter(sqlchemy.Equals(hosts.Field("host_status"), hostStatus))
	}
	if !enabled.IsNone() {
		cond := sqlchemy.IsFalse
		if enabled.Bool() {
			cond = sqlchemy.IsTrue
		}
		q = q.Filter(cond(hosts.Field("enabled")))
	}
	if !isBaremetal.IsNone() {
		if isBaremetal.Bool() {
			q = q.Filter(sqlchemy.AND(
				sqlchemy.IsTrue(hosts.Field("is_baremetal")),
				sqlchemy.Equals(hosts.Field("host_type"), api.HOST_TYPE_BAREMETAL),
			))
		} else {
			q = q.Filter(sqlchemy.OR(
				sqlchemy.IsFalse(hosts.Field("is_baremetal")),
				sqlchemy.NotEquals(hosts.Field("host_type"), api.HOST_TYPE_BAREMETAL),
			))
		}
	}

	q = db.ObjectIdQueryWithPolicyResult(q, HostManager, policyResult)

	isolatedDevices := IsolatedDeviceManager.Query().SubQuery()
	iq := isolatedDevices.Query(
		isolatedDevices.Field("host_id"),
		sqlchemy.SUM("isolated_reserved_memory", isolatedDevices.Field("reserved_memory")),
		sqlchemy.SUM("isolated_reserved_cpu", isolatedDevices.Field("reserved_cpu")),
		sqlchemy.SUM("isolated_reserved_storage", isolatedDevices.Field("reserved_storage")),
	).IsNullOrEmpty("guest_id").GroupBy(isolatedDevices.Field("host_id")).SubQuery()
	q = q.LeftJoin(iq, sqlchemy.Equals(q.Field("id"), iq.Field("host_id")))
	q.AppendField(
		iq.Field("isolated_reserved_memory"),
		iq.Field("isolated_reserved_cpu"),
		iq.Field("isolated_reserved_storage"),
	)
	q = AttachUsageQuery(q, hosts, hostTypes, resourceTypes, providers, brands, cloudEnv, rangeObjs)
	// log.Debugf("hostCount: %s", q.String())
	return q
}

type HostStat struct {
	MemSize                 int
	MemReserved             int
	MemCmtbound             float32
	CpuCount                int
	CpuReserved             int
	CpuCmtbound             float32
	StorageSize             int
	IsolatedReservedMemory  int64
	IsolatedReservedCpu     int64
	IsolatedReservedStorage int64
}

type HostsCountStat struct {
	StorageSize             int64
	Count                   int64
	Memory                  int64
	MemoryTotal             int64
	MemoryVirtual           float64
	MemoryReserved          int64
	CPU                     int64
	CPUTotal                int64
	CPUVirtual              float64
	IsolatedReservedMemory  int64
	IsolatedReservedCpu     int64
	IsolatedReservedStorage int64
}

func (manager *SHostManager) calculateCount(q *sqlchemy.SQuery) HostsCountStat {
	usableSize := func(act, reserved int) int {
		aSize := 0
		if reserved > 0 && reserved < act {
			aSize = act - reserved
		} else {
			aSize = act
		}
		return aSize
	}
	var (
		tStore  int64   = 0
		tCnt    int64   = 0
		tMem    int64   = 0
		tVmem   float64 = 0.0
		rMem    int64   = 0
		tCPU    int64   = 0
		tVCPU   float64 = 0.0
		irMem   int64   = 0
		irCpu   int64   = 0
		irStore int64   = 0

		totalMem int64 = 0
		totalCPU int64 = 0
	)
	stats := make([]HostStat, 0)
	err := q.All(&stats)
	if err != nil {
		log.Errorf("%v", err)
	}
	for _, stat := range stats {
		if stat.MemSize == 0 {
			continue
		}
		tCnt += 1
		if stat.StorageSize > 0 {
			tStore += int64(stat.StorageSize)
		}
		aMem := usableSize(stat.MemSize, stat.MemReserved)
		aCpu := usableSize(int(stat.CpuCount), int(stat.CpuReserved))
		tMem += int64(aMem)
		totalMem += int64(stat.MemSize)
		tCPU += int64(aCpu)
		totalCPU += int64(stat.CpuCount)
		if stat.MemCmtbound <= 0.0 {
			stat.MemCmtbound = options.Options.DefaultMemoryOvercommitBound
		}
		if stat.CpuCmtbound <= 0.0 {
			stat.CpuCmtbound = options.Options.DefaultCPUOvercommitBound
		}
		rMem += int64(stat.MemReserved)
		tVmem += float64(float32(aMem) * stat.MemCmtbound)
		tVCPU += float64(float32(aCpu) * stat.CpuCmtbound)
		irMem += stat.IsolatedReservedMemory
		irCpu += stat.IsolatedReservedCpu
		irStore += stat.IsolatedReservedStorage
	}
	return HostsCountStat{
		StorageSize:             tStore,
		Count:                   tCnt,
		Memory:                  tMem,
		MemoryTotal:             totalMem,
		MemoryVirtual:           tVmem,
		MemoryReserved:          rMem,
		CPU:                     tCPU,
		CPUTotal:                totalCPU,
		CPUVirtual:              tVCPU,
		IsolatedReservedCpu:     irCpu,
		IsolatedReservedMemory:  irMem,
		IsolatedReservedStorage: irStore,
	}
}

func (manager *SHostManager) TotalCount(
	userCred mcclient.IIdentityProvider,
	scope rbacscope.TRbacScope,
	rangeObjs []db.IStandaloneModel,
	hostStatus, status string,
	hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	enabled, isBaremetal tristate.TriState,
	policyResult rbacutils.SPolicyResult,
) HostsCountStat {
	return manager.calculateCount(
		manager.totalCountQ(
			userCred,
			scope,
			rangeObjs,
			hostStatus,
			status,
			hostTypes,
			resourceTypes,
			providers,
			brands,
			cloudEnv,
			enabled,
			isBaremetal,
			policyResult,
		),
	)
}

func (hh *SHost) GetIHost(ctx context.Context) (cloudprovider.ICloudHost, error) {
	host, _, err := hh.GetIHostAndProvider(ctx)
	return host, err
}

func (hh *SHost) GetIHostAndProvider(ctx context.Context) (cloudprovider.ICloudHost, cloudprovider.ICloudProvider, error) {
	iregion, provider, err := hh.GetIRegionAndProvider(ctx)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "GetIRegionAndProvider")
	}
	ihost, err := iregion.GetIHostById(hh.ExternalId)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "iregion.GetIHostById(%s)", hh.ExternalId)
	}
	return ihost, provider, nil
}

func (hh *SHost) GetIRegionAndProvider(ctx context.Context) (cloudprovider.ICloudRegion, cloudprovider.ICloudProvider, error) {
	provider, err := hh.GetDriver(ctx)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "GetDriver")
	}
	var iregion cloudprovider.ICloudRegion
	if provider.GetFactory().IsOnPremise() {
		iregion, err = provider.GetOnPremiseIRegion()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "provider.GetOnPremiseIRegio")
		}
	} else {
		region, err := hh.GetRegion()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "GetRegion")
		}
		iregion, err = provider.GetIRegionById(region.ExternalId)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "provider.GetIRegionById(%s)", region.ExternalId)
		}
	}
	return iregion, provider, nil
}

func (hh *SHost) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	region, _, err := hh.GetIRegionAndProvider(ctx)
	return region, err
}

func (hh *SHost) getDiskConfig() jsonutils.JSONObject {
	bs := hh.GetBaremetalstorage()
	if bs != nil {
		return bs.Config
	}
	return nil
}

func (hh *SHost) GetBaremetalServer() *SGuest {
	if !hh.IsBaremetal {
		return nil
	}
	guest := SGuest{}
	guest.SetModelManager(GuestManager, &guest)
	q := GuestManager.Query().Equals("host_id", hh.Id).Equals("hypervisor", api.HOST_TYPE_BAREMETAL)
	err := q.First(&guest)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("query fail %s", err)
		}
		return nil
	}
	return &guest
}

func (hh *SHost) GetSchedtags() []SSchedtag {
	return GetSchedtags(HostschedtagManager, hh.Id)
}

type SHostGuestResourceUsage struct {
	GuestCount     int
	GuestVcpuCount int
	GuestVmemSize  int
}

func (hh *SHost) getGuestsResource(status string) *SHostGuestResourceUsage {
	guests := GuestManager.Query().SubQuery()
	q := guests.Query(sqlchemy.COUNT("guest_count"),
		sqlchemy.SUM("guest_vcpu_count", guests.Field("vcpu_count")),
		sqlchemy.SUM("guest_vmem_size", guests.Field("vmem_size")))
	cond := sqlchemy.OR(sqlchemy.Equals(q.Field("host_id"), hh.Id),
		sqlchemy.Equals(q.Field("backup_host_id"), hh.Id))
	q = q.Filter(cond)
	if len(status) > 0 {
		q = q.Equals("status", status)
	}
	stat := SHostGuestResourceUsage{}
	err := q.First(&stat)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return &stat
}

func (hh *SHost) getMoreDetails(ctx context.Context, out api.HostDetails, showReason bool) api.HostDetails {
	server := hh.GetBaremetalServer()
	if server != nil {
		out.ServerId = server.Id
		out.Server = server.Name
		out.ServerPendingDeleted = server.PendingDeleted
		if hh.HostType == api.HOST_TYPE_BAREMETAL {
			out.ServerIps = strings.Join(server.GetRealIPs(), ",")
		}
	}
	nics := hh.GetNics()
	if nics != nil && len(nics) > 0 {
		// nicInfos := []jsonutils.JSONObject{}
		// for i := 0; i < len(nics); i += 1 {
		// 	nicInfos = append(nicInfos, jsonutils.Marshal(nics[i]))
		// }
		out.NicCount = len(nics)
		out.NicInfo = nics
	}
	out.Schedtags = GetSchedtagsDetailsToResourceV2(hh, ctx)
	var usage *SHostGuestResourceUsage
	if options.Options.IgnoreNonrunningGuests {
		usage = hh.getGuestsResource(api.VM_RUNNING)
	} else {
		usage = hh.getGuestsResource("")
	}
	if usage != nil {
		out.CpuCommit = usage.GuestVcpuCount
		out.MemCommit = usage.GuestVmemSize
	}
	totalCpu := hh.GetCpuCount()
	cpuCommitRate := 0.0
	if totalCpu > 0 && usage.GuestVcpuCount > 0 {
		cpuCommitRate = float64(usage.GuestVcpuCount) * 1.0 / float64(totalCpu)
	}
	out.CpuCommitRate = cpuCommitRate
	totalMem := hh.GetMemSize()
	memCommitRate := 0.0
	if totalMem > 0 && usage.GuestVmemSize > 0 {
		memCommitRate = float64(usage.GuestVmemSize) * 1.0 / float64(totalMem)
	}
	out.MemCommitRate = memCommitRate
	capa := hh.GetAttachedLocalStorageCapacity()
	out.Storage = capa.Capacity
	out.StorageUsed = capa.Used
	out.ActualStorageUsed = capa.ActualUsed
	out.StorageWaste = capa.Wasted
	out.StorageVirtual = capa.VCapacity
	out.StorageFree = capa.GetFree()
	out.StorageCommitRate = capa.GetCommitRate()
	out.Spec = hh.GetHardwareSpecification()

	// custom cpu mem commit bound
	out.CpuCommitBound = hh.GetCPUOvercommitBound()
	out.MemCommitBound = hh.GetMemoryOvercommitBound()

	// extra = hh.SManagedResourceBase.getExtraDetails(ctx, extra)

	out.IsPrepaidRecycle = false
	if hh.IsPrepaidRecycle() {
		out.IsPrepaidRecycle = true
	}

	if hh.IsBaremetal {
		out.CanPrepare = true
		err := hh.canPrepare()
		if err != nil {
			out.CanPrepare = false
			if showReason {
				out.PrepareFailReason = err.Error()
			}
		}
	}

	if hh.EnableHealthCheck && hostHealthChecker != nil {
		out.AllowHealthCheck = true
	}
	if hh.GetMetadata(ctx, api.HOSTMETA_AUTO_MIGRATE_ON_HOST_DOWN, nil) == "enable" {
		out.AutoMigrateOnHostDown = true
	}
	if hh.GetMetadata(ctx, api.HOSTMETA_AUTO_MIGRATE_ON_HOST_SHUTDOWN, nil) == "enable" {
		out.AutoMigrateOnHostShutdown = true
	}

	if count, rs := hh.GetReservedResourceForIsolatedDevice(); rs != nil {
		out.ReservedResourceForGpu = *rs
		out.IsolatedDeviceCount = count
	}
	return out
}

type sGuestCnt struct {
	GuestCnt               int
	BackupGuestCnt         int
	RunningGuestCnt        int
	ReadyGuestCnt          int
	OtherGuestCnt          int
	PendingDeletedGuestCnt int
	NonsystemGuestCnt      int
}

func (manager *SHostManager) FetchGuestCnt(hostIds []string) map[string]*sGuestCnt {
	ret := map[string]*sGuestCnt{}
	if len(hostIds) == 0 {
		return ret
	}
	guests := []SGuest{}
	err := GuestManager.RawQuery().IsFalse("deleted").In("host_id", hostIds).NotEquals("hypervisor", api.HYPERVISOR_POD).All(&guests)
	if err != nil {
		log.Errorf("query host %s guests error: %v", hostIds, err)
	}
	for _, guest := range guests {
		_, ok := ret[guest.HostId]
		if !ok {
			ret[guest.HostId] = &sGuestCnt{}
		}
		if guest.PendingDeleted {
			ret[guest.HostId].PendingDeletedGuestCnt += 1
			continue
		}
		ret[guest.HostId].GuestCnt += 1
		switch guest.Status {
		case api.VM_RUNNING:
			ret[guest.HostId].RunningGuestCnt += 1
		case api.VM_READY:
			ret[guest.HostId].ReadyGuestCnt += 1
		default:
			ret[guest.HostId].OtherGuestCnt += 1
		}
		if !guest.IsSystem {
			ret[guest.HostId].NonsystemGuestCnt += 1
		}
	}

	GuestManager.RawQuery().IsFalse("deleted").In("backup_host_id", hostIds).NotEquals("hypervisor", api.HYPERVISOR_POD).All(&guests)
	for _, guest := range guests {
		_, ok := ret[guest.BackupHostId]
		if !ok {
			ret[guest.BackupHostId] = &sGuestCnt{}
		}
		ret[guest.BackupHostId].BackupGuestCnt += 1
	}

	return ret
}

func (hh *SHost) GetReservedResourceForIsolatedDevice() (int, *api.IsolatedDeviceReservedResourceInput) {
	if devs := IsolatedDeviceManager.FindByHost(hh.Id); len(devs) == 0 {
		return -1, nil
	} else {
		return len(devs), hh.GetDevsReservedResource(devs)
	}
}

func (hh *SHost) GetDevsReservedResource(devs []SIsolatedDevice) *api.IsolatedDeviceReservedResourceInput {
	reservedCpu, reservedMem, reservedStorage := 0, 0, 0
	reservedResourceForGpu := api.IsolatedDeviceReservedResourceInput{
		ReservedStorage: &reservedStorage,
		ReservedMemory:  &reservedMem,
		ReservedCpu:     &reservedCpu,
	}
	for _, dev := range devs {
		if !utils.IsInStringArray(dev.DevType, api.VALID_GPU_TYPES) {
			continue
		}
		reservedCpu += dev.ReservedCpu
		reservedMem += dev.ReservedMemory
		reservedStorage += dev.ReservedStorage
	}
	return &reservedResourceForGpu
}

func (hh *SHost) GetMetadataHiddenKeys() []string {
	return []string{}
}

func (manager *SHostManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.HostDetails {
	rows := make([]api.HostDetails, len(objs))
	stdRows := manager.SEnabledStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	zoneRows := manager.SZoneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	showReason := false
	if query.Contains("show_fail_reason") {
		showReason = true
	}
	hostIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.HostDetails{
			EnabledStatusInfrasResourceBaseDetails: stdRows[i],
			ManagedResourceInfo:                    managerRows[i],
			ZoneResourceInfo:                       zoneRows[i],
		}
		host := objs[i].(*SHost)
		hostIds[i] = host.Id
		rows[i] = host.getMoreDetails(ctx, rows[i], showReason)
	}
	guestCnts := manager.FetchGuestCnt(hostIds)
	for i := range rows {
		cnt, ok := guestCnts[hostIds[i]]
		if ok {
			rows[i].Guests = cnt.GuestCnt
			rows[i].RunningGuests = cnt.RunningGuestCnt
			rows[i].ReadyGuests = cnt.ReadyGuestCnt
			rows[i].OtherGuests = cnt.OtherGuestCnt
			rows[i].NonsystemGuests = cnt.NonsystemGuestCnt
			rows[i].PendingDeletedGuests = cnt.PendingDeletedGuestCnt
		}
	}
	return rows
}

func (hh *SHost) GetDetailsVnc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(hh.Status, []string{api.BAREMETAL_READY, api.BAREMETAL_RUNNING}) {
		retval := jsonutils.NewDict()
		retval.Set("host_id", jsonutils.NewString(hh.Id))
		zone, _ := hh.GetZone()
		retval.Set("zone", jsonutils.NewString(zone.GetName()))
		return retval, nil
	}
	return jsonutils.NewDict(), nil
}

func (hh *SHost) GetDetailsIpmi(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret, ok := hh.IpmiInfo.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.NewNotFoundError("No ipmi information was found for host %s", hh.Name)
	}
	password, err := ret.GetString("password")
	if err != nil {
		return nil, httperrors.NewNotFoundError("IPMI has no password information")
	}
	descryptedPassword, err := utils.DescryptAESBase64(hh.Id, password)
	if err != nil {
		return nil, err
	}
	ret.Set("password", jsonutils.NewString(descryptedPassword))
	return ret, nil
}

func (manager *SHostManager) GetHostsByManagerAndRegion(managerId string, regionId string) []SHost {
	zones := ZoneManager.Query().Equals("cloudregion_id", regionId).SubQuery()
	hosts := HostManager.Query()
	q := hosts.Equals("manager_id", managerId)
	q = q.Join(zones, sqlchemy.Equals(zones.Field("id"), hosts.Field("zone_id")))
	ret := make([]SHost, 0)
	err := db.FetchModelObjects(HostManager, q, &ret)
	if err != nil {
		log.Errorf("GetHostsByManagerAndRegion fail %s", err)
		return nil
	}
	return ret
}

func (hh *SHost) RequestScanIsolatedDevices(ctx context.Context, userCred mcclient.TokenCredential) error {
	_, err := hh.Request(ctx, userCred, "POST", fmt.Sprintf("/hosts/%s/probe-isolated-devices", hh.Id), mcclient.GetTokenHeaders(userCred), nil)
	if err != nil {
		return errors.Wrapf(err, "request host %s probe isolaed devices", hh.Id)
	}
	return nil
}

func (hh *SHost) Request(ctx context.Context, userCred mcclient.TokenCredential, method httputils.THttpMethod, url string, headers http.Header, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	s := auth.GetSession(ctx, userCred, "")
	_, ret, err := s.JSONRequest(hh.ManagerUri, "", method, url, headers, body)
	return ret, err
}

func (hh *SHost) GetLocalStoragecache() *SStoragecache {
	localStorages := hh.GetAttachedLocalStorages()
	for i := 0; i < len(localStorages); i += 1 {
		sc := localStorages[i].GetStoragecache()
		if sc != nil {
			return sc
		}
	}
	return nil
}

func (hh *SHost) GetStoragecache() *SStoragecache {
	localStorages := hh.GetAttachedEnabledHostStorages(nil)
	for i := 0; i < len(localStorages); i += 1 {
		sc := localStorages[i].GetStoragecache()
		if sc != nil {
			return sc
		}
	}
	return nil
}

func (hh *SHost) PostCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	hh.SEnabledStatusInfrasResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	input := api.HostCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		log.Errorf("data.Unmarshal fail %s", err)
		return
	}
	kwargs := data.(*jsonutils.JSONDict)
	ipmiInfo, err := fetchIpmiInfo(input.HostIpmiAttributes, hh.Id)
	if err != nil {
		log.Errorf("fetchIpmiInfo fail %s", err)
		return
	}
	ipmiInfoJson := jsonutils.Marshal(ipmiInfo).(*jsonutils.JSONDict)
	if ipmiInfoJson.Length() > 0 {
		_, err := hh.SaveUpdates(func() error {
			hh.IpmiInfo = ipmiInfoJson
			return nil
		})
		if err != nil {
			log.Errorf("save updates: %v", err)
		} else if len(ipmiInfo.IpAddr) > 0 {
			hh.setIpmiIp(userCred, ipmiInfo.IpAddr)
		}
	}
	if len(input.AccessIp) > 0 {
		hh.setAccessIp(userCred, input.AccessIp)
	}
	if len(input.AccessMac) > 0 {
		hh.setAccessMac(userCred, input.AccessMac)
	}
	noProbe := false
	if input.NoProbe != nil {
		noProbe = *input.NoProbe
	}
	if len(hh.ZoneId) > 0 && hh.HostType == api.HOST_TYPE_BAREMETAL && !noProbe {
		// ipmiInfo, _ := hh.GetIpmiInfo()
		if len(ipmiInfo.IpAddr) > 0 {
			hh.StartBaremetalCreateTask(ctx, userCred, kwargs, "")
		}
	}
	if hh.OvnVersion != "" && hh.OvnMappedIpAddr == "" {
		HostManager.lockAllocOvnMappedIpAddr(ctx)
		defer HostManager.unlockAllocOvnMappedIpAddr(ctx)
		addr, err := HostManager.allocOvnMappedIpAddr(ctx)
		if err != nil {
			log.Errorf("host %s(%s): alloc vpc mapped addr: %v",
				hh.Name, hh.Id, err)
		}
		if _, err := db.Update(hh, func() error {
			hh.OvnMappedIpAddr = addr
			return nil
		}); err != nil {
			log.Errorf("host %s(%s): db update vpc mapped addr: %v",
				hh.Name, hh.Id, err)
		}
	}

	keys := GetHostQuotaKeysFromCreateInput(ownerId, input)
	quota := SInfrasQuota{Host: 1}
	quota.SetKeys(keys)
	err = quotas.CancelPendingUsage(ctx, userCred, &quota, &quota, true)
	if err != nil {
		log.Errorf("CancelPendingUsage fail %s", err)
	}
	hh.SEnabledStatusInfrasResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    hh,
		Action: notifyclient.ActionCreate,
	})
}

func (hh *SHost) StartBaremetalCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	if task, err := taskman.TaskManager.NewTask(ctx, "BaremetalCreateTask", hh, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
		return nil
	}
}

func (manager *SHostManager) ValidateSizeParams(input api.HostSizeAttributes) (api.HostSizeAttributes, error) {
	memStr := input.MemSize
	if len(memStr) > 0 {
		if !regutils.MatchSize(memStr) {
			return input, errors.Wrap(httperrors.ErrInputParameter, "Memory size must be number[+unit], like 256M, 1G or 256")
		}
		memSize, err := fileutils.GetSizeMb(memStr, 'M', 1024)
		if err != nil {
			return input, errors.Wrap(err, "fileutils.GetSizeMb")
		}
		input.MemSize = strconv.FormatInt(int64(memSize), 10)
		// data.Set("mem_size", jsonutils.NewInt(int64(memSize)))
	}
	memReservedStr := input.MemReserved
	if len(memReservedStr) > 0 {
		if !regutils.MatchSize(memReservedStr) {
			return input, errors.Wrap(httperrors.ErrInputParameter, "Memory size must be number[+unit], like 256M, 1G or 256")
		}
		memSize, err := fileutils.GetSizeMb(memReservedStr, 'M', 1024)
		if err != nil {
			return input, errors.Wrap(err, "fileutils.GetSizeMb")
		}
		input.MemReserved = strconv.FormatInt(int64(memSize), 10)
		// data.Set("mem_reserved", jsonutils.NewInt(int64(memSize)))
	}
	cpuCacheStr := input.CpuCache
	if len(cpuCacheStr) > 0 {
		if !regutils.MatchSize(cpuCacheStr) {
			return input, errors.Wrapf(httperrors.ErrInputParameter, "Illegal cpu cache size %s", cpuCacheStr)
		}
		cpuCache, err := fileutils.GetSizeKb(cpuCacheStr, 'K', 1024)
		if err != nil {
			return input, errors.Wrap(err, "fileutils.GetSizeKb")
		}
		input.CpuCache = strconv.FormatInt(int64(cpuCache), 10)
		// data.Set("cpu_cache", jsonutils.NewInt(int64(cpuCache)))
	}
	return input, nil
}

func (manager *SHostManager) inputUniquenessCheck(input api.HostAccessAttributes, zoneId string, hostId string) (api.HostAccessAttributes, error) {
	for key, val := range map[string]string{
		"manager_uri": input.ManagerUri,
		"access_ip":   input.AccessIp,
	} {
		if len(val) > 0 {
			q := manager.Query().Equals(key, val)
			if len(zoneId) > 0 {
				q = q.Equals("zone_id", zoneId)
			} else {
				q = q.IsNullOrEmpty("zone_id")
			}
			if len(hostId) > 0 {
				q = q.NotEquals("id", hostId)
			}
			cnt, err := q.CountWithError()
			if err != nil {
				return input, httperrors.NewInternalServerError("check %s duplication fail %s", key, err)
			}
			if cnt > 0 {
				return input, httperrors.NewConflictError("duplicate %s %s", key, val)
			}
		}
	}

	accessMac := input.AccessMac
	if len(accessMac) > 0 {
		accessMac2 := netutils.FormatMacAddr(accessMac)
		if len(accessMac2) == 0 {
			return input, httperrors.NewInputParameterError("invalid macAddr %s", accessMac)
		}
		if accessMac2 != api.ACCESS_MAC_ANY {
			q := manager.Query().Equals("access_mac", accessMac2)
			if len(hostId) > 0 {
				q = q.NotEquals("id", hostId)
			}
			cnt, err := q.CountWithError()
			if err != nil {
				return input, httperrors.NewInternalServerError("check access_mac duplication fail %s", err)
			}
			if cnt > 0 {
				return input, httperrors.NewConflictError("duplicate access_mac %s", accessMac)
			}
			input.AccessMac = accessMac2
		}
	}
	return input, nil
}

func (manager *SHostManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.HostCreateInput,
) (api.HostCreateInput, error) {
	var err error

	if len(input.ZoneId) > 0 {
		_, input.ZoneResourceInput, err = ValidateZoneResourceInput(userCred, input.ZoneResourceInput)
		if err != nil {
			return input, errors.Wrap(err, "ValidateZoneResourceInput")
		}
	}

	noProbe := false
	if input.NoProbe != nil {
		noProbe = *input.NoProbe
	}

	input.HostAccessAttributes, err = manager.inputUniquenessCheck(input.HostAccessAttributes, input.ZoneId, "")
	if err != nil {
		return input, errors.Wrap(err, "manager.inputUniquenessCheck")
	}

	input.HostSizeAttributes, err = manager.ValidateSizeParams(input.HostSizeAttributes)
	if err != nil {
		return input, errors.Wrap(err, "manager.ValidateSizeParams")
	}

	if len(input.MemReserved) == 0 {
		if input.HostType != api.HOST_TYPE_BAREMETAL {
			memSize, _ := strconv.ParseInt(input.MemSize, 10, 64)
			memReserved := memSize / 8
			if memReserved > 4096 {
				memReserved = 4096
			}
			input.MemReserved = strconv.FormatInt(memReserved, 10)
			// data.Set("mem_reserved", jsonutils.NewInt(memReserved))
		} else {
			input.MemReserved = "0"
			// data.Set("mem_reserved", jsonutils.NewInt(0))
		}
	}

	ipmiInfo, err := fetchIpmiInfo(input.HostIpmiAttributes, "")
	if err != nil {
		return input, errors.Wrap(err, "fetchIpmiInfo")
	}
	ipmiIpAddr := ipmiInfo.IpAddr
	if len(ipmiIpAddr) == 0 {
		noProbe = true
	}
	if len(ipmiIpAddr) > 0 && !noProbe {
		net, _ := NetworkManager.GetOnPremiseNetworkOfIP(ipmiIpAddr, "", tristate.None)
		if net == nil {
			return input, httperrors.NewInputParameterError("%s is out of network IP ranges", ipmiIpAddr)
		}
		// check ip has been reserved
		rip := ReservedipManager.GetReservedIP(net, ipmiIpAddr, api.AddressTypeIPv4)
		if rip == nil {
			// if not, reserve this IP temporarily
			err := net.reserveIpWithDuration(ctx, userCred, ipmiIpAddr, "reserve for baremetal ipmi IP", 30*time.Minute)
			if err != nil {
				return input, errors.Wrap(err, "net.reserveIpWithDuration")
			}
		}
		zoneObj, _ := net.GetZone()
		if zoneObj == nil {
			return input, httperrors.NewInputParameterError("IPMI network has no zone???")
		}
		originZoneId := input.ZoneId
		if len(originZoneId) > 0 && originZoneId != zoneObj.GetId() {
			return input, httperrors.NewInputParameterError("IPMI address located in different zone than specified")
		}
		input.ZoneId = zoneObj.GetId()
		// data.Set("zone_id", jsonutils.NewString(zoneObj.GetId()))
	}
	if !noProbe || input.NoBMC {
		var accessNet *SNetwork
		accessIpAddr := input.AccessIp // tString("access_ip")
		if len(accessIpAddr) > 0 {
			net, _ := NetworkManager.GetOnPremiseNetworkOfIP(accessIpAddr, "", tristate.None)
			if net == nil {
				return input, httperrors.NewInputParameterError("%s is out of network IP ranges", accessIpAddr)
			}
			accessNet = net
		} else {
			accessNetStr := input.AccessNet // data.GetString("access_net")
			if len(accessNetStr) > 0 {
				netObj, err := NetworkManager.FetchByIdOrName(userCred, accessNetStr)
				if err != nil {
					if errors.Cause(err) == sql.ErrNoRows {
						return input, httperrors.NewResourceNotFoundError2("network", accessNetStr)
					} else {
						return input, httperrors.NewGeneralError(err)
					}
				}
				accessNet = netObj.(*SNetwork)
			} else {
				accessWireStr := input.AccessWire // data.GetString("access_wire")
				if len(accessWireStr) > 0 {
					wireObj, err := WireManager.FetchByIdOrName(userCred, accessWireStr)
					if err != nil {
						if errors.Cause(err) == sql.ErrNoRows {
							return input, httperrors.NewResourceNotFoundError2("wire", accessWireStr)
						} else {
							return input, httperrors.NewGeneralError(err)
						}
					}
					wire := wireObj.(*SWire)
					lockman.LockObject(ctx, wire)
					defer lockman.ReleaseObject(ctx, wire)
					net, err := wire.GetCandidatePrivateNetwork(userCred, userCred, NetworkManager.AllowScope(userCred), false, []string{api.NETWORK_TYPE_PXE, api.NETWORK_TYPE_BAREMETAL, api.NETWORK_TYPE_GUEST})
					if err != nil {
						return input, httperrors.NewGeneralError(err)
					}
					accessNet = net
				}
			}
		}
		if accessNet != nil {
			lockman.LockObject(ctx, accessNet)
			defer lockman.ReleaseObject(ctx, accessNet)

			accessIp, err := accessNet.GetFreeIP(ctx, userCred, nil, nil, accessIpAddr, api.IPAllocationNone, true, api.AddressTypeIPv4)
			if err != nil {
				return input, httperrors.NewGeneralError(err)
			}

			if len(accessIpAddr) > 0 && accessIpAddr != accessIp {
				return input, httperrors.NewConflictError("Access ip %s has been used", accessIpAddr)
			}

			zoneObj, _ := accessNet.GetZone()
			if zoneObj == nil {
				return input, httperrors.NewInputParameterError("Access network has no zone???")
			}
			originZoneId := input.ZoneId // data.GetString("zone_id")
			if len(originZoneId) > 0 && originZoneId != zoneObj.GetId() {
				return input, httperrors.NewInputParameterError("Access address located in different zone than specified")
			}

			// check ip has been reserved
			rip := ReservedipManager.GetReservedIP(accessNet, accessIp, api.AddressTypeIPv4)
			if rip == nil {
				// if not reserved, reserve this IP temporarily
				err = accessNet.reserveIpWithDuration(ctx, userCred, accessIp, "reserve for baremetal access IP", 30*time.Minute)
				if err != nil {
					return input, err
				}
			}

			input.AccessIp = accessIp
			input.ZoneId = zoneObj.GetId()
			// data.Set("access_ip", jsonutils.NewString(accessIp))
			// data.Set("zone_id", jsonutils.NewString(zoneObj.GetId()))
		}
	}
	// only baremetal can be created
	hostType := input.HostType // .GetString("host_type")
	if len(hostType) == 0 {
		hostType = api.HOST_TYPE_BAREMETAL
		input.HostType = hostType
		// data.Set("host_type", jsonutils.NewString(hostType))
	}
	if hostType == api.HOST_TYPE_BAREMETAL {
		isBaremetal := true
		input.IsBaremetal = &isBaremetal
		// data.Set("is_baremetal", jsonutils.JSONTrue)
	}

	if noProbe {
		// accessMac := input.AccessMac // data.GetString("access_mac")
		// uuid := input.Uuid // data.GetString("uuid")
		if len(input.AccessMac) == 0 && len(input.Uuid) == 0 {
			return input, httperrors.NewInputParameterError("missing access_mac and uuid in no_probe mode")
		}
	}

	input.EnabledStatusInfrasResourceBaseCreateInput, err = manager.SEnabledStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ValidateCreateData")
	}
	name := input.Name
	if len(name) == 0 {
		name = input.GenerateName
	}
	input.HostnameInput, err = manager.SHostnameResourceBaseManager.ValidateHostname(name, "", input.HostnameInput)
	if err != nil {
		return input, err
	}

	keys := GetHostQuotaKeysFromCreateInput(ownerId, input)
	quota := SInfrasQuota{Host: 1}
	quota.SetKeys(keys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, &quota)
	if err != nil {
		return input, errors.Wrapf(err, "CheckSetPendingQuota")
	}

	return input, nil
}

func (hh *SHost) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.HostUpdateInput) (api.HostUpdateInput, error) {
	// validate Hostname
	if len(input.Hostname) > 0 {
		if !regutils.MatchDomainName(input.Hostname) {
			return input, httperrors.NewInputParameterError("hostname should be a legal domain name")
		}
	}

	var err error
	input.HostAccessAttributes, err = HostManager.inputUniquenessCheck(input.HostAccessAttributes, hh.ZoneId, hh.Id)
	if err != nil {
		return input, errors.Wrap(err, "inputUniquenessCheck")
	}

	if hh.IsHugePage() && input.MemCmtbound != nil && *input.MemCmtbound != hh.MemCmtbound {
		return input, errors.Errorf("host mem is hugepage, cannot update mem_cmtbound")
	}

	if input.CpuReserved != nil {
		info := hh.GetMetadata(ctx, api.HOSTMETA_RESERVED_CPUS_INFO, nil)
		if len(info) > 0 {
			return input, errors.Wrap(httperrors.ErrInputParameter, "host cpu has been reserved, cannot update cpu_reserved")
		}
	}

	input.HostSizeAttributes, err = HostManager.ValidateSizeParams(input.HostSizeAttributes)
	if err != nil {
		return input, errors.Wrap(err, "ValidateSizeParams")
	}

	ipmiInfo, err := fetchIpmiInfo(input.HostIpmiAttributes, hh.Id)
	if err != nil {
		return input, errors.Wrap(err, "fetchIpmiInfo")
	}
	ipmiInfoJson := jsonutils.Marshal(ipmiInfo).(*jsonutils.JSONDict)
	if ipmiInfoJson.Length() > 0 {
		ipmiIpAddr := ipmiInfo.IpAddr
		if len(ipmiIpAddr) > 0 {
			net, _ := NetworkManager.GetOnPremiseNetworkOfIP(ipmiIpAddr, "", tristate.None)
			if net == nil {
				return input, httperrors.NewInputParameterError("%s is out of network IP ranges", ipmiIpAddr)
			}
			zoneObj, _ := net.GetZone()
			if zoneObj == nil {
				return input, httperrors.NewInputParameterError("IPMI network has not zone???")
			}
			if zoneObj.GetId() != hh.ZoneId {
				return input, httperrors.NewInputParameterError("New IPMI address located in another zone!")
			}
		}
		val := jsonutils.NewDict()
		val.Update(hh.IpmiInfo)
		val.Update(ipmiInfoJson)
		input.IpmiInfo = val
	}
	input.EnabledStatusInfrasResourceBaseUpdateInput, err = hh.SEnabledStatusInfrasResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusInfrasResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusInfrasResourceBase.ValidateUpdateData")
	}
	if len(input.Name) > 0 {
		hh.UpdateDnsRecords(false)
	}
	return input, nil
}

func (hh *SHost) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	hh.SEnabledStatusInfrasResourceBase.PostUpdate(ctx, userCred, query, data)

	if data.Contains("cpu_cmtbound") || data.Contains("mem_cmtbound") {
		hh.ClearSchedDescCache()
	}

	if hh.OvnVersion != "" && hh.OvnMappedIpAddr == "" {
		HostManager.lockAllocOvnMappedIpAddr(ctx)
		defer HostManager.unlockAllocOvnMappedIpAddr(ctx)
		addr, err := HostManager.allocOvnMappedIpAddr(ctx)
		if err != nil {
			log.Errorf("host %s(%s): alloc vpc mapped addr: %v",
				hh.Name, hh.Id, err)
			return
		}
		if _, err := db.Update(hh, func() error {
			hh.OvnMappedIpAddr = addr
			return nil
		}); err != nil {
			log.Errorf("host %s(%s): db update vpc mapped addr: %v",
				hh.Name, hh.Id, err)
			return
		}
	}

	// update baremetal host related server
	if guest := hh.GetBaremetalServer(); guest != nil && hh.HostType == api.HOST_TYPE_BAREMETAL {
		if _, err := db.Update(guest, func() error {
			guest.VmemSize = hh.MemSize
			guest.VcpuCount = hh.CpuCount
			return nil
		}); err != nil {
			log.Errorf("baremetal host %s update related server %s spec error: %v", hh.GetName(), guest.GetName(), err)
		}
	}

	notSyncConf, _ := data.Bool("not_sync_config")

	if !notSyncConf {
		if err := hh.startSyncConfig(ctx, userCred, "", true); err != nil {
			log.Errorf("start sync host %q config after updated", hh.GetName())
		}
	}
}

func (hh *SHost) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	hh.SEnabledStatusInfrasResourceBase.PostDelete(ctx, userCred)
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    hh,
		Action: notifyclient.ActionDelete,
	})
}

func (hh *SHost) UpdateDnsRecords(isAdd bool) {
	for _, netif := range hh.GetHostNetInterfaces() {
		hh.UpdateDnsRecord(&netif, isAdd)
	}
}

func (hh *SHost) UpdateDnsRecord(netif *SNetInterface, isAdd bool) {
	name := hh.GetNetifName(netif)
	if len(name) == 0 {
		return
	}
	bn := netif.GetHostNetwork()
	if bn == nil {
		log.Errorf("Interface %s not enable", netif.GetId())
		return
	}
	net := bn.GetNetwork()
	if net == nil {
		log.Errorf("BaremetalNetwoke %s not found network", bn.GetId())
	}
	net._updateDnsRecord(name, bn.IpAddr, isAdd)
}

func (hh *SHost) GetNetifName(netif *SNetInterface) string {
	if netif.NicType == api.NIC_TYPE_IPMI {
		return hh.GetName()
	} else if netif.NicType == api.NIC_TYPE_ADMIN {
		return hh.GetName() + "-admin"
	}
	return ""
}

func fetchIpmiInfo(data api.HostIpmiAttributes, hostId string) (types.SIPMIInfo, error) {
	info := types.SIPMIInfo{}
	info.Username = data.IpmiUsername
	if len(data.IpmiPassword) > 0 {
		if len(hostId) > 0 {
			value, err := utils.EncryptAESBase64(hostId, data.IpmiPassword)
			if err != nil {
				log.Errorf("encrypt password failed %s", err)
				return info, errors.Wrap(err, "utils.EncryptAESBase64")
			}
			info.Password = value
		} else {
			info.Password = data.IpmiPassword
		}
	}
	if len(data.IpmiIpAddr) > 0 && !regutils.MatchIP4Addr(data.IpmiIpAddr) {
		msg := fmt.Sprintf("ipmi_ip_addr: %s not valid ipv4 address", data.IpmiIpAddr)
		log.Errorf(msg)
		return info, errors.Wrap(httperrors.ErrInvalidFormat, msg)
	}
	info.IpAddr = data.IpmiIpAddr
	if data.IpmiPresent != nil {
		info.Present = *data.IpmiPresent
	}
	if data.IpmiLanChannel != nil {
		info.LanChannel = *data.IpmiLanChannel
	}
	if data.IpmiVerified != nil {
		info.Verified = *data.IpmiVerified
	}
	if data.IpmiRedfishApi != nil {
		info.RedfishApi = *data.IpmiRedfishApi
	}
	if data.IpmiCdromBoot != nil {
		info.CdromBoot = *data.IpmiCdromBoot
	}
	if data.IpmiPxeBoot != nil {
		info.PxeBoot = *data.IpmiPxeBoot
	}
	return info, nil
}

func (hh *SHost) PerformStart(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.HostPerformStartInput,
) (jsonutils.JSONObject, error) {
	if !hh.IsBaremetal {
		return nil, httperrors.NewBadRequestError("Cannot start a non-baremetal host")
	}
	if !utils.IsInStringArray(hh.Status, []string{api.BAREMETAL_READY}) {
		return nil, httperrors.NewInvalidStatusError("Cannot start baremetal with active guest")
	}
	guest := hh.GetBaremetalServer()
	if guest != nil {
		if hh.HostType == api.HOST_TYPE_BAREMETAL && utils.ToBool(guest.GetMetadata(ctx, "is_fake_baremetal_server", userCred)) {
			return nil, hh.InitializedGuestStart(ctx, userCred, guest)
		}
		//	if !utils.IsInStringArray(guest.Status, []string{VM_ADMIN}) {
		//		return nil, httperrors.NewBadRequestError("Cannot start baremetal with active guest")
		//	}
		hh.SetStatus(userCred, api.BAREMETAL_START_MAINTAIN, "")
		return guest.PerformStart(ctx, userCred, query, api.GuestPerformStartInput{})
	}
	params := jsonutils.NewDict()
	params.Set("force_reboot", jsonutils.NewBool(false))
	params.Set("action", jsonutils.NewString("start"))
	return hh.PerformMaintenance(ctx, userCred, nil, params)
}

func (hh *SHost) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if hh.GetEnabled() {
		return nil, httperrors.NewInvalidStatusError("Host is not disabled")
	}
	return nil, hh.purge(ctx, userCred)
}

func (hh *SHost) PerformStop(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !hh.IsBaremetal {
		return nil, httperrors.NewBadRequestError("Cannot stop a non-baremetal host")
	}
	if !utils.IsInStringArray(hh.Status, []string{api.BAREMETAL_RUNNING}) {
		return nil, httperrors.NewInvalidStatusError("Cannot stop baremetal with non-active guest")
	}
	guest := hh.GetBaremetalServer()
	if guest != nil {
		if hh.HostType != api.HOST_TYPE_BAREMETAL {
			if !utils.IsInStringArray(guest.Status, []string{api.VM_ADMIN}) {
				return nil, httperrors.NewBadRequestError("Cannot stop baremetal with active guest")
			}
		} else {
			if utils.ToBool(guest.GetMetadata(ctx, "is_fake_baremetal_server", userCred)) {
				return nil, hh.InitializedGuestStop(ctx, userCred, guest)
			}
			hh.SetStatus(userCred, api.BAREMETAL_START_MAINTAIN, "")
			input := api.ServerStopInput{}
			data.Unmarshal(&input)
			return guest.PerformStop(ctx, userCred, query, input)
		}
	}
	return nil, hh.StartBaremetalUnmaintenanceTask(ctx, userCred, false, "stop")
}

func (hh *SHost) InitializedGuestStart(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest) error {
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalServerStartTask", guest, userCred, nil, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (hh *SHost) InitializedGuestStop(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest) error {
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalServerStopTask", guest, userCred, nil, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (hh *SHost) PerformMaintenance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(hh.Status, []string{api.BAREMETAL_READY, api.BAREMETAL_RUNNING}) {
		return nil, httperrors.NewInvalidStatusError("Cannot do maintenance in status %s", hh.Status)
	}
	guest := hh.GetBaremetalServer()
	if guest != nil && !utils.IsInStringArray(guest.Status, []string{api.VM_READY, api.VM_RUNNING, api.VM_ADMIN}) {
		return nil, httperrors.NewInvalidStatusError("Cannot do maintenance while guest status %s", guest.Status)
	}
	params := jsonutils.NewDict()
	if guest != nil {
		if guest.Status == api.VM_RUNNING {
			params.Set("guest_running", jsonutils.NewBool(true))
		}
		guest.SetStatus(userCred, api.VM_ADMIN, "")
	}
	if hh.Status == api.BAREMETAL_RUNNING && jsonutils.QueryBoolean(data, "force_reboot", false) {
		params.Set("force_reboot", jsonutils.NewBool(true))
	}
	action := "maintenance"
	if data.Contains("action") {
		action, _ = data.GetString("action")
	}
	params.Set("action", jsonutils.NewString(action))
	hh.SetStatus(userCred, api.BAREMETAL_START_MAINTAIN, "")
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalMaintenanceTask", hh, userCred, params, "", "", nil)
	if err != nil {
		return nil, err
	}
	task.ScheduleRun(nil)
	return nil, nil
}

func (hh *SHost) PerformUnmaintenance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(hh.Status, []string{api.BAREMETAL_RUNNING, api.BAREMETAL_READY}) {
		return nil, httperrors.NewInvalidStatusError("Cannot do unmaintenance in status %s", hh.Status)
	}
	guest := hh.GetBaremetalServer()
	if guest != nil && guest.Status != api.VM_ADMIN {
		return nil, httperrors.NewInvalidStatusError("Wrong guest status %s", guest.Status)
	}
	action, _ := data.GetString("action")
	if len(action) == 0 {
		action = "unmaintenance"
	}
	guestRunning := hh.GetMetadata(ctx, "__maint_guest_running", userCred)
	var startGuest = false
	if utils.ToBool(guestRunning) {
		startGuest = true
	}
	return nil, hh.StartBaremetalUnmaintenanceTask(ctx, userCred, startGuest, action)
}

func (hh *SHost) StartBaremetalUnmaintenanceTask(ctx context.Context, userCred mcclient.TokenCredential, startGuest bool, action string) error {
	hh.SetStatus(userCred, api.BAREMETAL_START_MAINTAIN, "")
	params := jsonutils.NewDict()
	params.Set("guest_running", jsonutils.NewBool(startGuest))
	if len(action) == 0 {
		action = "unmaintenance"
	}
	params.Set("action", jsonutils.NewString(action))
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalUnmaintenanceTask", hh, userCred, params, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (hh *SHost) StartSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	guest := hh.GetBaremetalServer()
	if guest != nil {
		return guest.StartSyncstatus(ctx, userCred, parentTaskId)
	}
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalSyncStatusTask", hh, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (hh *SHost) PerformOffline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.HostOfflineInput) (jsonutils.JSONObject, error) {
	if hh.HostStatus != api.HOST_OFFLINE {
		_, err := hh.SaveUpdates(func() error {
			hh.HostStatus = api.HOST_OFFLINE
			if input.UpdateHealthStatus != nil && *input.UpdateHealthStatus {
				hh.EnableHealthCheck = false
			}
			// Note: update host status to unknown on host offline
			// we did not have host status after host offline
			hh.Status = api.BAREMETAL_UNKNOWN
			return nil
		})
		if err != nil {
			return nil, err
		}
		db.OpsLog.LogEvent(hh, db.ACT_OFFLINE, input.Reason, userCred)
		logclient.AddActionLogWithContext(ctx, hh, logclient.ACT_OFFLINE, input, userCred, true)
		ndata := jsonutils.Marshal(hh).(*jsonutils.JSONDict)
		if len(input.Reason) > 0 {
			ndata.Add(jsonutils.NewString(input.Reason), "reason")
		}
		notifyclient.SystemExceptionNotify(ctx, napi.ActionOffline, HostManager.Keyword(), ndata)
		hh.SyncAttachedStorageStatus()
	}
	return nil, nil
}

func (hh *SHost) PerformOnline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if hh.HostStatus != api.HOST_ONLINE {
		_, err := hh.SaveUpdates(func() error {
			hh.LastPingAt = time.Now()
			hh.HostStatus = api.HOST_ONLINE
			hh.EnableHealthCheck = true
			if !hh.IsMaintaining() {
				hh.Status = api.BAREMETAL_RUNNING
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if hostHealthChecker != nil {
			hostHealthChecker.WatchHost(context.Background(), hh.GetHostnameByName())
		}
		db.OpsLog.LogEvent(hh, db.ACT_ONLINE, "", userCred)
		logclient.AddActionLogWithContext(ctx, hh, logclient.ACT_ONLINE, data, userCred, true)
		hh.SyncAttachedStorageStatus()
		hh.StartSyncAllGuestsStatusTask(ctx, userCred)
	}
	return nil, nil
}

func (hh *SHost) PerformRestartHostAgent(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	_, err := hh.Request(ctx, userCred, "POST", fmt.Sprintf("/hosts/%s/restart-host-agent", hh.Id),
		mcclient.GetTokenHeaders(userCred), data)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (hh *SHost) PerformAutoMigrateOnHostDown(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.HostAutoMigrateInput,
) (jsonutils.JSONObject, error) {
	if input.AutoMigrateOnHostShutdown == "enable" &&
		input.AutoMigrateOnHostDown != "enable" {
		return nil, httperrors.NewBadRequestError("must enable auto_migrate_on_host_down at same time")
	}

	var meta = make(map[string]interface{})
	if input.AutoMigrateOnHostShutdown == "enable" {
		meta[api.HOSTMETA_AUTO_MIGRATE_ON_HOST_SHUTDOWN] = "enable"
	} else if input.AutoMigrateOnHostShutdown == "disable" {
		meta[api.HOSTMETA_AUTO_MIGRATE_ON_HOST_SHUTDOWN] = "disable"
	}

	data := jsonutils.NewDict()
	if input.AutoMigrateOnHostDown == "enable" {
		data.Set("shutdown_servers", jsonutils.JSONTrue)
		meta[api.HOSTMETA_AUTO_MIGRATE_ON_HOST_DOWN] = "enable"
	} else if input.AutoMigrateOnHostDown == "disable" {
		meta[api.HOSTMETA_AUTO_MIGRATE_ON_HOST_DOWN] = "disable"
		data.Set("shutdown_servers", jsonutils.JSONFalse)
	}
	_, err := hh.Request(ctx, userCred, "POST", fmt.Sprintf("/hosts/%s/shutdown-servers-on-host-down", hh.Id),
		mcclient.GetTokenHeaders(userCred), data)
	if err != nil {
		return nil, err
	}

	return nil, hh.SetAllMetadata(ctx, meta, userCred)
}

func (hh *SHost) StartSyncAllGuestsStatusTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	if task, err := taskman.TaskManager.NewTask(ctx, "BaremetalSyncAllGuestsStatusTask", hh, userCred, nil, "", "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
		return nil
	}
}

func (hh *SHost) PerformPing(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SHostPingInput) (jsonutils.JSONObject, error) {
	if hh.HostType == api.HOST_TYPE_BAREMETAL {
		return nil, httperrors.NewNotSupportedError("ping host type %s not support", hh.HostType)
	}
	if input.WithData {
		// piggyback storage stats info
		log.Debugf("host ping %s", jsonutils.Marshal(input))
		for _, si := range input.StorageStats {
			storageObj, err := StorageManager.FetchById(si.StorageId)
			if err != nil {
				log.Errorf("fetch storage %s error %s", si.StorageId, err)
			} else {
				storage := storageObj.(*SStorage)
				_, err := db.Update(storage, func() error {
					storage.Capacity = si.CapacityMb
					storage.ActualCapacityUsed = si.ActualCapacityUsedMb
					return nil
				})
				if err != nil {
					log.Errorf("update storage info error %s", err)
				}
			}
		}
		hh.SetMetadata(ctx, "root_partition_used_capacity_mb", input.RootPartitionUsedCapacityMb, userCred)
		hh.SetMetadata(ctx, "memory_used_mb", input.MemoryUsedMb, userCred)
	}
	if hh.HostStatus != api.HOST_ONLINE {
		hh.PerformOnline(ctx, userCred, query, nil)
	} else {
		hh.SaveUpdates(func() error {
			hh.LastPingAt = time.Now()
			return nil
		})
	}
	result := jsonutils.NewDict()
	result.Set("name", jsonutils.NewString(hh.GetName()))
	dependSvcs := []string{"ntpd", "kafka", apis.SERVICE_TYPE_INFLUXDB, apis.SERVICE_TYPE_VICTORIA_METRICS, "elasticsearch"}
	catalog := auth.GetCatalogData(dependSvcs, options.Options.Region)
	if catalog == nil {
		return nil, fmt.Errorf("Get catalog error")
	}
	result.Set("catalog", catalog)

	appParams := appsrv.AppContextGetParams(ctx)
	if appParams != nil {
		// skip log&trace, when everything is normal
		appParams.SkipTrace = true
		appParams.SkipLog = true
	}

	return result, nil
}

func (host *SHost) getHostLogicalCores() ([]int, error) {
	cpuObj, err := host.SysInfo.Get("cpu_info")
	if err != nil {
		return nil, errors.Wrap(err, "get cpu info from host sys_info")
	}
	cpuInfo := new(hostapi.HostCPUInfo)
	if err := cpuObj.Unmarshal(cpuInfo); err != nil {
		return nil, errors.Wrap(err, "Unmarshal host cpu info struct")
	}

	// get host logical cores
	allCores := []int{}

	if len(cpuInfo.Processors) != 0 {
		for _, p := range cpuInfo.Processors {
			for _, core := range p.Cores {
				allCores = append(allCores, core.LogicalProcessors...)
			}
		}
		sort.Ints(allCores)
	} else {
		topoObj, err := host.SysInfo.Get("topology")
		if err != nil {
			return nil, errors.Wrap(err, "get topology from host sys_info")
		}

		hostTopo := new(hostapi.HostTopology)
		if err := topoObj.Unmarshal(hostTopo); err != nil {
			return nil, errors.Wrap(err, "Unmarshal host topology struct")
		}

		for _, node := range hostTopo.Nodes {
			for _, cores := range node.Cores {
				allCores = append(allCores, cores.LogicalProcessors...)
			}
		}
	}
	return allCores, nil
}

func (hh *SHost) PerformUnreserveCpus(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	return nil, hh.RemoveMetadata(ctx, api.HOSTMETA_RESERVED_CPUS_INFO, userCred)
}

func (hh *SHost) PerformReserveCpus(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, input api.HostReserveCpusInput,
) (jsonutils.JSONObject, error) {
	if hh.HostType != api.HOST_TYPE_HYPERVISOR {
		return nil, httperrors.NewNotSupportedError("host type %s not support reserve cpus", hh.HostType)
	}

	cnt, err := hh.GetRunningGuestCount()
	if err != nil {
		return nil, err
	}
	if cnt > 0 {
		return nil, httperrors.NewBadRequestError("host %s has %d guests, can't update reserve cpus", hh.Id, cnt)
	}

	if input.Cpus == "" {
		return nil, httperrors.NewInputParameterError("missing cpus")
	}
	cs, err := cpuset.Parse(input.Cpus)
	if err != nil {
		return nil, httperrors.NewInputParameterError("cpus %s not valid", input.Cpus)
	}

	allCores, err := hh.getHostLogicalCores()
	if err != nil {
		return nil, err
	}

	hSets := sets.NewInt(allCores...)
	cSlice := cs.ToSlice()
	if !hSets.HasAll(cSlice...) {
		return nil, httperrors.NewInputParameterError("Host cores not contains input %v", input.Cpus)
	}
	if hSets.Len() == len(cSlice) {
		return nil, httperrors.NewInputParameterError("Can't reserve host all cpus")
	}

	if input.Mems != "" {
		mems, err := cpuset.Parse(input.Mems)
		if err != nil {
			return nil, httperrors.NewInputParameterError("mems %s not valid", input.Mems)
		}
		// to slice will sort slice default
		memSlice := mems.ToSlice()
		if 0 > memSlice[len(memSlice)-1] || memSlice[len(memSlice)-1] >= int(hh.NodeCount) {
			return nil, httperrors.NewInputParameterError("mems %s out of range", input.Mems)
		}
	}

	err = hh.SetMetadata(ctx, api.HOSTMETA_RESERVED_CPUS_INFO, input, userCred)
	if err != nil {
		return nil, err
	}
	if hh.CpuReserved < cs.Size() {
		_, err = db.Update(hh, func() error {
			hh.CpuReserved = cs.Size()
			return nil
		})
	}
	return nil, err
}

func (hh *SHost) HasBMC() bool {
	ipmiInfo, _ := hh.GetIpmiInfo()
	if ipmiInfo.Username != "" && ipmiInfo.Password != "" {
		return true
	}
	return false
}

func (hh *SHost) IsUEFIBoot() bool {
	info, _ := hh.GetUEFIInfo()
	if info == nil {
		return false
	}
	if len(info.PxeBootNum) == 0 {
		return false
	}
	return true
}

func (hh *SHost) isRedfishCapable() bool {
	ipmiInfo, _ := hh.GetIpmiInfo()
	if ipmiInfo.Verified && ipmiInfo.RedfishApi {
		return true
	}
	return false
}

func (hh *SHost) canPrepare() error {
	if !hh.IsBaremetal {
		return httperrors.NewInvalidStatusError("not a baremetal")
	}
	if !hh.isRedfishCapable() && len(hh.AccessMac) == 0 && len(hh.Uuid) == 0 {
		return httperrors.NewInvalidStatusError("need valid access_mac and uuid to do prepare")
	}
	if !utils.IsInStringArray(hh.Status, []string{api.BAREMETAL_READY, api.BAREMETAL_RUNNING, api.BAREMETAL_PREPARE_FAIL}) {
		return httperrors.NewInvalidStatusError("Cannot prepare baremetal in status %s", hh.Status)
	}
	server := hh.GetBaremetalServer()
	if server != nil && server.Status != api.VM_ADMIN {
		return httperrors.NewInvalidStatusError("Cannot prepare baremetal in server status %s", server.Status)
	}
	return nil
}

func (hh *SHost) PerformPrepare(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := hh.canPrepare()
	if err != nil {
		return nil, err
	}
	var onfinish string
	server := hh.GetBaremetalServer()
	if server != nil && hh.Status == api.BAREMETAL_READY {
		onfinish = "shutdown"
	}
	return nil, hh.StartPrepareTask(ctx, userCred, onfinish, "")
}

func (hh *SHost) StartPrepareTask(ctx context.Context, userCred mcclient.TokenCredential, onfinish, parentTaskId string) error {
	data := jsonutils.NewDict()
	if len(onfinish) > 0 {
		data.Set("on_finish", jsonutils.NewString(onfinish))
	}
	hh.SetStatus(userCred, api.BAREMETAL_PREPARE, "start prepare task")
	if task, err := taskman.TaskManager.NewTask(ctx, "BaremetalPrepareTask", hh, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
		return nil
	}
}

func (hh *SHost) PerformIpmiProbe(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(hh.Status, []string{api.BAREMETAL_INIT, api.BAREMETAL_READY, api.BAREMETAL_RUNNING, api.BAREMETAL_PROBE_FAIL, api.BAREMETAL_UNKNOWN}) {
		return nil, hh.StartIpmiProbeTask(ctx, userCred, "")
	}
	return nil, httperrors.NewInvalidStatusError("Cannot do Ipmi-probe in status %s", hh.Status)
}

func (hh *SHost) StartIpmiProbeTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	data := jsonutils.NewDict()
	hh.SetStatus(userCred, api.BAREMETAL_START_PROBE, "start ipmi-probe task")
	if task, err := taskman.TaskManager.NewTask(ctx, "BaremetalIpmiProbeTask", hh, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
		return nil
	}
}

func (hh *SHost) PerformInitialize(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(
		hh.Status, []string{api.BAREMETAL_INIT, api.BAREMETAL_PREPARE_FAIL}) {
		return nil, httperrors.NewBadRequestError(
			"Cannot do initialization in status %s", hh.Status)
	}

	name, err := data.GetString("name")
	if err != nil || hh.GetBaremetalServer() != nil {
		return nil, nil
	}
	err = db.NewNameValidator(GuestManager, userCred, name, nil)
	if err != nil {
		return nil, err
	}

	if hh.IpmiInfo == nil || !hh.IpmiInfo.Contains("ip_addr") ||
		!hh.IpmiInfo.Contains("password") {
		return nil, httperrors.NewBadRequestError("IPMI infomation not configured")
	}
	guest := &SGuest{}
	guest.Name = name
	guest.VmemSize = hh.MemSize
	guest.VcpuCount = hh.CpuCount
	guest.DisableDelete = tristate.True
	guest.Hypervisor = api.HYPERVISOR_BAREMETAL
	guest.HostId = hh.Id
	guest.ProjectId = userCred.GetProjectId()
	guest.DomainId = userCred.GetProjectDomainId()
	guest.Status = api.VM_RUNNING
	guest.OsType = "Linux"
	guest.SetModelManager(GuestManager, guest)
	err = GuestManager.TableSpec().Insert(ctx, guest)
	if err != nil {
		return nil, httperrors.NewInternalServerError("Guest Insert error: %s", err)
	}
	guest.SetAllMetadata(ctx, map[string]interface{}{
		"is_fake_baremetal_server": true, "host_ip": hh.AccessIp}, userCred)

	caps := hh.GetAttachedLocalStorageCapacity()
	diskConfig := &api.DiskConfig{SizeMb: int(caps.GetFree())}
	err = guest.CreateDisksOnHost(ctx, userCred, hh, []*api.DiskConfig{diskConfig}, nil, true, true, nil, nil, true)
	if err != nil {
		log.Errorf("Host perform initialize failed on create disk %s", err)
	}
	net, err := hh.getNetworkOfIPOnHost(hh.AccessIp)
	if err != nil {
		log.Errorf("host perfrom initialize failed fetch net of access ip %s", err)
	} else {
		if options.Options.BaremetalServerReuseHostIp {
			_, err = guest.attach2NetworkDesc(ctx, userCred, hh, &api.NetworkConfig{Network: net.Id}, nil, nil)
			if err != nil {
				log.Errorf("host perform initialize failed on attach network %s", err)
			}
		}
	}
	return nil, nil
}

func validateHostNetif(input api.HostNetifInput) (api.HostNetifInput, error) {
	mac := input.Mac
	if len(input.Mac) > 0 {
		mac = netutils.FormatMacAddr(input.Mac)
	}
	if len(mac) == 0 {
		return input, errors.Wrapf(httperrors.ErrInputParameter, "Invaild mac address %s", input.Mac)
	}
	input.Mac = mac
	vlan := input.VlanId
	if vlan == 0 {
		vlan = 1
	}
	if vlan < 0 || vlan > 4095 {
		return input, errors.Wrapf(httperrors.ErrInputParameter, "Invalid vlan_id %d", input.VlanId)
	}
	input.VlanId = vlan
	return input, nil
}

func (h *SHost) PerformAddNetif(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.HostAddNetifInput,
) (jsonutils.JSONObject, error) {
	log.Debugf("add_netif %s", jsonutils.Marshal(input))
	var err error
	input.HostNetifInput, err = validateHostNetif(input.HostNetifInput)
	if err != nil {
		return nil, errors.Wrap(err, "validateHostNetif")
	}
	mac := input.Mac
	vlan := input.VlanId

	wire := input.WireId
	if len(input.WireId) > 0 {
		wireObj, err := WireManager.FetchByIdOrName(userCred, input.WireId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(WireManager.Keyword(), input.WireId)
			} else {
				return nil, errors.Wrap(err, "FetchByIdOrName")
			}
		}
		wire = wireObj.GetId()
	}
	ipAddr := input.IpAddr
	if len(ipAddr) > 0 && !regutils.MatchIP4Addr(ipAddr) {
		return nil, errors.Wrapf(httperrors.ErrInputParameter, "invalid ip_addr %s", ipAddr)
	}
	rate := input.Rate
	nicType := input.NicType
	index := input.Index
	linkUp := input.LinkUp
	mtu := input.Mtu
	reset := (input.Reset != nil && *input.Reset)
	netIf := input.Interface
	bridge := input.Bridge
	reserve := (input.Reserve != nil && *input.Reserve)
	requireDesignatedIp := (input.RequireDesignatedIp != nil && *input.RequireDesignatedIp)

	isLinkUp := tristate.None
	if linkUp != "" {
		if utils.ToBool(linkUp) {
			isLinkUp = tristate.True
		} else {
			isLinkUp = tristate.False
		}
	}

	err = h.addNetif(ctx, userCred, mac, vlan, wire, ipAddr, int(rate), nicType, int8(index), isLinkUp,
		int16(mtu), reset, netIf, bridge, reserve, requireDesignatedIp)
	return nil, errors.Wrap(err, "addNetif")
}

func (h *SHost) addNetif(ctx context.Context, userCred mcclient.TokenCredential,
	mac string, vlanId int, wire string, ipAddr string,
	rate int, nicType compute.TNicType, index int8, linkUp tristate.TriState, mtu int16,
	reset bool, strInterface *string, strBridge *string,
	reserve bool, requireDesignatedIp bool,
) error {
	var sw *SWire
	if len(wire) > 0 {
		iWire, err := WireManager.FetchByIdOrName(userCred, wire)
		if err != nil {
			if err == sql.ErrNoRows {
				return httperrors.NewResourceNotFoundError2(WireManager.Keyword(), wire)
			} else {
				return httperrors.NewInternalServerError("find Wire %s error: %s", wire, err)
			}
		}
		sw = iWire.(*SWire)
		if len(ipAddr) > 0 {
			iIpAddr, err := netutils.NewIPV4Addr(ipAddr)
			if err != nil {
				return httperrors.NewInputParameterError("invalid ipaddr %s", ipAddr)
			}
			findAddr := false
			swNets, err := sw.getNetworks(userCred, userCred, NetworkManager.AllowScope(userCred))
			if err != nil {
				return httperrors.NewInputParameterError("no networks on wire %s", wire)
			}
			for i := range swNets {
				if swNets[i].IsAddressInRange(iIpAddr) {
					findAddr = true
					break
				}
			}
			if !findAddr {
				return httperrors.NewBadRequestError("IP %s not attach to wire %s", ipAddr, wire)
			}
		}
	} else if len(ipAddr) > 0 && len(wire) == 0 {
		ipWire, err := WireManager.GetOnPremiseWireOfIp(ipAddr)
		if err != nil {
			return httperrors.NewBadRequestError("IP %s not attach to any wire", ipAddr)
		}
		sw = ipWire
	}
	netif, err := NetInterfaceManager.FetchByMacVlan(mac, vlanId)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return httperrors.NewInternalServerError("fail to fetch netif by mac %s: %s", mac, err)
		}
		// else not found
		netif = &SNetInterface{}
		netif.SetModelManager(NetInterfaceManager, netif)
		netif.Mac = mac
		netif.VlanId = vlanId
	}
	var changed bool
	if netif.BaremetalId != h.Id {
		if len(netif.BaremetalId) > 0 {
			changed = true
			// previously conencted to another host
		}
		netif.BaremetalId = h.Id
	}
	if sw != nil && netif.WireId != sw.Id {
		if len(netif.WireId) > 0 {
			changed = true
		}
		netif.WireId = sw.Id
	} else if netif.WireId != "" && sw == nil {
		changed = true
		netif.WireId = ""
	}
	if rate > 0 && rate != netif.Rate {
		netif.Rate = rate
	}
	if nicType != "" && nicType != netif.NicType {
		netif.NicType = nicType
	}
	if index >= 0 && index != netif.Index {
		netif.Index = index
	}
	if !linkUp.IsNone() && linkUp.Bool() != netif.LinkUp {
		netif.LinkUp = linkUp.Bool()
	}
	if mtu > 0 && mtu != netif.Mtu {
		netif.Mtu = mtu
	}
	if strInterface != nil {
		netif.Interface = *strInterface
	}
	if strBridge != nil {
		netif.Bridge = *strBridge
	}
	err = NetInterfaceManager.TableSpec().InsertOrUpdate(ctx, netif)
	if err != nil {
		return errors.Wrap(err, "InsertOrUpdate")
	}
	if changed || reset {
		h.DisableNetif(ctx, userCred, netif, false)
	}
	if netif.NicType == api.NIC_TYPE_ADMIN {
		oldadmins := h.GetAdminNetInterfaces()
		for i := range oldadmins {
			oldNetif := oldadmins[i]
			if oldNetif.Mac != netif.Mac || oldNetif.VlanId != netif.VlanId {
				// make normal netif
				err := oldNetif.setNicType(api.NIC_TYPE_NORMAL)
				if err != nil {
					return errors.Wrapf(err, "setNicType %s", oldNetif.String())
				}
			}
		}
		err := h.setAccessMac(userCred, netif.Mac)
		if err != nil {
			return errors.Wrap(err, "setAccessMac")
		}
		// inherit wire's class metadata
		sw = netif.GetWire()
		if sw != nil {
			err := db.InheritFromTo(ctx, userCred, sw, h)
			if err != nil {
				return errors.Wrapf(err, "unable to inherit class metadata from sw %s", sw.GetName())
			}
		}
	}
	if len(ipAddr) > 0 {
		err = h.EnableNetif(ctx, userCred, netif, "", ipAddr, "", "", reserve, requireDesignatedIp)
		if err != nil {
			return httperrors.NewBadRequestError("%v", err)
		}
	}
	h.ClearSchedDescCache()
	return nil
}

func (h *SHost) PerformEnableNetif(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.HostEnableNetifInput,
) (jsonutils.JSONObject, error) {
	log.Debugf("enable_netif %s", jsonutils.Marshal(input))
	var err error
	input.HostNetifInput, err = validateHostNetif(input.HostNetifInput)
	if err != nil {
		return nil, errors.Wrap(err, "validateHostNetif")
	}

	netif := h.GetNetInterface(input.Mac, input.VlanId)
	if netif == nil {
		return nil, httperrors.NewBadRequestError("Interface %s(vlan:%d) not exist", input.Mac, input.VlanId)
	}
	// if netif.NicType ! !utils.IsInArray(netif.NicType, api.NIC_TYPES) {
	//	return nil, httperrors.NewBadRequestError("Only ADMIN and IPMI nic can be enable")
	// }

	reserve := (input.Reserve != nil && *input.Reserve)
	requireDesignatedIp := (input.RequireDesignatedIp != nil && *input.RequireDesignatedIp)

	err = h.EnableNetif(ctx, userCred, netif, input.NetworkId, input.IpAddr, input.AllocDir, input.NetType, reserve, requireDesignatedIp)
	if err != nil {
		return nil, httperrors.NewBadRequestError("%v", err)
	}
	return nil, nil
}

func (h *SHost) EnableNetif(ctx context.Context, userCred mcclient.TokenCredential, netif *SNetInterface,
	network, ipAddr, allocDir string, netType string, reserve, requireDesignatedIp bool) error {
	bn := netif.GetHostNetwork()
	if bn != nil {
		log.Debugf("Netif has been attach2network? %s", jsonutils.Marshal(bn))
		return nil
	}
	var net *SNetwork
	var err error
	if len(ipAddr) > 0 {
		net, err = netif.GetCandidateNetworkForIp(userCred, userCred, NetworkManager.AllowScope(userCred), ipAddr)
		if net != nil {
			log.Infof("find network %s for ip %s", net.GetName(), ipAddr)
		} else if requireDesignatedIp {
			log.Errorf("Cannot allocate IP %s, not reachable", ipAddr)
			return fmt.Errorf("Cannot allocate IP %s, not reachable", ipAddr)
		} else {
			// the ipaddr is not usable, should be reset to empty
			ipAddr = ""
		}
	}
	wire := netif.GetWire()
	if wire == nil {
		return fmt.Errorf("No wire attached")
	}
	if h.ZoneId == "" {
		if _, err := h.SaveUpdates(func() error {
			h.ZoneId = wire.ZoneId
			return nil
		}); err != nil {
			return errors.Wrapf(err, "set host zone_id %s by wire", wire.ZoneId)
		}
	}
	if net == nil {
		if len(network) > 0 {
			iNet, err := NetworkManager.FetchByIdOrName(userCred, network)
			if err != nil {
				return fmt.Errorf("Network %s not found: %s", network, err)
			}
			net = iNet.(*SNetwork)
			if len(net.WireId) == 0 || net.WireId != wire.Id {
				return fmt.Errorf("Network %s not reacheable on mac %s", network, netif.Mac)
			}
		} else {
			var netTypes []string
			if len(netType) > 0 && netType != api.NETWORK_TYPE_BAREMETAL {
				netTypes = []string{netType, api.NETWORK_TYPE_BAREMETAL}
			} else {
				netTypes = []string{api.NETWORK_TYPE_BAREMETAL}
			}
			net, err = wire.GetCandidatePrivateNetwork(userCred, userCred, NetworkManager.AllowScope(userCred), false, netTypes)
			if err != nil {
				return fmt.Errorf("fail to find private network %s", err)
			}
			if net == nil {
				net, err = wire.GetCandidateAutoAllocNetwork(userCred, userCred, NetworkManager.AllowScope(userCred), false, netTypes)
				if err != nil {
					return fmt.Errorf("fail to find public network %s", err)
				}
				if net == nil {
					return fmt.Errorf("No network found")
				}
			}
		}
	} else if net.WireId != wire.Id {
		return fmt.Errorf("conflict??? candiate net is not on wire")
	}

	attachOpt := &hostAttachNetworkOption{
		netif:               netif,
		net:                 net,
		ipAddr:              ipAddr,
		allocDir:            allocDir,
		reserved:            reserve,
		requireDesignatedIp: requireDesignatedIp,
	}

	bn, err = h.Attach2Network(ctx, userCred, attachOpt)
	if err != nil {
		return errors.Wrap(err, "hh.Attach2Network")
	}
	switch netif.NicType {
	case api.NIC_TYPE_IPMI:
		err = h.setIpmiIp(userCred, bn.IpAddr)
		if err != nil {
			return errors.Wrap(err, "setIpmiIp")
		}
	case api.NIC_TYPE_ADMIN:
		err = h.setAccessIp(userCred, bn.IpAddr)
		if err != nil {
			return errors.Wrap(err, "setAccessIp")
		}
	}
	return nil
}

func (hh *SHost) PerformDisableNetif(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.HostDisableNetifInput,
) (jsonutils.JSONObject, error) {
	var err error
	input.HostNetifInput, err = validateHostNetif(input.HostNetifInput)
	if err != nil {
		return nil, errors.Wrap(err, "validateHostNetif")
	}
	netif := hh.GetNetInterface(input.Mac, input.VlanId)
	if netif == nil {
		return nil, httperrors.NewBadRequestError("Interface %s(vlan:%d) not exists", input.Mac, input.VlanId)
	}
	reserve := (input.Reserve != nil && *input.Reserve)
	err = hh.DisableNetif(ctx, userCred, netif, reserve)
	if err != nil {
		return nil, httperrors.NewBadRequestError("%v", err)
	}
	return nil, nil
}

/*
 * Disable a net interface, remove IP address if assigned
 */
func (hh *SHost) DisableNetif(ctx context.Context, userCred mcclient.TokenCredential, netif *SNetInterface, reserve bool) error {
	bn := netif.GetHostNetwork()
	var ipAddr string
	if bn != nil {
		ipAddr = bn.IpAddr
		hh.UpdateDnsRecord(netif, false)
		hh.DeleteBaremetalnetwork(ctx, userCred, bn, reserve)
	}
	var err error
	switch netif.NicType {
	case api.NIC_TYPE_IPMI:
		if ipAddr == hh.IpmiIp {
			err = hh.setIpmiIp(userCred, "")
		}
	case api.NIC_TYPE_ADMIN:
		if ipAddr == hh.AccessIp {
			err = hh.setAccessIp(userCred, "")
		}
	}
	return err
}

type hostAttachNetworkOption struct {
	netif               *SNetInterface
	net                 *SNetwork
	ipAddr              string
	allocDir            string
	reserved            bool
	requireDesignatedIp bool
}

func (hh *SHost) IsIpAddrWithinConvertedGuest(ctx context.Context, userCred mcclient.TokenCredential, ipAddr string, netif *SNetInterface) error {
	if !hh.IsBaremetal {
		return httperrors.NewNotAcceptableError("Not a baremetal")
	}

	if hh.HostType == api.HOST_TYPE_KVM {
		return httperrors.NewNotAcceptableError("Not being convert to hypervisor")
	}

	bmServer := hh.GetBaremetalServer()
	if bmServer == nil {
		return httperrors.NewNotAcceptableError("Not found baremetal server record")
	}

	guestNics, err := bmServer.GetNetworks("")
	if err != nil {
		return errors.Wrap(err, "Get guest networks")
	}
	var findNic *SGuestnetwork
	for idx := range guestNics {
		nic := guestNics[idx]
		if nic.MacAddr == netif.Mac {
			findNic = &nic
			break
		}
	}
	if findNic == nil {
		return httperrors.NewNotFoundError("Not found guest nic by mac %s", netif.Mac)
	}

	if findNic.IpAddr != ipAddr {
		return httperrors.NewNotAcceptableError("Guest nic ip addr %s not equal %s", findNic.IpAddr, ipAddr)
	}

	return nil
}

func (hh *SHost) Attach2Network(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	opt *hostAttachNetworkOption,
) (*SHostnetwork, error) {
	netif := opt.netif
	net := opt.net
	ipAddr := opt.ipAddr
	allocDir := opt.allocDir
	reserved := opt.reserved
	requireDesignatedIp := opt.requireDesignatedIp

	lockman.LockObject(ctx, net)
	defer lockman.ReleaseObject(ctx, net)

	usedAddrs := net.GetUsedAddresses()
	if ipAddr != "" {
		// converted baremetal can resuse related guest network ip
		if err := hh.IsIpAddrWithinConvertedGuest(ctx, userCred, ipAddr, netif); err == nil {
			// force remove used server addr for reuse
			delete(usedAddrs, ipAddr)
		} else {
			log.Warningf("check IsIpAddrWithinConvertedGuest: %v", err)
		}
	}

	freeIp, err := net.GetFreeIP(ctx, userCred, usedAddrs, nil, ipAddr, api.IPAllocationDirection(allocDir), reserved, api.AddressTypeIPv4)
	if err != nil {
		return nil, errors.Wrap(err, "net.GetFreeIP")
	}
	if len(ipAddr) > 0 && ipAddr != freeIp && requireDesignatedIp {
		return nil, fmt.Errorf("IP address %s is occupied, get %s instead", ipAddr, freeIp)
	}
	bn := &SHostnetwork{}
	bn.BaremetalId = hh.Id
	bn.SetModelManager(HostnetworkManager, bn)
	bn.NetworkId = net.Id
	bn.IpAddr = freeIp
	bn.MacAddr = netif.Mac
	err = HostnetworkManager.TableSpec().Insert(ctx, bn)
	if err != nil {
		return nil, errors.Wrap(err, "HostnetworkManager.TableSpec().Insert")
	}
	db.OpsLog.LogAttachEvent(ctx, hh, net, userCred, jsonutils.NewString(freeIp))
	hh.UpdateDnsRecord(netif, true)
	net.UpdateBaremetalNetmap(bn, hh.GetNetifName(netif))
	return bn, nil
}

func (hh *SHost) PerformRemoveNetif(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.HostRemoveNetifInput,
) (jsonutils.JSONObject, error) {
	var err error
	input.HostNetifInput, err = validateHostNetif(input.HostNetifInput)
	if err != nil {
		return nil, errors.Wrap(err, "validateHostNetif")
	}

	netif := hh.GetNetInterface(input.Mac, input.VlanId)
	if netif == nil {
		return nil, httperrors.NewBadRequestError("Interface %s(vlan:%d) not exists", input.Mac, input.VlanId)
	}
	reserve := (input.Reserve != nil && *input.Reserve)

	return nil, hh.RemoveNetif(ctx, userCred, netif, reserve)
}

func (h *SHost) RemoveNetif(ctx context.Context, userCred mcclient.TokenCredential, netif *SNetInterface, reserve bool) error {
	h.DisableNetif(ctx, userCred, netif, reserve)
	// is this a converted host?
	if h.HostType == api.HOST_TYPE_HYPERVISOR && h.IsBaremetal {
		guests, err := h.GetGuests()
		if err != nil {
			return errors.Wrap(err, "GetGuests")
		}
		for i := range guests {
			guest := &guests[i]
			if guest.Hypervisor == api.HYPERVISOR_BAREMETAL {
				gn, err := guest.GetGuestnetworkByMac(netif.Mac)
				if err != nil && errors.Cause(err) != sql.ErrNoRows {
					return errors.Wrap(err, "GetGuestnetworkByMac")
				} else if gn != nil {
					err = gn.Detach(ctx, userCred)
					if err != nil {
						return errors.Wrap(err, "detach guest nic")
					}
				}
			}
		}
	}
	if netif.NicType == api.NIC_TYPE_ADMIN && h.AccessMac == netif.Mac {
		err := h.setAccessMac(userCred, "")
		if err != nil {
			return errors.Wrap(err, "setAccessMac")
		}
	}
	err := netif.Delete(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "netif.Remove")
	}
	h.ClearSchedDescCache()
	return nil
}

func (hh *SHost) getNetifsOnWire(wireId string) []SNetInterface {
	return hh.getNetInterfacesInternal(wireId, api.HOST_NIC_TYPES)
}

func (hh *SHost) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if hh.HostType != api.HOST_TYPE_BAREMETAL {
		return nil, httperrors.NewBadRequestError("Cannot sync status a non-baremetal host")
	}
	hh.SetStatus(userCred, api.BAREMETAL_SYNCING_STATUS, "")
	return nil, hh.StartSyncstatus(ctx, userCred, "")
}

func (hh *SHost) PerformReset(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !hh.IsBaremetal {
		return nil, httperrors.NewBadRequestError("Cannot start a non-baremetal host")
	}
	if hh.Status != api.BAREMETAL_RUNNING {
		return nil, httperrors.NewBadRequestError("Cannot reset baremetal in status %s", hh.Status)
	}
	guest := hh.GetBaremetalServer()
	if guest != nil {
		if hh.HostType == api.HOST_TYPE_BAREMETAL {
			if guest.Status != api.VM_ADMIN {
				return nil, httperrors.NewBadRequestError("Cannot reset baremetal with active guest")
			}
		} else {
			return guest.PerformReset(ctx, userCred, query, data)
		}
	}
	kwargs := jsonutils.NewDict()
	kwargs.Set("force_reboot", jsonutils.JSONTrue)
	kwargs.Set("action", jsonutils.NewString("reset"))
	return hh.PerformMaintenance(ctx, userCred, query, kwargs)
}

func (hh *SHost) PerformRemoveAllNetifs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	netifs := hh.GetAllNetInterfaces()
	for i := 0; i < len(netifs); i++ {
		if netifs[i].NicType == api.NIC_TYPE_NORMAL {
			hh.RemoveNetif(ctx, userCred, &netifs[i], false)
		}
	}
	return nil, nil
}

func (hh *SHost) PerformEnable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.PerformEnableInput,
) (jsonutils.JSONObject, error) {
	if !hh.GetEnabled() {
		_, err := hh.SEnabledStatusInfrasResourceBase.PerformEnable(ctx, userCred, query, input)
		if err != nil {
			return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBase.PerformEnable")
		}
		hh.SyncAttachedStorageStatus()
		hh.updateNotify(ctx, userCred)
	}
	return nil, nil
}

func (hh *SHost) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	if hh.GetEnabled() {
		_, err := hh.SEnabledStatusInfrasResourceBase.PerformDisable(ctx, userCred, query, input)
		if err != nil {
			return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBase.PerformDisable")
		}
		hh.SyncAttachedStorageStatus()
		hh.updateNotify(ctx, userCred)
	}
	return nil, nil
}

func (hh *SHost) PerformCacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CacheImageInput) (jsonutils.JSONObject, error) {
	if hh.HostType == api.HOST_TYPE_BAREMETAL || hh.HostStatus != api.HOST_ONLINE {
		return nil, httperrors.NewInvalidStatusError("Cannot perform cache image in status %s", hh.Status)
	}
	if len(input.ImageId) == 0 {
		return nil, httperrors.NewMissingParameterError("image_id")
	}
	img, err := CachedimageManager.getImageInfo(ctx, userCred, input.ImageId, false)
	if err != nil {
		return nil, httperrors.NewNotFoundError("image %s not found", input.ImageId)
	}
	input.ImageId = img.Id
	if len(img.Checksum) != 0 && regutils.MatchUUID(img.Checksum) {
		return nil, httperrors.NewInvalidStatusError("Cannot cache image with no checksum")
	}
	return nil, hh.StartImageCacheTask(ctx, userCred, input)
}

func (hh *SHost) StartImageCacheTask(ctx context.Context, userCred mcclient.TokenCredential, input api.CacheImageInput) error {
	var sc *SStoragecache
	switch hh.HostType {
	case api.HOST_TYPE_BAREMETAL:
	case api.HOST_TYPE_HYPERVISOR, api.HOST_TYPE_ESXI:
		sc = hh.GetLocalStoragecache()
	default:
		sc = hh.GetStoragecache()
	}
	if sc == nil {
		return errors.Wrap(errors.ErrNotSupported, "No associate storage cache found")
	}
	return sc.StartImageCacheTask(ctx, userCred, input)
}

func (hh *SHost) isAlterNameUnique(name string) (bool, error) {
	q := HostManager.Query().Equals("name", name).NotEquals("id", hh.Id).Equals("zone_id", hh.ZoneId)
	cnt, err := q.CountWithError()
	if err != nil {
		return false, err
	}
	return cnt == 0, nil
}

func (hh *SHost) PerformConvertHypervisor(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	hostType, err := data.GetString("host_type")
	if err != nil {
		return nil, httperrors.NewNotAcceptableError("host_type must be specified")
	}
	if hh.HostType != api.HOST_TYPE_BAREMETAL {
		return nil, httperrors.NewNotAcceptableError("Must be a baremetal host")
	}
	if hh.GetBaremetalServer() != nil {
		return nil, httperrors.NewNotAcceptableError("Baremetal host is aleady occupied")
	}
	if !utils.IsInStringArray(hh.Status, []string{api.BAREMETAL_READY, api.BAREMETAL_RUNNING}) {
		return nil, httperrors.NewNotAcceptableError("Connot convert hypervisor in status %s", hh.Status)
	}
	// check ownership
	var ownerId mcclient.IIdentityProvider
	hostOwnerId := hh.GetOwnerId()
	if userCred.GetProjectDomainId() != hostOwnerId.GetProjectDomainId() {
		if !db.IsAdminAllowPerform(ctx, userCred, hh, "convert-hypervisor") {
			return nil, httperrors.NewNotSufficientPrivilegeError("require system previleges to convert host in other domain")
		}
		firstProject, err := db.TenantCacheManager.FindFirstProjectOfDomain(ctx, hostOwnerId.GetProjectDomainId())
		if err != nil {
			return nil, errors.Wrap(err, "FindFirstProjectOfDomain")
		}
		ownerId = firstProject
	} else {
		ownerId = userCred
	}
	driver := GetHostDriver(hostType)
	if driver == nil {
		return nil, httperrors.NewNotAcceptableError("Unsupport driver type %s", hostType)
	}
	if data.Contains("name") {
		name, _ := data.GetString("name")
		err := hh.GetModelManager().ValidateName(name)
		if err != nil {
			return nil, err
		}
		uniq, err := hh.isAlterNameUnique(name)
		if err != nil {
			return nil, httperrors.NewInternalServerError("isAlterNameUnique fail %s", err)
		}
		if !uniq {
			return nil, httperrors.NewDuplicateNameError(name, hh.Id)
		}
	}
	image, _ := data.GetString("image")
	raid, _ := data.GetString("raid")
	input, err := driver.PrepareConvert(hh, image, raid, data)
	if err != nil {
		return nil, httperrors.NewNotAcceptableError("Convert error: %s", err.Error())
	}
	// admin delegate user to create system resource
	input.ProjectDomainId = ownerId.GetProjectDomainId()
	input.ProjectId = ownerId.GetProjectId()
	params := input.JSON(input)
	adminCred := auth.AdminCredential()
	guest, err := db.DoCreate(GuestManager, ctx, adminCred, nil, params, ownerId)
	if err != nil {
		return nil, err
	}
	func() {
		lockman.LockObject(ctx, guest)
		defer lockman.ReleaseObject(ctx, guest)

		guest.PostCreate(ctx, adminCred, ownerId, nil, params)
	}()
	log.Infof("Host convert to %s", guest.GetName())
	db.OpsLog.LogEvent(hh, db.ACT_CONVERT_START, "", userCred)
	db.OpsLog.LogEvent(guest, db.ACT_CREATE, "Convert hypervisor", userCred)

	opts := jsonutils.NewDict()
	opts.Set("server_params", params)
	opts.Set("server_id", jsonutils.NewString(guest.GetId()))
	opts.Set("convert_host_type", jsonutils.NewString(hostType))

	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalConvertHypervisorTask", hh, adminCred, opts, "", "", nil)
	if err != nil {
		return nil, err
	}
	task.ScheduleRun(nil)

	hh.SetStatus(userCred, api.BAREMETAL_START_CONVERT, "")
	return nil, nil
}

func (hh *SHost) PerformUndoConvert(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !hh.IsBaremetal {
		return nil, httperrors.NewNotAcceptableError("Not a baremetal")
	}
	if hh.HostType == api.HOST_TYPE_BAREMETAL {
		return nil, httperrors.NewNotAcceptableError("Not being convert to hypervisor")
	}
	if hh.GetEnabled() {
		return nil, httperrors.NewNotAcceptableError("Host should be disabled")
	}
	if !utils.IsInStringArray(hh.Status, []string{api.BAREMETAL_READY, api.BAREMETAL_RUNNING}) {
		return nil, httperrors.NewNotAcceptableError("Cannot unconvert in status %s", hh.Status)
	}
	driver := hh.GetDriverWithDefault()
	if driver == nil {
		return nil, httperrors.NewNotAcceptableError("Unsupport driver type %s", hh.HostType)
	}
	err := driver.PrepareUnconvert(hh)
	if err != nil {
		return nil, httperrors.NewNotAcceptableError("%v", err)
	}
	guests, err := hh.GetGuests()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetGuests"))
	}
	if len(guests) > 1 {
		return nil, httperrors.NewNotAcceptableError("Not an empty host")
	} else if len(guests) == 1 {
		guest := guests[0]
		if guest.Hypervisor != api.HYPERVISOR_BAREMETAL {
			return nil, httperrors.NewNotAcceptableError("Not an converted hypervisor")
		}
		err := guest.SetDisableDelete(userCred, false)
		if err != nil {
			return nil, err
		}
		db.OpsLog.LogEvent(&guest, db.ACT_DELETE, "Unconvert baremetal", userCred)
	}
	db.OpsLog.LogEvent(hh, db.ACT_UNCONVERT_START, "", userCred)
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalUnconvertHypervisorTask", hh, userCred, nil, "", "", nil)
	if err != nil {
		return nil, err
	}
	task.ScheduleRun(nil)
	return nil, nil
}

func (hh *SHost) GetDriverWithDefault() IHostDriver {
	hostType := hh.HostType
	if len(hostType) == 0 {
		hostType = api.HOST_TYPE_DEFAULT
	}
	return GetHostDriver(hostType)
}

func (hh *SHost) UpdateDiskConfig(userCred mcclient.TokenCredential, layouts []baremetal.Layout) error {
	bs := hh.GetBaremetalstorage()
	if bs != nil {
		diff, err := db.Update(bs, func() error {
			if len(layouts) != 0 {
				bs.Config = jsonutils.Marshal(layouts).(*jsonutils.JSONArray)
				var size int64
				for i := 0; i < len(layouts); i++ {
					size += layouts[i].Size
				}
				bs.RealCapacity = size
			} else {
				bs.Config = jsonutils.NewArray()
				bs.RealCapacity = bs.GetStorage().Capacity
			}
			return nil
		})
		if err != nil {
			log.Errorln(err)
			return err
		}
		db.OpsLog.LogEvent(bs, db.ACT_UPDATE, diff, userCred)
	}
	return nil
}

// TODO: support multithreaded operation
/*func (host *SHost) SyncEsxiHostWires(ctx context.Context, userCred mcclient.TokenCredential, remoteHost cloudprovider.ICloudHost) compare.SyncResult {
	lockman.LockObject(ctx, host)
	defer lockman.ReleaseObject(ctx, host)

	result := compare.SyncResult{}
	ca := host.GetCloudaccount()
	host2wires, err := ca.GetHost2Wire(ctx, userCred)
	if err != nil {
		result.Error(errors.Wrap(err, "unable to GetHost2Wire"))
		return result
	}
	log.Infof("host2wires: %s", jsonutils.Marshal(host2wires))
	ihost := remoteHost.(*esxi.SHost)
	remoteHostId := ihost.GetId()
	vsWires := host2wires[remoteHostId]

	log.Infof("vsWires: %s", jsonutils.Marshal(vsWires))
	netIfs := host.GetHostNetInterfaces()

	for i := range vsWires {
		vsWire := vsWires[i]
		if vsWire.SyncTimes > 0 {
			continue
		}
		netif := host.findNetIfs(netIfs, vsWire.Mac, 1)
		if netif == nil {
			// do nothing
			continue
		}
		if netif.Bridge != vsWire.VsId {
			db.Update(netif, func() error {
				netif.Bridge = vsWire.VsId
				return nil
			})
		}
		if len(netif.WireId) == 0 {
			db.Update(netif, func() error {
				netif.WireId = vsWire.WireId
				return nil
			})
		}
		vsWires[i].SyncTimes += 1
	}
	log.Infof("after sync: %s", jsonutils.Marshal(host2wires))
	ca.SetHost2Wire(ctx, userCred, host2wires)
	return result
}*/

/*func (host *SHost) findHostwire(hostwires []SHostwire, wireId string, mac string) *SHostwire {
	for i := range hostwires {
		if hostwires[i].WireId == wireId && hostwires[i].MacAddr == mac {
			return &hostwires[i]
		}
	}
	return nil
}*/

func (host *SHost) findNetIfs(netIfs []SNetInterface, mac string, vlanId int) *SNetInterface {
	for i := range netIfs {
		if netIfs[i].Mac == mac && netIfs[i].VlanId == vlanId {
			return &netIfs[i]
		}
	}
	return nil
}

func (host *SHost) SyncHostExternalNics(ctx context.Context, userCred mcclient.TokenCredential, ihost cloudprovider.ICloudHost, provider *SCloudprovider) compare.SyncResult {
	result := compare.SyncResult{}

	netIfs := host.GetHostNetInterfaces()
	extNics, err := ihost.GetIHostNics()
	if err != nil {
		log.Errorf("GetIHostNics fail %s", err)
		result.Error(err)
		return result
	}

	log.Debugf("SyncHostExternalNics for host %s netIfs %d ihost %s extNics %d", host.Name, len(netIfs), ihost.GetName(), len(extNics))

	disables := make([]*SNetInterface, 0)
	enables := make([]cloudprovider.ICloudHostNetInterface, 0)

	type sRemoveNetInterface struct {
		netif     *SNetInterface
		reserveIp bool
	}

	type sAddNetInterface struct {
		netif     cloudprovider.ICloudHostNetInterface
		reserveIp bool
	}

	removes := make([]sRemoveNetInterface, 0)
	adds := make([]sAddNetInterface, 0)

	for i := 0; i < len(netIfs); i++ {
		find := false
		for j := 0; j < len(extNics); j++ {
			if netIfs[i].Mac == extNics[j].GetMac() && netIfs[i].VlanId == extNics[j].GetVlanId() {
				// find! need to update
				find = true
				obn := netIfs[i].GetHostNetwork()
				var oip string
				if obn != nil {
					oip = obn.IpAddr
				}
				nip := extNics[j].GetIpAddr()
				if oip != nip {
					if obn != nil {
						disables = append(disables, &netIfs[i])
					}
					if len(nip) > 0 {
						enables = append(enables, extNics[j])
					}
				} else {
					wireId := ""
					extWire := extNics[j].GetIWire()
					if extWire != nil {
						wire, err := WireManager.FetchWireByExternalId(provider.Id, extWire.GetGlobalId())
						if err != nil {
							result.AddError(err)
						} else {
							wireId = wire.Id
						}
					}
					// in sync, sync interface and bridge
					if netIfs[i].Bridge != extNics[j].GetBridge() || netIfs[i].Interface != extNics[j].GetDevice() || netIfs[i].WireId != wireId {
						_, err := db.Update(&netIfs[i], func() error {
							netIfs[i].Interface = extNics[j].GetDevice()
							netIfs[i].Bridge = extNics[j].GetBridge()
							netIfs[i].WireId = wireId
							return nil
						})
						if err != nil {
							result.Error(errors.Wrap(err, "update interface and bridge fail"))
							return result
						}
					}
				}
				break
			}
		}
		if !find {
			// need to remove
			removes = append(removes, sRemoveNetInterface{netif: &netIfs[i], reserveIp: false})
		}
	}
	for j := 0; j < len(extNics); j++ {
		find := false
		for i := 0; i < len(netIfs); i++ {
			if netIfs[i].Mac == extNics[j].GetMac() && netIfs[i].VlanId == extNics[j].GetVlanId() {
				find = true
				break
			}
		}
		if !find {
			// need to add
			adds = append(adds, sAddNetInterface{netif: extNics[j], reserveIp: false})
		}
	}
	// find out which ip need to be reserved
	for i := 0; i < len(removes); i++ {
		var oip string
		obn := removes[i].netif.GetHostNetwork()
		if obn != nil {
			oip = obn.IpAddr
		}
		if len(oip) == 0 {
			// skip
			continue
		}
		for j := 0; j < len(adds); j++ {
			if oip == adds[j].netif.GetIpAddr() {
				// find out ! IP reserved but interface changed!
				removes[i].reserveIp = true
				adds[j].reserveIp = true
				break
			}
		}
	}

	log.Debugf("SyncHostExternalNics %s remove %d disable %d enable %d add %d", host.Name, len(removes), len(disables), len(enables), len(adds))

	for i := len(removes) - 1; i >= 0; i -= 1 {
		log.Debugf("remove netif %s", removes[i].netif.Mac)
		err := host.RemoveNetif(ctx, userCred, removes[i].netif, removes[i].reserveIp)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}

	for i := len(disables) - 1; i >= 0; i -= 1 {
		log.Debugf("disable netif %s", disables[i].Mac)
		err := host.DisableNetif(ctx, userCred, disables[i], false)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}

	for i := 0; i < len(enables); i += 1 {
		netif := host.GetNetInterface(enables[i].GetMac(), enables[i].GetVlanId())
		// always true reserved address pool
		log.Debugf("enable netif %s", enables[i].GetMac())
		err = host.EnableNetif(ctx, userCred, netif, "", enables[i].GetIpAddr(), "", "", true, true)
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
		}
	}

	for i := 0; i < len(adds); i += 1 {
		log.Debugf("add netif %s", adds[i].netif.GetMac())
		// always try reserved pool
		extNic := adds[i].netif
		var strNetIf, strBridge *string
		netif := extNic.GetDevice()
		bridge := extNic.GetBridge()
		if len(netif) > 0 {
			strNetIf = &netif
		}
		if len(bridge) > 0 {
			strBridge = &bridge
		}
		wireId := ""
		extWire := extNic.GetIWire()
		if extWire != nil {
			wire, err := WireManager.FetchWireByExternalId(provider.Id, extWire.GetGlobalId())
			if err != nil {
				result.AddError(err)
			} else {
				wireId = wire.Id
			}
		}
		err = host.addNetif(ctx, userCred, extNic.GetMac(), extNic.GetVlanId(), wireId, extNic.GetIpAddr(), 0,
			compute.TNicType(extNic.GetNicType()), extNic.GetIndex(),
			extNic.IsLinkUp(), int16(extNic.GetMtu()), false, strNetIf, strBridge, true, true)
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
		}
	}

	return result
}

// func (manager *SHostManager) GetEsxiAgentHostId(key string) (string, error) {
// 	q := HostManager.Query("id")
// 	q = q.Equals("host_status", HOST_ONLINE)
// 	q = q.Equals("host_type", HOST_TYPE_HYPERVISOR)
// 	q = q.IsTrue("enabled")
//
// 	rows, err := q.Rows()
// 	if err != nil {
// 		return "", err
// 	}
// 	defer rows.Close()
//
// 	var hostId string
// 	hostIds := make([]string, 0)
// 	for rows.Next() {
// 		err = rows.Scan(&hostId)
// 		if err != nil {
// 			return "", err
// 		}
// 		hostIds = append(hostIds, hostId)
// 	}
//
// 	ring := hashring.New(hostIds)
// 	ret, _ := ring.GetNode(key)
// 	return ret, nil
// }
//
// func (manager *SHostManager) GetEsxiAgentHost(key string) (*SHost, error) {
// 	hostId, err := manager.GetEsxiAgentHostId(key)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return manager.FetchHostById(hostId), nil
// }
//
// func (host *SHost) GetEsxiAgentHost() (*SHost, error) {
// 	return HostManager.GetEsxiAgentHost(host.Id)
// }

func (hh *SHost) IsBaremetalAgentReady() bool {
	return hh.isAgentReady(api.AgentTypeBaremetal)
}

func (hh *SHost) BaremetalSyncRequest(ctx context.Context, method httputils.THttpMethod, url string, headers http.Header, body *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	return hh.doAgentRequest(api.AgentTypeBaremetal, ctx, method, url, headers, body)
}

func (hh *SHost) IsEsxiAgentReady() bool {
	return hh.isAgentReady(api.AgentTypeEsxi)
}

func (hh *SHost) EsxiRequest(ctx context.Context, method httputils.THttpMethod, url string, headers http.Header, body *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	return hh.doAgentRequest(api.AgentTypeEsxi, ctx, method, url, headers, body)
}

func (hh *SHost) GetAgent(at api.TAgentType) *SBaremetalagent {
	agent := BaremetalagentManager.GetAgent(at, hh.ZoneId)
	if agent == nil {
		agent = BaremetalagentManager.GetAgent(at, "")
	}
	return agent
}

func (hh *SHost) isAgentReady(agentType api.TAgentType) bool {
	agent := hh.GetAgent(agentType)
	if agent == nil {
		log.Errorf("%s ready: false", agentType)
		return false
	}
	return true
}

func (hh *SHost) doAgentRequest(agentType api.TAgentType, ctx context.Context, method httputils.THttpMethod, url string, headers http.Header, body *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	agent := hh.GetAgent(agentType)
	if agent == nil {
		return nil, fmt.Errorf("no valid %s", agentType)
	}
	serviceUrl := agent.ManagerUri
	if url[0] != '/' && serviceUrl[len(serviceUrl)-1] != '/' {
		serviceUrl += "/"
	}
	url = serviceUrl + url
	_, data, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, method, url, headers, body, false)
	return data, err
}

func (manager *SHostManager) GetHostByIp(managerId, hostType, hostIp string) (*SHost, error) {
	q := manager.Query()
	q = q.Equals("access_ip", hostIp).Equals("host_type", hostType)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}

	ret := []SHost{}
	err := db.FetchModelObjects(manager, q, &ret)
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s %s", hostType, hostIp)
	}
	if len(ret) > 1 {
		return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, "%s %s", hostType, hostIp)
	}
	return &ret[0], nil
}

func (hh *SHost) getCloudProviderInfo() SCloudProviderInfo {
	var region *SCloudregion
	zone, _ := hh.GetZone()
	if zone != nil {
		region, _ = zone.GetRegion()
	}
	provider := hh.GetCloudprovider()
	return MakeCloudProviderInfo(region, zone, provider)
}

func (hh *SHost) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := hh.SEnabledStatusInfrasResourceBase.GetShortDesc(ctx)
	info := hh.getCloudProviderInfo()
	desc.Update(jsonutils.Marshal(&info))
	return desc
}

func (hh *SHost) MarkGuestUnknown(userCred mcclient.TokenCredential) {
	guests, _ := hh.GetGuests()
	for _, guest := range guests {
		guest.SetStatus(userCred, api.VM_UNKNOWN, "host offline")
	}
	guests2 := hh.GetGuestsBackupOnThisHost()
	for _, guest := range guests2 {
		guest.SetBackupGuestStatus(userCred, api.VM_UNKNOWN, "host offline")
	}
}

func (manager *SHostManager) PingDetectionTask(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	deadline := time.Now().Add(-1 * time.Duration(options.Options.HostOfflineMaxSeconds) * time.Second)

	q := manager.Query().Equals("host_status", api.HOST_ONLINE).
		Equals("host_type", api.HOST_TYPE_HYPERVISOR)
	q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("last_ping_at")),
		sqlchemy.LT(q.Field("last_ping_at"), deadline)))

	hosts := []SHost{}
	err := db.FetchModelObjects(manager, q, &hosts)
	if err != nil {
		return
	}

	updateHealthStatus := false
	for i := range hosts {
		func() {
			lockman.LockObject(ctx, &hosts[i])
			defer lockman.ReleaseObject(ctx, &hosts[i])
			hosts[i].PerformOffline(ctx, userCred, nil, &api.HostOfflineInput{UpdateHealthStatus: &updateHealthStatus, Reason: fmt.Sprintf("last ping detection at %s", deadline)})
			hosts[i].MarkGuestUnknown(userCred)
		}()
	}
}

func (hh *SHost) IsPrepaidRecycleResource() bool {
	return hh.ResourceType == api.HostResourceTypePrepaidRecycle
}

func (host *SHost) PerformSetSchedtag(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return PerformSetResourceSchedtag(host, ctx, userCred, query, data)
}

func (host *SHost) GetDynamicConditionInput() *jsonutils.JSONDict {
	return jsonutils.Marshal(host).(*jsonutils.JSONDict)
}

func (host *SHost) PerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformStatusInput) (jsonutils.JSONObject, error) {
	ret, err := host.SEnabledStatusInfrasResourceBase.PerformStatus(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBase.PerformStatus")
	}
	host.ClearSchedDescCache()
	return ret, nil
}

func (host *SHost) GetSchedtagJointManager() ISchedtagJointManager {
	return HostschedtagManager
}

func (host *SHost) PerformHostExitMaintenance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(host.Status, []string{api.BAREMETAL_MAINTAIN_FAIL, api.BAREMETAL_MAINTAINING}) {
		return nil, httperrors.NewInvalidStatusError("host status %s can't exit maintenance", host.Status)
	}
	err := host.SetStatus(userCred, api.HOST_STATUS_RUNNING, "exit maintenance")
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (host *SHost) PerformHostMaintenance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if host.HostType != api.HOST_TYPE_HYPERVISOR {
		return nil, httperrors.NewBadRequestError("host type %s can't do host maintenance", host.HostType)
	}
	if host.HostStatus == api.BAREMETAL_START_MAINTAIN {
		return nil, httperrors.NewBadRequestError("unsupport on host status %s", host.HostStatus)
	}

	var preferHostId string
	preferHost, _ := data.GetString("prefer_host")
	if len(preferHost) > 0 {
		iHost, _ := HostManager.FetchByIdOrName(userCred, preferHost)
		if iHost == nil {
			return nil, httperrors.NewBadRequestError("Host %s not found", preferHost)
		}
		host := iHost.(*SHost)
		preferHostId = host.Id
		err := host.IsAssignable(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "IsAssignable")
		}
	}

	guests := host.GetKvmGuests()
	for i := 0; i < len(guests); i++ {
		lockman.LockObject(ctx, &guests[i])
		defer lockman.ReleaseObject(ctx, &guests[i])
		guest, err := guests[i].validateForBatchMigrate(ctx, false)
		if err != nil {
			return nil, err
		}
		guests[i] = *guest
		if host.HostStatus == api.HOST_OFFLINE && guests[i].Status != api.VM_UNKNOWN {
			return nil, httperrors.NewBadRequestError("Host %s can't migrate guests %s in status %s",
				host.HostStatus, guests[i].Name, guests[i].Status)
		}
	}

	var hostGuests = []*api.GuestBatchMigrateParams{}
	for i := 0; i < len(guests); i++ {
		bmp := &api.GuestBatchMigrateParams{
			Id:          guests[i].Id,
			LiveMigrate: guests[i].Status == api.VM_RUNNING,
			RescueMode:  guests[i].Status == api.VM_UNKNOWN,
			OldStatus:   guests[i].Status,
		}
		guests[i].SetStatus(userCred, api.VM_START_MIGRATE, "host maintainence")
		hostGuests = append(hostGuests, bmp)
	}

	kwargs := jsonutils.NewDict()
	kwargs.Set("guests", jsonutils.Marshal(hostGuests))
	kwargs.Set("prefer_host_id", jsonutils.NewString(preferHostId))
	return nil, host.StartMaintainTask(ctx, userCred, kwargs)
}

func (host *SHost) autoMigrateOnHostShutdown(ctx context.Context) bool {
	return host.GetMetadata(ctx, api.HOSTMETA_AUTO_MIGRATE_ON_HOST_SHUTDOWN, nil) == "enable"
}

func (host *SHost) RemoteHealthStatus(ctx context.Context) string {
	var status = api.HOST_HEALTH_STATUS_UNKNOWN
	userCred := auth.AdminCredential()
	res, err := host.Request(
		ctx, userCred, "GET", "/hosts/health-status",
		mcclient.GetTokenHeaders(userCred), nil,
	)
	if err != nil {
		log.Errorf("failed get remote health status %s", err)
	} else {
		status, _ = res.GetString("status")
	}

	log.Infof("remote health status %s", status)
	return status
}

func (host *SHost) GetHostnameByName() string {
	hostname := host.Name
	accessIp := strings.Replace(host.AccessIp, ".", "-", -1)
	if strings.HasSuffix(host.Name, "-"+accessIp) {
		hostname = hostname[0 : len(hostname)-len(accessIp)-1]
	}
	return hostname
}

func (host *SHost) OnHostDown(ctx context.Context, userCred mcclient.TokenCredential) {
	log.Errorf("watched host down %s, status %s", host.Name, host.HostStatus)
	hostHealthChecker.UnwatchHost(ctx, host.GetHostnameByName())

	if host.HostStatus == api.HOST_OFFLINE && !host.EnableHealthCheck &&
		!host.autoMigrateOnHostShutdown(ctx) {
		// hostagent requested offline, and not enable auto migrate on host shutdown
		log.Infof("host not need auto migrate on host shutdown")
		return
	}

	hostname := host.Name
	if host.HostStatus == api.HOST_OFFLINE {
		// host has been marked offline, check host status in k8s
		coreCli, err := tokens.GetCoreClient()
		if err != nil {
			log.Errorf("failed get k8s client %s", err)
			return
		}
		hostname = host.GetHostnameByName()

		node, err := coreCli.Nodes().Get(context.TODO(), hostname, metav1.GetOptions{})
		if err != nil {
			log.Errorf("failed get node %s info %s", hostname, err)
			return
		}

		// check node status is ready
		if length := len(node.Status.Conditions); length > 0 {
			if node.Status.Conditions[length-1].Type == v1.NodeReady &&
				node.Status.Conditions[length-1].Status == v1.ConditionTrue {
				log.Infof("node %s status ready, no need entry rescue", hostname)
				return
			}
		}
	}

	log.Errorf("host %s down, try rescue guests", hostname)
	db.OpsLog.LogEvent(host, db.ACT_HOST_DOWN, "", userCred)
	if _, err := host.SaveCleanUpdates(func() error {
		host.EnableHealthCheck = false
		host.HostStatus = api.HOST_OFFLINE
		return nil
	}); err != nil {
		log.Errorf("update host %s failed %s", host.Id, err)
	}

	logclient.AddActionLogWithContext(ctx, host, logclient.ACT_OFFLINE, map[string]string{"reason": "host down"}, userCred, false)
	host.SyncCleanSchedDescCache()
	host.switchWithBackup(ctx, userCred)
	host.migrateOnHostDown(ctx, userCred)
}

func (host *SHost) switchWithBackup(ctx context.Context, userCred mcclient.TokenCredential) {
	guests := host.GetGuestsMasterOnThisHost()
	for i := 0; i < len(guests); i++ {
		data := jsonutils.NewDict()
		_, err := guests[i].PerformSwitchToBackup(ctx, userCred, nil, data)
		if err != nil {
			db.OpsLog.LogEvent(
				&guests[i], db.ACT_SWITCH_FAILED, fmt.Sprintf("PerformSwitchToBackup on host down: %s", err), userCred,
			)
			logclient.AddSimpleActionLog(
				&guests[i], logclient.ACT_SWITCH_TO_BACKUP,
				fmt.Sprintf("PerformSwitchToBackup on host down: %s", err), userCred, false,
			)
		}
	}
}

func (host *SHost) migrateOnHostDown(ctx context.Context, userCred mcclient.TokenCredential) {
	if host.GetMetadata(ctx, api.HOSTMETA_AUTO_MIGRATE_ON_HOST_DOWN, nil) == "enable" {
		if err := host.MigrateSharedStorageServers(ctx, userCred); err != nil {
			db.OpsLog.LogEvent(host, db.ACT_HOST_DOWN, fmt.Sprintf("migrate servers failed %s", err), userCred)
		}
	}
}

func (host *SHost) MigrateSharedStorageServers(ctx context.Context, userCred mcclient.TokenCredential) error {
	guests, err := host.GetGuests()
	if err != nil {
		return errors.Wrapf(err, "host %s(%s) get guests", host.Name, host.Id)
	}
	migGuests := []*SGuest{}
	hostGuests := []*api.GuestBatchMigrateParams{}

	for i := 0; i < len(guests); i++ {
		if guests[i].isNotRunningStatus(guests[i].Status) {
			// skip not running guests
			continue
		}

		lockman.LockObject(ctx, &guests[i])
		defer lockman.ReleaseObject(ctx, &guests[i])
		_, err := guests[i].validateForBatchMigrate(ctx, true)
		if err != nil {
			continue
		} else {
			bmp := &api.GuestBatchMigrateParams{
				Id:          guests[i].Id,
				LiveMigrate: false,
				RescueMode:  true,
				OldStatus:   guests[i].Status,
			}
			guests[i].SetStatus(userCred, api.VM_START_MIGRATE, "host down")
			hostGuests = append(hostGuests, bmp)
			migGuests = append(migGuests, &guests[i])
		}
	}
	kwargs := jsonutils.NewDict()
	kwargs.Set("guests", jsonutils.Marshal(hostGuests))
	return GuestManager.StartHostGuestsMigrateTask(ctx, userCred, migGuests, kwargs, "")
}

func (host *SHost) SetStatus(userCred mcclient.TokenCredential, status string, reason string) error {
	err := host.SEnabledStatusInfrasResourceBase.SetStatus(userCred, status, reason)
	if err != nil {
		return err
	}
	host.ClearSchedDescCache()
	notifyclient.EventNotify(context.Background(), userCred, notifyclient.SEventNotifyParam{
		Obj:    host,
		Action: notifyclient.ActionUpdate,
	})
	return nil
}

func (host *SHost) StartMaintainTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) error {
	host.SetStatus(userCred, api.BAREMETAL_START_MAINTAIN, "start maintenance")
	if task, err := taskman.TaskManager.NewTask(ctx, "HostMaintainTask", host, userCred, data, "", "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (host *SHost) IsMaintaining() bool {
	return utils.IsInStringArray(host.Status, []string{api.BAREMETAL_START_MAINTAIN, api.BAREMETAL_MAINTAINING, api.BAREMETAL_MAINTAIN_FAIL})
}

// InstanceGroups returns the enabled group of guest in host and their frequency of occurrence
func (host *SHost) InstanceGroups() ([]SGroup, map[string]int, error) {
	q := GuestManager.Query("id")
	guestQ := q.Filter(sqlchemy.OR(sqlchemy.Equals(q.Field("host_id"), host.Id),
		sqlchemy.Equals(q.Field("backup_host_id"), host.Id))).SubQuery()
	groupQ := GroupguestManager.Query().SubQuery()
	q = groupQ.Query().Join(guestQ, sqlchemy.Equals(guestQ.Field("id"), groupQ.Field("guest_id")))
	groupguests := make([]SGroupguest, 0, 1)
	err := db.FetchModelObjects(GroupguestManager, q, &groupguests)
	if err != nil {
		return nil, nil, err
	}
	groupIds, groupSet := make([]string, 0, len(groupguests)), make(map[string]int)
	for i := range groupguests {
		id := groupguests[i].GroupId
		if _, ok := groupSet[id]; !ok {
			groupIds = append(groupIds, id)
			groupSet[id] = 1
			continue
		}
		groupSet[id] += 1
	}
	if len(groupIds) == 0 {
		return []SGroup{}, make(map[string]int), nil
	}
	groups := make([]SGroup, 0, len(groupIds))
	q = GroupManager.Query().In("id", groupIds).IsTrue("enabled")
	err = db.FetchModelObjects(GroupManager, q, &groups)
	if err != nil {
		return nil, nil, err
	}
	retSet := make(map[string]int)
	for i := range groups {
		retSet[groups[i].GetId()] = groupSet[groups[i].GetId()]
	}
	return groups, retSet, nil
}

func (host *SHost) setIpmiIp(userCred mcclient.TokenCredential, ipAddr string) error {
	if host.IpmiIp == ipAddr {
		return nil
	}
	diff, err := db.Update(host, func() error {
		host.IpmiIp = ipAddr
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "db.Update")
	}
	db.OpsLog.LogEvent(host, db.ACT_UPDATE, diff, userCred)
	return nil
}

func (host *SHost) setAccessIp(userCred mcclient.TokenCredential, ipAddr string) error {
	if host.AccessIp == ipAddr {
		return nil
	}
	diff, err := db.Update(host, func() error {
		host.AccessIp = ipAddr
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "db.Update")
	}
	db.OpsLog.LogEvent(host, db.ACT_UPDATE, diff, userCred)
	return nil
}

func (host *SHost) setAccessMac(userCred mcclient.TokenCredential, mac string) error {
	mac = netutils.FormatMacAddr(mac)
	if host.AccessMac == mac {
		return nil
	}
	diff, err := db.Update(host, func() error {
		host.AccessMac = mac
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "db.Update")
	}
	db.OpsLog.LogEvent(host, db.ACT_UPDATE, diff, userCred)
	return nil
}

func (host *SHost) GetIpmiInfo() (types.SIPMIInfo, error) {
	info := types.SIPMIInfo{}
	if host.IpmiInfo != nil {
		err := host.IpmiInfo.Unmarshal(&info)
		if err != nil {
			return info, errors.Wrap(err, "host.IpmiInfo.Unmarshal")
		}
	}
	return info, nil
}

func (host *SHost) GetNics() []*types.SNic {
	netifs := host.GetAllNetInterfaces()
	nicInfos := []*types.SNic{}
	if netifs != nil && len(netifs) > 0 {
		for i := 0; i < len(netifs); i += 1 {
			nicInfos = append(nicInfos, netifs[i].getBaremetalJsonDesc())
		}
	}
	return nicInfos
}

func (host *SHost) GetUEFIInfo() (*types.EFIBootMgrInfo, error) {
	if host.UefiInfo == nil {
		return nil, nil
	}
	info := new(types.EFIBootMgrInfo)
	if err := host.UefiInfo.Unmarshal(info); err != nil {
		return nil, errors.Wrap(err, "host.UefiInfo.Unmarshal")
	}
	return info, nil
}

func (hh *SHost) GetDetailsJnlp(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("/baremetals/%s/jnlp", hh.Id)
	header := mcclient.GetTokenHeaders(userCred)
	resp, err := hh.BaremetalSyncRequest(ctx, "POST", url, header, nil)
	if err != nil {
		return nil, errors.Wrap(err, "BaremetalSyncRequest")
	}
	return resp, nil
}

func (hh *SHost) PerformInsertIso(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(hh.Status, []string{api.BAREMETAL_READY, api.BAREMETAL_RUNNING}) {
		imageStr, err := data.GetString("image")
		image, err := CachedimageManager.getImageInfo(ctx, userCred, imageStr, false)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("image", imageStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		if image.Status != cloudprovider.IMAGE_STATUS_ACTIVE {
			return nil, httperrors.NewInvalidStatusError("Image status is not active")
		}
		boot := jsonutils.QueryBoolean(data, "boot", false)
		return nil, hh.StartInsertIsoTask(ctx, userCred, image.Id, boot, "")
	}
	return nil, httperrors.NewInvalidStatusError("Cannot do insert-iso in status %s", hh.Status)
}

func (hh *SHost) StartInsertIsoTask(ctx context.Context, userCred mcclient.TokenCredential, imageId string, boot bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(imageId), "image_id")
	if boot {
		data.Add(jsonutils.JSONTrue, "boot")
	}
	data.Add(jsonutils.NewString(api.BAREMETAL_CDROM_ACTION_INSERT), "action")
	hh.SetStatus(userCred, api.BAREMETAL_START_INSERT_ISO, "start insert iso task")
	if task, err := taskman.TaskManager.NewTask(ctx, "BaremetalCdromTask", hh, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
		return nil
	}
}

func (hh *SHost) PerformEjectIso(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(hh.Status, []string{api.BAREMETAL_READY, api.BAREMETAL_RUNNING}) {
		return nil, hh.StartEjectIsoTask(ctx, userCred, "")
	}
	return nil, httperrors.NewInvalidStatusError("Cannot do eject-iso in status %s", hh.Status)
}

func (hh *SHost) StartEjectIsoTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(api.BAREMETAL_CDROM_ACTION_EJECT), "action")
	hh.SetStatus(userCred, api.BAREMETAL_START_EJECT_ISO, "start eject iso task")
	if task, err := taskman.TaskManager.NewTask(ctx, "BaremetalCdromTask", hh, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
		return nil
	}
}

func (hh *SHost) PerformSyncConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if hh.HostType != api.HOST_TYPE_BAREMETAL {
		return nil, httperrors.NewBadRequestError("Cannot sync config a non-baremetal host")
	}
	hh.SetStatus(userCred, api.BAREMETAL_SYNCING_STATUS, "")
	return nil, hh.StartSyncConfig(ctx, userCred, "")
}

func (hh *SHost) StartSyncConfig(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return hh.startSyncConfig(ctx, userCred, parentTaskId, false)
}

func (hh *SHost) startSyncConfig(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, noStatus bool) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewBool(noStatus), "not_sync_status")
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalSyncConfigTask", hh, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (model *SHost) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	// make host default public
	return model.SEnabledStatusInfrasResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (host *SHost) PerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformChangeDomainOwnerInput) (jsonutils.JSONObject, error) {
	ret, err := host.SEnabledStatusInfrasResourceBase.PerformChangeOwner(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBase.PerformChangeOwner")
	}

	localStorages := host._getAttachedStorages(tristate.None, tristate.None, api.HOST_STORAGE_LOCAL_TYPES)
	for i := range localStorages {
		_, err := localStorages[i].performChangeOwnerInternal(ctx, userCred, query, input)
		if err != nil {
			return nil, errors.Wrap(err, "local storage change owner")
		}
	}
	err = host.StartSyncTask(ctx, userCred, "")
	if err != nil {
		return nil, errors.Wrap(err, "PerformChangeOwner StartSyncTask err")
	}
	return ret, nil
}

func (hh *SHost) StartSyncTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	if task, err := taskman.TaskManager.NewTask(ctx, "HostSyncTask", hh, userCred, jsonutils.NewDict(), parentTaskId, "",
		nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (host *SHost) GetChangeOwnerRequiredDomainIds() []string {
	requires := stringutils2.SSortedStrings{}
	guests, _ := host.GetGuests()
	for i := range guests {
		requires = stringutils2.Append(requires, guests[i].DomainId)
	}
	return requires
}

func GetHostQuotaKeysFromCreateInput(owner mcclient.IIdentityProvider, input api.HostCreateInput) quotas.SDomainRegionalCloudResourceKeys {
	ownerId := &db.SOwnerId{DomainId: owner.GetProjectDomainId()}
	var zone *SZone
	if len(input.ZoneId) > 0 {
		zone = ZoneManager.FetchZoneById(input.ZoneId)
	}
	zoneKeys := fetchZonalQuotaKeys(rbacscope.ScopeDomain, ownerId, zone, nil)
	keys := quotas.SDomainRegionalCloudResourceKeys{}
	keys.SBaseDomainQuotaKeys = zoneKeys.SBaseDomainQuotaKeys
	keys.SRegionalBaseKeys = zoneKeys.SRegionalBaseKeys
	return keys
}

func (model *SHost) GetQuotaKeys() quotas.SDomainRegionalCloudResourceKeys {
	zone, _ := model.GetZone()
	manager := model.GetCloudprovider()
	ownerId := model.GetOwnerId()
	zoneKeys := fetchZonalQuotaKeys(rbacscope.ScopeDomain, ownerId, zone, manager)
	keys := quotas.SDomainRegionalCloudResourceKeys{}
	keys.SBaseDomainQuotaKeys = zoneKeys.SBaseDomainQuotaKeys
	keys.SRegionalBaseKeys = zoneKeys.SRegionalBaseKeys
	keys.SCloudResourceBaseKeys = zoneKeys.SCloudResourceBaseKeys
	return keys
}

func (host *SHost) GetUsages() []db.IUsage {
	if host.Deleted {
		return nil
	}
	usage := SInfrasQuota{Host: 1}
	keys := host.GetQuotaKeys()
	usage.SetKeys(keys)
	return []db.IUsage{
		&usage,
	}
}

func (host *SHost) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPublicDomainInput) (jsonutils.JSONObject, error) {
	// perform public for all connected local storage
	storages := host._getAttachedStorages(tristate.None, tristate.None, api.HOST_STORAGE_LOCAL_TYPES)
	for i := range storages {
		_, err := storages[i].performPublicInternal(ctx, userCred, query, input)
		if err != nil {
			return nil, errors.Wrap(err, "storage.PerformPublic")
		}
	}
	return host.SEnabledStatusInfrasResourceBase.PerformPublic(ctx, userCred, query, input)
}

func (host *SHost) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPrivateInput) (jsonutils.JSONObject, error) {
	// perform private for all connected local storage
	storages := host._getAttachedStorages(tristate.None, tristate.None, api.HOST_STORAGE_LOCAL_TYPES)
	for i := range storages {
		_, err := storages[i].performPrivateInternal(ctx, userCred, query, input)
		if err != nil {
			return nil, errors.Wrap(err, "storage.PerformPrivate")
		}
	}
	return host.SEnabledStatusInfrasResourceBase.PerformPrivate(ctx, userCred, query, input)
}

func (host *SHost) PerformSetReservedResourceForIsolatedDevice(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, input api.IsolatedDeviceReservedResourceInput,
) (jsonutils.JSONObject, error) {
	if input.ReservedCpu != nil && *input.ReservedCpu < 0 {
		return nil, httperrors.NewInputParameterError("reserved cpu must >= 0")
	}
	if input.ReservedMemory != nil && *input.ReservedMemory < 0 {
		return nil, httperrors.NewInputParameterError("reserved memory must >= 0")
	}
	if input.ReservedStorage != nil && *input.ReservedStorage < 0 {
		return nil, httperrors.NewInputParameterError("reserved storage must >= 0")
	}
	devs := IsolatedDeviceManager.FindByHost(host.Id)
	if len(devs) == 0 {
		return nil, nil
	}
	if input.ReservedCpu != nil && host.GetCpuCount() < *input.ReservedCpu*len(devs) {
		return nil, httperrors.NewBadRequestError(
			"host %s can't reserve %d cpu for each isolated device, not enough", host.Name, *input.ReservedCpu)
	}
	if input.ReservedMemory != nil && host.GetMemSize() < *input.ReservedMemory*len(devs) {
		return nil, httperrors.NewBadRequestError(
			"host %s can't reserve %dM memory for each isolated device, not enough", host.Name, *input.ReservedMemory)
	}
	caps := host.GetAttachedLocalStorageCapacity()
	if input.ReservedStorage != nil && caps.Capacity < int64(*input.ReservedStorage*len(devs)) {
		return nil, httperrors.NewBadRequestError(
			"host %s can't reserve %dM storage for each isolated device, not enough", host.Name, input.ReservedStorage)
	}
	defer func() {
		go host.ClearSchedDescCache()
	}()
	for i := 0; i < len(devs); i++ {
		_, err := db.Update(&devs[i], func() error {
			if input.ReservedCpu != nil {
				devs[i].ReservedCpu = *input.ReservedCpu
			}
			if input.ReservedMemory != nil {
				devs[i].ReservedMemory = *input.ReservedMemory
			}
			if input.ReservedStorage != nil {
				devs[i].ReservedStorage = *input.ReservedStorage
			}
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "update isolated device")
		}
	}
	return nil, nil
}

func (manager *SHostManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SZoneResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SZoneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}

func (manager *SHostManager) FetchHostByExtId(extid string) *SHost {
	host := SHost{}
	host.SetModelManager(manager, &host)
	err := manager.Query().Equals("external_id", extid).First(&host)
	if err != nil {
		log.Errorf("fetchHostByExtId fail %s", err)
		return nil
	} else {
		return &host
	}
}

func (host *SHost) IsAssignable(ctx context.Context, userCred mcclient.TokenCredential) error {
	if db.IsAdminAllowPerform(ctx, userCred, host, "assign-host") {
		return nil
	} else if db.IsDomainAllowPerform(ctx, userCred, host, "assign-host") &&
		(userCred.GetProjectDomainId() == host.DomainId ||
			host.PublicScope == string(rbacscope.ScopeSystem) ||
			(host.PublicScope == string(rbacscope.ScopeDomain) && utils.IsInStringArray(userCred.GetProjectDomainId(), host.GetSharedDomains()))) {
		return nil
	} else {
		return httperrors.NewNotSufficientPrivilegeError("Only system admin can assign host")
	}
}

func (manager *SHostManager) initHostname() error {
	hosts := []SHost{}
	q := manager.Query().IsNullOrEmpty("hostname")
	err := db.FetchModelObjects(manager, q, &hosts)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range hosts {
		db.Update(&hosts[i], func() error {
			hostname, _ := manager.SHostnameResourceBaseManager.ValidateHostname(
				hosts[i].Hostname,
				"",
				api.HostnameInput{
					Hostname: hosts[i].Name,
				},
			)
			hosts[i].Hostname = hostname.Hostname
			return nil
		})
	}
	return nil
}

func (manager *SHostManager) InitializeData() error {
	return manager.initHostname()
}

func (hh *SHost) PerformProbeIsolatedDevices(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return hh.GetHostDriver().RequestProbeIsolatedDevices(ctx, userCred, hh, data)
}

func (hh *SHost) GetPinnedCpusetCores(ctx context.Context, userCred mcclient.TokenCredential) (map[string][]int, error) {
	gsts, err := hh.GetGuests()
	if err != nil {
		return nil, errors.Wrap(err, "Get all guests")
	}
	ret := make(map[string][]int, 0)
	for _, gst := range gsts {
		pinned, err := gst.getPinnedCpusetCores(ctx, userCred)
		if err != nil {
			return nil, errors.Wrapf(err, "get guest %s pinned cpuset cores", gst.GetName())
		}
		ret[gst.GetId()] = pinned
	}
	return ret, nil
}

func (h *SHost) PerformSyncGuestNicTraffics(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	guestTraffics, err := data.GetMap()
	if err != nil {
		return nil, errors.Wrap(err, "get guest traffics")
	}
	for guestId, nicTraffics := range guestTraffics {
		nicTrafficMap := make(map[string]api.SNicTrafficRecord)
		err = nicTraffics.Unmarshal(&nicTrafficMap)
		if err != nil {
			log.Errorf("failed unmarshal guest %s nic traffics %s", guestId, err)
			continue
		}

		guest := GuestManager.FetchGuestById(guestId)
		gns, err := guest.GetNetworks("")
		if err != nil {
			log.Errorf("failed fetch guest %s networks %s", guestId, err)
			continue
		}
		for i := range gns {
			nicTraffic, ok := nicTrafficMap[strconv.Itoa(int(gns[i].Index))]
			if !ok {
				continue
			}
			if err = gns[i].UpdateNicTrafficUsed(nicTraffic.RxTraffic, nicTraffic.TxTraffic); err != nil {
				log.Errorf("failed update guestnetwork %d traffic used %s", gns[i].RowId, err)
				continue
			}
		}
	}
	return nil, nil
}

func (h *SHost) GetDetailsAppOptions(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return h.Request(ctx, userCred, httputils.GET, "/app-options", nil, nil)
}

func (hh *SHost) IsAttach2Wire(wireId string) bool {
	netifs := hh.getNetifsOnWire(wireId)
	return len(netifs) > 0
}

func (h *SHost) updateNotify(ctx context.Context, userCred mcclient.TokenCredential) {
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Action: notifyclient.ActionUpdate,
		Obj:    h,
	})
}
