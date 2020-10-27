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

type SNetworkResourceBase struct {
	NetworkId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"network_id"`
}

type SNetworkResourceBaseManager struct {
	SWireResourceBaseManager
}

func ValidateNetworkResourceInput(userCred mcclient.TokenCredential, query api.NetworkResourceInput) (*SNetwork, api.NetworkResourceInput, error) {
	netObj, err := NetworkManager.FetchByIdOrName(userCred, query.NetworkId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, query, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", NetworkManager.Keyword(), query.NetworkId)
		} else {
			return nil, query, errors.Wrap(err, "NetworkManager.FetchByIdOrName")
		}
	}
	query.NetworkId = netObj.GetId()
	return netObj.(*SNetwork), query, nil
}

func (self *SNetworkResourceBase) GetNetwork() *SNetwork {
	obj, _ := NetworkManager.FetchById(self.NetworkId)
	if obj != nil {
		return obj.(*SNetwork)
	}
	return nil
}

func (self *SNetworkResourceBase) GetWire() *SWire {
	net := self.GetNetwork()
	if net != nil {
		return net.GetWire()
	}
	return nil
}

func (self *SNetworkResourceBase) GetZone() *SZone {
	wire := self.GetWire()
	if wire != nil {
		return wire.GetZone()
	}
	return nil
}

func (self *SNetworkResourceBase) GetVpc() *SVpc {
	wire := self.GetWire()
	if wire != nil {
		return wire.GetVpc()
	}
	return nil
}

func (self *SNetworkResourceBase) GetRegion() *SCloudregion {
	vpc := self.GetVpc()
	if vpc == nil {
		return nil
	}
	region, _ := vpc.GetRegion()
	return region
}

func (self *SNetworkResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.NetworkResourceInfo, error) {
	return api.NetworkResourceInfo{}, nil
}

func (manager *SNetworkResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.NetworkResourceInfo {
	rows := make([]api.NetworkResourceInfo, len(objs))
	netIds := make([]string, len(objs))
	for i := range objs {
		var base *SNetworkResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SCloudregionResourceBase in object %s", objs[i])
			continue
		}
		netIds[i] = base.NetworkId
	}
	networks := make(map[string]SNetwork)
	err := db.FetchStandaloneObjectsByIds(NetworkManager, netIds, networks)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	wireList := make([]interface{}, len(rows))
	for i := range rows {
		rows[i] = api.NetworkResourceInfo{}
		if _, ok := networks[netIds[i]]; ok {
			network := networks[netIds[i]]
			rows[i].Network = network.Name
			rows[i].WireId = network.WireId
		}
		wireList[i] = &SWireResourceBase{rows[i].WireId}
	}

	wireRows := manager.SWireResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, wireList, fields, isList)
	for i := range rows {
		rows[i].WireResourceInfo = wireRows[i]
	}
	return rows
}

func (manager *SNetworkResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NetworkFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.NetworkId) > 0 {
		netObj, _, err := ValidateNetworkResourceInput(userCred, query.NetworkResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateNetworkResourceInput")
		}
		q = q.Equals("network_id", netObj.GetId())
	}
	netQ := NetworkManager.Query("id").Snapshot()
	netQ, err := manager.SWireResourceBaseManager.ListItemFilter(ctx, netQ, userCred, query.WireFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SWireResourceBaseManager.ListItemFilter")
	}
	if netQ.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("network_id"), netQ.SubQuery()))
	}
	return q, nil
}

func (manager *SNetworkResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "network":
		netQuery := NetworkManager.Query("name", "id").SubQuery()
		q = q.AppendField(netQuery.Field("name", field)).Distinct()
		q = q.Join(netQuery, sqlchemy.Equals(q.Field("network_id"), netQuery.Field("id")))
		return q, nil
	default:
		nets := NetworkManager.Query("id", "wire_id").SubQuery()
		q = q.LeftJoin(nets, sqlchemy.Equals(q.Field("network_id"), nets.Field("id")))
		q, err := manager.SWireResourceBaseManager.QueryDistinctExtraField(q, field)
		if err == nil {
			return q, nil
		}
	}
	return q, httperrors.ErrNotFound
}

func (manager *SNetworkResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NetworkFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := NetworkManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("network_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SNetworkResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.NetworkFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	netQ := NetworkManager.Query().SubQuery()
	q = q.LeftJoin(netQ, sqlchemy.Equals(joinField, netQ.Field("id")))
	q = q.AppendField(netQ.Field("name").Label("network"))
	orders = append(orders, query.OrderByNetwork)
	fields = append(fields, subq.Field("network"))
	q, orders, fields = manager.SWireResourceBaseManager.GetOrderBySubQuery(q, subq, netQ.Field("wire_id"), userCred, query.WireFilterListInput, orders, fields)
	return q, orders, fields
}

func (manager *SNetworkResourceBaseManager) GetOrderByFields(query api.NetworkFilterListInput) []string {
	orders := make([]string, 0)
	wireOrders := manager.SWireResourceBaseManager.GetOrderByFields(query.WireFilterListInput)
	orders = append(orders, wireOrders...)
	orders = append(orders, query.OrderByNetwork)
	return orders
}

func (manager *SNetworkResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		var err error
		subq := NetworkManager.Query("id", "name", "wire_id").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("network_id"), subq.Field("id")))
		if keys.Contains("network") {
			q = q.AppendField(subq.Field("name", "network"))
		}
		if keys.ContainsAny(manager.SWireResourceBaseManager.GetExportKeys()...) {
			q, err = manager.SWireResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SWireResourceBaseManager.ListItemExportKeys")
			}
		}
	}
	return q, nil
}

func (manager *SNetworkResourceBaseManager) GetExportKeys() []string {
	keys := []string{"network"}
	keys = append(keys, manager.SWireResourceBaseManager.GetExportKeys()...)
	return keys
}

func (self *SNetworkResourceBase) GetChangeOwnerCandidateDomainIds() []string {
	network := self.GetNetwork()
	if network != nil {
		return network.GetChangeOwnerCandidateDomainIds()
	}
	return nil
}
