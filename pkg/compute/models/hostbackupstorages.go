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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SHostBackupstorageManager struct {
	SHostJointsManager
	SBackupstorageResourceBaseManager
}

var HostBackupstorageManager *SHostBackupstorageManager

func init() {
	db.InitManager(func() {
		HostBackupstorageManager = &SHostBackupstorageManager{
			SHostJointsManager: NewHostJointsManager(
				"host_id",
				SHostBackupstorage{},
				"hostbackupstorages_tbl",
				"hostbackupstorage",
				"hostbackupstorages",
				BackupStorageManager,
			),
		}
		HostBackupstorageManager.SetVirtualObject(HostBackupstorageManager)
		HostBackupstorageManager.TableSpec().AddIndex(false, "host_id", "backupstorage_id")
	})
}

// +onecloud:model-api-gen
type SHostBackupstorage struct {
	SHostJointsBase

	// 宿主机Id
	HostId string `width:"36" charset:"ascii" nullable:"false" list:"domain" create:"required" json:"host_id"`
	// 存储Id
	BackupstorageId string `width:"36" charset:"ascii" nullable:"false" list:"domain" create:"required" json:"backupstorage_id" index:"true"`
}

func (manager *SHostBackupstorageManager) GetMasterFieldName() string {
	return "host_id"
}

func (manager *SHostBackupstorageManager) GetSlaveFieldName() string {
	return "backupstorage_id"
}

func (manager *SHostBackupstorageManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.HostBackupstorageDetails {
	rows := make([]api.HostBackupstorageDetails, len(objs))

	hostRows := manager.SHostJointsManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	backupStorageIds := make([]string, len(rows))

	for i := range rows {
		rows[i] = api.HostBackupstorageDetails{
			HostJointResourceDetails: hostRows[i],
		}
		backupStorageIds[i] = objs[i].(*SHostBackupstorage).BackupstorageId
	}

	backupStorages := make(map[string]SBackupStorage)
	err := db.FetchStandaloneObjectsByIds(BackupStorageManager, backupStorageIds, &backupStorages)
	if err != nil {
		log.Errorf("db.FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	for i := range rows {
		if bs, ok := backupStorages[backupStorageIds[i]]; ok {
			rows[i] = objs[i].(*SHostBackupstorage).getExtraDetails(bs, rows[i])
		}
	}

	return rows
}

func (hb *SHostBackupstorage) GetHost() *SHost {
	host, _ := HostManager.FetchById(hb.HostId)
	if host != nil {
		return host.(*SHost)
	}
	return nil
}

func (hb *SHostBackupstorage) GetBackupStorage() *SBackupStorage {
	bs, err := BackupStorageManager.FetchById(hb.BackupstorageId)
	if err != nil {
		log.Errorf("Hoststorage fetch storage %q error: %v", hb.BackupstorageId, err)
	}
	if bs != nil {
		return bs.(*SBackupStorage)
	}
	return nil
}

func (manager *SHostBackupstorageManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.HostBackupstorageCreateInput) (api.HostBackupstorageCreateInput, error) {
	{
		backStorageObj, err := validators.ValidateModel(ctx, userCred, BackupStorageManager, &input.BackupstorageId)
		if err != nil {
			return input, err
		}
		backupStorage := backStorageObj.(*SBackupStorage)
		input.BackupstorageId = backupStorage.Id
	}
	{
		hostObj, err := validators.ValidateModel(ctx, userCred, HostManager, &input.HostId)
		if err != nil {
			return input, err
		}
		host := hostObj.(*SHost)
		input.HostId = host.Id
	}
	{
		hs, err := manager.GetBackupStoragesByHost(input.HostId, api.BACKUPSTORAGE_TYPE_NFS)
		if err != nil {
			return input, errors.Wrap(err, "GetBackupStoragesByHost")
		}
		if len(hs) >= 1 {
			return input, errors.Wrapf(httperrors.ErrResourceBusy, "host %s has been attached to a NFS backupstorage", input.HostId)
		}
	}
	var err error
	input.JoinResourceBaseCreateInput, err = manager.SJointResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.JoinResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (hb *SHostBackupstorage) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	hb.SHostJointsBase.PostCreate(ctx, userCred, ownerId, query, data)

	bs := hb.GetBackupStorage()
	if bs != nil {
		bs.startSyncStatusTask(ctx, userCred, "")
	}
}

func (hb *SHostBackupstorage) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	hb.SHostJointsBase.PostDelete(ctx, userCred)

	bs := hb.GetBackupStorage()
	if bs != nil {
		bs.startSyncStatusTask(ctx, userCred, "")
	}
}

func (hb *SHostBackupstorage) getExtraDetails(storage SBackupStorage, out api.HostBackupstorageDetails) api.HostBackupstorageDetails {
	out.Backupstorage = storage.Name
	out.CapacityMb = int64(storage.CapacityMb)
	out.StorageType = storage.StorageType
	out.Enabled = storage.Enabled.Bool()
	return out
}

func (hb *SHostBackupstorage) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	return hb.SHostJointsBase.ValidateDeleteCondition(ctx, nil)
}

func (hb *SHostBackupstorage) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, hb)
}

func (hb *SHostBackupstorage) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, hb)
}

func (manager *SHostBackupstorageManager) GetBackupStoragesByBackup(backupStorageId string) ([]SHostBackupstorage, error) {
	hoststorages := make([]SHostBackupstorage, 0)
	q := HostBackupstorageManager.Query().Equals("backupstorage_id", backupStorageId)
	err := db.FetchModelObjects(manager, q, &hoststorages)
	if err != nil {
		return nil, err
	}
	return hoststorages, nil
}

func (manager *SHostBackupstorageManager) GetBackupStoragesByHost(hostId string, backupType api.TBackupStorageType) ([]SHostBackupstorage, error) {
	hoststorages := make([]SHostBackupstorage, 0)
	backups := BackupStorageManager.Query().Equals("storage_type", backupType).SubQuery()
	q := HostBackupstorageManager.Query().Equals("host_id", hostId)
	q = q.Join(backups, sqlchemy.Equals(q.Field("backupstorage_id"), backups.Field("id")))
	err := db.FetchModelObjects(manager, q, &hoststorages)
	if err != nil {
		return nil, err
	}
	return hoststorages, nil
}

func (manager *SHostBackupstorageManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.HostBackupstorageListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SHostJointsManager.ListItemFilter(ctx, q, userCred, query.HostJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SHostResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SBackupstorageResourceBaseManager.ListItemFilter(ctx, q, userCred, query.BackupstorageFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SBackupstorageResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SHostBackupstorageManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.HostBackupstorageListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SHostJointsManager.OrderByExtraFields(ctx, q, userCred, query.HostJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SHostResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SBackupstorageResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.BackupstorageFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SBackupstorageResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SHostBackupstorageManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SHostJointsManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SHostJointsManager.ListItemExportKeys")
	}

	return q, nil
}
