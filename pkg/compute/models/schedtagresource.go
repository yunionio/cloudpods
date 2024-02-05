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

type SSchedtagResourceBase struct {
	// 归属调度标签ID
	SchedtagId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user" json:"schedtag_id"`
}

type SSchedtagResourceBaseManager struct{}

func ValidateSchedtagResourceInput(ctx context.Context, userCred mcclient.TokenCredential, query api.SchedtagResourceInput) (*SSchedtag, api.SchedtagResourceInput, error) {
	tagObj, err := SchedtagManager.FetchByIdOrName(ctx, userCred, query.SchedtagId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, query, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", SchedtagManager.Keyword(), query.SchedtagId)
		} else {
			return nil, query, errors.Wrap(err, "SchedtagManager.FetchByIdOrName")
		}
	}
	query.SchedtagId = tagObj.GetId()
	return tagObj.(*SSchedtag), query, nil
}

func (self *SSchedtagResourceBase) GetSchedtag() *SSchedtag {
	obj, err := SchedtagManager.FetchById(self.SchedtagId)
	if err != nil {
		log.Errorf("fail to fetch sched tag by id %s", err)
		return nil
	}
	return obj.(*SSchedtag)
}

func (manager *SSchedtagResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.SchedtagResourceInfo {
	rows := make([]api.SchedtagResourceInfo, len(objs))
	schedTagIds := make([]string, len(objs))
	for i := range objs {
		var base *SSchedtagResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SSchedtagResourceBase in object %s", objs[i])
			continue
		}
		schedTagIds[i] = base.SchedtagId
	}
	tags := make(map[string]SSchedtag)
	err := db.FetchStandaloneObjectsByIds(SchedtagManager, schedTagIds, tags)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}
	for i := range rows {
		rows[i] = api.SchedtagResourceInfo{}
		if tag, ok := tags[schedTagIds[i]]; ok {
			rows[i].Schedtag = tag.Name
			rows[i].ResourceType = tag.ResourceType
		}
	}
	return rows
}

func (manager *SSchedtagResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SchedtagFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.SchedtagId) > 0 {
		tagObj, _, err := ValidateSchedtagResourceInput(ctx, userCred, query.SchedtagResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateSchedtagResourceInput")
		}
		q = q.Equals("schedtag_id", tagObj.GetId())
	}
	return q, nil
}

func (manager *SSchedtagResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SchedtagFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := SchedtagManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("schedtag_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SSchedtagResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	if field == "schedtag" {
		tagQuery := SchedtagManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(tagQuery.Field("name", field))
		q = q.Join(tagQuery, sqlchemy.Equals(q.Field("schedtag_id"), tagQuery.Field("id")))
		q.GroupBy(tagQuery.Field("name"))
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SSchedtagResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.SchedtagFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	tagQ := SchedtagManager.Query().SubQuery()
	q = q.LeftJoin(tagQ, sqlchemy.Equals(joinField, tagQ.Field("id")))
	q = q.AppendField(tagQ.Field("name").Label("schedtag"))
	q = q.AppendField(tagQ.Field("resource_type").Label("resource_type"))
	orders = append(orders, query.OrderBySchedtag, query.OrderByResourceType)
	fields = append(fields, subq.Field("schedtag"), subq.Field("resource_type"))
	return q, orders, fields
}

func (manager *SSchedtagResourceBaseManager) GetOrderByFields(query api.SchedtagFilterListInput) []string {
	return []string{query.OrderBySchedtag, query.OrderByResourceType}
}

func InsertJointResourceSchedtag(ctx context.Context, jointMan ISchedtagJointManager, resourceId, schedtagId string) (ISchedtagJointModel, error) {
	newTagObj, err := db.NewModelObject(jointMan)
	if err != nil {
		return nil, errors.Wrap(err, "NewModelObject")
	}

	objectKey := jointMan.GetResourceIdKey(jointMan)
	createData := jsonutils.NewDict()
	createData.Add(jsonutils.NewString(schedtagId), "schedtag_id")
	createData.Add(jsonutils.NewString(resourceId), objectKey)
	if err := createData.Unmarshal(newTagObj); err != nil {
		return nil, errors.Wrapf(err, "Create %s joint schedtag", jointMan.Keyword())
	}
	if err := newTagObj.GetModelManager().TableSpec().Insert(ctx, newTagObj); err != nil {
		return nil, errors.Wrap(err, "Insert to database")
	}

	return newTagObj.(ISchedtagJointModel), nil
}
