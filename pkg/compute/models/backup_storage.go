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
	"net/url"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=backupstorage
// +onecloud:swagger-gen-model-plural=backupstorages
type SBackupStorageManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
}

type SBackupStorage struct {
	db.SEnabledStatusInfrasResourceBase

	AccessInfo  *api.SBackupStorageAccessInfo
	StorageType api.TBackupStorageType `width:"32" charset:"ascii" nullable:"false" list:"user" create:"domain_required"`

	CapacityMb int `nullable:"false" list:"user" update:"domain" create:"domain_optional"`
}

var BackupStorageManager *SBackupStorageManager

func init() {
	BackupStorageManager = &SBackupStorageManager{
		SEnabledStatusInfrasResourceBaseManager: db.NewEnabledStatusInfrasResourceBaseManager(
			SBackupStorage{},
			"backupstorages_tbl",
			"backupstorage",
			"backupstorages",
		),
	}
	BackupStorageManager.SetVirtualObject(BackupStorageManager)
}

func (bs *SBackupStorageManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.BackupStorageCreateInput) (api.BackupStorageCreateInput, error) {
	var err error
	input.EnabledStatusInfrasResourceBaseCreateInput, err = bs.SEnabledStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	if !utils.IsInArray(input.StorageType, []string{string(api.BACKUPSTORAGE_TYPE_NFS), string(api.BACKUPSTORAGE_TYPE_OBJECT_STORAGE)}) {
		return input, httperrors.NewInputParameterError("Invalid storage type %s", input.StorageType)
	}
	switch input.StorageType {
	case string(api.BACKUPSTORAGE_TYPE_NFS):
		if input.NfsHost == "" {
			return input, httperrors.NewInputParameterError("nfs_host is required when storage type is nfs")
		}
		if input.NfsSharedDir == "" {
			return input, httperrors.NewInputParameterError("nfs_shared_dir is required when storage type is nfs")
		}
	case string(api.BACKUPSTORAGE_TYPE_OBJECT_STORAGE):
		if input.ObjectBucketUrl == "" {
			return input, httperrors.NewInputParameterError("object_bucket_url is required when storage type is object")
		}
		_, err := url.Parse(input.ObjectBucketUrl)
		if err != nil {
			return input, httperrors.NewInputParameterError("invalid object_bucket_url %s: %s", input.ObjectBucketUrl, err)
		}
		if input.ObjectAccessKey == "" {
			return input, httperrors.NewInputParameterError("object_access_key is required when storage type is object")
		}
		if input.ObjectSecret == "" {
			return input, httperrors.NewInputParameterError("object_secret is required when storage type is object")
		}
	}
	return input, nil
}

func (bs *SBackupStorage) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	bs.SetEnabled(true)
	input := api.BackupStorageCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return errors.Wrap(err, "Unmarshal BackupStorageCreateInput")
	}
	// nfsHost, _ := data.GetString("nfs_host")
	// nfsSharedDir, _ := data.GetString("nfs_shared_dir")
	bs.Status = api.BACKUPSTORAGE_STATUS_ONLINE
	bs.AccessInfo = &api.SBackupStorageAccessInfo{
		NfsHost:      input.NfsHost,
		NfsSharedDir: input.NfsSharedDir,

		ObjectBucketUrl: input.ObjectBucketUrl,
		ObjectAccessKey: input.ObjectAccessKey,
		ObjectSecret:    input.ObjectSecret,
	}
	return bs.SEnabledStatusInfrasResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (bs *SBackupStorage) BackupCount() (int, error) {
	return DiskBackupManager.Query().Equals("backup_storage_id", bs.GetId()).CountWithError()
}

func (bs *SBackupStorage) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	cnt, err := bs.BackupCount()
	if err != nil {
		return httperrors.NewInternalServerError("BackupCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("storage has been used")
	}
	return bs.SEnabledStatusInfrasResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (bs *SBackupStorage) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	bs.SEnabledStatusInfrasResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	bs.SetStatus(ctx, userCred, api.BACKUPSTORAGE_STATUS_OFFLINE, "")
	if bs.StorageType == api.BACKUPSTORAGE_TYPE_OBJECT_STORAGE {
		err := bs.saveObjectSecret(bs.AccessInfo.ObjectSecret)
		if err != nil {
			log.Errorf("convert object secret fail %s", err)
		}
	}
	err := bs.startSyncStatusTask(ctx, userCred, "")
	if err != nil {
		log.Errorf("unable to sync backup storage status")
	}
}

