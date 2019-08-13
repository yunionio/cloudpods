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

package utils

import (
	"fmt"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func MustNewKey() wgtypes.Key {
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		panic(fmt.Errorf("new key: %v", err))
	}
	return key
}

func MustNewKeyString() string {
	key := MustNewKey()
	return key.String()
}

func MustParseKeyString(k string) wgtypes.Key {
	key, err := wgtypes.ParseKey(k)
	if err != nil {
		panic(fmt.Errorf("invalid key: %v", err))
	}
	return key
}
