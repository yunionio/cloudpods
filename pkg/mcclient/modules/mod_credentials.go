package modules

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/seclib"

	"time"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCredentialManager struct {
	ResourceManager
}

const (
	DEFAULT_PROJECT = "default"

	TOTP_TYPE             = "totp"
	RECOVERY_SECRETS_TYPE = "recovery_secret"
)

type STotpSecret struct {
	Totp      string
	Timestamp int64
}

type SRecoverySecret struct {
	Question string
	Answer   string
}

type SRecoverySecretSet struct {
	Questions []SRecoverySecret
	Timestamp int64
}

func (manager *SCredentialManager) fetchCredentials(s *mcclient.ClientSession, secType string, uid string) ([]jsonutils.JSONObject, error) {
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString(secType), "type")
	query.Add(jsonutils.NewString(uid), "user_id")
	results, err := manager.List(s, query)
	if err != nil {
		return nil, err
	}
	return results.Data, nil
}

func (manager *SCredentialManager) FetchTotpSecrets(s *mcclient.ClientSession, uid string) ([]jsonutils.JSONObject, error) {
	return manager.fetchCredentials(s, TOTP_TYPE, uid)
}

func (manager *SCredentialManager) FetchRecoverySecrets(s *mcclient.ClientSession, uid string) ([]jsonutils.JSONObject, error) {
	return manager.fetchCredentials(s, RECOVERY_SECRETS_TYPE, uid)
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

func (manager *SCredentialManager) removeCredentials(s *mcclient.ClientSession, secType string, uid string) error {
	secrets, err := manager.fetchCredentials(s, secType, uid)
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

func (manager *SCredentialManager) RemoveTotpSecrets(s *mcclient.ClientSession, uid string) error {
	return manager.removeCredentials(s, TOTP_TYPE, uid)
}

func (manager *SCredentialManager) RemoveRecoverySecrets(s *mcclient.ClientSession, uid string) error {
	return manager.removeCredentials(s, RECOVERY_SECRETS_TYPE, uid)
}

var (
	Credentials SCredentialManager
)

func init() {
	Credentials = SCredentialManager{
		NewIdentityV3Manager("credential", "credentials",
			[]string{},
			[]string{"ID", "Type", "user_id", "project_id", "blob"}),
	}
}
