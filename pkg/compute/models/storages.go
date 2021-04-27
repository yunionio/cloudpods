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
	"path"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SStorageManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SZoneResourceBaseManager
}

var StorageManager *SStorageManager

func init() {
	StorageManager = &SStorageManager{
		SEnabledStatusInfrasResourceBaseManager: db.NewEnabledStatusInfrasResourceBaseManager(
			SStorage{},
			"storages_tbl",
			"storage",
			"storages",
		),
	}
	StorageManager.SetVirtualObject(StorageManager)
}

type SStorage struct {
	db.SEnabledStatusInfrasResourceBase `"status->default":"offline" "status->update":"domain" "enabled->default":"true"`
	db.SExternalizedResourceBase

	SManagedResourceBase
	SZoneResourceBase `update:""`

	// 容量大小,单位Mb
	Capacity int64 `nullable:"false" list:"user" update:"domain" create:"domain_required"`
	// 实际容量大小，单位Mb
	// we always expect actual capacity great or equal than zero, otherwise something wrong
	ActualCapacityUsed int64 `nullable:"true" list:"user" update:"domain" create:"domain_optional"`
	// 预留容量大小
	Reserved int64 `nullable:"true" default:"0" list:"domain" update:"domain"`
	// 存储类型
	// example: local
	StorageType string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"domain_required"`
	// 介质类型
	// example: ssd
	MediumType string `width:"32" charset:"ascii" nullable:"false" list:"user" update:"domain" create:"domain_required"`
	// 超售比
	Cmtbound float32 `nullable:"true" default:"1" list:"domain" update:"domain"`
	// 存储配置信息
	StorageConf jsonutils.JSONObject `nullable:"true" get:"domain" list:"domain" update:"domain"`

	// 存储缓存Id
	StoragecacheId string `width:"36" charset:"ascii" nullable:"true" list:"domain" get:"domain" update:"domain" create:"domain_optional"`

	// 是否启用
	// Enabled tristate.TriState `nullable:"false" default:"true" list:"user" create:"optional"`
	// 状态
	// Status string `width:"36" charset:"ascii" nullable:"false" default:"offline" update:"domain" list:"user" create:"optional"`

	// indicating whether system disk can be allocated in this storage
	// 是否可以用作系统盘存储
	// example: true
	IsSysDiskStore tristate.TriState `nullable:"false" default:"true" list:"user" create:"optional" update:"domain"`
}

func (manager *SStorageManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{ZoneManager},
		{StoragecacheManager},
	}
}

func (manager *SStorageManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SStorageManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SStorage) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SStorage) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SStorage) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.StorageUpdateInput) (api.StorageUpdateInput, error) {
	var err error
	input.EnabledStatusInfrasResourceBaseUpdateInput, err = self.SEnabledStatusInfrasResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusInfrasResourceBaseUpdateInput)
	if err != nil {
		return input, err
	}
	if self.StorageConf != nil {
		input.StorageConf = self.StorageConf.(*jsonutils.JSONDict)
	} else {
		input.StorageConf = jsonutils.NewDict()
	}
	driver := GetStorageDriver(self.StorageType)
	if driver != nil {
		return driver.ValidateUpdateData(ctx, userCred, input)
	}
	return input, nil
}

func (self *SStorage) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusInfrasResourceBase.PostUpdate(ctx, userCred, query, data)

	if data.Contains("cmtbound") || data.Contains("capacity") {
		hosts := self.GetAttachedHosts()
		for _, host := range hosts {
			if err := host.ClearSchedDescCache(); err != nil {
				log.Errorf("clear host %s sched cache failed %v", host.GetName(), err)
			}
		}
	}

	if update, _ := data.Bool("update_storage_conf"); update {
		self.StartStorageUpdateTask(ctx, userCred)
	}
}

func (self *SStorage) StartStorageUpdateTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "StorageUpdateTask", self, userCred, nil, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SStorage) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SStorage) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	DeleteResourceJointSchedtags(self, ctx, userCred)
	return self.SEnabledStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (manager *SStorageManager) GetStorageTypesByHostType(hostType string) ([]string, error) {
	q := manager.Query("storage_type")
	hosts := HostManager.Query().SubQuery()
	hs := HoststorageManager.Query().SubQuery()
	q = q.Join(hs, sqlchemy.Equals(q.Field("id"), hs.Field("storage_id"))).
		Join(hosts, sqlchemy.Equals(hosts.Field("id"), hs.Field("host_id"))).
		Filter(sqlchemy.Equals(hosts.Field("host_type"), hostType)).Distinct()
	storages := []string{}
	rows, err := q.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var storage string
		err = rows.Scan(&storage)
		if err != nil {
			return nil, errors.Wrap(err, "rows.Scan(&storage)")
		}
		storages = append(storages, storage)
	}
	return storages, nil
}

func (manager *SStorageManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.StorageCreateInput,
) (api.StorageCreateInput, error) {
	if !utils.IsInStringArray(input.StorageType, api.STORAGE_TYPES) {
		return input, httperrors.NewInputParameterError("Invalid storage type %s", input.StorageType)
	}
	if len(input.MediumType) == 0 {
		input.MediumType = api.DISK_TYPE_SSD
	}
	if !utils.IsInStringArray(input.MediumType, api.DISK_TYPES) {
		return input, httperrors.NewInputParameterError("Invalid medium type %s", input.MediumType)
	}
	if len(input.ZoneId) == 0 {
		return input, httperrors.NewMissingParameterError("zone_id")
	}
	_, err := validators.ValidateModel(userCred, ZoneManager, &input.ZoneId)
	if err != nil {
		return input, err
	}
	storageDirver := GetStorageDriver(input.StorageType)
	if storageDirver == nil {
		return input, httperrors.NewUnsupportOperationError("Not support create %s storage", input.StorageType)
	}

	err = storageDirver.ValidateCreateData(ctx, userCred, &input)
	if err != nil {
		return input, errors.Wrap(err, "storageDirver.ValidateCreateData")
	}

	input.EnabledStatusInfrasResourceBaseCreateInput, err = manager.SEnabledStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ValidateCreateData")
	}

	return input, nil
}

func (self *SStorage) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	self.SetEnabled(true)
	self.SetStatus(userCred, api.STORAGE_OFFLINE, "CustomizeCreate")
	return self.SEnabledStatusInfrasResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (self *SStorage) ValidateDeleteCondition(ctx context.Context) error {
	cnt, err := self.GetHostCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetHostCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("storage has associate hosts")
	}
	cnt, err = self.GetDiskCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetDiskCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("storage has disks")
	}
	cnt, err = self.GetSnapshotCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetSnapshotCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("storage has snapshots")
	}
	return self.SEnabledStatusInfrasResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SStorage) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusInfrasResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	storageDriver := GetStorageDriver(self.StorageType)
	if storageDriver != nil {
		storageDriver.PostCreate(ctx, userCred, self, data)
	}
}

func (self *SStorage) SetStatus(userCred mcclient.TokenCredential, status string, reason string) error {
	if self.Status == status {
		return nil
	}
	oldStatus := self.Status
	_, err := db.Update(self, func() error {
		self.Status = status
		return nil
	})
	if err != nil {
		return err
	}
	if userCred != nil {
		notes := fmt.Sprintf("%s=>%s", oldStatus, status)
		if len(reason) > 0 {
			notes = fmt.Sprintf("%s: %s", notes, reason)
		}
		db.OpsLog.LogEvent(self, db.ACT_UPDATE_STATUS, notes, userCred)
		// if strings.Contains(notes, "fail") {
		// 	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_SYNC_STATUS, notes, userCred, false)
		// }
	}
	return nil
}

func (self *SStorage) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "enable")
}

func (self *SStorage) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Enabled.IsFalse() {
		_, err := db.Update(self, func() error {
			self.Enabled = tristate.True
			return nil
		})
		if err != nil {
			log.Errorf("PerformEnable save update fail %s", err)
			return nil, err
		}
		db.OpsLog.LogEvent(self, db.ACT_ENABLE, "", userCred)
		self.ClearSchedDescCache()
	}
	return nil, nil
}

func (self *SStorage) AllowPerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "disable")
}

func (self *SStorage) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Enabled.IsTrue() {
		_, err := db.Update(self, func() error {
			self.Enabled = tristate.False
			return nil
		})
		if err != nil {
			log.Errorf("PerformDisable save update fail %s", err)
			return nil, err
		}
		db.OpsLog.LogEvent(self, db.ACT_DISABLE, "", userCred)
		self.ClearSchedDescCache()
	}
	return nil, nil
}

func (self *SStorage) AllowPerformOnline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "online")
}

func (self *SStorage) PerformOnline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != api.STORAGE_ONLINE {
		err := self.SetStatus(userCred, api.STORAGE_ONLINE, "")
		if err != nil {
			return nil, err
		}
		db.OpsLog.LogEvent(self, db.ACT_ONLINE, "", userCred)
		self.ClearSchedDescCache()
	}
	return nil, nil
}

func (self *SStorage) AllowPerformOffline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "offline")
}

func (self *SStorage) PerformOffline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != api.STORAGE_OFFLINE {
		err := self.SetStatus(userCred, api.STORAGE_OFFLINE, "")
		if err != nil {
			return nil, err
		}
		db.OpsLog.LogEvent(self, db.ACT_OFFLINE, "", userCred)
		self.ClearSchedDescCache()
	}
	return nil, nil
}

func (self *SStorage) GetHostCount() (int, error) {
	return HoststorageManager.Query().Equals("storage_id", self.Id).CountWithError()
}

func (self *SStorage) GetDiskCount() (int, error) {
	return DiskManager.Query().Equals("storage_id", self.Id).CountWithError()
}

func (self *SStorage) GetDisks() []SDisk {
	disks := make([]SDisk, 0)
	q := DiskManager.Query().Equals("storage_id", self.Id)
	err := db.FetchModelObjects(DiskManager, q, &disks)
	if err != nil {
		log.Errorf("GetDisks fail %s", err)
		return nil
	}
	return disks
}

func (self *SStorage) GetSnapshotCount() (int, error) {
	return SnapshotManager.Query().Equals("storage_id", self.Id).CountWithError()
}

func (self *SStorage) IsLocal() bool {
	return utils.IsInStringArray(self.StorageType, api.HOST_STORAGE_LOCAL_TYPES)
}

func (self *SStorage) GetStorageCachePath(mountPoint, imageCachePath string) string {
	if utils.IsInStringArray(self.StorageType, api.SHARED_FILE_STORAGE) {
		return path.Join(mountPoint, imageCachePath)
	} else {
		return imageCachePath
	}
}

func (self *SStorage) getStorageCapacity() SStorageCapacity {
	capa := SStorageCapacity{}

	capa.Capacity = self.GetCapacity()
	capa.Used = self.GetUsedCapacity(tristate.True)
	capa.Wasted = self.GetUsedCapacity(tristate.False)
	capa.VCapacity = int64(float32(self.GetCapacity()) * self.GetOvercommitBound())
	capa.ActualUsed = self.ActualCapacityUsed

	return capa
}

func (self *SStorage) getMoreDetails(ctx context.Context, out api.StorageDetails) api.StorageDetails {
	capa := self.getStorageCapacity()
	out.SStorageCapacityInfo = capa.toCapacityInfo()
	out.CommitBound = self.GetOvercommitBound()
	out.Schedtags = GetSchedtagsDetailsToResourceV2(self, ctx)

	return out
}

func (self *SStorage) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.StorageDetails, error) {
	return api.StorageDetails{}, nil
}

func (manager *SStorageManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.StorageDetails {
	rows := make([]api.StorageDetails, len(objs))
	stdRows := manager.SEnabledStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	zoneRows := manager.SZoneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manageRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	storageIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.StorageDetails{
			EnabledStatusInfrasResourceBaseDetails: stdRows[i],
			ZoneResourceInfo:                       zoneRows[i],
			ManagedResourceInfo:                    manageRows[i],
		}
		storage := objs[i].(*SStorage)
		storageIds[i] = storage.Id
		rows[i] = storage.getMoreDetails(ctx, rows[i])
	}
	q := HoststorageManager.Query("storage_id", "host_id").In("storage_id", storageIds)
	hoststorages, err := q.Rows()
	if err != nil {
		return rows
	}
	defer hoststorages.Close()
	hostIds := []string{}
	storageMap := make(map[string][]string, len(objs))
	for hoststorages.Next() {
		var storageId, hostId string
		hoststorages.Scan(&storageId, &hostId)
		if _, ok := storageMap[storageId]; !ok {
			storageMap[storageId] = []string{}
		}
		storageMap[storageId] = append(storageMap[storageId], hostId)
		hostIds = append(hostIds, hostId)
	}
	hostMap, err := db.FetchIdNameMap2(HostManager, hostIds)
	if err != nil {
		return rows
	}
	for i := range rows {
		hostIds, ok := storageMap[storageIds[i]]
		if ok {
			rows[i].Hosts = []api.StorageHost{}
			for _, hostId := range hostIds {
				if name, ok := hostMap[hostId]; ok {
					rows[i].Hosts = append(rows[i].Hosts, api.StorageHost{Id: hostId, Name: name})
				}
			}
		}
	}
	return rows
}

func (self *SStorage) GetUsedCapacity(isReady tristate.TriState) int64 {
	disks := DiskManager.Query().SubQuery()
	q := disks.Query(sqlchemy.SUM("sum", disks.Field("disk_size"))).Equals("storage_id", self.Id)
	switch isReady {
	case tristate.True:
		q = q.Equals("status", api.DISK_READY)
	case tristate.False:
		q = q.NotEquals("status", api.DISK_READY)
	}
	row := q.Row()

	// sum can be null, deal with null:
	// https://github.com/golang/go/wiki/SQLInterface#dealing-with-null
	var sum sql.NullInt64
	err := row.Scan(&sum)
	if err != nil {
		log.Errorf("GetUsedCapacity fail: %s", err)
		return 0
	}
	if sum.Valid {
		return sum.Int64
	} else {
		return 0
	}
}

func (self *SStorage) GetOvercommitBound() float32 {
	if self.Cmtbound > 0 {
		return self.Cmtbound
	} else {
		return options.Options.DefaultStorageOvercommitBound
	}
}

func (self *SStorage) GetMasterHost() *SHost {
	hosts := HostManager.Query().SubQuery()
	hoststorages := HoststorageManager.Query().SubQuery()

	q := hosts.Query().Join(hoststorages, sqlchemy.Equals(hoststorages.Field("host_id"), hosts.Field("id")))
	q = q.Filter(sqlchemy.Equals(hoststorages.Field("storage_id"), self.Id))
	q = q.IsTrue("enabled")
	q = q.Equals("host_status", api.HOST_ONLINE).Asc("id")
	host := SHost{}
	host.SetModelManager(HostManager, &host)
	err := q.First(&host)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("GetMasterHost fail %s", err)
		}
		return nil
	}
	return &host
}

func (self *SStorage) GetZoneId() string {
	if len(self.ZoneId) > 0 {
		return self.ZoneId
	}
	host := self.GetMasterHost()
	if host != nil {
		_, err := db.Update(self, func() error {
			self.ZoneId = host.ZoneId
			return nil
		})
		if err != nil {
			log.Errorf("%s", err)
			return ""
		}
		return self.ZoneId
	} else {
		log.Errorf("No mater host for storage")
		return ""
	}
}

func (self *SStorage) getZone() *SZone {
	zoneId := self.GetZoneId()
	if len(zoneId) > 0 {
		return ZoneManager.FetchZoneById(zoneId)
	}
	return nil
}

func (self *SStorage) GetRegion() *SCloudregion {
	zone := self.getZone()
	if zone == nil {
		return nil
	}
	return zone.GetRegion()
}

func (self *SStorage) GetReserved() int64 {
	return self.Reserved
}

func (self *SStorage) GetCapacity() int64 {
	return self.Capacity - self.GetReserved()
}

func (self *SStorage) GetFreeCapacity() int64 {
	return int64(float32(self.GetCapacity())*self.GetOvercommitBound()) - self.GetUsedCapacity(tristate.None)
}

func (self *SStorage) GetAttachedHosts() []SHost {
	hosts := HostManager.Query().SubQuery()
	hoststorages := HoststorageManager.Query().SubQuery()

	q := hosts.Query()
	q = q.Join(hoststorages, sqlchemy.Equals(hoststorages.Field("host_id"), hosts.Field("id")))
	q = q.Filter(sqlchemy.Equals(hoststorages.Field("storage_id"), self.Id))

	hostList := make([]SHost, 0)
	err := db.FetchModelObjects(HostManager, q, &hostList)
	if err != nil {
		log.Errorf("GetAttachedHosts fail %s", err)
		return nil
	}
	return hostList
}

func (self *SStorage) SyncStatusWithHosts() {
	hosts := self.GetAttachedHosts()
	if hosts == nil {
		return
	}
	total := 0
	online := 0
	offline := 0
	for _, h := range hosts {
		if h.HostStatus == api.HOST_ONLINE {
			online += 1
		} else {
			offline += 1
		}
		total += 1
	}
	var status string
	if !self.IsLocal() {
		status = self.Status
		if online == 0 {
			status = api.STORAGE_OFFLINE
		} else {
			status = api.STORAGE_ONLINE
		}
	} else if online > 0 {
		status = api.STORAGE_ONLINE
	} else if offline > 0 {
		status = api.STORAGE_OFFLINE
	} else {
		status = api.STORAGE_OFFLINE
	}
	if status != self.Status {
		self.SetStatus(nil, status, "SyncStatusWithHosts")
	}
}

func (manager *SStorageManager) getStoragesByZoneId(zoneId string, provider *SCloudprovider) ([]SStorage, error) {
	storages := make([]SStorage, 0)
	q := manager.Query().Equals("zone_id", zoneId)
	if provider != nil {
		q = q.Equals("manager_id", provider.Id)
	}
	err := db.FetchModelObjects(manager, q, &storages)
	if err != nil {
		log.Errorf("getStoragesByZoneId fail %s", err)
		return nil, err
	}
	return storages, nil
}

func (manager *SStorageManager) scanLegacyStorages() error {
	storages := make([]SStorage, 0)
	table := manager.Query().SubQuery()
	q := table.Query().Filter(sqlchemy.OR(sqlchemy.IsNull(table.Field("zone_id")), sqlchemy.IsEmpty(table.Field("zone_id"))))
	err := db.FetchModelObjects(manager, q, &storages)
	if err != nil {
		log.Errorf("getLegacyStoragesByZoneId fail %s", err)
		return err
	}
	for i := 0; i < len(storages); i += 1 {
		storages[i].GetZoneId()
	}
	return nil
}

func (manager *SStorageManager) SyncStorages(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, zone *SZone, storages []cloudprovider.ICloudStorage) ([]SStorage, []cloudprovider.ICloudStorage, compare.SyncResult) {
	lockman.LockRawObject(ctx, "storages", fmt.Sprintf("%s-%s", provider.Id, zone.Id))
	defer lockman.ReleaseRawObject(ctx, "storages", fmt.Sprintf("%s-%s", provider.Id, zone.Id))

	localStorages := make([]SStorage, 0)
	remoteStorages := make([]cloudprovider.ICloudStorage, 0)
	syncResult := compare.SyncResult{}

	err := manager.scanLegacyStorages()
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	dbStorages, err := manager.getStoragesByZoneId(zone.Id, provider)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := make([]SStorage, 0)
	commondb := make([]SStorage, 0)
	commonext := make([]cloudprovider.ICloudStorage, 0)
	added := make([]cloudprovider.ICloudStorage, 0)

	err = compare.CompareSets(dbStorages, storages, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		// may be a fake storage for prepaid recycle host
		if removed[i].IsPrepaidRecycleResource() {
			continue
		}
		err = removed[i].syncRemoveCloudStorage(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].syncWithCloudStorage(ctx, userCred, commonext[i], provider)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			localStorages = append(localStorages, commondb[i])
			remoteStorages = append(remoteStorages, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		new, err := manager.newFromCloudStorage(ctx, userCred, added[i], provider, zone)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, new, added[i])
			localStorages = append(localStorages, *new)
			remoteStorages = append(remoteStorages, added[i])
			syncResult.Add()
		}
	}

	return localStorages, remoteStorages, syncResult
}

func (self *SStorage) syncRemoveCloudStorage(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		err = self.SetStatus(userCred, api.STORAGE_OFFLINE, "sync to delete")
		if err == nil {
			_, err = self.PerformDisable(ctx, userCred, nil, nil)
		}
	} else {
		err = self.Delete(ctx, userCred)
	}
	return err
}

var CapacityUsedCloudStorageProvider = []string{
	api.CLOUD_PROVIDER_VMWARE,
}

func (sm *SStorageManager) SyncCapacityUsedForEsxiStorage(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	cpQ := CloudproviderManager.Query("id").Equals("provider", api.CLOUD_PROVIDER_VMWARE)
	cloudproviders := make([]SCloudprovider, 0)
	err := db.FetchModelObjects(CloudproviderManager, cpQ, &cloudproviders)
	if err != nil {
		log.Errorf("unable to FetchModelObjects: %v", err)
	}
	for i := range cloudproviders {
		cp := cloudproviders[i]
		icp, err := cp.GetProvider()
		if err != nil {
			log.Errorf("unable to GetProvider: %v", err)
			continue
		}
		iregion, err := icp.GetOnPremiseIRegion()
		if err != nil {
			log.Errorf("unable to GetOnPremiseIRegion: %v", err)
			continue
		}
		css, err := iregion.GetIStorages()
		if err != nil {
			log.Errorf("unable to GetIStorages: %v", err)
			continue
		}
		storageSizeMap := make(map[string]int64, len(css))
		for i := range css {
			id := css[i].GetGlobalId()
			size := css[i].GetCapacityUsedMB()
			storageSizeMap[id] = size
		}
		sQ := sm.Query().Equals("manager_id", cp.GetId())
		storages := make([]SStorage, 0, 5)
		err = db.FetchModelObjects(sm, sQ, &storages)
		if err != nil {
			log.Errorf("unable to fetch storages with sql %q: %v", sQ.String(), err)
			continue
		}
		for i := range storages {
			s := &storages[i]
			newSize, ok := storageSizeMap[s.GetExternalId()]
			if !ok {
				log.Warningf("can't find usedSize for storage %q", s.GetId())
				continue
			}
			_, err = db.Update(s, func() error {
				s.ActualCapacityUsed = newSize
				return nil
			})
			if err != nil {
				log.Errorf("unable to udpate storage %q: %v", s.GetId(), err)
			}
		}
	}
}

func (sm *SStorageManager) SyncCapacityUsedForStorage(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	cpSubQ := CloudproviderManager.Query("id").In("provider", CapacityUsedCloudStorageProvider).SubQuery()
	sQ := sm.Query()
	sQ = sQ.Join(cpSubQ, sqlchemy.Equals(sQ.Field("manager_id"), cpSubQ.Field("id")))
	storages := make([]SStorage, 0, 5)
	err := db.FetchModelObjects(sm, sQ, &storages)
	if err != nil {
		log.Errorf("unable to fetch storages with sql %q: %v", sQ.String(), err)
	}
	for i := range storages {
		err := storages[i].SyncCapacityUsed(ctx)
		if err != nil {
			log.Errorf("unable to sync CapacityUsed for storage %q: %v", storages[i].Id, err)
		}
	}
}

func (s *SStorage) SyncCapacityUsed(ctx context.Context) error {
	cp := s.GetCloudprovider()
	if cp == nil {
		return errors.Wrapf(errors.ErrNotFound, "no cloudprovider for storage %s", s.Id)
	}
	if !utils.IsInStringArray(cp.Provider, CapacityUsedCloudStorageProvider) {
		return nil
	}
	icp, err := cp.GetProvider()
	if err != nil {
		return errors.Wrap(err, "GetProvider")
	}
	iregion, err := icp.GetOnPremiseIRegion()
	if err != nil {
		return errors.Wrap(err, "GetOnPremiseIRegion")
	}
	cloudStorage, err := iregion.GetIStorageById(s.ExternalId)
	if err != nil {
		return errors.Wrap(err, "GetIStorageById")
	}
	capacityUsed := cloudStorage.GetCapacityUsedMB()
	if s.ActualCapacityUsed == capacityUsed {
		return nil
	}
	_, err = db.UpdateWithLock(ctx, s, func() error {
		s.ActualCapacityUsed = capacityUsed
		return nil
	})
	return err
}

func (self *SStorage) syncWithCloudStorage(ctx context.Context, userCred mcclient.TokenCredential, extStorage cloudprovider.ICloudStorage, provider *SCloudprovider) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		// self.Name = extStorage.GetName()
		self.Status = extStorage.GetStatus()
		self.StorageType = extStorage.GetStorageType()
		self.MediumType = extStorage.GetMediumType()
		if capacity := extStorage.GetCapacityMB(); capacity != 0 {
			self.Capacity = capacity
		}
		if capacity := extStorage.GetCapacityUsedMB(); capacity != 0 {
			self.ActualCapacityUsed = capacity
		}
		self.StorageConf = extStorage.GetStorageConf()

		self.Enabled = tristate.NewFromBool(extStorage.GetEnabled())

		self.IsEmulated = extStorage.IsEmulated()

		self.IsSysDiskStore = tristate.NewFromBool(extStorage.IsSysDiskStore())

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudZone error %s", err)
		return err
	}

	if provider != nil {
		SyncCloudDomain(userCred, self, provider.GetOwnerId())
		self.SyncShareState(ctx, userCred, provider.getAccountShareInfo())
	}

	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SStorageManager) newFromCloudStorage(ctx context.Context, userCred mcclient.TokenCredential, extStorage cloudprovider.ICloudStorage, provider *SCloudprovider, zone *SZone) (*SStorage, error) {
	storage := SStorage{}
	storage.SetModelManager(manager, &storage)

	storage.Status = extStorage.GetStatus()
	storage.ExternalId = extStorage.GetGlobalId()
	storage.ZoneId = zone.Id
	storage.StorageType = extStorage.GetStorageType()
	storage.MediumType = extStorage.GetMediumType()
	storage.StorageConf = extStorage.GetStorageConf()
	storage.Capacity = extStorage.GetCapacityMB()
	storage.ActualCapacityUsed = extStorage.GetCapacityUsedMB()
	storage.Cmtbound = 1.0

	storage.Enabled = tristate.NewFromBool(extStorage.GetEnabled())

	storage.IsEmulated = extStorage.IsEmulated()
	storage.ManagerId = provider.Id

	storage.IsSysDiskStore = tristate.NewFromBool(extStorage.IsSysDiskStore())

	var err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, manager, userCred, extStorage.GetName())
		if err != nil {
			return err
		}
		storage.Name = newName

		return manager.TableSpec().Insert(ctx, &storage)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	SyncCloudDomain(userCred, &storage, provider.GetOwnerId())

	if provider != nil {
		storage.SyncShareState(ctx, userCred, provider.getAccountShareInfo())
	}

	db.OpsLog.LogEvent(&storage, db.ACT_CREATE, storage.GetShortDesc(ctx), userCred)

	return &storage, nil
}

type StorageCapacityStat struct {
	TotalSize        int64
	TotalSizeVirtual float64
}

func filterDisksByScope(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, pendingDeleted bool, includeSystem bool) *sqlchemy.SSubQuery {
	q := DiskManager.Query()
	switch scope {
	case rbacutils.ScopeSystem:
	case rbacutils.ScopeDomain:
		q = q.Filter(sqlchemy.Equals(q.Field("domain_id"), ownerId.GetProjectDomainId()))
	case rbacutils.ScopeProject:
		q = q.Filter(sqlchemy.Equals(q.Field("tenant_id"), ownerId.GetProjectId()))
	}
	if pendingDeleted {
		q = q.IsTrue("pending_deleted")
	} else {
		q = q.IsFalse("pending_deleted")
	}
	if !includeSystem {
		q = q.IsFalse("is_system")
	}
	return q.SubQuery()
}

func (manager *SStorageManager) disksReadyQ(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, pendingDeleted bool, includeSystem bool) *sqlchemy.SSubQuery {
	disks := filterDisksByScope(scope, ownerId, pendingDeleted, includeSystem)
	q := disks.Query(
		disks.Field("storage_id"),
		sqlchemy.SUM("used_capacity", disks.Field("disk_size")),
		sqlchemy.COUNT("used_count"),
	).Equals("status", api.DISK_READY)
	q = q.GroupBy(disks.Field("storage_id"))
	return q.SubQuery()
}

func (manager *SStorageManager) diskIsAttachedQ(isAttached bool, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, pendingDeleted bool, includeSystem bool) *sqlchemy.SSubQuery {
	sumKey := "attached_used_capacity"
	countKey := "attached_count"
	cond := sqlchemy.In
	if !isAttached {
		sumKey = "detached_used_capacity"
		countKey = "detached_count"
		cond = sqlchemy.NotIn
	}
	sq := GuestdiskManager.Query("disk_id").SubQuery()
	disks := filterDisksByScope(scope, ownerId, pendingDeleted, includeSystem)
	disks = disks.Query().Filter(cond(disks.Field("id"), sq)).SubQuery()
	q := disks.Query(
		disks.Field("storage_id"),
		sqlchemy.SUM(sumKey, disks.Field("disk_size")),
		sqlchemy.COUNT(countKey),
	).Equals("status", api.DISK_READY).GroupBy(disks.Field("storage_id"))
	return q.SubQuery()
}

func (manager *SStorageManager) diskAttachedQ(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, pendingDeleted bool, includeSystem bool) *sqlchemy.SSubQuery {
	return manager.diskIsAttachedQ(true, scope, ownerId, pendingDeleted, includeSystem)
}

func (manager *SStorageManager) diskDetachedQ(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, pendingDeleted bool, includeSystem bool) *sqlchemy.SSubQuery {
	return manager.diskIsAttachedQ(false, scope, ownerId, pendingDeleted, includeSystem)
}

func (manager *SStorageManager) disksFailedQ(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, pendingDeleted bool, includeSystem bool) *sqlchemy.SSubQuery {
	disks := filterDisksByScope(scope, ownerId, pendingDeleted, includeSystem)
	q := disks.Query(
		disks.Field("storage_id"),
		sqlchemy.SUM("failed_capacity", disks.Field("disk_size")),
		sqlchemy.COUNT("failed_count"),
	).NotEquals("status", api.DISK_READY)
	q = q.GroupBy(disks.Field("storage_id"))
	return q.SubQuery()
}

func (manager *SStorageManager) totalCapacityQ(
	rangeObjs []db.IStandaloneModel, hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider,
	pendingDeleted bool, includeSystem bool,
	storageOwnership bool,
) *sqlchemy.SQuery {
	stmt := manager.disksReadyQ(scope, ownerId, pendingDeleted, includeSystem)
	stmt2 := manager.disksFailedQ(scope, ownerId, pendingDeleted, includeSystem)
	attachedDisks := manager.diskAttachedQ(scope, ownerId, pendingDeleted, includeSystem)
	detachedDisks := manager.diskDetachedQ(scope, ownerId, pendingDeleted, includeSystem)
	storages := manager.Query().SubQuery()
	q := storages.Query(
		storages.Field("capacity"),
		storages.Field("reserved"),
		storages.Field("cmtbound"),
		stmt.Field("used_capacity"),
		stmt.Field("used_count"),
		stmt2.Field("failed_capacity"),
		stmt2.Field("failed_count"),
		attachedDisks.Field("attached_used_capacity"),
		attachedDisks.Field("attached_count"),
		detachedDisks.Field("detached_used_capacity"),
		detachedDisks.Field("detached_count"),
	)
	q = q.LeftJoin(stmt, sqlchemy.Equals(stmt.Field("storage_id"), storages.Field("id")))
	q = q.LeftJoin(stmt2, sqlchemy.Equals(stmt2.Field("storage_id"), storages.Field("id")))
	q = q.LeftJoin(attachedDisks, sqlchemy.Equals(attachedDisks.Field("storage_id"), storages.Field("id")))
	q = q.LeftJoin(detachedDisks, sqlchemy.Equals(detachedDisks.Field("storage_id"), storages.Field("id")))

	if len(hostTypes) > 0 || len(resourceTypes) > 0 || len(rangeObjs) > 0 {
		hosts := HostManager.Query().SubQuery()
		subq := HoststorageManager.Query("storage_id")
		subq = subq.Join(hosts, sqlchemy.Equals(hosts.Field("id"), subq.Field("host_id")))
		subq = subq.Filter(sqlchemy.IsTrue(hosts.Field("enabled")))
		subq = subq.Filter(sqlchemy.Equals(hosts.Field("host_status"), api.HOST_ONLINE))
		subq = AttachUsageQuery(subq, hosts, hostTypes, resourceTypes, nil, nil, "", rangeObjs)

		q = q.Filter(sqlchemy.In(storages.Field("id"), subq.SubQuery()))
	}

	if len(rangeObjs) > 0 || len(providers) > 0 || len(brands) > 0 || cloudEnv != "" {
		q = CloudProviderFilter(q, storages.Field("manager_id"), providers, brands, cloudEnv)
		q = RangeObjectsFilter(q, rangeObjs, nil, storages.Field("zone_id"), storages.Field("manager_id"), nil, storages.Field("id"))
	}

	if storageOwnership {
		switch scope {
		case rbacutils.ScopeSystem:
		case rbacutils.ScopeDomain, rbacutils.ScopeProject:
			q = q.Equals("domain_id", ownerId.GetProjectDomainId())
		}
	}

	return q
}

type StorageStat struct {
	Capacity             int
	Reserved             int
	Cmtbound             float32
	UsedCapacity         int
	UsedCount            int
	FailedCapacity       int
	FailedCount          int
	AttachedUsedCapacity int
	AttachedCount        int
	DetachedUsedCapacity int
	DetachedCount        int
}

type StoragesCapacityStat struct {
	Capacity         int64
	CapacityVirtual  float64
	CapacityUsed     int64
	CountUsed        int
	CapacityUnready  int64
	CountUnready     int
	AttachedCapacity int64
	CountAttached    int
	DetachedCapacity int64
	CountDetached    int
}

func (manager *SStorageManager) calculateCapacity(q *sqlchemy.SQuery) StoragesCapacityStat {
	stats := make([]StorageStat, 0)
	err := q.All(&stats)
	if err != nil {
		log.Errorf("calculateCapacity: %v", err)
	}
	var (
		tCapa   int64   = 0
		tVCapa  float64 = 0
		tUsed   int64   = 0
		cUsed   int     = 0
		tFailed int64   = 0
		cFailed int     = 0
		atCapa  int64   = 0
		atCount int     = 0
		dtCapa  int64   = 0
		dtCount int     = 0
	)
	for _, stat := range stats {
		tCapa += int64(stat.Capacity - stat.Reserved)
		if stat.Cmtbound == 0 {
			stat.Cmtbound = options.Options.DefaultStorageOvercommitBound
		}
		tVCapa += float64(stat.Capacity-stat.Reserved) * float64(stat.Cmtbound)
		tUsed += int64(stat.UsedCapacity)
		cUsed += stat.UsedCount
		tFailed += int64(stat.FailedCapacity)
		cFailed += stat.FailedCount
		atCapa += int64(stat.AttachedUsedCapacity)
		atCount += stat.AttachedCount
		dtCapa += int64(stat.DetachedUsedCapacity)
		dtCount += stat.DetachedCount
	}
	return StoragesCapacityStat{
		Capacity:         tCapa,
		CapacityVirtual:  tVCapa,
		CapacityUsed:     tUsed,
		CountUsed:        cUsed,
		CapacityUnready:  tFailed,
		CountUnready:     cFailed,
		AttachedCapacity: atCapa,
		CountAttached:    atCount,
		DetachedCapacity: dtCapa,
		CountDetached:    dtCount,
	}
}

func (manager *SStorageManager) TotalCapacity(
	rangeObjs []db.IStandaloneModel,
	hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	scope rbacutils.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	pendingDeleted bool, includeSystem bool,
	storageOwnership bool,
) StoragesCapacityStat {
	res1 := manager.calculateCapacity(
		manager.totalCapacityQ(
			rangeObjs,
			hostTypes,
			resourceTypes,
			providers, brands, cloudEnv,
			scope, ownerId,
			pendingDeleted, includeSystem,
			storageOwnership,
		),
	)
	return res1
}

func (self *SStorage) createDisk(ctx context.Context, name string, diskConfig *api.DiskConfig, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, autoDelete bool, isSystem bool,
	billingType string, billingCycle string,
) (*SDisk, error) {
	disk := SDisk{}
	disk.SetModelManager(DiskManager, &disk)

	disk.Name = name
	disk.fetchDiskInfo(diskConfig)

	disk.StorageId = self.Id
	disk.AutoDelete = autoDelete
	disk.ProjectId = ownerId.GetProjectId()
	disk.ProjectSrc = string(apis.OWNER_SOURCE_LOCAL)
	disk.DomainId = ownerId.GetProjectDomainId()
	disk.IsSystem = isSystem

	disk.BillingType = billingType
	disk.BillingCycle = billingCycle

	err := disk.GetModelManager().TableSpec().Insert(ctx, &disk)
	if err != nil {
		return nil, err
	}
	db.OpsLog.LogEvent(&disk, db.ACT_CREATE, nil, userCred)
	return &disk, nil
}

func (self *SStorage) GetAllAttachingHosts() []SHost {
	hosts := HostManager.Query().SubQuery()
	hoststorages := HoststorageManager.Query().SubQuery()

	q := hosts.Query()
	q = q.Join(hoststorages, sqlchemy.Equals(hoststorages.Field("host_id"), hosts.Field("id")))
	q = q.Filter(sqlchemy.Equals(hoststorages.Field("storage_id"), self.Id))
	q = q.Filter(sqlchemy.IsTrue(hosts.Field("enabled")))
	q = q.Filter(sqlchemy.Equals(hosts.Field("host_status"), api.HOST_ONLINE))

	ret := make([]SHost, 0)
	err := db.FetchModelObjects(HostManager, q, &ret)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return ret
}

func (self *SStorage) SetStoragecache(userCred mcclient.TokenCredential, cache *SStoragecache) error {
	if self.StoragecacheId == cache.Id {
		return nil
	}
	diff, err := db.Update(self, func() error {
		self.StoragecacheId = cache.Id
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)
	return nil
}

func (self *SStorage) AllowPerformCacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "cache-image")
}

func (self *SStorage) GetStoragecache() *SStoragecache {
	obj, err := StoragecacheManager.FetchById(self.StoragecacheId)
	if err != nil {
		log.Errorf("cannot find storage cache??? %s", err)
		return nil
	}
	return obj.(*SStoragecache)
}

func (self *SStorage) PerformCacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	cache := self.GetStoragecache()
	if cache == nil {
		return nil, httperrors.NewInternalServerError("storage cache is missing")
	}

	return cache.PerformCacheImage(ctx, userCred, query, data)
}

func (self *SStorage) AllowPerformUncacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "uncache-image")
}

func (self *SStorage) PerformUncacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	cache := self.GetStoragecache()
	if cache == nil {
		return nil, httperrors.NewInternalServerError("storage cache is missing")
	}

	return cache.PerformUncacheImage(ctx, userCred, query, data)
}

func (self *SStorage) GetIStorage() (cloudprovider.ICloudStorage, error) {
	provider, err := self.GetDriver()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetDriver")
	}
	var iRegion cloudprovider.ICloudRegion
	if provider.GetFactory().IsOnPremise() {
		iRegion, err = provider.GetOnPremiseIRegion()
	} else {
		region := self.GetRegion()
		if region == nil {
			msg := "cannot find region for storage???"
			log.Errorf(msg)
			return nil, fmt.Errorf(msg)
		}
		iRegion, err = provider.GetIRegionById(region.ExternalId)
	}
	if err != nil {
		return nil, errors.Wrap(err, "provider.GetIRegionById")
	}
	istore, err := iRegion.GetIStorageById(self.GetExternalId())
	if err != nil {
		log.Errorf("iRegion.GetIStorageById fail %s", err)
		return nil, errors.Wrapf(err, "iRegion.GetIStorageById(%s)", self.GetExternalId())
	}
	return istore, nil
}

func (manager *SStorageManager) FetchStorageById(storageId string) *SStorage {
	obj, err := manager.FetchById(storageId)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return obj.(*SStorage)
}

func (manager *SStorageManager) InitializeData() error {
	storages := make([]SStorage, 0)
	q := manager.Query()
	err := db.FetchModelObjects(manager, q, &storages)
	if err != nil {
		return err
	}
	for _, s := range storages {
		if len(s.ZoneId) == 0 {
			zoneId := ""
			hosts := s.GetAttachedHosts()
			if hosts != nil && len(hosts) > 0 {
				zoneId = hosts[0].ZoneId
			} else {
				log.Fatalf("Cannot locate zoneId for storage %s", s.Name)
			}
			db.Update(&s, func() error {
				s.ZoneId = zoneId
				return nil
			})
		}
		if len(s.StoragecacheId) == 0 && s.StorageType == api.STORAGE_RBD {
			storagecache := &SStoragecache{}
			storagecache.SetModelManager(StoragecacheManager, storagecache)
			storagecache.Name = "rbd-" + s.Id
			if pool, err := s.StorageConf.GetString("pool"); err != nil {
				log.Fatalf("Get storage %s pool info error", s.Name)
			} else {
				storagecache.Path = "rbd:" + pool
				if err := StoragecacheManager.TableSpec().Insert(context.TODO(), storagecache); err != nil {
					log.Fatalf("Cannot Add storagecache for %s", s.Name)
				} else {
					db.Update(&s, func() error {
						s.StoragecacheId = storagecache.Id
						return nil
					})
				}
			}
		}
	}
	return nil
}

func (manager *SStorageManager) IsStorageTypeExist(storageType string) (string, bool) {
	storages := []SStorage{}
	q := manager.Query().Equals("storage_type", storageType)
	if err := db.FetchModelObjects(manager, q, &storages); err != nil {
		return "", false
	}
	if len(storages) == 0 {
		return "", false
	}
	return storages[0].StorageType, true
}

// 块存储列表
func (manager *SStorageManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.StorageListInput,
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

	q, err = manager.SEnabledStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemFilter")
	}

	if query.Share != nil && *query.Share {
		q = q.Filter(sqlchemy.NotIn(q.Field("storage_type"), api.STORAGE_LOCAL_TYPES))
	}

	if query.Local != nil && *query.Local {
		q = q.Filter(sqlchemy.In(q.Field("storage_type"), api.STORAGE_LOCAL_TYPES))
	}

	if len(query.SchedtagId) > 0 {
		schedTag, err := SchedtagManager.FetchByIdOrName(nil, query.SchedtagId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(SchedtagManager.Keyword(), query.SchedtagId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		sq := StorageschedtagManager.Query("storage_id").Equals("schedtag_id", schedTag.GetId()).SubQuery()
		q = q.In("id", sq)
	}

	if query.Usable != nil && *query.Usable {
		hostStorageTable := HoststorageManager.Query().SubQuery()
		hostTable := HostManager.Query().SubQuery()
		sq1 := hostStorageTable.Query(hostStorageTable.Field("storage_id")).
			Join(hostTable, sqlchemy.Equals(hostTable.Field("id"), hostStorageTable.Field("host_id"))).
			Filter(sqlchemy.Equals(hostTable.Field("host_status"), api.HOST_ONLINE)).
			Filter(sqlchemy.IsTrue(hostTable.Field("enabled"))).
			Filter(sqlchemy.IsNullOrEmpty(hostTable.Field("manager_id")))

		providerTable := CloudproviderManager.Query().SubQuery()
		sq2 := hostStorageTable.Query(hostStorageTable.Field("storage_id")).
			Join(hostTable, sqlchemy.Equals(hostTable.Field("id"), hostStorageTable.Field("host_id"))).
			Join(providerTable, sqlchemy.Equals(hostTable.Field("manager_id"), providerTable.Field("id"))).
			Filter(sqlchemy.IsTrue(providerTable.Field("enabled"))).
			Filter(sqlchemy.In(providerTable.Field("status"), api.CLOUD_PROVIDER_VALID_STATUS)).
			Filter(sqlchemy.In(providerTable.Field("health_status"), api.CLOUD_PROVIDER_VALID_HEALTH_STATUS))

		q = q.Filter(
			sqlchemy.OR(
				sqlchemy.In(q.Field("id"), sq1),
				sqlchemy.In(q.Field("id"), sq2),
			)).
			Filter(sqlchemy.In(q.Field("status"), []string{api.STORAGE_ENABLED, api.STORAGE_ONLINE})).
			Filter(sqlchemy.IsTrue(q.Field("enabled")))
	}

	if len(query.HostSchedtagId) > 0 {
		schedTagObj, err := SchedtagManager.FetchByIdOrName(userCred, query.HostSchedtagId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", SchedtagManager.Keyword(), query.HostSchedtagId)
			} else {
				return nil, errors.Wrap(err, "SchedtagManager.FetchByIdOrName")
			}
		}
		subq := HoststorageManager.Query("storage_id")
		hostschedtags := HostschedtagManager.Query().Equals("schedtag_id", schedTagObj.GetId()).SubQuery()
		subq = subq.Join(hostschedtags, sqlchemy.Equals(hostschedtags.Field("host_id"), subq.Field("host_id")))
		q = q.In("id", subq.SubQuery())
	}

	if len(query.ImageId) > 0 {
		image, err := CachedimageManager.getImageInfo(ctx, userCred, query.ImageId, false)
		if err != nil {
			return nil, errors.Wrap(err, "CachedimageManager.getImageInfo")
		}
		subq := StorageManager.Query("id")
		storagecaches := StoragecachedimageManager.Query("storagecache_id").Equals("cachedimage_id", image.Id).SubQuery()
		subq = subq.Join(storagecaches, sqlchemy.Equals(subq.Field("storagecache_id"), storagecaches.Field("storagecache_id")))
		q = q.In("id", subq.SubQuery())
	}

	return q, err
}

func (manager *SStorageManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.StorageListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SZoneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ZonalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SStorageManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SZoneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SStorage) ClearSchedDescCache() error {
	hosts := self.GetAllAttachingHosts()
	if hosts == nil {
		msg := "get attaching host error"
		log.Errorf(msg)
		return fmt.Errorf(msg)
	}
	for i := 0; i < len(hosts); i += 1 {
		err := hosts[i].ClearSchedDescCache()
		if err != nil {
			log.Errorf("host CleanHostSchedCache error: %v", err)
			return err
		}
	}
	return nil
}

func (self *SStorage) getCloudProviderInfo() SCloudProviderInfo {
	var region *SCloudregion
	zone := self.getZone()
	if zone != nil {
		region = zone.GetRegion()
	}
	provider := self.GetCloudprovider()
	return MakeCloudProviderInfo(region, zone, provider)
}

func (self *SStorage) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := self.SStandaloneResourceBase.GetShortDesc(ctx)
	info := self.getCloudProviderInfo()
	desc.Update(jsonutils.Marshal(&info))
	return desc
}

func (self *SStorage) IsPrepaidRecycleResource() bool {
	if !self.IsLocal() {
		return false
	}
	hosts := self.GetAttachedHosts()
	if len(hosts) != 1 {
		return false
	}
	return hosts[0].IsPrepaidRecycleResource()
}

func (self *SStorage) GetSchedtags() []SSchedtag {
	return GetSchedtags(StorageschedtagManager, self.Id)
}

func (self *SStorage) GetDynamicConditionInput() *jsonutils.JSONDict {
	return jsonutils.Marshal(self).(*jsonutils.JSONDict)
}

func (self *SStorage) AllowPerformSetSchedtag(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return AllowPerformSetResourceSchedtag(self, ctx, userCred, query, data)
}

func (self *SStorage) PerformSetSchedtag(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return PerformSetResourceSchedtag(self, ctx, userCred, query, data)
}

func (self *SStorage) GetSchedtagJointManager() ISchedtagJointManager {
	return StorageschedtagManager
}

func (manager *SStorageManager) StorageSnapshotsRecycle(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	storages := make([]SStorage, 0)
	err := manager.Query().Equals("enabled", true).
		In("status", []string{api.STORAGE_ENABLED, api.STORAGE_ONLINE}).
		In("storage_type", api.SHARED_FILE_STORAGE).All(&storages)
	if err != nil {
		log.Errorf("Get shared file storage failed %s", err)
		return
	}
	for i := 0; i < len(storages); i++ {
		storages[i].SetModelManager(manager, &storages[i])
		host := storages[i].GetMasterHost()
		url := fmt.Sprintf("%s/storages/%s/snapshots-recycle", host.ManagerUri, storages[i].Id)
		headers := mcclient.GetTokenHeaders(userCred)
		_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, headers, nil, false)
		if err != nil {
			log.Errorf("Storage request snapshots recycle failed %s", err)
		}
	}
}

func (self *SStorage) StartDeleteRbdDisks(ctx context.Context, userCred mcclient.TokenCredential, disksId []string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewStringArray(disksId), "disks_id")
	task, err := taskman.TaskManager.NewTask(ctx, "StorageDeleteRbdDiskTask", self, userCred, data, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (storage *SStorage) PerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformChangeDomainOwnerInput) (jsonutils.JSONObject, error) {
	// not allow to perform public for locally connected storage
	if storage.IsLocal() {
		hosts := storage.GetAttachedHosts()
		if len(hosts) > 0 {
			return nil, errors.Wrap(httperrors.ErrForbidden, "not allow to change owner for local storage")
		}
	}
	return storage.performChangeOwnerInternal(ctx, userCred, query, input)
}

func (storage *SStorage) performChangeOwnerInternal(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformChangeDomainOwnerInput) (jsonutils.JSONObject, error) {
	return storage.SEnabledStatusInfrasResourceBase.PerformChangeOwner(ctx, userCred, query, input)
}

func (storage *SStorage) GetChangeOwnerRequiredDomainIds() []string {
	requires := stringutils2.SSortedStrings{}
	disks := storage.GetDisks()
	for i := range disks {
		requires = stringutils2.Append(requires, disks[i].DomainId)
	}
	return requires
}

func (storage *SStorage) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPublicDomainInput) (jsonutils.JSONObject, error) {
	// not allow to perform public for locally connected storage
	if storage.IsLocal() {
		hosts := storage.GetAttachedHosts()
		if len(hosts) > 0 {
			return nil, errors.Wrap(httperrors.ErrForbidden, "not allow to perform public for local storage")
		}
	}
	return storage.performPublicInternal(ctx, userCred, query, input)
}

func (storage *SStorage) performPublicInternal(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPublicDomainInput) (jsonutils.JSONObject, error) {
	return storage.SEnabledStatusInfrasResourceBase.PerformPublic(ctx, userCred, query, input)
}

func (storage *SStorage) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPrivateInput) (jsonutils.JSONObject, error) {
	// not allow to perform private for locally conencted storage
	if storage.IsLocal() {
		hosts := storage.GetAttachedHosts()
		if len(hosts) > 0 {
			return nil, errors.Wrap(httperrors.ErrForbidden, "not allow to perform private for local storage")
		}
	}
	return storage.performPrivateInternal(ctx, userCred, query, input)
}

func (storage *SStorage) performPrivateInternal(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPrivateInput) (jsonutils.JSONObject, error) {
	return storage.SEnabledStatusInfrasResourceBase.PerformPrivate(ctx, userCred, query, input)
}

func (manager *SStorageManager) ListItemExportKeys(ctx context.Context,
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
	if keys.Contains("schedtag") {
		schedtagsQ := SchedtagManager.Query("id", "name").SubQuery()
		storageSchedtagQ := StorageschedtagManager.Query("storage_id", "schedtag_id").SubQuery()

		subQ := storageSchedtagQ.Query(storageSchedtagQ.Field("storage_id"), sqlchemy.GROUP_CONCAT("schedtag", schedtagsQ.Field("name")))
		subQ = subQ.Join(schedtagsQ, sqlchemy.Equals(schedtagsQ.Field("id"), storageSchedtagQ.Field("schedtag_id")))
		subQ = subQ.GroupBy(storageSchedtagQ.Field("storage_id"))
		subQT := subQ.SubQuery()
		q = q.LeftJoin(subQT, sqlchemy.Equals(q.Field("id"), subQT.Field("storage_id")))
		q = q.AppendField(subQT.Field("schedtag"))
	}
	return q, nil
}

func (storage *SStorage) AllowPerformForceDetachHost(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, storage, "force-detach-host")
}

func (storage *SStorage) PerformForceDetachHost(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.StorageForceDetachHostInput) (jsonutils.JSONObject, error) {
	if storage.Enabled.Bool() {
		return nil, httperrors.NewBadRequestError("storage is enabled")
	}
	iHost, err := HostManager.FetchByIdOrName(userCred, input.HostId)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewNotFoundError("host %s not found", input.HostId)
	} else if err != nil {
		return nil, err
	}
	host := iHost.(*SHost)
	if host.Status == api.HOST_ONLINE {
		return nil, httperrors.NewBadRequestError("can't detach host in status online")
	}
	iHostStorage, err := db.FetchJointByIds(HoststorageManager, host.GetId(), storage.Id, nil)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewNotFoundError("host %s storage %s not found", input.HostId, storage.Name)
	} else if err != nil {
		return nil, err
	}
	hostStorage := iHostStorage.(*SHoststorage)
	hostStorage.SetModelManager(HoststorageManager, hostStorage)
	err = hostStorage.Delete(ctx, userCred)
	if err == nil {
		db.OpsLog.LogDetachEvent(ctx, db.JointMaster(hostStorage), db.JointSlave(hostStorage), userCred, jsonutils.NewString("force detach"))
	}
	return nil, err
}
