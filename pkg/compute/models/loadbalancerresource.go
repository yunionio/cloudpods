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

type SLoadbalancerResourceBase struct {
	// 负载均衡ID
	LoadbalancerId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

type SLoadbalancerResourceBaseManager struct {
	SVpcResourceBaseManager
	SZoneResourceBaseManager
}

func ValidateLoadbalancerResourceInput(ctx context.Context, userCred mcclient.TokenCredential, input api.LoadbalancerResourceInput) (*SLoadbalancer, api.LoadbalancerResourceInput, error) {
	lbObj, err := LoadbalancerManager.FetchByIdOrName(ctx, userCred, input.LoadbalancerId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, input, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", LoadbalancerManager.Keyword(), input.LoadbalancerId)
		} else {
			return nil, input, errors.Wrap(err, "LoadbalancerManager.FetchByIdOrName")
		}
	}
	input.LoadbalancerId = lbObj.GetId()
	return lbObj.(*SLoadbalancer), input, nil
}

func (self *SLoadbalancerResourceBase) GetLoadbalancer() (*SLoadbalancer, error) {
	w, err := LoadbalancerManager.FetchById(self.LoadbalancerId)
	if err != nil {
		return nil, err
	}
	return w.(*SLoadbalancer), nil
}

func (self *SLoadbalancerResourceBase) GetVpc() (*SVpc, error) {
	lb, err := self.GetLoadbalancer()
	if err != nil {
		return nil, err
	}
	return lb.GetVpc()
}

func (self *SLoadbalancerResourceBase) GetCloudprovider() *SCloudprovider {
	vpc, _ := self.GetVpc()
	if vpc != nil {
		return vpc.GetCloudprovider()
	}

	lb, _ := self.GetLoadbalancer()
	if lb != nil {
		return lb.GetCloudprovider()
	}

	return nil
}

func (self *SLoadbalancerResourceBase) GetCloudproviderId() string {
	cloudprovider := self.GetCloudprovider()
	if cloudprovider != nil {
		return cloudprovider.Id
	}
	return ""
}

func (self *SLoadbalancerResourceBase) GetProviderName() string {
	vpc, _ := self.GetVpc()
	if vpc != nil {
		return vpc.GetProviderName()
	}
	return ""
}

func (self *SLoadbalancerResourceBase) GetCloudaccount() *SCloudaccount {
	vpc, _ := self.GetVpc()
	if vpc != nil {
		return vpc.GetCloudaccount()
	}
	return nil
}

func (self *SLoadbalancerResourceBase) GetRegion() (*SCloudregion, error) {
	vpc, err := self.GetVpc()
	if err != nil {
		return nil, err
	}
	return vpc.GetRegion()
}

func (self *SLoadbalancerResourceBase) GetRegionId() string {
	region, _ := self.GetRegion()
	if region != nil {
		return region.Id
	}
	return ""
}

func (self *SLoadbalancerResourceBase) GetZone() (*SZone, error) {
	lb, err := self.GetLoadbalancer()
	if err != nil {
		return nil, err
	}
	return lb.GetZone()
}

func (manager *SLoadbalancerResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancerResourceInfo {
	rows := make([]api.LoadbalancerResourceInfo, len(objs))

	lbIds := make([]string, len(objs))
	for i := range objs {
		var base *SLoadbalancerResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SLoadbalancerResourceBase in object %#v: %s", objs[i], err)
			continue
		}
		lbIds[i] = base.LoadbalancerId
	}

	lbs := make(map[string]SLoadbalancer)
	err := db.FetchStandaloneObjectsByIds(LoadbalancerManager, lbIds, &lbs)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return nil
	}

	vpcList := make([]interface{}, len(rows))
	zoneList := make([]interface{}, len(rows))
	manList := make([]interface{}, len(rows))
	for i := range rows {
		rows[i] = api.LoadbalancerResourceInfo{}
		if lb, ok := lbs[lbIds[i]]; ok {
			rows[i].Loadbalancer = lb.Name
			rows[i].VpcId = lb.VpcId
			rows[i].ZoneId = lb.ZoneId
			rows[i].ManagerId = lb.ManagerId
		}
		vpcList[i] = &SVpcResourceBase{rows[i].VpcId}
		zoneList[i] = &SZoneResourceBase{rows[i].ZoneId}
		manList[i] = &SManagedResourceBase{rows[i].ManagerId}
	}

	vpcRows := manager.SVpcResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, vpcList, fields, isList)
	zoneRows := manager.SZoneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, zoneList, fields, isList)
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, manList, fields, isList)

	for i := range rows {
		rows[i].VpcResourceInfo = vpcRows[i]
		rows[i].ZoneResourceInfo = zoneRows[i]
		rows[i].ManagedResourceInfo = manRows[i]
	}
	return rows
}

