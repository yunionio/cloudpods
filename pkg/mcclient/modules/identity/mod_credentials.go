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

package identity

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/seclib"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SCredentialManager struct {
	modulebase.ResourceManager
}

const (
	DEFAULT_PROJECT = api.DEFAULT_PROJECT

	ACCESS_SECRET_TYPE    = api.ACCESS_SECRET_TYPE
	TOTP_TYPE             = api.TOTP_TYPE
	RECOVERY_SECRETS_TYPE = api.RECOVERY_SECRETS_TYPE
	OIDC_CREDENTIAL_TYPE  = api.OIDC_CREDENTIAL_TYPE
	ENCRYPT_KEY_TYPE      = api.ENCRYPT_KEY_TYPE
)

type STotpSecret struct {
	Totp      string `json:"totp"`
	Timestamp int64  `json:"timestamp"`
}

type SRecoverySecret struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type SAccessKeySecret struct {
	KeyId     string    `json:"-"`
	ProjectId string    `json:"-"`
	TimeStamp time.Time `json:"-"`
	api.SAccessKeySecretBlob
}

type SRecoverySecretSet struct {
	Questions []SRecoverySecret
	Timestamp int64
}

type SOpenIDConnectCredential struct {
	ClientId string `json:"client_id"`
	// Secret      string `json:"secret"`
	RedirectUri string `json:"redirect_uri"`
	api.SAccessKeySecretBlob
}

type SEncryptKeySecret struct {
	KeyId     string             `json:"-"`
	KeyName   string             `json:"-"`
	Alg       seclib2.TSymEncAlg `json:"alg"`
	Key       string             `json:"key"`
	TimeStamp time.Time          `json:"-"`
	UserId    string             `json:"user_id"`
	User      string             `json:"user"`
	Domain    string             `json:"domain"`
	DomainId  string             `json:"domain_id"`
}

func (key SEncryptKeySecret) Marshal() jsonutils.JSONObject {
	json := jsonutils.NewDict()
	json.Add(jsonutils.NewString(string(key.Alg)), "alg")
	json.Add(jsonutils.NewString(key.KeyId), "key_id")
	json.Add(jsonutils.NewString(key.KeyName), "key_name")
	json.Add(jsonutils.NewTimeString(key.TimeStamp), "timestamp")
	json.Add(jsonutils.NewString(key.UserId), "user_id")
	json.Add(jsonutils.NewString(key.User), "user")
	json.Add(jsonutils.NewString(key.DomainId), "domain_id")
	json.Add(jsonutils.NewString(key.Domain), "domain")
	return json
}

func (key SEncryptKeySecret) Encrypt(secret []byte) ([]byte, error) {
	bKey, err := base64.StdEncoding.DecodeString(key.Key)
	if err != nil {
		return nil, errors.Wrap(err, "base64.StdEncoding.DecodeString")
	}
	return seclib2.Alg(key.Alg).CbcEncode(secret, bKey)
}

func (key SEncryptKeySecret) Decrypt(secret []byte) ([]byte, error) {
	bKey, err := base64.StdEncoding.DecodeString(key.Key)
	if err != nil {
		return nil, errors.Wrap(err, "base64.StdEncoding.DecodeString")
	}
	return seclib2.Alg(key.Alg).CbcDecode(secret, bKey)
}

func (key SEncryptKeySecret) EncryptBase64(secret []byte) (string, error) {
	bKey, err := base64.StdEncoding.DecodeString(key.Key)
	if err != nil {
		return "", errors.Wrap(err, "base64.StdEncoding.DecodeString")
	}
	return seclib2.Alg(key.Alg).CbcEncodeBase64(secret, bKey)
}

func (key SEncryptKeySecret) DecryptBase64(secret string) ([]byte, error) {
	bKey, err := base64.StdEncoding.DecodeString(key.Key)
	if err != nil {
		return nil, errors.Wrap(err, "base64.StdEncoding.DecodeString")
	}
	return seclib2.Alg(key.Alg).CbcDecodeBase64(secret, bKey)
}

