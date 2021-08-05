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

package db

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SStatusInfrasResourceBase struct {
	SInfrasResourceBase
	SStatusResourceBase
}

type SStatusInfrasResourceBaseManager struct {
	SInfrasResourceBaseManager
	SStatusResourceBaseManager
}

func NewStatusInfrasResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SStatusInfrasResourceBaseManager {
	return SStatusInfrasResourceBaseManager{
		SInfrasResourceBaseManager: NewInfrasResourceBaseManager(dt, tableName, keyword, keywordPlural),
	}
}

func (model *SStatusInfrasResourceBase) AllowGetDetailsStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return model.IsOwner(userCred) || IsDomainAllowGetSpec(userCred, model, "status")
}

func (self *SStatusInfrasResourceBase) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformStatusInput) bool {
	return IsDomainAllowPerform(userCred, self, "status")
}

func (self *SStatusInfrasResourceBase) GetIStatusInfrasModel() IStatusInfrasModel {
	return self.GetVirtualObject().(IStatusInfrasModel)
}

func (manager *SStatusInfrasResourceBaseManager) AllowGetPropertyStatistics(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return IsAdminAllowGetSpec(userCred, manager, "statistics")
}

func (manager *SStatusInfrasResourceBaseManager) GetPropertyStatistics(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (map[string]apis.StatusStatistic, error) {
	im, ok := manager.GetVirtualObject().(IModelManager)
	if !ok {
		im = manager
	}

	var err error
	q := manager.Query()
	q, err = ListItemQueryFilters(im, ctx, q, userCred, query, policy.PolicyActionList)
	if err != nil {
		return nil, err
	}

	sq := q.SubQuery()
	statQ := sq.Query(sq.Field("status"), sqlchemy.COUNT("total_count", sq.Field("id")))
	statQ = statQ.GroupBy(sq.Field("status"))

	ret := []struct {
		Status     string
		TotalCount int64
	}{}
	err = statQ.All(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "q.All")
	}
	result := map[string]apis.StatusStatistic{}
	for _, s := range ret {
		result[s.Status] = apis.StatusStatistic{
			TotalCount: s.TotalCount,
		}
	}
	return result, nil
}

// 更新资源状态
func (self *SStatusInfrasResourceBase) PerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformStatusInput) (jsonutils.JSONObject, error) {
	err := StatusBasePerformStatus(self.GetIStatusInfrasModel(), userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "StatusBasePerformStatus")
	}
	return nil, nil
}

func (model *SStatusInfrasResourceBase) SetStatus(userCred mcclient.TokenCredential, status string, reason string) error {
	return statusBaseSetStatus(model.GetIStatusInfrasModel(), userCred, status, reason)
}

func (manager *SStatusInfrasResourceBaseManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input apis.StatusInfrasResourceBaseCreateInput,
) (apis.StatusInfrasResourceBaseCreateInput, error) {
	var err error
	input.InfrasResourceBaseCreateInput, err = manager.SInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.InfrasResourceBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SInfrasResourceBaseManager.ValidateCreateData")
	}
	return input, nil
}

func (manager *SStatusInfrasResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.StatusInfrasResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.InfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SInfrasResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SStatusResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SStatusInfrasResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.StatusInfrasResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.InfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SInfrasResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SStatusResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SStatusInfrasResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SStatusInfrasResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.StatusInfrasResourceBaseDetails {
	rows := make([]apis.StatusInfrasResourceBaseDetails, len(objs))
	infRows := manager.SInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = apis.StatusInfrasResourceBaseDetails{
			InfrasResourceBaseDetails: infRows[i],
		}
	}
	return rows
}

func (model *SStatusInfrasResourceBase) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.StatusInfrasResourceBaseUpdateInput,
) (apis.StatusInfrasResourceBaseUpdateInput, error) {
	var err error
	input.InfrasResourceBaseUpdateInput, err = model.SInfrasResourceBase.ValidateUpdateData(ctx, userCred, query, input.InfrasResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SInfrasResourceBase.ValidateUpdateData")
	}
	return input, nil
}
