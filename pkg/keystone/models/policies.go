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

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	policyman "yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SPolicyManager struct {
	SEnabledIdentityBaseResourceManager
}

var PolicyManager *SPolicyManager

func init() {
	PolicyManager = &SPolicyManager{
		SEnabledIdentityBaseResourceManager: NewEnabledIdentityBaseResourceManager(
			SPolicy{},
			"policy",
			"policy",
			"policies",
		),
	}
	PolicyManager.SetVirtualObject(PolicyManager)
}

/*
+-------+--------------+------+-----+---------+-------+
| Field | Type         | Null | Key | Default | Extra |
+-------+--------------+------+-----+---------+-------+
| id    | varchar(64)  | NO   | PRI | NULL    |       |
| type  | varchar(255) | NO   |     | NULL    |       |
| blob  | text         | NO   |     | NULL    |       |
| extra | text         | YES  |     | NULL    |       |
+-------+--------------+------+-----+---------+-------+
*/

type SPolicy struct {
	SEnabledIdentityBaseResource

	Type string               `width:"255" charset:"utf8" nullable:"false" list:"user" update:"domain"`
	Blob jsonutils.JSONObject `nullable:"false" list:"user" update:"domain"`

	IsPublic bool `default:"false" nullable:"false" list:"user"`
}

func (manager *SPolicyManager) InitializeData() error {
	q := manager.Query()
	q = q.IsNullOrEmpty("name")
	policies := make([]SPolicy, 0)
	err := db.FetchModelObjects(manager, q, &policies)
	if err != nil {
		return err
	}
	for i := range policies {
		db.Update(&policies[i], func() error {
			policies[i].Name = policies[i].Type
			policies[i].Description, _ = policies[i].Extra.GetString("description")
			return nil
		})
	}
	return nil
}

func (manager *SPolicyManager) FetchEnabledPolicies() ([]SPolicy, error) {
	q := manager.Query().IsTrue("enabled")

	policies := make([]SPolicy, 0)
	err := db.FetchModelObjects(manager, q, &policies)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return policies, nil
}

func (manager *SPolicyManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	typeStr, _ := data.GetString("type")
	if len(typeStr) == 0 {
		return nil, httperrors.NewInputParameterError("missing input field type")
	}
	if !data.Contains("name") {
		data.Set("name", jsonutils.NewString(typeStr))
	}
	blobJson, err := data.Get("blob")
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid policy data")
	}
	policy := rbacutils.SRbacPolicy{}
	err = policy.Decode(blobJson)
	if err != nil {
		return nil, httperrors.NewInputParameterError("fail to decode policy data")
	}
	if policy.IsSystemWidePolicy() && policyman.PolicyManager.Allow(rbacutils.ScopeSystem, userCred, consts.GetServiceType(), manager.KeywordPlural(), policyman.PolicyActionCreate) == rbacutils.Deny {
		return nil, httperrors.NewNotSufficientPrivilegeError("not allow to create system-wide policy")
	}
	return manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (policy *SPolicy) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("blob") {
		blobJson, err := data.Get("blob")
		if err != nil {
			return nil, httperrors.NewInputParameterError("invalid policy data")
		}
		p := rbacutils.SRbacPolicy{}
		err = p.Decode(blobJson)
		if err != nil {
			return nil, httperrors.NewInputParameterError("fail to decode policy data")
		}
		if p.IsSystemWidePolicy() && policyman.PolicyManager.Allow(rbacutils.ScopeSystem, userCred, consts.GetServiceType(), policy.GetModelManager().KeywordPlural(), policyman.PolicyActionUpdate) == rbacutils.Deny {
			return nil, httperrors.NewNotSufficientPrivilegeError("not allow to update system-wide policy")
		}
	}
	if data.Contains("type") {
		typeStr, _ := data.GetString("type")
		if len(typeStr) == 0 {
			return nil, httperrors.NewInputParameterError("empty name")
		}
		if len(typeStr) > 0 {
			data.Set("name", jsonutils.NewString(typeStr))
		}
	}
	return policy.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (policy *SPolicy) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	policy.SStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	policyman.PolicyManager.SyncOnce()
}

func (policy *SPolicy) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	policy.SStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
	policyman.PolicyManager.SyncOnce()
}

func (policy *SPolicy) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	policy.SStandaloneResourceBase.PostDelete(ctx, userCred)
	policyman.PolicyManager.SyncOnce()
}

func (policy *SPolicy) AllowPerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAllowPerform(rbacutils.ScopeSystem, userCred, policy, "public")
}

func (policy *SPolicy) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if policy.IsPublic {
		return nil, nil
	}

	scope := policyman.PolicyManager.AllowScope(userCred, consts.GetServiceType(), policy.GetModelManager().KeywordPlural(), policyman.PolicyActionPerform, "public")
	if scope != rbacutils.ScopeSystem {
		return nil, httperrors.NewForbiddenError("not enough privilege")
	}

	p := rbacutils.SRbacPolicy{}
	err := p.Decode(policy.Blob)
	if err != nil {
		return nil, httperrors.NewInputParameterError("fail to decode policy data")
	}
	if !p.IsSystemWidePolicy() {
		return nil, httperrors.NewInvalidStatusError("only system-wide policy (no roles and projects constraints) is sharable")
	}

	diff, err := db.Update(policy, func() error {
		policy.IsPublic = true
		return nil
	})
	if err == nil {
		db.OpsLog.LogEvent(policy, db.ACT_UPDATE, diff, userCred)
	}
	return nil, err
}

func (policy *SPolicy) AllowPerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAllowPerform(rbacutils.ScopeSystem, userCred, policy, "private")
}

func (policy *SPolicy) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !policy.IsPublic {
		return nil, nil
	}

	scope := policyman.PolicyManager.AllowScope(userCred, consts.GetServiceType(), policy.GetModelManager().KeywordPlural(), policyman.PolicyActionPerform, "private")
	if scope != rbacutils.ScopeSystem {
		return nil, httperrors.NewForbiddenError("not enough privilege")
	}

	diff, err := db.Update(policy, func() error {
		policy.IsPublic = false
		return nil
	})
	if err == nil {
		db.OpsLog.LogEvent(policy, db.ACT_UPDATE, diff, userCred)
	}
	return nil, err
}

func (manager *SPolicyManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
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
