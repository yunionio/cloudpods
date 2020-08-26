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

package compute

import "yunion.io/x/onecloud/pkg/apis"

var KEYPAIR_SCHEMAS = []string{
	KEYPAIRE_SCHEME_RSA,
}

type KeypairCreateInput struct {
	apis.UserResourceCreateInput

	// 公钥内容,若为空则自动生成公钥
	PublicKey string `json:"public_key"`

	// swagger:ignore
	PrivateKey string

	// swagger:ignore
	Fingerprint string

	// 秘钥类型
	// enum: RSA
	// default: RSA
	Scheme string `json:"scheme"`
}

type KeypairDetails struct {
	apis.UserResourceDetails
	SKeypair

	// 私钥长度
	PrivateKeyLen int `json:"private_key_len"`
	// 关联云主机次数
	LinkedGuestCount int `json:"linked_guest_count"`
}