func (manager *SCredentialManager) fetchCredentials(s *mcclient.ClientSession, secType string, uid string, pid string) ([]jsonutils.JSONObject, error) {
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString(secType), "type")
	if len(uid) > 0 {
		query.Add(jsonutils.NewString(uid), "user_id")
	}
	if len(pid) > 0 {
		query.Add(jsonutils.NewString(pid), "project_id")
	}
	query.Add(jsonutils.JSONTrue, "details")
	results, err := manager.List(s, query)
	if err != nil {
		return nil, err
	}
	return results.Data, nil
}

func (manager *SCredentialManager) FetchAccessKeySecrets(s *mcclient.ClientSession, uid string, pid string) ([]jsonutils.JSONObject, error) {
	return manager.fetchCredentials(s, ACCESS_SECRET_TYPE, uid, pid)
}

func (manager *SCredentialManager) FetchTotpSecrets(s *mcclient.ClientSession, uid string) ([]jsonutils.JSONObject, error) {
	return manager.fetchCredentials(s, TOTP_TYPE, uid, "")
}

func (manager *SCredentialManager) FetchRecoverySecrets(s *mcclient.ClientSession, uid string) ([]jsonutils.JSONObject, error) {
	return manager.fetchCredentials(s, RECOVERY_SECRETS_TYPE, uid, "")
}

func (manager *SCredentialManager) FetchOIDCSecrets(s *mcclient.ClientSession, uid string, pid string) ([]jsonutils.JSONObject, error) {
	return manager.fetchCredentials(s, OIDC_CREDENTIAL_TYPE, uid, pid)
}

func (manager *SCredentialManager) FetchEncryptionKeys(s *mcclient.ClientSession, uid string) ([]jsonutils.JSONObject, error) {
	return manager.fetchCredentials(s, ENCRYPT_KEY_TYPE, uid, "")
}

func (manager *SCredentialManager) GetTotpSecret(s *mcclient.ClientSession, uid string) (string, error) {
	secrets, err := manager.FetchTotpSecrets(s, uid)
	if err != nil {
		return "", err
	}
	latestTotp := STotpSecret{}
	find := false
	for i := range secrets {
		blobStr, _ := secrets[i].GetString("blob")
		blobJson, _ := jsonutils.ParseString(blobStr)
		if blobJson != nil {
			totp := STotpSecret{}
			blobJson.Unmarshal(&totp)
			if latestTotp.Timestamp == 0 || totp.Timestamp > latestTotp.Timestamp {
				latestTotp = totp
				find = true
			}
		}
	}
	if !find {
		return "", httperrors.NewNotFoundError("no totp for %s", uid)
	}
	return latestTotp.Totp, nil
}

func (manager *SCredentialManager) GetRecoverySecrets(s *mcclient.ClientSession, uid string) ([]SRecoverySecret, error) {
	secrets, err := manager.FetchRecoverySecrets(s, uid)
	if err != nil {
		return nil, err
	}
	latestQ := SRecoverySecretSet{}
	find := false
	for i := range secrets {
		blobStr, _ := secrets[i].GetString("blob")
		blobJson, _ := jsonutils.ParseString(blobStr)
		if blobJson != nil {
			curr := SRecoverySecretSet{}
			blobJson.Unmarshal(&curr)
			if latestQ.Timestamp == 0 || curr.Timestamp > latestQ.Timestamp {
				latestQ = curr
				find = true
			}
		}
	}
	if !find {
		return nil, httperrors.NewNotFoundError("no recovery secrets for %s", uid)
	}
	return latestQ.Questions, nil
}

func DecodeAccessKeySecret(secret jsonutils.JSONObject) (SAccessKeySecret, error) {
	curr := SAccessKeySecret{}
	blobStr, err := secret.GetString("blob")
	if err != nil {
		return curr, errors.Wrap(err, "secret.GetString")
	}
	blobJson, err := jsonutils.ParseString(blobStr)
	if err != nil {
		return curr, errors.Wrap(err, "jsonutils.ParseString")
	}
	err = blobJson.Unmarshal(&curr)
	if err != nil {
		return curr, errors.Wrap(err, "blobJson.Unmarshal")
	}
	curr.ProjectId, err = secret.GetString("project_id")
	if err != nil {
		return curr, errors.Wrap(err, "secret.GetString('project_id')")
	}
	curr.TimeStamp, err = secret.GetTime("created_at")
	if err != nil {
		return curr, errors.Wrap(err, "secret.GetTime('created_at')")
	}
	curr.KeyId, err = secret.GetString("id")
	if err != nil {
		return curr, errors.Wrap(err, "secret.GetString('id')")
	}
	return curr, nil
}

