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

type SDiskResourceBase struct {
	DiskId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

type SDiskResourceBaseManager struct {
	SStorageResourceBaseManager
}

func ValidateDiskResourceInput(ctx context.Context, userCred mcclient.TokenCredential, input api.DiskResourceInput) (*SDisk, api.DiskResourceInput, error) {
	diskObj, err := DiskManager.FetchByIdOrName(ctx, userCred, input.DiskId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, input, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", DiskManager.Keyword(), input.DiskId)
		} else {
			return nil, input, errors.Wrap(err, "DiskManager.FetchByIdOrName")
		}
	}
	input.DiskId = diskObj.GetId()
	return diskObj.(*SDisk), input, nil
}

func (self *SDiskResourceBase) GetDisk() (*SDisk, error) {
	obj, err := DiskManager.FetchById(self.DiskId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDisk(%s)", self.DiskId)
	}
	return obj.(*SDisk), nil
}

func (self *SDiskResourceBase) GetStorage() (*SStorage, error) {
	disk, err := self.GetDisk()
	if err != nil {
		return nil, err
	}
	return disk.GetStorage()
}

func (self *SDiskResourceBase) GetZone() (*SZone, error) {
	storage, err := self.GetStorage()
	if err != nil {
		return nil, err
	}
	return storage.GetZone()
}

func (self *SDiskResourceBase) GetRegion() (*SCloudregion, error) {
	storage, err := self.GetStorage()
	if err != nil {
		return nil, errors.Wrapf(err, "GetStorage")
	}
	return storage.GetRegion()
}

func (manager *SDiskResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DiskResourceInfo {
	rows := make([]api.DiskResourceInfo, len(objs))
	diskIds := make([]string, len(objs))
	for i := range objs {
		var base *SDiskResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find DiskResourceBase in object %s", objs[i])
			continue
		}
		diskIds[i] = base.DiskId
	}
	disks := make(map[string]SDisk)
	err := db.FetchStandaloneObjectsByIds(DiskManager, diskIds, disks)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	storageList := make([]interface{}, len(rows))
	for i := range rows {
		rows[i] = api.DiskResourceInfo{}
		if disk, ok := disks[diskIds[i]]; ok {
			rows[i].Disk = disk.Name
			rows[i].StorageId = disk.StorageId
		}
		storageList[i] = &SStorageResourceBase{rows[i].StorageId}
	}

	storageRows := manager.SStorageResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, storageList, fields, isList)
	for i := range rows {
		rows[i].StorageResourceInfo = storageRows[i]
	}
	return rows
}

func (manager *SDiskResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DiskFilterListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	if len(query.DiskId) > 0 {
		diskObj, _, err := ValidateDiskResourceInput(ctx, userCred, query.DiskResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateDiskResourceInput")
		}
		q = q.Equals("disk_id", diskObj.GetId())
	}
	diskQ := DiskManager.Query("id").Snapshot()
	diskQ, err = manager.SStorageResourceBaseManager.ListItemFilter(ctx, diskQ, userCred, query.StorageFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStorageResourceBaseManager.ListItemFilter")
	}
	if diskQ.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("disk_id"), diskQ.SubQuery()))
	}
	return q, nil
}

func (manager *SDiskResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "disk":
		diskQuery := DiskManager.Query("name", "id").SubQuery()
		q = q.AppendField(diskQuery.Field("name", field)).Distinct()
		q = q.Join(diskQuery, sqlchemy.Equals(q.Field("disk_id"), diskQuery.Field("id")))
		return q, nil
	default:
		disks := DiskManager.Query("id", "storage_id").SubQuery()
		q = q.LeftJoin(disks, sqlchemy.Equals(q.Field("disk_id"), disks.Field("id")))
		q, err := manager.SStorageResourceBaseManager.QueryDistinctExtraField(q, field)
		if err == nil {
			return q, nil
		}
	}
	return q, httperrors.ErrNotFound
}

func (manager *SDiskResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DiskFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := DiskManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("disk_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SDiskResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.DiskFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	diskQ := DiskManager.Query().SubQuery()
	q = q.LeftJoin(diskQ, sqlchemy.Equals(joinField, diskQ.Field("id")))
	q = q.AppendField(diskQ.Field("name").Label("disk"))
	orders = append(orders, query.OrderByDisk)
	fields = append(fields, subq.Field("disk"))
	q, orders, fields = manager.SStorageResourceBaseManager.GetOrderBySubQuery(q, subq, diskQ.Field("storage_id"), userCred, query.StorageFilterListInput, orders, fields)
	return q, orders, fields
}

func (manager *SDiskResourceBaseManager) GetOrderByFields(query api.DiskFilterListInput) []string {
	orders := make([]string, 0)
	storageOrders := manager.SStorageResourceBaseManager.GetOrderByFields(query.StorageFilterListInput)
	orders = append(orders, storageOrders...)
	orders = append(orders, query.OrderByDisk)
	return orders
}

func (manager *SDiskResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		subq := DiskManager.Query("id", "name", "storage_id").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("storage_id"), subq.Field("id")))
		if keys.Contains("disk") {
			q = q.AppendField(subq.Field("name", "disk"))
		}
		if keys.ContainsAny(manager.SStorageResourceBaseManager.GetExportKeys()...) {
			var err error
			q, err = manager.SStorageResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SStorageResourceBaseManager.ListItemExportKeys")
			}
		}
	}
	return q, nil
}

func (manager *SDiskResourceBaseManager) GetExportKeys() []string {
	keys := []string{"disk"}
	keys = append(keys, manager.SStorageResourceBaseManager.GetExportKeys()...)
	return keys
}

func (self *SDiskResourceBase) GetChangeOwnerCandidateDomainIds() []string {
	disk, _ := self.GetDisk()
	if disk != nil {
		return disk.GetChangeOwnerCandidateDomainIds()
	}
	return nil
}
