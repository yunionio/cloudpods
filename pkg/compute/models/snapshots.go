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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SSnapshotManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
	SDiskResourceBaseManager
	SStorageResourceBaseManager
	db.SMultiArchResourceBaseManager
}

type SSnapshot struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	db.SMultiArchResourceBase

	// 磁盘Id
	DiskId string `width:"36" charset:"ascii" nullable:"true" create:"required" list:"user" index:"true"`

	// Only onecloud has StorageId
	StorageId string `width:"36" charset:"ascii" nullable:"true" list:"admin" create:"optional"`

	CreatedBy string `width:"36" charset:"ascii" nullable:"false" default:"manual" list:"user" create:"optional"`
	Location  string `charset:"ascii" nullable:"true" list:"admin" create:"optional"`
	// 快照大小,单位Mb
	Size        int    `nullable:"false" list:"user" create:"required"`
	OutOfChain  bool   `nullable:"false" default:"false" list:"admin" create:"optional"`
	FakeDeleted bool   `nullable:"false" default:"false"`
	DiskType    string `width:"32" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	// 操作系统类型
	OsType string `width:"32" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	// create disk from snapshot, snapshot as disk backing file
	RefCount int `nullable:"false" default:"0" list:"user"`

	// 区域Id
	// CloudregionId string    `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	BackingDiskId string    `width:"36" charset:"ascii" nullable:"true" default:""`
	ExpiredAt     time.Time `nullable:"true" list:"user" create:"optional"`
}

var SnapshotManager *SSnapshotManager

func init() {
	SnapshotManager = &SSnapshotManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SSnapshot{},
			"snapshots_tbl",
			"snapshot",
			"snapshots",
		),
	}
	SnapshotManager.SetVirtualObject(SnapshotManager)
}

func (self *SSnapshotManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

// 快照列表
func (manager *SSnapshotManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SnapshotListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SMultiArchResourceBaseManager.ListItemFilter(ctx, q, userCred, query.MultiArchResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SMultiArchResourceBaseManager.ListItemFilter")
	}

	if query.FakeDeleted != nil && *query.FakeDeleted {
		q = q.IsTrue("fake_deleted")
	} else {
		q = q.IsFalse("fake_deleted")
	}

	if query.Local != nil && *query.Local {
		storages := StorageManager.Query().SubQuery()
		sq := storages.Query(storages.Field("id")).Filter(sqlchemy.Equals(storages.Field("storage_type"), api.STORAGE_LOCAL))
		q = q.Filter(sqlchemy.In(q.Field("storage_id"), sq))
	}

	// Public cloud snapshot doesn't have storage id
	if query.Share != nil && *query.Share {
		storages := StorageManager.Query().SubQuery()
		sq := storages.Query(storages.Field("id")).NotEquals("storage_type", "local")
		q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("storage_id")),
			sqlchemy.In(q.Field("storage_id"), sq)))
	}

	if len(query.DiskType) > 0 {
		q = q.Equals("disk_type", query.DiskType)
	}

	if query.IsInstanceSnapshot != nil {
		insjsq := InstanceSnapshotJointManager.Query().SubQuery()
		if !*query.IsInstanceSnapshot {
			q = q.LeftJoin(insjsq, sqlchemy.Equals(q.Field("id"), insjsq.Field("snapshot_id"))).
				Filter(sqlchemy.IsNull(insjsq.Field("snapshot_id")))
		} else {
			q = q.Join(insjsq, sqlchemy.Equals(q.Field("id"), insjsq.Field("snapshot_id")))
		}
	}

	diskInput := api.DiskFilterListInput{
		DiskFilterListInputBase: query.DiskFilterListInputBase,
	}
	q, err = manager.SDiskResourceBaseManager.ListItemFilter(ctx, q, userCred, diskInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDiskResourceBaseManager.ListItemFilter")
	}
	storageInput := api.StorageFilterListInput{
		StorageFilterListInputBase: query.StorageFilterListInputBase,
	}
	q, err = manager.SStorageResourceBaseManager.ListItemFilter(ctx, q, userCred, storageInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStorageResourceBaseManager.ListItemFilter")
	}

	if query.OutOfChain != nil {
		if *query.OutOfChain {
			q = q.IsTrue("out_of_chain")
		} else {
			q = q.IsFalse("out_of_chain")
		}
	}
	if len(query.OsType) > 0 {
		q = q.In("os_type", query.OsType)
	}
	if len(query.ServerId) > 0 {
		iG, err := GuestManager.FetchByIdOrName(userCred, query.ServerId)
		if err != nil && err == sql.ErrNoRows {
			return nil, httperrors.NewNotFoundError("guest %s not found", query.ServerId)
		} else if err != nil {
			return nil, errors.Wrap(err, "fetch guest")
		}
		guest := iG.(*SGuest)
		gdq := GuestdiskManager.Query("disk_id").Equals("guest_id", guest.Id).SubQuery()
		q = q.In("disk_id", gdq)
	}

	return q, nil
}