func (manager *SCredentialManager) GetAccessKeySecrets(s *mcclient.ClientSession, uid string, pid string) ([]SAccessKeySecret, error) {
	secrets, err := manager.FetchAccessKeySecrets(s, uid, pid)
	if err != nil {
		return nil, err
	}
	aksk := make([]SAccessKeySecret, 0)
	for i := range secrets {
		curr, err := DecodeAccessKeySecret(secrets[i])
		if err != nil {
			return nil, errors.Wrap(err, "DecodeAccessKeySecret")
		}
		aksk = append(aksk, curr)
	}
	return aksk, nil
}

func DecodeOIDCSecret(secret jsonutils.JSONObject) (SOpenIDConnectCredential, error) {
	curr := SOpenIDConnectCredential{}
	blobStr, err := secret.GetString("blob")
	if err != nil {
		return curr, errors.Wrap(err, "secret.GetString")
	}
	blobJson, err := jsonutils.ParseString(blobStr)
	if err != nil {
		return curr, errors.Wrap(err, "jsonutils.ParseString")
	}
	err = blobJson.Unmarshal(&curr)
	if err != nil {
		return curr, errors.Wrap(err, "blobJson.Unmarshal")
	}
	curr.ClientId, err = secret.GetString("id")
	if err != nil {
		return curr, errors.Wrap(err, "secret.GetString('id')")
	}
	return curr, nil
}

func (manager *SCredentialManager) GetOIDCSecret(s *mcclient.ClientSession, uid string, pid string) ([]SOpenIDConnectCredential, error) {
	secrets, err := manager.FetchOIDCSecrets(s, uid, pid)
	if err != nil {
		return nil, err
	}
	oidcCreds := make([]SOpenIDConnectCredential, 0)
	for i := range secrets {
		curr, err := DecodeOIDCSecret(secrets[i])
		if err != nil {
			return nil, errors.Wrap(err, "DecodeOIDCSecret")
		}
		oidcCreds = append(oidcCreds, curr)
	}
	return oidcCreds, nil
}

func DecodeEncryptKey(secret jsonutils.JSONObject) (SEncryptKeySecret, error) {
	curr := SEncryptKeySecret{}
	blobStr, err := secret.GetString("blob")
	if err != nil {
		return curr, errors.Wrap(err, "secret.GetString")
	}
	blobJson, err := jsonutils.ParseString(blobStr)
	if err != nil {
		return curr, errors.Wrap(err, "jsonutils.ParseString")
	}
	err = blobJson.Unmarshal(&curr)
	if err != nil {
		return curr, errors.Wrap(err, "blobJson.Unmarshal")
	}
	curr.KeyId, err = secret.GetString("id")
	if err != nil {
		return curr, errors.Wrap(err, "secret.GetString('id')")
	}
	curr.KeyName, err = secret.GetString("name")
	if err != nil {
		return curr, errors.Wrap(err, "secret.GetString('name')")
	}
	curr.TimeStamp, err = secret.GetTime("created_at")
	if err != nil {
		return curr, errors.Wrap(err, "secret.GetTime('created_at')")
	}
	curr.User, err = secret.GetString("user")
	if err != nil {
		return curr, errors.Wrap(err, "secret.GetString('user')")
	}
	curr.UserId, err = secret.GetString("user_id")
	if err != nil {
		return curr, errors.Wrap(err, "secret.GetString('user_id')")
	}
	curr.Domain, err = secret.GetString("domain")
	if err != nil {
		return curr, errors.Wrap(err, "secret.GetString('domain')")
	}
	curr.DomainId, err = secret.GetString("domain_id")
	if err != nil {
		return curr, errors.Wrap(err, "secret.GetString('domain_id')")
	}
	return curr, nil
}

