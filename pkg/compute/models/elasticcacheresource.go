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
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SElasticcacheResourceBase struct {
	// 弹性缓存ID
	ElasticcacheId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true" json:"elasticcache_id"`
}

type SElasticcacheResourceBaseManager struct {
	SVpcResourceBaseManager
	SZoneResourceBaseManager
}

func ValidateElasticcacheResourceInput(userCred mcclient.TokenCredential, input api.ELasticcacheResourceInput) (*SElasticcache, api.ELasticcacheResourceInput, error) {
	cacheObj, err := ElasticcacheManager.FetchByIdOrName(userCred, input.ElasticcacheId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, input, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", ElasticcacheManager.Keyword(), input.ElasticcacheId)
		} else {
			return nil, input, errors.Wrap(err, "ElasticcacheManager.FetchByIdOrName")
		}
	}
	input.ElasticcacheId = cacheObj.GetId()
	return cacheObj.(*SElasticcache), input, nil
}

func (self *SElasticcacheResourceBase) GetElasticcache() (*SElasticcache, error) {
	instance, err := ElasticcacheManager.FetchById(self.ElasticcacheId)
	if err != nil {
		return nil, errors.Wrap(err, "DBInstanceManager.FetchById")
	}
	return instance.(*SElasticcache), nil
}

func (self *SElasticcacheResourceBase) GetVpc() *SVpc {
	cache, err := self.GetElasticcache()
	if err != nil {
		log.Errorf("GetElasticcache fail %s", err)
		return nil
	}
	return cache.GetVpc()
}

func (self *SElasticcacheResourceBase) GetIRegion() (cloudprovider.ICloudRegion, error) {
	vpc := self.GetVpc()
	if vpc == nil {
		return nil, errors.Wrap(httperrors.ErrNotFound, "no vpc found")
	}
	return vpc.GetIRegion()
}

func (self *SElasticcacheResourceBase) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) api.ElasticcacheResourceInfo {
	return api.ElasticcacheResourceInfo{}
}

func (manager *SElasticcacheResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ElasticcacheResourceInfo {
	rows := make([]api.ElasticcacheResourceInfo, len(objs))
	elasticcacheIds := make([]string, len(objs))
	for i := range objs {
		var base *SElasticcacheResourceBase
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil {
			elasticcacheIds[i] = base.ElasticcacheId
		}
	}
	elasticcaches := make(map[string]SElasticcache)
	err := db.FetchStandaloneObjectsByIds(ElasticcacheManager, elasticcacheIds, &elasticcaches)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	vpcList := make([]interface{}, len(rows))
	zoneList := make([]interface{}, len(rows))
	for i := range rows {
		rows[i] = api.ElasticcacheResourceInfo{}
		if elasticache, ok := elasticcaches[elasticcacheIds[i]]; ok {
			rows[i].Elasticcache = elasticache.Name
			rows[i].Engine = elasticache.Engine
			rows[i].EngineVersion = elasticache.EngineVersion
			rows[i].VpcId = elasticache.VpcId
			rows[i].ZoneId = elasticache.ZoneId
		}
		vpcList[i] = &SVpcResourceBase{rows[i].VpcId}
		zoneList[i] = &SZoneResourceBase{rows[i].ZoneId}
	}

	vpcRows := manager.SVpcResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, vpcList, fields, isList)
	zoneRows := manager.SZoneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, zoneList, fields, isList)
	for i := range rows {
		rows[i].VpcResourceInfo = vpcRows[i]
		rows[i].ZoneResourceInfoBase = zoneRows[i].ZoneResourceInfoBase
	}

	return rows
}

func (manager *SElasticcacheResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ElasticcacheFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.ElasticcacheId) > 0 {
		dbObj, _, err := ValidateElasticcacheResourceInput(userCred, query.ELasticcacheResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateElasticcacheResourceInput")
		}
		q = q.Equals("elasticcache_id", dbObj.GetId())
	}

	subq := ElasticcacheManager.Query("id").Snapshot()
	subq, err := manager.SVpcResourceBaseManager.ListItemFilter(ctx, subq, userCred, query.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemFilter")
	}
	zoneQuery := api.ZonalFilterListInput{
		ZonalFilterListBase: query.ZonalFilterListBase,
	}
	subq, err = manager.SZoneResourceBaseManager.ListItemFilter(ctx, subq, userCred, zoneQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemFilter")
	}
	if subq.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("elasticcache_id"), subq.SubQuery()))
	}
	return q, nil
}

