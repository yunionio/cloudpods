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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/multicloud/esxi"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SHostManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	db.SHostNameValidatorManager
	SZoneResourceBaseManager
	SManagedResourceBaseManager
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
}

type SHost struct {
	db.SEnabledStatusInfrasResourceBase
	db.SExternalizedResourceBase
	SZoneResourceBase `update:""`
	SManagedResourceBase
	SBillingResourceBase

	// 机架
	Rack string `width:"16" charset:"ascii" nullable:"true" get:"domain" update:"domain" create:"domain_optional"`
	// 机位
	Slots string `width:"16" charset:"ascii" nullable:"true" get:"domain" update:"domain" create:"domain_optional"`

	// 管理口MAC
	AccessMac string `width:"32" charset:"ascii" nullable:"false" index:"true" list:"domain" update:"domain"`

	// 管理口Ip地址
	AccessIp string `width:"16" charset:"ascii" nullable:"true" list:"domain"`

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
	CpuDesc string `width:"64" charset:"ascii" nullable:"true" get:"domain" update:"domain" create:"domain_optional"`
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

	// 存储大小,单位Mb
	StorageSize int `nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
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
	Version string `width:"64" charset:"ascii" list:"domain" update:"domain" create:"domain_optional"`
	// OVN软件版本
	OvnVersion string `width:"64" charset:"ascii" list:"domain" update:"domain" create:"domain_optional"`

	IsBaremetal bool `nullable:"true" default:"false" list:"domain" update:"domain" create:"domain_optional"`

	// 是否处于维护状态
	IsMaintenance bool `nullable:"true" default:"false" list:"domain"`

	LastPingAt        time.Time ``
	EnableHealthCheck bool      `nullable:"true" default:"false"`

	ResourceType string `width:"36" charset:"ascii" nullable:"false" list:"domain" update:"domain" create:"domain_optional" default:"shared"`

	RealExternalId string `width:"256" charset:"utf8" get:"domain"`

	// 是否为导入的宿主机
	IsImport bool `nullable:"true" default:"false" list:"domain" create:"domain_optional"`

	// 是否允许PXE启动
	EnablePxeBoot tristate.TriState `nullable:"false" default:"true" list:"domain" create:"domain_optional" update:"domain"`

	// 主机UUID
	Uuid string `width:"64" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`

	// 主机启动模式, 可能值位PXE和ISO
	BootMode string `width:"8" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`

	// IPv4地址，作为么有云vpc访问外网时的网关
	OvnMappedIpAddr string `width:"16" charset:"ascii" nullable:"true" list:"user"`
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

	anyMac := query.AnyMac
	if len(anyMac) > 0 {
		anyMac := netutils.FormatMacAddr(anyMac)
		if len(anyMac) == 0 {
			return nil, httperrors.NewInputParameterError("invalid any_mac address")
		}
		netif, _ := NetInterfaceManager.FetchByMac(anyMac)
		if netif != nil && len(netif.BaremetalId) > 0 {
			q = q.Equals("id", netif.BaremetalId)
		} else {
			q = q.Equals("access_mac", anyMac)
		}
	}
	if len(query.AnyIp) > 0 {
		hn := HostnetworkManager.Query("baremetal_id").Contains("ip_addr", query.AnyIp).SubQuery()
		q = q.Filter(sqlchemy.OR(
			sqlchemy.Contains(q.Field("access_ip"), query.AnyIp),
			sqlchemy.Contains(q.Field("ipmi_ip"), query.AnyIp),
			sqlchemy.In(q.Field("id"), hn),
		))
	}
	// var scopeQuery *sqlchemy.SSubQuery

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
		hostwires := HostwireManager.Query().SubQuery()
		scopeQuery := hostwires.Query(hostwires.Field("host_id")).Equals("wire_id", wire.GetId()).SubQuery()
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
		hostwires := HostwireManager.Query().SubQuery()
		networks := NetworkManager.Query().SubQuery()
		providers := CloudproviderManager.Query().SubQuery()

		hostQ1 := hosts.Query(hosts.Field("id"))
		hostQ1 = hostQ1.Join(providers, sqlchemy.Equals(hosts.Field("manager_id"), providers.Field("id")))
		hostQ1 = hostQ1.Join(hostwires, sqlchemy.Equals(hosts.Field("id"), hostwires.Field("host_id")))
		hostQ1 = hostQ1.Join(networks, sqlchemy.Equals(hostwires.Field("wire_id"), networks.Field("wire_id")))
		hostQ1 = hostQ1.Filter(sqlchemy.IsTrue(providers.Field("enabled")))
		hostQ1 = hostQ1.Filter(sqlchemy.In(providers.Field("status"), api.CLOUD_PROVIDER_VALID_STATUS))
		hostQ1 = hostQ1.Filter(sqlchemy.In(providers.Field("health_status"), api.CLOUD_PROVIDER_VALID_HEALTH_STATUS))
		hostQ1 = hostQ1.Filter(sqlchemy.Equals(networks.Field("status"), api.NETWORK_STATUS_AVAILABLE))
		hostQ1 = hostQ1.Filter(sqlchemy.IsTrue(hosts.Field("enabled")))

		hostQ2 := hosts.Query(hosts.Field("id"))
		hostQ2 = hostQ2.Join(hostwires, sqlchemy.Equals(hosts.Field("id"), hostwires.Field("host_id")))
		hostQ2 = hostQ2.Join(networks, sqlchemy.Equals(hostwires.Field("wire_id"), networks.Field("wire_id")))
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

	if len(query.Rack) > 0 {
		q = q.In("rack", query.Rack)
	}
	if len(query.Slots) > 0 {
		q = q.In("slots", query.Slots)
	}
	if len(query.AccessMac) > 0 {
		q = q.In("access_mac", query.AccessMac)
	}
	if len(query.AccessIp) > 0 {
		q = q.In("access_ip", query.AccessIp)
	}
	if len(query.SN) > 0 {
		q = q.In("sn", query.SN)
	}
	if len(query.CpuCount) > 0 {
		q = q.In("cpu_count", query.CpuCount)
	}
	if len(query.MemSize) > 0 {
		q = q.In("mem_size", query.MemSize)
	}
	if len(query.StorageType) > 0 {
		q = q.In("storage_type", query.StorageType)
	}
	if len(query.IpmiIp) > 0 {
		q = q.In("ipmi_ip", query.IpmiIp)
	}
	if len(query.HostStatus) > 0 {
		q = q.In("host_status", query.HostStatus)
	}
	if len(query.HostType) > 0 {
		q = q.In("host_type", query.HostType)
	}
	if len(query.Version) > 0 {
		q = q.In("version", query.Version)
	}
	if len(query.OvnVersion) > 0 {
		q = q.In("ovn_version", query.OvnVersion)
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
	if len(query.Uuid) > 0 {
		q = q.In("uuid", query.Uuid)
	}
	if len(query.BootMode) > 0 {
		q = q.In("boot_mode", query.BootMode)
	}
	if len(query.CpuArchitecture) > 0 {
		q = q.Equals("cpu_architecture", query.CpuArchitecture)
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
					if net.GetVpc().Id != api.DEFAULT_VPC_ID {
						q = q.IsNotEmpty("ovn_version")
					} else {
						if !utils.IsInStringArray(net.WireId, wires) {
							wires = append(wires, net.WireId)
							hostwires := HostwireManager.Query().SubQuery()
							scopeQuery := hostwires.Query(hostwires.Field("host_id")).Equals("wire_id", net.WireId).SubQuery()
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

func (self *SHost) IsArmHost() bool {
	return self.CpuArchitecture == apis.OS_ARCH_AARCH64
}

func (self *SHost) GetZone() *SZone {
	if len(self.ZoneId) == 0 {
		return nil
	}
	zone, _ := ZoneManager.FetchById(self.ZoneId)
	if zone != nil {
		return zone.(*SZone)
	}
	return nil
}

func (self *SHost) GetRegion() *SCloudregion {
	zone := self.GetZone()
	if zone != nil {
		return zone.GetRegion()
	}
	return nil
}

func (self *SHost) GetCpuCount() int {
	if self.CpuReserved > 0 && self.CpuReserved < self.CpuCount {
		return int(self.CpuCount - self.CpuReserved)
	} else {
		return int(self.CpuCount)
	}
}

func (self *SHost) GetMemSize() int {
	if self.MemReserved > 0 && self.MemReserved < self.MemSize {
		return self.MemSize - self.MemReserved
	} else {
		return self.MemSize
	}
}

func (self *SHost) GetMemoryOvercommitBound() float32 {
	if self.MemCmtbound > 0 {
		return self.MemCmtbound
	}
	return options.Options.DefaultMemoryOvercommitBound
}

func (self *SHost) GetVirtualMemorySize() float32 {
	return float32(self.GetMemSize()) * self.GetMemoryOvercommitBound()
}

func (self *SHost) GetCPUOvercommitBound() float32 {
	if self.CpuCmtbound > 0 {
		return self.CpuCmtbound
	}
	return options.Options.DefaultCPUOvercommitBound
}

func (self *SHost) GetVirtualCPUCount() float32 {
	return float32(self.GetCpuCount()) * self.GetCPUOvercommitBound()
}

/*func (manager *SHostManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (manager *SHostManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SHost) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SHost) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return userCred.IsSystemAdmin()
}

func (self *SHost) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return userCred.IsSystemAdmin()
}*/

func (self *SHost) ValidateDeleteCondition(ctx context.Context) error {
	return self.validateDeleteCondition(ctx, false)
}

func (self *SHost) ValidatePurgeCondition(ctx context.Context) error {
	return self.validateDeleteCondition(ctx, true)
}

func (self *SHost) validateDeleteCondition(ctx context.Context, purge bool) error {
	if !purge && self.IsBaremetal && self.HostType != api.HOST_TYPE_BAREMETAL {
		return httperrors.NewInvalidStatusError("Host is a converted baremetal, should be unconverted before delete")
	}
	if self.GetEnabled() {
		return httperrors.NewInvalidStatusError("Host is not disabled")
	}
	cnt, err := self.GetGuestCount()
	if err != nil {
		return httperrors.NewInternalServerError("getGuestCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("Not an empty host")
	}
	for _, hoststorage := range self.GetHoststorages() {
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
	return self.SEnabledStatusInfrasResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SHost) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("Host delete do nothing")
	return nil
}

func (self *SHost) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if self.IsBaremetal {
		return self.StartDeleteBaremetalTask(ctx, userCred, "")
	} else {
		return self.RealDelete(ctx, userCred)
	}
}

func (self *SHost) StartDeleteBaremetalTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SHost) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	DeleteResourceJointSchedtags(self, ctx, userCred)

	IsolatedDeviceManager.DeleteDevicesByHost(ctx, userCred, self)

	for _, hoststorage := range self.GetHoststorages() {
		storage := hoststorage.GetStorage()
		if storage != nil && storage.IsLocal() {
			cnt, err := storage.GetDiskCount()
			if err != nil {
				return errors.Wrapf(err, "GetDiskCount")
			}
			if cnt > 0 {
				return httperrors.NewNotEmptyError("Inconsistent: local storage is not empty???")
			}
		}
	}
	for _, hoststorage := range self.GetHoststorages() {
		storage := hoststorage.GetStorage()
		hoststorage.Delete(ctx, userCred)
		if storage != nil && storage.IsLocal() {
			storage.Delete(ctx, userCred)
		}
	}
	for _, bn := range self.GetBaremetalnetworks() {
		self.DeleteBaremetalnetwork(ctx, userCred, &bn, false)
	}
	for _, netif := range self.GetNetInterfaces() {
		netif.Remove(ctx, userCred)
	}
	for _, hostwire := range self.GetHostwires() {
		hostwire.Detach(ctx, userCred)
		// hostwire.Delete(ctx, userCred) ???
	}
	baremetalStorage := self.GetBaremetalstorage()
	if baremetalStorage != nil {
		store := baremetalStorage.GetStorage()
		baremetalStorage.Delete(ctx, userCred)
		if store != nil {
			store.Delete(ctx, userCred)
		}
	}
	return self.SEnabledStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SHost) GetHoststoragesQuery() *sqlchemy.SQuery {
	return HoststorageManager.Query().Equals("host_id", self.Id)
}

func (self *SHost) GetStorageCount() (int, error) {
	return self.GetHoststoragesQuery().CountWithError()
}

func (self *SHost) GetHoststorages() []SHoststorage {
	hoststorages := make([]SHoststorage, 0)
	q := self.GetHoststoragesQuery()
	err := db.FetchModelObjects(HoststorageManager, q, &hoststorages)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		log.Errorf("GetHoststorages error %s", err)
		return nil
	}
	return hoststorages
}

func (self *SHost) GetHoststorageOfId(storageId string) *SHoststorage {
	hoststorage := SHoststorage{}
	hoststorage.SetModelManager(HoststorageManager, &hoststorage)
	err := self.GetHoststoragesQuery().Equals("storage_id", storageId).First(&hoststorage)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("GetHoststorageOfId fail %s", err)
		}
		return nil
	}
	return &hoststorage
}

func (self *SHost) GetHoststorageByExternalId(extId string) *SHoststorage {
	hoststorage := SHoststorage{}
	hoststorage.SetModelManager(HoststorageManager, &hoststorage)

	hoststorages := HoststorageManager.Query().SubQuery()
	storages := StorageManager.Query().SubQuery()
	q := hoststorages.Query()
	q = q.Join(storages, sqlchemy.Equals(hoststorages.Field("storage_id"), storages.Field("id")))
	q = q.Filter(sqlchemy.Equals(hoststorages.Field("host_id"), self.Id))
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

func (self *SHost) GetStorageByFilePath(path string) *SStorage {
	hoststorages := self.GetHoststorages()
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

func (self *SHost) GetBaremetalstorage() *SHoststorage {
	if !self.IsBaremetal {
		return nil
	}
	hoststorages := HoststorageManager.Query().SubQuery()
	storages := StorageManager.Query().SubQuery()
	q := hoststorages.Query()
	q = q.Join(storages, sqlchemy.AND(sqlchemy.Equals(storages.Field("id"), hoststorages.Field("storage_id")),
		sqlchemy.IsFalse(storages.Field("deleted"))))
	q = q.Filter(sqlchemy.Equals(storages.Field("storage_type"), api.STORAGE_BAREMETAL))
	q = q.Filter(sqlchemy.Equals(hoststorages.Field("host_id"), self.Id))
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

func (self *SHost) SaveCleanUpdates(doUpdate func() error) (map[string]sqlchemy.SUpdateDiff, error) {
	return self.saveUpdates(doUpdate, true)
}

func (self *SHost) SaveUpdates(doUpdate func() error) (map[string]sqlchemy.SUpdateDiff, error) {
	return self.saveUpdates(doUpdate, false)
}

func (self *SHost) saveUpdates(doUpdate func() error, doSchedClean bool) (map[string]sqlchemy.SUpdateDiff, error) {
	diff, err := db.Update(self, doUpdate)
	if err != nil {
		return nil, err
	}
	if doSchedClean {
		self.ClearSchedDescCache()
	}
	return diff, nil
}

func (self *SHost) AllowPerformUpdateStorage(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return db.IsAdminAllowPerform(userCred, self, "update-storage")
}

func (self *SHost) PerformUpdateStorage(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	bs := self.GetBaremetalstorage()
	capacity, _ := data.Int("capacity")
	zoneId, _ := data.GetString("zone_id")
	storageCacheId, _ := data.GetString("storagecache_id")
	if bs == nil {
		// 1. create storage
		storage := SStorage{}
		storage.Name = fmt.Sprintf("storage%s", self.GetName())
		storage.Capacity = capacity
		storage.StorageType = api.STORAGE_BAREMETAL
		storage.MediumType = self.StorageType
		storage.Cmtbound = 1.0
		storage.Status = api.STORAGE_ONLINE
		storage.Enabled = tristate.True
		storage.ZoneId = zoneId
		storage.StoragecacheId = storageCacheId
		storage.DomainId = self.DomainId
		storage.DomainSrc = string(apis.OWNER_SOURCE_LOCAL)
		err := StorageManager.TableSpec().Insert(ctx, &storage)
		if err != nil {
			return nil, fmt.Errorf("Create baremetal storage error: %v", err)
		}
		storage.SetModelManager(StorageManager, &storage)
		db.OpsLog.LogEvent(&storage, db.ACT_CREATE, storage.GetShortDesc(ctx), userCred)
		// 2. create host storage
		bmStorage := SHoststorage{}
		bmStorage.HostId = self.Id
		bmStorage.StorageId = storage.Id
		bmStorage.RealCapacity = capacity
		bmStorage.MountPoint = ""
		err = HoststorageManager.TableSpec().Insert(ctx, &bmStorage)
		if err != nil {
			return nil, fmt.Errorf("Create baremetal hostStorage error: %v", err)
		}
		bmStorage.SetModelManager(HoststorageManager, &bmStorage)
		db.OpsLog.LogAttachEvent(ctx, self, &storage, userCred, bmStorage.GetShortDesc(ctx))
		bmStorage.syncLocalStorageShare(ctx, userCred)
		return nil, nil
	}
	storage := bs.GetStorage()
	//if capacity != int64(storage.Capacity)  {
	diff, err := db.Update(storage, func() error {
		storage.Capacity = capacity
		storage.StoragecacheId = storageCacheId
		storage.Enabled = tristate.True
		storage.DomainId = self.DomainId
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

func (self *SHost) GetFetchUrl(disableHttps bool) string {
	managerUrl, err := url.Parse(self.ManagerUri)
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

func (self *SHost) GetAttachedEnabledHostStorages(storageType []string) []SStorage {
	return self._getAttachedStorages(tristate.False, tristate.True, storageType)
}

func (self *SHost) _getAttachedStorages(isBaremetal tristate.TriState, enabled tristate.TriState, storageType []string) []SStorage {
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
	q = q.Filter(sqlchemy.Equals(hoststorages.Field("host_id"), self.Id))
	ret := make([]SStorage, 0)
	err := db.FetchModelObjects(StorageManager, q, &ret)
	if err != nil {
		log.Errorf("GetAttachedStorages fail %s", err)
		return nil
	}
	return ret
}

func (self *SHost) SyncAttachedStorageStatus() {
	storages := self.GetAttachedEnabledHostStorages(nil)
	if storages != nil {
		for _, storage := range storages {
			storage.SyncStatusWithHosts()
		}
		self.ClearSchedDescCache()
	}
}

func (self *SHostManager) IsNewNameUnique(name string, userCred mcclient.TokenCredential, kwargs *jsonutils.JSONDict) (bool, error) {
	q := self.Query().Equals("name", name)
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

func (self *SHostManager) AllowGetPropertyBmStartRegisterScript(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SHostManager) GetPropertyBmStartRegisterScript(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
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

func (self *SHostManager) AllowGetPropertyNodeCount(ctx context.Context, userCred mcclient.TokenCredential, query api.HostListInput) bool {
	return userCred.HasSystemAdminPrivilege()
}

func (self *SHostManager) GetPropertyNodeCount(ctx context.Context, userCred mcclient.TokenCredential, query api.HostListInput) (jsonutils.JSONObject, error) {
	hosts := self.Query().SubQuery()
	q := hosts.Query(hosts.Field("host_type"), sqlchemy.SUM("node_count_total", hosts.Field("node_count")))
	q, err := self.ListItemFilter(ctx, q, userCred, query)
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
			log.Errorf("Get host node count scan err: %v", err)
			return ret, nil
		}

		ret.Add(jsonutils.NewInt(count), hostType)
	}

	return ret, nil
}

func (self *SHostManager) ClearAllSchedDescCache() error {
	return self.ClearSchedDescSessionCache("", "")
}

func (self *SHostManager) ClearSchedDescCache(hostId string) error {
	return self.ClearSchedDescSessionCache(hostId, "")
}

func (self *SHostManager) ClearSchedDescSessionCache(hostId, sessionId string) error {
	s := auth.GetAdminSession(context.Background(), options.Options.Region, "")
	return modules.SchedManager.CleanCache(s, hostId, sessionId, false)
}

func (self *SHost) ClearSchedDescCache() error {
	return self.ClearSchedDescSessionCache("")
}

func (self *SHost) ClearSchedDescSessionCache(sessionId string) error {
	return HostManager.ClearSchedDescSessionCache(self.Id, sessionId)
}

// sync clear sched desc on scheduler side
func (self *SHostManager) SyncClearSchedDescSessionCache(hostId, sessionId string) error {
	s := auth.GetAdminSession(context.Background(), options.Options.Region, "")
	return modules.SchedManager.CleanCache(s, hostId, sessionId, true)
}

func (self *SHost) SyncCleanSchedDescCache() error {
	return self.SyncClearSchedDescSessionCache("")
}

func (self *SHost) SyncClearSchedDescSessionCache(sessionId string) error {
	return HostManager.SyncClearSchedDescSessionCache(self.Id, sessionId)
}

func (self *SHost) AllowGetDetailsSpec(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGetSpec(userCred, self, "spec")
}

func (self *SHost) GetDetailsSpec(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return GetModelSpec(HostManager, self)
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

func (self *SHost) GetSpec(statusCheck bool) *jsonutils.JSONDict {
	if statusCheck {
		if !self.GetEnabled() {
			return nil
		}
		if utils.IsInStringArray(self.Status, []string{api.BAREMETAL_INIT, api.BAREMETAL_PREPARE_FAIL, api.BAREMETAL_PREPARE}) ||
			self.GetBaremetalServer() != nil {
			return nil
		}
		if self.MemSize == 0 || self.CpuCount == 0 {
			return nil
		}
		if self.ResourceType == api.HostResourceTypePrepaidRecycle {
			cnt, err := self.GetGuestCount()
			if err != nil {
				return nil
			}
			if cnt > 0 {
				// occupied
				return nil
			}
		}

		if len(self.ManagerId) > 0 {
			providerObj, _ := CloudproviderManager.FetchById(self.ManagerId)
			if providerObj == nil {
				return nil
			}
			provider := providerObj.(*SCloudprovider)
			if !provider.IsAvailable() {
				return nil
			}
		}
	}
	spec := self.GetHardwareSpecification()
	specInfo := new(api.HostSpec)
	if err := spec.Unmarshal(specInfo); err != nil {
		return spec
	}
	nifs := self.GetNetInterfaces()
	var nicCount int
	for _, nif := range nifs {
		if nif.NicType != api.NIC_TYPE_IPMI {
			nicCount++
		}
	}
	specInfo.NicCount = nicCount

	var manufacture string
	var model string
	if self.SysInfo != nil {
		manufacture, _ = self.SysInfo.GetString("manufacture")
		model, _ = self.SysInfo.GetString("model")
	}
	if manufacture == "" {
		manufacture = "Unknown"
	}
	if model == "" {
		model = "Unknown"
	}
	specInfo.Manufacture = strings.ReplaceAll(manufacture, " ", "_")
	specInfo.Model = strings.ReplaceAll(model, " ", "_")
	devices := IsolatedDeviceManager.FindByHost(self.Id)
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

func (self *SHost) GetHardwareSpecification() *jsonutils.JSONDict {
	spec := &api.HostSpec{
		Cpu: int(self.CpuCount),
		Mem: self.MemSize,
	}
	if self.StorageInfo != nil {
		spec.Disk = GetDiskSpecV2(self.StorageInfo)
		spec.Driver = self.StorageDriver
	}
	ret := spec.JSON(spec)
	if self.StorageInfo != nil {
		ret.Set("storage_info", self.StorageInfo)
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
	info.Capacity = cap.Capacity
	info.UsedCapacity = cap.Used
	info.WasteCapacity = cap.Wasted
	info.VirtualCapacity = cap.VCapacity
	info.CommitRate = cap.GetCommitRate()
	info.FreeCapacity = cap.GetFree()
	return info
}

func (self *SHost) GetAttachedLocalStorageCapacity() SStorageCapacity {
	ret := SStorageCapacity{}
	storages := self.GetAttachedEnabledHostStorages(api.HOST_STORAGE_LOCAL_TYPES)
	for _, s := range storages {
		ret.Add(s.getStorageCapacity())
	}
	return ret
}

func (self *SHost) GetAttachedLocalStorages() []SStorage {
	return self.GetAttachedEnabledHostStorages(api.HOST_STORAGE_LOCAL_TYPES)
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

func getLeastUsedStorage(storages []SStorage, backend string) *SStorage {
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

func (self *SHost) GetLeastUsedStorage(backend string) *SStorage {
	storages := self.GetAttachedEnabledHostStorages(nil)
	if storages != nil {
		return getLeastUsedStorage(storages, backend)
	}
	return nil
}

func (self *SHost) GetWiresQuery() *sqlchemy.SQuery {
	return HostwireManager.Query().Equals("host_id", self.Id)
}

func (self *SHost) GetWireCount() (int, error) {
	return self.GetWiresQuery().CountWithError()
}

func (self *SHost) GetHostwires() []SHostwire {
	hw := make([]SHostwire, 0)
	q := self.GetWiresQuery()
	err := db.FetchModelObjects(HostwireManager, q, &hw)
	if err != nil {
		log.Errorf("GetWires error %s", err)
		return nil
	}
	return hw
}

func (self *SHost) getAttachedWires() []SWire {
	wires := WireManager.Query().SubQuery()
	hostwires := HostwireManager.Query().SubQuery()
	q := wires.Query()
	q = q.Join(hostwires, sqlchemy.AND(sqlchemy.IsFalse(hostwires.Field("deleted")),
		sqlchemy.Equals(hostwires.Field("wire_id"), wires.Field("id"))))
	q = q.Filter(sqlchemy.Equals(hostwires.Field("host_id"), self.Id))
	ret := make([]SWire, 0)
	err := db.FetchModelObjects(WireManager, q, &ret)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return ret
}

func (self *SHost) GetMasterHostwire() *SHostwire {
	hw := SHostwire{}
	hw.SetModelManager(HostwireManager, &hw)

	q := self.GetWiresQuery().IsTrue("is_master")
	err := q.First(&hw)
	if err != nil {
		log.Errorf("GetMasterHostwire %s", err)
		return nil
	}
	return &hw
}

func (self *SHost) GetMasterWire() *SWire {
	wires := WireManager.Query().SubQuery()
	hostwires := HostwireManager.Query().SubQuery()
	q := wires.Query()
	q = q.Join(hostwires, sqlchemy.AND(sqlchemy.Equals(hostwires.Field("wire_id"), wires.Field("id")),
		sqlchemy.IsFalse(hostwires.Field("deleted"))))
	q = q.Filter(sqlchemy.Equals(hostwires.Field("host_id"), self.Id))
	q = q.Filter(sqlchemy.IsTrue(hostwires.Field("is_master")))
	wire := SWire{}
	wire.SetModelManager(WireManager, &wire)

	err := q.First(&wire)
	if err != nil {
		log.Errorf("GetMasterWire fail %s", err)
		return nil
	}
	return &wire
}

func (self *SHost) getHostwires() ([]SHostwire, error) {
	hostwires := make([]SHostwire, 0)
	q := self.GetWiresQuery()
	err := db.FetchModelObjects(HostwireManager, q, &hostwires)
	if err != nil {
		return nil, err
	}
	return hostwires, nil
}

func (self *SHost) getHostwiresOfId(wireId string) []SHostwire {
	hostwires := make([]SHostwire, 0)

	q := self.GetWiresQuery().Equals("wire_id", wireId)
	err := db.FetchModelObjects(HostwireManager, q, &hostwires)
	if err != nil {
		log.Errorf("getHostwiresOfId fail %s", err)
		return nil
	}
	return hostwires
}

func (self *SHost) getHostwireOfIdAndMac(wireId string, mac string) *SHostwire {
	hostwire := SHostwire{}
	hostwire.SetModelManager(HostwireManager, &hostwire)

	q := self.GetWiresQuery().Equals("wire_id", wireId)
	q = q.Equals("mac_addr", mac)
	err := q.First(&hostwire)
	if err != nil {
		log.Errorf("getHostwireOfIdAndMac fail %s", err)
		return nil
	}
	return &hostwire
}

func (self *SHost) GetGuestsQuery() *sqlchemy.SQuery {
	return GuestManager.Query().Equals("host_id", self.Id)
}

func (self *SHost) GetGuests() ([]SGuest, error) {
	q := self.GetGuestsQuery()
	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return guests, nil
}

func (self *SHost) GetKvmGuests() []SGuest {
	q := GuestManager.Query().Equals("host_id", self.Id).Equals("hypervisor", api.HYPERVISOR_KVM)
	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("GetGuests %s", err)
		return nil
	}
	return guests
}

func (self *SHost) GetGuestsMasterOnThisHost() []SGuest {
	q := self.GetGuestsQuery().IsNotEmpty("backup_host_id")
	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("GetGuests %s", err)
		return nil
	}
	return guests
}

func (self *SHost) GetGuestsBackupOnThisHost() []SGuest {
	q := GuestManager.Query().Equals("backup_host_id", self.Id)
	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("GetGuests %s", err)
		return nil
	}
	return guests
}

func (self *SHost) GetGuestCount() (int, error) {
	q := self.GetGuestsQuery()
	return q.CountWithError()
}

func (self *SHost) GetContainerCount(status []string) (int, error) {
	q := self.GetGuestsQuery()
	q = q.Filter(sqlchemy.Equals(q.Field("hypervisor"), api.HYPERVISOR_CONTAINER))
	if len(status) > 0 {
		q = q.In("status", status)
	}
	return q.CountWithError()
}

func (self *SHost) GetNonsystemGuestCount() (int, error) {
	q := self.GetGuestsQuery()
	q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("is_system")), sqlchemy.IsFalse(q.Field("is_system"))))
	return q.CountWithError()
}

func (self *SHost) GetRunningGuestCount() (int, error) {
	q := self.GetGuestsQuery()
	q = q.In("status", api.VM_RUNNING_STATUS)
	return q.CountWithError()
}

func (self *SHost) GetNotReadyGuestsMemorySize() (int, error) {
	guests := GuestManager.Query().SubQuery()
	q := guests.Query(sqlchemy.COUNT("guest_count"),
		sqlchemy.SUM("guest_vcpu_count", guests.Field("vcpu_count")),
		sqlchemy.SUM("guest_vmem_size", guests.Field("vmem_size")))
	cond := sqlchemy.OR(sqlchemy.Equals(q.Field("host_id"), self.Id),
		sqlchemy.Equals(q.Field("backup_host_id"), self.Id))
	q = q.Filter(cond)
	q = q.NotEquals("status", api.VM_READY)
	stat := SHostGuestResourceUsage{}
	err := q.First(&stat)
	if err != nil {
		return -1, err
	}
	return stat.GuestVmemSize, nil
}

func (self *SHost) GetRunningGuestMemorySize() int {
	res := self.getGuestsResource(api.VM_RUNNING)
	if res != nil {
		return res.GuestVmemSize
	}
	return -1
}

func (self *SHost) GetBaremetalnetworksQuery() *sqlchemy.SQuery {
	return HostnetworkManager.Query().Equals("baremetal_id", self.Id)
}

func (self *SHost) GetBaremetalnetworks() []SHostnetwork {
	q := self.GetBaremetalnetworksQuery()
	hns := make([]SHostnetwork, 0)
	err := db.FetchModelObjects(HostnetworkManager, q, &hns)
	if err != nil {
		log.Errorf("GetBaremetalnetworks error: %s", err)
	}
	return hns
}

func (self *SHost) GetAttach2Network(netId string) *SHostnetwork {
	q := self.GetBaremetalnetworksQuery()
	netifs := NetInterfaceManager.Query().Equals("baremetal_id", self.Id)
	netifs = netifs.Filter(sqlchemy.OR(
		sqlchemy.IsNullOrEmpty(netifs.Field("nic_type")),
		sqlchemy.NotEquals(netifs.Field("nic_type"), api.NIC_TYPE_IPMI),
	))
	netifsSub := netifs.SubQuery()
	q = q.Join(netifsSub, sqlchemy.Equals(q.Field("mac_addr"), netifsSub.Field("mac")))
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

func (self *SHost) GetNetInterfaces() []SNetInterface {
	q := NetInterfaceManager.Query().Equals("baremetal_id", self.Id).Asc("index")
	netifs := make([]SNetInterface, 0)
	err := db.FetchModelObjects(NetInterfaceManager, q, &netifs)
	if err != nil {
		log.Errorf("GetNetInterfaces fail %s", err)
		return nil
	}
	return netifs
}

func (self *SHost) GetAdminNetInterface() *SNetInterface {
	netif := SNetInterface{}
	netif.SetModelManager(NetInterfaceManager, &netif)

	q := NetInterfaceManager.Query().Equals("baremetal_id", self.Id).Equals("nic_type", api.NIC_TYPE_ADMIN)
	err := q.First(&netif)
	if err != nil {
		log.Errorf("GetAdminNetInterface fail %s", err)
		return nil
	}
	return &netif
}

func (self *SHost) GetNetInterface(mac string) *SNetInterface {
	netif, _ := NetInterfaceManager.FetchByMac(mac)
	if netif != nil && netif.BaremetalId == self.Id {
		return netif
	}
	return nil
}

func (self *SHost) DeleteBaremetalnetwork(ctx context.Context, userCred mcclient.TokenCredential, bn *SHostnetwork, reserve bool) {
	net := bn.GetNetwork()
	bn.Delete(ctx, userCred)
	db.OpsLog.LogDetachEvent(ctx, self, net, userCred, nil)
	if reserve && net != nil && len(bn.IpAddr) > 0 && regutils.MatchIP4Addr(bn.IpAddr) {
		ReservedipManager.ReserveIP(userCred, net, bn.IpAddr, "Delete baremetalnetwork to reserve")
	}
}

func (self *SHost) GetHostDriver() IHostDriver {
	if !utils.IsInStringArray(self.HostType, api.HOST_TYPES) {
		log.Fatalf("Unsupported host type %s", self.HostType)
	}
	return GetHostDriver(self.HostType)
}

func (manager *SHostManager) getHostsByZoneProvider(zone *SZone, provider *SCloudprovider) ([]SHost, error) {
	hosts := make([]SHost, 0)
	q := manager.Query()
	if zone != nil {
		q = q.Equals("zone_id", zone.Id)
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

func (manager *SHostManager) SyncHosts(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, zone *SZone, hosts []cloudprovider.ICloudHost) ([]SHost, []cloudprovider.ICloudHost, compare.SyncResult) {
	lockman.LockRawObject(ctx, "hosts", fmt.Sprintf("%s-%s", zone.Id, provider.Id))
	defer lockman.ReleaseRawObject(ctx, "hosts", fmt.Sprintf("%s-%s", zone.Id, provider.Id))

	localHosts := make([]SHost, 0)
	remoteHosts := make([]cloudprovider.ICloudHost, 0)
	syncResult := compare.SyncResult{}

	dbHosts, err := manager.getHostsByZoneProvider(zone, provider)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

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
		err = commondb[i].syncWithCloudHost(ctx, userCred, commonext[i], provider)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			localHosts = append(localHosts, commondb[i])
			remoteHosts = append(remoteHosts, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		new, err := manager.newFromCloudHost(ctx, userCred, added[i], provider, zone)
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

func (self *SHost) syncRemoveCloudHost(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidatePurgeCondition(ctx)
	if err != nil {
		err = self.SetStatus(userCred, api.HOST_OFFLINE, "sync to delete")
		if err == nil {
			_, err = self.PerformDisable(ctx, userCred, nil, apis.PerformDisableInput{})
		}
		guests, err := self.GetGuests()
		if err != nil {
			return errors.Wrapf(err, "GetGuests")
		}
		for _, guest := range guests {
			err = guest.SetStatus(userCred, api.VM_UNKNOWN, "sync to delete")
			if err != nil {
				return err
			}
		}
	} else {
		err = self.RealDelete(ctx, userCred)
	}
	return err
}

func (self *SHost) syncWithCloudHost(ctx context.Context, userCred mcclient.TokenCredential, extHost cloudprovider.ICloudHost, provider *SCloudprovider) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		// self.Name = extHost.GetName()

		self.Status = extHost.GetStatus()
		self.HostStatus = extHost.GetHostStatus()
		self.AccessIp = extHost.GetAccessIp()
		self.AccessMac = extHost.GetAccessMac()
		self.SN = extHost.GetSN()
		self.SysInfo = extHost.GetSysInfo()
		self.CpuCount = extHost.GetCpuCount()
		self.NodeCount = extHost.GetNodeCount()
		self.CpuDesc = extHost.GetCpuDesc()
		self.CpuMhz = extHost.GetCpuMhz()
		self.MemSize = extHost.GetMemSizeMB()
		self.StorageSize = extHost.GetStorageSizeMB()
		self.StorageType = extHost.GetStorageType()
		self.HostType = extHost.GetHostType()

		if cpuCmt := extHost.GetCpuCmtbound(); cpuCmt > 0 {
			self.CpuCmtbound = cpuCmt
		}

		if memCmt := extHost.GetMemCmtbound(); memCmt > 0 {
			self.MemCmtbound = memCmt
		}

		if reservedMem := extHost.GetReservedMemoryMb(); reservedMem > 0 {
			self.MemReserved = reservedMem
		}

		self.IsEmulated = extHost.IsEmulated()
		self.SetEnabled(extHost.GetEnabled())

		self.IsMaintenance = extHost.GetIsMaintenance()
		self.Version = extHost.GetVersion()

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudZone error %s", err)
		return err
	}

	db.OpsLog.LogSyncUpdate(self, diff, userCred)

	if provider != nil {
		SyncCloudDomain(userCred, self, provider.GetOwnerId())
		self.SyncShareState(ctx, userCred, provider.getAccountShareInfo())
	}

	if err := self.syncSchedtags(ctx, userCred, extHost); err != nil {
		log.Errorf("syncSchedtags fail:  %v", err)
		return err
	}

	if err := HostManager.ClearSchedDescCache(self.Id); err != nil {
		log.Errorf("ClearSchedDescCache for host %s error %v", self.Name, err)
	}

	return nil
}

func (self *SHost) syncWithCloudPrepaidVM(extVM cloudprovider.ICloudVM, host *SHost) error {
	_, err := self.SaveUpdates(func() error {

		self.CpuCount = extVM.GetVcpuCount()
		self.MemSize = extVM.GetVmemSizeMB()

		self.BillingType = extVM.GetBillingType()
		self.ExpiredAt = extVM.GetExpiredAt()

		self.ExternalId = host.ExternalId

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudZone error %s", err)
	}

	if err := HostManager.ClearSchedDescCache(self.Id); err != nil {
		log.Errorf("ClearSchedDescCache for host %s error %v", self.Name, err)
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
		extTagName := sts[i].GetMetadata(METADATA_EXT_SCHEDTAG_KEY, userCred)
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
		extTagName := stag.GetMetadata(METADATA_EXT_SCHEDTAG_KEY, userCred)
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
			err := SchedtagManager.TableSpec().Insert(ctx, st)
			if err != nil {
				return errors.Wrapf(err, "unable to create schedtag %q", stStr)
			}
			st.SetModelManager(SchedtagManager, st)
			st.SetMetadata(ctx, METADATA_EXT_SCHEDTAG_KEY, stStr, userCred)
		}
		// attach
		hostschedtag := &SHostschedtag{
			HostId: s.GetId(),
		}
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

func (manager *SHostManager) newFromCloudHost(ctx context.Context, userCred mcclient.TokenCredential, extHost cloudprovider.ICloudHost, provider *SCloudprovider, izone *SZone) (*SHost, error) {
	host := SHost{}
	host.SetModelManager(manager, &host)

	if izone == nil {
		// onpremise host
		accessIp := extHost.GetAccessIp()
		if len(accessIp) == 0 {
			msg := fmt.Sprintf("fail to find wire for host %s: empty host access ip", extHost.GetName())
			log.Errorf(msg)
			return nil, fmt.Errorf(msg)
		}
		wire, err := WireManager.GetOnPremiseWireOfIp(accessIp)
		if err != nil {
			msg := fmt.Sprintf("fail to find wire for host %s %s: %s", extHost.GetName(), accessIp, err)
			log.Errorf(msg)
			return nil, fmt.Errorf(msg)
		}
		izone = wire.GetZone()
	}

	host.ExternalId = extHost.GetGlobalId()
	host.ZoneId = izone.Id

	host.HostType = extHost.GetHostType()

	host.Status = extHost.GetStatus()
	host.HostStatus = extHost.GetHostStatus()
	host.SetEnabled(extHost.GetEnabled())

	host.AccessIp = extHost.GetAccessIp()
	host.AccessMac = extHost.GetAccessMac()
	host.SN = extHost.GetSN()
	host.SysInfo = extHost.GetSysInfo()
	host.CpuCount = extHost.GetCpuCount()
	host.NodeCount = extHost.GetNodeCount()
	host.CpuDesc = extHost.GetCpuDesc()
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

	if reservedMem := extHost.GetReservedMemoryMb(); reservedMem > 0 {
		host.MemReserved = reservedMem
	}

	host.ManagerId = provider.Id
	host.IsEmulated = extHost.IsEmulated()

	host.IsMaintenance = extHost.GetIsMaintenance()
	host.Version = extHost.GetVersion()

	host.IsPublic = false
	host.PublicScope = string(rbacutils.ScopeNone)

	var err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, manager, userCred, extHost.GetName())
		if err != nil {
			return errors.Wrapf(err, "db.GenerateName")
		}
		host.Name = newName

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

func (self *SHost) SyncHostStorages(ctx context.Context, userCred mcclient.TokenCredential, storages []cloudprovider.ICloudStorage, provider *SCloudprovider) ([]SStorage, []cloudprovider.ICloudStorage, compare.SyncResult) {
	lockman.LockRawObject(ctx, "storages", self.Id)
	defer lockman.ReleaseRawObject(ctx, "storages", self.Id)

	localStorages := make([]SStorage, 0)
	remoteStorages := make([]cloudprovider.ICloudStorage, 0)
	syncResult := compare.SyncResult{}

	dbStorages := make([]SStorage, 0)

	hostStorages := self.GetHoststorages()
	for i := 0; i < len(hostStorages); i += 1 {
		storage := hostStorages[i].GetStorage()
		if storage == nil {
			hostStorages[i].Delete(ctx, userCred)
		} else {
			dbStorages = append(dbStorages, *storage)
		}
	}

	// dbStorages := self._getAttachedStorages(tristate.None, tristate.None)

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
		log.Infof("host %s not connected with %s any more, to detach...", self.Id, removed[i].Id)
		err := self.syncRemoveCloudHostStorage(ctx, userCred, &removed[i])
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
		log.Infof("host %s is still connected with %s, to update ...", self.Id, commondb[i].Id)
		err := self.syncWithCloudHostStorage(ctx, userCred, &commondb[i], commonext[i], provider)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			localStorages = append(localStorages, commondb[i])
			remoteStorages = append(remoteStorages, commonext[i])
			syncResult.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		log.Infof("host %s is found connected with %s, to add ...", self.Id, added[i].GetId())
		local, err := self.newCloudHostStorage(ctx, userCred, added[i], provider)
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

func (self *SHost) syncRemoveCloudHostStorage(ctx context.Context, userCred mcclient.TokenCredential, localStorage *SStorage) error {
	hs := self.GetHoststorageOfId(localStorage.Id)
	err := hs.ValidateDeleteCondition(ctx)
	if err == nil {
		log.Errorf("sync remove hoststorage fail: %s", err)
		err = hs.Detach(ctx, userCred)
	} else {

	}
	return err
}

func (self *SHost) syncWithCloudHostStorage(ctx context.Context, userCred mcclient.TokenCredential, localStorage *SStorage, extStorage cloudprovider.ICloudStorage, provider *SCloudprovider) error {
	// do nothing
	hs := self.GetHoststorageOfId(localStorage.Id)
	err := hs.syncWithCloudHostStorage(userCred, extStorage)
	if err != nil {
		return err
	}
	s := hs.GetStorage()
	err = s.syncWithCloudStorage(ctx, userCred, extStorage, provider)
	return err
}

func (self *SHost) isAttach2Storage(storage *SStorage) bool {
	hs := self.GetHoststorageOfId(storage.Id)
	return hs != nil
}

func (self *SHost) Attach2Storage(ctx context.Context, userCred mcclient.TokenCredential, storage *SStorage, mountPoint string) error {
	if self.isAttach2Storage(storage) {
		return nil
	}

	hs := SHoststorage{}
	hs.SetModelManager(HoststorageManager, &hs)

	hs.StorageId = storage.Id
	hs.HostId = self.Id
	hs.MountPoint = mountPoint
	err := HoststorageManager.TableSpec().Insert(ctx, &hs)
	if err != nil {
		return err
	}

	db.OpsLog.LogAttachEvent(ctx, self, storage, userCred, nil)

	return nil
}

func (self *SHost) newCloudHostStorage(ctx context.Context, userCred mcclient.TokenCredential, extStorage cloudprovider.ICloudStorage, provider *SCloudprovider) (*SStorage, error) {
	storageObj, err := db.FetchByExternalIdAndManagerId(StorageManager, extStorage.GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("manager_id", provider.Id)
	})
	if err != nil {
		if err == sql.ErrNoRows {
			// no cloud storage found, this may happen for on-premise host
			// create the storage right now
			storageObj, err = StorageManager.newFromCloudStorage(ctx, userCred, extStorage, provider, self.GetZone())
			if err != nil {
				return nil, errors.Wrapf(err, "StorageManager.newFromCloudStorage")
			}
		} else {
			return nil, errors.Wrapf(err, "FetchByExternalIdAndManagerId(%s)", extStorage.GetGlobalId())
		}
	}
	storage := storageObj.(*SStorage)
	err = self.Attach2Storage(ctx, userCred, storage, extStorage.GetMountPoint())
	return storage, err
}

func (self *SHost) SyncHostWires(ctx context.Context, userCred mcclient.TokenCredential, wires []cloudprovider.ICloudWire) compare.SyncResult {
	lockman.LockRawObject(ctx, "wires", self.Id)
	defer lockman.ReleaseRawObject(ctx, "wires", self.Id)

	syncResult := compare.SyncResult{}

	dbWires := make([]SWire, 0)

	hostWires := self.GetHostwires()
	for i := 0; i < len(hostWires); i += 1 {
		wire := hostWires[i].GetWire()
		if wire == nil {
			hostWires[i].Delete(ctx, userCred)
		} else {
			dbWires = append(dbWires, *wire)
		}
	}

	// dbWires := self.getAttachedWires()

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
		log.Infof("host %s not connected with %s any more, to detach...", self.Id, removed[i].Id)
		err := self.syncRemoveCloudHostWire(ctx, userCred, &removed[i])
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		log.Infof("host %s is still connected with %s, to update...", self.Id, commondb[i].Id)
		err := self.syncWithCloudHostWire(commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		log.Infof("host %s is found connected with %s, to add...", self.Id, added[i].GetId())
		err := self.newCloudHostWire(ctx, userCred, added[i])
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}

	return syncResult
}

func (self *SHost) syncRemoveCloudHostWire(ctx context.Context, userCred mcclient.TokenCredential, localwire *SWire) error {
	hws := self.getHostwiresOfId(localwire.Id)
	for i := range hws {
		err := hws[i].Detach(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SHost) syncWithCloudHostWire(extWire cloudprovider.ICloudWire) error {
	// do nothing
	return nil
}

func (self *SHost) Attach2Wire(ctx context.Context, userCred mcclient.TokenCredential, wire *SWire) error {
	hs := SHostwire{}
	hs.SetModelManager(HostwireManager, &hs)

	hs.WireId = wire.Id
	hs.HostId = self.Id
	err := HostwireManager.TableSpec().Insert(ctx, &hs)
	if err != nil {
		return err
	}
	db.OpsLog.LogAttachEvent(ctx, self, wire, userCred, nil)
	return nil
}

func (self *SHost) newCloudHostWire(ctx context.Context, userCred mcclient.TokenCredential, extWire cloudprovider.ICloudWire) error {
	wireObj, err := db.FetchByExternalIdAndManagerId(WireManager, extWire.GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		sq := VpcManager.Query().SubQuery()
		return q.Join(sq, sqlchemy.Equals(sq.Field("id"), q.Field("vpc_id"))).Filter(sqlchemy.Equals(sq.Field("manager_id"), self.ManagerId))
	})
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	wire := wireObj.(*SWire)
	err = self.Attach2Wire(ctx, userCred, wire)
	return err
}

type SGuestSyncResult struct {
	Local  *SGuest
	Remote cloudprovider.ICloudVM
	IsNew  bool
}

func (self *SHost) SyncHostVMs(ctx context.Context, userCred mcclient.TokenCredential, iprovider cloudprovider.ICloudProvider, vms []cloudprovider.ICloudVM, syncOwnerId mcclient.IIdentityProvider) ([]SGuestSyncResult, compare.SyncResult) {
	lockman.LockRawObject(ctx, "guests", self.Id)
	defer lockman.ReleaseRawObject(ctx, "guests", self.Id)

	syncVMPairs := make([]SGuestSyncResult, 0)
	syncResult := compare.SyncResult{}

	dbVMs, err := self.GetGuests()
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

	err = compare.CompareSets(dbVMs, vms, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err := removed[i].syncRemoveCloudVM(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		err := commondb[i].syncWithCloudVM(ctx, userCred, iprovider, self, commonext[i], syncOwnerId)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
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
		vm, err := db.FetchByExternalIdAndManagerId(GuestManager, added[i].GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			sq := HostManager.Query().SubQuery()
			return q.Join(sq, sqlchemy.Equals(sq.Field("id"), q.Field("host_id"))).Filter(sqlchemy.Equals(sq.Field("manager_id"), self.ManagerId))
		})
		if err != nil && err != sql.ErrNoRows {
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
				return q.Equals("manager_id", self.ManagerId)
			})
			if err != nil {
				log.Errorf("failed to found host by externalId %s", ihost.GetGlobalId())
				continue
			}
			host := _host.(*SHost)
			err = guest.syncWithCloudVM(ctx, userCred, iprovider, host, added[i], syncOwnerId)
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
		new, err := GuestManager.newCloudVM(ctx, userCred, iprovider, self, added[i], syncOwnerId)
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

	return syncVMPairs, syncResult
}

func (self *SHost) getNetworkOfIPOnHost(ipAddr string) (*SNetwork, error) {
	netInterfaces := self.GetNetInterfaces()
	for _, netInterface := range netInterfaces {
		network, err := netInterface.GetCandidateNetworkForIp(nil, rbacutils.ScopeNone, ipAddr)
		if err == nil && network != nil {
			return network, nil
		}
	}

	return nil, fmt.Errorf("IP %s not reachable on this host", ipAddr)
}

func (self *SHost) GetNetinterfacesWithIdAndCredential(netId string, userCred mcclient.TokenCredential, reserved bool) ([]SNetInterface, *SNetwork) {
	netObj, err := NetworkManager.FetchById(netId)
	if err != nil {
		return nil, nil
	}
	net := netObj.(*SNetwork)
	used, err := net.getFreeAddressCount()
	if err != nil {
		return nil, nil
	}
	if used == 0 && !reserved {
		return nil, nil
	}
	matchNetIfs := make([]SNetInterface, 0)
	netifs := self.GetNetInterfaces()
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
		return matchNetIfs, net
	}
	return nil, nil
}

func (self *SHost) GetNetworkWithId(netId string, reserved bool) (*SNetwork, error) {
	var q1, q2 *sqlchemy.SQuery
	{
		networks := NetworkManager.Query()
		hostwires := HostwireManager.Query().SubQuery()
		hosts := HostManager.Query().SubQuery()
		q1 = networks
		q1 = q1.Join(hostwires, sqlchemy.Equals(hostwires.Field("wire_id"), networks.Field("wire_id")))
		q1 = q1.Join(hosts, sqlchemy.Equals(hosts.Field("id"), hostwires.Field("host_id")))
		q1 = q1.Filter(sqlchemy.Equals(networks.Field("id"), netId))
		q1 = q1.Filter(sqlchemy.Equals(hosts.Field("id"), self.Id))
	}
	{
		networks := NetworkManager.Query()
		wires := WireManager.Query().SubQuery()
		vpcs := VpcManager.Query().SubQuery()
		regions := CloudregionManager.Query().SubQuery()
		q2 = networks
		q2 = q2.Join(wires, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id")))
		q2 = q2.Join(vpcs, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
		q2 = q2.Join(regions, sqlchemy.Equals(regions.Field("id"), vpcs.Field("cloudregion_id")))
		q2 = q2.Filter(sqlchemy.Equals(networks.Field("id"), netId))
		q2 = q2.Filter(sqlchemy.AND(
			sqlchemy.Equals(regions.Field("provider"), api.CLOUD_PROVIDER_ONECLOUD),
			sqlchemy.NOT(sqlchemy.Equals(vpcs.Field("id"), api.DEFAULT_VPC_ID)),
		))
	}

	q := sqlchemy.Union(q1, q2).Query()

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

func (manager *SHostManager) totalCountQ(
	userCred mcclient.IIdentityProvider,
	scope rbacutils.TRbacScope,
	rangeObjs []db.IStandaloneModel,
	hostStatus, status string,
	hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	enabled, isBaremetal tristate.TriState,
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
	if scope != rbacutils.ScopeSystem && userCred != nil {
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
	scope rbacutils.TRbacScope,
	rangeObjs []db.IStandaloneModel,
	hostStatus, status string,
	hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	enabled, isBaremetal tristate.TriState,
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
		),
	)
}

/*
func (self *SHost) GetIZone() (cloudprovider.ICloudZone, error) {
	provider, err := self.GetCloudProvider()
	if err != nil {
		return nil, fmt.Errorf("No cloudprovider for host: %s", err)
	}
	zone := self.GetZone()
	if zone == nil {
		return nil, fmt.Errorf("no zone for host???")
	}
	region := zone.GetRegion()
	if region == nil {
		return nil, fmt.Errorf("No region for zone???")
	}
	iregion, err := provider.GetIRegionById(region.ExternalId)
	if err != nil {
		return nil, fmt.Errorf("fail to find iregion by id %s", err)
	}
	izone, err := iregion.GetIZoneById(zone.ExternalId)
	if err != nil {
		return nil, fmt.Errorf("fail to find izone by id %s", err)
	}
	return izone, nil
}
*/

func (self *SHost) GetIHost() (cloudprovider.ICloudHost, error) {
	host, _, err := self.GetIHostAndProvider()
	return host, err
}

func (self *SHost) GetIHostAndProvider() (cloudprovider.ICloudHost, cloudprovider.ICloudProvider, error) {
	provider, err := self.GetDriver()
	if err != nil {
		return nil, nil, fmt.Errorf("No cloudprovider for host: %s", err)
	}
	var iregion cloudprovider.ICloudRegion
	if provider.GetFactory().IsOnPremise() {
		iregion, err = provider.GetOnPremiseIRegion()
	} else {
		region := self.GetRegion()
		if region == nil {
			msg := "fail to find region of host???"
			log.Errorf(msg)
			return nil, nil, fmt.Errorf(msg)
		}
		iregion, err = provider.GetIRegionById(region.ExternalId)
	}
	if err != nil {
		log.Errorf("fail to find iregion: %s", err)
		return nil, nil, err
	}
	ihost, err := iregion.GetIHostById(self.ExternalId)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "iregion.GetIHostById(%s)", self.ExternalId)
	}
	return ihost, provider, nil
}

func (self *SHost) GetIRegion() (cloudprovider.ICloudRegion, error) {
	provider, err := self.GetDriver()
	if err != nil {
		return nil, fmt.Errorf("No cloudprovider for host %s: %s", self.Name, err)
	}
	region := self.GetRegion()
	if region == nil {
		return nil, fmt.Errorf("failed to find host %s region info", self.Name)
	}
	iregion, err := provider.GetIRegionById(region.ExternalId)
	if err != nil {
		msg := fmt.Sprintf("fail to find iregion by id %s: %v", region.ExternalId, err)
		return nil, fmt.Errorf(msg)
	}
	return iregion, nil
}

func (self *SHost) getDiskConfig() jsonutils.JSONObject {
	bs := self.GetBaremetalstorage()
	if bs != nil {
		return bs.Config
	}
	return nil
}

func (self *SHost) GetBaremetalServer() *SGuest {
	if !self.IsBaremetal {
		return nil
	}
	guest := SGuest{}
	guest.SetModelManager(GuestManager, &guest)
	q := GuestManager.Query().Equals("host_id", self.Id).Equals("hypervisor", api.HOST_TYPE_BAREMETAL)
	err := q.First(&guest)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("query fail %s", err)
		}
		return nil
	}
	return &guest
}

func (self *SHost) GetSchedtags() []SSchedtag {
	return GetSchedtags(HostschedtagManager, self.Id)
}

type SHostGuestResourceUsage struct {
	GuestCount     int
	GuestVcpuCount int
	GuestVmemSize  int
}

func (self *SHost) getGuestsResource(status string) *SHostGuestResourceUsage {
	guests := GuestManager.Query().SubQuery()
	q := guests.Query(sqlchemy.COUNT("guest_count"),
		sqlchemy.SUM("guest_vcpu_count", guests.Field("vcpu_count")),
		sqlchemy.SUM("guest_vmem_size", guests.Field("vmem_size")))
	cond := sqlchemy.OR(sqlchemy.Equals(q.Field("host_id"), self.Id),
		sqlchemy.Equals(q.Field("backup_host_id"), self.Id))
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

func (self *SHost) getMoreDetails(ctx context.Context, out api.HostDetails, showReason bool) api.HostDetails {
	server := self.GetBaremetalServer()
	if server != nil {
		out.ServerId = server.Id
		out.Server = server.Name
		out.ServerPendingDeleted = server.PendingDeleted
		if self.HostType == api.HOST_TYPE_BAREMETAL {
			out.ServerIps = strings.Join(server.GetRealIPs(), ",")
		}
	}
	netifs := self.GetNetInterfaces()
	if netifs != nil && len(netifs) > 0 {
		nicInfos := []jsonutils.JSONObject{}
		for i := 0; i < len(netifs); i += 1 {
			nicInfo := netifs[i].getBaremetalJsonDesc()
			if nicInfo == nil {
				log.Errorf("netif %s get baremetal desc failed", netifs[i].GetId())
				continue
			}
			nicInfos = append(nicInfos, nicInfo)
		}
		out.NicCount = len(nicInfos)
		out.NicInfo = nicInfos
	}
	out.Schedtags = GetSchedtagsDetailsToResourceV2(self, ctx)
	var usage *SHostGuestResourceUsage
	if options.Options.IgnoreNonrunningGuests {
		usage = self.getGuestsResource(api.VM_RUNNING)
	} else {
		usage = self.getGuestsResource("")
	}
	if usage != nil {
		out.CpuCommit = usage.GuestVcpuCount
		out.MemCommit = usage.GuestVmemSize
	}
	containerCount, _ := self.GetContainerCount(nil)
	runningContainerCount, _ := self.GetContainerCount(api.VM_RUNNING_STATUS)
	guestCount, _ := self.GetGuestCount()
	nonesysGuestCnt, _ := self.GetNonsystemGuestCount()
	runningGuestCnt, _ := self.GetRunningGuestCount()
	out.Guests = guestCount - containerCount
	out.NonsystemGuests = nonesysGuestCnt - containerCount
	out.RunningGuests = runningGuestCnt - runningContainerCount
	totalCpu := self.GetCpuCount()
	cpuCommitRate := 0.0
	if totalCpu > 0 && usage.GuestVcpuCount > 0 {
		cpuCommitRate = float64(usage.GuestVcpuCount) * 1.0 / float64(totalCpu)
	}
	out.CpuCommitRate = cpuCommitRate
	totalMem := self.GetMemSize()
	memCommitRate := 0.0
	if totalMem > 0 && usage.GuestVmemSize > 0 {
		memCommitRate = float64(usage.GuestVmemSize) * 1.0 / float64(totalMem)
	}
	out.MemCommitRate = memCommitRate
	capa := self.GetAttachedLocalStorageCapacity()
	out.Storage = capa.Capacity
	out.StorageUsed = capa.Used
	out.ActualStorageUsed = capa.ActualUsed
	out.StorageWaste = capa.Wasted
	out.StorageVirtual = capa.VCapacity
	out.StorageFree = capa.GetFree()
	out.StorageCommitRate = capa.GetCommitRate()
	out.Spec = self.GetHardwareSpecification()

	// custom cpu mem commit bound
	out.CpuCommitBound = self.GetCPUOvercommitBound()
	out.MemCommitBound = self.GetMemoryOvercommitBound()

	// extra = self.SManagedResourceBase.getExtraDetails(ctx, extra)

	out.IsPrepaidRecycle = false
	if self.IsPrepaidRecycle() {
		out.IsPrepaidRecycle = true
	}

	if self.IsBaremetal {
		out.CanPrepare = true
		err := self.canPrepare()
		if err != nil {
			out.CanPrepare = false
			if showReason {
				out.PrepareFailReason = err.Error()
			}
		}
	}

	if self.EnableHealthCheck && hostHealthChecker != nil {
		out.AllowHealthCheck = true
	}
	if self.GetMetadata("__auto_migrate_on_host_down", nil) == "enable" {
		out.AutoMigrateOnHostDown = true
	}

	if count, rs := self.GetReservedResourceForIsolatedDevice(); rs != nil {
		out.ReservedResourceForGpu = *rs
		out.IsolatedDeviceCount = count
	}
	return out
}

func (self *SHost) GetReservedResourceForIsolatedDevice() (int, *api.IsolatedDeviceReservedResourceInput) {
	if devs := IsolatedDeviceManager.FindByHost(self.Id); len(devs) == 0 {
		return -1, nil
	} else {
		return len(devs), self.GetDevsReservedResource(devs)
	}
}

func (self *SHost) GetDevsReservedResource(devs []SIsolatedDevice) *api.IsolatedDeviceReservedResourceInput {
	reservedCpu, reservedMem, reservedStorage := 0, 0, 0
	reservedResourceForGpu := api.IsolatedDeviceReservedResourceInput{
		ReservedStorage: &reservedStorage,
		ReservedMemory:  &reservedMem,
		ReservedCpu:     &reservedCpu,
	}
	for _, dev := range devs {
		reservedCpu += dev.ReservedCpu
		reservedMem += dev.ReservedMemory
		reservedStorage += dev.ReservedStorage
	}
	return &reservedResourceForGpu
}

func (self *SHost) GetMetadataHiddenKeys() []string {
	return []string{}
}

func (self *SHost) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.HostDetails, error) {
	return api.HostDetails{}, nil
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
	for i := range rows {
		rows[i] = api.HostDetails{
			EnabledStatusInfrasResourceBaseDetails: stdRows[i],
			ManagedResourceInfo:                    managerRows[i],
			ZoneResourceInfo:                       zoneRows[i],
		}
		rows[i] = objs[i].(*SHost).getMoreDetails(ctx, rows[i], showReason)
	}
	return rows
}

func (self *SHost) AllowGetDetailsVnc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGetSpec(userCred, self, "vnc")
}

func (self *SHost) GetDetailsVnc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(self.Status, []string{api.BAREMETAL_READY, api.BAREMETAL_RUNNING}) {
		retval := jsonutils.NewDict()
		retval.Set("host_id", jsonutils.NewString(self.Id))
		zone := self.GetZone()
		retval.Set("zone", jsonutils.NewString(zone.GetName()))
		return retval, nil
	}
	return jsonutils.NewDict(), nil
}

func (self *SHost) AllowGetDetailsIpmi(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGetSpec(userCred, self, "ipmi")
}

func (self *SHost) GetDetailsIpmi(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret, ok := self.IpmiInfo.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.NewNotFoundError("No ipmi information was found for host %s", self.Name)
	}
	password, err := ret.GetString("password")
	if err != nil {
		return nil, httperrors.NewNotFoundError("IPMI has no password information")
	}
	descryptedPassword, err := utils.DescryptAESBase64(self.Id, password)
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

func (self *SHost) Request(ctx context.Context, userCred mcclient.TokenCredential, method httputils.THttpMethod, url string, headers http.Header, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	s := auth.GetSession(ctx, userCred, "", "")
	_, ret, err := s.JSONRequest(self.ManagerUri, "", method, url, headers, body)
	return ret, err
}

func (self *SHost) GetLocalStoragecache() *SStoragecache {
	localStorages := self.GetAttachedLocalStorages()
	for i := 0; i < len(localStorages); i += 1 {
		sc := localStorages[i].GetStoragecache()
		if sc != nil {
			return sc
		}
	}
	return nil
}

func (self *SHost) GetStoragecache() *SStoragecache {
	localStorages := self.GetAttachedEnabledHostStorages(nil)
	for i := 0; i < len(localStorages); i += 1 {
		sc := localStorages[i].GetStoragecache()
		if sc != nil {
			return sc
		}
	}
	return nil
}

func (self *SHost) PostCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	self.SEnabledStatusInfrasResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	input := api.HostCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		log.Errorf("data.Unmarshal fail %s", err)
		return
	}
	kwargs := data.(*jsonutils.JSONDict)
	ipmiInfo, err := fetchIpmiInfo(input.HostIpmiAttributes, self.Id)
	if err != nil {
		log.Errorf("fetchIpmiInfo fail %s", err)
		return
	}
	ipmiInfoJson := jsonutils.Marshal(ipmiInfo).(*jsonutils.JSONDict)
	if ipmiInfoJson.Length() > 0 {
		_, err := self.SaveUpdates(func() error {
			self.IpmiInfo = ipmiInfoJson
			return nil
		})
		if err != nil {
			log.Errorf("save updates: %v", err)
		} else if len(ipmiInfo.IpAddr) > 0 {
			self.setIpmiIp(userCred, ipmiInfo.IpAddr)
		}
	}
	if len(input.AccessIp) > 0 {
		self.setAccessIp(userCred, input.AccessIp)
	}
	if len(input.AccessMac) > 0 {
		self.setAccessMac(userCred, input.AccessMac)
	}
	noProbe := false
	if input.NoProbe != nil {
		noProbe = *input.NoProbe
	}
	if len(self.ZoneId) > 0 && self.HostType == api.HOST_TYPE_BAREMETAL && !noProbe {
		// ipmiInfo, _ := self.GetIpmiInfo()
		if len(ipmiInfo.IpAddr) > 0 {
			self.StartBaremetalCreateTask(ctx, userCred, kwargs, "")
		}
	}

	keys := GetHostQuotaKeysFromCreateInput(ownerId, input)
	quota := SInfrasQuota{Host: 1}
	quota.SetKeys(keys)
	err = quotas.CancelPendingUsage(ctx, userCred, &quota, &quota, true)
	if err != nil {
		log.Errorf("CancelPendingUsage fail %s", err)
	}
}

func (self *SHost) StartBaremetalCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	if task, err := taskman.TaskManager.NewTask(ctx, "BaremetalCreateTask", self, userCred, data, parentTaskId, "", nil); err != nil {
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
		rip := ReservedipManager.GetReservedIP(net, ipmiIpAddr)
		if rip == nil {
			// if not, reserve this IP temporarily
			err := net.reserveIpWithDuration(ctx, userCred, ipmiIpAddr, "reserve for baremetal ipmi IP", 30*time.Minute)
			if err != nil {
				return input, errors.Wrap(err, "net.reserveIpWithDuration")
			}
		}
		zoneObj := net.GetZone()
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
	if !noProbe {
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
					net, err := wire.GetCandidatePrivateNetwork(userCred, NetworkManager.AllowScope(userCred), false, []string{api.NETWORK_TYPE_PXE, api.NETWORK_TYPE_BAREMETAL, api.NETWORK_TYPE_GUEST})
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

			accessIp, err := accessNet.GetFreeIP(ctx, userCred, nil, nil, accessIpAddr, api.IPAllocationNone, true)
			if err != nil {
				return input, httperrors.NewGeneralError(err)
			}

			if len(accessIpAddr) > 0 && accessIpAddr != accessIp {
				return input, httperrors.NewConflictError("Access ip %s has been used", accessIpAddr)
			}

			zoneObj := accessNet.GetZone()
			if zoneObj == nil {
				return input, httperrors.NewInputParameterError("Access network has no zone???")
			}
			originZoneId := input.ZoneId // data.GetString("zone_id")
			if len(originZoneId) > 0 && originZoneId != zoneObj.GetId() {
				return input, httperrors.NewInputParameterError("Access address located in different zone than specified")
			}

			// check ip has been reserved
			rip := ReservedipManager.GetReservedIP(accessNet, accessIp)
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

	keys := GetHostQuotaKeysFromCreateInput(ownerId, input)
	quota := SInfrasQuota{Host: 1}
	quota.SetKeys(keys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, &quota)
	if err != nil {
		return input, errors.Wrapf(err, "CheckSetPendingQuota")
	}

	return input, nil
}

func (self *SHost) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.HostUpdateInput) (api.HostUpdateInput, error) {
	var err error
	input.HostAccessAttributes, err = HostManager.inputUniquenessCheck(input.HostAccessAttributes, self.ZoneId, self.Id)
	if err != nil {
		return input, errors.Wrap(err, "inputUniquenessCheck")
	}

	input.HostSizeAttributes, err = HostManager.ValidateSizeParams(input.HostSizeAttributes)
	if err != nil {
		return input, errors.Wrap(err, "ValidateSizeParams")
	}

	ipmiInfo, err := fetchIpmiInfo(input.HostIpmiAttributes, self.Id)
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
			zoneObj := net.GetZone()
			if zoneObj == nil {
				return input, httperrors.NewInputParameterError("IPMI network has not zone???")
			}
			if zoneObj.GetId() != self.ZoneId {
				return input, httperrors.NewInputParameterError("New IPMI address located in another zone!")
			}
		}
		val := jsonutils.NewDict()
		val.Update(self.IpmiInfo)
		val.Update(ipmiInfoJson)
		input.IpmiInfo = val
	}
	input.EnabledStatusInfrasResourceBaseUpdateInput, err = self.SEnabledStatusInfrasResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusInfrasResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusInfrasResourceBase.ValidateUpdateData")
	}
	if len(input.Name) > 0 {
		self.UpdateDnsRecords(false)
	}
	return input, nil
}

func (self *SHost) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusInfrasResourceBase.PostUpdate(ctx, userCred, query, data)

	if data.Contains("cpu_cmtbound") || data.Contains("mem_cmtbound") {
		self.ClearSchedDescCache()
	}

	if self.OvnVersion != "" && self.OvnMappedIpAddr == "" {
		HostManager.lockAllocOvnMappedIpAddr(ctx)
		defer HostManager.unlockAllocOvnMappedIpAddr(ctx)
		addr, err := HostManager.allocOvnMappedIpAddr(ctx)
		if err != nil {
			log.Errorf("host %s(%s): alloc vpc mapped addr: %v",
				self.Name, self.Id, err)
			return
		}
		if _, err := db.Update(self, func() error {
			self.OvnMappedIpAddr = addr
			return nil
		}); err != nil {
			log.Errorf("host %s(%s): db update vpc mapped addr: %v",
				self.Name, self.Id, err)
			return
		}
	}

	// update baremetal host related server
	if guest := self.GetBaremetalServer(); guest != nil && self.HostType == api.HOST_TYPE_BAREMETAL {
		if _, err := db.Update(guest, func() error {
			guest.VmemSize = self.MemSize
			guest.VcpuCount = self.CpuCount
			return nil
		}); err != nil {
			log.Errorf("baremetal host %s update related server %s spec error: %v", self.GetName(), guest.GetName(), err)
		}
	}
}

func (self *SHost) UpdateDnsRecords(isAdd bool) {
	for _, netif := range self.GetNetInterfaces() {
		self.UpdateDnsRecord(&netif, isAdd)
	}
}

func (self *SHost) UpdateDnsRecord(netif *SNetInterface, isAdd bool) {
	name := self.GetNetifName(netif)
	if len(name) == 0 {
		return
	}
	bn := netif.GetBaremetalNetwork()
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

func (self *SHost) GetNetifName(netif *SNetInterface) string {
	if netif.NicType == api.NIC_TYPE_IPMI {
		return self.GetName()
	} else if netif.NicType == api.NIC_TYPE_ADMIN {
		return self.GetName() + "-admin"
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

func (self *SHost) AllowPerformStart(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "start")
}

func (self *SHost) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.IsBaremetal {
		return nil, httperrors.NewBadRequestError("Cannot start a non-baremetal host")
	}
	if !utils.IsInStringArray(self.Status, []string{api.BAREMETAL_READY}) {
		return nil, httperrors.NewInvalidStatusError("Cannot start baremetal with active guest")
	}
	guest := self.GetBaremetalServer()
	if guest != nil {
		if self.HostType == api.HOST_TYPE_BAREMETAL && utils.ToBool(guest.GetMetadata("is_fake_baremetal_server", userCred)) {
			return nil, self.InitializedGuestStart(ctx, userCred, guest)
		}
		//	if !utils.IsInStringArray(guest.Status, []string{VM_ADMIN}) {
		//		return nil, httperrors.NewBadRequestError("Cannot start baremetal with active guest")
		//	}
		self.SetStatus(userCred, api.BAREMETAL_START_MAINTAIN, "")
		return guest.PerformStart(ctx, userCred, query, data)
	}
	params := jsonutils.NewDict()
	params.Set("force_reboot", jsonutils.NewBool(false))
	params.Set("action", jsonutils.NewString("start"))
	return self.PerformMaintenance(ctx, userCred, nil, params)
}

func (self *SHost) AllowPerformStop(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "stop")
}

func (self *SHost) PerformStop(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.IsBaremetal {
		return nil, httperrors.NewBadRequestError("Cannot stop a non-baremetal host")
	}
	if !utils.IsInStringArray(self.Status, []string{api.BAREMETAL_RUNNING}) {
		return nil, httperrors.NewInvalidStatusError("Cannot stop baremetal with non-active guest")
	}
	guest := self.GetBaremetalServer()
	if guest != nil {
		if self.HostType != api.HOST_TYPE_BAREMETAL {
			if !utils.IsInStringArray(guest.Status, []string{api.VM_ADMIN}) {
				return nil, httperrors.NewBadRequestError("Cannot stop baremetal with active guest")
			}
		} else {
			if utils.ToBool(guest.GetMetadata("is_fake_baremetal_server", userCred)) {
				return nil, self.InitializedGuestStop(ctx, userCred, guest)
			}
			self.SetStatus(userCred, api.BAREMETAL_START_MAINTAIN, "")
			input := api.ServerStopInput{}
			data.Unmarshal(&input)
			return guest.PerformStop(ctx, userCred, query, input)
		}
	}
	return nil, self.StartBaremetalUnmaintenanceTask(ctx, userCred, false, "stop")
}

func (self *SHost) InitializedGuestStart(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest) error {
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalServerStartTask", guest, userCred, nil, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SHost) InitializedGuestStop(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest) error {
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalServerStopTask", guest, userCred, nil, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SHost) AllowPerformMaintenance(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "maintenance")

}

func (self *SHost) PerformMaintenance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.BAREMETAL_READY, api.BAREMETAL_RUNNING}) {
		return nil, httperrors.NewInvalidStatusError("Cannot do maintenance in status %s", self.Status)
	}
	guest := self.GetBaremetalServer()
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
	if self.Status == api.BAREMETAL_RUNNING && jsonutils.QueryBoolean(data, "force_reboot", false) {
		params.Set("force_reboot", jsonutils.NewBool(true))
	}
	action := "maintenance"
	if data.Contains("action") {
		action, _ = data.GetString("action")
	}
	params.Set("action", jsonutils.NewString(action))
	self.SetStatus(userCred, api.BAREMETAL_START_MAINTAIN, "")
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalMaintenanceTask", self, userCred, params, "", "", nil)
	if err != nil {
		return nil, err
	}
	task.ScheduleRun(nil)
	return nil, nil
}

func (self *SHost) AllowPerformUnmaintenance(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "unmaintenance")
}

func (self *SHost) PerformUnmaintenance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.BAREMETAL_RUNNING, api.BAREMETAL_READY}) {
		return nil, httperrors.NewInvalidStatusError("Cannot do unmaintenance in status %s", self.Status)
	}
	guest := self.GetBaremetalServer()
	if guest != nil && guest.Status != api.VM_ADMIN {
		return nil, httperrors.NewInvalidStatusError("Wrong guest status %s", guest.Status)
	}
	action, _ := data.GetString("action")
	if len(action) == 0 {
		action = "unmaintenance"
	}
	guestRunning := self.GetMetadata("__maint_guest_running", userCred)
	var startGuest = false
	if utils.ToBool(guestRunning) {
		startGuest = true
	}
	return nil, self.StartBaremetalUnmaintenanceTask(ctx, userCred, startGuest, action)
}

func (self *SHost) StartBaremetalUnmaintenanceTask(ctx context.Context, userCred mcclient.TokenCredential, startGuest bool, action string) error {
	self.SetStatus(userCred, api.BAREMETAL_START_MAINTAIN, "")
	params := jsonutils.NewDict()
	params.Set("guest_running", jsonutils.NewBool(startGuest))
	if len(action) == 0 {
		action = "unmaintenance"
	}
	params.Set("action", jsonutils.NewString(action))
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalUnmaintenanceTask", self, userCred, params, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SHost) StartSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	guest := self.GetBaremetalServer()
	if guest != nil {
		return guest.StartSyncstatus(ctx, userCred, parentTaskId)
	}
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalSyncStatusTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SHost) AllowPerformOffline(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "offline")
}

func (self *SHost) PerformOffline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.HostStatus != api.HOST_OFFLINE {
		_, err := self.SaveUpdates(func() error {
			self.HostStatus = api.HOST_OFFLINE
			if jsonutils.QueryBoolean(data, "update_health_status", false) {
				self.EnableHealthCheck = false
			}
			// Note: update host status to unknown on host offline
			// we did not have host status after host offline
			self.Status = api.BAREMETAL_UNKNOWN
			return nil
		})
		if err != nil {
			return nil, err
		}
		if hostHealthChecker != nil {
			hostHealthChecker.UnwatchHost(context.Background(), self.Id)
		}
		db.OpsLog.LogEvent(self, db.ACT_OFFLINE, "", userCred)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_OFFLINE, nil, userCred, true)
		self.SyncAttachedStorageStatus()
	}
	return nil, nil
}

func (self *SHost) AllowPerformOnline(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "online")
}

func (self *SHost) PerformOnline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.HostStatus != api.HOST_ONLINE {
		_, err := self.SaveUpdates(func() error {
			self.LastPingAt = time.Now()
			self.HostStatus = api.HOST_ONLINE
			self.EnableHealthCheck = true
			if !self.IsMaintaining() {
				self.Status = api.BAREMETAL_RUNNING
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if hostHealthChecker != nil {
			hostHealthChecker.WatchHost(context.Background(), self.Id)
		}
		db.OpsLog.LogEvent(self, db.ACT_ONLINE, "", userCred)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_ONLINE, nil, userCred, true)
		self.SyncAttachedStorageStatus()
		self.StartSyncAllGuestsStatusTask(ctx, userCred)
	}
	return nil, nil
}

func (self *SHost) AllowPerformAutoMigrateOnHostDown(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "auto-migrate-on-host-down")
}

func (self *SHost) PerformAutoMigrateOnHostDown(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	val, _ := data.GetString("auto_migrate_on_host_down")

	var meta map[string]interface{}
	if val == "enable" {
		meta = map[string]interface{}{
			"__auto_migrate_on_host_down": "enable",
			"__on_host_down":              "shutdown-servers",
		}
		_, err := self.Request(ctx, userCred, "POST", "/hosts/shutdown-servers-on-host-down",
			mcclient.GetTokenHeaders(userCred), nil)
		if err != nil {
			return nil, err
		}
	} else {
		meta = map[string]interface{}{
			"__auto_migrate_on_host_down": "disable",
			"__on_host_down":              "",
		}
	}

	return nil, self.SetAllMetadata(ctx, meta, userCred)
}

func (self *SHost) StartSyncAllGuestsStatusTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	if task, err := taskman.TaskManager.NewTask(ctx, "BaremetalSyncAllGuestsStatusTask", self, userCred, nil, "", "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
		return nil
	}
}

func (self *SHost) AllowPerformPing(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "ping")
}

func (self *SHost) PerformPing(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.HostStatus != api.HOST_ONLINE {
		self.PerformOnline(ctx, userCred, query, data)
	} else {
		self.SaveUpdates(func() error {
			self.LastPingAt = time.Now()
			return nil
		})
	}
	result := jsonutils.NewDict()
	result.Set("name", jsonutils.NewString(self.GetName()))
	dependSvcs := []string{"ntpd", "kafka", "influxdb", "elasticsearch"}
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

func (self *SHost) AllowPerformPrepare(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "prepare")
}

func (self *SHost) isRedfishCapable() bool {
	ipmiInfo, _ := self.GetIpmiInfo()
	if ipmiInfo.Verified && ipmiInfo.RedfishApi {
		return true
	}
	return false
}

func (self *SHost) canPrepare() error {
	if !self.IsBaremetal {
		return httperrors.NewInvalidStatusError("not a baremetal")
	}
	if !self.isRedfishCapable() && len(self.AccessMac) == 0 && len(self.Uuid) == 0 {
		return httperrors.NewInvalidStatusError("need valid access_mac and uuid to do prepare")
	}
	if !utils.IsInStringArray(self.Status, []string{api.BAREMETAL_READY, api.BAREMETAL_RUNNING, api.BAREMETAL_PREPARE_FAIL}) {
		return httperrors.NewInvalidStatusError("Cannot prepare baremetal in status %s", self.Status)
	}
	server := self.GetBaremetalServer()
	if server != nil && server.Status != api.VM_ADMIN {
		return httperrors.NewInvalidStatusError("Cannot prepare baremetal in server status %s", server.Status)
	}
	return nil
}

func (self *SHost) PerformPrepare(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := self.canPrepare()
	if err != nil {
		return nil, err
	}
	var onfinish string
	server := self.GetBaremetalServer()
	if server != nil && self.Status == api.BAREMETAL_READY {
		onfinish = "shutdown"
	}
	return nil, self.StartPrepareTask(ctx, userCred, onfinish, "")
}

func (self *SHost) StartPrepareTask(ctx context.Context, userCred mcclient.TokenCredential, onfinish, parentTaskId string) error {
	data := jsonutils.NewDict()
	if len(onfinish) > 0 {
		data.Set("on_finish", jsonutils.NewString(onfinish))
	}
	self.SetStatus(userCred, api.BAREMETAL_PREPARE, "start prepare task")
	if task, err := taskman.TaskManager.NewTask(ctx, "BaremetalPrepareTask", self, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
		return nil
	}
}

func (self *SHost) AllowPerformIpmiProbe(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "ipmi-probe")
}

func (self *SHost) PerformIpmiProbe(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(self.Status, []string{api.BAREMETAL_INIT, api.BAREMETAL_READY, api.BAREMETAL_RUNNING, api.BAREMETAL_PROBE_FAIL, api.BAREMETAL_UNKNOWN}) {
		return nil, self.StartIpmiProbeTask(ctx, userCred, "")
	}
	return nil, httperrors.NewInvalidStatusError("Cannot do Ipmi-probe in status %s", self.Status)
}

func (self *SHost) StartIpmiProbeTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	data := jsonutils.NewDict()
	self.SetStatus(userCred, api.BAREMETAL_START_PROBE, "start ipmi-probe task")
	if task, err := taskman.TaskManager.NewTask(ctx, "BaremetalIpmiProbeTask", self, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
		return nil
	}
}

func (self *SHost) AllowPerformInitialize(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject,
) bool {
	return db.IsAdminAllowPerform(userCred, self, "initialize")
}

func (self *SHost) PerformInitialize(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(
		self.Status, []string{api.BAREMETAL_INIT, api.BAREMETAL_PREPARE_FAIL}) {
		return nil, httperrors.NewBadRequestError(
			"Cannot do initialization in status %s", self.Status)
	}

	name, err := data.GetString("name")
	if err != nil || self.GetBaremetalServer() != nil {
		return nil, nil
	}
	err = db.NewNameValidator(GuestManager, userCred, name, nil)
	if err != nil {
		return nil, err
	}

	if self.IpmiInfo == nil || !self.IpmiInfo.Contains("ip_addr") ||
		!self.IpmiInfo.Contains("password") {
		return nil, httperrors.NewBadRequestError("IPMI infomation not configured")
	}
	guest := &SGuest{}
	guest.Name = name
	guest.VmemSize = self.MemSize
	guest.VcpuCount = self.CpuCount
	guest.DisableDelete = tristate.True
	guest.Hypervisor = api.HYPERVISOR_BAREMETAL
	guest.HostId = self.Id
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
		"is_fake_baremetal_server": true, "host_ip": self.AccessIp}, userCred)

	caps := self.GetAttachedLocalStorageCapacity()
	diskConfig := &api.DiskConfig{SizeMb: int(caps.GetFree())}
	err = guest.CreateDisksOnHost(ctx, userCred, self, []*api.DiskConfig{diskConfig}, nil, true, true, nil, nil, true)
	if err != nil {
		log.Errorf("Host perform initialize failed on create disk %s", err)
	}
	net, err := self.getNetworkOfIPOnHost(self.AccessIp)
	if err != nil {
		log.Errorf("host perfrom initialize failed fetch net of access ip %s", err)
	} else {
		if options.Options.BaremetalServerReuseHostIp {
			_, err = guest.attach2NetworkDesc(ctx, userCred, self, &api.NetworkConfig{Network: net.Id}, nil, nil)
			if err != nil {
				log.Errorf("host perform initialize failed on attach network %s", err)
			}
		}
	}
	return nil, nil
}

func (self *SHost) AllowPerformAddNetif(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "add-netif")
}

func (self *SHost) PerformAddNetif(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	log.Debugf("add_netif %s", data)
	mac, _ := data.GetString("mac")
	if len(mac) == 0 || len(netutils.FormatMacAddr(mac)) == 0 {
		return nil, httperrors.NewBadRequestError("Invaild mac address")
	}
	wire, _ := data.GetString("wire")
	ipAddr, _ := data.GetString("ip_addr")
	rate, _ := data.Int("rate")
	nicType, _ := data.GetString("nic_type")
	index, _ := data.Int("index")
	linkUp, _ := data.GetString("link_up")
	mtu, _ := data.Int("mtu")
	reset := jsonutils.QueryBoolean(data, "reset", false)
	strInterface, _ := data.GetString("interface")
	bridge, _ := data.GetString("bridge")
	reserve := jsonutils.QueryBoolean(data, "reserve", false)
	requireDesignatedIp := jsonutils.QueryBoolean(data, "require_designated_ip", false)

	isLinkUp := tristate.None
	if linkUp != "" {
		if utils.ToBool(linkUp) {
			isLinkUp = tristate.True
		} else {
			isLinkUp = tristate.False
		}
	}

	err := self.addNetif(ctx, userCred, mac, wire, ipAddr, int(rate), nicType, int8(index), isLinkUp,
		int16(mtu), reset, strInterface, bridge, reserve, requireDesignatedIp)
	return nil, err
}

func (self *SHost) addNetif(ctx context.Context, userCred mcclient.TokenCredential,
	mac string, wire string, ipAddr string,
	rate int, nicType string, index int8, linkUp tristate.TriState, mtu int16,
	reset bool, strInterface string, bridge string,
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
			swNets, err := sw.getNetworks(userCred, NetworkManager.AllowScope(userCred))
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
	netif, err := NetInterfaceManager.FetchByMac(mac)
	if err != nil {
		if err != sql.ErrNoRows {
			return httperrors.NewInternalServerError("fail to fetch netif by mac %s: %s", mac, err)
		}
		// else not found
		netif = &SNetInterface{}
		netif.Mac = mac
		netif.BaremetalId = self.Id
		if sw != nil {
			netif.WireId = sw.Id
		}
		netif.Rate = rate
		netif.NicType = nicType
		if index >= 0 {
			netif.Index = index
		}
		if !linkUp.IsNone() {
			netif.LinkUp = linkUp.Bool()
		}
		netif.Mtu = mtu
		err = NetInterfaceManager.TableSpec().Insert(ctx, netif)
		if err != nil {
			return err
		}
	} else {
		var changed = false
		_, err := db.Update(netif, func() error {
			if netif.BaremetalId != self.Id {
				changed = true
				netif.BaremetalId = self.Id
			}
			if sw != nil && netif.WireId != sw.Id {
				changed = true
				netif.WireId = sw.Id
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
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "db.Update")
		}
		if changed || reset {
			self.DisableNetif(ctx, userCred, netif, false)
		}
	}
	sw = netif.GetWire()
	if sw != nil {
		if len(strInterface) == 0 {
			strInterface = fmt.Sprintf("eth%d", netif.Index)
		}
		if len(strInterface) > 0 {
			if len(bridge) == 0 {
				bridge = fmt.Sprintf("br%s", sw.GetName())
			}
			var isMaster = netif.NicType == api.NIC_TYPE_ADMIN
			hw, err := HostwireManager.FetchByHostIdAndMac(self.Id, mac)
			if err != nil {
				if err != sql.ErrNoRows {
					return httperrors.NewInternalServerError("fail to fetch hostwire by mac %s: %s", mac, err)
				}
				hw = &SHostwire{}
				hw.Bridge = bridge
				hw.Interface = strInterface
				hw.HostId = self.Id
				hw.WireId = sw.Id
				hw.IsMaster = isMaster
				hw.MacAddr = mac
				err := HostwireManager.TableSpec().Insert(ctx, hw)
				if err != nil {
					return err
				}
			} else {
				db.Update(hw, func() error {
					hw.Bridge = bridge
					hw.Interface = strInterface
					// hw.MacAddr = mac
					hw.WireId = sw.Id
					hw.IsMaster = isMaster
					return nil
				})
			}
		}
	}
	if netif.NicType == api.NIC_TYPE_ADMIN {
		err := self.setAccessMac(userCred, netif.Mac)
		if err != nil {
			return httperrors.NewBadRequestError("%v", err)
		}
	}
	if len(ipAddr) > 0 {
		err = self.EnableNetif(ctx, userCred, netif, "", ipAddr, "", "", reserve, requireDesignatedIp)
		if err != nil {
			return httperrors.NewBadRequestError("%v", err)
		}
	}
	return nil
}

func (self *SHost) AllowPerformEnableNetif(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "enable-netif")
}

func (self *SHost) PerformEnableNetif(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	log.Debugf("enable_netif %s", data)
	mac, _ := data.GetString("mac")
	netif := self.GetNetInterface(mac)
	if netif == nil {
		return nil, httperrors.NewBadRequestError("Interface %s not exist", mac)
	}
	if !utils.IsInStringArray(netif.NicType, api.NIC_TYPES) {
		return nil, httperrors.NewBadRequestError("Only ADMIN and IPMI nic can be enable")
	}
	network, _ := data.GetString("network")
	ipAddr, _ := data.GetString("ip_addr")
	allocDir, _ := data.GetString("alloc_dir")
	netType, _ := data.GetString("net_type")
	reserve := jsonutils.QueryBoolean(data, "reserve", false)
	requireDesignatedIp := jsonutils.QueryBoolean(data, "require_designated_ip", false)
	err := self.EnableNetif(ctx, userCred, netif, network, ipAddr, allocDir, netType, reserve, requireDesignatedIp)
	if err != nil {
		return nil, httperrors.NewBadRequestError("%v", err)
	}
	return nil, nil
}

func (self *SHost) EnableNetif(ctx context.Context, userCred mcclient.TokenCredential, netif *SNetInterface, network, ipAddr, allocDir string, netType string, reserve, requireDesignatedIp bool) error {
	bn := netif.GetBaremetalNetwork()
	if bn != nil {
		log.Debugf("Netif has been attach2network? %s", jsonutils.Marshal(bn))
		return nil
	}
	var net *SNetwork
	var err error
	if len(ipAddr) > 0 {
		net, err = netif.GetCandidateNetworkForIp(userCred, NetworkManager.AllowScope(userCred), ipAddr)
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
	if self.ZoneId == "" {
		if _, err := self.SaveUpdates(func() error {
			self.ZoneId = wire.ZoneId
			return nil
		}); err != nil {
			return errors.Wrapf(err, "set host zone_id %s by wire", wire.ZoneId)
		}
	}
	hw, err := HostwireManager.FetchByHostIdAndMac(self.Id, netif.Mac)
	if err != nil {
		return err
	}
	if hw.WireId != wire.Id {
		return fmt.Errorf("host not attach to this wire")
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
			net, err = wire.GetCandidatePrivateNetwork(userCred, NetworkManager.AllowScope(userCred), false, netTypes)
			if err != nil {
				return fmt.Errorf("fail to find private network %s", err)
			}
			if net == nil {
				net, err = wire.GetCandidateAutoAllocNetwork(userCred, NetworkManager.AllowScope(userCred), false, netTypes)
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

	bn, err = self.Attach2Network(ctx, userCred, attachOpt)
	if err != nil {
		return errors.Wrap(err, "self.Attach2Network")
	}
	switch netif.NicType {
	case api.NIC_TYPE_IPMI:
		err = self.setIpmiIp(userCred, bn.IpAddr)
	case api.NIC_TYPE_ADMIN:
		err = self.setAccessIp(userCred, bn.IpAddr)
	}
	return err
}

func (self *SHost) AllowPerformDisableNetif(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "disable-netif")
}

func (self *SHost) PerformDisableNetif(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	mac, _ := data.GetString("mac")
	netif := self.GetNetInterface(mac)
	if netif == nil {
		return nil, httperrors.NewBadRequestError("Interface %s not exists", mac)
	}
	reserve := jsonutils.QueryBoolean(data, "reserve", false)
	err := self.DisableNetif(ctx, userCred, netif, reserve)
	if err != nil {
		return nil, httperrors.NewBadRequestError("%v", err)
	}
	return nil, nil
}

func (self *SHost) DisableNetif(ctx context.Context, userCred mcclient.TokenCredential, netif *SNetInterface, reserve bool) error {
	bn := netif.GetBaremetalNetwork()
	var ipAddr string
	if bn != nil {
		ipAddr = bn.IpAddr
		self.UpdateDnsRecord(netif, false)
		self.DeleteBaremetalnetwork(ctx, userCred, bn, reserve)
	}
	var err error
	switch netif.NicType {
	case api.NIC_TYPE_IPMI:
		if ipAddr == self.IpmiIp {
			err = self.setIpmiIp(userCred, "")
		}
	case api.NIC_TYPE_ADMIN:
		if ipAddr == self.AccessIp {
			err = self.setAccessIp(userCred, "")
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

func (self *SHost) IsIpAddrWithinConvertedGuest(ctx context.Context, userCred mcclient.TokenCredential, ipAddr string, netif *SNetInterface) error {
	if !self.IsBaremetal {
		return httperrors.NewNotAcceptableError("Not a baremetal")
	}

	if self.HostType == api.HOST_TYPE_KVM {
		return httperrors.NewNotAcceptableError("Not being convert to hypervisor")
	}

	bmServer := self.GetBaremetalServer()
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

func (self *SHost) Attach2Network(
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
		if err := self.IsIpAddrWithinConvertedGuest(ctx, userCred, ipAddr, netif); err == nil {
			// force remove used server addr for reuse
			delete(usedAddrs, ipAddr)
		} else {
			log.Warningf("check IsIpAddrWithinConvertedGuest: %v", err)
		}
	}

	freeIp, err := net.GetFreeIP(ctx, userCred, usedAddrs, nil, ipAddr, api.IPAllocationDirection(allocDir), reserved)
	if err != nil {
		return nil, errors.Wrap(err, "net.GetFreeIP")
	}
	if len(ipAddr) > 0 && ipAddr != freeIp && requireDesignatedIp {
		return nil, fmt.Errorf("IP address %s is occupied, get %s instead", ipAddr, freeIp)
	}
	bn := &SHostnetwork{}
	bn.BaremetalId = self.Id
	bn.NetworkId = net.Id
	bn.IpAddr = freeIp
	bn.MacAddr = netif.Mac
	err = HostnetworkManager.TableSpec().Insert(ctx, bn)
	if err != nil {
		return nil, errors.Wrap(err, "HostnetworkManager.TableSpec().Insert")
	}
	db.OpsLog.LogAttachEvent(ctx, self, net, userCred, jsonutils.NewString(freeIp))
	self.UpdateDnsRecord(netif, true)
	net.UpdateBaremetalNetmap(bn, self.GetNetifName(netif))
	return bn, nil
}

func (self *SHost) AllowPerformRemoveNetif(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "remove-netif")
}

func (self *SHost) PerformRemoveNetif(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	mac, _ := data.GetString("mac")
	mac = netutils.FormatMacAddr(mac)
	if len(mac) == 0 {
		return nil, httperrors.NewBadRequestError("Invalid mac address")
	}
	netif, err := NetInterfaceManager.FetchByMac(mac)
	if err != nil {
		return nil, httperrors.NewBadRequestError("Fetch netif error %s", err)
	}
	return nil, self.RemoveNetif(ctx, userCred, netif, jsonutils.QueryBoolean(data, "reserve", false))
}

func (self *SHost) RemoveNetif(ctx context.Context, userCred mcclient.TokenCredential, netif *SNetInterface, reserve bool) error {
	wire := netif.GetWire()
	self.DisableNetif(ctx, userCred, netif, reserve)
	nicType := netif.NicType
	mac := netif.Mac
	err := netif.Remove(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "netif.Remove")
	}
	if nicType == api.NIC_TYPE_ADMIN && self.AccessMac == mac {
		err := self.setAccessMac(userCred, "")
		if err != nil {
			return errors.Wrap(err, "self.setAccessMac")
		}
	}
	if wire != nil {
		others := self.GetNetifsOnWire(wire)
		if len(others) == 0 {
			hw, _ := HostwireManager.FetchByHostIdAndMac(self.Id, netif.Mac)
			if hw != nil {
				db.OpsLog.LogDetachEvent(ctx, self, wire, userCred, jsonutils.NewString(fmt.Sprintf("disable netif %s", self.AccessMac)))
				log.Debugf("Detach host wire because of remove netif %s", netif.Mac)
				return hw.Delete(ctx, userCred)
			}
		}
	}
	return nil
}

func (self *SHost) GetNetifsOnWire(wire *SWire) []SNetInterface {
	dest := make([]SNetInterface, 0)
	q := NetInterfaceManager.Query()
	err := q.Filter(sqlchemy.Equals(q.Field("baremetal_id"), self.Id)).Filter(sqlchemy.Equals(q.Field("wire_id"), wire.Id)).Desc(q.Field("index")).All(&dest)
	if err != nil {
		log.Errorln(err)
		return nil
	}
	return dest
}

func (self *SHost) AllowPerformSyncstatus(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "syncstatus")
}

func (self *SHost) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.HostType != api.HOST_TYPE_BAREMETAL {
		return nil, httperrors.NewBadRequestError("Cannot sync status a non-baremetal host")
	}
	self.SetStatus(userCred, api.BAREMETAL_SYNCING_STATUS, "")
	return nil, self.StartSyncstatus(ctx, userCred, "")
}

func (self *SHost) AllowPerformReset(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "reset")
}

func (self *SHost) PerformReset(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.IsBaremetal {
		return nil, httperrors.NewBadRequestError("Cannot start a non-baremetal host")
	}
	if self.Status != api.BAREMETAL_RUNNING {
		return nil, httperrors.NewBadRequestError("Cannot reset baremetal in status %s", self.Status)
	}
	guest := self.GetBaremetalServer()
	if guest != nil {
		if self.HostType == api.HOST_TYPE_BAREMETAL {
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
	return self.PerformMaintenance(ctx, userCred, query, kwargs)
}

func (self *SHost) AllowPerformRemoveAllNetifs(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "remove-all-netifs")
}

func (self *SHost) PerformRemoveAllNetifs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	netifs := self.GetNetInterfaces()
	for i := 0; i < len(netifs); i++ {
		if !utils.IsInStringArray(netifs[i].NicType, api.NIC_TYPES) {
			self.RemoveNetif(ctx, userCred, &netifs[i], false)
		}
	}
	return nil, nil
}

func (self *SHost) AllowPerformEnable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.PerformEnableInput,
) bool {
	return self.SEnabledStatusInfrasResourceBase.AllowPerformEnable(ctx, userCred, query, input)
}

func (self *SHost) PerformEnable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.PerformEnableInput,
) (jsonutils.JSONObject, error) {
	if !self.GetEnabled() {
		_, err := self.SEnabledStatusInfrasResourceBase.PerformEnable(ctx, userCred, query, input)
		if err != nil {
			return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBase.PerformEnable")
		}
		self.SyncAttachedStorageStatus()
	}
	return nil, nil
}

func (self *SHost) AllowPerformDisable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.PerformDisableInput,
) bool {
	return self.SEnabledStatusInfrasResourceBase.AllowPerformDisable(ctx, userCred, query, input)
}

func (self *SHost) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	if self.GetEnabled() {
		_, err := self.SEnabledStatusInfrasResourceBase.PerformDisable(ctx, userCred, query, input)
		if err != nil {
			return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBase.PerformDisable")
		}
		self.SyncAttachedStorageStatus()
	}
	return nil, nil
}

func (self *SHost) AllowPerformCacheImage(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "cache-image")
}

func (self *SHost) PerformCacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.HostType == api.HOST_TYPE_BAREMETAL || self.HostStatus != api.HOST_ONLINE {
		return nil, httperrors.NewInvalidStatusError("Cannot perform cache image in status %s", self.Status)
	}
	imageId, _ := data.GetString("image")
	img, err := CachedimageManager.getImageInfo(ctx, userCred, imageId, false)
	if err != nil {
		log.Errorln(err)
		return nil, httperrors.NewNotFoundError("image %s not found", imageId)
	}
	if len(img.Checksum) != 0 && regutils.MatchUUID(img.Checksum) {
		return nil, httperrors.NewInvalidStatusError("Cannot cache image with no checksum")
	}
	isForce := jsonutils.QueryBoolean(data, "is_force", false)
	format, _ := data.GetString("format")
	return nil, self.StartImageCacheTask(ctx, userCred, img.Id, format, isForce)
}

func (self *SHost) StartImageCacheTask(ctx context.Context, userCred mcclient.TokenCredential, imageId string, format string, isForce bool) error {
	var sc *SStoragecache
	switch self.HostType {
	case api.HOST_TYPE_BAREMETAL:
	case api.HOST_TYPE_HYPERVISOR, api.HOST_TYPE_ESXI:
		sc = self.GetLocalStoragecache()
	default:
		sc = self.GetStoragecache()
	}
	if sc == nil {
		return errors.Wrap(errors.ErrNotSupported, "No associate storage cache found")
	}
	return sc.StartImageCacheTask(ctx, userCred, imageId, format, isForce, "")
}

func (self *SHost) AllowPerformConvertHypervisor(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "convert-hypervisor")
}

func (self *SHost) isAlterNameUnique(name string) (bool, error) {
	q := HostManager.Query().Equals("name", name).NotEquals("id", self.Id).Equals("zone_id", self.ZoneId)
	cnt, err := q.CountWithError()
	if err != nil {
		return false, err
	}
	return cnt == 0, nil
}

func (self *SHost) PerformConvertHypervisor(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	hostType, err := data.GetString("host_type")
	if err != nil {
		return nil, httperrors.NewNotAcceptableError("host_type must be specified")
	}
	if self.HostType != api.HOST_TYPE_BAREMETAL {
		return nil, httperrors.NewNotAcceptableError("Must be a baremetal host")
	}
	if self.GetBaremetalServer() != nil {
		return nil, httperrors.NewNotAcceptableError("Baremetal host is aleady occupied")
	}
	if !utils.IsInStringArray(self.Status, []string{api.BAREMETAL_READY, api.BAREMETAL_RUNNING}) {
		return nil, httperrors.NewNotAcceptableError("Connot convert hypervisor in status %s", self.Status)
	}
	// check ownership
	var ownerId mcclient.IIdentityProvider
	hostOwnerId := self.GetOwnerId()
	if userCred.GetProjectDomainId() != hostOwnerId.GetProjectDomainId() {
		if !db.IsAdminAllowPerform(userCred, self, "convert-hypervisor") {
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
		err := self.GetModelManager().ValidateName(name)
		if err != nil {
			return nil, err
		}
		uniq, err := self.isAlterNameUnique(name)
		if err != nil {
			return nil, httperrors.NewInternalServerError("isAlterNameUnique fail %s", err)
		}
		if !uniq {
			return nil, httperrors.NewDuplicateNameError(name, self.Id)
		}
	}
	image, _ := data.GetString("image")
	raid, _ := data.GetString("raid")
	input, err := driver.PrepareConvert(self, image, raid, data)
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
	db.OpsLog.LogEvent(self, db.ACT_CONVERT_START, "", userCred)
	db.OpsLog.LogEvent(guest, db.ACT_CREATE, "Convert hypervisor", userCred)

	opts := jsonutils.NewDict()
	opts.Set("server_params", params)
	opts.Set("server_id", jsonutils.NewString(guest.GetId()))
	opts.Set("convert_host_type", jsonutils.NewString(hostType))

	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalConvertHypervisorTask", self, adminCred, opts, "", "", nil)
	if err != nil {
		return nil, err
	}
	task.ScheduleRun(nil)

	self.SetStatus(userCred, api.BAREMETAL_START_CONVERT, "")
	return nil, nil
}

func (self *SHost) AllowPerformUndoConvert(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "undo-convert")
}

func (self *SHost) PerformUndoConvert(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.IsBaremetal {
		return nil, httperrors.NewNotAcceptableError("Not a baremetal")
	}
	if self.HostType == api.HOST_TYPE_BAREMETAL {
		return nil, httperrors.NewNotAcceptableError("Not being convert to hypervisor")
	}
	if self.GetEnabled() {
		return nil, httperrors.NewNotAcceptableError("Host should be disabled")
	}
	if !utils.IsInStringArray(self.Status, []string{api.BAREMETAL_READY, api.BAREMETAL_RUNNING}) {
		return nil, httperrors.NewNotAcceptableError("Cannot unconvert in status %s", self.Status)
	}
	driver := self.GetDriverWithDefault()
	if driver == nil {
		return nil, httperrors.NewNotAcceptableError("Unsupport driver type %s", self.HostType)
	}
	err := driver.PrepareUnconvert(self)
	if err != nil {
		return nil, httperrors.NewNotAcceptableError("%v", err)
	}
	guests, err := self.GetGuests()
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
	db.OpsLog.LogEvent(self, db.ACT_UNCONVERT_START, "", userCred)
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalUnconvertHypervisorTask", self, userCred, nil, "", "", nil)
	if err != nil {
		return nil, err
	}
	task.ScheduleRun(nil)
	return nil, nil
}

func (self *SHost) GetDriverWithDefault() IHostDriver {
	hostType := self.HostType
	if len(hostType) == 0 {
		hostType = api.HOST_TYPE_DEFAULT
	}
	return GetHostDriver(hostType)
}

func (self *SHost) UpdateDiskConfig(userCred mcclient.TokenCredential, layouts []baremetal.Layout) error {
	bs := self.GetBaremetalstorage()
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
func (host *SHost) SyncEsxiHostWires(ctx context.Context, userCred mcclient.TokenCredential, remoteHost cloudprovider.ICloudHost) compare.SyncResult {
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
	netIfs := host.GetNetInterfaces()
	hostwires, err := host.getHostwires()
	if err != nil {
		result.Error(errors.Wrapf(err, "unable to getHostwires of host %s", host.GetId()))
		return result
	}

	for i := range vsWires {
		vsWire := vsWires[i]
		if vsWire.SyncTimes > 0 {
			continue
		}
		netif := host.findNetIfs(netIfs, vsWire.Mac)
		if netif == nil {
			// do nothing
			continue
		}
		hostwire := host.findHostwire(hostwires, vsWire.WireId, vsWire.Mac)
		if hostwire == nil {
			hostwire = &SHostwire{
				Bridge:  vsWire.VsId,
				MacAddr: vsWire.Mac,
				HostId:  host.GetId(),
				WireId:  vsWire.WireId,
			}
			hostwire.MacAddr = vsWire.Mac
			err := HostwireManager.TableSpec().Insert(ctx, hostwire)
			if err != nil {
				result.Error(errors.Wrapf(err, "unable to create hostwire for host %q", host.GetId()))
				continue
			}
		}
		if hostwire.Bridge != vsWire.VsId {
			db.Update(hostwire, func() error {
				hostwire.Bridge = vsWire.VsId
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
}

func (host *SHost) findHostwire(hostwires []SHostwire, wireId string, mac string) *SHostwire {
	for i := range hostwires {
		if hostwires[i].WireId == wireId && hostwires[i].MacAddr == mac {
			return &hostwires[i]
		}
	}
	return nil
}

func (host *SHost) findNetIfs(netIfs []SNetInterface, mac string) *SNetInterface {
	for i := range netIfs {
		if netIfs[i].Mac == mac {
			return &netIfs[i]
		}
	}
	return nil
}

func (host *SHost) SyncHostExternalNics(ctx context.Context, userCred mcclient.TokenCredential, ihost cloudprovider.ICloudHost) compare.SyncResult {
	result := compare.SyncResult{}

	netIfs := host.GetNetInterfaces()
	extNics, err := ihost.GetIHostNics()
	if err != nil {
		result.Error(err)
		return result
	}

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

	nicMax := len(netIfs)
	if nicMax < len(extNics) {
		nicMax = len(extNics)
	}
	for i := 0; i < nicMax; i += 1 {
		if i < len(netIfs) && i < len(extNics) {
			obn := netIfs[i].GetBaremetalNetwork()
			var oip string
			if obn != nil {
				oip = obn.IpAddr
			}
			nip := extNics[i].GetIpAddr()
			if netIfs[i].Mac == extNics[i].GetMac() {
				if oip != nip {
					if obn != nil {
						disables = append(disables, &netIfs[i])
					}
					if len(nip) > 0 {
						enables = append(enables, extNics[i])
					}
				} else {
					// in sync, sync interface and bridge
					hw := host.getHostwireOfIdAndMac(netIfs[i].WireId, netIfs[i].Mac)
					if hw != nil && (hw.Bridge != extNics[i].GetBridge() || hw.Interface != extNics[i].GetDevice()) {
						db.Update(hw, func() error {
							hw.Interface = extNics[i].GetDevice()
							// hw.Bridge = extNics[i].GetBridge()
							return nil
						})
					}
				}
			} else {
				reserveIp := false
				if len(oip) > 0 && oip == nip {
					// # mac change case
					reserveIp = true
				}
				removes = append(removes, sRemoveNetInterface{netif: &netIfs[i], reserveIp: reserveIp})
				adds = append(adds, sAddNetInterface{netif: extNics[i], reserveIp: reserveIp})
			}
		} else if i < len(netIfs) && i >= len(extNics) {
			removes = append(removes, sRemoveNetInterface{netif: &netIfs[i], reserveIp: false})
		} else if i >= len(netIfs) && i < len(extNics) {
			adds = append(adds, sAddNetInterface{netif: extNics[i], reserveIp: false})
		}
	}

	for i := len(removes) - 1; i >= 0; i -= 1 {
		err = host.RemoveNetif(ctx, userCred, removes[i].netif, removes[i].reserveIp)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}

	for i := len(disables) - 1; i >= 0; i -= 1 {
		err = host.DisableNetif(ctx, userCred, disables[i], false)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}

	for i := 0; i < len(enables); i += 1 {
		netif := host.GetNetInterface(enables[i].GetMac())
		// always true reserved address pool
		err = host.EnableNetif(ctx, userCred, netif, "", enables[i].GetIpAddr(), "", "", true, true)
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
		}
	}

	for i := 0; i < len(adds); i += 1 {
		// always try reserved pool
		extNic := adds[i].netif
		err = host.addNetif(ctx, userCred, extNic.GetMac(), "", extNic.GetIpAddr(), 0, extNic.GetNicType(), extNic.GetIndex(),
			extNic.IsLinkUp(), int16(extNic.GetMtu()), false, extNic.GetDevice(), extNic.GetBridge(), true, true)
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

func (self *SHost) IsBaremetalAgentReady() bool {
	return self.isAgentReady(api.AgentTypeBaremetal)
}

func (self *SHost) BaremetalSyncRequest(ctx context.Context, method httputils.THttpMethod, url string, headers http.Header, body *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	return self.doAgentRequest(api.AgentTypeBaremetal, ctx, method, url, headers, body)
}

func (self *SHost) IsEsxiAgentReady() bool {
	return self.isAgentReady(api.AgentTypeEsxi)
}

func (self *SHost) EsxiRequest(ctx context.Context, method httputils.THttpMethod, url string, headers http.Header, body *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	return self.doAgentRequest(api.AgentTypeEsxi, ctx, method, url, headers, body)
}

func (self *SHost) isAgentReady(agentType api.TAgentType) bool {
	agent := BaremetalagentManager.GetAgent(agentType, self.ZoneId)
	if agent == nil {
		log.Errorf("%s ready: false", agentType)
		return false
	}
	return true
}

func (self *SHost) doAgentRequest(agentType api.TAgentType, ctx context.Context, method httputils.THttpMethod, url string, headers http.Header, body *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	agent := BaremetalagentManager.GetAgent(agentType, self.ZoneId)
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

func (manager *SHostManager) GetHostByIp(hostIp string) (*SHost, error) {
	q := manager.Query()
	q = q.Equals("access_ip", hostIp)

	host, err := db.NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	err = q.First(host)
	if err != nil {
		return nil, err
	}

	return host.(*SHost), nil
}

func (self *SHost) getCloudProviderInfo() SCloudProviderInfo {
	var region *SCloudregion
	zone := self.GetZone()
	if zone != nil {
		region = zone.GetRegion()
	}
	provider := self.GetCloudprovider()
	return MakeCloudProviderInfo(region, zone, provider)
}

func (self *SHost) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := self.SEnabledStatusInfrasResourceBase.GetShortDesc(ctx)
	info := self.getCloudProviderInfo()
	desc.Update(jsonutils.Marshal(&info))
	return desc
}

func (self *SHost) MarkGuestUnknown(userCred mcclient.TokenCredential) {
	guests, _ := self.GetGuests()
	for _, guest := range guests {
		guest.SetStatus(userCred, api.VM_UNKNOWN, "host offline")
	}
}

func (manager *SHostManager) PingDetectionTask(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	deadline := time.Now().Add(-1 * time.Duration(options.Options.HostOfflineMaxSeconds) * time.Second)

	q := manager.Query().Equals("host_status", api.HOST_ONLINE).
		Equals("host_type", api.HOST_TYPE_HYPERVISOR)
	q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("last_ping_at")),
		sqlchemy.LT(q.Field("last_ping_at"), deadline)))

	rows, err := q.Rows()
	if err != nil {
		log.Errorln(err)
		return
	}
	defer rows.Close()

	data := jsonutils.NewDict()
	data.Set("update_health_status", jsonutils.JSONFalse)
	for rows.Next() {
		var host = new(SHost)
		q.Row2Struct(rows, host)
		host.SetModelManager(manager, host)
		func() {
			lockman.LockObject(ctx, host)
			defer lockman.ReleaseObject(ctx, host)
			host.PerformOffline(ctx, userCred, nil, data)
			host.MarkGuestUnknown(userCred)
		}()
	}
}

func (self *SHost) IsPrepaidRecycleResource() bool {
	return self.ResourceType == api.HostResourceTypePrepaidRecycle
}

func (host *SHost) AllowPerformSetSchedtag(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return AllowPerformSetResourceSchedtag(host, ctx, userCred, query, data)
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

func (host *SHost) AllowPerformHostExitMaintenance(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, host, "host maintenance")
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

func (host *SHost) AllowPerformHostMaintenance(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, host, "host maintenance")
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
		err := host.IsAssignable(userCred)
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

func (host *SHost) OnHostDown(ctx context.Context, userCred mcclient.TokenCredential) {
	log.Errorf("watched host down %s", host.Id)
	db.OpsLog.LogEvent(host, db.ACT_HOST_DOWN, "", userCred)
	if _, err := host.SaveCleanUpdates(func() error {
		host.EnableHealthCheck = false
		host.HostStatus = api.HOST_OFFLINE
		return nil
	}); err != nil {
		log.Errorf("update host %s failed %s", host.Id, err)
	}
	host.SyncCleanSchedDescCache()
	host.switchWithBackup(ctx, userCred)
	host.migrateOnHostDown(ctx, userCred)
}

func (host *SHost) switchWithBackup(ctx context.Context, userCred mcclient.TokenCredential) {
	guests := host.GetGuestsMasterOnThisHost()
	for i := 0; i < len(guests); i++ {
		if guests[i].isInReconcile(userCred) {
			log.Warningf("guest %s is in reconcile", guests[i].GetName())
			continue
		}
		data := jsonutils.NewDict()
		data.Set("purge_backup", jsonutils.JSONTrue)
		_, err := guests[i].PerformSwitchToBackup(ctx, userCred, nil, data)
		if err != nil {
			db.OpsLog.LogEvent(
				&guests[i], db.ACT_SWITCH_FAILED, fmt.Sprintf("PerformSwitchToBackup on host down: %s", err), userCred,
			)
			logclient.AddSimpleActionLog(
				&guests[i], logclient.ACT_SWITCH_TO_BACKUP,
				fmt.Sprintf("PerformSwitchToBackup on host down: %s", err), userCred, false,
			)
		} else {
			guests[i].SetMetadata(ctx, "origin_status", guests[i].Status, userCred)
		}
	}

	guests2 := host.GetGuestsBackupOnThisHost()
	for i := 0; i < len(guests2); i++ {
		if guests2[i].isInReconcile(userCred) {
			log.Warningf("guest %s is in reconcile", guests2[i].GetName())
			continue
		}
		data := jsonutils.NewDict()
		data.Set("purge", jsonutils.JSONTrue)
		data.Set("create", jsonutils.JSONTrue)
		_, err := guests2[i].PerformDeleteBackup(ctx, userCred, nil, data)
		if err != nil {
			db.OpsLog.LogEvent(
				&guests2[i], db.ACT_DELETE_BACKUP_FAILED, fmt.Sprintf("PerformDeleteBackup on host down: %s", err), userCred,
			)
			logclient.AddSimpleActionLog(
				&guests2[i], logclient.ACT_DELETE_BACKUP,
				fmt.Sprintf("PerformDeleteBackup on host down: %s", err), userCred, false,
			)
		}
	}
}

func (host *SHost) migrateOnHostDown(ctx context.Context, userCred mcclient.TokenCredential) {
	if host.GetMetadata("__auto_migrate_on_host_down", nil) == "enable" {
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
	hostGuests := []*api.GuestBatchMigrateParams{}

	for i := 0; i < len(guests); i++ {
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
		}
	}
	kwargs := jsonutils.NewDict()
	kwargs.Set("guests", jsonutils.Marshal(hostGuests))
	return GuestManager.StartHostGuestsMigrateTask(ctx, userCred, host, kwargs, "")
}

func (host *SHost) SetStatus(userCred mcclient.TokenCredential, status string, reason string) error {
	err := host.SEnabledStatusInfrasResourceBase.SetStatus(userCred, status, reason)
	if err != nil {
		return err
	}
	host.ClearSchedDescCache()
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

func (self *SHost) AllowGetDetailsJnlp(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGetSpec(userCred, self, "jnlp")
}

func (self *SHost) GetDetailsJnlp(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("/baremetals/%s/jnlp", self.Id)
	header := mcclient.GetTokenHeaders(userCred)
	resp, err := self.BaremetalSyncRequest(ctx, "POST", url, header, nil)
	if err != nil {
		return nil, errors.Wrap(err, "BaremetalSyncRequest")
	}
	return resp, nil
}

func (self *SHost) AllowPerformInsertIso(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "insert-iso")
}

func (self *SHost) PerformInsertIso(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(self.Status, []string{api.BAREMETAL_READY, api.BAREMETAL_RUNNING}) {
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
		return nil, self.StartInsertIsoTask(ctx, userCred, image.Id, boot, "")
	}
	return nil, httperrors.NewInvalidStatusError("Cannot do insert-iso in status %s", self.Status)
}

func (self *SHost) StartInsertIsoTask(ctx context.Context, userCred mcclient.TokenCredential, imageId string, boot bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(imageId), "image_id")
	if boot {
		data.Add(jsonutils.JSONTrue, "boot")
	}
	data.Add(jsonutils.NewString(api.BAREMETAL_CDROM_ACTION_INSERT), "action")
	self.SetStatus(userCred, api.BAREMETAL_START_INSERT_ISO, "start insert iso task")
	if task, err := taskman.TaskManager.NewTask(ctx, "BaremetalCdromTask", self, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
		return nil
	}
}

func (self *SHost) AllowPerformEjectIso(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "eject-iso")
}

func (self *SHost) PerformEjectIso(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(self.Status, []string{api.BAREMETAL_READY, api.BAREMETAL_RUNNING}) {
		return nil, self.StartEjectIsoTask(ctx, userCred, "")
	}
	return nil, httperrors.NewInvalidStatusError("Cannot do eject-iso in status %s", self.Status)
}

func (self *SHost) StartEjectIsoTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(api.BAREMETAL_CDROM_ACTION_EJECT), "action")
	self.SetStatus(userCred, api.BAREMETAL_START_EJECT_ISO, "start eject iso task")
	if task, err := taskman.TaskManager.NewTask(ctx, "BaremetalCdromTask", self, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
		return nil
	}
}

func (self *SHost) AllowPerformSyncConfig(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "sync-config")
}

func (self *SHost) PerformSyncConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.HostType != api.HOST_TYPE_BAREMETAL {
		return nil, httperrors.NewBadRequestError("Cannot sync config a non-baremetal host")
	}
	self.SetStatus(userCred, api.BAREMETAL_SYNCING_STATUS, "")
	return nil, self.StartSyncConfig(ctx, userCred, "")
}

func (self *SHost) StartSyncConfig(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalSyncConfigTask", self, userCred, nil, parentTaskId, "", nil)
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

func (self *SHost) StartSyncTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	if task, err := taskman.TaskManager.NewTask(ctx, "HostSyncTask", self, userCred, jsonutils.NewDict(), parentTaskId, "",
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
	zoneKeys := fetchZonalQuotaKeys(rbacutils.ScopeDomain, ownerId, zone, nil)
	keys := quotas.SDomainRegionalCloudResourceKeys{}
	keys.SBaseDomainQuotaKeys = zoneKeys.SBaseDomainQuotaKeys
	keys.SRegionalBaseKeys = zoneKeys.SRegionalBaseKeys
	return keys
}

func (model *SHost) GetQuotaKeys() quotas.SDomainRegionalCloudResourceKeys {
	zone := model.GetZone()
	manager := model.GetCloudprovider()
	ownerId := model.GetOwnerId()
	zoneKeys := fetchZonalQuotaKeys(rbacutils.ScopeDomain, ownerId, zone, manager)
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

func (host *SHost) AllowPerformSetReservedResourceForIsolatedDevice(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, input api.IsolatedDeviceReservedResourceInput,
) bool {
	return db.IsDomainAllowPerform(userCred, host, "set-reserved-resource-for-isolated-device")
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

func (host *SHost) IsAssignable(userCred mcclient.TokenCredential) error {
	if db.IsAdminAllowPerform(userCred, host, "assign-host") {
		return nil
	} else if db.IsDomainAllowPerform(userCred, host, "assign-host") &&
		(userCred.GetProjectDomainId() == host.DomainId ||
			host.PublicScope == string(rbacutils.ScopeSystem) ||
			(host.PublicScope == string(rbacutils.ScopeDomain) && utils.IsInStringArray(userCred.GetProjectDomainId(), host.GetSharedDomains()))) {
		return nil
	} else {
		return httperrors.NewNotSufficientPrivilegeError("Only system admin can assign host")
	}
}
