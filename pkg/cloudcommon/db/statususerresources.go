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

type SStatusUserResourceBase struct {
	SUserResourceBase
	SStatusResourceBase
}

type SStatusUserResourceBaseManager struct {
	SUserResourceBaseManager
	SStatusResourceBaseManager
}

func NewStatusUserResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SStatusUserResourceBaseManager {
	return SStatusUserResourceBaseManager{
		SUserResourceBaseManager: NewUserResourceBaseManager(dt, tableName, keyword, keywordPlural),
	}
}

func (model *SStatusUserResourceBase) SetStatus(userCred mcclient.TokenCredential, status string, reason string) error {
	return statusBaseSetStatus(model, userCred, status, reason)
}

func (manager *SStatusUserResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input apis.StatusUserResourceCreateInput) (apis.StatusUserResourceCreateInput, error) {
	var err error
	input.UserResourceCreateInput, err = manager.SUserResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.UserResourceCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (manager *SStatusUserResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.StatusUserResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SUserResourceBaseManager.ListItemFilter(ctx, q, userCred, query.UserResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SUserResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SStatusResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SStatusUserResourceBaseManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query apis.StatusUserResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := manager.SUserResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.UserResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SUserResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SStatusResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SStatusUserResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SUserResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SStatusUserResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.StatusUserResourceDetails {
	rows := make([]apis.StatusUserResourceDetails, len(objs))
	userRows := manager.SUserResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = apis.StatusUserResourceDetails{
			UserResourceDetails: userRows[i],
		}
	}
	return rows
}

func (model *SStatusUserResourceBase) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (apis.StatusUserResourceDetails, error) {
	return apis.StatusUserResourceDetails{}, nil
}
