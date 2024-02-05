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

type SLoadbalancerClusterResourceBase struct {
	// 归属LB集群
	ClusterId string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional" json:"cluster_id"`
}

type SLoadbalancerClusterResourceBaseManager struct {
	SZoneResourceBaseManager
	SWireResourceBaseManager
}

func ValidateLoadbalancerClusterResourceInput(ctx context.Context, userCred mcclient.TokenCredential, input api.LoadbalancerClusterResourceInput) (*SLoadbalancerCluster, api.LoadbalancerClusterResourceInput, error) {
	clusterObj, err := LoadbalancerClusterManager.FetchByIdOrName(ctx, userCred, input.ClusterId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, input, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", LoadbalancerClusterManager.Keyword(), input.ClusterId)
		} else {
			return nil, input, errors.Wrap(err, "LoadbalancerClusterManager.FetchByIdOrName")
		}
	}
	input.ClusterId = clusterObj.GetId()
	return clusterObj.(*SLoadbalancerCluster), input, nil
}

func (self *SLoadbalancerClusterResourceBase) GetLoadbalancerCluster() *SLoadbalancerCluster {
	cluster, err := LoadbalancerClusterManager.FetchById(self.ClusterId)
	if err != nil {
		log.Errorf("failed to find LoadbalancerCluster %s error: %v", self.ClusterId, err)
		return nil
	}
	return cluster.(*SLoadbalancerCluster)
}

func (manager *SLoadbalancerClusterResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancerClusterResourceInfo {
	rows := make([]api.LoadbalancerClusterResourceInfo, len(objs))
	clusterIds := make([]string, len(objs))
	for i := range objs {
		var base *SLoadbalancerClusterResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SCloudregionResourceBase in object %s", objs[i])
			continue
		}
		clusterIds[i] = base.ClusterId
	}
	clusters := make(map[string]SLoadbalancerCluster)
	err := db.FetchStandaloneObjectsByIds(LoadbalancerClusterManager, clusterIds, clusters)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}
	zones := make([]interface{}, len(rows))
	wires := make([]interface{}, len(rows))
	for i := range rows {
		rows[i] = api.LoadbalancerClusterResourceInfo{}
		if cluster, ok := clusters[clusterIds[i]]; ok {
			rows[i].Cluster = cluster.Name
			zones[i] = &SZoneResourceBase{cluster.ZoneId}
			wires[i] = &SWireResourceBase{cluster.WireId}
		} else {
			zones[i] = &SZoneResourceBase{}
			wires[i] = &SWireResourceBase{}
		}
	}

	zoneRows := manager.SZoneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, zones, fields, isList)
	wireRows := manager.SWireResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, wires, fields, isList)

	for i := range rows {
		rows[i].ZoneResourceInfo = zoneRows[i]
		rows[i].WireResourceInfoBase = wireRows[i].WireResourceInfoBase
		rows[i].VpcId = wireRows[i].VpcId
		rows[i].Vpc = wireRows[i].Vpc
	}

	return rows
}

func (manager *SLoadbalancerClusterResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerClusterFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.ClusterId) > 0 {
		clusterObj, _, err := ValidateLoadbalancerClusterResourceInput(ctx, userCred, query.LoadbalancerClusterResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateLoadbalancerClusterResourceInput")
		}
		q = q.Equals("cluster_id", clusterObj.GetId())
	}
	subq := LoadbalancerClusterManager.Query("id").Snapshot()
	subq, err := manager.SZoneResourceBaseManager.ListItemFilter(ctx, subq, userCred, query.ZonalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemFilter")
	}
	wireQuery := api.WireFilterListInput{
		WireFilterListBase: query.WireFilterListBase,
	}
	subq, err = manager.SWireResourceBaseManager.ListItemFilter(ctx, subq, userCred, wireQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SWireResourceBaseManager.ListItemFilter")
	}
	if subq.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("cluster_id"), subq.SubQuery()))
	}
	return q, nil
}

func (manager *SLoadbalancerClusterResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerClusterFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := LoadbalancerClusterManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("cluster_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SLoadbalancerClusterResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	if field == "cluster" {
		clusterQuery := LoadbalancerClusterManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(clusterQuery.Field("name", field))
		q = q.Join(clusterQuery, sqlchemy.Equals(q.Field("cluster_id"), clusterQuery.Field("id")))
		q.GroupBy(clusterQuery.Field("name"))
		return q, nil
	}
	clusters := LoadbalancerClusterManager.Query("id", "zone_id", "wire_id").SubQuery()
	q = q.LeftJoin(clusters, sqlchemy.Equals(q.Field("cluster_id"), clusters.Field("id")))
	q, err := manager.SZoneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SWireResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SLoadbalancerClusterResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerClusterFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	clusterQ := LoadbalancerClusterManager.Query().SubQuery()
	q = q.LeftJoin(clusterQ, sqlchemy.Equals(joinField, clusterQ.Field("id")))
	q = q.AppendField(clusterQ.Field("name").Label("cluster"))
	orders = append(orders, query.OrderByCluster)
	fields = append(fields, subq.Field("cluster"))
	q, orders, fields = manager.SZoneResourceBaseManager.GetOrderBySubQuery(q, subq, clusterQ.Field("zone_id"), userCred, query.ZonalFilterListInput, orders, fields)
	return q, orders, fields
}

func (manager *SLoadbalancerClusterResourceBaseManager) GetOrderByFields(query api.LoadbalancerClusterFilterListInput) []string {
	fields := make([]string, 0)
	zoneFields := manager.SZoneResourceBaseManager.GetOrderByFields(query.ZonalFilterListInput)
	fields = append(fields, zoneFields...)
	fields = append(fields, query.OrderByCluster)
	return fields
}

func (manager *SLoadbalancerClusterResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		var err error
		subq := LoadbalancerClusterManager.Query("id", "name", "wire_id", "zone_id").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("cluster_id"), subq.Field("id")))

		if keys.Contains("cluster") {
			q = q.AppendField(subq.Field("name", "cluster"))
		}
		if keys.ContainsAny(manager.SZoneResourceBaseManager.GetExportKeys()...) {
			q, err = manager.SZoneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemExportKeys")
			}
		}
		if keys.Contains("wire") {
			q, err = manager.SWireResourceBaseManager.ListItemExportKeys(ctx, q, userCred, stringutils2.NewSortedStrings([]string{"wire"}))
			if err != nil {
				return nil, errors.Wrap(err, "SWireResourceBaseManager.ListItemExportKeys")
			}
		}
	}
	return q, nil
}

func (manager *SLoadbalancerClusterResourceBaseManager) GetExportKeys() []string {
	keys := []string{"cluster"}
	keys = append(keys, manager.SWireResourceBaseManager.GetExportKeys()...)
	return keys
}
