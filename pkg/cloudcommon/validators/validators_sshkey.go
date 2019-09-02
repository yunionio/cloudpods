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

package validators

import (
	"golang.org/x/crypto/ssh"

	"yunion.io/x/jsonutils"
)

type ValidatorSSHKey struct {
	Validator
	Value  string
	Signer ssh.Signer
}

func NewSSHKeyValidator(key string) *ValidatorSSHKey {
	v := &ValidatorSSHKey{
		Validator: Validator{Key: key},
	}
	v.SetParent(v)
	return v
}

func (v *ValidatorSSHKey) parseKey(s string) (ssh.Signer, error) {
	signer, err := ssh.ParsePrivateKey([]byte(s))
	if err != nil {
		return nil, err
	}
	return signer, nil
}

func (v *ValidatorSSHKey) Default(s string) IValidator {
	_, err := v.parseKey(s)
	if err != nil {
		panic(err)
	}
	v.Validator.Default(s)
	return v
}

func (v *ValidatorSSHKey) getValue() interface{} {
	return v.Value
}

func (v *ValidatorSSHKey) Validate(data *jsonutils.JSONDict) error {
	if err, isSet := v.Validator.validateEx(data); err != nil || !isSet {
		return err
	}
	s, err := v.value.GetString()
	if err != nil {
		return newGeneralError(v.Key, err)
	}
	if signer, err := v.parseKey(s); err != nil {
		return newInvalidValueError(v.Key, s)
	} else {
		v.Signer = signer
	}
	data.Set(v.Key, jsonutils.NewString(s))
	v.Value = s
	return nil
}
