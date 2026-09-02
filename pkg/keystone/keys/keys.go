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
)

var (
	TokenKeysManager     = fernetool.SFernetKeyManager{}
	CredentialKeyManager = fernetool.SFernetKeyManager{}
)

var (
	// zeroCredentialKeyManager holds the all-zero key that older versions
	// used to encrypt credential blobs when SetupCredentialKeys was false
	// (the old default). Kept only to decrypt and migrate legacy blobs.
	zeroCredentialKeyManager = fernetool.SFernetKeyManager{}
	// legacyCredentialKeyManagers are tried in order when the dedicated
	// credential keys fail to decrypt a blob: first the all-zero key, then
	// the token keys (older versions with SetupCredentialKeys=true reused
	// the token keys for credentials due to a key-type bug).
	legacyCredentialKeyManagers []*fernetool.SFernetKeyManager
)

// InitLegacyCredentialKeys prepares the key managers able to decrypt
// credential blobs encrypted by older versions, for lazy migration.
// It must be called after both TokenKeysManager and CredentialKeyManager
// have been initialized.
func InitLegacyCredentialKeys() {
	zeroCredentialKeyManager.InitEmpty()
	legacyCredentialKeyManagers = []*fernetool.SFernetKeyManager{
		&zeroCredentialKeyManager,
		&TokenKeysManager,
	}
}

// DecryptCredentialBlob decrypts a credential blob with the dedicated
// credential keys and falls back to the legacy keys for blobs written by
// older versions. It reports whether a legacy key was used, so that callers
// can migrate the blob to the dedicated keys.
func DecryptCredentialBlob(tok []byte) ([]byte, bool) {
	blob := CredentialKeyManager.Decrypt(tok)
	if len(blob) > 0 {
		return blob, false
	}
	for _, m := range legacyCredentialKeyManagers {
		if m == nil {
			continue
		}
		blob = m.Decrypt(tok)
		if len(blob) > 0 {
			return blob, true
		}
	}
	return nil, false
}
