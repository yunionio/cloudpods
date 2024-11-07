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
	"reflect"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=diskbackup
// +onecloud:swagger-gen-model-plural=diskbackups
type SDiskBackupManager struct {
	db.SVirtualResourceBaseManager
	SDiskResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
	db.SMultiArchResourceBaseManager
	db.SEncryptedResourceManager
}

type SDiskBackup struct {
	db.SVirtualResourceBase

	SManagedResourceBase
	SCloudregionResourceBase `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	db.SMultiArchResourceBase

	db.SEncryptedResource

	DiskId          string `width:"36" charset:"ascii" nullable:"true" create:"required" list:"user" index:"true"`
	BackupStorageId string `width:"36" charset:"ascii" nullable:"true" create:"required" list:"user" index:"true"`
	StorageId       string `width:"36" charset:"ascii" nullable:"true" list:"user"`

	// 备份大小
	SizeMb     int    `nullable:"false" list:"user" create:"optional"`
	DiskSizeMb int    `nullable:"false" list:"user" create:"optional"`
	DiskType   string `width:"32" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	// 操作系统类型
	OsType     string `width:"32" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	DiskConfig *SBackupDiskConfig
}

var DiskBackupManager *SDiskBackupManager

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SBackupDiskConfig{}), func() gotypes.ISerializable {
		return &SBackupDiskConfig{}
	})
	DiskBackupManager = &SDiskBackupManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SDiskBackup{},
			"diskbackups_tbl",
			"diskbackup",
			"diskbackups",
		),
	}
	DiskBackupManager.SetVirtualObject(DiskBackupManager)
}

type SBackupDiskConfig struct {
	api.DiskConfig
	Name        string
	BackupAsTar *api.DiskBackupAsTarInput
}

func (dc *SBackupDiskConfig) String() string {
	return jsonutils.Marshal(dc).String()
}

func (dc *SBackupDiskConfig) IsZero() bool {
	return dc == nil
}

func (dm *SDiskBackupManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.DiskBackupListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = dm.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = dm.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = dm.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, input.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	q, err = dm.SMultiArchResourceBaseManager.ListItemFilter(ctx, q, userCred, input.MultiArchResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SMultiArchResourceBaseManager.ListItemFilter")
	}
	if input.DiskId != "" {
		q = q.Equals("disk_id", input.DiskId)
	}
	if input.BackupStorageId != "" {
		q = q.Equals("backup_storage_id", input.BackupStorageId)
	}
	if input.IsInstanceBackup != nil {
		insjsq := InstanceBackupJointManager.Query().SubQuery()
		if !*input.IsInstanceBackup {
			q = q.LeftJoin(insjsq, sqlchemy.Equals(q.Field("id"), insjsq.Field("disk_backup_id"))).
				Filter(sqlchemy.IsNull(insjsq.Field("disk_backup_id")))
		} else {
			q = q.Join(insjsq, sqlchemy.Equals(q.Field("id"), insjsq.Field("disk_backup_id")))
		}
	}
	return q, nil
}

func (self *SDiskBackup) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if self.Status == api.BACKUP_STATUS_DELETING {
		return httperrors.NewBadRequestError("Cannot delete disk backup in status %s", self.Status)
	}
	is, err := InstanceBackupJointManager.IsSubBackup(self.Id)
	if err != nil {
		return err
	}
	if is {
		return httperrors.NewBadRequestError("disk backup referenced by instance backup")
	}
	return nil
}

func (dm *SDiskBackupManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DiskBackupDetails {
	rows := make([]api.DiskBackupDetails, len(objs))
	virtRows := dm.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := dm.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := dm.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	encRows := dm.SEncryptedResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i].VirtualResourceDetails = virtRows[i]
		rows[i].ManagedResourceInfo = manRows[i]
		rows[i].CloudregionResourceInfo = regionRows[i]
		rows[i].EncryptedResourceDetails = encRows[i]
		rows[i] = objs[i].(*SDiskBackup).getMoreDetails(rows[i])
	}
	return rows
}

func (db *SDiskBackup) getMoreDetails(out api.DiskBackupDetails) api.DiskBackupDetails {
	disk, _ := db.GetDisk()
	if disk != nil {
		out.DiskName = disk.Name
	}
	backupStorage, _ := db.GetBackupStorage()
	if backupStorage != nil {
		out.BackupStorageName = backupStorage.GetName()
	}
	if t, _ := InstanceBackupJointManager.IsSubBackup(db.Id); t {
		out.IsSubBackup = true
	}
	return out
}

func (db *SDiskBackup) GetDisk() (*SDisk, error) {
	iDisk, err := DiskManager.FetchById(db.DiskId)
	if err != nil {
		return nil, err
	}
	disk := iDisk.(*SDisk)
	return disk, nil
}

func (db *SDiskBackup) GetStorage() (*SStorage, error) {
	iStorage, err := StorageManager.FetchById(db.StorageId)
	if err != nil {
		return nil, err
	}
	return iStorage.(*SStorage), nil
}

func (db *SDiskBackup) GetBackupStorage() (*SBackupStorage, error) {
	ibs, err := BackupStorageManager.FetchById(db.BackupStorageId)
	if err != nil {
		return nil, err
	}
	return ibs.(*SBackupStorage), nil
}

func (db *SDiskBackup) GetRegionDriver() (IRegionDriver, error) {
	cloudRegion, err := db.GetRegion()
	if err != nil {
		return nil, errors.Wrap(err, "db.GetRegion")
	}
	return cloudRegion.GetDriver(), nil
}

func (dm *SDiskBackupManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.DiskBackupCreateInput,
) (api.DiskBackupCreateInput, error) {
	if input.NeedEncrypt() {
		return input, errors.Wrap(httperrors.ErrInputParameter, "encryption should not be specified")
	}
	if len(input.DiskId) == 0 {
		return input, httperrors.NewMissingParameterError("disk_id")
	}
	if len(input.BackupStorageId) == 0 {
		return input, httperrors.NewMissingParameterError("backup_storage_id")
	}
	// check disk
	_disk, err := validators.ValidateModel(ctx, userCred, DiskManager, &input.DiskId)
	if err != nil {
		return input, err
	}
	disk := _disk.(*SDisk)
	if disk.Status != api.DISK_READY {
		return input, httperrors.NewInvalidStatusError("disk %s status is not %s", disk.Name, api.DISK_READY)
	}
	if len(disk.EncryptKeyId) > 0 {
		input.EncryptKeyId = &disk.EncryptKeyId
		input.EncryptedResourceCreateInput, err = dm.SEncryptedResourceManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EncryptedResourceCreateInput)
		if err != nil {
			return input, errors.Wrap(err, "SEncryptedResourceManager.ValidateCreateData")
		}
	}

	ibs, err := BackupStorageManager.FetchByIdOrName(ctx, userCred, input.BackupStorageId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return input, httperrors.NewResourceNotFoundError2(BackupStorageManager.Keyword(), input.BackupStorageId)
		}
		if errors.Cause(err) == sqlchemy.ErrDuplicateEntry {
			return input, httperrors.NewDuplicateResourceError(BackupStorageManager.Keyword(), input.BackupStorageId)
		}
		return input, httperrors.NewGeneralError(err)
	}
	if err != nil {
		return input, err
	}
	bs := ibs.(*SBackupStorage)
	if bs.Status != api.BACKUPSTORAGE_STATUS_ONLINE {
		return input, httperrors.NewForbiddenError("can't backup guest to backup storage with status %s", bs.Status)
	}
	input.BackupStorageId = bs.GetId()
	storage, err := disk.GetStorage()
	if err != nil {
		return input, errors.Wrapf(err, "unable to get storage of disk %s", disk.GetId())
	}
	input.ManagerId = storage.ManagerId
	region, err := storage.GetRegion()
	if err != nil {
		return input, err
	}
	input.CloudregionId = region.Id

	if input.BackupAsTar != nil {
		if input.BackupAsTar.ContainerId == "" {
			return input, httperrors.NewMissingParameterError("container_id")
		}
		ctr, err := GetContainerManager().FetchByIdOrName(ctx, userCred, input.BackupAsTar.ContainerId)
		if err != nil {
			return input, httperrors.NewNotFoundError("fetch container by %s", input.BackupAsTar.ContainerId)
		}
		input.BackupAsTar.ContainerId = ctr.GetId()
		if err := dm.validateBackupAsTarFiles(input.BackupAsTar.IncludeFiles); err != nil {
			return input, httperrors.NewInputParameterError("validate include_files: %s", err)
		}
		if err := dm.validateBackupAsTarFiles(input.BackupAsTar.ExcludeFiles); err != nil {
			return input, httperrors.NewInputParameterError("validate exclude_files: %s", err)
		}
	}

	return input, nil
}

func (dm *SDiskBackupManager) validateBackupAsTarFiles(paths []string) error {
	for _, p := range paths {
		if strings.HasPrefix(p, "/") {
			return httperrors.NewInputParameterError("%s can't start with /", p)
		}
	}
	return nil
}

func (db *SDiskBackup) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	input := new(api.DiskBackupCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return err
	}
	err := db.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
	if err != nil {
		return err
	}
	diskObj, err := DiskManager.FetchById(db.DiskId)
	if err != nil {
		return errors.Wrap(err, "DiskManager.FetchById")
	}
	disk := diskObj.(*SDisk)
	db.DiskConfig = &SBackupDiskConfig{
		DiskConfig:  *disk.ToDiskConfig(),
		Name:        disk.GetName(),
		BackupAsTar: input.BackupAsTar,
	}
	db.DiskType = disk.DiskType
	db.DiskSizeMb = disk.DiskSize
	db.OsArch = disk.OsArch
	db.StorageId = disk.StorageId
	db.DomainId = disk.DomainId
	db.ProjectId = disk.ProjectId
	return nil
}

func (db *SDiskBackup) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	db.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	disk, err := db.GetDisk()
	if err != nil {
		log.Errorf("unable to GetDisk: %s", err.Error())
	}
	err = disk.InheritTo(ctx, userCred, db)
	if err != nil {
		log.Errorf("unable to inherit from disk %s to backup %s: %s", disk.GetId(), db.GetId(), err.Error())
	}
	db.StartBackupCreateTask(ctx, userCred, nil, "")
}

func (db *SDiskBackup) StartBackupCreateTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "DiskBackupCreateTask", db, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (manager *SDiskBackupManager) ListItemExportKeys(ctx context.Context,
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

func (manager *SDiskBackupManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
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

func (manager *SDiskBackupManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.DiskBackupListInput) (*sqlchemy.SQuery, error) {
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

	if db.NeedOrderQuery([]string{query.OrderByDiskName}) {
		dQ := DiskManager.Query()
		dSQ := dQ.AppendField(dQ.Field("name").Label("disk_name"), dQ.Field("id")).SubQuery()
		q = q.LeftJoin(dSQ, sqlchemy.Equals(dSQ.Field("id"), q.Field("disk_id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(dSQ.Field("disk_name"))
		q = db.OrderByFields(q, []string{query.OrderByDiskName}, []sqlchemy.IQueryField{q.Field("disk_name")})
	}
	return q, nil
}

func (self *SDiskBackup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SDiskBackup) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SDiskBackup) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	forceDelete := jsonutils.QueryBoolean(query, "force", false)
	return self.StartBackupDeleteTask(ctx, userCred, "", forceDelete)
}

func (self *SDiskBackup) StartBackupDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, forceDelete bool) error {
	self.SetStatus(ctx, userCred, api.BACKUP_STATUS_DELETING, "")
	log.Infof("start to delete diskbackup %s and set deleting", self.GetId())
	params := jsonutils.NewDict()
	if forceDelete {
		params.Set("force_delete", jsonutils.JSONTrue)
	}
	task, err := taskman.TaskManager.NewTask(ctx, "DiskBackupDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SDiskBackup) PerformRecovery(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DiskBackupRecoveryInput) (jsonutils.JSONObject, error) {
	if self.Status != api.BACKUP_STATUS_READY {
		return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "cannot recover backup in status %s", self.Status)
	}
	return nil, self.StartRecoveryTask(ctx, userCred, "", input.Name)
}

func (self *SDiskBackup) StartRecoveryTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, diskName string) error {
	self.SetStatus(ctx, userCred, api.BACKUP_STATUS_RECOVERY, "")
	var params *jsonutils.JSONDict
	if diskName != "" {
		params = jsonutils.NewDict()
		params.Set("disk_name", jsonutils.NewString(diskName))
	}
	task, err := taskman.TaskManager.NewTask(ctx, "DiskBackupRecoveryTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (manager *SDiskBackupManager) CreateBackup(ctx context.Context, owner mcclient.IIdentityProvider, diskId, backupStorageId, name string) (*SDiskBackup, error) {
	iDisk, err := DiskManager.FetchById(diskId)
	if err != nil {
		return nil, err
	}
	disk := iDisk.(*SDisk)
	storage, _ := disk.GetStorage()
	backup := &SDiskBackup{}
	backup.SetModelManager(manager, backup)
	backup.ProjectId = owner.GetProjectId()
	backup.DomainId = owner.GetProjectDomainId()
	backup.DiskId = disk.Id

	// inherit encrypt_key_id
	backup.EncryptKeyId = disk.EncryptKeyId

	backup.DiskConfig = &SBackupDiskConfig{
		DiskConfig: *disk.ToDiskConfig(),
		Name:       disk.GetName(),
	}
	backup.DiskType = disk.DiskType
	backup.DiskSizeMb = disk.DiskSize
	backup.OsArch = disk.OsArch
	backup.StorageId = disk.StorageId
	backup.ManagerId = storage.ManagerId
	if cloudregion, _ := storage.GetRegion(); cloudregion != nil {
		backup.CloudregionId = cloudregion.GetId()
	}
	backup.BackupStorageId = backupStorageId
	backup.Name = name
	backup.Status = api.BACKUP_STATUS_CREATING
	err = DiskBackupManager.TableSpec().Insert(ctx, backup)
	if err != nil {
		return nil, err
	}
	return backup, nil
}

func (self *SDiskBackup) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DiskBackupSyncstatusInput) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(self, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("Backup has %d task active, can't sync status", count)
	}

	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "DiskBackupSyncstatusTask", "")
}

func (self *SDiskBackup) PackMetadata() *api.DiskBackupPackMetadata {
	return &api.DiskBackupPackMetadata{
		OsArch:     self.OsArch,
		SizeMb:     self.SizeMb,
		DiskSizeMb: self.DiskSizeMb,
		DiskType:   self.DiskType,
		// 操作系统类型
		OsType: self.OsType,
		DiskConfig: &api.SBackupDiskConfig{
			DiskConfig: self.DiskConfig.DiskConfig,
			Name:       self.DiskConfig.Name,
		},
	}
}

func (manager *SDiskBackupManager) CreateFromPackMetadata(ctx context.Context, owner mcclient.TokenCredential, backupStorageId, id, name string, metadata *api.DiskBackupPackMetadata) (*SDiskBackup, error) {
	backup := &SDiskBackup{}
	backup.SetModelManager(manager, backup)
	backup.ProjectId = owner.GetProjectId()
	backup.DomainId = owner.GetProjectDomainId()
	backup.DiskConfig = &SBackupDiskConfig{
		DiskConfig: metadata.DiskConfig.DiskConfig,
		Name:       metadata.DiskConfig.Name,
	}
	backup.DiskType = metadata.DiskType
	backup.DiskSizeMb = metadata.DiskSizeMb
	backup.SizeMb = metadata.SizeMb
	backup.OsArch = metadata.OsArch
	backup.DiskType = metadata.DiskType
	backup.OsType = metadata.OsType
	backup.CloudregionId = "default"
	backup.BackupStorageId = backupStorageId
	backup.Name = name
	backup.Id = id
	backup.Status = api.BACKUP_STATUS_READY
	err := DiskBackupManager.TableSpec().Insert(ctx, backup)
	if err != nil {
		return nil, err
	}
	return backup, nil
}
