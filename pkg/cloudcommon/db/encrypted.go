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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"

	"yunion.io/x/onecloud/pkg/apis"
	identity_apis "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/esxi/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	identity_modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SEncryptedResourceManager struct {
}

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
	session := auth.GetSession(ctx, userCred, consts.GetRegion(), "")
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

func (manager *SEncryptedResourceManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input apis.EncryptedResourceCreateInput,
) (apis.EncryptedResourceCreateInput, error) {
	if input.EncryptKeyId != nil && len(*input.EncryptKeyId) > 0 {
		session := auth.GetSession(ctx, userCred, options.Options.Region, "v1")
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
		userId, err := keyObj.GetString("user_id")
		if err != nil {
			return input, errors.Wrap(err, "GetString user id")
		}
		if userId != ownerId.GetUserId() {
			return input, errors.Wrap(httperrors.ErrForbidden, "non-owner not allow to access key")
		}
		input.EncryptKeyId = &keyId
	}
	return input, nil
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

	session := auth.GetSession(ctx, userCred, consts.GetRegion(), "")
	encKeys, err := identity_modules.Credentials.GetEncryptKeys(session, userCred.GetUserId())
	if err != nil {
		return rets
	}
	encKeyMap := make(map[string]identity_modules.SEncryptKeySecret)
	for i := range encKeys {
		encKeyMap[encKeys[i].KeyId] = encKeys[i]
	}
	for i := range objs {
		var base *SEncryptedResource
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil && len(base.EncryptKeyId) > 0 {
			if encKey, ok := encKeyMap[base.EncryptKeyId]; ok {
				rets[i].EncryptKey = encKey.KeyName
				rets[i].EncryptAlg = string(encKey.Alg)
			}
		}
	}
	return rets
}
