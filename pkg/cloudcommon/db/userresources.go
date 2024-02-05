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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SUserResourceBaseManager struct {
	SStandaloneResourceBaseManager
}

func NewUserResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SUserResourceBaseManager {
	return SUserResourceBaseManager{
		SStandaloneResourceBaseManager: NewStandaloneResourceBaseManager(dt,
			tableName, keyword, keywordPlural),
	}
}

type SUserResourceBase struct {
	SStandaloneResourceBase

	// 本地用户Id
	OwnerId string `width:"128" charset:"ascii" index:"true" list:"user" nullable:"false" create:"required"`
}

func (manager *SUserResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.UserResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	if ((query.Admin != nil && *query.Admin) || query.Scope == string(rbacscope.ScopeSystem)) && IsAdminAllowList(userCred, manager).Result.IsAllow() {
		user := query.UserId
		if len(user) > 0 {
			uc, _ := UserCacheManager.FetchUserByIdOrName(ctx, user)
			if uc == nil {
				return nil, httperrors.NewUserNotFoundError("user %s not found", user)
			}
			q = q.Equals("owner_id", uc.Id)
		}
	} else {
		q = q.Equals("owner_id", userCred.GetUserId())
	}

	return q, nil
}

func (manager *SUserResourceBaseManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query apis.UserResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (manager *SUserResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SUserResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.UserResourceDetails {
	rows := make([]apis.UserResourceDetails, len(objs))
	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	userIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = apis.UserResourceDetails{
			StandaloneResourceDetails: stdRows[i],
		}
		var base *SUserResourceBase
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil && len(base.OwnerId) > 0 {
			userIds[i] = base.OwnerId
		}
	}

	userMaps, err := FetchIdNameMap2(UserCacheManager, userIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail: %v", err)
		return rows
	}

	for i := range rows {
		rows[i].OwnerName, _ = userMaps[userIds[i]]
	}

	return rows
}

func (manager *SUserResourceBaseManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		if scope == rbacscope.ScopeUser {
			if len(owner.GetUserId()) > 0 {
				q = q.Equals("owner_id", owner.GetUserId())
			}
		}
	}
	return q
}

func (self *SUserResourceBase) GetOwnerId() mcclient.IIdentityProvider {
	owner := SOwnerId{UserId: self.OwnerId}
	return &owner
}

func (model *SUserResourceBase) IsOwner(userCred mcclient.TokenCredential) bool {
	return userCred.GetProjectId() == model.OwnerId
}

func (manager *SUserResourceBaseManager) GetIUserModelManager() IUserModelManager {
	return manager.GetVirtualObject().(IUserModelManager)
}

func (manager *SUserResourceBaseManager) FetchByName(ctx context.Context, userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	return FetchByName(ctx, manager, userCred, idStr)
}

func (manager *SUserResourceBaseManager) FetchByIdOrName(ctx context.Context, userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	return FetchByIdOrName(ctx, manager, userCred, idStr)
}

func (manager *SUserResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input apis.UserResourceCreateInput) (apis.UserResourceCreateInput, error) {
	if len(input.OwnerId) == 0 {
		input.OwnerId = userCred.GetUserId()
	}
	var err error
	input.StandaloneResourceCreateInput, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneResourceBaseManager.ValidateCreateData")
	}
	return input, nil
}

func (manager *SUserResourceBaseManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return FetchUserInfo(ctx, data)
}

func (manager *SUserResourceBaseManager) NamespaceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeUser
}

func (manager *SUserResourceBaseManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeUser
}
