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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
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

	input.Values = uniqueNonEmptyStrings(input.Values)
	if len(input.Values) == 0 {
		return input, httperrors.NewInputParameterError("values is required")
	}

	return input, nil
}

func uniqueNonEmptyStrings(values []string) []string {
	ret := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if len(v) == 0 {
			continue
		}
		if !utils.IsInStringArray(v, ret) {
			ret = append(ret, v)
		}
	}
	return ret
}

func mergeTagValues(exist, added []string) ([]string, bool) {
	changed := false
	ret := make([]string, len(exist))
	copy(ret, exist)
	for _, v := range added {
		if !utils.IsInStringArray(v, ret) {
			ret = append(ret, v)
			changed = true
		}
	}
	return ret, changed
}

// 批量导入预置标签：同名标签合并去重 values，不存在则新建
func (manager *STagManager) PerformBatchImport(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.TagBatchImportInput,
) (api.TagBatchImportResult, error) {
	result := api.TagBatchImportResult{}
	if len(input.Tags) == 0 {
		return result, httperrors.NewInputParameterError("tags is required")
	}

	// 请求内按名称合并，避免同名重复导入
	merged := make(map[string][]string, len(input.Tags))
	order := make([]string, 0, len(input.Tags))
	for i := range input.Tags {
		name := strings.TrimSpace(input.Tags[i].Name)
		if len(name) == 0 {
			return result, httperrors.NewInputParameterError("tag name is required")
		}
		values := uniqueNonEmptyStrings(input.Tags[i].Values)
		if exist, ok := merged[name]; ok {
			merged[name], _ = mergeTagValues(exist, values)
			continue
		}
		merged[name] = values
		order = append(order, name)
	}

	ownerId := userCred
	for _, name := range order {
		values := merged[name]
		obj, err := manager.FetchByName(ctx, ownerId, name)
		if err != nil {
			if errors.Cause(err) != sql.ErrNoRows {
				return result, errors.Wrapf(err, "FetchByName %s", name)
			}
			if len(values) == 0 {
				return result, httperrors.NewInputParameterError("values is required for new tag %s", name)
			}
			createInput := api.TagCreateInput{}
			createInput.Name = name
			createInput.Values = values
			_, err = db.DoCreate(manager, ctx, userCred, query, jsonutils.Marshal(createInput), ownerId)
			if err != nil {
				return result, errors.Wrapf(err, "DoCreate tag %s", name)
			}
			result.Created++
			continue
		}

		tag := obj.(*STag)
		newValues, changed := mergeTagValues(tag.Values, values)
		if !changed {
			result.Unchanged++
			continue
		}
		_, err = db.Update(tag, func() error {
			tag.Values = newValues
			return nil
		})
		if err != nil {
			return result, errors.Wrapf(err, "Update tag %s values", name)
		}
		db.OpsLog.LogEvent(tag, db.ACT_UPDATE, jsonutils.Marshal(map[string]interface{}{
			"values": newValues,
		}), userCred)
		result.Updated++
	}

	return result, nil
}
