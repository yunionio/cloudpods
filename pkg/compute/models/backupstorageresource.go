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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SBackupstorageResourceBase struct {
	BackupstorageId string `width:"36" charset:"ascii" nullable:"true" list:"user" index:"true" create:"optional"`
}

type SBackupstorageResourceBaseManager struct {
}

func ValidateBackupstorageResourceInput(ctx context.Context, userCred mcclient.TokenCredential, query api.BackupstorageResourceInput) (*SBackupStorage, api.BackupstorageResourceInput, error) {
	storageObj, err := BackupStorageManager.FetchByIdOrName(ctx, userCred, query.BackupstorageId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, query, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", BackupStorageManager.Keyword(), query.BackupstorageId)
		} else {
			return nil, query, errors.Wrap(err, "BackupStorageManager.FetchByIdOrName")
		}
	}
	query.BackupstorageId = storageObj.GetId()
	return storageObj.(*SBackupStorage), query, nil
}

func (self *SBackupstorageResourceBase) GetBackupstorage() *SBackupStorage {
	obj, err := BackupStorageManager.FetchById(self.BackupstorageId)
	if err != nil {
		log.Errorf("fail to fetch storage by id: %s: %s", self.BackupstorageId, err)
		return nil
	}
	return obj.(*SBackupStorage)
}

func (manager *SBackupstorageResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.BackupstorageResourceInfo {
	rows := make([]api.BackupstorageResourceInfo, len(objs))
	storageIds := make([]string, len(objs))
	for i := range objs {
		var base *SBackupstorageResourceBase
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil {
			storageIds[i] = base.BackupstorageId
		}
	}

	storages := make(map[string]SBackupStorage)
	err := db.FetchStandaloneObjectsByIds(BackupStorageManager, storageIds, &storages)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return nil
	}

	for i := range rows {
		rows[i] = api.BackupstorageResourceInfo{}
		if _, ok := storages[storageIds[i]]; ok {
			storage := storages[storageIds[i]]
			rows[i].Backupstorage = storage.Name
			rows[i].BackupstorageStatus = storage.Status
			rows[i].BackupstorageType = storage.StorageType
		}
	}

	return rows
}

func (manager *SBackupstorageResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.BackupstorageFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.BackupstorageId) > 0 {
		storageObj, _, err := ValidateBackupstorageResourceInput(ctx, userCred, query.BackupstorageResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateBackupstorageResourceInput")
		}
		q = q.Equals("backupstorage_id", storageObj.GetId())
	}
	return q, nil
}

func (manager *SBackupstorageResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "backupstorage":
		storages := BackupStorageManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(storages.Field("name", field))
		q = q.Join(storages, sqlchemy.Equals(q.Field("backupstorage_id"), storages.Field("id")))
		q.GroupBy(storages.Field("name"))
		return q, nil
	case "storage_type":
		storages := BackupStorageManager.Query(field, "id").Distinct().SubQuery()
		q.AppendField(storages.Field(field))
		q = q.Join(storages, sqlchemy.Equals(q.Field("backupstorage_id"), storages.Field("id")))
		q.GroupBy(storages.Field(field))
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SBackupstorageResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.BackupstorageFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := BackupStorageManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("backupstorage_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SBackupstorageResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.BackupstorageFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	backupStorageQ := BackupStorageManager.Query().SubQuery()
	q = q.LeftJoin(backupStorageQ, sqlchemy.Equals(joinField, backupStorageQ.Field("id")))
	q = q.AppendField(backupStorageQ.Field("name").Label("backupstorage"))
	orders = append(orders, query.OrderByBackupstorage)
	fields = append(fields, subq.Field("backupstorage"))
	return q, orders, fields
}

func (manager *SBackupstorageResourceBaseManager) GetOrderByFields(query api.BackupstorageFilterListInput) []string {
	fields := make([]string, 0)
	fields = append(fields, query.OrderByBackupstorage)
	return fields
}

func (manager *SBackupstorageResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		subq := BackupStorageManager.Query("id", "name").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("backupstorage_id"), subq.Field("id")))
		if keys.Contains("backupstorage") {
			q = q.AppendField(subq.Field("name", "backupstorage"))
		}
	}
	return q, nil
}

func (manager *SBackupstorageResourceBaseManager) GetExportKeys() []string {
	keys := []string{"backupstorage"}
	return keys
}

func (model *SBackupstorageResourceBase) GetChangeOwnerCandidateDomainIds() []string {
	storage := model.GetBackupstorage()
	if storage != nil {
		return storage.GetChangeOwnerCandidateDomainIds()
	}
	return nil
}
