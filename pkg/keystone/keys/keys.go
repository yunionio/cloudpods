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

package keys

import (
	"yunion.io/x/onecloud/pkg/util/fernetool"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

var (
	TokenKeysManager     = fernetool.SFernetKeyManager{}
	CredentialKeyManager = fernetool.SFernetKeyManager{}
)

func Init(tokenKeyRepo, credKeyRepo string) error {
	err := TokenKeysManager.LoadKeys(tokenKeyRepo)
	if err != nil {
		return err
	}
	if fileutils2.IsDir(credKeyRepo) {
		err = CredentialKeyManager.LoadKeys(credKeyRepo)
	} else {
		err = CredentialKeyManager.InitEmpty()
	}
	if err != nil {
		return err
	}
	return nil
}
