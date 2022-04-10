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

package apis

import (
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SEncryptInfo struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`

	Alg seclib2.TSymEncAlg `json:"alg"`
}

type EncryptedResourceCreateInput struct {
	// 是否新建密钥
	EncryptKeyNew *bool `json:"encrypt_key_new"`

	// 新建密钥算法
	EncryptKeyAlg *string `json:"encrypt_key_alg"`

	// 加密秘钥的ID
	EncryptKeyId *string `json:"encrypt_key_id"`
}

type EncryptedResourceDetails struct {
	// 秘钥名称
	EncryptKey string `json:"encrypt_key"`

	// 加密算法，aes-256 or sm4
	EncryptAlg string `json:"encrypt_alg"`
}
