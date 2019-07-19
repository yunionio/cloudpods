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
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SHostManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var HostManager *SHostManager

func init() {
	HostManager = &SHostManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
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
	db.SEnabledStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SBillingResourceBase

	Rack  string `width:"16" charset:"ascii" nullable:"true" get:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True)
	Slots string `width:"16" charset:"ascii" nullable:"true" get:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True)

	AccessMac  string `width:"32" charset:"ascii" nullable:"false" index:"true" list:"admin" update:"admin" create:"admin_required"` // Column(VARCHAR(32, charset='ascii'), nullable=False, index=True)
	AccessIp   string `width:"16" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"`               // Column(VARCHAR(16, charset='ascii'), nullable=True)
	ManagerUri string `width:"256" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"`              // Column(VARCHAR(256, charset='ascii'), nullable=True)

	SysInfo jsonutils.JSONObject `nullable:"true" search:"admin" list:"admin" update:"admin" create:"admin_optional"`              // Column(JSONEncodedDict, nullable=True)
	SN      string               `width:"128" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(128, charset='ascii'), nullable=True)

	CpuCount     int     `nullable:"true" list:"admin" update:"admin" create:"admin_optional"`                           // Column(TINYINT, nullable=True) # cpu count
	NodeCount    int8    `nullable:"true" list:"admin" update:"admin" create:"admin_optional"`                           // Column(TINYINT, nullable=True)
	CpuDesc      string  `width:"64" charset:"ascii" nullable:"true" get:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(64, charset='ascii'), nullable=True)
	CpuMhz       int     `nullable:"true" get:"admin" update:"admin" create:"admin_optional"`                            // Column(Integer, nullable=True) # cpu MHz
	CpuCache     int     `nullable:"true" get:"admin" update:"admin" create:"admin_optional"`                            // Column(Integer, nullable=True) # cpu Cache in KB
	CpuReserved  int     `nullable:"true" default:"0" list:"admin" update:"admin" create:"admin_optional"`               // Column(TINYINT, nullable=True, default=0)
	CpuCmtbound  float32 `nullable:"true" default:"8" list:"admin" update:"admin" create:"admin_optional"`               // = Column(Float, nullable=True)
	CpuMicrocode string  `width:"64" charset:"ascii" nullable:"true" get:"admin" update:"admin" create:"admin_optional"`

	MemSize     int     `nullable:"true" list:"admin" update:"admin" create:"admin_optional"`             // Column(Integer, nullable=True) # memory size in MB
	MemReserved int     `nullable:"true" default:"0" list:"admin" update:"admin" create:"admin_optional"` // Column(Integer, nullable=True, default=0) # memory reserved in MB
	MemCmtbound float32 `nullable:"true" default:"1" list:"admin" update:"admin" create:"admin_optional"` // = Column(Float, nullable=True)

	StorageSize   int                  `nullable:"true" list:"admin" update:"admin" create:"admin_optional"`                            // Column(Integer, nullable=True) # storage size in MB
	StorageType   string               `width:"20" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(20, charset='ascii'), nullable=True)
	StorageDriver string               `width:"20" charset:"ascii" nullable:"true" get:"admin" update:"admin" create:"admin_optional"`  // Column(VARCHAR(20, charset='ascii'), nullable=True)
	StorageInfo   jsonutils.JSONObject `nullable:"true" get:"admin" update:"admin" create:"admin_optional"`                             // Column(JSONEncodedDict, nullable=True)

	IpmiInfo jsonutils.JSONObject `nullable:"true" get:"admin" update:"admin" create:"admin_optional"` // Column(JSONEncodedDict, nullable=True)

	// Status  string = Column(VARCHAR(16, charset='ascii'), nullable=False, default=baremetalstatus.INIT) # status
	HostStatus string `width:"16" charset:"ascii" nullable:"false" default:"offline" list:"admin"` // Column(VARCHAR(16, charset='ascii'), nullable=False, server_default=HOST_OFFLINE, default=HOST_OFFLINE)

	ZoneId string `width:"128" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=False)

	HostType string `width:"36" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)

	Version string `width:"64" charset:"ascii" list:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(64, charset='ascii'))

	IsBaremetal bool `nullable:"true" default:"false" list:"admin" update:"admin" create:"admin_optional"` // Column(Boolean, nullable=True, default=False)

	IsMaintenance bool `nullable:"true" default:"false" list:"admin"` // Column(Boolean, nullable=True, default=False)

	LastPingAt time.Time ``

	ResourceType string `width:"36" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_optional" default:"shared"` // Column(VARCHAR(36, charset='ascii'), nullable=False)

	RealExternalId string `width:"256" charset:"utf8" get:"admin"`

	IsImport bool `nullable:"true" default:"false" list:"admin" create:"admin_optional"`
}

func (manager *SHostManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{ZoneManager},
	}
}