func (manager *SSnapshotManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SnapshotListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}

	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}

	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}

	diskInput := api.DiskFilterListInput{
		DiskFilterListInputBase: query.DiskFilterListInputBase,
	}
	q, err = manager.SDiskResourceBaseManager.OrderByExtraFields(ctx, q, userCred, diskInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDiskResourceBaseManager.OrderByExtraFields")
	}
	storageInput := api.StorageFilterListInput{
		StorageFilterListInputBase: query.StorageFilterListInputBase,
	}
	q, err = manager.SStorageResourceBaseManager.OrderByExtraFields(ctx, q, userCred, storageInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStorageResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SSnapshotManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SSnapshotManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.SnapshotDetails {
	rows := make([]api.SnapshotDetails, len(objs))

	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.SnapshotDetails{
			VirtualResourceDetails:  virtRows[i],
			ManagedResourceInfo:     manRows[i],
			CloudregionResourceInfo: regionRows[i],
		}
		rows[i] = objs[i].(*SSnapshot).getMoreDetails(rows[i])
	}

	return rows
}

func (self *SSnapshot) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.SnapshotDetails, error) {
	return api.SnapshotDetails{}, nil
}

func (self *SSnapshot) getMoreDetails(out api.SnapshotDetails) api.SnapshotDetails {
	if IStorage, _ := StorageManager.FetchById(self.StorageId); IStorage != nil {
		storage := IStorage.(*SStorage)
		out.StorageType = storage.StorageType
	}
	disk, _ := self.GetDisk()
	if disk != nil {
		out.DiskStatus = disk.Status
		out.DiskName = disk.Name
		guests := disk.GetGuests()
		if len(guests) == 1 {
			out.Guest = guests[0].Name
			out.GuestId = guests[0].Id
			out.GuestStatus = guests[0].Status
		}
	}
	if t, _ := InstanceSnapshotJointManager.IsSubSnapshot(self.Id); t {
		out.IsSubSnapshot = true
	}

	return out
}

func (self *SSnapshot) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	res := self.SVirtualResourceBase.GetShortDesc(ctx)
	res.Add(jsonutils.NewInt(int64(self.Size)), "size")
	info := self.getCloudProviderInfo()
	res.Update(jsonutils.Marshal(&info))
	return res
}

