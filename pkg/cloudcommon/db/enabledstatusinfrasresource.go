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

type SEnabledStatusInfrasResourceBase struct {
	SStatusInfrasResourceBase
	SEnabledResourceBase
}

type SEnabledStatusInfrasResourceBaseManager struct {
	SStatusInfrasResourceBaseManager
	SEnabledResourceBaseManager
}

func NewEnabledStatusInfrasResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SEnabledStatusInfrasResourceBaseManager {
	return SEnabledStatusInfrasResourceBaseManager{
		SStatusInfrasResourceBaseManager: NewStatusInfrasResourceBaseManager(dt, tableName, keyword, keywordPlural),
	}
}

func (self *SEnabledStatusInfrasResourceBase) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) bool {
	return IsDomainAllowPerform(userCred, self, "enable")
}

// 启用资源
func (self *SEnabledStatusInfrasResourceBase) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	err := EnabledPerformEnable(self, ctx, userCred, true)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (self *SEnabledStatusInfrasResourceBase) AllowPerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) bool {
	return IsDomainAllowPerform(userCred, self, "disable")
}

// 禁用资源
func (self *SEnabledStatusInfrasResourceBase) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	err := EnabledPerformEnable(self, ctx, userCred, false)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (manager *SEnabledStatusInfrasResourceBaseManager) AllowGetPropertyStatistics(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return IsAdminAllowGetSpec(userCred, manager, "statistics")
}

func (manager *SEnabledStatusInfrasResourceBaseManager) GetPropertyStatistics(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (map[string]apis.StatusStatistic, error) {
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

func (manager *SEnabledStatusInfrasResourceBaseManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input apis.EnabledStatusInfrasResourceBaseCreateInput,
) (apis.EnabledStatusInfrasResourceBaseCreateInput, error) {
	var err error
	input.StatusInfrasResourceBaseCreateInput, err = manager.SStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStatusInfrasResourceBaseManager.ValidateCreateData")
	}
	return input, nil
}

func (manager *SEnabledStatusInfrasResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.EnabledStatusInfrasResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SEnabledStatusInfrasResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SEnabledStatusInfrasResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.EnabledStatusInfrasResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SEnabledStatusInfrasResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.EnabledStatusInfrasResourceBaseDetails {
	rows := make([]apis.EnabledStatusInfrasResourceBaseDetails, len(objs))
	domainRows := manager.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = apis.EnabledStatusInfrasResourceBaseDetails{
			StatusInfrasResourceBaseDetails: domainRows[i],
		}
	}
	return rows
}

func (model *SEnabledStatusInfrasResourceBase) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.EnabledStatusInfrasResourceBaseUpdateInput,
) (apis.EnabledStatusInfrasResourceBaseUpdateInput, error) {
	var err error
	input.StatusInfrasResourceBaseUpdateInput, err = model.SStatusInfrasResourceBase.ValidateUpdateData(ctx, userCred, query, input.StatusInfrasResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStatusInfrasResourceBase.ValidateUpdateData")
	}
	return input, nil
}
