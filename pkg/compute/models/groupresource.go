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

type SGroupResourceBase struct {
	// 实例组ID
	GroupId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

type SGroupResourceBaseManager struct {
}

func ValidateGroupResourceInput(ctx context.Context, userCred mcclient.TokenCredential, input api.GroupResourceInput) (*SGroup, api.GroupResourceInput, error) {
	groupObj, err := GroupManager.FetchByIdOrName(ctx, userCred, input.GroupId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, input, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", GroupManager.Keyword(), input.GroupId)
		} else {
			return nil, input, errors.Wrap(err, "GroupManager.FetchByIdOrName")
		}
	}
	input.GroupId = groupObj.GetId()
	return groupObj.(*SGroup), input, nil
}

func (self *SGroupResourceBase) GetGroup() *SGroup {
	obj, _ := GroupManager.FetchById(self.GroupId)
	if obj != nil {
		return obj.(*SGroup)
	}
	return nil
}

func (manager *SGroupResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.GroupResourceInfo {
	rows := make([]api.GroupResourceInfo, len(objs))
	groupIds := make([]string, len(objs))
	for i := range objs {
		var base *SGroupResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SGroupResourceBase in object %s", objs[i])
			continue
		}
		groupIds[i] = base.GroupId
	}
	groupNames, err := db.FetchIdNameMap2(GroupManager, groupIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail %s", err)
		return rows
	}

	for i := range rows {
		if name, ok := groupNames[groupIds[i]]; ok {
			rows[i].Group = name
		}
	}
	return rows
}

func (manager *SGroupResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GroupFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.GroupId) > 0 {
		groupObj, _, err := ValidateGroupResourceInput(ctx, userCred, query.GroupResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateGroupResourceInput")
		}
		q = q.Equals("group_id", groupObj.GetId())
	}
	return q, nil
}

func (manager *SGroupResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "group":
		groupQuery := GroupManager.Query("name", "id").SubQuery()
		q = q.AppendField(groupQuery.Field("name", field)).Distinct()
		q = q.Join(groupQuery, sqlchemy.Equals(q.Field("group_id"), groupQuery.Field("id")))
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SGroupResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GroupFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := GroupManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("group_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SGroupResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.GroupFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	groupQ := GroupManager.Query().SubQuery()
	q = q.LeftJoin(groupQ, sqlchemy.Equals(joinField, groupQ.Field("id")))
	q = q.AppendField(groupQ.Field("name").Label("group"))
	orders = append(orders, query.OrderByGroup)
	fields = append(fields, subq.Field("group"))
	return q, orders, fields
}

func (manager *SGroupResourceBaseManager) GetOrderByFields(query api.GroupFilterListInput) []string {
	return []string{query.OrderByGroup}
}

func (manager *SGroupResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		subq := GroupManager.Query("id", "name").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("group_id"), subq.Field("id")))
		if keys.Contains("group") {
			q = q.AppendField(subq.Field("name", "group"))
		}
	}
	return q, nil
}

func (manager *SGroupResourceBaseManager) GetExportKeys() []string {
	return []string{"group"}
}
