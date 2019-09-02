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
	"yunion.io/x/jsonutils"
)

type ValidatorStringLen struct {
	Validator
	Value  string
	minLen int
	maxLen int
}

func NewStringLenRangeValidator(key string, minLen, maxLen int) *ValidatorStringLen {
	v := &ValidatorStringLen{
		Validator: Validator{Key: key},
		minLen:    minLen,
		maxLen:    maxLen,
	}
	v.SetParent(v)
	return v
}

func NewMinStringLenValidator(key string, minLen int) *ValidatorStringLen {
	return NewStringLenRangeValidator(key, minLen, -1)
}

func NewMaxStringLenValidator(key string, maxLen int) *ValidatorStringLen {
	return NewStringLenRangeValidator(key, -1, maxLen)
}

func NewStringNonEmptyValidator(key string) *ValidatorStringLen {
	return NewMinStringLenValidator(key, 1)
}

func (v *ValidatorStringLen) Default(s string) IValidator {
	if v.minLen >= 0 && len(s) < v.minLen {
		panic("invalid default string: shorter than validator requirement")
	}
	if v.maxLen >= 0 && len(s) > v.maxLen {
		panic("invalid default string: shorter than validator requirement")
	}
	return v.Validator.Default(s)
}

func (v *ValidatorStringLen) getValue() interface{} {
	return v.Value
}

func (v *ValidatorStringLen) Validate(data *jsonutils.JSONDict) error {
	if err, isSet := v.Validator.validateEx(data); err != nil || !isSet {
		return err
	}
	s, err := v.value.GetString()
	if err != nil {
		return newGeneralError(v.Key, err)
	}
	if v.minLen >= 0 && len(s) < v.minLen {
		return newStringTooShortError(v.Key, len(s), v.minLen)
	}
	if v.maxLen >= 0 && len(s) > v.maxLen {
		return newStringTooLongError(v.Key, len(s), v.maxLen)
	}
	data.Set(v.Key, jsonutils.NewString(s))
	v.Value = s
	return nil
}
