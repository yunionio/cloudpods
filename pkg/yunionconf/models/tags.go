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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/yunionconf"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=tag
// +onecloud:swagger-gen-model-plural=tags
type STagManager struct {
	db.SInfrasResourceBaseManager
}

var (
	TagManager *STagManager
)

func init() {
	TagManager = &STagManager{
		SInfrasResourceBaseManager: db.NewInfrasResourceBaseManager(
			STag{},
			"tags_tbl",
			"tag",
			"tags",
		),
	}
	TagManager.SetVirtualObject(TagManager)
}

type STag struct {
	db.SInfrasResourceBase

	Values []string `charset:"utf8" get:"user" list:"user" update:"admin" create:"required"`
}

// 预置标签列表
func (manager *STagManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.TagListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.InfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SInfrasResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *STagManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.TagListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.InfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SInfrasResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *STagManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *STagManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.TagDetails {
	rows := make([]api.TagDetails, len(objs))

	stdRows := manager.SInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.TagDetails{
			InfrasResourceBaseDetails: stdRows[i],
		}
	}

	return rows
}

func (manager *STagManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *api.TagCreateInput,
) (*api.TagCreateInput, error) {
	var err error

	input.InfrasResourceBaseCreateInput, err = manager.SInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.InfrasResourceBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SInfrasResourceBaseManager.ValidateCreateData")
	}

	return input, nil
}
