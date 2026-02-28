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
	"sync/atomic"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
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
	db.SEncryptedResourceManager
}

type SSnapshot struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	db.SMultiArchResourceBase

	db.SEncryptedResource

	// 磁盘Id
	DiskId string `width:"36" charset:"ascii" nullable:"true" create:"required" list:"user" index:"true"`

	// Only onecloud has StorageId
	StorageId string `width:"36" charset:"ascii" nullable:"true" list:"admin" create:"optional"`

	CreatedBy string `width:"36" charset:"ascii" nullable:"false" default:"manual" list:"user" create:"optional"`
	Location  string `charset:"ascii" nullable:"true" list:"admin" create:"optional"`
	// 快照大小,单位Mb
	Size int `nullable:"false" list:"user" create:"optional"`
	// Virtual size, for kvm is origin disk size
	VirtualSize int    `nullable:"false" list:"user" create:"optional"`
	OutOfChain  bool   `nullable:"false" default:"false" list:"admin" create:"optional"`
	FakeDeleted bool   `nullable:"false" default:"false"`
	DiskType    string `width:"32" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	// 操作系统类型
	OsType string `width:"32" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	// create disk from snapshot, snapshot as disk backing file
	RefCount int `nullable:"false" default:"0" list:"user"`

	BackingDiskId string    `width:"36" charset:"ascii" nullable:"true" default:""`
	DiskBackupId  string    `width:"36" charset:"ascii" nullable:"true" default:""`
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
		iG, err := GuestManager.FetchByIdOrName(ctx, userCred, query.ServerId)
		if err != nil && err == sql.ErrNoRows {
			return nil, httperrors.NewNotFoundError("guest %s not found", query.ServerId)
		} else if err != nil {
			return nil, errors.Wrap(err, "fetch guest")
		}
		guest := iG.(*SGuest)
		gdq := GuestdiskManager.Query("disk_id").Equals("guest_id", guest.Id).SubQuery()
		q = q.In("disk_id", gdq)
	}

	if query.Unused {
		sq := DiskManager.Query("id").Distinct().SubQuery()
		q = q.NotIn("disk_id", sq)
	}

	if len(query.StorageId) > 0 {
		q = q.Equals("storage_id", query.StorageId)
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
	if db.NeedOrderQuery([]string{query.OrderByGuest}) {
		guestSQ := GuestManager.Query("name", "id").SubQuery()
		guestdiskQ := GuestdiskManager.Query()
		guestdiskQ = guestdiskQ.Join(guestSQ, sqlchemy.Equals(guestSQ.Field("id"), guestdiskQ.Field("guest_id")))
		guestdiskSQ := guestdiskQ.AppendField(guestdiskQ.Field("disk_id"), guestSQ.Field("name").Label("guest_name")).SubQuery()
		q = q.LeftJoin(guestdiskSQ, sqlchemy.Equals(guestdiskSQ.Field("disk_id"), q.Field("disk_id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(guestdiskSQ.Field("guest_name"))
		q = db.OrderByFields(q, []string{query.OrderByGuest}, []sqlchemy.IQueryField{q.Field("guest_name")})
	}
	if db.NeedOrderQuery([]string{query.OrderByDiskName}) {
		dSQ := DiskManager.Query("name", "id").SubQuery()
		guestdiskQ := GuestdiskManager.Query()
		guestdiskQ = guestdiskQ.LeftJoin(dSQ, sqlchemy.Equals(dSQ.Field("id"), guestdiskQ.Field("disk_id")))
		guestdiskSQ := guestdiskQ.AppendField(guestdiskQ.Field("disk_id"), dSQ.Field("name").Label("disk_name")).SubQuery()
		q = q.LeftJoin(guestdiskSQ, sqlchemy.Equals(guestdiskSQ.Field("disk_id"), q.Field("disk_id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(guestdiskSQ.Field("disk_name"))
		q = db.OrderByFields(q, []string{query.OrderByDiskName}, []sqlchemy.IQueryField{q.Field("disk_name")})
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

func (manager *SSnapshotManager) QueryDistinctExtraFields(q *sqlchemy.SQuery, resource string, fields []string) (*sqlchemy.SQuery, error) {
	switch resource {
	case StorageManager.Keyword():
		storages := StorageManager.Query().SubQuery()
		for _, field := range fields {
			q = q.AppendField(storages.Field(field))
		}
		q = q.Join(storages, sqlchemy.Equals(q.Field("storage_id"), storages.Field("id")))
		return q, nil

	case CloudproviderManager.Keyword():
		return manager.SManagedResourceBaseManager.QueryDistinctExtraFields(q, resource, fields)
	}
	return q, httperrors.ErrNotFound
}

type sSnapshotGuest struct {
	DiskId      string
	GuestId     string
	GuestName   string
	GuestStatus string
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
	encRows := manager.SEncryptedResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	storageIds := make([]string, len(objs))
	diskIds := make([]string, len(objs))
	snapshotIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.SnapshotDetails{
			VirtualResourceDetails:  virtRows[i],
			ManagedResourceInfo:     manRows[i],
			CloudregionResourceInfo: regionRows[i],

			EncryptedResourceDetails: encRows[i],
		}
		snapshot := objs[i].(*SSnapshot)
		storageIds[i] = snapshot.StorageId
		diskIds[i] = snapshot.DiskId
		snapshotIds[i] = snapshot.Id
	}

	storages := map[string]SStorage{}
	err := db.FetchModelObjectsByIds(StorageManager, "id", storageIds, storages)
	if err != nil {
		log.Errorf("FetchModelObjectsByIds")
		return rows
	}
	disks := map[string]SDisk{}
	err = db.FetchModelObjectsByIds(DiskManager, "id", diskIds, disks)
	if err != nil {
		log.Errorf("FetchModelObjectsByIds")
		return rows
	}
	iss := []SInstanceSnapshotJoint{}
	err = InstanceSnapshotJointManager.Query().In("snapshot_id", snapshotIds).All(&iss)
	if err != nil {
		log.Errorf("query instance snapshot joint")
		return rows
	}
	issMap := map[string]bool{}
	for i := range iss {
		issMap[iss[i].SnapshotId] = true
	}
	q := GuestManager.Query()
	gds := GuestdiskManager.Query().SubQuery()
	sq := q.SubQuery()
	guests := sq.Query(
		sq.Field("id").Label("guest_id"),
		sq.Field("name").Label("guest_name"),
		sq.Field("status").Label("guest_status"),
		gds.Field("disk_id"),
	).Join(gds, sqlchemy.Equals(gds.Field("guest_id"), sq.Field("id"))).Filter(sqlchemy.In(gds.Field("disk_id"), diskIds))
	guestdisks := []struct {
		DiskId      string
		GuestId     string
		GuestName   string
		GuestStatus string
	}{}
	err = guests.All(&guestdisks)
	if err != nil {
		log.Errorf("guests.All")
		return rows
	}
	guestMap := map[string]struct {
		GuestId     string
		GuestName   string
		GuestStatus string
	}{}
	for _, gd := range guestdisks {
		guestMap[gd.DiskId] = struct {
			GuestId     string
			GuestName   string
			GuestStatus string
		}{
			GuestId:     gd.GuestId,
			GuestName:   gd.GuestName,
			GuestStatus: gd.GuestStatus,
		}
	}

	for i := range rows {
		if storage, ok := storages[storageIds[i]]; ok {
			rows[i].StorageType = storage.StorageType
			rows[i].Storage = storage.Name
		}
		if disk, ok := disks[diskIds[i]]; ok {
			rows[i].DiskStatus = disk.Status
			rows[i].DiskName = disk.Name
		}
		if guest, ok := guestMap[diskIds[i]]; ok {
			rows[i].GuestId = guest.GuestId
			rows[i].Guest = guest.GuestName
			rows[i].GuestStatus = guest.GuestStatus
		}
		rows[i].IsSubSnapshot, _ = issMap[snapshotIds[i]]
	}

	return rows
}

func (self *SSnapshot) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	res := self.SVirtualResourceBase.GetShortDesc(ctx)
	res.Add(jsonutils.NewInt(int64(self.VirtualSize)), "virtual_size")
	res.Add(jsonutils.NewInt(int64(self.Size)), "size")
	res.Add(jsonutils.NewString(self.DiskId), "disk_id")
	disk, _ := self.GetDisk()
	if disk != nil {
		if guest := disk.GetGuest(); guest != nil {
			res.Add(jsonutils.NewString(guest.Id), "guest_id")
		}
	}
	info := self.getCloudProviderInfo()
	res.Update(jsonutils.Marshal(&info))
	return res
}

// 创建快照
func (manager *SSnapshotManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.SnapshotCreateInput,
) (api.SnapshotCreateInput, error) {
	if input.NeedEncrypt() {
		return input, errors.Wrap(httperrors.ErrInputParameter, "encryption should not be set")
	}

	if len(input.DiskId) == 0 {
		return input, httperrors.NewMissingParameterError("disk_id")
	}
	_disk, err := validators.ValidateModel(ctx, userCred, DiskManager, &input.DiskId)
	if err != nil {
		return input, err
	}

	disk := _disk.(*SDisk)
	if disk.Status != api.DISK_READY {
		return input, httperrors.NewInvalidStatusError("disk %s status is not %s", disk.Name, api.DISK_READY)
	}

	if len(disk.SnapshotId) > 0 {
		if disk.GetMetadata(ctx, "merge_snapshot", userCred) == "true" {
			return input, httperrors.NewBadRequestError("disk %s backing snapshot not merged", disk.Id)
		}
	}

	if len(disk.EncryptKeyId) > 0 {
		input.EncryptKeyId = &disk.EncryptKeyId
		input.EncryptedResourceCreateInput, err = manager.SEncryptedResourceManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EncryptedResourceCreateInput)
		if err != nil {
			return input, errors.Wrap(err, "SEncryptedResourceManager.ValidateCreateData")
		}
	}

	input.DiskType = disk.DiskType
	input.Size = disk.DiskSize
	input.VirtualSize = disk.DiskSize
	input.OsArch = disk.OsArch

	storage, _ := disk.GetStorage()
	if len(disk.ExternalId) == 0 {
		input.StorageId = disk.StorageId
	}
	input.ManagerId = storage.ManagerId
	region, err := storage.GetRegion()
	if err != nil {
		return input, err
	}
	input.CloudregionId = region.Id

	driver, err := storage.GetRegionDriver()
	if err != nil {
		return input, errors.Wrap(err, "storage.GetRegionDriver")
	}
	input.OutOfChain = driver.SnapshotIsOutOfChain(disk)

	err = driver.ValidateCreateSnapshotData(ctx, userCred, disk, storage, &input)
	if err != nil {
		return input, errors.Wrap(err, "driver.ValidateCreateSnapshotData")
	}

	input.VirtualResourceCreateInput, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, err
	}

	pendingUsage := &SRegionQuota{Snapshot: 1}
	keys, err := disk.GetQuotaKeys()
	if err != nil {
		return input, err
	}
	pendingUsage.SetKeys(keys.(SComputeResourceKeys).SRegionalCloudResourceKeys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, pendingUsage)
	if err != nil {
		return input, err
	}

	return input, nil
}

func (self *SSnapshot) CustomizeCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) error {
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
	disk, err := snapshot.GetDisk()
	if err != nil {
		log.Errorf("unable to GetDisk: %s", err.Error())
	}
	err = disk.InheritTo(ctx, userCred, snapshot)
	if err != nil {
		log.Errorf("unable to inherit from disk %s to snapshot %s: %s", disk.GetId(), snapshot.GetId(), err.Error())
	}
}

func (manager *SSnapshotManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data []jsonutils.JSONObject) {
	snapshot := items[0].(*SSnapshot)
	snapshot.StartSnapshotCreateTask(ctx, userCred, nil, "")
}

func (self *SSnapshot) StartSnapshotCreateTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "SnapshotCreateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
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

func (self *SSnapshot) GetHost() (*SHost, error) {
	iStorage, err := StorageManager.FetchById(self.StorageId)
	if err != nil {
		return nil, errors.Wrapf(err, "StorageManager.FetchById(%s)", self.StorageId)
	}
	storage := iStorage.(*SStorage)
	return storage.GetMasterHost()
}

func (self *SSnapshot) GetFuseUrl() (string, error) {
	iStorage, err := StorageManager.FetchById(self.StorageId)
	if err != nil {
		return "", errors.Wrapf(err, "StorageManager.FetchById(%s)", self.StorageId)
	}
	storage := iStorage.(*SStorage)
	if storage.StorageType != api.STORAGE_LOCAL && storage.StorageType != api.STORAGE_LVM {
		return "", nil
	}
	host, err := storage.GetMasterHost()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/snapshots/%s/%s", host.GetFetchUrl(true), self.DiskId, self.Id), nil
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
	createdBy, diskId, guestId, location, name string, retentionDay int, isSystem bool, diskBackupId string) (*SSnapshot, error) {
	iDisk, err := DiskManager.FetchById(diskId)
	if err != nil {
		return nil, err
	}
	disk := iDisk.(*SDisk)
	storage, _ := disk.GetStorage()
	snapshot := &SSnapshot{}
	snapshot.SetModelManager(self, snapshot)
	snapshot.ProjectId = owner.GetProjectId()
	snapshot.DomainId = owner.GetProjectDomainId()
	snapshot.DiskId = disk.Id
	if len(disk.ExternalId) == 0 {
		snapshot.StorageId = disk.StorageId
	}

	// inherit encrypt_key_id
	snapshot.EncryptKeyId = disk.EncryptKeyId

	driver, err := storage.GetRegionDriver()
	if err != nil {
		return nil, err
	}
	snapshot.OutOfChain = driver.SnapshotIsOutOfChain(disk)
	snapshot.VirtualSize = disk.DiskSize
	snapshot.DiskType = disk.DiskType
	snapshot.Location = location
	snapshot.CreatedBy = createdBy
	snapshot.ManagerId = storage.ManagerId
	if cloudregion, _ := storage.GetRegion(); cloudregion != nil {
		snapshot.CloudregionId = cloudregion.GetId()
	}
	snapshot.Name = name
	snapshot.Status = api.SNAPSHOT_CREATING
	if retentionDay > 0 {
		snapshot.ExpiredAt = time.Now().AddDate(0, 0, retentionDay)
	}
	snapshot.IsSystem = isSystem
	snapshot.DiskBackupId = diskBackupId
	err = SnapshotManager.TableSpec().Insert(ctx, snapshot)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (self *SSnapshot) StartSnapshotDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, reloadDisk bool, parentTaskId string, deleteSnapshotTotalCnt int, deletedSnapshotCnt int) error {
	params := jsonutils.NewDict()
	params.Set("reload_disk", jsonutils.NewBool(reloadDisk))
	if deleteSnapshotTotalCnt <= 0 {
		deleteSnapshotTotalCnt = 1
	}
	params.Set("snapshot_total_count", jsonutils.NewInt(int64(deleteSnapshotTotalCnt)))
	params.Set("deleted_snapshot_count", jsonutils.NewInt(int64(deletedSnapshotCnt)))
	self.SetStatus(ctx, userCred, api.SNAPSHOT_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "SnapshotDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SSnapshot) ValidateDeleteCondition(ctx context.Context, info *api.SnapshotDetails) error {
	if self.Status == api.SNAPSHOT_DELETING {
		return httperrors.NewBadRequestError("Cannot delete snapshot in status %s", self.Status)
	}
	if gotypes.IsNil(info) {
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
	} else {
		if info.IsSubSnapshot {
			return httperrors.NewBadRequestError("snapshot referenced by instance snapshot")
		}
		if info.DiskStatus == api.DISK_RESET {
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
	cloudRegion, _ := self.GetRegion()
	if cloudRegion != nil {
		return cloudRegion.GetDriver()
	}
	return nil
}

func (self *SSnapshot) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartSnapshotDeleteTask(ctx, userCred, false, "", 0, 0)
}

// +onecloud:swagger-gen-ignore
func (self *SSnapshot) PerformDeleted(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	_, err := db.Update(self, func() error {
		self.OutOfChain = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	err = self.StartSnapshotDeleteTask(ctx, userCred, true, "", 0, 0)
	return nil, err
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

// +onecloud:swagger-gen-ignore
func (self *SSnapshotManager) PerformDeleteDiskSnapshots(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.SnapshotDeleteDiskSnapshotsInput) (jsonutils.JSONObject, error) {
	disk, err := DiskManager.FetchById(input.DiskId)
	if disk != nil {
		return nil, httperrors.NewBadRequestError("Cannot Delete disk %s snapshots, disk exist", input.DiskId)
	}
	snapshots := self.GetDiskSnapshots(input.DiskId)
	if snapshots == nil || len(snapshots) == 0 {
		return nil, httperrors.NewNotFoundError("Disk %s dose not have snapshot", input.DiskId)
	}
	snapshotIds := []string{}
	for i := 0; i < len(snapshots); i++ {
		if snapshots[i].FakeDeleted == false {
			return nil, httperrors.NewBadRequestError("Can not delete disk snapshots, have manual snapshot")
		}
		snapshotIds = append(snapshotIds, snapshots[i].Id)
	}
	err = snapshots[0].StartSnapshotsDeleteTask(ctx, userCred, "", snapshotIds)
	return nil, err
}

func (self *SSnapshot) StartSnapshotsDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, snapshotIds []string) error {
	data := jsonutils.NewDict()
	data.Set("snapshot_ids", jsonutils.NewStringArray(snapshotIds))
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
					val := disk.GetMetadata(ctx, "disk_delete_after_snapshots", userCred)
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

func TotalSnapshotCount(ctx context.Context, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) (int, error) {
	q := SnapshotManager.Query()

	switch scope {
	case rbacscope.ScopeSystem:
	case rbacscope.ScopeDomain:
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	case rbacscope.ScopeProject:
		q = q.Equals("tenant_id", ownerId.GetProjectId())
	}

	q = db.ObjectIdQueryWithPolicyResult(ctx, q, SnapshotManager, policyResult)

	q = RangeObjectsFilter(q, rangeObjs, q.Field("cloudregion_id"), nil, q.Field("manager_id"), nil, nil)
	q = CloudProviderFilter(q, q.Field("manager_id"), providers, brands, cloudEnv)
	q = q.Equals("created_by", api.SNAPSHOT_MANUAL)
	q = q.Equals("fake_deleted", false)
	return q.CountWithError()
}

func (self *SSnapshot) syncRemoveCloudSnapshot(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		err = self.SetStatus(ctx, userCred, api.SNAPSHOT_UNKNOWN, "sync to delete")
	} else {
		err = self.RealDelete(ctx, userCred)
	}
	return err
}

// Only sync snapshot status
func (self *SSnapshot) SyncWithCloudSnapshot(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudSnapshot, syncOwnerId mcclient.IIdentityProvider, region *SCloudregion) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		if options.Options.EnableSyncName {
			newName, _ := db.GenerateAlterName(self, ext.GetName())
			if len(newName) > 0 {
				self.Name = newName
			}
		}
		self.Status = ext.GetStatus()
		self.DiskType = ext.GetDiskType()
		self.VirtualSize = int(ext.GetSizeMb())
		self.Size = int(ext.GetSizeMb())
		disk, _ := self.GetDisk()
		if gotypes.IsNil(disk) && len(ext.GetDiskId()) > 0 {
			disk, err := db.FetchByExternalIdAndManagerId(DiskManager, ext.GetDiskId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				sq := StorageManager.Query().SubQuery()
				return q.Join(sq, sqlchemy.Equals(q.Field("storage_id"), sq.Field("id"))).Filter(sqlchemy.Equals(sq.Field("manager_id"), self.ManagerId))
			})
			if err != nil {
				log.Errorf("snapshot %s missing disk?", self.Name)
			} else {
				self.DiskId = disk.GetId()
			}
		}

		self.CloudregionId = region.Id
		return nil
	})
	if err != nil {
		log.Errorf("SyncWithCloudSnapshot fail %s", err)
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)

	if account := self.GetCloudaccount(); account != nil {
		syncVirtualResourceMetadata(ctx, userCred, self, ext, account.ReadOnly)
	}

	// bugfix for now:
	disk, _ := self.GetDisk()
	if disk != nil {
		self.SyncCloudProjectId(userCred, disk.GetOwnerId())
	} else {
		SyncCloudProject(ctx, userCred, self, syncOwnerId, ext, self.GetCloudprovider())
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
	snapshot.VirtualSize = int(extSnapshot.GetSizeMb())
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

	syncVirtualResourceMetadata(ctx, userCred, &snapshot, extSnapshot, false)

	// bugfix for now:
	if localDisk != nil {
		snapshot.SyncCloudProjectId(userCred, localDisk.GetOwnerId())
	} else {
		SyncCloudProject(ctx, userCred, &snapshot, syncOwnerId, extSnapshot, provider)
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

func (manager *SSnapshotManager) SyncSnapshots(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	region *SCloudregion,
	snapshots []cloudprovider.ICloudSnapshot,
	syncOwnerId mcclient.IIdentityProvider,
	xor bool,
) compare.SyncResult {
	lockman.LockRawObject(ctx, manager.Keyword(), fmt.Sprintf("%s-%s", provider.Id, region.Id))
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), fmt.Sprintf("%s-%s", provider.Id, region.Id))

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
			continue
		}
		syncResult.Delete()
	}
	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			err = commondb[i].SyncWithCloudSnapshot(ctx, userCred, commonext[i], syncOwnerId, region)
			if err != nil {
				syncResult.UpdateError(err)
				continue
			}
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		_, err := manager.newFromCloudSnapshot(ctx, userCred, added[i], region, syncOwnerId, provider)
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		syncResult.Add()
	}
	return syncResult
}

func (self *SSnapshot) GetISnapshotRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	provider, err := self.GetDriver(ctx)
	if err != nil {
		return nil, err
	}

	region, err := self.GetRegion()
	if err != nil {
		return nil, err
	}
	return provider.GetIRegionById(region.GetExternalId())
}

