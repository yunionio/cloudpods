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
	HostId string `width:"36" charset:"ascii" nullable:"true" list:"user"`
}

type SHostResourceBaseManager struct {
	SZoneResourceBaseManager
	SManagedResourceBaseManager
	hostIdFieldName string
}

func ValidateHostResourceInput(ctx context.Context, userCred mcclient.TokenCredential, input api.HostResourceInput) (*SHost, api.HostResourceInput, error) {
	hostObj, err := HostManager.FetchByIdOrName(ctx, userCred, input.HostId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, input, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", HostManager.Keyword(), input.HostId)
		} else {
			return nil, input, errors.Wrap(err, "HostManager.FetchByIdOrName")
		}
	}
	input.HostId = hostObj.GetId()
	return hostObj.(*SHost), input, nil
}

func (self *SHostResourceBase) GetHost() *SHost {
	obj, err := HostManager.FetchById(self.HostId)
	if err != nil {
		log.Errorf("fail to get host by id %s: %s", self.HostId, err)
		return nil
	}
	return obj.(*SHost)
}

func (manager *SHostResourceBaseManager) getHostIdFieldName() string {
	if len(manager.hostIdFieldName) > 0 {
		return manager.hostIdFieldName
	}
	return "host_id"
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
			rows[i].HostStatus = host.HostStatus
			rows[i].HostEnabled = host.Enabled.Bool()
			rows[i].HostServiceStatus = host.HostStatus
			rows[i].HostType = host.HostType
			rows[i].HostAccessIp = host.AccessIp
			rows[i].HostEIP = host.PublicIp
			rows[i].ManagerId = host.ManagerId
			rows[i].HostResourceType = host.ResourceType
			rows[i].HostBillingType = host.BillingType
			rows[i].ZoneId = host.ZoneId
		}
		zoneList[i] = &SZoneResourceBase{rows[i].ZoneId}
		managerList[i] = &SManagedResourceBase{rows[i].ManagerId}
	}

	zoneRows := manager.SZoneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, zoneList, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, managerList, fields, isList)
	for i := range rows {
		// 避免调度失败的机器返回错误的provider信息
		if len(hostIds[i]) > 0 {
			rows[i].ZoneResourceInfo = zoneRows[i]
			rows[i].ManagedResourceInfo = managerRows[i]
		}
	}

	return rows
}

func (manager *SHostResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.HostFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.HostId) > 0 {
		hostObj, _, err := ValidateHostResourceInput(ctx, userCred, query.HostResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateHostResourceInput")
		}
		q = q.Equals(manager.getHostIdFieldName(), hostObj.GetId())
	}
	if len(query.HostSN) > 0 {
		sq := HostManager.Query("id").In("sn", query.HostSN).SubQuery()
		q = q.In(manager.getHostIdFieldName(), sq)
	}
	if len(query.HostWireId) > 0 {
		wireObj, err := WireManager.FetchByIdOrName(ctx, userCred, query.HostWireId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(WireManager.Keyword(), query.HostWireId)
			} else {
				return nil, errors.Wrap(err, "WireManager.FetchByIdOrName")
			}
		}
		netifsQ := NetInterfaceManager.Query("baremetal_id").Equals("wire_id", wireObj.GetId()).Distinct().SubQuery()
		q = q.In(manager.getHostIdFieldName(), netifsQ)
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
		q = q.Filter(sqlchemy.In(q.Field(manager.getHostIdFieldName()), subq.SubQuery()))
	}
	return q, nil
}

func (manager *SHostResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "host":
		hostQuery := HostManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(hostQuery.Field("name", field))
		q = q.Join(hostQuery, sqlchemy.Equals(q.Field(manager.getHostIdFieldName()), hostQuery.Field("id")))
		q.GroupBy(hostQuery.Field("name"))
		return q, nil
	case "host_type":
		hostQuery := HostManager.Query(field, "id").Distinct().SubQuery()
		q.AppendField(hostQuery.Field(field))
		q = q.Join(hostQuery, sqlchemy.Equals(q.Field(manager.getHostIdFieldName()), hostQuery.Field("id")))
		q.GroupBy(hostQuery.Field(field))
		return q, nil
	case "manager", "account", "provider", "brand":
		hosts := HostManager.Query("id", "manager_id").SubQuery()
		q = q.LeftJoin(hosts, sqlchemy.Equals(q.Field(manager.getHostIdFieldName()), hosts.Field("id")))
		return manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	default:
		hosts := HostManager.Query("id", "zone_id").SubQuery()
		q = q.LeftJoin(hosts, sqlchemy.Equals(q.Field(manager.getHostIdFieldName()), hosts.Field("id")))
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
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := HostManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field(manager.getHostIdFieldName()), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SHostResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.HostFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	hostQ := HostManager.Query().SubQuery()
	q = q.LeftJoin(hostQ, sqlchemy.Equals(joinField, hostQ.Field("id")))
	q = q.AppendField(hostQ.Field("name").Label("host"))
	q = q.AppendField(hostQ.Field("sn").Label("host_sn"))
	orders = append(orders, query.OrderByHost, query.OrderByHostSN)
	fields = append(fields, subq.Field("host"), subq.Field("host_sn"))
	q, orders, fields = manager.SZoneResourceBaseManager.GetOrderBySubQuery(q, subq, hostQ.Field("zone_id"), userCred, query.ZonalFilterListInput, orders, fields)
	q, orders, fields = manager.SManagedResourceBaseManager.GetOrderBySubQuery(q, subq, hostQ.Field("manager_id"), userCred, query.ManagedResourceListInput, orders, fields)
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

func (manager *SHostResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		hostsQ := HostManager.Query("id", "name", "sn", "zone_id", "manager_id").SubQuery()
		q = q.LeftJoin(hostsQ, sqlchemy.Equals(q.Field(manager.getHostIdFieldName()), hostsQ.Field("id")))
		if keys.Contains("host") {
			q = q.AppendField(hostsQ.Field("name", "host"))
		}
		if keys.Contains("host_sn") {
			q = q.AppendField(hostsQ.Field("sn", "host_sn"))
		}
		if keys.ContainsAny(manager.SZoneResourceBaseManager.GetExportKeys()...) {
			var err error
			q, err = manager.SZoneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
			}
		}
		if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
			var err error
			q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
			}
		}
	}
	return q, nil
}

func (manager *SHostResourceBaseManager) GetExportKeys() []string {
	keys := []string{"host", "host_sn"}
	keys = append(keys, manager.SZoneResourceBaseManager.GetExportKeys()...)
	keys = append(keys, manager.SManagedResourceBaseManager.GetExportKeys()...)
	return keys
}

func (model *SHostResourceBase) GetChangeOwnerCandidateDomainIds() []string {
	host := model.GetHost()
	if host != nil {
		return host.GetChangeOwnerCandidateDomainIds()
	}
	return nil
}
