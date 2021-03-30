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

package ecloud

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"

	"yunion.io/x/pkg/util/stringutils"
)

type ISigner interface {
	GetName() string
	GetVersion() string
	GetAccessKeyId() string
	GetNonce() string
	Sign(stringToSign, secretPrefix string) string
}

type SRamRoleSigner struct {
	accessKeyId     string
	accessKeySecret string
}

func NewRamRoleSigner(accessKeyId, accessKeySecret string) *SRamRoleSigner {
	return &SRamRoleSigner{
		accessKeyId:     accessKeyId,
		accessKeySecret: accessKeySecret,
	}
}

func (s *SRamRoleSigner) GetName() string {
	return "HmacSHA1"
}

func (s *SRamRoleSigner) GetVersion() string {
	return "V2.0"
}

func (s *SRamRoleSigner) GetAccessKeyId() string {
	return s.accessKeyId
}

func (s *SRamRoleSigner) GetNonce() string {
	return stringutils.UUID4()
}

func (s *SRamRoleSigner) Sign(stringToSign, secretPrefix string) string {
	secret := secretPrefix + s.accessKeySecret
	return shaHmac1(stringToSign, secret)
}

func shaHmac1(source, secret string) string {
	key := []byte(secret)
	hmac := hmac.New(sha1.New, key)
	hmac.Write([]byte(source))
	signedBytes := hmac.Sum(nil)
	return hex.EncodeToString(signedBytes)
}
