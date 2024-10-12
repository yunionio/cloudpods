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
	"time"
)

const (
	DEFAULT_PROJECT = "default"

	ACCESS_SECRET_TYPE    = "aksk"
	TOTP_TYPE             = "totp"
	RECOVERY_SECRETS_TYPE = "recovery_secret"
	OIDC_CREDENTIAL_TYPE  = "oidc"
	ENCRYPT_KEY_TYPE      = "enc_key"
	CONTAINER_IMAGE_TYPE  = "container_image"
)

type SAccessKeySecretBlob struct {
	Secret string `json:"secret"`
	Expire int64  `json:"expire"`
}

func (info SAccessKeySecretBlob) IsValid() bool {
	if info.Expire <= 0 || info.Expire > time.Now().Unix() {
		return true
	}
	return false
}

type SAccessKeySecretInfo struct {
	AccessKey string
	SAccessKeySecretBlob
}
