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
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SIsolatedDeviceResourceBase struct {
	IsolatedDeviceId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

type SIsolatedDeviceResourceBaseManager struct {
}

func (manager *SIsolatedDeviceResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.IsolateDeviceDetails {
	rows := make([]api.IsolateDeviceDetails, len(objs))
	devIds := make([]string, len(objs))
	for i := range objs {
		var base *SIsolatedDeviceResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SIsolatedDeviceResourceBase in object %s", objs[i])
			continue
		}
		devIds[i] = base.IsolatedDeviceId
	}

	devMap := make(map[string]SIsolatedDevice)
	err := db.FetchStandaloneObjectsByIds(IsolatedDeviceManager, devIds, devMap)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	devObjs := make([]interface{}, len(objs))
	devs := make([]SIsolatedDevice, len(objs))
	for i := range rows {
		if dev, ok := devMap[devIds[i]]; ok {
			devs[i] = dev
			if err := jsonutils.Marshal(dev).Unmarshal(&rows[i].SIsolatedDevice); err != nil {
				log.Errorf("unmarshal isolated device %s details failed %s", dev.Id, err)
			}
		}
		devObjs[i] = &devs[i]
	}

	devRows := IsolatedDeviceManager.FetchCustomizeColumns(ctx, userCred, query, devObjs, fields, isList)
	for i := range rows {
		rows[i].StandaloneResourceDetails = devRows[i].StandaloneResourceDetails
		rows[i].SharableResourceBaseInfo = devRows[i].SharableResourceBaseInfo
		rows[i].HostResourceInfo = devRows[i].HostResourceInfo
		rows[i].Guest = devRows[i].Guest
		rows[i].GuestStatus = devRows[i].GuestStatus
	}
	return rows
}

func (manager *SIsolatedDeviceResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestIsolatedDeviceListInput,
) (*sqlchemy.SQuery, error) {
	devQ := IsolatedDeviceManager.Query("id")
	devQ, err := IsolatedDeviceManager.ListItemFilter(ctx, devQ, userCred, query.IsolatedDeviceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "IsolatedDeviceManager.ListItemFilter")
	}
	q = q.Filter(sqlchemy.In(q.Field("isolated_device_id"), devQ.SubQuery()))
	return q, nil
}

func (manager *SIsolatedDeviceResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "isolated_device":
		devQ := IsolatedDeviceManager.Query("name", "id").SubQuery()
		q = q.AppendField(devQ.Field("name", field)).Distinct()
		q = q.Join(devQ, sqlchemy.Equals(q.Field("isolated_device_id"), devQ.Field("id")))
		return q, nil
	case "dev_type", "model", "addr", "device_path", "vendor_device_id", "numa_node":
		devQ := IsolatedDeviceManager.Query(field, "id").SubQuery()
		q = q.AppendField(devQ.Field(field)).Distinct()
		q = q.Join(devQ, sqlchemy.Equals(q.Field("isolated_device_id"), devQ.Field("id")))
		return q, nil
	default:
		devQ := IsolatedDeviceManager.Query("id", "host_id").SubQuery()
		q = q.LeftJoin(devQ, sqlchemy.Equals(q.Field("isolated_device_id"), devQ.Field("id")))
		q, err := IsolatedDeviceManager.SHostResourceBaseManager.QueryDistinctExtraField(q, field)
		if err == nil {
			return q, nil
		}
	}
	return q, httperrors.ErrNotFound
}

func (manager *SIsolatedDeviceResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestIsolatedDeviceListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := IsolatedDeviceManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("isolated_device_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SIsolatedDeviceResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.GuestIsolatedDeviceListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	devQ := IsolatedDeviceManager.Query().SubQuery()
	q = q.LeftJoin(devQ, sqlchemy.Equals(joinField, devQ.Field("id")))
	q, orders, fields = IsolatedDeviceManager.SHostResourceBaseManager.GetOrderBySubQuery(q, subq, devQ.Field("host_id"), userCred, query.HostFilterListInput, orders, fields)
	return q, orders, fields
}

func (manager *SIsolatedDeviceResourceBaseManager) GetOrderByFields(query api.GuestIsolatedDeviceListInput) []string {
	return IsolatedDeviceManager.SHostResourceBaseManager.GetOrderByFields(query.HostFilterListInput)
}

func (manager *SIsolatedDeviceResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		devQ := IsolatedDeviceManager.Query("id", "name", "dev_type", "model", "addr", "device_path", "vendor_device_id", "numa_node", "host_id").SubQuery()
		q = q.LeftJoin(devQ, sqlchemy.Equals(q.Field("isolated_device_id"), devQ.Field("id")))
		if keys.Contains("isolated_device") {
			q = q.AppendField(devQ.Field("name", "isolated_device"))
		}
		for _, key := range []string{"dev_type", "model", "addr", "device_path", "vendor_device_id", "numa_node"} {
			if keys.Contains(key) {
				q = q.AppendField(devQ.Field(key))
			}
		}
		if keys.ContainsAny(IsolatedDeviceManager.SHostResourceBaseManager.GetExportKeys()...) {
			var err error
			q, err = IsolatedDeviceManager.SHostResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SHostResourceBaseManager.ListItemExportKeys")
			}
		}
	}
	return q, nil
}

func (manager *SIsolatedDeviceResourceBaseManager) GetExportKeys() []string {
	keys := []string{"isolated_device", "dev_type", "model", "addr", "device_path", "vendor_device_id", "numa_node"}
	keys = append(keys, IsolatedDeviceManager.SHostResourceBaseManager.GetExportKeys()...)
	return keys
}