func (bs *SBackupStorage) startSyncStatusTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return StartResourceSyncStatusTask(ctx, userCred, bs, "BackupStorageSyncstatusTask", parentTaskId)
}

func (bs *SBackupStorage) saveObjectSecret(secret string) error {
	sec, err := utils.EncryptAESBase64(bs.Id, secret)
	if err != nil {
		return errors.Wrap(err, "EncryptAESBase64")
	}
	accessInfo := *bs.AccessInfo
	accessInfo.ObjectSecret = sec
	_, err = db.Update(bs, func() error {
		bs.AccessInfo = &accessInfo
		return nil
	})
	return errors.Wrap(err, "Update")
}

func (bs *SBackupStorage) getMoreDetails(ctx context.Context, out api.BackupStorageDetails) api.BackupStorageDetails {
	out.NfsHost = bs.AccessInfo.NfsHost
	out.NfsSharedDir = bs.AccessInfo.NfsSharedDir
	out.ObjectBucketUrl = bs.AccessInfo.ObjectBucketUrl
	out.ObjectAccessKey = bs.AccessInfo.ObjectAccessKey
	out.ObjectSignVer = bs.AccessInfo.ObjectSignVer
	// should not return secret
	out.ObjectSecret = "" // bs.AccessInfo.ObjectSecret
	return out
}

func (bm *SBackupStorageManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []api.BackupStorageDetails {
	rows := make([]api.BackupStorageDetails, len(objs))
	esiRows := bm.SEnabledStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i].EnabledStatusInfrasResourceBaseDetails = esiRows[i]
		bs := objs[i].(*SBackupStorage)
		rows[i] = bs.getMoreDetails(ctx, rows[i])
	}
	return rows
}

func (self *SBackupStorage) GetRegionDriver() IRegionDriver {
	return GetRegionDriver(api.CLOUD_PROVIDER_ONECLOUD)
}

func (bm *SBackupStorageManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.BackupStorageListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = bm.SEnabledStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, input.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemFilter")
	}
	if len(input.ServerId) > 0 {
		serverObj, err := GuestManager.FetchByIdOrName(ctx, userCred, input.ServerId)
		if err != nil {
			if errors.Cause(err) == errors.ErrNotFound {
				return nil, httperrors.NewResourceNotFoundError2(GuestManager.Keyword(), input.ServerId)
			} else {
				return nil, errors.Wrap(err, "GuestManager.FetchByIdOrName")
			}
		}
		server := serverObj.(*SGuest)
		input.ServerId = server.Id
		hostIds, err := server.getDisksCandidateHostIds()
		if err != nil {
			return nil, errors.Wrap(err, "getDisksCandidateHostIds")
		}
		q = bm.filterByCandidateHostIds(q, hostIds)
	}
	if len(input.DiskId) > 0 {
		diskObj, err := DiskManager.FetchByIdOrName(ctx, userCred, input.DiskId)
		if err != nil {
			if errors.Cause(err) == errors.ErrNotFound {
				return nil, httperrors.NewResourceNotFoundError2(DiskManager.Keyword(), input.DiskId)
			} else {
				return nil, errors.Wrap(err, "DiskManager.FetchByIdOrName")
			}
		}
		disk := diskObj.(*SDisk)
		input.DiskId = disk.Id
		hostIds, err := disk.getCandidateHostIds()
		if err != nil {
			return nil, errors.Wrap(err, "getDisksCandidateHostIds")
		}
		q = bm.filterByCandidateHostIds(q, hostIds)
	}
	return q, nil
}

func (bm *SBackupStorageManager) filterByCandidateHostIds(q *sqlchemy.SQuery, candidateIds []string) *sqlchemy.SQuery {
	hbsSubQ := HostBackupstorageManager.Query().SubQuery()

	q = q.LeftJoin(hbsSubQ, sqlchemy.Equals(q.Field("id"), hbsSubQ.Field("backupstorage_id")))
	q = q.Filter(sqlchemy.OR(
		sqlchemy.IsNull(hbsSubQ.Field("host_id")),
		sqlchemy.In(hbsSubQ.Field("host_id"), candidateIds),
	))

	return q
}

