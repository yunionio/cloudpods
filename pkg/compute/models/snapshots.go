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
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SSnapshotManager struct {
	db.SVirtualResourceBaseManager
}

type SSnapshot struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase

	DiskId string `width:"36" charset:"ascii" nullable:"true" create:"required" list:"user" index:"true"`

	// Only onecloud has StorageId
	StorageId   string `width:"36" charset:"ascii" nullable:"true" list:"admin" create:"optional"`
	CreatedBy   string `width:"36" charset:"ascii" nullable:"false" default:"manual" list:"user" create:"optional"`
	Location    string `charset:"ascii" nullable:"true" list:"admin" create:"optional"`
	Size        int    `nullable:"false" list:"user" create:"required"` // MB
	OutOfChain  bool   `nullable:"false" default:"false" list:"admin" create:"optional"`
	FakeDeleted bool   `nullable:"false" default:"false"`
	DiskType    string `width:"32" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	// create disk from snapshot, snapshot as disk backing file
	RefCount int `nullable:"false" default:"0" list:"user"`

	CloudregionId string    `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
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

func ValidateSnapshotName(name string, owner mcclient.IIdentityProvider) error {
	q := SnapshotManager.Query()
	q = SnapshotManager.FilterByName(q, name)
	q = SnapshotManager.FilterByOwner(q, owner, SnapshotManager.NamespaceScope())
	q = SnapshotManager.FilterBySystemAttributes(q, nil, nil, SnapshotManager.ResourceScope())
	cnt, err := q.CountWithError()
	if err != nil {
		return err
	}
	if cnt != 0 {
		return httperrors.NewConflictError("Name %s conflict", name)
	}
	if !('A' <= name[0] && name[0] <= 'Z' || 'a' <= name[0] && name[0] <= 'z') {
		return httperrors.NewBadRequestError("Name must start with letter")
	}
	if len(name) < 2 || len(name) > 128 {
		return httperrors.NewBadRequestError("Snapshot name length must within 2~128")
	}
	return nil
}

func (self *SSnapshotManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (manager *SSnapshotManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	var err error
	q, err = managedResourceFilterByAccount(q, query, "", nil)
	if err != nil {
		return nil, err
	}
	q = managedResourceFilterByCloudType(q, query, "", nil)

	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	if jsonutils.QueryBoolean(query, "fake_deleted", false) {
		q = q.Equals("fake_deleted", true)
	} else {
		q = q.Equals("fake_deleted", false)
	}

	if jsonutils.QueryBoolean(query, "local", false) {
		storages := StorageManager.Query().SubQuery()
		sq := storages.Query(storages.Field("id")).Filter(sqlchemy.Equals(storages.Field("storage_type"), api.STORAGE_LOCAL))
		q = q.Filter(sqlchemy.In(q.Field("storage_id"), sq))
	}

	// Public cloud snapshot doesn't have storage id
	if jsonutils.QueryBoolean(query, "share", false) {
		storages := StorageManager.Query().SubQuery()
		sq := storages.Query(storages.Field("id")).NotEquals("storage_type", "local")
		q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("storage_id")),
			sqlchemy.In(q.Field("storage_id"), sq)))
	}

	if diskType, err := query.GetString("disk_type"); err == nil {
		diskTbl := DiskManager.Query().SubQuery()
		sq := diskTbl.Query(diskTbl.Field("id")).Equals("disk_type", diskType).SubQuery()
		q = q.In("disk_id", sq)
	}

	/*if provider, err := query.GetString("provider"); err == nil {
		cloudproviderTbl := CloudproviderManager.Query().SubQuery()
		sq := cloudproviderTbl.Query(cloudproviderTbl.Field("id")).Equals("provider", provider)
		q = q.In("manager_id", sq)
	}*/

	/*if managerStr := jsonutils.GetAnyString(query, []string{"manager", "manager_id"}); len(managerStr) > 0 {
		managerObj, err := CloudproviderManager.FetchByIdOrName(nil, managerStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewNotFoundError("manager %s not found", managerStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Equals("manager_id", managerObj.GetId())
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
	}*/

	return q, nil
}

func (self *SSnapshot) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(extra)
}

func (self *SSnapshot) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreDetails(extra), nil
}

func (self *SSnapshot) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	if IStorage, _ := StorageManager.FetchById(self.StorageId); IStorage != nil {
		storage := IStorage.(*SStorage)
		extra.Add(jsonutils.NewString(storage.StorageType), "storage_type")
		// if provider := storage.GetCloudprovider(); provider != nil {
		// 	extra.Add(jsonutils.NewString(provider.Name), "provider")
		// }
	} else {
		// if cloudprovider := self.GetCloudprovider(); cloudprovider != nil {
		// 	extra.Add(jsonutils.NewString(cloudprovider.Provider), "provider")
		// }
	}
	disk, _ := self.GetDisk()
	if disk != nil {
		extra.Add(jsonutils.NewString(disk.Status), "disk_status")
		guests := disk.GetGuests()
		if len(guests) == 1 {
			extra.Add(jsonutils.NewString(guests[0].Name), "guest")
			extra.Add(jsonutils.NewString(guests[0].Id), "guest_id")
			extra.Add(jsonutils.NewString(guests[0].Status), "guest_status")
		}
		extra.Add(jsonutils.NewString(disk.Name), "disk_name")
	}

	info := self.getCloudProviderInfo()
	extra.Update(jsonutils.Marshal(&info))

	return extra
}

