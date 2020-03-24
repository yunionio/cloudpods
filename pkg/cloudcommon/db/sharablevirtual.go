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
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SSharableVirtualResourceBase struct {
	SVirtualResourceBase
	SSharableBaseResource
	// IsPublic    bool   `default:"false" nullable:"false" create:"domain_optional" list:"user" json:"is_public"`
	// PublicScope string `width:"16" charset:"ascii" nullable:"false" default:"system" create:"domain_optional" list:"user" json:"public_scope"`
}

type SSharableVirtualResourceBaseManager struct {
	SVirtualResourceBaseManager
	SSharableBaseResourceManager
}

func NewSharableVirtualResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SSharableVirtualResourceBaseManager {
	return SSharableVirtualResourceBaseManager{SVirtualResourceBaseManager: NewVirtualResourceBaseManager(dt, tableName, keyword, keywordPlural)}
}

func (manager *SSharableVirtualResourceBaseManager) GetISharableVirtualModelManager() ISharableVirtualModelManager {
	return manager.GetVirtualObject().(ISharableVirtualModelManager)
}

func (manager *SSharableVirtualResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	return SharableManagerFilterByOwner(manager.GetISharableVirtualModelManager(), q, owner, scope)
}

func (model *SSharableVirtualResourceBase) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return model.IsOwner(userCred) || model.IsSharable(userCred) || IsAllowGet(rbacutils.ScopeSystem, userCred, model)
}

func (model *SSharableVirtualResourceBase) IsSharable(reqUsrId mcclient.IIdentityProvider) bool {
	return SharableModelIsSharable(model.GetISharableVirtualModel(), reqUsrId)
}

func (model *SSharableVirtualResourceBase) IsShared() bool {
	return SharableModelIsShared(model)
}

func (model *SSharableVirtualResourceBase) AllowPerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPublicInput) bool {
	return true
}

func (model *SSharableVirtualResourceBase) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPublicInput) (jsonutils.JSONObject, error) {
	err := SharablePerformPublic(model, ctx, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "SharablePerformPublic")
	}
	return nil, nil
}

func (model *SSharableVirtualResourceBase) AllowPerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPrivateInput) bool {
	return true
}

func (model *SSharableVirtualResourceBase) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPrivateInput) (jsonutils.JSONObject, error) {
	err := SharablePerformPrivate(model, ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "SharablePerformPrivate")
	}
	return nil, nil
}

func (model *SSharableVirtualResourceBase) GetISharableVirtualModel() ISharableVirtualModel {
	return model.GetVirtualObject().(ISharableVirtualModel)
}

func (manager *SSharableVirtualResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input apis.SharableVirtualResourceCreateInput) (apis.SharableVirtualResourceCreateInput, error) {
	var err error
	input.VirtualResourceCreateInput, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "manager.VirtualResourceBaseManager.ValidateCreateData")
	}
	return input, nil
}

func (manager *SSharableVirtualResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.SharableVirtualResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SSharableBaseResourceManager.ListItemFilter(ctx, q, userCred, query.SharableResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableBaseResourceManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SSharableVirtualResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SSharableVirtualResourceBaseManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query apis.SharableVirtualResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (model *SSharableVirtualResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (apis.SharableVirtualResourceDetails, error) {
	return apis.SharableVirtualResourceDetails{}, nil
}

func (manager *SSharableVirtualResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.SharableVirtualResourceDetails {
	rows := make([]apis.SharableVirtualResourceDetails, len(objs))

	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	shareRows := manager.SSharableBaseResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = apis.SharableVirtualResourceDetails{
			VirtualResourceDetails:   virtRows[i],
			SharableResourceBaseInfo: shareRows[i],
		}
	}

	return rows
}

func (model *SSharableVirtualResourceBase) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.SharableVirtualResourceBaseUpdateInput,
) (apis.SharableVirtualResourceBaseUpdateInput, error) {
	var err error
	input.VirtualResourceBaseUpdateInput, err = model.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.VirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (model *SSharableVirtualResourceBase) GetSharedProjects() []string {
	return SharableGetSharedProjects(model, SharedTargetProject)
}

func (model *SSharableVirtualResourceBase) GetSharedDomains() []string {
	return SharableGetSharedProjects(model, SharedTargetDomain)
}

func (model *SSharableVirtualResourceBase) PerformChangeOwner(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.PerformChangeProjectOwnerInput,
) (jsonutils.JSONObject, error) {
	if model.IsShared() {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "cannot change owner when shared!")
	}
	return model.SVirtualResourceBase.PerformChangeOwner(ctx, userCred, query, input)
}
