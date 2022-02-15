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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SStandaloneResourceBase struct {
	SStandaloneAnonResourceBase

	// 资源名称
	Name string `width:"128" charset:"utf8" nullable:"false" index:"true" list:"user" update:"user" create:"required" json:"name"`
}

type SStandaloneResourceBaseManager struct {
	SStandaloneAnonResourceBaseManager

	NameRequireAscii bool
	NameLength       int
}

func NewStandaloneResourceBaseManager(
	dt interface{},
	tableName string,
	keyword string,
	keywordPlural string,
) SStandaloneResourceBaseManager {
	return SStandaloneResourceBaseManager{
		SStandaloneAnonResourceBaseManager: NewStandaloneAnonResourceBaseManager(dt, tableName, keyword, keywordPlural),
	}
}

func (manager *SStandaloneResourceBaseManager) HasName() bool {
	return true
}

func (manager *SStandaloneResourceBaseManager) FilterByName(q *sqlchemy.SQuery, name string) *sqlchemy.SQuery {
	return q.Equals("name", name)
}

func (manager *SStandaloneResourceBaseManager) ValidateName(name string) error {
	if manager.NameRequireAscii && !regutils.MatchName(name) {
		return httperrors.NewInputParameterError("name starts with letter, and contains letter, number and ._@- only")
	}
	if manager.NameLength > 0 && len(name) > manager.NameLength {
		return httperrors.NewInputParameterError("name longer than %d", manager.NameLength)
	}
	return nil
}

func (manager *SStandaloneResourceBaseManager) FetchByName(userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	return FetchByName(manager.GetIStandaloneModelManager(), userCred, idStr)
}

func (manager *SStandaloneResourceBaseManager) FetchByIdOrName(userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	return FetchByIdOrName(manager.GetIStandaloneModelManager(), userCred, idStr)
}

func (manager *SStandaloneResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input apis.StandaloneResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneAnonResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StandaloneAnonResourceListInput)
	if err != nil {
		return q, errors.Wrap(err, "SStandaloneAnonResourceBaseManager.ListItemFilte")
	}
	if len(input.Names) > 0 {
		q = q.In("name", input.Names)
	}

	return q, nil
}

func (manager *SStandaloneResourceBaseManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneAnonResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneAnonResourceBaseManager.ListItemExportKeys")
	}

	if keys.Contains("user_tags") {
		userTagsQuery := Metadata.Query().Startswith("id", manager.keyword+"::").
			Startswith("key", USER_TAG_PREFIX).GroupBy("id")
		userTagsQuery.AppendField(sqlchemy.SubStr("resource_id", userTagsQuery.Field("id"), len(manager.keyword)+3, 0))
		userTagsQuery.AppendField(
			sqlchemy.GROUP_CONCAT("user_tags", sqlchemy.CONCAT("",
				sqlchemy.SubStr("", userTagsQuery.Field("key"), len(USER_TAG_PREFIX)+1, 0),
				sqlchemy.NewStringField(":"),
				userTagsQuery.Field("value"),
			)))
		subQ := userTagsQuery.SubQuery()
		q.LeftJoin(subQ, sqlchemy.Equals(q.Field("id"), subQ.Field("resource_id")))
		q.AppendField(subQ.Field("user_tags"))
	}

	return q, nil
}

func (manager *SStandaloneResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input apis.StandaloneResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneAnonResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.StandaloneAnonResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (model *SStandaloneResourceBase) GetName() string {
	return model.Name
}

func (model *SStandaloneResourceBase) GetIStandaloneModel() IStandaloneModel {
	return model.GetVirtualObject().(IStandaloneModel)
}

func (model *SStandaloneResourceBase) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := model.SStandaloneAnonResourceBase.GetShortDesc(ctx)
	desc.Add(jsonutils.NewString(model.GetName()), "name")
	return desc
}

func (model *SStandaloneResourceBase) GetShortDescV2(ctx context.Context) *apis.StandaloneResourceShortDescDetail {
	desc := &apis.StandaloneResourceShortDescDetail{}
	desc.StandaloneAnonResourceShortDescDetail = *model.SStandaloneAnonResourceBase.GetShortDescV2(ctx)
	desc.Name = model.GetName()
	return desc
}

func (manager *SStandaloneResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input apis.StandaloneResourceCreateInput) (apis.StandaloneResourceCreateInput, error) {
	var err error
	input.StandaloneAnonResourceCreateInput, err = manager.SStandaloneAnonResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneAnonResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneAnonResourceBaseManager.ValidateCreateData")
	}
	input.Name = strings.TrimSpace(input.Name)
	if strings.ContainsAny(input.Name, "\n\r\t") {
		return input, errors.Wrap(httperrors.ErrInputParameter, "name should not contains any \\n\\r\\t")
	}
	return input, nil
}

func (manager *SStandaloneResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.StandaloneResourceDetails {
	ret := make([]apis.StandaloneResourceDetails, len(objs))
	upperRet := manager.SStandaloneAnonResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range objs {
		ret[i] = apis.StandaloneResourceDetails{
			StandaloneAnonResourceDetails: upperRet[i],
		}
	}
	return ret
}

func (model *SStandaloneResourceBase) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.StandaloneResourceBaseUpdateInput) (apis.StandaloneResourceBaseUpdateInput, error) {
	var err error
	input.StandaloneAnonResourceBaseUpdateInput, err = model.SStandaloneAnonResourceBase.ValidateUpdateData(ctx, userCred, query, input.StandaloneAnonResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SModelBase.ValidateUpdateData")
	}

	if len(input.Name) > 0 {
		input.Name = strings.TrimSpace(input.Name)
		if strings.ContainsAny(input.Name, "\n\r\t") {
			return input, errors.Wrap(httperrors.ErrInputParameter, "name should not contains any \\n\\r\\t")
		}
		err = alterNameValidator(model.GetIStandaloneModel(), input.Name)
		if err != nil {
			return input, errors.Wrap(err, "alterNameValidator")
		}
	}
	return input, nil
}
