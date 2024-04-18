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
	"reflect"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SBackupStorageManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
}

type SBackupStorage struct {
	db.SEnabledStatusInfrasResourceBase

	AccessInfo  *SBackupStorageAccessInfo
	StorageType string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"domain_required"`
	CapacityMb  int    `nullable:"false" list:"user" update:"domain" create:"domain_required"`
}

var BackupStorageManager *SBackupStorageManager

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SBackupStorageAccessInfo{}), func() gotypes.ISerializable {
		return &SBackupStorageAccessInfo{}
	})
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

type SBackupStorageAccessInfo struct {
	NfsHost      string `json:"nfs_host"`
	NfsSharedDir string `json:"nfs_shared_dir"`
}

func (ba *SBackupStorageAccessInfo) String() string {
	return jsonutils.Marshal(ba).String()
}

func (ba *SBackupStorageAccessInfo) IsZero() bool {
	return ba == nil
}

func (bs *SBackupStorageManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.BackupStorageCreateInput) (api.BackupStorageCreateInput, error) {
	var err error
	input.EnabledStatusInfrasResourceBaseCreateInput, err = bs.SEnabledStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	if !utils.IsInStringArray(input.StorageType, []string{api.BACKUPSTORAGE_TYPE_NFS}) {
		return input, httperrors.NewInputParameterError("Invalid storage type %s", input.StorageType)
	}
	switch input.StorageType {
	case api.BACKUPSTORAGE_TYPE_NFS:
		if input.NfsHost == "" {
			return input, httperrors.NewInputParameterError("nfs_host is required when storage type is nfs")
		}
		if input.NfsSharedDir == "" {
			return input, httperrors.NewInputParameterError("nfs_shared_dir is required when storage type is nfs")
		}
	}
	return input, nil
}

func (bs *SBackupStorage) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	bs.SetEnabled(true)
	nfsHost, _ := data.GetString("nfs_host")
	nfsSharedDir, _ := data.GetString("nfs_shared_dir")
	bs.Status = api.BACKUPSTORAGE_STATUS_ONLINE
	bs.AccessInfo = &SBackupStorageAccessInfo{
		NfsHost:      nfsHost,
		NfsSharedDir: nfsSharedDir,
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
		return httperrors.NewNotEmptyError("storage has backup")
	}
	return bs.SEnabledStatusInfrasResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (bs *SBackupStorage) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	bs.SEnabledStatusInfrasResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	bs.SetStatus(userCred, api.BACKUPSTORAGE_STATUS_OFFLINE, "")
	err := bs.startSyncStatusTask(ctx, userCred, "")
	if err != nil {
		log.Errorf("unable to sync backup storage status")
	}
}

func (bs *SBackupStorage) startSyncStatusTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return StartResourceSyncStatusTask(ctx, userCred, bs, "BackupStorageSyncstatusTask", parentTaskId)
}

func (bs *SBackupStorage) getMoreDetails(ctx context.Context, out api.BackupStorageDetails) api.BackupStorageDetails {
	out.NfsHost = bs.AccessInfo.NfsHost
	out.NfsSharedDir = bs.AccessInfo.NfsSharedDir
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
		serverObj, err := GuestManager.FetchByIdOrName(userCred, input.ServerId)
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
		diskObj, err := DiskManager.FetchByIdOrName(userCred, input.DiskId)
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
