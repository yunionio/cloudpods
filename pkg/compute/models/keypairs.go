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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
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

	// 加密类型
	// example: RSA
	Scheme string `width:"12" charset:"ascii" nullable:"true" list:"user" create:"required"`
	// 指纹信息
	// example: 1d:3a:83:4a:a1:f3:75:97:ec:d1:ef:f8:3f:a7:5d:9e
	Fingerprint string `width:"48" charset:"ascii" nullable:"false" list:"user" create:"required"`
	// 私钥
	PrivateKey string `width:"2048" charset:"ascii" nullable:"true" create:"optional"`
	// 公钥
	PublicKey string `width:"1024" charset:"ascii" nullable:"false" list:"user" create:"required"`
	// 用户Id
	OwnerId string `width:"128" charset:"ascii" index:"true" nullable:"false" create:"required"`
}

// 列出ssh密钥对
func (manager *SKeypairManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.KeypairListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	if query.Admin != nil && *query.Admin && db.IsAdminAllowList(userCred, manager) {
		user := query.User
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

	if len(query.Scheme) > 0 {
		q = q.In("scheme", query.Scheme)
	}
	if len(query.Fingerprint) > 0 {
		q = q.In("fingerprint", query.Fingerprint)
	}

	return q, nil
}

func (manager *SKeypairManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.KeypairListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SKeypairManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
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

func (self *SKeypair) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.KeypairDetails, error) {
	return api.KeypairDetails{}, nil
}

func (manager *SKeypairManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.KeypairDetails {
	rows := make([]api.KeypairDetails, len(objs))
	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	userIds := make([]string, len(objs))
	for i := range rows {
		keypair := objs[i].(*SKeypair)
		rows[i] = api.KeypairDetails{
			StandaloneResourceDetails: stdRows[i],
			PrivateKeyLen:             len(keypair.PrivateKey),
		}
		rows[i].LinkedGuestCount, _ = keypair.GetLinkedGuestsCount()
		userIds[i] = keypair.OwnerId
	}

	users := make(map[string]db.SUser)
	err := db.FetchStandaloneObjectsByIds(db.UserCacheManager, userIds, &users)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds for users fail %s", err)
		return rows
	}

	for i := range rows {
		if owner, ok := users[userIds[i]]; ok {
			rows[i].OwnerName = owner.Name
		}
	}

	return rows
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

func (manager *SKeypairManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.KeypairCreateInput) (*jsonutils.JSONDict, error) {
	if len(input.PublicKey) == 0 {
		if len(input.Scheme) == 0 {
			input.Scheme = api.KEYPAIRE_SCHEME_RSA
		}
		if !utils.IsInStringArray(input.Scheme, api.KEYPAIR_SCHEMAS) {
			return nil, httperrors.NewInputParameterError("Unsupported scheme %s", input.Scheme)
		}

		var err error
		if input.Scheme == api.KEYPAIRE_SCHEME_RSA {
			input.PrivateKey, input.PublicKey, err = seclib2.GenerateRSASSHKeypair()
		} else {
			input.PrivateKey, input.PublicKey, err = seclib2.GenerateDSASSHKeypair()
		}
		if err != nil {
			return nil, httperrors.NewGeneralError(errors.Wrapf(err, "Generate%sSSHKeypair", input.Scheme))
		}
	}
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(input.PublicKey))
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid public error: %v", err)
	}

	// 只允许上传RSA格式密钥。PS: AWS只支持RSA格式。
	input.Scheme = seclib2.GetPublicKeyScheme(pubKey)
	if input.Scheme != api.KEYPAIRE_SCHEME_RSA {
		return nil, httperrors.NewInputParameterError("Unsupported scheme %s", input.Scheme)
	}

	input.Fingerprint = ssh.FingerprintLegacyMD5(pubKey)
	input.OwnerId = userCred.GetUserId()

	input.StandaloneResourceCreateInput, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneResourceCreateInput)
	if err != nil {
		return nil, err
	}
	return input.JSON(input), nil
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