// +onecloud:swagger-gen-ignore
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
	region, _ := self.GetRegion()
	provider := self.GetCloudprovider()
	return MakeCloudProviderInfo(region, nil, provider)
}

func (manager *SSnapshotManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	virts := manager.Query().IsFalse("fake_deleted")
	return db.CalculateResourceCount(virts, "tenant_id")
}

var SnapshotCleanupTaskRunning int32 = 0

func SnapshotCleanupTaskIsRunning() bool {
	return atomic.LoadInt32(&SnapshotCleanupTaskRunning) == 1
}

func (manager *SSnapshotManager) CleanupSnapshots(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if SnapshotCleanupTaskIsRunning() {
		log.Errorf("Previous CleanupSnapshots tasks still running !!!")
		return
	}
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
	region, _ := snapshot.GetRegion()
	if err = manager.StartSnapshotCleanupTask(ctx, userCred, region, now); err != nil {
		log.Errorf("Start snaphsot cleanup task failed %s", err)
		return
	}

	sq := manager.Query().Equals("status", api.SNAPSHOT_READY).Equals("created_by", api.SNAPSHOT_AUTO).Equals("fake_deleted", false).SubQuery()

	disks := []struct {
		DiskCnt int
		DiskId  string
	}{}
	q := sq.Query(
		sqlchemy.COUNT("disk_cnt", sq.Field("disk_id")),
		sq.Field("disk_id"),
	).GroupBy(sq.Field("disk_id"))
	err = q.All(&disks)
	if err != nil {
		log.Errorf("Cleanup snapshots job fetch disk count failed %s", err)
		return
	}

	diskCount := map[string]int{}
	for i := range disks {
		diskCount[disks[i].DiskId] = disks[i].DiskCnt
	}

	{
		sq = SnapshotPolicyManager.Query().Equals("type", api.SNAPSHOT_POLICY_TYPE_DISK).GT("retention_count", 0).SubQuery()
		spd := SnapshotPolicyResourceManager.Query().Equals("resource_type", api.SNAPSHOT_POLICY_TYPE_DISK).SubQuery()
		q = sq.Query(
			sq.Field("retention_count"),
			spd.Field("resource_id").Label("disk_id"),
		)
		q = q.Join(spd, sqlchemy.Equals(q.Field("id"), spd.Field("snapshotpolicy_id")))

		diskRetentions := []struct {
			DiskId         string
			RetentionCount int
		}{}
		err = q.All(&diskRetentions)
		if err != nil {
			log.Errorf("Cleanup snapshots job fetch disk retentions failed %s", err)
			return
		}

		diskRetentionMap := map[string]int{}
		for i := range diskRetentions {
			if _, ok := diskRetentionMap[diskRetentions[i].DiskId]; !ok {
				diskRetentionMap[diskRetentions[i].DiskId] = diskRetentions[i].RetentionCount
			}
			// 取最小保留个数
			if diskRetentionMap[diskRetentions[i].DiskId] > diskRetentions[i].RetentionCount {
				diskRetentionMap[diskRetentions[i].DiskId] = diskRetentions[i].RetentionCount
			}
		}

		for diskId, retentionCnt := range diskRetentionMap {
			if cnt, ok := diskCount[diskId]; ok && cnt > retentionCnt {
				manager.startCleanupRetentionCount(ctx, userCred, diskId, cnt-retentionCnt)
			}
		}
	}
}

