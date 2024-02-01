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

type SStatusDomainLevelResourceBase struct {
	SDomainLevelResourceBase
	SStatusResourceBase
}

type SStatusDomainLevelResourceBaseManager struct {
	SDomainLevelResourceBaseManager
	SStatusResourceBaseManager
}

func NewStatusDomainLevelResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SStatusDomainLevelResourceBaseManager {
	return SStatusDomainLevelResourceBaseManager{
		SDomainLevelResourceBaseManager: NewDomainLevelResourceBaseManager(dt, tableName, keyword, keywordPlural),
	}
}

func (self *SStatusDomainLevelResourceBase) GetIStatusDomainLevelModel() IStatusDomainLevelModel {
	return self.GetVirtualObject().(IStatusDomainLevelModel)
}

// 更新资源状态
func (self *SStatusDomainLevelResourceBase) PerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformStatusInput) (jsonutils.JSONObject, error) {
	err := StatusBasePerformStatus(ctx, self.GetIStatusDomainLevelModel(), userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "StatusBasePerformStatus")
	}
	return nil, nil
}

func (model *SStatusDomainLevelResourceBase) SetStatus(ctx context.Context, userCred mcclient.TokenCredential, status string, reason string) error {
	return statusBaseSetStatus(ctx, model.GetIStatusDomainLevelModel(), userCred, status, reason)
}

func (manager *SStatusDomainLevelResourceBaseManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input apis.StatusDomainLevelResourceCreateInput,
) (apis.StatusDomainLevelResourceCreateInput, error) {
	var err error
	input.DomainLevelResourceCreateInput, err = manager.SDomainLevelResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.DomainLevelResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ValidateCreateData")
	}
	return input, nil
}

func (manager *SStatusDomainLevelResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.StatusDomainLevelResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SDomainLevelResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DomainLevelResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDomainLevelResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SStatusResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SStatusDomainLevelResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.StatusDomainLevelResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SDomainLevelResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.DomainLevelResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDomainLevelResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SStatusResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SStatusDomainLevelResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SDomainLevelResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SStatusDomainLevelResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.StatusDomainLevelResourceDetails {
	rows := make([]apis.StatusDomainLevelResourceDetails, len(objs))
	domainRows := manager.SDomainLevelResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = apis.StatusDomainLevelResourceDetails{
			DomainLevelResourceDetails: domainRows[i],
		}
	}
	return rows
}

func (model *SStatusDomainLevelResourceBase) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.StatusDomainLevelResourceBaseUpdateInput,
) (apis.StatusDomainLevelResourceBaseUpdateInput, error) {
	var err error
	input.DomainLevelResourceBaseUpdateInput, err = model.SDomainLevelResourceBase.ValidateUpdateData(ctx, userCred, query, input.DomainLevelResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SDomainLevelResourceBase.ValidateUpdateData")
	}
	return input, nil
}
