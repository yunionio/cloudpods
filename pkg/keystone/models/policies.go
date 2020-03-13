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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	policyman "yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SPolicyManager struct {
	SEnabledIdentityBaseResourceManager
	db.SSharableBaseResourceManager
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
	db.SSharableBaseResource

	Type string               `width:"255" charset:"utf8" nullable:"false" list:"user" update:"domain"`
	Blob jsonutils.JSONObject `nullable:"false" list:"user" update:"domain"`
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
	data.Set("name", jsonutils.NewString(typeStr))
	blobJson, err := data.Get("blob")
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid policy data")
	}
	policy := rbacutils.SRbacPolicy{}
	err = policy.Decode(blobJson)
	if err != nil {
		return nil, httperrors.NewInputParameterError("fail to decode policy data")
	}
	err = db.ValidateCreateDomainId(ownerId.GetProjectDomainId())
	if err != nil {
		return nil, err
	}
	input := api.EnabledIdentityBaseResourceCreateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal IdentityBaseResourceCreateInput fail %s", err)
	}
	input, err = manager.SEnabledIdentityBaseResourceManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))
	return data, nil
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
		/* if p.IsSystemWidePolicy() && policyman.PolicyManager.Allow(rbacutils.ScopeSystem, userCred, consts.GetServiceType(), policy.GetModelManager().KeywordPlural(), policyman.PolicyActionUpdate) == rbacutils.Deny {
			return nil, httperrors.NewNotSufficientPrivilegeError("not allow to update system-wide policy")
		} */
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
	return policy.SEnabledIdentityBaseResource.ValidateUpdateData(ctx, userCred, query, data)
}

func (policy *SPolicy) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	policy.SEnabledIdentityBaseResource.PostCreate(ctx, userCred, ownerId, query, data)
	policyman.PolicyManager.SyncOnce()
}

func (policy *SPolicy) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	policy.SEnabledIdentityBaseResource.PostUpdate(ctx, userCred, query, data)
	policyman.PolicyManager.SyncOnce()
}

func (policy *SPolicy) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	policy.SEnabledIdentityBaseResource.PostDelete(ctx, userCred)
	policyman.PolicyManager.SyncOnce()
}

func (policy *SPolicy) AllowPerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.SharableAllowPerformPublic(policy, userCred)
}

func (policy *SPolicy) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	res, err := db.SharablePerformPublic(policy, ctx, userCred, query, data)
	if err == nil {
		policyman.PolicyManager.SyncOnce()
	}
	return res, err
}

func (policy *SPolicy) AllowPerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.SharableAllowPerformPrivate(policy, userCred)
}

func (policy *SPolicy) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	res, err := db.SharablePerformPrivate(policy, ctx, userCred, query, data)
	if err == nil {
		policyman.PolicyManager.SyncOnce()
	}
	return res, err
}

func (policy *SPolicy) ValidateDeleteCondition(ctx context.Context) error {
	if policy.IsPublic {
		return httperrors.NewInvalidStatusError("cannot delete shared policy")
	}
	if policy.Enabled.IsTrue() {
		return httperrors.NewInvalidStatusError("cannot delete enabled policy")
	}
	return policy.SEnabledIdentityBaseResource.ValidateDeleteCondition(ctx)
}

// 权限策略列表
func (manager *SPolicyManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.PolicyListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledIdentityBaseResourceManager.ListItemFilter(ctx, q, userCred, query.EnabledIdentityBaseResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledIdentityBaseResourceManager.ListItemFilter")
	}
	q, err = manager.SSharableBaseResourceManager.ListItemFilter(ctx, q, userCred, query.SharableResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableBaseResourceManager.ListItemFilter")
	}
	if len(query.Type) > 0 {
		q = q.In("type", query.Type)
	}
	return q, nil
}

func (manager *SPolicyManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.PolicyListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledIdentityBaseResourceManager.OrderByExtraFields(ctx, q, userCred, query.EnabledIdentityBaseResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledIdentityBaseResourceManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SPolicyManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledIdentityBaseResourceManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (policy *SPolicy) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.PolicyDetails, error) {
	return api.PolicyDetails{}, nil
}

func (manager *SPolicyManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.PolicyDetails {
	rows := make([]api.PolicyDetails, len(objs))
	identRows := manager.SEnabledIdentityBaseResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.PolicyDetails{
			EnabledIdentityBaseResourceDetails: identRows[i],
		}
	}
	return rows
}