func (manager *SSnapshotManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.SnapshotCreateInput,
) (*jsonutils.JSONDict, error) {
	for _, disk := range []string{input.Disk, input.DiskId} {
		if len(disk) > 0 {
			input.Disk = disk
			break
		}
	}
	if len(input.Disk) == 0 {
		return nil, httperrors.NewMissingParameterError("disk")
	}

	_disk, err := DiskManager.FetchByIdOrName(userCred, input.Disk)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError("failed to found disk %s", input.Disk)
		}
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "DiskManager.FetchByIdOrName"))
	}
	disk := _disk.(*SDisk)
	input.DiskId = disk.Id
	input.DiskType = disk.DiskType
	input.Size = disk.DiskSize
	input.OsArch = disk.OsArch

	storage := disk.GetStorage()
	if len(disk.ExternalId) == 0 {
		input.StorageId = disk.StorageId
	}
	input.ManagerId = storage.ManagerId
	region := storage.GetRegion()
	if region == nil {
		return nil, httperrors.NewInputParameterError("failed to found region for disk's storage %s(%s)", storage.Name, storage.Id)
	}
	input.CloudregionId = region.Id

	driver, err := storage.GetRegionDriver()
	if err != nil {
		return nil, errors.Wrap(err, "storage.GetRegionDriver")
	}
	input.OutOfChain = driver.SnapshotIsOutOfChain(disk)

	err = driver.ValidateCreateSnapshotData(ctx, userCred, disk, storage, &input)
	if err != nil {
		return nil, errors.Wrap(err, "driver.ValidateCreateSnapshotData")
	}

	input.VirtualResourceCreateInput, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return nil, err
	}

	pendingUsage := &SRegionQuota{Snapshot: 1}
	keys, err := disk.GetQuotaKeys()
	if err != nil {
		return nil, err
	}
	pendingUsage.SetKeys(keys.(SComputeResourceKeys).SRegionalCloudResourceKeys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, pendingUsage)
	if err != nil {
		return nil, err
	}

	return input.JSON(input), nil
}

func (self *SSnapshot) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	// use disk's ownerId instead of default ownerId
	diskObj, err := DiskManager.FetchById(self.DiskId)
	if err != nil {
		return errors.Wrap(err, "DiskManager.FetchById")
	}
	ownerId = diskObj.(*SDisk).GetOwnerId()
	return self.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (snapshot *SSnapshot) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	snapshot.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	pendingUsage := SRegionQuota{Snapshot: 1}
	keys := snapshot.GetQuotaKeys()
	pendingUsage.SetKeys(keys)
	err := quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, true)
	if err != nil {
		log.Errorf("quotas.CancelPendingUsage fail %s", err)
	}
}

func (manager *SSnapshotManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	snapshot := items[0].(*SSnapshot)
	snapshot.StartSnapshotCreateTask(ctx, userCred, nil, "")
}

func (self *SSnapshot) StartSnapshotCreateTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "SnapshotCreateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SSnapshot) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SSnapshot) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

func (self *SSnapshot) GetGuest() (*SGuest, error) {
	iDisk, err := DiskManager.FetchById(self.DiskId)
	if err != nil {
		return nil, err
	}
	disk := iDisk.(*SDisk)
	guests := disk.GetGuests()
	if len(guests) > 1 {
		return nil, fmt.Errorf("Snapshot disk attach mutil guest")
	} else if len(guests) == 1 {
		return &guests[0], nil
	} else {
		return nil, sql.ErrNoRows
	}
}

func (self *SSnapshot) GetDisk() (*SDisk, error) {
	iDisk, err := DiskManager.FetchById(self.DiskId)
	if err != nil {
		return nil, err
	}
	disk := iDisk.(*SDisk)
	return disk, nil
}

func (self *SSnapshot) GetHost() *SHost {
	iStorage, err := StorageManager.FetchById(self.StorageId)
	if err != nil {
		log.Errorln(err)
		return nil
	}
	storage := iStorage.(*SStorage)
	return storage.GetMasterHost()
}

func (self *SSnapshotManager) AddRefCount(snapshotId string, count int) {
	iSnapshot, _ := self.FetchById(snapshotId)
	if iSnapshot != nil {
		snapshot := iSnapshot.(*SSnapshot)
		_, err := db.Update(snapshot, func() error {
			snapshot.RefCount += count
			return nil
		})
		if err != nil {
			log.Errorf("Snapshot add refence count error: %s", err)
		}
	}
}

func (self *SSnapshotManager) GetDiskSnapshotsByCreate(diskId, createdBy string) []SSnapshot {
	dest := make([]SSnapshot, 0)
	q := self.Query().SubQuery()
	sq := q.Query().Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), diskId),
		sqlchemy.Equals(q.Field("created_by"), createdBy),
		sqlchemy.Equals(q.Field("fake_deleted"), false)))
	err := db.FetchModelObjects(self, sq, &dest)
	if err != nil {
		log.Errorf("GetDiskSnapshots error: %s", err)
		return nil
	}
	return dest
}

