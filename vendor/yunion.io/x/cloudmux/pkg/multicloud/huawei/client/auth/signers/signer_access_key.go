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

package signers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/auth/credentials"
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
