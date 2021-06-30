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
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/keys"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCredentialManager struct {
	db.SStandaloneResourceBaseManager
	SUserResourceBaseManager
	SProjectResourceBaseManager
}

var CredentialManager *SCredentialManager

func init() {
	CredentialManager = &SCredentialManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SCredential{},
			"credential",
			"credential",
			"credentials",
		),
	}
	CredentialManager.SetVirtualObject(CredentialManager)
}

/*
+----------------+--------------+------+-----+---------+-------+
| Field          | Type         | Null | Key | Default | Extra |
+----------------+--------------+------+-----+---------+-------+
| id             | varchar(64)  | NO   | PRI | NULL    |       |
| user_id        | varchar(64)  | NO   |     | NULL    |       |
| project_id     | varchar(64)  | YES  |     | NULL    |       |
| type           | varchar(255) | NO   |     | NULL    |       |
| extra          | text         | YES  |     | NULL    |       |
| key_hash       | varchar(64)  | NO   |     | NULL    |       |
| encrypted_blob | text         | NO   |     | NULL    |       |
+----------------+--------------+------+-----+---------+-------+
*/

type SCredential struct {
	db.SStandaloneResourceBase

	UserId    string `width:"64" charset:"ascii" nullable:"false" list:"user" create:"required"`
	ProjectId string `width:"64" charset:"ascii" nullable:"true" list:"user" create:"required"`
	Type      string `width:"255" charset:"utf8" nullable:"false" list:"user" create:"required"`
	KeyHash   string `width:"64" charset:"ascii" nullable:"false" create:"required"`

	Extra *jsonutils.JSONDict `nullable:"true" list:"admin"`

	EncryptedBlob string `nullable:"false" create:"required"`

	Enabled tristate.TriState `nullable:"false" default:"true" list:"user" update:"user" create:"optional"`
}

func (manager *SCredentialManager) InitializeData() error {
	q := manager.Query()
	q = q.IsNullOrEmpty("name")
	creds := make([]SCredential, 0)
	err := db.FetchModelObjects(manager, q, &creds)
	if err != nil {
		return err
	}
	for i := range creds {
		if gotypes.IsNil(creds[i].Extra) {
			continue
		}
		name, _ := creds[i].Extra.GetString("name")
		desc, _ := creds[i].Extra.GetString("description")
		if len(name) == 0 {
			name = creds[i].Type
		}
		db.Update(&creds[i], func() error {
			creds[i].Name = name
			creds[i].Description = desc
			return nil
		})
	}
	return nil
}

func (manager *SCredentialManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if !data.Contains("type") {
		return nil, httperrors.NewInputParameterError("missing input field type")
	}
	projectId, _ := data.GetString("project_id")
	userId := ownerId.GetUserId()
	if len(userId) == 0 {
		userId = userCred.GetUserId()
	}
	data.Set("user_id", jsonutils.NewString(userId))
	if len(projectId) == 0 {
		projectId = userCred.GetProjectId()
		data.Set("project_id", jsonutils.NewString(projectId))
	} else if projectId == api.DEFAULT_PROJECT {
		// do nothing
	} else {
		_, err := ProjectManager.FetchById(projectId)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(ProjectManager.Keyword(), projectId)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
	}
	if !data.Contains("name") {
		typeStr, _ := data.GetString("type")
		data.Add(jsonutils.NewString(fmt.Sprintf("%s-%s-%s", typeStr, projectId, userId)), "name")
	}
	blob, _ := data.GetString("blob")
	if len(blob) == 0 {
		return nil, httperrors.NewInputParameterError("missing input field blob")
	}
	blobEnc, err := keys.CredentialKeyManager.Encrypt([]byte(blob))
	if err != nil {
		return nil, httperrors.NewInternalServerError("encrypt error %s", err)
	}
	data.Add(jsonutils.NewString(string(blobEnc)), "encrypted_blob")
	data.Add(jsonutils.NewString(keys.CredentialKeyManager.PrimaryKeyHash()), "key_hash")

	input := apis.StandaloneResourceCreateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal StandaloneResourceCreateInput fail %s", err)
	}
	input, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))
	return data, nil
}