func (self *SSnapshotManager) GetDiskSnapshots(diskId string) []SSnapshot {
	dest := make([]SSnapshot, 0)
	q := self.Query().Equals("disk_id", diskId).Asc("created_at")
	err := db.FetchModelObjects(self, q, &dest)
	if err != nil {
		log.Errorf("GetDiskSnapshots error: %s", err)
		return nil
	}
	return dest
}

func (self *SSnapshotManager) GetDiskManualSnapshotCount(diskId string) (int, error) {
	return self.Query().Equals("disk_id", diskId).Equals("fake_deleted", false).CountWithError()
}

func (self *SSnapshotManager) IsDiskSnapshotsNeedConvert(diskId string) (bool, error) {
	count, err := self.Query().Equals("disk_id", diskId).
		In("status", []string{api.SNAPSHOT_READY, api.SNAPSHOT_DELETING}).
		Equals("out_of_chain", false).CountWithError()
	if err != nil {
		return false, err
	}
	return count >= options.Options.DefaultMaxSnapshotCount, nil
}

func (self *SSnapshotManager) GetDiskFirstSnapshot(diskId string) *SSnapshot {
	dest := &SSnapshot{}
	q := self.Query().SubQuery()
	err := q.Query().Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), diskId),
		sqlchemy.In(q.Field("status"), []string{api.SNAPSHOT_READY, api.SNAPSHOT_DELETING}),
		sqlchemy.Equals(q.Field("out_of_chain"), false))).Asc("created_at").First(dest)
	if err != nil {
		log.Errorf("Get Disk First snapshot error: %s", err.Error())
		return nil
	}
	dest.SetModelManager(self, dest)
	return dest
}

func (self *SSnapshotManager) GetDiskSnapshotCount(diskId string) (int, error) {
	q := self.Query().SubQuery()
	return q.Query().Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), diskId),
		sqlchemy.Equals(q.Field("fake_deleted"), false))).CountWithError()
}

func (self *SSnapshotManager) CreateSnapshot(ctx context.Context, owner mcclient.IIdentityProvider,
	createdBy, diskId, guestId, location, name string, retentionDay int) (*SSnapshot, error) {
	iDisk, err := DiskManager.FetchById(diskId)
	if err != nil {
		return nil, err
	}
	disk := iDisk.(*SDisk)
	storage := disk.GetStorage()
	snapshot := &SSnapshot{}
	snapshot.SetModelManager(self, snapshot)
	snapshot.ProjectId = owner.GetProjectId()
	snapshot.DomainId = owner.GetProjectDomainId()
	snapshot.DiskId = disk.Id
	if len(disk.ExternalId) == 0 {
		snapshot.StorageId = disk.StorageId
	}
	driver, err := storage.GetRegionDriver()
	if err != nil {
		return nil, err
	}
	snapshot.OutOfChain = driver.SnapshotIsOutOfChain(disk)
	snapshot.Size = disk.DiskSize
	snapshot.DiskType = disk.DiskType
	snapshot.Location = location
	snapshot.CreatedBy = createdBy
	snapshot.ManagerId = storage.ManagerId
	if cloudregion := storage.GetRegion(); cloudregion != nil {
		snapshot.CloudregionId = cloudregion.GetId()
	}
	snapshot.Name = name
	snapshot.Status = api.SNAPSHOT_CREATING
	if retentionDay > 0 {
		snapshot.ExpiredAt = time.Now().AddDate(0, 0, retentionDay)
	}
	err = SnapshotManager.TableSpec().Insert(ctx, snapshot)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (self *SSnapshot) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowDelete(userCred, self)
}