func (bs *SBackupStorage) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DiskBackupSyncstatusInput) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(bs, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("Backup has %d task active, can't sync status", count)
	}

	return nil, bs.startSyncStatusTask(ctx, userCred, "")
}

func (bs *SBackupStorage) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BackupStorageUpdateInput,
) (api.BackupStorageUpdateInput, error) {
	var err error
	if len(input.Name) > 0 {
		err := isValidBucketName(input.Name)
		if err != nil {
			return input, httperrors.NewInputParameterError("invalid bucket name(%s): %s", input.Name, err)
		}
	}
	input.EnabledStatusInfrasResourceBaseUpdateInput, err = bs.SEnabledStatusInfrasResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusInfrasResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SSharableVirtualResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (bs *SBackupStorage) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	bs.SEnabledStatusInfrasResourceBase.PostUpdate(ctx, userCred, query, data)
	input := api.BackupStorageUpdateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		log.Errorf("SBackupStorage.PostUpdate Unmarshal data %s fail %s", data, err)
		return
	}
	// update accessinfo
	accessInfoChanged := false
	accessInfo := *bs.AccessInfo
	switch bs.StorageType {
	case api.BACKUPSTORAGE_TYPE_NFS:
		if len(input.NfsHost) > 0 {
			accessInfo.NfsHost = input.NfsHost
			accessInfoChanged = true
		}
		if len(input.NfsSharedDir) > 0 {
			accessInfo.NfsSharedDir = input.NfsSharedDir
			accessInfoChanged = true
		}
	case api.BACKUPSTORAGE_TYPE_OBJECT_STORAGE:
		if len(input.ObjectBucketUrl) > 0 {
			accessInfo.ObjectBucketUrl = input.ObjectBucketUrl
			accessInfoChanged = true
		}
		if len(input.ObjectAccessKey) > 0 {
			accessInfo.ObjectAccessKey = input.ObjectAccessKey
			accessInfoChanged = true
		}
		if len(input.ObjectSecret) > 0 {
			sec, err := utils.EncryptAESBase64(bs.Id, input.ObjectSecret)
			if err != nil {
				log.Errorf("EncryptAESBase64 fail %s", err)
				return
			}
			accessInfo.ObjectSecret = sec
			accessInfoChanged = true
		}
		if input.ObjectSignVer != accessInfo.ObjectSignVer {
			accessInfo.ObjectSignVer = input.ObjectSignVer
			accessInfoChanged = true
		}
	}
	if accessInfoChanged {
		_, err = db.Update(bs, func() error {
			bs.AccessInfo = &accessInfo
			return nil
		})
		if err != nil {
			log.Errorf("update fail %s", err)
		} else {
			err := StartResourceSyncStatusTask(ctx, userCred, bs, "BackupStorageSyncstatusTask", "")
			if err != nil {
				log.Errorf("unable to sync backup storage status")
			}
		}
	}
}

func (bs *SBackupStorage) GetAccessInfo() (*api.SBackupStorageAccessInfo, error) {
	accessInfo := *bs.AccessInfo
	switch bs.StorageType {
	case api.BACKUPSTORAGE_TYPE_OBJECT_STORAGE:
		secret, err := utils.DescryptAESBase64(bs.Id, accessInfo.ObjectSecret)
		if err != nil {
			return nil, errors.Wrap(err, "DescryptAESBase64")
		}
		accessInfo.ObjectSecret = secret
	}
	return &accessInfo, nil
}

func (bs *SBackupStorage) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("Host delete do nothing")
	// cleanup hostbackupstorage
	hbs, err := HostBackupstorageManager.GetBackupStoragesByBackup(bs.Id)
	if err != nil {
		return errors.Wrap(err, "GetBackupStoragesByBackup")
	}
	var errs []error
	for i := range hbs {
		err := hbs[i].Detach(ctx, userCred)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Detach %s %s", hbs[i].HostId, hbs[i].BackupstorageId))
		}
	}
	if len(errs) > 0 {
		return errors.NewAggregate(errs)
	}
	return bs.SEnabledStatusInfrasResourceBase.Delete(ctx, userCred)
}
