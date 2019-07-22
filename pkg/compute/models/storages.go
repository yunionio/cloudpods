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
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SStorageManager struct {
	db.SStandaloneResourceBaseManager
}

var StorageManager *SStorageManager

func init() {
	StorageManager = &SStorageManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SStorage{},
			"storages_tbl",
			"storage",
			"storages",
		),
	}
	StorageManager.SetVirtualObject(StorageManager)
}

type SStorage struct {
	db.SStandaloneResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase

	Capacity    int64                `nullable:"false" list:"admin" update:"admin" create:"admin_required"`                           // Column(Integer, nullable=False) # capacity of disk in MB
	Reserved    int64                `nullable:"true" default:"0" list:"admin" update:"admin"`                                        // Column(Integer, nullable=True, default=0)
	StorageType string               `width:"32" charset:"ascii" nullable:"false" list:"user" update:"admin" create:"admin_required"` // Column(VARCHAR(32, charset='ascii'), nullable=False)
	MediumType  string               `width:"32" charset:"ascii" nullable:"false" list:"user" update:"admin" create:"admin_required"` // Column(VARCHAR(32, charset='ascii'), nullable=False)
	Cmtbound    float32              `nullable:"true" default:"1" list:"admin" update:"admin"`                                        // Column(Float, nullable=True)
	StorageConf jsonutils.JSONObject `nullable:"true" get:"admin" update:"admin"`                                                     // = Column(JSONEncodedDict, nullable=True)

	ZoneId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"admin_required"`

	StoragecacheId string `width:"36" charset:"ascii" nullable:"true" list:"admin" get:"admin" update:"admin" create:"optional"`

	Enabled tristate.TriState `nullable:"false" default:"true" list:"user" create:"optional"`
	Status  string            `width:"36" charset:"ascii" nullable:"false" default:"offline" update:"admin" list:"user" create:"optional"`

	// indicating whether system disk can be allocated in this storage
	IsSysDiskStore tristate.TriState `nullable:"false" default:"true" list:"user" create:"optional" update:"admin"`
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

func (self *SStorage) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)

	if data.Contains("cmtbound") {
		hosts := self.GetAttachedHosts()
		for _, host := range hosts {
			if err := host.ClearSchedDescCache(); err != nil {
				log.Errorf("clear host %s sched cache failed %v", host.GetName(), err)
			}
		}
	}
}

func (self *SStorage) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SStorage) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	DeleteResourceJointSchedtags(self, ctx, userCred)
	return self.SStandaloneResourceBase.Delete(ctx, userCred)
}

func (manager *SStorageManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	storageType, _ := data.GetString("storage_type")
	mediumType, _ := data.GetString("medium_type")

	if !utils.IsInStringArray(storageType, api.STORAGE_TYPES) {
		return nil, httperrors.NewInputParameterError("Invalid storage type %s", storageType)
	}
	if !utils.IsInStringArray(mediumType, api.DISK_TYPES) {
		return nil, httperrors.NewInputParameterError("Invalid medium type %s", mediumType)
	}
	zoneId, err := data.GetString("zone")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("zone")
	}
	zone, _ := ZoneManager.FetchByIdOrName(userCred, zoneId)
	if zone == nil {
		return nil, httperrors.NewResourceNotFoundError("zone %s", zoneId)
	}
	data.Set("zone_id", jsonutils.NewString(zone.GetId()))

	storageDirver := GetStorageDriver(storageType)
	if storageDirver == nil {
		return nil, httperrors.NewUnsupportOperationError("Not support create %s storage", storageType)
	}

	data, err = storageDirver.ValidateCreateData(ctx, userCred, data)
	if err != nil {
		return nil, err
	}

	return manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
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
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SStorage) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

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
	return self.StorageType == api.STORAGE_LOCAL || self.StorageType == api.STORAGE_BAREMETAL
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

	return capa
}

func (self *SStorage) getMoreDetails(ctx context.Context, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	capa := self.getStorageCapacity()
	extra.Update(capa.ToJson())

	extra.Add(jsonutils.NewFloat(float64(self.GetOvercommitBound())), "commit_bound")

	info := self.getCloudProviderInfo()
	extra.Update(jsonutils.Marshal(&info))
	extra = GetSchedtagsDetailsToResource(self, ctx, extra)

	return extra
}

func (self *SStorage) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(ctx, extra)
}

func (self *SStorage) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreDetails(ctx, extra), nil
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

	q := hosts.Query().Join(hoststorages, sqlchemy.AND(sqlchemy.Equals(hoststorages.Field("host_id"), hosts.Field("id")),
		sqlchemy.IsFalse(hoststorages.Field("deleted"))))
	q = q.Filter(sqlchemy.Equals(hoststorages.Field("storage_id"), self.Id))
	q = q.IsTrue("enabled")
	q = q.Equals("host_status", api.HOST_ONLINE).Asc("id")
	host := SHost{}
	host.SetModelManager(HostManager, &host)
	err := q.First(&host)
	if err != nil {
		log.Errorf("GetMasterHost fail %s", err)
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
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

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
		err = commondb[i].syncWithCloudStorage(ctx, userCred, commonext[i])
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

func (self *SStorage) syncWithCloudStorage(ctx context.Context, userCred mcclient.TokenCredential, extStorage cloudprovider.ICloudStorage) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		// self.Name = extStorage.GetName()
		self.Status = extStorage.GetStatus()
		self.StorageType = extStorage.GetStorageType()
		self.MediumType = extStorage.GetMediumType()
		if capacity := extStorage.GetCapacityMB(); capacity != 0 {
			self.Capacity = capacity
		}
		self.StorageConf = extStorage.GetStorageConf()

		self.Enabled = tristate.NewFromBool(extStorage.GetEnabled())

		self.IsEmulated = extStorage.IsEmulated()

		self.IsSysDiskStore = tristate.NewFromBool(extStorage.IsSysDiskStore())

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudZone error %s", err)
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return err
}

func (manager *SStorageManager) newFromCloudStorage(ctx context.Context, userCred mcclient.TokenCredential, extStorage cloudprovider.ICloudStorage, provider *SCloudprovider, zone *SZone) (*SStorage, error) {
	storage := SStorage{}
	storage.SetModelManager(manager, &storage)

	newName, err := db.GenerateName(manager, userCred, extStorage.GetName())
	if err != nil {
		return nil, err
	}
	storage.Name = newName
	storage.Status = extStorage.GetStatus()
	storage.ExternalId = extStorage.GetGlobalId()
	storage.ZoneId = zone.Id
	storage.StorageType = extStorage.GetStorageType()
	storage.MediumType = extStorage.GetMediumType()
	storage.StorageConf = extStorage.GetStorageConf()
	storage.Capacity = extStorage.GetCapacityMB()
	storage.Cmtbound = 1.0

	storage.Enabled = tristate.NewFromBool(extStorage.GetEnabled())

	storage.IsEmulated = extStorage.IsEmulated()
	storage.ManagerId = provider.Id

	storage.IsSysDiskStore = tristate.NewFromBool(extStorage.IsSysDiskStore())

	err = manager.TableSpec().Insert(&storage)
	if err != nil {
		log.Errorf("newFromCloudStorage fail %s", err)
		return nil, err
	}

	db.OpsLog.LogEvent(&storage, db.ACT_CREATE, storage.GetShortDesc(ctx), userCred)

	return &storage, nil
}

type StorageCapacityStat struct {
	TotalSize        int64
	TotalSizeVirtual float64
}

func filterDisksByScope(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) *sqlchemy.SSubQuery {
	q := DiskManager.Query()
	switch scope {
	case rbacutils.ScopeSystem:
	case rbacutils.ScopeDomain:
		q = q.Filter(sqlchemy.Equals(q.Field("domain_id"), ownerId.GetProjectDomainId()))
	case rbacutils.ScopeProject:
		q = q.Filter(sqlchemy.Equals(q.Field("tenant_id"), ownerId.GetProjectId()))
	}
	return q.SubQuery()
}

func (manager *SStorageManager) disksReadyQ(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) *sqlchemy.SSubQuery {
	disks := filterDisksByScope(scope, ownerId)
	q := disks.Query(
		disks.Field("storage_id"),
		sqlchemy.SUM("used_capacity", disks.Field("disk_size")),
	).Equals("status", api.DISK_READY)
	q = q.GroupBy(disks.Field("storage_id"))
	return q.SubQuery()
}

func (manager *SStorageManager) diskIsAttachedQ(isAttached bool, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) *sqlchemy.SSubQuery {
	sumKey := "attached_used_capacity"
	cond := sqlchemy.In
	if !isAttached {
		sumKey = "detached_used_capacity"
		cond = sqlchemy.NotIn
	}
	sq := GuestdiskManager.Query("disk_id").SubQuery()
	disks := filterDisksByScope(scope, ownerId)
	disks = disks.Query().Filter(cond(disks.Field("id"), sq)).SubQuery()
	q := disks.Query(
		disks.Field("storage_id"),
		sqlchemy.SUM(sumKey, disks.Field("disk_size")),
	).Equals("status", api.DISK_READY).GroupBy(disks.Field("storage_id"))
	return q.SubQuery()
}

func (manager *SStorageManager) diskAttachedQ(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) *sqlchemy.SSubQuery {
	return manager.diskIsAttachedQ(true, scope, ownerId)
}

func (manager *SStorageManager) diskDetachedQ(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) *sqlchemy.SSubQuery {
	return manager.diskIsAttachedQ(false, scope, ownerId)
}

func (manager *SStorageManager) disksFailedQ(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) *sqlchemy.SSubQuery {
	disks := filterDisksByScope(scope, ownerId)
	q := disks.Query(
		disks.Field("storage_id"),
		sqlchemy.SUM("failed_capacity", disks.Field("disk_size")),
	).NotEquals("status", api.DISK_READY)
	q = q.GroupBy(disks.Field("storage_id"))
	return q.SubQuery()
}

func (manager *SStorageManager) totalCapacityQ(
	rangeObj db.IStandaloneModel, hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider,
) *sqlchemy.SQuery {
	stmt := manager.disksReadyQ(scope, ownerId)
	stmt2 := manager.disksFailedQ(scope, ownerId)
	attachedDisks := manager.diskAttachedQ(scope, ownerId)
	detachedDisks := manager.diskDetachedQ(scope, ownerId)
	storages := manager.Query().SubQuery()
	q := storages.Query(
		storages.Field("capacity"),
		storages.Field("reserved"),
		storages.Field("cmtbound"),
		stmt.Field("used_capacity"),
		stmt2.Field("failed_capacity"),
		attachedDisks.Field("attached_used_capacity"),
		detachedDisks.Field("detached_used_capacity"),
	)
	q = q.LeftJoin(stmt, sqlchemy.Equals(stmt.Field("storage_id"), storages.Field("id")))
	q = q.LeftJoin(stmt2, sqlchemy.Equals(stmt2.Field("storage_id"), storages.Field("id")))
	q = q.LeftJoin(attachedDisks, sqlchemy.Equals(attachedDisks.Field("storage_id"), storages.Field("id")))
	q = q.LeftJoin(detachedDisks, sqlchemy.Equals(detachedDisks.Field("storage_id"), storages.Field("id")))

	if len(hostTypes) > 0 || len(resourceTypes) > 0 || rangeObj != nil {
		hosts := HostManager.Query().SubQuery()
		hostStorages := HoststorageManager.Query().SubQuery()

		q = q.Join(hostStorages, sqlchemy.Equals(hostStorages.Field("storage_id"), storages.Field("id")))
		q = q.Join(hosts, sqlchemy.Equals(hosts.Field("id"), hostStorages.Field("host_id")))
		q = q.Filter(sqlchemy.IsTrue(hosts.Field("enabled")))
		q = q.Filter(sqlchemy.Equals(hosts.Field("host_status"), api.HOST_ONLINE))

		q = AttachUsageQuery(q, hosts, hostTypes, resourceTypes, nil, nil, "", rangeObj)
	}

	q = CloudProviderFilter(q, storages.Field("manager_id"), providers, brands, cloudEnv)

	return q
}

type StorageStat struct {
	Capacity             int
	Reserved             int
	Cmtbound             float32
	UsedCapacity         int
	FailedCapacity       int
	AttachedUsedCapacity int
	DetachedUsedCapacity int
}

type StoragesCapacityStat struct {
	Capacity         int64
	CapacityVirtual  float64
	CapacityUsed     int64
	CapacityUnready  int64
	AttachedCapacity int64
	DetachedCapacity int64
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
		tFailed int64   = 0
		atCapa  int64   = 0
		dtCapa  int64   = 0
	)
	for _, stat := range stats {
		tCapa += int64(stat.Capacity - stat.Reserved)
		if stat.Cmtbound == 0 {
			stat.Cmtbound = options.Options.DefaultStorageOvercommitBound
		}
		tVCapa += float64(stat.Capacity-stat.Reserved) * float64(stat.Cmtbound)
		tUsed += int64(stat.UsedCapacity)
		tFailed += int64(stat.FailedCapacity)
		atCapa += int64(stat.AttachedUsedCapacity)
		dtCapa += int64(stat.DetachedUsedCapacity)
	}
	return StoragesCapacityStat{
		Capacity:         tCapa,
		CapacityVirtual:  tVCapa,
		CapacityUsed:     tUsed,
		CapacityUnready:  tFailed,
		AttachedCapacity: atCapa,
		DetachedCapacity: dtCapa,
	}
}

func (manager *SStorageManager) TotalCapacity(rangeObj db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) StoragesCapacityStat {
	res1 := manager.calculateCapacity(manager.totalCapacityQ(rangeObj, hostTypes, resourceTypes, providers, brands, cloudEnv, scope, ownerId))
	return res1
}

func (self *SStorage) createDisk(name string, diskConfig *api.DiskConfig, userCred mcclient.TokenCredential,
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
	disk.DomainId = ownerId.GetProjectDomainId()
	disk.IsSystem = isSystem

	disk.BillingType = billingType
	disk.BillingCycle = billingCycle

	err := disk.GetModelManager().TableSpec().Insert(&disk)
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
	q = q.Join(hoststorages, sqlchemy.AND(sqlchemy.Equals(hoststorages.Field("host_id"), hosts.Field("id")),
		sqlchemy.IsFalse(hoststorages.Field("deleted"))))
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
		log.Errorf("fail to find cloud provider")
		return nil, err
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
		log.Errorf("provider.GetIRegionById fail %s", err)
		return nil, err
	}
	istore, err := iRegion.GetIStorageById(self.GetExternalId())
	if err != nil {
		log.Errorf("iRegion.GetIStorageById fail %s", err)
		return nil, err
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
				if err := StoragecacheManager.TableSpec().Insert(storagecache); err != nil {
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

func (manager *SStorageManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
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

	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	regionStr, _ := query.GetString("region")
	if len(regionStr) > 0 {
		regionObj, err := CloudregionManager.FetchByIdOrName(userCred, regionStr)
		if err != nil {
			return nil, httperrors.NewNotFoundError("Region %s not found: %s", regionStr, err)
		}
		sq := ZoneManager.Query("id").Equals("cloudregion_id", regionObj.GetId())
		q = q.Filter(sqlchemy.In(q.Field("zone_id"), sq.SubQuery()))
	}

	if jsonutils.QueryBoolean(query, "share", false) {
		q = q.Filter(sqlchemy.NotIn(q.Field("storage_type"), api.STORAGE_LOCAL_TYPES))
	}

	if jsonutils.QueryBoolean(query, "local", false) {
		q = q.Filter(sqlchemy.In(q.Field("storage_type"), api.STORAGE_LOCAL_TYPES))
	}

	if jsonutils.QueryBoolean(query, "usable", false) {
		hostStorageTable := HoststorageManager.Query().SubQuery()
		hostTable := HostManager.Query().SubQuery()
		sq := hostStorageTable.Query(hostStorageTable.Field("storage_id")).Join(hostTable,
			sqlchemy.Equals(hostTable.Field("id"), hostStorageTable.Field("host_id"))).
			Filter(sqlchemy.Equals(hostTable.Field("host_status"), api.HOST_ONLINE))

		q = q.Filter(sqlchemy.In(q.Field("id"), sq)).
			Filter(sqlchemy.In(q.Field("status"), []string{api.STORAGE_ENABLED, api.STORAGE_ONLINE})).
			Filter(sqlchemy.IsTrue(q.Field("enabled")))
	}

	/*managerStr := jsonutils.GetAnyString(query, []string{"manager", "cloudprovider", "cloudprovider_id", "manager_id"})
	if len(managerStr) > 0 {
		provider, err := CloudproviderManager.FetchByIdOrName(nil, managerStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudproviderManager.Keyword(), managerStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Filter(sqlchemy.Equals(q.Field("manager_id"), provider.GetId()))
	}

	accountStr := jsonutils.GetAnyString(query, []string{"account", "account_id", "cloudaccount", "cloudaccount_id"})
	if len(accountStr) > 0 {
		account, err := CloudaccountManager.FetchByIdOrName(nil, accountStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudaccountManager.Keyword(), accountStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", account.GetId()).SubQuery()
		q = q.Filter(sqlchemy.In(q.Field("manager_id"), subq))
	}

	providerStr := jsonutils.GetAnyString(query, []string{"provider"})
	if len(providerStr) > 0 {
		subq := CloudproviderManager.Query("id").Equals("provider", providerStr).SubQuery()
		q = q.Filter(sqlchemy.In(q.Field("manager_id"), subq))
	}*/

	return q, err
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