func (self *SSnapshot) StartSnapshotDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, reloadDisk bool, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Set("reload_disk", jsonutils.NewBool(reloadDisk))
	self.SetStatus(userCred, api.SNAPSHOT_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "SnapshotDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SSnapshot) ValidateDeleteCondition(ctx context.Context) error {
	if self.Status == api.SNAPSHOT_DELETING {
		return httperrors.NewBadRequestError("Cannot delete snapshot in status %s", self.Status)
	}
	return self.ValidatePurgeCondition(ctx)
}

func (self *SSnapshot) ValidatePurgeCondition(ctx context.Context) error {
	count, err := InstanceSnapshotJointManager.Query().Equals("snapshot_id", self.Id).CountWithError()
	if err != nil {
		return httperrors.NewInternalServerError("Fetch instance snapshot error %s", err)
	}
	if count > 0 {
		return httperrors.NewBadRequestError("snapshot referenced by instance snapshot")
	}
	if disk, err := self.GetDisk(); err == nil {
		if disk.Status == api.DISK_RESET {
			return httperrors.NewBadRequestError("Cannot delete snapshot on disk reset")
		}
	}
	driver := self.GetRegionDriver()
	if driver != nil {
		return driver.ValidateSnapshotDelete(ctx, self)
	}
	return nil
}

func (self *SSnapshot) GetStorage() *SStorage {
	return StorageManager.FetchStorageById(self.StorageId)
}

func (self *SSnapshot) GetStorageType() string {
	if storage := self.GetStorage(); storage != nil {
		return storage.StorageType
	}
	return ""
}

func (self *SSnapshot) GetRegionDriver() IRegionDriver {
	cloudRegion := self.GetRegion()
	if cloudRegion != nil {
		return cloudRegion.GetDriver()
	}
	return nil
}

func (self *SSnapshot) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartSnapshotDeleteTask(ctx, userCred, false, "")
}

func (self *SSnapshot) AllowPerformDeleted(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "deleted")
}

func (self *SSnapshot) PerformDeleted(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	_, err := db.Update(self, func() error {
		self.OutOfChain = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	err = self.StartSnapshotDeleteTask(ctx, userCred, true, "")
	return nil, err
}

func (self *SSnapshot) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "syncstatus")
}

// 同步快照状态
func (self *SSnapshot) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SnapshotSyncstatusInput) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(self, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("Snapshot has %d task active, can't sync status", count)
	}

	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "SnapshotSyncstatusTask", "")
}

func (self *SSnapshotManager) AllowGetPropertyMaxCount(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SSnapshotManager) GetPropertyMaxCount(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	ret.Set("max_count", jsonutils.NewInt(int64(options.Options.DefaultMaxSnapshotCount)))
	return ret, nil
}

func (self *SSnapshotManager) GetConvertSnapshot(deleteSnapshot *SSnapshot) (*SSnapshot, error) {
	dest := &SSnapshot{}
	q := self.Query()
	err := q.Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), deleteSnapshot.DiskId),
		sqlchemy.In(q.Field("status"), []string{api.SNAPSHOT_READY, api.SNAPSHOT_DELETING}),
		sqlchemy.Equals(q.Field("out_of_chain"), false),
		sqlchemy.GT(q.Field("created_at"), deleteSnapshot.CreatedAt))).
		Asc("created_at").First(dest)
	if err != nil {
		return nil, err
	}
	return dest, nil
}

func (self *SSnapshotManager) AllowPerformDeleteDiskSnapshots(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowClassPerform(userCred, self, "delete-disk-snapshots")
}

func (self *SSnapshotManager) PerformDeleteDiskSnapshots(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	diskId, err := data.GetString("disk_id")
	if err != nil {
		return nil, err
	}
	disk, err := DiskManager.FetchById(diskId)
	if disk != nil {
		return nil, httperrors.NewBadRequestError("Cannot Delete disk %s snapshots, disk exist", diskId)
	}
	snapshots := self.GetDiskSnapshots(diskId)
	if snapshots == nil || len(snapshots) == 0 {
		return nil, httperrors.NewNotFoundError("Disk %s dose not have snapshot", diskId)
	}
	for i := 0; i < len(snapshots); i++ {
		if snapshots[i].FakeDeleted == false {
			return nil, httperrors.NewBadRequestError("Can not delete disk snapshots, have manual snapshot")
		}
	}
	err = snapshots[0].StartSnapshotsDeleteTask(ctx, userCred, "")
	return nil, err
}

