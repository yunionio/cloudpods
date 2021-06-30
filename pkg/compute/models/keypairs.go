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
	"strings"

	"golang.org/x/crypto/ssh"

	"yunion.io/x/jsonutils"
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
	db.SUserResourceBaseManager
}

var KeypairManager *SKeypairManager

func init() {
	KeypairManager = &SKeypairManager{
		SUserResourceBaseManager: db.NewUserResourceBaseManager(
			SKeypair{},
			"keypairs_tbl",
			"keypair",
			"keypairs",
		),
	}
	KeypairManager.SetVirtualObject(KeypairManager)
}

type SKeypair struct {
	db.SUserResourceBase

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
}

// 列出ssh密钥对
func (manager *SKeypairManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.KeypairListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SUserResourceBaseManager.ListItemFilter(ctx, q, userCred, query.UserResourceListInput)
	if err != nil {
		return nil, err
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
	q, err = manager.SUserResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.UserResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SUserResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SKeypairManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SUserResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
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
	userRows := manager.SUserResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		keypair := objs[i].(*SKeypair)
		rows[i] = api.KeypairDetails{
			UserResourceDetails: userRows[i],
			PrivateKeyLen:       len(keypair.PrivateKey),
		}
		rows[i].LinkedGuestCount, _ = keypair.GetLinkedGuestsCount()
	}

	return rows
}

func (self *SKeypair) GetLinkedGuestsCount() (int, error) {
	return GuestManager.Query().Equals("keypair_id", self.Id).CountWithError()
}

func (manager *SKeypairManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.KeypairCreateInput) (api.KeypairCreateInput, error) {
	input.PublicKey = strings.TrimSpace(input.PublicKey)
	if len(input.PublicKey) == 0 {
		if len(input.Scheme) == 0 {
			input.Scheme = api.KEYPAIRE_SCHEME_RSA
		}
		if !utils.IsInStringArray(input.Scheme, api.KEYPAIR_SCHEMAS) {
			return input, httperrors.NewInputParameterError("Unsupported scheme %s", input.Scheme)
		}

		var err error
		if input.Scheme == api.KEYPAIRE_SCHEME_RSA {
			input.PrivateKey, input.PublicKey, err = seclib2.GenerateRSASSHKeypair()
		} else {
			input.PrivateKey, input.PublicKey, err = seclib2.GenerateDSASSHKeypair()
		}
		if err != nil {
			return input, httperrors.NewGeneralError(errors.Wrapf(err, "Generate%sSSHKeypair", input.Scheme))
		}
	}
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(input.PublicKey))
	if err != nil {
		return input, httperrors.NewInputParameterError("invalid public error: %v", err)
	}

	// 只允许上传RSA格式密钥。PS: AWS只支持RSA格式。
	input.Scheme = seclib2.GetPublicKeyScheme(pubKey)
	if input.Scheme != api.KEYPAIRE_SCHEME_RSA {
		return input, httperrors.NewInputParameterError("Unsupported scheme %s", input.Scheme)
	}

	input.Fingerprint = ssh.FingerprintLegacyMD5(pubKey)
	input.UserResourceCreateInput, err = manager.SUserResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.UserResourceCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
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
