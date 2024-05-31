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

type SStoragecacheResourceBase struct {
	// 存储缓存Id
	StoragecacheId string `width:"36" charset:"ascii" nullable:"true" list:"user" index:"true" create:"optional"`
}

type SStoragecacheResourceBaseManager struct {
	SManagedResourceBaseManager
}

func ValidateStoragecacheResourceInput(ctx context.Context, userCred mcclient.TokenCredential, query api.StoragecacheResourceInput) (*SStoragecache, api.StoragecacheResourceInput, error) {
	scObj, err := StoragecacheManager.FetchByIdOrName(ctx, userCred, query.StoragecacheId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, query, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", StorageManager.Keyword(), query.StoragecacheId)
		} else {
			return nil, query, errors.Wrap(err, "StorageManager.FetchByIdOrName")
		}
	}
	query.StoragecacheId = scObj.GetId()
	return scObj.(*SStoragecache), query, nil
}

func (self *SStoragecacheResourceBase) GetStoragecache() *SStoragecache {
	obj, err := StoragecacheManager.FetchById(self.StoragecacheId)
	if err != nil {
		log.Errorf("fail to fetch storagecache by id: %s: %s", self.StoragecacheId, err)
		return nil
	}
	return obj.(*SStoragecache)
}

func (manager *SStoragecacheResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.StoragecacheResourceInfo {
	rows := make([]api.StoragecacheResourceInfo, len(objs))
	storagecacheIds := make([]string, len(objs))
	for i := range objs {
		var base *SStoragecacheResourceBase
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil {
			storagecacheIds[i] = base.StoragecacheId
		}
	}

	storagecaches := make(map[string]SStoragecache)
	err := db.FetchStandaloneObjectsByIds(StoragecacheManager, storagecacheIds, storagecaches)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return nil
	}

	storageMap := make(map[string][]api.StorageInfo, 0)
	{
		q := StorageManager.Query("id", "name", "storage_type", "medium_type", "storagecache_id", "zone_id").In("storagecache_id", storagecacheIds)
		zones := ZoneManager.Query().SubQuery()
		q = q.Join(zones, sqlchemy.Equals(q.Field("zone_id"), zones.Field("id")))
		q = q.AppendField(zones.Field("name").Label("zone"))

		storages := make([]struct {
			api.StorageInfo
			StoragecacheId string `json:"storagecache_id"`
		}, 0)
		err := q.All(&storages)
		if err != nil {
			log.Errorf("Storage Info Query query fail %s", err)
		} else {
			for _, si := range storages {
				storageMap[si.StoragecacheId] = append(storageMap[si.StoragecacheId], si.StorageInfo)
			}
		}
	}

	managerList := make([]interface{}, len(rows))

	for i := range rows {
		rows[i] = api.StoragecacheResourceInfo{}
		if _, ok := storagecaches[storagecacheIds[i]]; ok {
			storagecache := storagecaches[storagecacheIds[i]]
			rows[i].Storagecache = storagecache.Name
			rows[i].ManagerId = storagecache.ManagerId
		}
		if info, ok := storageMap[storagecacheIds[i]]; ok {
			rows[i].StorageInfo = info
			for j := range info {
				rows[i].Storages = append(rows[i].Storages, info[j].Name)
			}
		}
		managerList[i] = &SManagedResourceBase{rows[i].ManagerId}
	}

	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, managerList, fields, isList)
	for i := range rows {
		rows[i].ManagedResourceInfo = managerRows[i]
	}

	return rows
}

func (manager *SStoragecacheResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.StoragecacheFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.StoragecacheId) > 0 {
		scObj, _, err := ValidateStoragecacheResourceInput(ctx, userCred, query.StoragecacheResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateStoragecacheResourceInput")
		}
		q = q.Equals("storagecache_id", scObj.GetId())
	}
	subq := StoragecacheManager.Query("id").Snapshot()
	subq, err := manager.SManagedResourceBaseManager.ListItemFilter(ctx, subq, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	if subq.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("storagecache_id"), subq.SubQuery()))
	}
	return q, nil
}

func (manager *SStoragecacheResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "storagecache":
		storagecaches := StoragecacheManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(storagecaches.Field("name", field))
		q = q.Join(storagecaches, sqlchemy.Equals(q.Field("storagecache_id"), storagecaches.Field("id")))
		q.GroupBy(storagecaches.Field("name"))
		return q, nil
	case "manager", "account", "provider", "brand":
		storages := StorageManager.Query("id", "manager_id").SubQuery()
		q = q.LeftJoin(storages, sqlchemy.Equals(q.Field("storage_id"), storages.Field("id")))
		return manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	}
	return q, httperrors.ErrNotFound
}

func (manager *SStoragecacheResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.StoragecacheFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := StorageManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("storage_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SStoragecacheResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.StoragecacheFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	storagecacheQ := StoragecacheManager.Query().SubQuery()
	q = q.LeftJoin(storagecacheQ, sqlchemy.Equals(joinField, storagecacheQ.Field("id")))
	q = q.AppendField(storagecacheQ.Field("name").Label("storagecache"))
	orders = append(orders, query.OrderByStoragecache)
	fields = append(fields, subq.Field("storagecache"))
	q, orders, fields = manager.SManagedResourceBaseManager.GetOrderBySubQuery(q, subq, storagecacheQ.Field("manager_id"), userCred, query.ManagedResourceListInput, orders, fields)
	return q, orders, fields
}

func (manager *SStoragecacheResourceBaseManager) GetOrderByFields(query api.StoragecacheFilterListInput) []string {
	fields := make([]string, 0)
	managerFields := manager.SManagedResourceBaseManager.GetOrderByFields(query.ManagedResourceListInput)
	fields = append(fields, managerFields...)
	fields = append(fields, query.OrderByStoragecache)
	return fields
}

func (manager *SStoragecacheResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		var err error
		subq := StoragecacheManager.Query("id", "name", "manager_id").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("storagecache_id"), subq.Field("id")))
		if keys.Contains("storagecache") {
			q = q.AppendField(subq.Field("name", "storagecache"))
		}
		if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
			q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
			}
		}
	}
	return q, nil
}

func (manager *SStoragecacheResourceBaseManager) GetExportKeys() []string {
	keys := []string{"storagecache"}
	keys = append(keys, manager.SManagedResourceBaseManager.GetExportKeys()...)
	return keys
}

func (model *SStoragecacheResourceBase) GetChangeOwnerCandidateDomainIds() []string {
	sc := model.GetStoragecache()
	if sc != nil {
		return sc.GetChangeOwnerCandidateDomainIds()
	}
	return nil
}