func (self *SSnapshot) StartSnapshotsDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "BatchSnapshotsDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SSnapshot) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if len(self.DiskId) > 0 {
		storage := self.GetStorage()
		if storage != nil && storage.StorageType == api.STORAGE_RBD {
			disk := DiskManager.FetchDiskById(self.DiskId)
			if disk != nil {
				cnt, err := disk.GetGuestsCount()
				if err == nil {
					val := disk.GetMetadata("disk_delete_after_snapshots", userCred)
					if cnt == 0 && val == "true" {
						disk.StartDiskDeleteTask(ctx, userCred, "", false, true, false)
					}
				} else {
					// very unlikely
					log.Errorf("disk.GetGuestsCount fail %s", err)
				}
			} else {
				backingDisks, err := self.GetBackingDisks()
				if err != nil {
					// very unlikely
					log.Errorf("self.GetBackingDisks fail %s", err)
				} else {
					storage.StartDeleteRbdDisks(ctx, userCred, backingDisks)
				}
			}
		}
	}
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SSnapshot) GetBackingDisks() ([]string, error) {
	count, err := SnapshotManager.Query().Equals("disk_id", self.DiskId).IsNullOrEmpty("backing_disk_id").CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, nil
	} else {
		sps := make([]SSnapshot, 0)
		err := SnapshotManager.Query().Equals("disk_id", self.DiskId).All(&sps)
		if err != nil {
			return nil, err
		}
		res := make([]string, 0)
		for i := 0; i < len(sps); i++ {
			if len(sps[i].BackingDiskId) > 0 && !utils.IsInStringArray(sps[i].BackingDiskId, res) {
				res = append(res, sps[i].BackingDiskId)
			}
		}
		res = append(res, self.DiskId)
		return res, nil
	}
}

func (self *SSnapshot) FakeDelete(userCred mcclient.TokenCredential) error {
	_, err := db.Update(self, func() error {
		self.FakeDeleted = true
		self.Name += timeutils.IsoTime(time.Now())
		return nil
	})
	if err == nil {
		db.OpsLog.LogEvent(self, db.ACT_SNAPSHOT_FAKE_DELETE, "snapshot fake delete", userCred)
	}
	return err
}

func (self *SSnapshot) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SSnapshotManager) DeleteDiskSnapshots(ctx context.Context, userCred mcclient.TokenCredential, diskId string) error {
	snapshots := self.GetDiskSnapshots(diskId)
	for i := 0; i < len(snapshots); i++ {
		if err := snapshots[i].RealDelete(ctx, userCred); err != nil {
			return errors.Wrap(err, "delete snapshot")
		}
	}
	return nil
}

func TotalSnapshotCount(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string) (int, error) {
	q := SnapshotManager.Query()

	switch scope {
	case rbacutils.ScopeSystem:
	case rbacutils.ScopeDomain:
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	case rbacutils.ScopeProject:
		q = q.Equals("tenant_id", ownerId.GetProjectId())
	}

	q = RangeObjectsFilter(q, rangeObjs, q.Field("cloudregion_id"), nil, q.Field("manager_id"), nil, nil)
	q = CloudProviderFilter(q, q.Field("manager_id"), providers, brands, cloudEnv)
	q = q.Equals("created_by", api.SNAPSHOT_MANUAL)
	q = q.Equals("fake_deleted", false)
	return q.CountWithError()
}

func (self *SSnapshot) syncRemoveCloudSnapshot(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		err = self.SetStatus(userCred, api.SNAPSHOT_UNKNOWN, "sync to delete")
	} else {
		err = self.RealDelete(ctx, userCred)
	}
	return err
}

