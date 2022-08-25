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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/util/timeutils"

	"yunion.io/x/onecloud/pkg/apis"
	identity_apis "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	identity_modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SEncryptedResourceManager struct {
}

// +onecloud:model-api-gen
type SEncryptedResource struct {
	// 加密密钥ID
	EncryptKeyId string `width:"32" charset:"ascii" nullable:"true" get:"user" list:"user" create:"optional"`
}

func (res *SEncryptedResource) IsEncrypted() bool {
	return len(res.EncryptKeyId) > 0
}

func (res *SEncryptedResource) GetEncryptInfo(
	ctx context.Context,
	userCred mcclient.TokenCredential,
) (apis.SEncryptInfo, error) {
	ret := apis.SEncryptInfo{}
	session := auth.GetSession(ctx, userCred, consts.GetRegion())
	secKey, err := identity_modules.Credentials.GetEncryptKey(session, res.EncryptKeyId)
	if err != nil {
		return ret, errors.Wrap(err, "GetEncryptKey")
	}
	ret.Id = secKey.KeyId
	ret.Name = secKey.KeyName
	ret.Key = secKey.Key
	ret.Alg = secKey.Alg
	return ret, nil
}

func (res *SEncryptedResource) ValidateEncryption(ctx context.Context, userCred mcclient.TokenCredential) error {
	if res.IsEncrypted() {
		_, err := res.GetEncryptInfo(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "GetEncryptInfo")
		}
	}
	return nil
}

func (manager *SEncryptedResourceManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input apis.EncryptedResourceCreateInput,
) (apis.EncryptedResourceCreateInput, error) {
	if input.EncryptKeyId != nil && len(*input.EncryptKeyId) > 0 {
		session := auth.GetSession(ctx, userCred, consts.GetRegion())
		keyObj, err := identity_modules.Credentials.Get(session, *input.EncryptKeyId, nil)
		if err != nil {
			return input, errors.Wrap(err, "Credentials get key")
		}
		keyType, _ := keyObj.GetString("type")
		if keyType != identity_apis.ENCRYPT_KEY_TYPE {
			return input, errors.Wrap(httperrors.ErrInvalidFormat, "key type is not enc_key")
		}
		keyId, err := keyObj.GetString("id")
		if err != nil {
			return input, errors.Wrap(err, "GetString key Id")
		}
		input.EncryptKeyId = &keyId
	}
	return input, nil
}

func (res *SEncryptedResource) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data jsonutils.JSONObject, nameHint string) error {
	if len(res.EncryptKeyId) == 0 && jsonutils.QueryBoolean(data, "encrypt_key_new", false) && !jsonutils.QueryBoolean(data, "dry_run", false) {
		// create new encrypt key
		session := auth.GetAdminSession(ctx, consts.GetRegion())
		now := time.Now()
		keyName := "key-" + nameHint + "-" + timeutils.ShortDate(now)
		algName, _ := data.GetString("encrypt_key_alg")
		userId, _ := data.GetString("encrypt_key_user_id")
		secret, err := identity_modules.Credentials.CreateEncryptKey(session, userId, keyName, algName)
		if err != nil {
			return errors.Wrap(err, "Credentials.CreateEncryptKey")
		}
		res.EncryptKeyId = secret.KeyId
	}
	return nil
}

func (manager *SEncryptedResourceManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.EncryptedResourceDetails {
	rets := make([]apis.EncryptedResourceDetails, len(objs))

	session := auth.GetSession(ctx, userCred, consts.GetRegion())
	encKeyMap := make(map[string]identity_modules.SEncryptKeySecret)
	for i := range objs {
		var base *SEncryptedResource
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil && len(base.EncryptKeyId) > 0 {
			encKey, ok := encKeyMap[base.EncryptKeyId]
			if !ok {
				secKey, err := identity_modules.Credentials.GetEncryptKey(session, base.EncryptKeyId)
				if err != nil {
					log.Errorf("fail to fetch enc key %s: %s", base.EncryptKeyId, err)
					continue
				}
				encKey = secKey
				encKeyMap[base.EncryptKeyId] = secKey
			}
			rets[i].EncryptKey = encKey.KeyName
			rets[i].EncryptAlg = string(encKey.Alg)
			rets[i].EncryptKeyUser = string(encKey.User)
			rets[i].EncryptKeyUserId = string(encKey.UserId)
			rets[i].EncryptKeyUserDomain = string(encKey.Domain)
			rets[i].EncryptKeyUserDomainId = string(encKey.DomainId)
		}
	}
	return rets
}