func (self *SHostManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SHostManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SHost) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SHost) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SHost) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (manager *SHostManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	var err error
	q, err = managedResourceFilterByAccount(q, query, "", nil)
	if err != nil {
		return nil, err
	}
	q = managedResourceFilterByCloudType(q, query, "", nil)

	q, err = managedResourceFilterByDomain(q, query, "", nil)
	if err != nil {
		return nil, err
	}

	queryDict := query.(*jsonutils.JSONDict)

	resType, _ := query.GetString("resource_type")
	if len(resType) > 0 {
		queryDict.Remove("resource_type")

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

	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	anyMac, _ := query.GetString("any_mac")
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
	// var scopeQuery *sqlchemy.SSubQuery

	schedTagStr := jsonutils.GetAnyString(query, []string{"schedtag", "schedtag_id"})
	if len(schedTagStr) > 0 {
		schedTag, _ := SchedtagManager.FetchByIdOrName(nil, schedTagStr)
		if schedTag == nil {
			return nil, httperrors.NewResourceNotFoundError("Schedtag %s not found", schedTagStr)
		}
		hostschedtags := HostschedtagManager.Query().SubQuery()
		scopeQuery := hostschedtags.Query(hostschedtags.Field("host_id")).Equals("schedtag_id", schedTag.GetId()).SubQuery()
		q = q.In("id", scopeQuery)
	}

	wireStr := jsonutils.GetAnyString(query, []string{"wire", "wire_id"})
	if len(wireStr) > 0 {
		wire, _ := WireManager.FetchByIdOrName(nil, wireStr)
		if wire == nil {
			return nil, httperrors.NewResourceNotFoundError("Wire %s not found", wireStr)
		}
		hostwires := HostwireManager.Query().SubQuery()
		scopeQuery := hostwires.Query(hostwires.Field("host_id")).Equals("wire_id", wire.GetId()).SubQuery()
		q = q.In("id", scopeQuery)
	}

	storageStr := jsonutils.GetAnyString(query, []string{"storage", "storage_id"})
	notAttached := jsonutils.QueryBoolean(query, "storage_not_attached", false)
	if len(storageStr) > 0 {
		storage, _ := StorageManager.FetchByIdOrName(nil, storageStr)
		if storage == nil {
			return nil, httperrors.NewResourceNotFoundError("Storage %s not found", storageStr)
		}
		hoststorages := HoststorageManager.Query().SubQuery()
		scopeQuery := hoststorages.Query(hoststorages.Field("host_id")).Equals("storage_id", storage.GetId()).SubQuery()
		if !notAttached {
			q = q.In("id", scopeQuery)
		} else {
			q = q.NotIn("id", scopeQuery)
		}
	}

	q, err = managedResourceFilterByZone(q, query, "", nil)
	/*zoneStr := jsonutils.GetAnyString(query, []string{"zone", "zone_id"})
	if len(zoneStr) > 0 {
		zone, err := ZoneManager.FetchByIdOrName(nil, zoneStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(ZoneManager.Keyword(), zoneStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Filter(sqlchemy.Equals(q.Field("zone_id"), zone.GetId()))

		queryDict.Remove("zone_id")
	}*/

	q, err = managedResourceFilterByRegion(q, query, "zone_id", func() *sqlchemy.SQuery {
		return ZoneManager.Query("id")
	})

	/*regionStr := jsonutils.GetAnyString(query, []string{"region", "region_id"})
	if len(regionStr) > 0 {
		region, err := CloudregionManager.FetchByIdOrName(nil, regionStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudregionManager.Keyword(), regionStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		subq := ZoneManager.Query("id").Equals("cloudregion_id", region.GetId()).SubQuery()
		q = q.Filter(sqlchemy.In(q.Field("zone_id"), subq))
	}*/

	hypervisorStr := jsonutils.GetAnyString(query, []string{"hypervisor"})
	if len(hypervisorStr) > 0 {
		hostType, ok := api.HYPERVISOR_HOSTTYPE[hypervisorStr]
		if !ok {
			return nil, httperrors.NewInputParameterError("not supported hypervisor %s", hypervisorStr)
		}
		q = q.Filter(sqlchemy.Equals(q.Field("host_type"), hostType))
	}

	usable := jsonutils.QueryBoolean(query, "usable", false)
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
	}

	if query.Contains("is_empty") {
		isEmpty := jsonutils.QueryBoolean(query, "is_empty", false)
		sq := GuestManager.Query("host_id").IsNotEmpty("host_id").GroupBy("host_id").SubQuery()
		if isEmpty {
			q = q.NotIn("id", sq)
		} else {
			q = q.In("id", sq)
		}
	}

	if query.Contains("baremetal") {
		isBaremetal := jsonutils.QueryBoolean(query, "baremetal", false)
		if isBaremetal {
			q = q.Equals("host_type", api.HOST_TYPE_BAREMETAL)
		} else {
			q = q.NotEquals("host_type", api.HOST_TYPE_BAREMETAL)
		}
	}

	return q, nil
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
	if self.Enabled {
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
	return self.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
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
				return err
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
	return self.SEnabledStatusStandaloneResourceBase.Delete(ctx, userCred)
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
	if err != nil {
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
		log.Errorf("GetHoststorageOfId fail %s", err)
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
		log.Errorf("GetHoststorageByExternalId fail %s", err)
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
	log.Errorf("Cannot find baremetalstorage??")
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
	if bs == nil {
		// 1. create storage
		storage := SStorage{}
		storage.Name = fmt.Sprintf("storage%s", self.GetName())
		storage.Capacity = capacity
		storage.StorageType = api.STORAGE_BAREMETAL
		storage.MediumType = self.StorageType
		storage.Cmtbound = 1.0
		storage.Status = api.STORAGE_ONLINE
		storage.ZoneId = zoneId
		err := StorageManager.TableSpec().Insert(&storage)
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
		err = HoststorageManager.TableSpec().Insert(&bmStorage)
		if err != nil {
			return nil, fmt.Errorf("Create baremetal hostStorage error: %v", err)
		}
		bmStorage.SetModelManager(HoststorageManager, &bmStorage)
		db.OpsLog.LogAttachEvent(ctx, self, &storage, userCred, bmStorage.GetShortDesc(ctx))
		return nil, nil
	}
	storage := bs.GetStorage()
	if capacity != int64(storage.Capacity) {
		diff, err := db.Update(storage, func() error {
			storage.Capacity = capacity
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("Update baremetal storage error: %v", err)
		}
		db.OpsLog.LogEvent(storage, db.ACT_UPDATE, diff, userCred)
	}
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

func (self *SHost) GetAttachedStorages(storageType string) []SStorage {
	return self._getAttachedStorages(tristate.False, tristate.True, storageType)
}

func (self *SHost) _getAttachedStorages(isBaremetal tristate.TriState, enabled tristate.TriState, storageType string) []SStorage {
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
		q = q.Equals("storage_type", storageType)
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
	storages := self.GetAttachedStorages("")
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
	regionUri, err := auth.GetServiceURL("compute_v2", options.Options.Region, "", "")
	if err != nil {
		return nil, err
	}
	var script string
	script += fmt.Sprintf("curl -k -fsSL -H 'X-Auth-Token: %s' %s/misc/bm-prepare-script", userCred.GetTokenString(), regionUri)
	res := jsonutils.NewDict()
	res.Add(jsonutils.NewString(script), "script")
	return res, nil
}

func (self *SHostManager) ClearAllSchedDescCache() error {
	return self.ClearSchedDescSessionCache("", "")
}

func (self *SHostManager) ClearSchedDescCache(hostId string) error {
	return self.ClearSchedDescSessionCache(hostId, "")
}

func (self *SHostManager) ClearSchedDescSessionCache(hostId, sessionId string) error {
	s := auth.GetAdminSession(context.Background(), options.Options.Region, "")
	return modules.SchedManager.CleanCache(s, hostId, sessionId)
}

func (self *SHost) ClearSchedDescCache() error {
	return self.ClearSchedDescSessionCache("")
}

func (self *SHost) ClearSchedDescSessionCache(sessionId string) error {
	return HostManager.ClearSchedDescSessionCache(self.Id, sessionId)
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
		if !self.Enabled {
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
	specInfo.Manufacture = manufacture
	specInfo.Model = model
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
	Capacity  int64 `json:"capacity,omitzero"`
	Used      int64 `json:"used_capacity,omitzero"`
	Wasted    int64 `json:"waste_capacity,omitzero"`
	VCapacity int64 `json:"virtual_capacity,omitzero"`
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
}

func (cap *SStorageCapacity) ToJson() *jsonutils.JSONDict {
	ret := jsonutils.Marshal(cap).(*jsonutils.JSONDict)
	ret.Add(jsonutils.NewFloat(cap.GetCommitRate()), "commit_rate")
	ret.Add(jsonutils.NewInt(int64(cap.GetFree())), "free_capacity")
	return ret
}

func (self *SHost) GetAttachedLocalStorageCapacity() SStorageCapacity {
	ret := SStorageCapacity{}
	storages := self.GetAttachedStorages("")
	for _, s := range storages {
		if !utils.IsInStringArray(s.StorageType, api.HOST_STORAGE_LOCAL_TYPES) {
			continue
		}
		ret.Add(s.getStorageCapacity())
	}
	return ret
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
	storages := self.GetAttachedStorages("")
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

func (self *SHost) GetGuests() []SGuest {
	q := self.GetGuestsQuery()
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

func (self *SHost) GetAttach2Network(network *SNetwork) *SHostnetwork {
	q := self.GetBaremetalnetworksQuery()
	q = q.Equals("network_id", network.Id)
	hn := SHostnetwork{}
	hn.SetModelManager(HostnetworkManager, &hn)

	err := q.First(&hn)
	if err != nil {
		log.Errorf("GetAttach2Network fail %s", err)
		return nil
	}
	return &hn
}

func (self *SHost) isAttach2Network(network *SNetwork) bool {
	hn := self.GetAttach2Network(network)
	return hn != nil
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
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

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
		err = commondb[i].syncWithCloudHost(ctx, userCred, commonext[i])
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
			_, err = self.PerformDisable(ctx, userCred, nil, nil)
		}
	} else {
		err = self.RealDelete(ctx, userCred)
	}
	return err
}

func (self *SHost) syncWithCloudHost(ctx context.Context, userCred mcclient.TokenCredential, extHost cloudprovider.ICloudHost) error {
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

		self.IsEmulated = extHost.IsEmulated()
		self.Enabled = extHost.GetEnabled()

		self.IsMaintenance = extHost.GetIsMaintenance()
		self.Version = extHost.GetVersion()

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudZone error %s", err)
		return err
	}

	db.OpsLog.LogSyncUpdate(self, diff, userCred)

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

func (manager *SHostManager) newFromCloudHost(ctx context.Context, userCred mcclient.TokenCredential, extHost cloudprovider.ICloudHost, provider *SCloudprovider, izone *SZone) (*SHost, error) {
	host := SHost{}
	host.SetModelManager(manager, &host)

	if izone == nil {
		// onpremise host
		wire, err := WireManager.GetOnPremiseWireOfIp(extHost.GetAccessIp())
		if err != nil {
			msg := fmt.Sprintf("fail to find wire for host %s %s: %s", extHost.GetName(), extHost.GetAccessIp(), err)
			log.Errorf(msg)
			return nil, fmt.Errorf(msg)
		}
		izone = wire.GetZone()
	}

	newName, err := db.GenerateName(manager, userCred, extHost.GetName())
	if err != nil {
		return nil, fmt.Errorf("generate name fail %s", err)
	}
	host.Name = newName
	host.ExternalId = extHost.GetGlobalId()
	host.ZoneId = izone.Id

	host.HostType = extHost.GetHostType()

	host.Status = extHost.GetStatus()
	host.HostStatus = extHost.GetHostStatus()
	host.Enabled = extHost.GetEnabled()

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
	host.MemCmtbound = 1.0

	host.ManagerId = provider.Id
	host.IsEmulated = extHost.IsEmulated()

	host.IsMaintenance = extHost.GetIsMaintenance()
	host.Version = extHost.GetVersion()

	err = manager.TableSpec().Insert(&host)
	if err != nil {
		log.Errorf("newFromCloudHost fail %s", err)
		return nil, err
	}

	db.OpsLog.LogEvent(&host, db.ACT_CREATE, host.GetShortDesc(ctx), userCred)

	if err := manager.ClearSchedDescCache(host.Id); err != nil {
		log.Errorf("ClearSchedDescCache for host %s error %v", host.Name, err)
	}

	return &host, nil
}

func (self *SHost) SyncHostStorages(ctx context.Context, userCred mcclient.TokenCredential, storages []cloudprovider.ICloudStorage, provider *SCloudprovider) ([]SStorage, []cloudprovider.ICloudStorage, compare.SyncResult) {
	lockman.LockClass(ctx, StorageManager, db.GetLockClassKey(StorageManager, userCred))
	defer lockman.ReleaseClass(ctx, StorageManager, db.GetLockClassKey(StorageManager, userCred))

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
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		log.Infof("host %s is still connected with %s, to update ...", self.Id, commondb[i].Id)
		err := self.syncWithCloudHostStorage(ctx, userCred, &commondb[i], commonext[i])
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

func (self *SHost) syncWithCloudHostStorage(ctx context.Context, userCred mcclient.TokenCredential, localStorage *SStorage, extStorage cloudprovider.ICloudStorage) error {
	// do nothing
	hs := self.GetHoststorageOfId(localStorage.Id)
	err := hs.syncWithCloudHostStorage(userCred, extStorage)
	if err != nil {
		return err
	}
	s := hs.GetStorage()
	err = s.syncWithCloudStorage(ctx, userCred, extStorage)
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
	err := HoststorageManager.TableSpec().Insert(&hs)
	if err != nil {
		return err
	}

	db.OpsLog.LogAttachEvent(ctx, self, storage, userCred, nil)

	return nil
}

func (self *SHost) newCloudHostStorage(ctx context.Context, userCred mcclient.TokenCredential, extStorage cloudprovider.ICloudStorage, provider *SCloudprovider) (*SStorage, error) {
	storageObj, err := db.FetchByExternalId(StorageManager, extStorage.GetGlobalId())
	if err != nil {
		if err == sql.ErrNoRows {
			// no cloud storage found, this may happen for on-premise host
			// create the storage right now
			storageObj, err = StorageManager.newFromCloudStorage(ctx, userCred, extStorage, provider, self.GetZone())
			if err != nil {
				log.Errorf("create by cloud storage fail %s", err)
				return nil, err
			}
		} else {
			log.Errorf("%s", err)
			return nil, err
		}
	}
	storage := storageObj.(*SStorage)
	err = self.Attach2Storage(ctx, userCred, storage, extStorage.GetMountPoint())
	return storage, err
}

func (self *SHost) SyncHostWires(ctx context.Context, userCred mcclient.TokenCredential, wires []cloudprovider.ICloudWire) compare.SyncResult {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

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
	err := HostwireManager.TableSpec().Insert(&hs)
	if err != nil {
		return err
	}
	db.OpsLog.LogAttachEvent(ctx, self, wire, userCred, nil)
	return nil
}

func (self *SHost) newCloudHostWire(ctx context.Context, userCred mcclient.TokenCredential, extWire cloudprovider.ICloudWire) error {
	wireObj, err := db.FetchByExternalId(WireManager, extWire.GetGlobalId())
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
	lockman.LockClass(ctx, GuestManager, db.GetLockClassKey(GuestManager, syncOwnerId))
	defer lockman.ReleaseClass(ctx, GuestManager, db.GetLockClassKey(GuestManager, syncOwnerId))

	syncVMPairs := make([]SGuestSyncResult, 0)
	syncResult := compare.SyncResult{}

	dbVMs := self.GetGuests()

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

	err := compare.CompareSets(dbVMs, vms, &removed, &commondb, &commonext, &added)
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
		vm, err := db.FetchByExternalId(GuestManager, added[i].GetGlobalId())
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
			_host, err := db.FetchByExternalId(HostManager, ihost.GetGlobalId())
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
		network, err := netInterface.GetCandidateNetworkForIp(auth.AdminCredential(), ipAddr)
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

func (self *SHost) GetNetworkWithIdAndCredential(netId string, userCred mcclient.TokenCredential, reserved bool) (*SNetwork, error) {
	networks := NetworkManager.Query().SubQuery()
	hostwires := HostwireManager.Query().SubQuery()
	hosts := HostManager.Query().SubQuery()

	q := networks.Query()
	q = q.Join(hostwires, sqlchemy.Equals(hostwires.Field("wire_id"), networks.Field("wire_id")))
	q = q.Join(hosts, sqlchemy.Equals(hosts.Field("id"), hostwires.Field("host_id")))
	q = q.Filter(sqlchemy.Equals(hosts.Field("id"), self.Id))
	q = q.Filter(sqlchemy.Equals(networks.Field("id"), netId))

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
	rangeObj db.IStandaloneModel,
	hostStatus, status string,
	hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	enabled, isBaremetal tristate.TriState,
) *sqlchemy.SQuery {
	hosts := manager.Query().SubQuery()
	/*
			    MemSize     int
		    MemReserved int
		    MemCmtbound float32
		    CpuCount    int8
		    CpuReserved int8
		    CpuCmtbound float32
		    StorageSize int
	*/
	q := hosts.Query(
		hosts.Field("mem_size"),
		hosts.Field("mem_reserved"),
		hosts.Field("mem_cmtbound"),
		hosts.Field("cpu_count"),
		hosts.Field("cpu_reserved"),
		hosts.Field("cpu_cmtbound"),
		hosts.Field("storage_size"),
	)
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
		cond := sqlchemy.IsFalse
		if isBaremetal.Bool() {
			cond = sqlchemy.IsTrue
		}
		q = q.Filter(cond(hosts.Field("is_baremetal")))
	}
	q = AttachUsageQuery(q, hosts, hostTypes, resourceTypes, providers, brands, cloudEnv, rangeObj)
	return q
}

type HostStat struct {
	MemSize     int
	MemReserved int
	MemCmtbound float32
	CpuCount    int
	CpuReserved int
	CpuCmtbound float32
	StorageSize int
}

type HostsCountStat struct {
	StorageSize   int64
	Count         int64
	Memory        int64
	MemoryVirtual float64
	CPU           int64
	CPUVirtual    float64
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
		tStore int64   = 0
		tCnt   int64   = 0
		tMem   int64   = 0
		tVmem  float64 = 0.0
		tCPU   int64   = 0
		tVCPU  float64 = 0.0
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
		tCPU += int64(aCpu)
		if stat.MemCmtbound <= 0.0 {
			stat.MemCmtbound = options.Options.DefaultMemoryOvercommitBound
		}
		if stat.CpuCmtbound <= 0.0 {
			stat.CpuCmtbound = options.Options.DefaultCPUOvercommitBound
		}
		tVmem += float64(float32(aMem) * stat.MemCmtbound)
		tVCPU += float64(float32(aCpu) * stat.CpuCmtbound)
	}
	return HostsCountStat{
		StorageSize:   tStore,
		Count:         tCnt,
		Memory:        tMem,
		MemoryVirtual: tVmem,
		CPU:           tCPU,
		CPUVirtual:    tVCPU,
	}
}

func (manager *SHostManager) TotalCount(
	userCred mcclient.IIdentityProvider,
	rangeObj db.IStandaloneModel,
	hostStatus, status string,
	hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	enabled, isBaremetal tristate.TriState,
) HostsCountStat {
	return manager.calculateCount(manager.totalCountQ(userCred, rangeObj, hostStatus, status, hostTypes, resourceTypes, providers, brands, cloudEnv, enabled, isBaremetal))
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
		if err == cloudprovider.ErrNotFound {
			return nil, nil, cloudprovider.ErrNotFound
		}
		log.Errorf("fail to find ihost by id %s %s", self.ExternalId, err)
		return nil, nil, fmt.Errorf("fail to find ihost by id %s", err)
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

func (self *SHost) getMoreDetails(ctx context.Context, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	/*zone := self.GetZone()
	if zone != nil {
		extra.Add(jsonutils.NewString(zone.Id), "zone_id")
		extra.Add(jsonutils.NewString(zone.Name), "zone")
		if len(zone.ExternalId) > 0 {
			extra.Add(jsonutils.NewString(zone.ExternalId), "")
		}
		region := zone.GetRegion()
		if region != nil {
			extra.Add(jsonutils.NewString(zone.GetRegion().GetName()), "region")
			extra.Add(jsonutils.NewString(zone.GetRegion().GetId()), "region_id")
		}
	}*/

	info := self.getCloudProviderInfo()
	extra.Update(jsonutils.Marshal(&info))

	server := self.GetBaremetalServer()
	if server != nil {
		extra.Add(jsonutils.NewString(server.Id), "server_id")
		extra.Add(jsonutils.NewString(server.Name), "server")
		if self.HostType == api.HOST_TYPE_BAREMETAL {
			extra.Add(jsonutils.NewString(strings.Join(server.GetRealIPs(), ",")), "server_ips")
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
		extra.Add(jsonutils.NewInt(int64(len(nicInfos))), "nic_count")
		extra.Add(jsonutils.NewArray(nicInfos...), "nic_info")
	}
	extra = GetSchedtagsDetailsToResource(self, ctx, extra)
	var usage *SHostGuestResourceUsage
	if options.Options.IgnoreNonrunningGuests {
		usage = self.getGuestsResource(api.VM_RUNNING)
	} else {
		usage = self.getGuestsResource("")
	}
	if usage != nil {
		extra.Add(jsonutils.NewInt(int64(usage.GuestVcpuCount)), "cpu_commit")
		extra.Add(jsonutils.NewInt(int64(usage.GuestVmemSize)), "mem_commit")
	}
	containerCount, _ := self.GetContainerCount(nil)
	runningContainerCount, _ := self.GetContainerCount(api.VM_RUNNING_STATUS)
	guestCount, _ := self.GetGuestCount()
	nonesysGuestCnt, _ := self.GetNonsystemGuestCount()
	runningGuestCnt, _ := self.GetRunningGuestCount()
	extra.Add(jsonutils.NewInt(int64(guestCount-containerCount)), "guests")
	extra.Add(jsonutils.NewInt(int64(nonesysGuestCnt-containerCount)), "nonsystem_guests")
	extra.Add(jsonutils.NewInt(int64(runningGuestCnt-runningContainerCount)), "running_guests")
	totalCpu := self.GetCpuCount()
	cpuCommitRate := 0.0
	if totalCpu > 0 && usage.GuestVcpuCount > 0 {
		cpuCommitRate = float64(usage.GuestVcpuCount) * 1.0 / float64(totalCpu)
	}
	extra.Add(jsonutils.NewFloat(cpuCommitRate), "cpu_commit_rate")
	totalMem := self.GetMemSize()
	memCommitRate := 0.0
	if totalMem > 0 && usage.GuestVmemSize > 0 {
		memCommitRate = float64(usage.GuestVmemSize) * 1.0 / float64(totalMem)
	}
	extra.Add(jsonutils.NewFloat(memCommitRate), "mem_commit_rate")
	capa := self.GetAttachedLocalStorageCapacity()
	extra.Add(jsonutils.NewInt(int64(capa.Capacity)), "storage")
	extra.Add(jsonutils.NewInt(int64(capa.Used)), "storage_used")
	extra.Add(jsonutils.NewInt(int64(capa.Wasted)), "storage_waste")
	extra.Add(jsonutils.NewInt(int64(capa.VCapacity)), "storage_virtual")
	extra.Add(jsonutils.NewInt(int64(capa.GetFree())), "storage_free")
	extra.Add(jsonutils.NewFloat(capa.GetCommitRate()), "storage_commit_rate")
	extra.Add(self.GetHardwareSpecification(), "spec")

	// extra = self.SManagedResourceBase.getExtraDetails(ctx, extra)

	if self.IsPrepaidRecycle() {
		extra.Add(jsonutils.JSONTrue, "is_prepaid_recycle")
	} else {
		extra.Add(jsonutils.JSONFalse, "is_prepaid_recycle")
	}

	return extra
}

func (self *SHost) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SEnabledStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(ctx, extra)
}

func (self *SHost) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SEnabledStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreDetails(ctx, extra), nil
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
	localStorages := self.GetAttachedStorages(api.STORAGE_LOCAL)
	for i := 0; i < len(localStorages); i += 1 {
		sc := localStorages[i].GetStoragecache()
		if sc != nil {
			return sc
		}
	}
	return nil
}

func (self *SHost) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	kwargs := data.(*jsonutils.JSONDict)
	ipmiInfo, err := fetchIpmiInfo(kwargs, self.Id)
	if err != nil {
		log.Errorln(err.Error())
	} else if ipmiInfo.Length() > 0 {
		_, err := self.SaveUpdates(func() error {
			self.IpmiInfo = ipmiInfo
			return nil
		})
		if err != nil {
			log.Errorln(err.Error())
		}
	}
}

func (manager *SHostManager) ValidateSizeParams(data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	memStr, _ := data.GetString("mem_size")
	if len(memStr) > 0 {
		if !regutils.MatchSize(memStr) {
			return nil, fmt.Errorf("Memory size must be number[+unit], like 256M, 1G or 256")
		}
		memSize, err := fileutils.GetSizeMb(memStr, 'M', 1024)
		if err != nil {
			return nil, err
		}
		data.Set("mem_size", jsonutils.NewInt(int64(memSize)))
	}
	memReservedStr, _ := data.GetString("mem_reserved")
	if len(memReservedStr) > 0 {
		if !regutils.MatchSize(memReservedStr) {
			return nil, fmt.Errorf("Memory size must be number[+unit], like 256M, 1G or 256")
		}
		memSize, err := fileutils.GetSizeMb(memReservedStr, 'M', 1024)
		if err != nil {
			return nil, err
		}
		data.Set("mem_reserved", jsonutils.NewInt(int64(memSize)))
	}
	cpuCacheStr, _ := data.GetString("cpu_cache")
	if len(cpuCacheStr) > 0 {
		if !regutils.MatchSize(cpuCacheStr) {
			return nil, fmt.Errorf("Illegal cpu cache size %s", cpuCacheStr)
		}
		cpuCache, err := fileutils.GetSizeKb(cpuCacheStr, 'K', 1024)
		if err != nil {
			return nil, err
		}
		data.Set("cpu_cache", jsonutils.NewInt(int64(cpuCache)))
	}
	return data, nil
}

func inputUniquenessCheck(data *jsonutils.JSONDict, zoneId string, hostId string) (*jsonutils.JSONDict, error) {
	for _, key := range []string{
		"manager_uri",
		"access_ip",
	} {
		val, _ := data.GetString(key)
		if len(val) > 0 {
			q := HostManager.Query().Equals(key, val)
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
				return nil, httperrors.NewInternalServerError("check %s duplication fail %s", key, err)
			}
			if cnt > 0 {
				return nil, httperrors.NewConflictError("duplicate %s %s", key, val)
			}
		}
	}

	accessMac, _ := data.GetString("access_mac")
	if len(accessMac) > 0 {
		accessMac2 := netutils.FormatMacAddr(accessMac)
		if len(accessMac2) == 0 {
			return nil, httperrors.NewInputParameterError("invalid macAddr %s", accessMac)
		}
		q := HostManager.Query().Equals("access_mac", accessMac2)
		if len(hostId) > 0 {
			q = q.NotEquals("id", hostId)
		}
		cnt, err := q.CountWithError()
		if err != nil {
			return nil, httperrors.NewInternalServerError("check access_mac duplication fail %s", err)
		}
		if cnt > 0 {
			return nil, httperrors.NewConflictError("duplicate access_mac %s", accessMac)
		}
		data.Set("access_mac", jsonutils.NewString(accessMac2))
	}
	return data, nil
}

func (manager *SHostManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	zoneId := jsonutils.GetAnyString(data, []string{"zone_id", "zone"})
	if len(zoneId) > 0 {
		zoneObj, err := ZoneManager.FetchByIdOrName(userCred, zoneId)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(ZoneManager.Keyword(), zoneId)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		zoneId = zoneObj.GetId()
		data.Set("zone_id", jsonutils.NewString(zoneObj.GetId()))
	}

	data, err := inputUniquenessCheck(data, zoneId, "")
	if err != nil {
		return nil, err
	}

	data, err = manager.ValidateSizeParams(data)
	if err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}
	memReserved, err := data.Int("mem_reserved")
	if err != nil {
		hostType, _ := data.GetString("host_type")
		if hostType != api.HOST_TYPE_BAREMETAL {
			memSize, _ := data.Int("mem_size")
			memReserved = memSize / 8
			if memReserved > 4096 {
				memReserved = 4096
			}
			data.Set("mem_reserved", jsonutils.NewInt(memReserved))
		} else {
			data.Set("mem_reserved", jsonutils.NewInt(0))
		}
	}
	ipmiInfo, err := fetchIpmiInfo(data, "")
	if err != nil {
		log.Errorln(err.Error())
		return nil, httperrors.NewInputParameterError("%s", err)
	}
	ipmiIpAddr, _ := ipmiInfo.GetString("ip_addr")
	if len(ipmiIpAddr) > 0 && !NetworkManager.IsValidOnPremiseNetworkIP(ipmiIpAddr) {
		return nil, httperrors.NewInputParameterError("%s is out of network IP ranges", ipmiIpAddr)
	}
	ipmiPasswd, _ := ipmiInfo.GetString("password")
	if len(ipmiPasswd) > 0 && !seclib2.MeetComplxity(ipmiPasswd) {
		return nil, httperrors.NewWeakPasswordError()
	}
	return manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (self *SHost) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := inputUniquenessCheck(data, self.ZoneId, self.Id)
	if err != nil {
		return nil, err
	}

	data, err = HostManager.ValidateSizeParams(data)
	if err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}
	ipmiInfo, err := fetchIpmiInfo(data, self.Id)
	if err != nil {
		return nil, err
	}
	if ipmiInfo.Length() > 0 {
		ipmiIpAddr, _ := ipmiInfo.GetString("ip_addr")
		if len(ipmiIpAddr) > 0 && !NetworkManager.IsValidOnPremiseNetworkIP(ipmiIpAddr) {
			return nil, httperrors.NewInputParameterError("%s is out of network IP ranges", ipmiIpAddr)
		}
		val := jsonutils.NewDict()
		val.Update(self.IpmiInfo)
		val.Update(ipmiInfo)
		data.Set("ipmi_info", val)
	}
	data, err = self.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	if data.Contains("name") {
		self.UpdateDnsRecords(false)
	}
	return data, nil
}

func (self *SHost) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)

	if data.Contains("cpu_cmtbound") || data.Contains("mem_cmtbound") {
		self.ClearSchedDescCache()
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

func fetchIpmiInfo(data *jsonutils.JSONDict, hostId string) (*jsonutils.JSONDict, error) {
	IPMI_KEY_PERFIX := "ipmi_"
	ipmiInfo := jsonutils.NewDict()
	kv, _ := data.GetMap()
	var err error
	for key := range kv {
		if strings.HasPrefix(key, IPMI_KEY_PERFIX) {
			value, _ := data.GetString(key)
			subkey := key[len(IPMI_KEY_PERFIX):]
			if subkey == "password" && len(hostId) > 0 {
				value, err = utils.EncryptAESBase64(hostId, value)
				if err != nil {
					log.Errorf("encrypt password failed %s", err)
					return nil, err
				}
			} else if subkey == "ip_addr" {
				if !regutils.MatchIP4Addr(value) {
					msg := fmt.Sprintf("%s: %s not valid ipv4 address", key, value)
					log.Errorf(msg)
					err = fmt.Errorf(msg)
					return nil, err
				}
			}
			ipmiInfo.Set(subkey, jsonutils.NewString(value))
		}
	}
	return ipmiInfo, nil
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
			return guest.PerformStop(ctx, userCred, query, data)
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
			return nil
		})
		if err != nil {
			return nil, err
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
			self.Status = api.BAREMETAL_RUNNING
			return nil
		})
		if err != nil {
			return nil, err
		}
		db.OpsLog.LogEvent(self, db.ACT_ONLINE, "", userCred)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_ONLINE, nil, userCred, true)
		self.SyncAttachedStorageStatus()
		self.StartSyncAllGuestsStatusTask(ctx, userCred)
	}
	return nil, nil
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