// Only sync snapshot status
func (self *SSnapshot) SyncWithCloudSnapshot(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudSnapshot, syncOwnerId mcclient.IIdentityProvider, region *SCloudregion) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		// self.Name = ext.GetName()
		self.Status = ext.GetStatus()
		self.DiskType = ext.GetDiskType()
		self.Size = int(ext.GetSizeMb())

		self.CloudregionId = region.Id
		return nil
	})
	if err != nil {
		log.Errorf("SyncWithCloudSnapshot fail %s", err)
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)

	// bugfix for now:
	disk, err := self.GetDisk()
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrapf(err, "get disk of snapshot %s error", self.Id)
	}
	if err == nil {
		self.SyncCloudProjectId(userCred, disk.GetOwnerId())
	}

	return nil
}

func (manager *SSnapshotManager) newFromCloudSnapshot(ctx context.Context, userCred mcclient.TokenCredential, extSnapshot cloudprovider.ICloudSnapshot, region *SCloudregion, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider) (*SSnapshot, error) {
	snapshot := SSnapshot{}
	snapshot.SetModelManager(manager, &snapshot)

	snapshot.Status = extSnapshot.GetStatus()
	snapshot.ExternalId = extSnapshot.GetGlobalId()
	var localDisk *SDisk
	if len(extSnapshot.GetDiskId()) > 0 {
		disk, err := db.FetchByExternalIdAndManagerId(DiskManager, extSnapshot.GetDiskId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			sq := StorageManager.Query().SubQuery()
			return q.Join(sq, sqlchemy.Equals(q.Field("storage_id"), sq.Field("id"))).Filter(sqlchemy.Equals(sq.Field("manager_id"), provider.Id))
		})
		if err != nil {
			log.Errorf("snapshot %s missing disk?", snapshot.Name)
		} else {
			snapshot.DiskId = disk.GetId()
			localDisk = disk.(*SDisk)
		}
	}

	snapshot.DiskType = extSnapshot.GetDiskType()
	snapshot.Size = int(extSnapshot.GetSizeMb())
	snapshot.ManagerId = provider.Id
	snapshot.CloudregionId = region.Id

	var err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, manager, syncOwnerId, extSnapshot.GetName())
		if err != nil {
			return err
		}
		snapshot.Name = newName

		return manager.TableSpec().Insert(ctx, &snapshot)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	// bugfix for now:
	if localDisk != nil {
		snapshot.SyncCloudProjectId(userCred, localDisk.GetOwnerId())
	} else {
		SyncCloudProject(userCred, &snapshot, syncOwnerId, extSnapshot, snapshot.ManagerId)
	}

	db.OpsLog.LogEvent(&snapshot, db.ACT_CREATE, snapshot.GetShortDesc(ctx), userCred)

	return &snapshot, nil
}

func (manager *SSnapshotManager) getProviderSnapshotsByRegion(region *SCloudregion, provider *SCloudprovider) ([]SSnapshot, error) {
	if region == nil || provider == nil {
		return nil, fmt.Errorf("Region is nil or provider is nil")
	}
	snapshots := make([]SSnapshot, 0)
	q := manager.Query().Equals("cloudregion_id", region.Id).Equals("manager_id", provider.Id)
	err := db.FetchModelObjects(manager, q, &snapshots)
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}

func (manager *SSnapshotManager) SyncSnapshots(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, snapshots []cloudprovider.ICloudSnapshot, syncOwnerId mcclient.IIdentityProvider) compare.SyncResult {
	lockman.LockRawObject(ctx, "snapshots", fmt.Sprintf("%s-%s", provider.Id, region.Id))
	defer lockman.ReleaseRawObject(ctx, "snapshots", fmt.Sprintf("%s-%s", provider.Id, region.Id))

	syncResult := compare.SyncResult{}
	dbSnapshots, err := manager.getProviderSnapshotsByRegion(region, provider)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}
	removed := make([]SSnapshot, 0)
	commondb := make([]SSnapshot, 0)
	commonext := make([]cloudprovider.ICloudSnapshot, 0)
	added := make([]cloudprovider.ICloudSnapshot, 0)

	err = compare.CompareSets(dbSnapshots, snapshots, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}
	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudSnapshot(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].SyncWithCloudSnapshot(ctx, userCred, commonext[i], syncOwnerId, region)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncVirtualResourceMetadata(ctx, userCred, &commondb[i], commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		local, err := manager.newFromCloudSnapshot(ctx, userCred, added[i], region, syncOwnerId, provider)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncVirtualResourceMetadata(ctx, userCred, local, added[i])
			syncResult.Add()
		}
	}
	return syncResult
}

