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
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SEnabledStatusDomainLevelResourceBase struct {
	SStatusDomainLevelResourceBase
	SEnabledResourceBase
}

type SEnabledStatusDomainLevelResourceBaseManager struct {
	SStatusDomainLevelResourceBaseManager
	SEnabledResourceBaseManager
}

func NewEnabledStatusDomainLevelResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SEnabledStatusDomainLevelResourceBaseManager {
	return SEnabledStatusDomainLevelResourceBaseManager{
		SStatusDomainLevelResourceBaseManager: NewStatusDomainLevelResourceBaseManager(dt, tableName, keyword, keywordPlural),
	}
}

func (self *SEnabledStatusDomainLevelResourceBase) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) bool {
	return IsDomainAllowPerform(userCred, self, "enable")
}

// 启用资源
func (self *SEnabledStatusDomainLevelResourceBase) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	err := EnabledPerformEnable(self, ctx, userCred, true)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (self *SEnabledStatusDomainLevelResourceBase) AllowPerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) bool {
	return IsDomainAllowPerform(userCred, self, "disable")
}

// 禁用资源
func (self *SEnabledStatusDomainLevelResourceBase) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	err := EnabledPerformEnable(self, ctx, userCred, false)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (manager *SEnabledStatusDomainLevelResourceBaseManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input apis.EnabledStatusDomainLevelResourceCreateInput,
) (apis.EnabledStatusDomainLevelResourceCreateInput, error) {
	var err error
	input.StatusDomainLevelResourceCreateInput, err = manager.SStatusDomainLevelResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusDomainLevelResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ValidateCreateData")
	}
	return input, nil
}

func (manager *SEnabledStatusDomainLevelResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.EnabledStatusDomainLevelResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusDomainLevelResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusDomainLevelResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDomainLevelResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SEnabledStatusDomainLevelResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusDomainLevelResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SEnabledStatusDomainLevelResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.EnabledStatusDomainLevelResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusDomainLevelResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusDomainLevelResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDomainLevelResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SEnabledStatusDomainLevelResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.EnabledStatusDomainLevelResourceDetails {
	rows := make([]apis.EnabledStatusDomainLevelResourceDetails, len(objs))
	domainRows := manager.SStatusDomainLevelResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = apis.EnabledStatusDomainLevelResourceDetails{
			StatusDomainLevelResourceDetails: domainRows[i],
		}
	}
	return rows
}

func (model *SEnabledStatusDomainLevelResourceBase) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (apis.EnabledStatusDomainLevelResourceDetails, error) {
	return apis.EnabledStatusDomainLevelResourceDetails{}, nil
}

func (model *SEnabledStatusDomainLevelResourceBase) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.EnabledStatusDomainLevelResourceBaseUpdateInput,
) (apis.EnabledStatusDomainLevelResourceBaseUpdateInput, error) {
	var err error
	input.StatusDomainLevelResourceBaseUpdateInput, err = model.SStatusDomainLevelResourceBase.ValidateUpdateData(ctx, userCred, query, input.StatusDomainLevelResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStatusDomainLevelResourceBase.ValidateUpdateData")
	}
	return input, nil
}