func (self *SCredential) ValidateDeleteCondition(ctx context.Context) error {
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SCredential) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CredentialUpdateInput) (api.CredentialUpdateInput, error) {
	var err error

	input.StandaloneResourceBaseUpdateInput, err = self.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneResourceBase.ValidateUpdateData")
	}

	return input, nil
}

func (manager *SCredentialManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CredentialDetails {
	rows := make([]api.CredentialDetails, len(objs))

	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.CredentialDetails{
			StandaloneResourceDetails: stdRows[i],
		}
		rows[i] = credentialExtra(objs[i].(*SCredential), rows[i])
	}

	return rows
}

func credentialExtra(cred *SCredential, out api.CredentialDetails) api.CredentialDetails {
	out.Blob = string(cred.getBlob())

	usr, _ := UserManager.FetchUserExtended(cred.UserId, "", "", "")
	if usr != nil {
		out.User = usr.Name
		out.Domain = usr.DomainName
		out.DomainId = usr.DomainId
	}
	return out
}

func (self *SCredential) getBlob() []byte {
	return keys.CredentialKeyManager.Decrypt([]byte(self.EncryptedBlob), time.Duration(-1))
}

func (self *SCredential) GetAccessKeySecret() (*api.SAccessKeySecretBlob, error) {
	if self.Type == api.ACCESS_SECRET_TYPE || self.Type == api.OIDC_CREDENTIAL_TYPE {
		blobJson, err := jsonutils.Parse(self.getBlob())
		if err != nil {
			return nil, errors.Wrap(err, "jsonutils.Parse")
		}
		akBlob := api.SAccessKeySecretBlob{}
		err = blobJson.Unmarshal(&akBlob)
		if err != nil {
			return nil, errors.Wrap(err, "blobJson.Unmarshal")
		}
		return &akBlob, nil
	}
	return nil, errors.Error("no an AK/SK credential")
}

func (manager *SCredentialManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeUser
}

func (manager *SCredentialManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		if scope == rbacutils.ScopeUser {
			if len(owner.GetUserId()) > 0 {
				q = q.Equals("user_id", owner.GetUserId())
			}
		}
	}
	return q
}

func (self *SCredential) GetOwnerId() mcclient.IIdentityProvider {
	owner := db.SOwnerId{UserId: self.UserId}
	return &owner
}

func (manager *SCredentialManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	userStr, key := jsonutils.GetAnyString2(data, []string{"user", "user_id"})
	if len(userStr) > 0 {
		domainOwner, err := db.FetchDomainInfo(ctx, data)
		if err != nil {
			return nil, err
		}
		if domainOwner == nil {
			domainOwner = &db.SOwnerId{DomainId: api.DEFAULT_DOMAIN_ID}
		}
		data.(*jsonutils.JSONDict).Remove(key)
		usrObj, err := UserManager.FetchByIdOrName(domainOwner, userStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("user", userStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		usr := usrObj.(*SUser)
		ownerId := db.SOwnerId{
			UserDomainId: usr.DomainId,
			UserId:       usr.Id,
		}
		data.(*jsonutils.JSONDict).Set("user", jsonutils.NewString(usr.Id))
		return &ownerId, nil
	}
	return nil, nil
}

// 用户信用凭证列表
func (manager *SCredentialManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CredentialListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SUserResourceBaseManager.ListItemFilter(ctx, q, userCred, query.UserFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SUserResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SProjectResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ProjectFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SProjectResourceBaseManager.ListItemFilter")
	}
	if query.Enabled != nil {
		if *query.Enabled {
			q = q.IsTrue("enabled")
		} else {
			q = q.IsFalse("enabled")
		}
	}
	if len(query.Type) > 0 {
		q = q.In("type", query.Type)
	}
	return q, nil
}

func (manager *SCredentialManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CredentialListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SCredentialManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}
