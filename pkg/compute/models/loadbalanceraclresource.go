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

type SLoadbalancerAclResourceBase struct {
	// 本地Acl ID
	AclId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
}

type SLoadbalancerAclResourceBaseManager struct{}

func (self *SLoadbalancerAclResourceBase) GetAcl() (*SLoadbalancerAcl, error) {
	acl, err := LoadbalancerAclManager.FetchById(self.AclId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetAcl.FetchById(%s)", self.AclId)
	}
	return acl.(*SLoadbalancerAcl), nil
}

func (manager *SLoadbalancerAclResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancerAclResourceInfo {
	rows := make([]api.LoadbalancerAclResourceInfo, len(objs))
	aclIds := make([]string, len(objs))
	for i := range objs {
		var base *SLoadbalancerAclResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SCloudregionResourceBase in object %s", objs[i])
			continue
		}
		aclIds[i] = base.AclId
	}
	acls := make(map[string]SLoadbalancerAcl)
	err := db.FetchStandaloneObjectsByIds(LoadbalancerAclManager, aclIds, &acls)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}
	for i := range rows {
		rows[i] = api.LoadbalancerAclResourceInfo{}
		if acl, ok := acls[aclIds[i]]; ok {
			rows[i].Acl = acl.Name
		}
	}
	return rows
}

func (manager *SLoadbalancerAclResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerAclFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.AclId) > 0 {
		_, err := validators.ValidateModel(ctx, userCred, LoadbalancerAclManager, &query.AclId)
		if err != nil {
			return nil, err
		}
		q = q.Equals("acl_id", query.AclId)
	}
	return q, nil
}

func (manager *SLoadbalancerAclResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerAclFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := LoadbalancerAclManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("acl_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SLoadbalancerAclResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	if field == "acl" {
		aclQuery := LoadbalancerAclManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(aclQuery.Field("name", field))
		q = q.Join(aclQuery, sqlchemy.Equals(q.Field("acl_id"), aclQuery.Field("id")))
		q.GroupBy(aclQuery.Field("name"))
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SLoadbalancerAclResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerAclFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	aclQ := LoadbalancerAclManager.Query().SubQuery()
	q = q.LeftJoin(aclQ, sqlchemy.Equals(joinField, aclQ.Field("id")))
	q = q.AppendField(aclQ.Field("name").Label("acl"))
	orders = append(orders, query.OrderByAcl)
	fields = append(fields, subq.Field("acl"))
	return q, orders, fields
}

func (manager *SLoadbalancerAclResourceBaseManager) GetOrderByFields(query api.LoadbalancerAclFilterListInput) []string {
	return []string{query.OrderByAcl}
}

func (manager *SLoadbalancerAclResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		subq := LoadbalancerAclManager.Query("id", "name").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("acl_id"), subq.Field("id")))
		if keys.Contains("acl") {
			q = q.AppendField(subq.Field("name", "acl"))
		}
	}
	return q, nil
}

func (manager *SLoadbalancerAclResourceBaseManager) GetExportKeys() []string {
	return []string{"acl"}
}
