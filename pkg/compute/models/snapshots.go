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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
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
)

type SSnapshotManager struct {
	db.SVirtualResourceBaseManager
}

type SSnapshot struct {
	db.SVirtualResourceBase
	SManagedResourceBase

	DiskId      string `width:"36" charset:"ascii" nullable:"true" create:"required" list:"user"`
	StorageId   string `width:"36" charset:"ascii" nullable:"true" list:"admin" create:"optional"`
	CreatedBy   string `width:"36" charset:"ascii" nullable:"false" default:"manual" list:"admin" create:"optional"`
	Location    string `charset:"ascii" nullable:"true" list:"admin" create:"optional"`
	Size        int    `nullable:"false" list:"user" create:"required"` // MB
	OutOfChain  bool   `nullable:"false" default:"false" list:"admin"`
	FakeDeleted bool   `nullable:"false" default:"false"`
	DiskType    string `width:"32" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	// create disk from snapshot, snapshot as disk backing file
	RefCount int `nullable:"false" default:"0" list:"user"`

	CloudregionId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
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
}

func ValidateSnapshotName(hypervisor, name, owner string) error {
	q := SnapshotManager.Query()
	q = SnapshotManager.FilterByName(q, name)
	q = SnapshotManager.FilterByOwner(q, owner)
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
	if hypervisor == api.HYPERVISOR_ALIYUN {
		if strings.HasPrefix(name, "auto") || strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") {
			return httperrors.NewBadRequestError(
				"Snapshot for %s name can't start with auto, http:// or https://", hypervisor)
		}
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
	/*if cloudprovider := self.GetCloudprovider(); cloudprovider != nil {
		res.Add(jsonutils.NewString(cloudprovider.Provider), "hypervisor")
	}
	if len(self.CloudregionId) > 0 {
		cloudRegion := CloudregionManager.FetchRegionById(self.CloudregionId)
		if cloudRegion != nil {
			res.Add(jsonutils.NewString(cloudRegion.ExternalId), "region")
		}
	}*/
	info := self.getCloudProviderInfo()
	res.Update(jsonutils.Marshal(&info))
	return res
}

func (manager *SSnapshotManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	diskV := validators.NewModelIdOrNameValidator("disk", "disk", userCred.GetProjectId())
	if err := diskV.Validate(data); err != nil {
		return nil, err
	}
	disk := diskV.Model.(*SDisk)

	snapshotName, err := data.GetString("name")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("name")
	}

	guests := disk.GetGuests()
	if len(guests) != 1 {
		return nil, httperrors.NewBadRequestError("Disk %s dosen't attach guest ?", disk.Id)
	}

	guest := guests[0]
	if len(guest.BackupHostId) > 0 {
		return nil, httperrors.NewBadRequestError(
			"Disk attached Guest has backup, Can't create snapshot")
	}
	if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return nil, httperrors.NewInvalidStatusError("Cannot do snapshot when VM in status %s", guest.Status)
	}
	err = ValidateSnapshotName(guest.Hypervisor, snapshotName, userCred.GetProjectId())
	if err != nil {
		return nil, err
	}
	if guest.GetHypervisor() == api.HYPERVISOR_KVM {
		q := SnapshotManager.Query()
		cnt, err := q.Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), disk.Id),
			sqlchemy.Equals(q.Field("created_by"), api.SNAPSHOT_MANUAL),
			sqlchemy.IsFalse(q.Field("fake_deleted")))).CountWithError()
		if err != nil {
			return nil, httperrors.NewInternalServerError("check disk snapshot count fail %s", err)
		}
		if cnt >= options.Options.DefaultMaxManualSnapshotCount {
			return nil, httperrors.NewBadRequestError("Disk %s snapshot full, cannot take any more", disk.Id)
		}
	}
	pendingUsage := &SQuota{Snapshot: 1}
	_, err = QuotaManager.CheckQuota(ctx, userCred, ownerProjId, pendingUsage)
	if err != nil {
		return nil, httperrors.NewOutOfQuotaError("Check set pending quota error %s", err)
	}

	input := &api.SSnapshotCreateInput{}
	input.Name = snapshotName
	input.ProjectId = ownerProjId
	input.DiskId = disk.Id
	input.CreatedBy = api.SNAPSHOT_MANUAL
	input.Size = disk.DiskSize
	input.DiskType = disk.DiskType
	storage := disk.GetStorage()
	if len(disk.ExternalId) == 0 {
		input.StorageId = disk.StorageId
	}
	if cloudregion := storage.GetRegion(); cloudregion != nil {
		input.CloudregionId = cloudregion.GetId()
	}
	return input.JSON(input), nil
}

func (self *SSnapshot) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerProjId, query, data)
}

func (manager *SSnapshotManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	snapshot := items[0].(*SSnapshot)
	guest, _ := snapshot.GetGuest()
	params := jsonutils.NewDict()
	params.Set("snapshot_id", jsonutils.NewString(snapshot.Id))
	params.Set("disk_id", jsonutils.NewString(snapshot.DiskId))
	guest.GetDriver().StartGuestDiskSnapshotTask(ctx, userCred, guest, params)
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
	err := q.Query().Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), diskId),
		sqlchemy.Equals(q.Field("created_by"), createdBy),
		sqlchemy.Equals(q.Field("fake_deleted"), false))).All(&dest)
	if err != nil {
		log.Errorf("GetDiskSnapshots error: %s", err)
		return nil
	}
	for i := 0; i < len(dest); i++ {
		dest[i].SetModelManager(self)
	}
	return dest
}

func (self *SSnapshotManager) GetDiskSnapshots(diskId string) []SSnapshot {
	dest := make([]SSnapshot, 0)
	q := self.Query().SubQuery()
	err := q.Query().Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), diskId))).All(&dest)
	if err != nil {
		log.Errorf("GetDiskSnapshots error: %s", err)
		return nil
	}
	for i := 0; i < len(dest); i++ {
		dest[i].SetModelManager(self)
	}
	return dest
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
	dest.SetModelManager(self)
	return dest
}

func (self *SSnapshotManager) GetDiskSnapshotCount(diskId string) (int, error) {
	q := self.Query().SubQuery()
	return q.Query().Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), diskId),
		sqlchemy.Equals(q.Field("fake_deleted"), false))).CountWithError()
}

func (self *SSnapshotManager) CreateSnapshot(ctx context.Context, userCred mcclient.TokenCredential, createdBy, diskId, guestId, location, name string) (*SSnapshot, error) {
	iDisk, err := DiskManager.FetchById(diskId)
	if err != nil {
		return nil, err
	}
	disk := iDisk.(*SDisk)
	storage := disk.GetStorage()
	snapshot := &SSnapshot{}
	snapshot.SetModelManager(self)
	snapshot.ProjectId = userCred.GetProjectId()
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
	if self.RefCount > 0 {
		return fmt.Errorf("Snapshot reference(by disk) count > 0, can not delete")
	}
	return nil
}

func (self *SSnapshot) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if self.Status == api.SNAPSHOT_DELETING {
		return fmt.Errorf("Cannot delete snapshot in status %s", self.Status)
	}
	if len(self.ExternalId) == 0 {
		if self.CreatedBy == api.SNAPSHOT_MANUAL {
			if !self.FakeDeleted {
				return self.FakeDelete()
			}
			_, err := SnapshotManager.GetConvertSnapshot(self)
			if err != nil {
				return fmt.Errorf("Cannot delete snapshot: %s, disk need at least one of snapshot as backing file", err.Error())
			}
			return self.StartSnapshotDeleteTask(ctx, userCred, false, "")
		}
		return fmt.Errorf("Cannot delete snapshot created by %s", self.CreatedBy)
	}
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
	q := self.Query().SubQuery()
	err := q.Query().Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), deleteSnapshot.DiskId),
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
		if snapshots[i].CreatedBy == api.SNAPSHOT_MANUAL && snapshots[i].FakeDeleted == false {
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
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SSnapshot) FakeDelete() error {
	_, err := db.Update(self, func() error {
		self.FakeDeleted = true
		self.Name += timeutils.IsoTime(time.Now())
		return nil
	})
	return err
}

func (self *SSnapshot) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func TotalSnapshotCount(projectId string, rangeObj db.IStandaloneModel, providers []string) (int, error) {
	q := SnapshotManager.Query()
	if len(projectId) > 0 {
		q = q.Equals("tenant_id", projectId)
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
	if len(providers) > 0 {
		cloudproviders := CloudproviderManager.Query().SubQuery()
		subq := cloudproviders.Query(cloudproviders.Field("id"))
		subq = subq.In("provider", providers)
		q = q.In("manager_id", subq.SubQuery())
	}
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
func (self *SSnapshot) SyncWithCloudSnapshot(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudSnapshot, projectId string, region *SCloudregion) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		// self.Name = ext.GetName()
		self.Status = ext.GetStatus()
		self.DiskType = ext.GetDiskType()

		self.CloudregionId = region.Id
		return nil
	})
	if err != nil {
		log.Errorf("SyncWithCloudSnapshot fail %s", err)
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)

	SyncCloudProject(userCred, self, projectId, ext, self.ManagerId)

	return nil
}

func (manager *SSnapshotManager) newFromCloudSnapshot(ctx context.Context, userCred mcclient.TokenCredential, extSnapshot cloudprovider.ICloudSnapshot, region *SCloudregion, projectId string, provider *SCloudprovider) (*SSnapshot, error) {
	snapshot := SSnapshot{}
	snapshot.SetModelManager(manager)

	newName, err := db.GenerateName(manager, projectId, extSnapshot.GetName())
	if err != nil {
		return nil, err
	}
	snapshot.Name = newName
	snapshot.Status = extSnapshot.GetStatus()
	snapshot.ExternalId = extSnapshot.GetGlobalId()
	if len(extSnapshot.GetDiskId()) > 0 {
		disk, err := DiskManager.FetchByExternalId(extSnapshot.GetDiskId())
		if err != nil {
			log.Errorf("snapshot %s missing disk?", snapshot.Name)
		} else {
			snapshot.DiskId = disk.GetId()
		}
	}

	snapshot.DiskType = extSnapshot.GetDiskType()
	snapshot.Size = int(extSnapshot.GetSize()) * 1024
	snapshot.ManagerId = provider.Id
	snapshot.CloudregionId = region.Id

	err = manager.TableSpec().Insert(&snapshot)
	if err != nil {
		log.Errorf("newFromCloudEip fail %s", err)
		return nil, err
	}

	SyncCloudProject(userCred, &snapshot, projectId, extSnapshot, snapshot.ManagerId)

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

func (manager *SSnapshotManager) SyncSnapshots(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, snapshots []cloudprovider.ICloudSnapshot, projectId string) compare.SyncResult {
	syncOwnerProjId := projectId

	lockman.LockClass(ctx, manager, syncOwnerProjId)
	defer lockman.ReleaseClass(ctx, manager, syncOwnerProjId)

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
		err = commondb[i].SyncWithCloudSnapshot(ctx, userCred, commonext[i], projectId, region)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		local, err := manager.newFromCloudSnapshot(ctx, userCred, added[i], region, syncOwnerProjId, provider)
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
	err := self.ValidateDeleteCondition(ctx)
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