func (self *SHost) PerformPrepare(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(self.Status, []string{api.BAREMETAL_READY, api.BAREMETAL_RUNNING, api.BAREMETAL_PREPARE_FAIL}) {
		var onfinish string
		if self.GetBaremetalServer() != nil {
			if self.Status == api.BAREMETAL_RUNNING {
				onfinish = "restart"
			} else if self.Status == api.BAREMETAL_READY {
				onfinish = "shotdown"
			}
		}
		return nil, self.StartPrepareTask(ctx, userCred, onfinish, "")
	}
	return nil, httperrors.NewInvalidStatusError("Cannot prepare baremetal in status %s", self.Status)
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
	err = db.NewNameValidator(GuestManager, userCred, name)
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
	err = GuestManager.TableSpec().Insert(guest)
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
			swNets, err := sw.getNetworks()
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
		err = NetInterfaceManager.TableSpec().Insert(netif)
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
			return err
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
			hw, err := HostwireManager.FetchByIdsAndMac(self.Id, sw.Id, mac)
			if err != nil {
				hw = &SHostwire{}
				hw.Bridge = bridge
				hw.Interface = strInterface
				hw.HostId = self.Id
				hw.WireId = sw.Id
				hw.IsMaster = isMaster
				hw.MacAddr = mac
				err := HostwireManager.TableSpec().Insert(hw)
				if err != nil {
					return err
				}
			} else {
				db.Update(hw, func() error {
					hw.Bridge = bridge
					hw.Interface = strInterface
					// hw.MacAddr = mac
					hw.IsMaster = isMaster
					return nil
				})
			}
		}
	}
	if len(ipAddr) > 0 {
		err = self.EnableNetif(ctx, userCred, netif, "", ipAddr, "", "", reserve, requireDesignatedIp)
		if err != nil {
			return httperrors.NewBadRequestError(err.Error())
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
		return nil, httperrors.NewBadRequestError(err.Error())
	}
	return nil, nil
}

