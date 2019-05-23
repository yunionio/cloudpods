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
	"yunion.io/x/jsonutils"

	"context"
	"time"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/keys"
	"yunion.io/x/onecloud/pkg/mcclient"
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

	UserId    string `width:"64" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"`
	ProjectId string `width:"64" charset:"ascii" nullable:"true" list:"admin" create:"admin_required"`
	Type      string `width:"255" charset:"utf8" nullable:"false" list:"admin" create:"admin_required"`
	KeyHash   string `width:"64" charset:"ascii" nullable:"false" create:"admin_required"`

	Extra *jsonutils.JSONDict `nullable:"true" list:"admin"`

	EncryptedBlob string `nullable:"false" create:"admin_required"`
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
	if !data.Contains("name") {
		typeStr, _ := data.Get("type")
		data.Add(typeStr, "name")
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
	blob := keys.CredentialKeyManager.Decrypt([]byte(cred.EncryptedBlob), time.Duration(-1))
	extra.Add(jsonutils.NewString(string(blob)), "blob")

	usr, _ := UserManager.FetchUserExtended(cred.UserId, "", "", "")
	if usr != nil {
		extra.Add(jsonutils.NewString(usr.Name), "user")
		extra.Add(jsonutils.NewString(usr.DomainId), "domain_id")
		extra.Add(jsonutils.NewString(usr.DomainName), "domain")
	}
	return extra
}
