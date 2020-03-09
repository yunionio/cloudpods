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

func (self *SGuestResourceBase) GetGuest() *SGuest {
	obj, _ := GuestManager.FetchById(self.GuestId)
	if obj != nil {
		return obj.(*SGuest)
	}
	return nil
}

func (self *SGuestResourceBase) GetHost() *SHost {
	guest := self.GetGuest()
	if guest != nil {
		return guest.GetHost()
	}
	return nil
}

func (self *SGuestResourceBase) GetZone() *SZone {
	host := self.GetHost()
	if host != nil {
		return host.GetZone()
	}
	return nil
}

func (self *SGuestResourceBase) GetRegion() *SCloudregion {
	host := self.GetHost()
	if host == nil {
		return nil
	}
	region := host.GetRegion()
	return region
}

func (self *SGuestResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) api.GuestResourceInfo {
	return api.GuestResourceInfo{}
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
	query api.GuestFilterListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	if len(query.Server) > 0 {
		guestObj, err := GuestManager.FetchByIdOrName(userCred, query.Server)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(GuestManager.Keyword(), query.Server)
			} else {
				return nil, errors.Wrap(err, "GuestManager.FetchByIdOrName")
			}
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
	query api.GuestFilterListInput,
) (*sqlchemy.SQuery, error) {
	q, orders, fields := manager.GetOrderBySubQuery(q, userCred, query)
	if len(orders) > 0 {
		q = db.OrderByFields(q, orders, fields)
	}
	return q, nil
}

func (manager *SGuestResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestFilterListInput,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	guestQ := GuestManager.Query("id", "name")
	var orders []string
	var fields []sqlchemy.IQueryField

	if db.NeedOrderQuery(manager.SHostResourceBaseManager.GetOrderByFields(query.HostFilterListInput)) {
		var hostOrders []string
		var hostFields []sqlchemy.IQueryField
		guestQ, hostOrders, hostFields = manager.SHostResourceBaseManager.GetOrderBySubQuery(guestQ, userCred, query.HostFilterListInput)
		if len(hostOrders) > 0 {
			orders = append(orders, hostOrders...)
			fields = append(fields, hostFields...)
		}
	}
	if db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		subq := guestQ.SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("guest_id"), subq.Field("id")))
		if db.NeedOrderQuery([]string{query.OrderByServer}) {
			orders = append(orders, query.OrderByServer)
			fields = append(fields, subq.Field("name"))
		}
	}
	return q, orders, fields
}

func (manager *SGuestResourceBaseManager) GetOrderByFields(query api.GuestFilterListInput) []string {
	orders := make([]string, 0)
	hostOrders := manager.SHostResourceBaseManager.GetOrderByFields(query.HostFilterListInput)
	orders = append(orders, hostOrders...)
	orders = append(orders, query.OrderByServer)
	return orders
}