func (manager *SCredentialManager) GetEncryptKeysRpc(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.
	JSONObject, error) {
	keys, err := manager.GetEncryptKeys(s, "")
	if err != nil {
		return nil, errors.Wrap(err, "GetEncryptKeys")
	}
	ret := jsonutils.NewArray()
	for _, key := range keys {
		ret.Add(key.Marshal())
	}
	return ret, nil
}

func (manager *SCredentialManager) GetEncryptKeys(s *mcclient.ClientSession, uid string) ([]SEncryptKeySecret, error) {
	secrets, err := manager.FetchEncryptionKeys(s, uid)
	if err != nil {
		return nil, errors.Wrap(err, "FetchAesKeys")
	}
	aesKeys := make([]SEncryptKeySecret, 0)
	for i := range secrets {
		curr, err := DecodeEncryptKey(secrets[i])
		if err != nil {
			return nil, errors.Wrap(err, "DecodeAesKey")
		}
		aesKeys = append(aesKeys, curr)
	}
	return aesKeys, nil
}

func (manager *SCredentialManager) DoCreateAccessKeySecret(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	key, err := manager.CreateAccessKeySecret(s, "", "", time.Time{})
	if err != nil {
		return nil, err
	}
	result := jsonutils.Marshal(key)
	result.(*jsonutils.JSONDict).Add(jsonutils.NewString(key.KeyId), "key_id")
	return result, nil
}

func (manager *SCredentialManager) CreateAccessKeySecret(s *mcclient.ClientSession, uid string, pid string, expireAt time.Time) (SAccessKeySecret, error) {
	aksk := SAccessKeySecret{}
	aksk.Secret = base64.URLEncoding.EncodeToString([]byte(seclib.RandomPassword(32)))
	if !expireAt.IsZero() {
		aksk.Expire = expireAt.Unix()
	}
	blobJson := jsonutils.Marshal(&aksk)
	params := jsonutils.NewDict()
	name := fmt.Sprintf("%s-%s-%d", uid, pid, time.Now().Unix())
	if len(pid) > 0 {
		params.Add(jsonutils.NewString(pid), "project_id")
	}
	params.Add(jsonutils.NewString(ACCESS_SECRET_TYPE), "type")
	if len(uid) > 0 {
		params.Add(jsonutils.NewString(uid), "user_id")
	}
	params.Add(jsonutils.NewString(blobJson.String()), "blob")
	params.Add(jsonutils.NewString(name), "name")
	result, err := manager.Create(s, params)
	if err != nil {
		return aksk, err
	}
	aksk.ProjectId = pid
	aksk.TimeStamp, _ = result.GetTime("created_at")
	aksk.KeyId, _ = result.GetString("id")
	return aksk, nil
}

func (manager *SCredentialManager) DoCreateOidcSecret(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	redirectUri, _ := params.GetString("redirect_uri")

	key, err := manager.CreateOIDCSecret(s, "", "", redirectUri)
	if err != nil {
		return nil, errors.Wrap(err, "CreateOIDCSecret")
	}
	result := jsonutils.Marshal(key)
	// result.(*jsonutils.JSONDict).Add(jsonutils.NewString(key.ClientId), "client_id")
	return result, nil
}

func isValidRedirectURL(redirectUri string) error {
	if len(redirectUri) == 0 {
		return errors.Wrap(httperrors.ErrInputParameter, "empty redirect uri")
	}
	if !strings.HasPrefix(redirectUri, "http://") && !strings.HasPrefix(redirectUri, "https://") {
		return errors.Wrap(httperrors.ErrInputParameter, "invalid schema")
	}
	_, err := url.Parse(redirectUri)
	if err != nil {
		return errors.Wrapf(httperrors.ErrInputParameter, "invalid redirect_uri %s", redirectUri)
	}
	return nil
}