func (self *SHost) EnableNetif(ctx context.Context, userCred mcclient.TokenCredential, netif *SNetInterface, network, ipAddr, allocDir string, netType string, reserve, requireDesignatedIp bool) error {
	bn := netif.GetBaremetalNetwork()
	if bn != nil {
		return nil
	}
	var net *SNetwork
	var err error
	if len(ipAddr) > 0 {
		net, err = netif.GetCandidateNetworkForIp(userCred, ipAddr)
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
	hw, err := HostwireManager.FetchByIdsAndMac(self.Id, wire.Id, netif.Mac)
	if hw == nil {
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
			net, err = wire.GetCandidatePrivateNetwork(userCred, false, netTypes)
			if err != nil {
				return fmt.Errorf("fail to find network %s", err)
			}
			if net == nil {
				return fmt.Errorf("No network found")
			}
		}
	} else if net.WireId != wire.Id {
		return fmt.Errorf("conflict??? candiate net is not on wire")
	}
	return self.Attach2Network(ctx, userCred, netif, net, ipAddr, allocDir, reserve, requireDesignatedIp)
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
		return nil, httperrors.NewBadRequestError(err.Error())
	}
	return nil, nil
}

func (self *SHost) DisableNetif(ctx context.Context, userCred mcclient.TokenCredential, netif *SNetInterface, reserve bool) error {
	bn := netif.GetBaremetalNetwork()
	if bn != nil {
		self.UpdateDnsRecord(netif, false)
		self.DeleteBaremetalnetwork(ctx, userCred, bn, reserve)
	}
	return nil
}