func (self *SSnapshot) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	res := self.SVirtualResourceBase.GetShortDesc(ctx)
	res.Add(jsonutils.NewInt(int64(self.Size)), "size")
	info := self.getCloudProviderInfo()
	res.Update(jsonutils.Marshal(&info))
	return res
}

func (manager *SSnapshotManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	diskV := validators.NewModelIdOrNameValidator("disk", "disk", ownerId)
	if err := diskV.Validate(data); err != nil {
		return nil, err
	}
	disk := diskV.Model.(*SDisk)

	snapshotName, err := data.GetString("name")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("name")
	}
	err = ValidateSnapshotName(snapshotName, ownerId)
	if err != nil {
		return nil, err
	}

	err = disk.GetStorage().GetRegion().GetDriver().ValidateSnapshotCreate(ctx, userCred, disk, data)
	if err != nil {
		return nil, err
	}

	quotaPlatform := disk.GetQuotaPlatformID()
	pendingUsage := &SQuota{Snapshot: 1}
	_, err = QuotaManager.CheckQuota(ctx, userCred, rbacutils.ScopeProject, ownerId, quotaPlatform, pendingUsage)
	if err != nil {
		return nil, httperrors.NewOutOfQuotaError("Check set pending quota error %s", err)
	}

	input := &api.SSnapshotCreateInput{}
	input.Name = snapshotName
	input.ProjectId = ownerId.GetProjectId()
	input.DomainId = ownerId.GetProjectDomainId()
	input.DiskId = disk.Id
	input.CreatedBy = api.SNAPSHOT_MANUAL
	input.Size = disk.DiskSize
	input.DiskType = disk.DiskType
	input.OutOfChain = disk.GetStorage().GetRegion().GetDriver().SnapshotIsOutOfChain(disk)
	storage := disk.GetStorage()
	if len(disk.ExternalId) == 0 {
		input.StorageId = disk.StorageId
	}
	if cloudregion := storage.GetRegion(); cloudregion != nil {
		input.CloudregionId = cloudregion.GetId()
	}
	provider := disk.GetCloudprovider()
	if provider != nil {
		input.ManagerId = provider.Id
	}
	return input.JSON(input), nil
}

func (self *SSnapshot) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (manager *SSnapshotManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
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
	q := self.Query().SubQuery()
	sq := q.Query().Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), diskId)))
	err := db.FetchModelObjects(self, sq, &dest)
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
	err = SnapshotManager.TableSpec().Insert(snapshot)
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
	count, err := InstanceSnapshotJointManager.Query().Equals("snapshot_id", self.Id).CountWithError()
	if err != nil {
		return httperrors.NewInternalServerError("Fetch instance snapshot error %s", err)
	}
	if count > 0 {
		return httperrors.NewBadRequestError("snapshot referenced by instance snapshot")
	}
	return self.GetRegionDriver().ValidateSnapshotDelete(ctx, self)
}

func (self *SSnapshot) GetStorage() *SStorage {
	return StorageManager.FetchStorageById(self.StorageId)
}

func (self *SSnapshot) GetRegionDriver() IRegionDriver {
	return self.GetRegion().GetDriver()
}

func (self *SSnapshot) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartSnapshotDeleteTask(ctx, userCred, false, "")
}

func (self *SSnapshot) AllowPerformDeleted(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "deleted")
}

func (self *SSnapshot) PerformDeleted(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	db.Update(self, func() error {
		self.OutOfChain = true
		return nil
	})
	err := self.StartSnapshotDeleteTask(ctx, userCred, true, "")
	return nil, err
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

func TotalSnapshotCount(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObj db.IStandaloneModel, providers []string, brands []string, cloudEnv string) (int, error) {
	q := SnapshotManager.Query()

	switch scope {
	case rbacutils.ScopeSystem:
	case rbacutils.ScopeDomain:
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	case rbacutils.ScopeProject:
		q = q.Equals("tenant_id", ownerId.GetProjectId())
	}

	if rangeObj != nil {
		switch rangeObj.Keyword() {
		case "cloudprovider":
			q = q.Filter(sqlchemy.Equals(q.Field("manager_id"), rangeObj.GetId()))
		case "cloudaccount":
			cloudproviders := CloudproviderManager.Query().SubQuery()
			subq := cloudproviders.Query(cloudproviders.Field("id")).Equals("cloudaccount_id", rangeObj.GetId()).SubQuery()
			q = q.Filter(sqlchemy.In(q.Field("manager_id"), subq))
		case "cloudregion":
			q = q.Filter(sqlchemy.Equals(q.Field("cloudregion_id"), rangeObj.GetId()))
		}
	}

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

	newName, err := db.GenerateName(manager, syncOwnerId, extSnapshot.GetName())
	if err != nil {
		return nil, err
	}
	snapshot.Name = newName
	snapshot.Status = extSnapshot.GetStatus()
	snapshot.ExternalId = extSnapshot.GetGlobalId()
	var localDisk *SDisk
	if len(extSnapshot.GetDiskId()) > 0 {
		disk, err := db.FetchByExternalId(DiskManager, extSnapshot.GetDiskId())
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

	err = manager.TableSpec().Insert(&snapshot)
	if err != nil {
		log.Errorf("newFromCloudEip fail %s", err)
		return nil, err
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
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))

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
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		local, err := manager.newFromCloudSnapshot(ctx, userCred, added[i], region, syncOwnerId, provider)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, local, added[i])
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
		if provider.Enabled {
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

func (manager *SSnapshotManager) GetResourceCount() ([]db.SProjectResourceCount, error) {
	virts := manager.Query().IsFalse("fake_deleted")
	return db.CalculateProjectResourceCount(virts)
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
