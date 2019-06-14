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
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SSharableVirtualResourceBase struct {
	SVirtualResourceBase

	IsPublic    bool   `default:"false" nullable:"false" create:"domain_optional" list:"user"`
	PublicScope string `width:"16" charset:"ascii" nullable:"false" default:"system" create:"domain_optional" list:"user"`
}

type SSharableVirtualResourceBaseManager struct {
	SVirtualResourceBaseManager
}

func NewSharableVirtualResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SSharableVirtualResourceBaseManager {
	return SSharableVirtualResourceBaseManager{SVirtualResourceBaseManager: NewVirtualResourceBaseManager(dt, tableName, keyword, keywordPlural)}
}

func (manager *SSharableVirtualResourceBaseManager) GetISharableVirtualModelManager() ISharableVirtualModelManager {
	return manager.GetVirtualObject().(ISharableVirtualModelManager)
}

func (manager *SSharableVirtualResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacutils.ScopeProject:
			if len(owner.GetProjectId()) > 0 {
				q = q.Filter(sqlchemy.OR(
					sqlchemy.Equals(q.Field("tenant_id"), owner.GetProjectId()),
					sqlchemy.AND(
						sqlchemy.IsTrue(q.Field("is_public")),
						sqlchemy.Equals(q.Field("public_scope"), rbacutils.ScopeSystem),
					),
					sqlchemy.AND(
						sqlchemy.IsTrue(q.Field("is_public")),
						sqlchemy.Equals(q.Field("public_scope"), rbacutils.ScopeDomain),
						sqlchemy.Equals(q.Field("domain_id"), owner.GetProjectDomainId()),
					),
				))
			}
		case rbacutils.ScopeDomain:
			if len(owner.GetProjectDomainId()) > 0 {
				q = q.Filter(sqlchemy.OR(
					sqlchemy.Equals(q.Field("domain_id"), owner.GetProjectDomainId()),
					sqlchemy.AND(
						sqlchemy.IsTrue(q.Field("is_public")),
						sqlchemy.Equals(q.Field("public_scope"), rbacutils.ScopeSystem),
					),
				))
			}
		}
	}
	return q
}

func (model *SSharableVirtualResourceBase) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return model.IsOwner(userCred) || model.IsPublic || IsAllowGet(rbacutils.ScopeSystem, userCred, model)
}

func (model *SSharableVirtualResourceBase) IsSharable(reqUsrId mcclient.IIdentityProvider) bool {
	if model.IsPublic {
		switch rbacutils.String2Scope(model.PublicScope) {
		case rbacutils.ScopeSystem:
			return true
		case rbacutils.ScopeDomain:
			ownerId := model.GetOwnerId()
			if ownerId != nil && ownerId.GetProjectDomainId() == reqUsrId.GetProjectDomainId() {
				return true
			}
		}
	}
	return false
}

func (model *SSharableVirtualResourceBase) AllowPerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAllowPerform(rbacutils.ScopeSystem, userCred, model, "public")
}

func (model *SSharableVirtualResourceBase) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !model.IsPublic {
		targetScopeStr, _ := data.GetString("scope")
		targetScope := rbacutils.String2ScopeDefault(targetScopeStr, rbacutils.ScopeSystem)
		allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), model.GetModelManager().KeywordPlural(), policy.PolicyActionPerform, "public")
		if targetScope.HigherThan(allowScope) {
			return nil, httperrors.NewForbiddenError("not enough privilege")
		}
		if targetScope != rbacutils.ScopeSystem && targetScope != rbacutils.ScopeDomain {
			return nil, httperrors.NewInputParameterError("invalid scope %s", targetScope)
		}
		diff, err := Update(model, func() error {
			model.IsPublic = true
			model.PublicScope = string(targetScope)
			return nil
		})
		if err == nil {
			OpsLog.LogEvent(model, ACT_UPDATE, diff, userCred)
		}
		return nil, err
	}
	return nil, nil
}

func (model *SSharableVirtualResourceBase) AllowPerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAllowPerform(rbacutils.ScopeSystem, userCred, model, "private")
}

func (model *SSharableVirtualResourceBase) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if model.IsPublic {
		allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), model.GetModelManager().KeywordPlural(), policy.PolicyActionPerform, "private")
		requireScope := rbacutils.String2ScopeDefault(model.PublicScope, rbacutils.ScopeSystem)
		if requireScope.HigherThan(allowScope) {
			return nil, httperrors.NewForbiddenError("not enough privileges: allow %s require %s", allowScope, requireScope)
		}
		diff, err := Update(model, func() error {
			model.IsPublic = false
			return nil
		})
		if err == nil {
			OpsLog.LogEvent(model, ACT_UPDATE, diff, userCred)
		}
		return nil, err
	}
	return nil, nil
}

func (model *SSharableVirtualResourceBase) GetISharableVirtualModel() ISharableVirtualModel {
	return model.GetVirtualObject().(ISharableVirtualModel)
}
