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

type SHostResourceBase struct {
	HostId string `width:"36" charset:"ascii" nullable:"true" list:"user" index:"true" create:"optional"`
}

type SHostResourceBaseManager struct {
	SZoneResourceBaseManager
	SManagedResourceBaseManager
}

func (self *SHostResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) api.HostResourceInfo {
	return api.HostResourceInfo{}
}

func (manager *SHostResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.HostResourceInfo {
	rows := make([]api.HostResourceInfo, len(objs))
	hostIds := make([]string, len(objs))
	for i := range objs {
		var base *SHostResourceBase
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil {
			hostIds[i] = base.HostId
		}
	}

	hosts := make(map[string]SHost)
	err := db.FetchStandaloneObjectsByIds(HostManager, hostIds, hosts)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	zoneList := make([]interface{}, len(rows))
	managerList := make([]interface{}, len(rows))

	for i := range rows {
		rows[i] = api.HostResourceInfo{}
		if _, ok := hosts[hostIds[i]]; ok {
			host := hosts[hostIds[i]]
			rows[i].Host = host.Name
			rows[i].HostSN = host.SN
			rows[i].HostStatus = host.Status
			rows[i].HostServiceStatus = host.HostStatus
			rows[i].HostType = host.HostType
			rows[i].ManagerId = host.ManagerId
			rows[i].ZoneId = host.ZoneId
		}
		zoneList[i] = &SZoneResourceBase{rows[i].ZoneId}
		managerList[i] = &SManagedResourceBase{rows[i].ManagerId}
	}

	zoneRows := manager.SZoneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, zoneList, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, managerList, fields, isList)
	for i := range rows {
		rows[i].ZoneResourceInfo = zoneRows[i]
		rows[i].ManagedResourceInfo = managerRows[i]
	}

	return rows
}

func (manager *SHostResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.HostFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.Host) > 0 {
		hostObj, err := HostManager.FetchByIdOrName(userCred, query.Host)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(HostManager.Keyword(), query.Host)
			} else {
				return nil, errors.Wrap(err, "HostManager.FetchByIdOrName")
			}
		}
		q = q.Equals("host_id", hostObj.GetId())
	}
	if len(query.HostSN) > 0 {
		sq := HostManager.Query("id").Equals("sn", query.HostSN).SubQuery()
		q = q.In("host_id", sq)
	}
	subq := HostManager.Query("id").Snapshot()
	subq, err := manager.SZoneResourceBaseManager.ListItemFilter(ctx, subq, userCred, query.ZonalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemFilter")
	}
	subq, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, subq, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	if subq.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("host_id"), subq.SubQuery()))
	}
	return q, nil
}

func (manager *SHostResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "host":
		hostQuery := HostManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(hostQuery.Field("name", field))
		q = q.Join(hostQuery, sqlchemy.Equals(q.Field("host_id"), hostQuery.Field("id")))
		q.GroupBy(hostQuery.Field("name"))
		return q, nil
	case "host_type":
		hostQuery := HostManager.Query(field, "id").Distinct().SubQuery()
		q.AppendField(hostQuery.Field(field))
		q = q.Join(hostQuery, sqlchemy.Equals(q.Field("host_id"), hostQuery.Field("id")))
		q.GroupBy(hostQuery.Field(field))
		return q, nil
	case "manager", "account", "provider", "brand":
		hosts := HostManager.Query("id", "manager_id").SubQuery()
		q = q.LeftJoin(hosts, sqlchemy.Equals(q.Field("host_id"), hosts.Field("id")))
		return manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	default:
		hosts := HostManager.Query("id", "zone_id").SubQuery()
		q = q.LeftJoin(hosts, sqlchemy.Equals(q.Field("host_id"), hosts.Field("id")))
		q, err := manager.SZoneResourceBaseManager.QueryDistinctExtraField(q, field)
		if err == nil {
			return q, nil
		}
	}
	return q, httperrors.ErrNotFound
}

func (manager *SHostResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.HostFilterListInput,
) (*sqlchemy.SQuery, error) {
	q, orders, fields := manager.GetOrderBySubQuery(q, userCred, query)
	if len(orders) > 0 {
		q = db.OrderByFields(q, orders, fields)
	}
	return q, nil
}

func (manager *SHostResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.HostFilterListInput,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	hostQ := HostManager.Query("id", "name", "sn")
	var orders []string
	var fields []sqlchemy.IQueryField

	if db.NeedOrderQuery(manager.SZoneResourceBaseManager.GetOrderByFields(query.ZonalFilterListInput)) {
		var zoneOrders []string
		var zoneFields []sqlchemy.IQueryField
		hostQ, zoneOrders, zoneFields = manager.SZoneResourceBaseManager.GetOrderBySubQuery(hostQ, userCred, query.ZonalFilterListInput)
		if len(zoneOrders) > 0 {
			orders = append(orders, zoneOrders...)
			fields = append(fields, zoneFields...)
		}
	}
	if db.NeedOrderQuery(manager.SManagedResourceBaseManager.GetOrderByFields(query.ManagedResourceListInput)) {
		var manOrders []string
		var manFields []sqlchemy.IQueryField
		hostQ, manOrders, manFields = manager.SManagedResourceBaseManager.GetOrderBySubQuery(hostQ, userCred, query.ManagedResourceListInput)
		if len(manOrders) > 0 {
			orders = append(orders, manOrders...)
			fields = append(fields, manFields...)
		}
	}
	if db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		subq := hostQ.SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("host_id"), subq.Field("id")))
		if db.NeedOrderQuery([]string{query.OrderByHost, query.OrderByHostSN}) {
			orders = append(orders, query.OrderByHost, query.OrderByHostSN)
			fields = append(fields, subq.Field("name"), subq.Field("sn"))
		}
	}
	return q, orders, fields
}

func (manager *SHostResourceBaseManager) GetOrderByFields(query api.HostFilterListInput) []string {
	fields := make([]string, 0)
	zoneFields := manager.SZoneResourceBaseManager.GetOrderByFields(query.ZonalFilterListInput)
	fields = append(fields, zoneFields...)
	manFields := manager.SManagedResourceBaseManager.GetOrderByFields(query.ManagedResourceListInput)
	fields = append(fields, manFields...)
	fields = append(fields, query.OrderByHost, query.OrderByHostSN)
	return fields
}