func (manager *SCredentialManager) CreateOIDCSecret(s *mcclient.ClientSession, uid string, pid string, redirectUri string) (SOpenIDConnectCredential, error) {
	oidcCred := SOpenIDConnectCredential{}
	err := isValidRedirectURL(redirectUri)
	if err != nil {
		return oidcCred, errors.Wrap(err, "isValidRedirectURL")
	}
	oidcCred.Secret = base64.URLEncoding.EncodeToString([]byte(seclib.RandomPassword(32)))
	oidcCred.RedirectUri = redirectUri
	blobJson := jsonutils.Marshal(&oidcCred)
	params := jsonutils.NewDict()
	name := fmt.Sprintf("oidc-%s-%s-%d", uid, pid, time.Now().Unix())
	if len(pid) > 0 {
		params.Add(jsonutils.NewString(pid), "project_id")
	}
	params.Add(jsonutils.NewString(OIDC_CREDENTIAL_TYPE), "type")
	if len(uid) > 0 {
		params.Add(jsonutils.NewString(uid), "user_id")
	}
	params.Add(jsonutils.NewString(blobJson.String()), "blob")
	params.Add(jsonutils.NewString(name), "name")
	result, err := manager.Create(s, params)
	if err != nil {
		return oidcCred, err
	}
	oidcCred.ClientId, _ = result.GetString("id")
	return oidcCred, nil
}

func (manager *SCredentialManager) CreateTotpSecret(s *mcclient.ClientSession, uid string) (string, error) {
	_, err := manager.GetTotpSecret(s, uid)
	if err == nil {
		return "", httperrors.NewConflictError("totp secret exists")
	}
	totp := STotpSecret{
		Totp:      seclib.RandomPassword(20),
		Timestamp: time.Now().Unix(),
	}
	blobJson := jsonutils.Marshal(&totp)
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(DEFAULT_PROJECT), "project_id")
	params.Add(jsonutils.NewString(TOTP_TYPE), "type")
	params.Add(jsonutils.NewString(uid), "user_id")
	params.Add(jsonutils.NewString(blobJson.String()), "blob")
	_, err = manager.Create(s, params)
	if err != nil {
		return "", err
	}
	return totp.Totp, nil
}

func (manager *SCredentialManager) SaveRecoverySecrets(s *mcclient.ClientSession, uid string, questions []SRecoverySecret) error {
	_, err := manager.GetRecoverySecrets(s, uid)
	if err == nil {
		return httperrors.NewConflictError("totp secret exists")
	}
	sec := SRecoverySecretSet{
		Questions: questions,
		Timestamp: time.Now().Unix(),
	}
	blobJson := jsonutils.Marshal(&sec)
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(DEFAULT_PROJECT), "project_id")
	params.Add(jsonutils.NewString(RECOVERY_SECRETS_TYPE), "type")
	params.Add(jsonutils.NewString(uid), "user_id")
	params.Add(jsonutils.NewString(blobJson.String()), "blob")
	_, err = manager.Create(s, params)
	if err != nil {
		return err
	}
	return nil
}

func (manager *SCredentialManager) DoCreateEncryptKey(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	name, _ := params.GetString("name")
	alg, _ := params.GetString("alg")
	uid, _ := params.GetString("uid")
	key, err := manager.CreateEncryptKey(s, uid, name, alg)
	if err != nil {
		return nil, err
	}
	result := jsonutils.Marshal(key)
	result.(*jsonutils.JSONDict).Add(jsonutils.NewString(key.KeyId), "key_id")
	return result, nil
}

func (manager *SCredentialManager) CreateEncryptKey(s *mcclient.ClientSession, uid string, name string, algName string) (SEncryptKeySecret, error) {
	aesKey := SEncryptKeySecret{}
	alg := seclib2.Alg(seclib2.TSymEncAlg(algName))
	rawKey := alg.GenerateKey()
	aesKey.Key = base64.StdEncoding.EncodeToString([]byte(rawKey))
	aesKey.Alg = alg.Name()
	blobJson := jsonutils.Marshal(&aesKey)
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(ENCRYPT_KEY_TYPE), "type")
	if len(uid) > 0 {
		params.Add(jsonutils.NewString(uid), "user_id")
	}
	params.Add(jsonutils.NewString(api.DEFAULT_PROJECT), "project_id")
	params.Add(jsonutils.NewString(blobJson.String()), "blob")
	params.Add(jsonutils.NewString(name), "generate_name")
	result, err := manager.Create(s, params)
	if err != nil {
		return aesKey, errors.Wrap(err, "Create")
	}
	aesKey.KeyId, _ = result.GetString("id")
	aesKey.KeyName, _ = result.GetString("name")
	aesKey.TimeStamp, _ = result.GetTime("created_at")
	return aesKey, nil
}