func (self *SSnapshot) GetRegion() *SCloudregion {
	return CloudregionManager.FetchRegionById(self.CloudregionId)
}

func (self *SSnapshot) GetISnapshotRegion() (cloudprovider.ICloudRegion, error) {
	provider, err := self.GetDriver()
	if err != nil {
		return nil, err
	}

	region := self.GetRegion()
	if region == nil {
		return nil, fmt.Errorf("fail to find region for snapshot")
	}
	return provider.GetIRegionById(region.GetExternalId())
}

func (self *SSnapshot) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "purge")
}

func (self *SSnapshot) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := self.GetRegionDriver().ValidateSnapshotDelete(ctx, self)
	if err != nil {
		return nil, err
	}
	provider := self.GetCloudprovider()
	if provider != nil {
		if provider.GetEnabled() {
			return nil, httperrors.NewInvalidStatusError("Cannot purge snapshot on enabled cloud provider")
		}
	}
	err = self.RealDelete(ctx, userCred)
	return nil, err
}

func (self *SSnapshot) getCloudProviderInfo() SCloudProviderInfo {
	region := self.GetRegion()
	provider := self.GetCloudprovider()
	return MakeCloudProviderInfo(region, nil, provider)
}

func (manager *SSnapshotManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	virts := manager.Query().IsFalse("fake_deleted")
	return db.CalculateResourceCount(virts, "tenant_id")
}

func (manager *SSnapshotManager) CleanupSnapshots(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	var now = time.Now()
	var snapshot = new(SSnapshot)
	err := manager.Query().
		Equals("fake_deleted", false).
		Equals("created_by", api.SNAPSHOT_AUTO).
		LE("expired_at", now).First(snapshot)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("Cleanup snapshots job fetch snapshot failed %s", err)
		return
	} else if err == sql.ErrNoRows {
		log.Infof("No snapshot need to clean ......")
		return
	}

	snapshot.SetModelManager(manager, snapshot)
	region := snapshot.GetRegion()
	if err = manager.StartSnapshotCleanupTask(ctx, userCred, region, now); err != nil {
		log.Errorf("Start snaphsot cleanup task failed %s", err)
		return
	}
}

func (manager *SSnapshotManager) StartSnapshotCleanupTask(
	ctx context.Context, userCred mcclient.TokenCredential,
	region *SCloudregion, now time.Time,
) error {
	params := jsonutils.NewDict()
	params.Set("tick", jsonutils.NewTimeString(now))
	task, err := taskman.TaskManager.NewTask(ctx, "SnapshotCleanupTask", region, userCred, params, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (snapshot *SSnapshot) GetQuotaKeys() quotas.IQuotaKeys {
	return fetchRegionalQuotaKeys(
		rbacutils.ScopeProject,
		snapshot.GetOwnerId(),
		snapshot.GetRegion(),
		snapshot.GetCloudprovider(),
	)
}

func (snapshot *SSnapshot) GetUsages() []db.IUsage {
	if snapshot.PendingDeleted || snapshot.Deleted {
		return nil
	}
	usage := SRegionQuota{Snapshot: 1}
	keys := snapshot.GetQuotaKeys()
	usage.SetKeys(keys)
	return []db.IUsage{
		&usage,
	}
}

func (manager *SSnapshotManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, err
	}
	if keys.Contains("disk") {
		q, err = manager.SDiskResourceBaseManager.ListItemExportKeys(ctx, q, userCred, stringutils2.NewSortedStrings([]string{"disk"}))
		if err != nil {
			return nil, errors.Wrap(err, "SDiskResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SStorageResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SStorageResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SStorageResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
