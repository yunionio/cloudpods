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

type SZoneResourceBase struct {
	ZoneId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user" json:"zone_id"`
}

type SZoneResourceBaseManager struct {
	SCloudregionResourceBaseManager
}

func ValidateZoneResourceInput(ctx context.Context, userCred mcclient.TokenCredential, query api.ZoneResourceInput) (*SZone, api.ZoneResourceInput, error) {
	zoneObj, err := ZoneManager.FetchByIdOrName(ctx, userCred, query.ZoneId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, query, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", ZoneManager.Keyword(), query.ZoneId)
		} else {
			return nil, query, errors.Wrap(err, "ZoneManager.FetchByIdOrName")
		}
	}
	query.ZoneId = zoneObj.GetId()
	return zoneObj.(*SZone), query, nil
}

func (self *SZoneResourceBase) GetZone() (*SZone, error) {
	zone, err := ZoneManager.FetchById(self.ZoneId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetZone(%s)", self.ZoneId)
	}
	return zone.(*SZone), nil
}

func (manager *SZoneResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ZoneResourceInfo {
	rows := make([]api.ZoneResourceInfo, len(objs))
	zoneIds := make([]string, len(objs))
	for i := range objs {
		var base *SZoneResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SZoneResourceBase in object %#v: %s", objs[i], err)
			continue
		}
		zoneIds[i] = base.ZoneId
	}

	zones := make(map[string]SZone)
	err := db.FetchStandaloneObjectsByIds(ZoneManager, zoneIds, &zones)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	regions := make([]interface{}, len(rows))
	for i := range rows {
		rows[i] = api.ZoneResourceInfo{}
		if _, ok := zones[zoneIds[i]]; ok {
			z := zones[zoneIds[i]]
			name := z.Name
			if v, ok := z.GetModelKeyI18N(ctx, &z, "name"); ok {
				name = v
			}
			rows[i].Zone = name
			rows[i].ZoneExtId = fetchExternalId(zones[zoneIds[i]].ExternalId)
			rows[i].CloudregionId = zones[zoneIds[i]].CloudregionId
		}
		regions[i] = &SCloudregionResourceBase{rows[i].CloudregionId}
	}

	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, regions, fields, isList)
	for i := range rows {
		rows[i].CloudregionResourceInfo = regionRows[i]
	}

	return rows
}

func (manager *SZoneResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ZonalFilterListInput,
) (*sqlchemy.SQuery, error) {
	q, err := managedResourceFilterByZone(ctx, q, query, "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "managedResourceFilterByZone")
	}
	subq := ZoneManager.Query("id").Snapshot()
	subq, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, subq, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	if subq.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("zone_id"), subq.SubQuery()))
	}
	return q, nil
}

func (manager *SZoneResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "zone":
		zoneQuery := ZoneManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(zoneQuery.Field("name", field))
		q = q.Join(zoneQuery, sqlchemy.Equals(q.Field("zone_id"), zoneQuery.Field("id")))
		q = q.GroupBy(zoneQuery.Field("name"))
		return q, nil
	}
	zones := ZoneManager.Query("id", "cloudregion_id").SubQuery()
	q = q.LeftJoin(zones, sqlchemy.Equals(q.Field("zone_id"), zones.Field("id")))
	q, err := manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SZoneResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ZonalFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := ZoneManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("zone_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SZoneResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.ZonalFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	zoneQ := ZoneManager.Query().SubQuery()
	q = q.LeftJoin(zoneQ, sqlchemy.Equals(joinField, zoneQ.Field("id")))
	q = q.AppendField(zoneQ.Field("name").Label("zone"))
	orders = append(orders, query.OrderByZone)
	fields = append(fields, subq.Field("zone"))
	q, orders, fields = manager.SCloudregionResourceBaseManager.GetOrderBySubQuery(q, subq, zoneQ.Field("cloudregion_id"), userCred, query.RegionalFilterListInput, orders, fields)
	return q, orders, fields
}

func (manager *SZoneResourceBaseManager) GetOrderByFields(query api.ZonalFilterListInput) []string {
	orders := make([]string, 0)
	zoneOrders := manager.SCloudregionResourceBaseManager.GetOrderByFields(query.RegionalFilterListInput)
	orders = append(orders, zoneOrders...)
	orders = append(orders, query.OrderByZone)
	return orders
}

func (manager *SZoneResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		zonesQ := ZoneManager.Query("id", "name", "cloudregion_id").SubQuery()
		q = q.LeftJoin(zonesQ, sqlchemy.Equals(q.Field("zone_id"), zonesQ.Field("id")))
		if keys.Contains("zone") {
			q = q.AppendField(zonesQ.Field("name", "zone"))
		}
		if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
			var err error
			q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
			}
		}
	}
	return q, nil
}

func (manager *SZoneResourceBaseManager) GetExportKeys() []string {
	return append(manager.SCloudregionResourceBaseManager.GetExportKeys(), "zones")
}