func (manager *SLoadbalancerResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.LoadbalancerId) > 0 {
		lbObj, _, err := ValidateLoadbalancerResourceInput(ctx, userCred, query.LoadbalancerResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateLoadbalancerResourceInput")
		}
		q = q.Equals("loadbalancer_id", lbObj.GetId())
	}

	lbQ := LoadbalancerManager.Query("id").Snapshot()

	lbQ, err := manager.SVpcResourceBaseManager.ListItemFilter(ctx, lbQ, userCred, query.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemFilter")
	}

	zoneQuery := api.ZonalFilterListInput{
		ZonalFilterListBase: query.ZonalFilterListBase,
	}
	lbQ, err = manager.SZoneResourceBaseManager.ListItemFilter(ctx, lbQ, userCred, zoneQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemFilter")
	}

	if lbQ.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("loadbalancer_id"), lbQ.SubQuery()))
	}
	return q, nil
}

func (manager *SLoadbalancerResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	if field == "loadbalancer" {
		lbQuery := LoadbalancerManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(lbQuery.Field("name", field))
		q = q.Join(lbQuery, sqlchemy.Equals(q.Field("loadbalancer_id"), lbQuery.Field("id")))
		q.GroupBy(lbQuery.Field("name"))
		return q, nil
	} else {
		lbs := LoadbalancerManager.Query("id", "zone_id", "vpc_id").SubQuery()
		q = q.LeftJoin(lbs, sqlchemy.Equals(q.Field("loadbalancer_id"), lbs.Field("id")))
		q, err := manager.SZoneResourceBaseManager.QueryDistinctExtraField(q, field)
		if err == nil {
			return q, nil
		}

		q, err = manager.SVpcResourceBaseManager.QueryDistinctExtraField(q, field)
		if err == nil {
			return q, nil
		}

		return q, httperrors.ErrNotFound
	}
}

func (manager *SLoadbalancerResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := LoadbalancerManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("loadbalancer_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SLoadbalancerResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	lbQ := LoadbalancerManager.Query().SubQuery()
	q = q.LeftJoin(lbQ, sqlchemy.Equals(joinField, lbQ.Field("id")))
	q = q.AppendField(lbQ.Field("name").Label("loadbalancer"))
	orders = append(orders, query.OrderByLoadbalancer)
	fields = append(fields, subq.Field("loadbalancer"))

	zoneQuery := api.ZonalFilterListInput{
		ZonalFilterListBase: query.ZonalFilterListBase,
	}
	q, orders, fields = manager.SZoneResourceBaseManager.GetOrderBySubQuery(q, subq, lbQ.Field("zone_id"), userCred, zoneQuery, orders, fields)
	q, orders, fields = manager.SVpcResourceBaseManager.GetOrderBySubQuery(q, subq, lbQ.Field("vpc_id"), userCred, query.VpcFilterListInput, orders, fields)
	return q, orders, fields
}

func (manager *SLoadbalancerResourceBaseManager) GetOrderByFields(query api.LoadbalancerFilterListInput) []string {
	fields := make([]string, 0)
	zoneQuery := api.ZonalFilterListInput{
		ZonalFilterListBase: query.ZonalFilterListBase,
	}
	zoneFields := manager.SZoneResourceBaseManager.GetOrderByFields(zoneQuery)
	fields = append(fields, zoneFields...)
	vpcFields := manager.SVpcResourceBaseManager.GetOrderByFields(query.VpcFilterListInput)
	fields = append(fields, vpcFields...)
	fields = append(fields, query.OrderByLoadbalancer)
	return fields
}

func (manager *SLoadbalancerResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		var err error
		subq := LoadbalancerManager.Query("id", "name", "vpc_id", "zone_id").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("loadbalancer_id"), q.Field("id")))
		if keys.Contains("loadbalancer") {
			q = q.AppendField(subq.Field("name", "loadbalancer"))
		}
		if keys.Contains("vpc") {
			q, err = manager.SVpcResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemExportKeys")
			}
		}
		if keys.ContainsAny(manager.SZoneResourceBaseManager.GetExportKeys()...) {
			q, err = manager.SZoneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemExportKeys")
			}
		}
	}
	return q, nil
}

func (manager *SLoadbalancerResourceBaseManager) GetExportKeys() []string {
	keys := []string{"loadbalancer"}
	keys = append(keys, manager.SZoneResourceBaseManager.GetExportKeys()...)
	keys = append(keys, "vpc")
	return keys
}

func (self *SLoadbalancerResourceBase) GetChangeOwnerCandidateDomainIds() []string {
	lb, _ := self.GetLoadbalancer()
	if lb != nil {
		return lb.GetChangeOwnerCandidateDomainIds()
	}
	return nil
}