func (self *SHost) Attach2Network(ctx context.Context, userCred mcclient.TokenCredential, netif *SNetInterface, net *SNetwork, ipAddr, allocDir string, reserved, requireDesignatedIp bool) error {
	lockman.LockObject(ctx, net)
	defer lockman.ReleaseObject(ctx, net)

	usedAddr := net.GetUsedAddresses()
	freeIp, err := net.GetFreeIP(ctx, userCred, usedAddr, nil, ipAddr, IPAddlocationDirection(allocDir), reserved)
	if err != nil {
		log.Errorf("attach2network: %s", err)
		return err
	}
	if len(ipAddr) > 0 && ipAddr != freeIp && requireDesignatedIp {
		return fmt.Errorf("IP address %s is occupied, get %s instead", ipAddr, freeIp)
	}
	bn := &SHostnetwork{}
	bn.BaremetalId = self.Id
	bn.NetworkId = net.Id
	bn.IpAddr = freeIp
	bn.MacAddr = netif.Mac
	err = HostnetworkManager.TableSpec().Insert(bn)
	if err != nil {
		log.Errorf("HostnetworkManager.TableSpec().Insert fail %s", err)
		return err
	}
	db.OpsLog.LogAttachEvent(ctx, self, net, userCred, jsonutils.NewString(freeIp))
	self.UpdateDnsRecord(netif, true)
	net.UpdateBaremetalNetmap(bn, self.GetNetifName(netif))
	return nil
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
	log.Infof("Remove wire")
	err := netif.Remove(ctx, userCred)
	if err != nil {
		return err
	}
	if wire != nil {
		log.Infof("Remove wire")
		others := self.GetNetifsOnWire(wire)
		if len(others) == 0 {
			hw, _ := HostwireManager.FetchByIdsAndMac(self.Id, wire.Id, netif.Mac)
			if hw != nil {
				db.OpsLog.LogDetachEvent(ctx, self, wire, userCred, jsonutils.NewString(fmt.Sprintf("disable netif %s", self.AccessMac)))
				log.Infof("Detach host wire because of remove netif %s", netif.Mac)
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
	data jsonutils.JSONObject,
) bool {
	return self.SEnabledStatusStandaloneResourceBase.AllowPerformEnable(ctx, userCred, query, data)
}

func (self *SHost) PerformEnable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if !self.Enabled {
		_, err := self.SEnabledStatusStandaloneResourceBase.PerformEnable(ctx, userCred, query, data)
		if err != nil {
			return nil, err
		}
		self.SyncAttachedStorageStatus()
	}
	return nil, nil
}

func (self *SHost) AllowPerformDisable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return self.SEnabledStatusStandaloneResourceBase.AllowPerformDisable(ctx, userCred, query, data)
}

func (self *SHost) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Enabled {
		_, err := self.SEnabledStatusStandaloneResourceBase.PerformDisable(ctx, userCred, query, data)
		if err != nil {
			return nil, err
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
	if self.HostStatus != api.HOST_ONLINE {
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
	sc := self.GetLocalStoragecache()
	if sc == nil {
		return fmt.Errorf("No local storage cache found")
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
	params := input.JSON(input)
	// ownerId := userCred.GetProjectId()
	guest, err := db.DoCreate(GuestManager, ctx, userCred, nil, params, userCred)
	if err != nil {
		return nil, err
	}
	func() {
		lockman.LockObject(ctx, guest)
		defer lockman.ReleaseObject(ctx, guest)

		guest.PostCreate(ctx, userCred, userCred, nil, params)
	}()
	log.Infof("Host convert to %s", guest.GetName())
	db.OpsLog.LogEvent(self, db.ACT_CONVERT_START, "", userCred)
	db.OpsLog.LogEvent(guest, db.ACT_CREATE, "Convert hypervisor", userCred)

	opts := jsonutils.NewDict()
	opts.Set("server_params", params)
	opts.Set("server_id", jsonutils.NewString(guest.GetId()))
	opts.Set("convert_host_type", jsonutils.NewString(hostType))

	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalConvertHypervisorTask", self, userCred, opts, "", "", nil)
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
	if self.Enabled {
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
		return nil, httperrors.NewNotAcceptableError(err.Error())
	}
	guests := self.GetGuests()
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
					// do nothing, in sync
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
		err = host.EnableNetif(ctx, userCred, netif, "", enables[i].GetIpAddr(), "", "", false, true)
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
		}
	}

	for i := 0; i < len(adds); i += 1 {
		extNic := adds[i].netif
		err = host.addNetif(ctx, userCred, extNic.GetMac(), "", extNic.GetIpAddr(), 0, extNic.GetNicType(), extNic.GetIndex(),
			extNic.IsLinkUp(), int16(extNic.GetMtu()), false, "", "", false, true)
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
	desc := self.SEnabledStatusStandaloneResourceBase.GetShortDesc(ctx)
	info := self.getCloudProviderInfo()
	desc.Update(jsonutils.Marshal(&info))
	return desc
}

func (self *SHost) MarkGuestUnknown(userCred mcclient.TokenCredential) {
	log.Errorln(self.GetGuests())
	for _, guest := range self.GetGuests() {
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

	for rows.Next() {
		var host = SHost{}
		q.Row2Struct(rows, &host)
		host.SetModelManager(manager, &host)
		host.PerformOffline(ctx, userCred, nil, nil)
		host.MarkGuestUnknown(userCred)
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

func (host *SHost) PerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret, err := host.SEnabledStatusStandaloneResourceBase.PerformStatus(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	host.ClearSchedDescCache()
	return ret, nil
}

func (host *SHost) GetSchedtagJointManager() ISchedtagJointManager {
	return HostschedtagManager
}
