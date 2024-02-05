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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SAccessGroupResourceBase struct {
	// 权限组Id
	AccessGroupId string `width:"36" charset:"ascii" nullable:"false" create:"required" index:"true" list:"user"`
}

type SAccessGroupResourceBaseManager struct{}

func (self *SAccessGroupResourceBase) GetAccessGroup() (*SAccessGroup, error) {
	group, err := AccessGroupManager.FetchById(self.AccessGroupId)
	if err != nil {
		return nil, errors.Wrapf(err, "AccessGroupManager.FetchById(%s)", self.AccessGroupId)
	}
	return group.(*SAccessGroup), nil
}

func (manager *SAccessGroupResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.AccessGroupResourceInfo {
	rows := make([]api.AccessGroupResourceInfo, len(objs))
	groupIds := make([]string, len(objs))
	for i := range objs {
		var base *SAccessGroupResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SAccessGroupResourceBase in object %s", objs[i])
			continue
		}
		groupIds[i] = base.AccessGroupId
	}
	groupNames, err := db.FetchIdNameMap2(AccessGroupManager, groupIds)
	if err != nil {
		return rows
	}
	for i := range rows {
		rows[i].AccessGroup, _ = groupNames[groupIds[i]]
	}
	return rows
}

func (manager *SAccessGroupResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AccessGroupFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.AccessGroupId) > 0 {
		_, err := validators.ValidateModel(ctx, userCred, AccessGroupManager, &query.AccessGroupId)
		if err != nil {
			return nil, err
		}
		q = q.Equals("access_group_id", query.AccessGroupId)
	}
	return q, nil
}

func (manager *SAccessGroupResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AccessGroupFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := AccessGroupManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("access_group_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SAccessGroupResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	if field == "access_group" {
		groupQuery := AccessGroupManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(groupQuery.Field("name", field))
		q = q.Join(groupQuery, sqlchemy.Equals(q.Field("access_group_id"), groupQuery.Field("id")))
		q.GroupBy(groupQuery.Field("name"))
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SAccessGroupResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.AccessGroupFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	groupQ := AccessGroupManager.Query().SubQuery()
	q = q.LeftJoin(groupQ, sqlchemy.Equals(joinField, groupQ.Field("id")))
	q = q.AppendField(groupQ.Field("name").Label("access_group"))
	orders = append(orders, query.OrderByAccessGroup)
	fields = append(fields, subq.Field("access_group"))
	return q, orders, fields
}

func (manager *SAccessGroupResourceBaseManager) GetOrderByFields(query api.AccessGroupFilterListInput) []string {
	return []string{query.OrderByAccessGroup}
}

func (manager *SAccessGroupResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		subq := AccessGroupManager.Query("id", "name").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("access_group_id"), subq.Field("id")))
		q = q.AppendField(subq.Field("name", "access_group"))
	}
	return q, nil
}

func (manager *SAccessGroupResourceBaseManager) GetExportKeys() []string {
	return []string{"access_group"}
}