func (manager *SSnapshotManager) startCleanupRetentionCount(ctx context.Context, userCred mcclient.TokenCredential, diskId string, cnt int) error {
	q := manager.Query().Equals("disk_id", diskId).Equals("created_by", api.SNAPSHOT_AUTO).Asc("created_at").Limit(cnt)
	snapshots := []SSnapshot{}
	err := db.FetchModelObjects(manager, q, &snapshots)
	if err != nil {
		return errors.Wrapf(err, "FetchModelObjects")
	}
	for i := range snapshots {
		snapshots[i].StartSnapshotDeleteTask(ctx, userCred, false, "", 0, 0)
	}
	return nil
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

func (self *SSnapshot) GetQuotaKeys() quotas.IQuotaKeys {
	region, _ := self.GetRegion()
	return fetchRegionalQuotaKeys(
		rbacscope.ScopeProject,
		self.GetOwnerId(),
		region,
		self.GetCloudprovider(),
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

func (manager *SSnapshotManager) DataCleaning(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	err := dataCleaning(manager.TableSpec().Name())
	if err != nil {
		log.Errorf("*************  %s:dataCleaning error:%s  ************", manager.TableSpec().Name(), err.Error())
	}
}

func dataCleaning(tableName string) error {
	if options.Options.KeepDeletedSnapshotDays <= 0 {
		return nil
	}

	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"delete from %s  where deleted = 1 and deleted_at < ?",
			tableName,
		), time.Now().AddDate(0, 0, -options.Options.KeepDeletedSnapshotDays),
	)
	if err != nil {
		return errors.Wrapf(err, "unable to delete expired data in %q", tableName)
	}
	log.Infof("delete expired data in %q successfully", tableName)
	return nil
}
