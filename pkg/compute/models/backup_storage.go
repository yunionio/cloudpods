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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
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

func (bm *SBackupStorageManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.BackupStorageListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = bm.SEnabledStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, input.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}
