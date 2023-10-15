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
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SWireResourceBase struct {
	// 二层网络ID
	WireId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"wire_id"`
}

type SWireResourceBaseManager struct {
	SVpcResourceBaseManager
	SZoneResourceBaseManager
}

func ValidateWireResourceInput(userCred mcclient.TokenCredential, input api.WireResourceInput) (*SWire, api.WireResourceInput, error) {
	wireObj, err := WireManager.FetchByIdOrName(userCred, input.WireId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, input, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", WireManager.Keyword(), input.WireId)
		} else {
			return nil, input, errors.Wrap(err, "WireManager.FetchByIdOrName")
		}
	}
	input.WireId = wireObj.GetId()
	return wireObj.(*SWire), input, nil
}

func (wireRes *SWireResourceBase) GetWire() (*SWire, error) {
	w, err := WireManager.FetchById(wireRes.WireId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetWire(%s)", wireRes.WireId)
	}
	return w.(*SWire), nil
}

func (wireRes *SWireResourceBase) GetCloudproviderId() string {
	wire, _ := wireRes.GetWire()
	if wire != nil {
		return wire.ManagerId
	}
	return ""
}

func (wireRes *SWireResourceBase) GetProviderName() string {
	wire, _ := wireRes.GetWire()
	if wire == nil {
		return wire.GetProviderName()
	}
	return ""
}

func (wireRes *SWireResourceBase) GetVpc() (*SVpc, error) {
	wire, err := wireRes.GetWire()
	if err != nil {
		return nil, errors.Wrapf(err, "GetWire")
	}
	return wire.GetVpc()
}

func (wireRes *SWireResourceBase) GetRegion() (*SCloudregion, error) {
	vpc, err := wireRes.GetVpc()
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpc")
	}
	return vpc.GetRegion()
}

func (wireRes *SWireResourceBase) GetZone() (*SZone, error) {
	wire, err := wireRes.GetWire()
	if err != nil {
		return nil, errors.Wrapf(err, "GetWire")
	}
	return wire.GetZone()
}

func (manager *SWireResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.WireResourceInfo {
	rows := make([]api.WireResourceInfo, len(objs))

	wireIds := make([]string, len(objs))
	for i := range objs {
		var base *SWireResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SWireResourceBase in object %#v: %s", objs[i], err)
			continue
		}
		wireIds[i] = base.WireId
	}

	wires := make(map[string]SWire)
	err := db.FetchStandaloneObjectsByIds(WireManager, wireIds, &wires)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return nil
	}

	vpcList := make([]interface{}, len(rows))
	zoneList := make([]interface{}, len(rows))
	managerList := make([]interface{}, len(rows))
	for i := range rows {
		rows[i] = api.WireResourceInfo{}
		if _, ok := wires[wireIds[i]]; ok {
			wire := wires[wireIds[i]]
			rows[i].Wire = wire.Name
			rows[i].VpcId = wire.VpcId
			rows[i].ZoneId = wire.ZoneId
		}
		vpcList[i] = &SVpcResourceBase{rows[i].VpcId}
		zoneList[i] = &SZoneResourceBase{rows[i].ZoneId}
		managerList[i] = &SManagedResourceBase{wires[wireIds[i]].ManagerId}
	}

	vpcRows := manager.SVpcResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, vpcList, fields, isList)
	zoneRows := manager.SZoneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, zoneList, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, managerList, fields, isList)

	for i := range rows {
		rows[i].VpcResourceInfo = vpcRows[i]
		rows[i].Zone = zoneRows[i].Zone
		rows[i].ManagedResourceInfo = managerRows[i]
	}
	return rows
}

func (manager *SWireResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WireFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.WireId) > 0 {
		wireObj, _, err := ValidateWireResourceInput(userCred, query.WireResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateWireResourceInput")
		}
		q = q.Equals("wire_id", wireObj.GetId())
	}

	wireQ := WireManager.Query("id").Snapshot()

	{
		var err error
		mangedFilter := query.ManagedResourceListInput
		query.ManagedResourceListInput = api.ManagedResourceListInput{}
		wireQ, err = manager.SVpcResourceBaseManager.ListItemFilter(ctx, wireQ, userCred, query.VpcFilterListInput)
		if err != nil {
			return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemFilter")
		}
		wireQ, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, wireQ, userCred, mangedFilter)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
		}
		// recover managed filter
		query.ManagedResourceListInput = mangedFilter
	}

	if len(query.ZoneList()) > 0 {
		region := &SCloudregion{}
		firstZone := query.FirstZone()
		sq := ZoneManager.Query().SubQuery()
		regionQ := CloudregionManager.Query()
		regionQ = regionQ.Join(sq, sqlchemy.Equals(sq.Field("cloudregion_id"), regionQ.Field("id"))).Filter(sqlchemy.OR(
			sqlchemy.Equals(sq.Field("id"), firstZone),
			sqlchemy.Equals(sq.Field("name"), firstZone),
		))
		count, err := regionQ.CountWithError()
		if err != nil {
			return nil, errors.Wrap(err, "CountWithError")
		}
		if count < 1 {
			return nil, httperrors.NewResourceNotFoundError2("zone", firstZone)
		}
		err = regionQ.First(region)
		if err != nil {
			return nil, errors.Wrap(err, "regionQ.First")
		}
		if utils.IsInStringArray(region.Provider, api.REGIONAL_NETWORK_PROVIDERS) {
			vpcQ := VpcManager.Query().SubQuery()
			wireQ = wireQ.Join(vpcQ, sqlchemy.Equals(vpcQ.Field("id"), wireQ.Field("vpc_id"))).
				Filter(sqlchemy.Equals(vpcQ.Field("cloudregion_id"), region.Id))
		} else {
			zoneQuery := api.ZonalFilterListInput{
				ZonalFilterListBase: query.ZonalFilterListBase,
			}
			wireQ, err = manager.SZoneResourceBaseManager.ListItemFilter(ctx, wireQ, userCred, zoneQuery)
			if err != nil {
				return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemFilter")
			}
		}
	}

	if wireQ.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("wire_id"), wireQ.SubQuery()))
	}
	return q, nil
}

func (manager *SWireResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	if field == "wire" {
		wireQuery := WireManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(wireQuery.Field("name", field))
		q = q.Join(wireQuery, sqlchemy.Equals(q.Field("wire_id"), wireQuery.Field("id")))
		q.GroupBy(wireQuery.Field("name"))
		return q, nil
	} else {
		wires := WireManager.Query("id", "zone_id", "vpc_id").SubQuery()
		q = q.LeftJoin(wires, sqlchemy.Equals(q.Field("wire_id"), wires.Field("id")))
		if field == "zone" {
			return manager.SZoneResourceBaseManager.QueryDistinctExtraField(q, field)
		} else {
			q, err := manager.SVpcResourceBaseManager.QueryDistinctExtraField(q, field)
			if err == nil {
				return q, nil
			} else {
				return q, httperrors.ErrNotFound
			}
		}
	}
}

func (manager *SWireResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WireFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := WireManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	_, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("wire_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SWireResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.WireFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	wireQ := WireManager.Query().SubQuery()
	q = q.LeftJoin(wireQ, sqlchemy.Equals(joinField, wireQ.Field("id")))
	q = q.AppendField(wireQ.Field("name").Label("wire"))
	orders = append(orders, query.OrderByWire)
	fields = append(fields, subq.Field("wire"))
	zoneQuery := api.ZonalFilterListInput{
		ZonalFilterListBase: query.ZonalFilterListBase,
	}
	q, orders, fields = manager.SZoneResourceBaseManager.GetOrderBySubQuery(q, subq, wireQ.Field("zone_id"), userCred, zoneQuery, orders, fields)
	q, orders, fields = manager.SVpcResourceBaseManager.GetOrderBySubQuery(q, subq, wireQ.Field("vpc_id"), userCred, query.VpcFilterListInput, orders, fields)
	return q, orders, fields
}

func (manager *SWireResourceBaseManager) GetOrderByFields(query api.WireFilterListInput) []string {
	fields := make([]string, 0)
	zoneQuery := api.ZonalFilterListInput{
		ZonalFilterListBase: query.ZonalFilterListBase,
	}
	zoneFields := manager.SZoneResourceBaseManager.GetOrderByFields(zoneQuery)
	fields = append(fields, zoneFields...)
	vpcFields := manager.SVpcResourceBaseManager.GetOrderByFields(query.VpcFilterListInput)
	fields = append(fields, vpcFields...)
	fields = append(fields, query.OrderByWire)
	return fields
}

func (manager *SWireResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		var err error
		subq := WireManager.Query("id", "name", "vpc_id", "zone_id").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("wire_id"), subq.Field("id")))
		if keys.Contains("wire") {
			q = q.AppendField(subq.Field("name", "wire"))
		}
		if keys.Contains("zone") {
			q, err = manager.SZoneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, stringutils2.NewSortedStrings([]string{"zone"}))
			if err != nil {
				return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemExportKeys")
			}
		}
		if keys.ContainsAny(manager.SVpcResourceBaseManager.GetExportKeys()...) {
			q, err = manager.SVpcResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemExportKeys")
			}
		}
	}
	return q, nil
}

func (manager *SWireResourceBaseManager) GetExportKeys() []string {
	keys := []string{"wire"}
	keys = append(keys, "zone")
	keys = append(keys, manager.SVpcResourceBaseManager.GetExportKeys()...)
	return keys
}

func (wireRes *SWireResourceBase) GetChangeOwnerCandidateDomainIds() []string {
	wire, _ := wireRes.GetWire()
	if wire != nil {
		return wire.GetChangeOwnerCandidateDomainIds()
	}
	return nil
}
