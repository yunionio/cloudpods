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

type SSnapshotPolicyResourceBase struct {
	// 本地快照策略ID
	SnapshotpolicyId string `width:"36" charset:"ascii" nullable:"false" create:"required"  index:"true" list:"user" json:"snapshotpolicy_id"`
}

type SSnapshotPolicyResourceBaseManager struct{}

func ValidateSnapshotPolicyResourceInput(userCred mcclient.TokenCredential, query api.SnapshotPolicyResourceInput) (*SSnapshotPolicy, api.SnapshotPolicyResourceInput, error) {
	snapPolicyObj, err := SnapshotPolicyManager.FetchByIdOrName(userCred, query.SnapshotpolicyId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, query, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", SnapshotPolicyManager.Keyword(), query.SnapshotpolicyId)
		} else {
			return nil, query, errors.Wrap(err, "SnapshotPolicyManager.FetchByIdOrName")
		}
	}
	query.SnapshotpolicyId = snapPolicyObj.GetId()
	return snapPolicyObj.(*SSnapshotPolicy), query, nil
}

func (self *SSnapshotPolicyResourceBase) GetSnapshotPolicy() *SSnapshotPolicy {
	spObj, err := SnapshotPolicyManager.FetchById(self.SnapshotpolicyId)
	if err != nil {
		log.Errorf("failed to find snapshot policy %s error: %v", self.SnapshotpolicyId, err)
		return nil
	}
	return spObj.(*SSnapshotPolicy)
}

func (manager *SSnapshotPolicyResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.SnapshotPolicyResourceInfo {
	rows := make([]api.SnapshotPolicyResourceInfo, len(objs))
	snapshotPolicyIds := make([]string, len(objs))
	for i := range objs {
		var base *SSnapshotPolicyResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SSnapshotPolicyResourceBase in object %s", objs[i])
			continue
		}
		snapshotPolicyIds[i] = base.SnapshotpolicyId
	}
	snapshotPolicyNames, err := db.FetchIdNameMap2(SnapshotPolicyManager, snapshotPolicyIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail %s", err)
		return rows
	}
	for i := range rows {
		if name, ok := snapshotPolicyNames[snapshotPolicyIds[i]]; ok {
			rows[i].Snapshotpolicy = name
		}
	}
	return rows
}

func (manager *SSnapshotPolicyResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SnapshotPolicyFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.SnapshotpolicyId) > 0 {
		snapPObj, _, err := ValidateSnapshotPolicyResourceInput(userCred, query.SnapshotPolicyResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateSnapshotPolicyResourceInput")
		}
		q = q.Equals("snapshotpolicy_id", snapPObj.GetId())
	}
	return q, nil
}

func (manager *SSnapshotPolicyResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SnapshotPolicyFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := SnapshotPolicyManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("snapshotpolicy_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SSnapshotPolicyResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	if field == "snapshotpolicy" {
		snapPolicyQuery := SnapshotPolicyManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(snapPolicyQuery.Field("name", field))
		q = q.Join(snapPolicyQuery, sqlchemy.Equals(q.Field("snapshotpolicy_id"), snapPolicyQuery.Field("id")))
		q.GroupBy(snapPolicyQuery.Field("name"))
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SSnapshotPolicyResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.SnapshotPolicyFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	snapQ := SnapshotPolicyManager.Query().SubQuery()
	q = q.LeftJoin(snapQ, sqlchemy.Equals(joinField, snapQ.Field("id")))
	q = q.AppendField(snapQ.Field("name").Label("snapshotpolicy"))
	orders = append(orders, query.OrderBySnapshotpolicy)
	fields = append(fields, subq.Field("snapshotpolicy"))
	return q, orders, fields
}

func (manager *SSnapshotPolicyResourceBaseManager) GetOrderByFields(query api.SnapshotPolicyFilterListInput) []string {
	return []string{query.OrderBySnapshotpolicy}
}

func (manager *SSnapshotPolicyResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		subq := SnapshotPolicyManager.Query("id", "name").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("snapshotpolicy_id"), subq.Field("id")))
		if keys.Contains("snapshotpolicy") {
			q = q.AppendField(subq.Field("name", "snapshotpolicy"))
		}
	}
	return q, nil
}

func (manager *SSnapshotPolicyResourceBaseManager) GetExportKeys() []string {
	return []string{"snapshotpolicy"}
}