func (manager *SElasticcacheResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "elasticcache":
		dbQuery := ElasticcacheManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(dbQuery.Field("name", field))
		q = q.Join(dbQuery, sqlchemy.Equals(q.Field("elasticcache_id"), dbQuery.Field("id")))
		q.GroupBy(dbQuery.Field("name"))
		return q, nil
	}
	elasticcaches := ElasticcacheManager.Query("id", "vpc_id").SubQuery()
	q = q.LeftJoin(elasticcaches, sqlchemy.Equals(q.Field("elasticcache_id"), elasticcaches.Field("id")))
	q, err := manager.SVpcResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SElasticcacheResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ElasticcacheFilterListInput,
) (*sqlchemy.SQuery, error) {
	q, orders, fields := manager.GetOrderBySubQuery(q, userCred, query)
	if len(orders) > 0 {
		q = db.OrderByFields(q, orders, fields)
	}
	return q, nil
}

func (manager *SElasticcacheResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ElasticcacheFilterListInput,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	cacheQ := ElasticcacheManager.Query("id", "name")
	var orders []string
	var fields []sqlchemy.IQueryField

	if db.NeedOrderQuery(manager.SVpcResourceBaseManager.GetOrderByFields(query.VpcFilterListInput)) {
		var vpcOrders []string
		var vpcFields []sqlchemy.IQueryField
		cacheQ, vpcOrders, vpcFields = manager.SVpcResourceBaseManager.GetOrderBySubQuery(cacheQ, userCred, query.VpcFilterListInput)
		if len(vpcOrders) > 0 {
			orders = append(orders, vpcOrders...)
			fields = append(fields, vpcFields...)
		}
	}
	zoneQuery := api.ZonalFilterListInput{
		ZonalFilterListBase: query.ZonalFilterListBase,
	}
	if db.NeedOrderQuery(manager.SZoneResourceBaseManager.GetOrderByFields(zoneQuery)) {
		var zoneOrders []string
		var zoneFields []sqlchemy.IQueryField
		cacheQ, zoneOrders, zoneFields = manager.SZoneResourceBaseManager.GetOrderBySubQuery(cacheQ, userCred, zoneQuery)
		if len(zoneOrders) > 0 {
			orders = append(orders, zoneOrders...)
			fields = append(fields, zoneFields...)
		}
	}
	if db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		subq := cacheQ.SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("elasticcache_id"), subq.Field("id")))
		if db.NeedOrderQuery([]string{query.OrderByElasticcache}) {
			orders = append(orders, query.OrderByElasticcache)
			fields = append(fields, subq.Field("name"))
		}
	}
	return q, orders, fields
}

func (manager *SElasticcacheResourceBaseManager) GetOrderByFields(query api.ElasticcacheFilterListInput) []string {
	fields := make([]string, 0)
	vpcFields := manager.SVpcResourceBaseManager.GetOrderByFields(query.VpcFilterListInput)
	fields = append(fields, vpcFields...)
	zoneQuery := api.ZonalFilterListInput{
		ZonalFilterListBase: query.ZonalFilterListBase,
	}
	zoneFields := manager.SZoneResourceBaseManager.GetOrderByFields(zoneQuery)
	fields = append(fields, zoneFields...)
	fields = append(fields, query.OrderByElasticcache)
	return fields
}

func (manager *SElasticcacheResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		var err error
		subq := ElasticcacheManager.Query("id", "name", "vpc_id", "zone_id").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("elasticcache_id"), subq.Field("id")))
		if keys.Contains("elasticcache") {
			q = q.AppendField(subq.Field("name", "elasticcache"))
		}
		if keys.ContainsAny(manager.SVpcResourceBaseManager.GetExportKeys()...) {
			q, err = manager.SVpcResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemExportKeys")
			}
		}
		if keys.Contains("zone") {
			q, err = manager.SZoneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemExportKeys")
			}
		}
	}
	return q, nil
}

func (manager *SElasticcacheResourceBaseManager) GetExportKeys() []string {
	keys := []string{"elasticcache"}
	keys = append(keys, manager.SVpcResourceBaseManager.GetExportKeys()...)
	keys = append(keys, "zone")
	return keys
}

func (self *SElasticcacheResourceBase) GetChangeOwnerCandidateDomainIds() []string {
	cache, _ := self.GetElasticcache()
	if cache != nil {
		return cache.GetChangeOwnerCandidateDomainIds()
	}
	return nil
}
