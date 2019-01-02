package signers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth/credentials"
)

type AccessKeySigner struct {
	credential *credentials.AccessKeyCredential
}

func (signer *AccessKeySigner) GetName() string {
	return "HmacSha256"
}

func (signer *AccessKeySigner) GetAccessKeyId() (accessKeyId string, err error) {
	return signer.credential.AccessKeyId, nil
}

func (signer *AccessKeySigner) GetSecretKey() (secretKey string, err error) {
	return signer.credential.AccessKeySecret, nil
}

func (signer *AccessKeySigner) Sign(stringToSign, secretSuffix string) string {
	return hex.EncodeToString(HmacSha256(stringToSign, []byte(secretSuffix)))
}

func HmacSha256(data string, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func NewAccessKeySigner(credential *credentials.AccessKeyCredential) *AccessKeySigner {
	return &AccessKeySigner{
		credential: credential,
	}
}