func (manager *SCredentialManager) removeCredentials(s *mcclient.ClientSession, secType string, uid string, pid string) error {
	secrets, err := manager.fetchCredentials(s, secType, uid, pid)
	if err != nil {
		return err
	}
	for i := range secrets {
		sid, _ := secrets[i].GetString("id")
		if len(sid) > 0 {
			_, err := manager.Delete(s, sid, nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (manager *SCredentialManager) RemoveAccessKeySecrets(s *mcclient.ClientSession, uid string, pid string) error {
	return manager.removeCredentials(s, ACCESS_SECRET_TYPE, uid, pid)
}

func (manager *SCredentialManager) RemoveTotpSecrets(s *mcclient.ClientSession, uid string) error {
	return manager.removeCredentials(s, TOTP_TYPE, uid, "")
}

func (manager *SCredentialManager) RemoveRecoverySecrets(s *mcclient.ClientSession, uid string) error {
	return manager.removeCredentials(s, RECOVERY_SECRETS_TYPE, uid, "")
}

func (manager *SCredentialManager) RemoveOIDCSecrets(s *mcclient.ClientSession, uid string, pid string) error {
	return manager.removeCredentials(s, OIDC_CREDENTIAL_TYPE, uid, pid)
}

func (manager *SCredentialManager) RemoveEncryptKeys(s *mcclient.ClientSession, uid string) error {
	return manager.removeCredentials(s, ENCRYPT_KEY_TYPE, uid, "")
}

func (manager *SCredentialManager) EncryptKeyEncrypt(s *mcclient.ClientSession, keyId string, secret []byte) ([]byte, error) {
	aesKey, err := manager.GetEncryptKey(s, keyId)
	if err != nil {
		return nil, errors.Wrap(err, "GetEncryptKey")
	}
	return aesKey.Encrypt(secret)
}

func (manager *SCredentialManager) EncryptKeyDecrypt(s *mcclient.ClientSession, keyId string, secret []byte) ([]byte, error) {
	aesKey, err := manager.GetEncryptKey(s, keyId)
	if err != nil {
		return nil, errors.Wrap(err, "GetEncryptKey")
	}
	return aesKey.Decrypt(secret)
}

func (manager *SCredentialManager) EncryptKeyEncryptBase64(s *mcclient.ClientSession, keyId string, secret []byte) (string, error) {
	aesKey, err := manager.GetEncryptKey(s, keyId)
	if err != nil {
		return "", errors.Wrap(err, "GetEncryptKey")
	}
	return aesKey.EncryptBase64(secret)
}

func (manager *SCredentialManager) EncryptKeyDecryptBase64(s *mcclient.ClientSession, keyId string, secret string) ([]byte, error) {
	aesKey, err := manager.GetEncryptKey(s, keyId)
	if err != nil {
		return nil, errors.Wrap(err, "GetEncryptKey")
	}
	return aesKey.DecryptBase64(secret)
}

func (manager *SCredentialManager) GetEncryptKey(s *mcclient.ClientSession, kid string) (SEncryptKeySecret, error) {
	ret := SEncryptKeySecret{}
	cred, err := manager.Get(s, kid, nil)
	if err != nil {
		return ret, errors.Wrap(err, "Get")
	}
	ret, err = DecodeEncryptKey(cred)
	if err != nil {
		return ret, errors.Wrap(err, "DecodeAesKey")
	}
	return ret, nil
}

var (
	Credentials SCredentialManager
)

func init() {
	Credentials = SCredentialManager{
		modules.NewIdentityV3Manager("credential", "credentials",
			[]string{},
			[]string{"ID", "Type", "user_id", "project_id", "blob"}),
	}

	modules.Register(&Credentials)
}
