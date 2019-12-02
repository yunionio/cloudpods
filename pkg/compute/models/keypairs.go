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

	"golang.org/x/crypto/ssh"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SKeypairManager struct {
	db.SStandaloneResourceBaseManager
}

var KeypairManager *SKeypairManager

func init() {
	KeypairManager = &SKeypairManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SKeypair{},
			"keypairs_tbl",
			"keypair",
			"keypairs",
		),
	}
	KeypairManager.SetVirtualObject(KeypairManager)
}

type SKeypair struct {
	db.SStandaloneResourceBase

	Scheme      string `width:"12" charset:"ascii" nullable:"true" list:"user" create:"required"`    // Column(VARCHAR(length=12, charset='ascii'), nullable=True, default='RSA')
	Fingerprint string `width:"48" charset:"ascii" nullable:"false" list:"user" create:"required"`   // Column(VARCHAR(length=48, charset='ascii'), nullable=False)
	PrivateKey  string `width:"2048" charset:"ascii" nullable:"true" create:"optional"`              // Column(VARCHAR(length=2048, charset='ascii'), nullable=False)
	PublicKey   string `width:"1024" charset:"ascii" nullable:"false" list:"user" create:"required"` // Column(VARCHAR(length=1024, charset='ascii'), nullable=False)
	OwnerId     string `width:"128" charset:"ascii" index:"true" nullable:"false" create:"required"` // Column(VARCHAR(length=36, charset='ascii'), index=True, nullable=False)
}

func (manager *SKeypairManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	if jsonutils.QueryBoolean(query, "admin", false) && db.IsAdminAllowList(userCred, manager) {
		user, _ := query.GetString("user")
		if len(user) > 0 {
			uc, _ := db.UserCacheManager.FetchUserByIdOrName(ctx, user)
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

func (manager *SKeypairManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SKeypair) IsOwner(userCred mcclient.TokenCredential) bool {
	return self.OwnerId == userCred.GetUserId()
}

func (self *SKeypair) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowGet(userCred, self)
}

func (self *SKeypair) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra.Add(jsonutils.NewInt(int64(len(self.PrivateKey))), "private_key_len")

	guestCnt, err := self.GetLinkedGuestsCount()
	if err == nil {
		extra.Add(jsonutils.NewInt(int64(guestCnt)), "linked_guest_count")
	}

	return extra
}

func (self *SKeypair) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	extra.Add(jsonutils.NewInt(int64(len(self.PrivateKey))), "private_key_len")

	guestCnt, err := self.GetLinkedGuestsCount()
	if err != nil {
		return nil, httperrors.NewInternalServerError("GetLinkedGuestsCount fail %s", err)
	}
	extra.Add(jsonutils.NewInt(int64(guestCnt)), "linked_guest_count")

	if db.IsAdminAllowGet(userCred, self) {
		extra.Add(jsonutils.NewString(self.OwnerId), "owner_id")
		uc, _ := db.UserCacheManager.FetchUserById(ctx, self.OwnerId)
		if uc != nil {
			extra.Add(jsonutils.NewString(uc.Name), "owner_name")
		}
	}
	return extra, nil
}

func (manager *SKeypairManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
}

func (self *SKeypair) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowUpdate(userCred, self)
}

func (self *SKeypair) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowDelete(userCred, self)
}

func (self *SKeypair) GetLinkedGuestsCount() (int, error) {
	return GuestManager.Query().Equals("keypair_id", self.Id).CountWithError()
}

func (manager *SKeypairManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	publicKey, _ := data.GetString("public_key")
	if len(publicKey) == 0 {
		scheme, _ := data.GetString("scheme")
		if len(scheme) > 0 {
			if !utils.IsInStringArray(scheme, []string{"RSA", "DSA"}) {
				return nil, httperrors.NewInputParameterError("Unsupported scheme %s", scheme)
			}
		} else {
			scheme = "RSA"
		}
		var privKey, pubKey string
		var err error
		if scheme == "RSA" {
			privKey, pubKey, err = seclib2.GenerateRSASSHKeypair()
		} else {
			privKey, pubKey, err = seclib2.GenerateDSASSHKeypair()
		}
		if err != nil {
			log.Errorf("fail to generate ssh keypair %s", err)
			return nil, httperrors.NewGeneralError(err)
		}
		publicKey = pubKey
		data.Set("public_key", jsonutils.NewString(pubKey))
		data.Set("private_key", jsonutils.NewString(privKey))
	}
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		log.Errorf("invalid public key %s", err)
		return nil, httperrors.NewInputParameterError("invalid public")
	}

	// 只允许上传RSA格式密钥。PS: AWS只支持RSA格式。
	scheme := seclib2.GetPublicKeyScheme(pubKey)
	if scheme != "RSA" {
		return nil, httperrors.NewInputParameterError("Unsupported scheme %s", scheme)
	}

	data.Set("fingerprint", jsonutils.NewString(ssh.FingerprintLegacyMD5(pubKey)))
	data.Set("scheme", jsonutils.NewString(scheme))
	data.Set("owner_id", jsonutils.NewString(userCred.GetUserId()))

	input := apis.StandaloneResourceCreateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal StandaloneRes  ourceCreateInput fail %s", err)
	}
	input, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))
	return data, nil
}

func (self *SKeypair) ValidateDeleteCondition(ctx context.Context) error {
	guestCnt, err := self.GetLinkedGuestsCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetLinkedGuestsCount failed %s", err)
	}
	if guestCnt > 0 {
		return httperrors.NewNotEmptyError("Cannot delete keypair used by servers")
	}
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func totalKeypairCount(userId string) (int, error) {
	q := KeypairManager.Query().Equals("owner_id", userId)
	return q.CountWithError()
}

func (manager *SKeypairManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		if scope == rbacutils.ScopeUser {
			if len(owner.GetUserId()) > 0 {
				q = q.Equals("owner_id", owner.GetUserId())
			}
		}
	}
	return q
}

func (self *SKeypair) GetOwnerId() mcclient.IIdentityProvider {
	owner := db.SOwnerId{UserId: self.OwnerId}
	return &owner
}

func (manager *SKeypairManager) FetchByName(userCred mcclient.IIdentityProvider, idStr string) (db.IModel, error) {
	return db.FetchByName(manager, userCred, idStr)
}

func (manager *SKeypairManager) FetchByIdOrName(userCred mcclient.IIdentityProvider, idStr string) (db.IModel, error) {
	return db.FetchByIdOrName(manager, userCred, idStr)
}

func (keypair *SKeypair) AllowGetDetailsPrivatekey(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return keypair.OwnerId == userCred.GetUserId()
}

func (keypair *SKeypair) GetDetailsPrivatekey(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	retval := jsonutils.NewDict()
	if len(keypair.PrivateKey) > 0 {
		retval.Add(jsonutils.NewString(keypair.PrivateKey), "private_key")
		retval.Add(jsonutils.NewString(keypair.Name), "name")
		retval.Add(jsonutils.NewString(keypair.Scheme), "scheme")
		_, err := db.Update(keypair, func() error {
			keypair.PrivateKey = ""
			return nil
		})
		if err != nil {
			return nil, err
		}

		db.OpsLog.LogEvent(keypair, db.ACT_FETCH, nil, userCred)
		logclient.AddActionLogWithContext(ctx, keypair, logclient.ACT_FETCH, nil, userCred, true)
	}
	return retval, nil
}

func (manager *SKeypairManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return db.FetchUserInfo(ctx, data)
}

func (manager *SKeypairManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeUser
}

func (manager *SKeypairManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeUser
}
