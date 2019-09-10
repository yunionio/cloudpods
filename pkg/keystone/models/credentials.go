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
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/keys"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SCredentialManager struct {
	db.SStandaloneResourceBaseManager
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
		return nil, httperrors.NewInputParameterError("missing input feild type")
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

	return manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (self *SCredential) ValidateDeleteCondition(ctx context.Context) error {
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SCredential) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {

	return self.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (self *SCredential) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return credentialExtra(self, extra)
}

func (self *SCredential) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return credentialExtra(self, extra), nil
}

func credentialExtra(cred *SCredential, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	extra.Add(jsonutils.NewString(string(cred.getBlob())), "blob")

	usr, _ := UserManager.FetchUserExtended(cred.UserId, "", "", "")
	if usr != nil {
		extra.Add(jsonutils.NewString(usr.Name), "user")
		extra.Add(jsonutils.NewString(usr.DomainId), "domain_id")
		extra.Add(jsonutils.NewString(usr.DomainName), "domain")
	}
	return extra
}

func (self *SCredential) getBlob() []byte {
	return keys.CredentialKeyManager.Decrypt([]byte(self.EncryptedBlob), time.Duration(-1))
}

func (self *SCredential) GetAccessKeySecret() (*api.SAccessKeySecretBlob, error) {
	if self.Type == api.ACCESS_SECRET_TYPE {
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
		domainOwner, err := fetchDomainInfo(data)
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
