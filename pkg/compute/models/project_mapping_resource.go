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
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SProjectMappingResourceBase struct {
	ProjectMappingId   string            `width:"36" charset:"ascii" nullable:"true" create:"optional" index:"true" list:"user" json:"project_mapping_id"`
	EnableProjectSync  tristate.TriState `default:"false" list:"user" create:"optional"`
	EnableResourceSync tristate.TriState `default:"true" list:"user" create:"optional"`
}

type SProjectMappingResourceBaseManager struct{}

func (manager *SProjectMappingResourceBaseManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.SProjectMappingResourceInput,
) (api.SProjectMappingResourceInput, error) {
	if len(input.ProjectMappingId) > 0 {
		_, err := validators.ValidateModel(ctx, userCred, ProjectMappingManager, &input.ProjectMappingId)
		if err != nil {
			return input, err
		}
	}
	return input, nil
}

func (self *SProjectMappingResourceBase) GetProjectMapping() (*SProjectMapping, error) {
	pm, err := ProjectMappingManager.FetchById(self.ProjectMappingId)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchById")
	}
	return pm.(*SProjectMapping), nil
}

func (manager *SProjectMappingResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ProjectMappingResourceInfo {
	rows := make([]api.ProjectMappingResourceInfo, len(objs))
	pmIds := make([]string, len(objs))
	for i := range objs {
		var base *SProjectMappingResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SProjectMappingResourceBase in object %s", objs[i])
			continue
		}
		pmIds[i] = base.ProjectMappingId
	}
	pmNames, err := db.FetchIdNameMap2(ProjectMappingManager, pmIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail %s", err)
		return rows
	}
	for i := range rows {
		if name, ok := pmNames[pmIds[i]]; ok {
			rows[i].ProjectMapping = name
		}
	}
	return rows
}

func (manager *SProjectMappingResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ProjectMappingFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.ProjectMappingId) > 0 {
		_, err := validators.ValidateModel(ctx, userCred, ProjectMappingManager, &query.ProjectMappingId)
		if err != nil {
			return nil, err
		}
		q = q.Equals("project_mapping_id", query.ProjectMappingId)
	}
	return q, nil
}

func (manager *SProjectMappingResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ProjectMappingFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := ProjectMappingManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("project_mapping_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SProjectMappingResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	if field == "project_mapping" {
		mpQuery := ProjectMappingManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(mpQuery.Field("name", field))
		q = q.Join(mpQuery, sqlchemy.Equals(q.Field("project_mapping_id"), mpQuery.Field("id")))
		q.GroupBy(mpQuery.Field("name"))
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SProjectMappingResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.ProjectMappingFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	mpQ := ProjectMappingManager.Query().SubQuery()
	q = q.LeftJoin(mpQ, sqlchemy.Equals(joinField, mpQ.Field("id")))
	q = q.AppendField(mpQ.Field("name").Label("project_mapping"))
	orders = append(orders, query.OrderByProjectMapping)
	fields = append(fields, subq.Field("project_mapping"))
	return q, orders, fields
}

func (manager *SProjectMappingResourceBaseManager) GetOrderByFields(query api.ProjectMappingFilterListInput) []string {
	return []string{query.OrderByProjectMapping}
}

func (manager *SProjectMappingResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		subq := ProjectMappingManager.Query("id", "name").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("project_mapping_id"), subq.Field("id")))
		q = q.AppendField(subq.Field("name", "project_mapping"))
	}
	return q, nil
}

func (manager *SProjectMappingResourceBaseManager) GetExportKeys() []string {
	return []string{"project_mapping"}
}
