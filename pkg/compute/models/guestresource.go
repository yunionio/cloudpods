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

type SGuestResourceBase struct {
	GuestId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

type SGuestResourceBaseManager struct {
	SHostResourceBaseManager
}

func ValidateGuestResourceInput(ctx context.Context, userCred mcclient.TokenCredential, input api.ServerResourceInput) (*SGuest, api.ServerResourceInput, error) {
	srvObj, err := GuestManager.FetchByIdOrName(ctx, userCred, input.ServerId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, input, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", GuestManager.Keyword(), input.ServerId)
		} else {
			return nil, input, errors.Wrap(err, "GuestManager.FetchByIdOrName")
		}
	}
	input.ServerId = srvObj.GetId()
	return srvObj.(*SGuest), input, nil
}

func (self *SGuestResourceBase) GetGuest() (*SGuest, error) {
	obj, err := GuestManager.FetchById(self.GuestId)
	if err != nil {
		return nil, err
	}
	return obj.(*SGuest), nil
}

func (self *SGuestResourceBase) GetHost() (*SHost, error) {
	guest, err := self.GetGuest()
	if err != nil {
		return nil, err
	}
	return guest.GetHost()
}

func (self *SGuestResourceBase) GetZone() (*SZone, error) {
	host, err := self.GetHost()
	if err != nil {
		return nil, err
	}
	return host.GetZone()
}

func (self *SGuestResourceBase) GetRegion() (*SCloudregion, error) {
	host, err := self.GetHost()
	if err != nil {
		return nil, err
	}
	return host.GetRegion()
}

func (manager *SGuestResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.GuestResourceInfo {
	rows := make([]api.GuestResourceInfo, len(objs))
	guestIds := make([]string, len(objs))
	for i := range objs {
		var base *SGuestResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SGuestResourceBase in object %s", objs[i])
			continue
		}
		guestIds[i] = base.GuestId
	}
	guests := make(map[string]SGuest)
	err := db.FetchStandaloneObjectsByIds(GuestManager, guestIds, guests)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	hostList := make([]interface{}, len(rows))
	for i := range rows {
		if guest, ok := guests[guestIds[i]]; ok {
			rows[i].Guest = guest.Name
			rows[i].HostId = guest.HostId
		}
		hostList[i] = &SHostResourceBase{rows[i].HostId}
	}

	hostRows := manager.SHostResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, hostList, fields, isList)
	for i := range rows {
		rows[i].HostResourceInfo = hostRows[i]
	}
	return rows
}

func (manager *SGuestResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ServerFilterListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	if len(query.ServerId) > 0 {
		guestObj, _, err := ValidateGuestResourceInput(ctx, userCred, query.ServerResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateGuestResourceInput")
		}
		q = q.Equals("guest_id", guestObj.GetId())
	}
	guestQ := GuestManager.Query("id").Snapshot()
	guestQ, err = manager.SHostResourceBaseManager.ListItemFilter(ctx, guestQ, userCred, query.HostFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SHostResourceBaseManager.ListItemFilter")
	}
	if guestQ.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("guest_id"), guestQ.SubQuery()))
	}
	return q, nil
}

func (manager *SGuestResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "guest", "server":
		guestQuery := GuestManager.Query("name", "id").SubQuery()
		q = q.AppendField(guestQuery.Field("name", field)).Distinct()
		q = q.Join(guestQuery, sqlchemy.Equals(q.Field("guest_id"), guestQuery.Field("id")))
		return q, nil
	default:
		guests := GuestManager.Query("id", "host_id").SubQuery()
		q = q.LeftJoin(guests, sqlchemy.Equals(q.Field("guest_id"), guests.Field("id")))
		q, err := manager.SHostResourceBaseManager.QueryDistinctExtraField(q, field)
		if err == nil {
			return q, nil
		}
	}
	return q, httperrors.ErrNotFound
}

func (manager *SGuestResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ServerFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := GuestManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("guest_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SGuestResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.ServerFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	guestQ := GuestManager.Query().SubQuery()
	q = q.LeftJoin(guestQ, sqlchemy.Equals(joinField, guestQ.Field("id")))
	q = q.AppendField(guestQ.Field("name").Label("server"))
	orders = append(orders, query.OrderByServer)
	fields = append(fields, subq.Field("server"))
	q, orders, fields = manager.SHostResourceBaseManager.GetOrderBySubQuery(q, subq, guestQ.Field("host_id"), userCred, query.HostFilterListInput, orders, fields)
	return q, orders, fields
}

func (manager *SGuestResourceBaseManager) GetOrderByFields(query api.ServerFilterListInput) []string {
	orders := make([]string, 0)
	hostOrders := manager.SHostResourceBaseManager.GetOrderByFields(query.HostFilterListInput)
	orders = append(orders, hostOrders...)
	orders = append(orders, query.OrderByServer)
	return orders
}

func (manager *SGuestResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		var err error
		subq := GuestManager.Query("id", "name", "host_id").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("id"), subq.Field("id")))
		if keys.Contains("guest") {
			q = q.AppendField(subq.Field("name", "guest"))
		}
		if keys.ContainsAny(manager.SHostResourceBaseManager.GetExportKeys()...) {
			q, err = manager.SHostResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SHostResourceBaseManager.ListItemExportKeys")
			}
		}
	}
	return q, nil
}

func (manager *SGuestResourceBaseManager) GetExportKeys() []string {
	keys := []string{"guest"}
	keys = append(keys, manager.SHostResourceBaseManager.GetExportKeys()...)
	return keys
}

func (self *SGuestResourceBase) GetChangeOwnerCandidateDomainIds() []string {
	guest, _ := self.GetGuest()
	if guest != nil {
		return guest.GetChangeOwnerCandidateDomainIds()
	}
	return nil
}
