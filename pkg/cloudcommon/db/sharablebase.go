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

type SSharableBaseResourceManager struct{}

func (manager *SSharableBaseResourceManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacutils.ScopeDomain:
			if len(owner.GetProjectDomainId()) > 0 {
				q = q.Filter(sqlchemy.OR(
					sqlchemy.Equals(q.Field("domain_id"), owner.GetProjectDomainId()),
					sqlchemy.IsTrue(q.Field("is_public")),
				))
			}
		}
	}
	return q
}

type SSharableBaseResource struct {
	IsPublic bool `default:"false" nullable:"false" list:"user"`
}

type SSharableBaseInterface interface {
	IModel
	SetIsPublic(pub bool)
	GetIsPublic() bool
}

func (m *SSharableBaseResource) IsSharable(ownerId mcclient.IIdentityProvider) bool {
	return m.IsPublic
}

func (m *SSharableBaseResource) SetIsPublic(pub bool) {
	m.IsPublic = pub
}

func (m SSharableBaseResource) GetIsPublic() bool {
	return m.IsPublic
}

func SharableAllowPerformPublic(model SSharableBaseInterface, userCred mcclient.TokenCredential) bool {
	return IsAllowPerform(rbacutils.ScopeSystem, userCred, model, "public")
}

func SharablePerformPublic(model SSharableBaseInterface, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if model.GetIsPublic() {
		return nil, nil
	}

	scope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), model.GetModelManager().KeywordPlural(), policy.PolicyActionPerform, "public")
	if scope != rbacutils.ScopeSystem {
		return nil, httperrors.NewForbiddenError("not enough privilege")
	}

	ownerId := model.GetOwnerId()
	if userCred.GetProjectDomainId() != ownerId.GetProjectDomainId() {
		return nil, httperrors.NewForbiddenError("not owner")
	}

	diff, err := Update(model, func() error {
		model.SetIsPublic(true)
		return nil
	})
	if err == nil {
		OpsLog.LogEvent(model, ACT_UPDATE, diff, userCred)
	}
	return nil, err
}

func SharableAllowPerformPrivate(model SSharableBaseInterface, userCred mcclient.TokenCredential) bool {
	return IsAllowPerform(rbacutils.ScopeSystem, userCred, model, "private")
}

func SharablePerformPrivate(model SSharableBaseInterface, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !model.GetIsPublic() {
		return nil, nil
	}

	scope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), model.GetModelManager().KeywordPlural(), policy.PolicyActionPerform, "private")
	if scope != rbacutils.ScopeSystem {
		return nil, httperrors.NewForbiddenError("not enough privilege")
	}

	ownerId := model.GetOwnerId()
	if userCred.GetProjectDomainId() != ownerId.GetProjectDomainId() {
		return nil, httperrors.NewForbiddenError("not owner")
	}

	diff, err := Update(model, func() error {
		model.SetIsPublic(false)
		return nil
	})
	if err == nil {
		OpsLog.LogEvent(model, ACT_UPDATE, diff, userCred)
	}
	return nil, err
}
