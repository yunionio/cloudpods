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
	"testing"

	"yunion.io/x/onecloud/pkg/util/fernetool"
)

func setupManagers(t *testing.T) {
	CredentialKeyManager = fernetool.SFernetKeyManager{}
	TokenKeysManager = fernetool.SFernetKeyManager{}
	if err := CredentialKeyManager.InitKeys("", 2); err != nil {
		t.Fatalf("init credential keys: %v", err)
	}
	if err := TokenKeysManager.InitKeys("", 2); err != nil {
		t.Fatalf("init token keys: %v", err)
	}
	InitLegacyCredentialKeys()
}

func TestDecryptCredentialBlobDedicatedKeys(t *testing.T) {
	setupManagers(t)
	blob := []byte(`{"secret":"abc"}`)
	enc, err := CredentialKeyManager.Encrypt(blob)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	dec, legacy := DecryptCredentialBlob(enc)
	if legacy {
		t.Fatalf("dedicated key blob reported as legacy")
	}
	if string(dec) != string(blob) {
		t.Fatalf("decrypt mismatch, got %q", string(dec))
	}
}

func TestDecryptCredentialBlobZeroKeyLegacy(t *testing.T) {
	setupManagers(t)
	zero := fernetool.SFernetKeyManager{}
	if err := zero.InitEmpty(); err != nil {
		t.Fatalf("init zero key: %v", err)
	}
	blob := []byte(`{"secret":"legacy"}`)
	enc, err := zero.Encrypt(blob)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	dec, legacy := DecryptCredentialBlob(enc)
	if !legacy {
		t.Fatalf("zero key blob not reported as legacy")
	}
	if string(dec) != string(blob) {
		t.Fatalf("decrypt mismatch, got %q", string(dec))
	}
	// after migration the blob decrypts with the dedicated keys only
	mig, err := CredentialKeyManager.Encrypt(dec)
	if err != nil {
		t.Fatalf("re-encrypt: %v", err)
	}
	dec2, legacy2 := DecryptCredentialBlob(mig)
	if legacy2 {
		t.Fatalf("migrated blob reported as legacy")
	}
	if string(dec2) != string(blob) {
		t.Fatalf("migrated decrypt mismatch, got %q", string(dec2))
	}
}

func TestDecryptCredentialBlobTokenKeysLegacy(t *testing.T) {
	setupManagers(t)
	blob := []byte(`{"secret":"token-keyed"}`)
	enc, err := TokenKeysManager.Encrypt(blob)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	dec, legacy := DecryptCredentialBlob(enc)
	if !legacy {
		t.Fatalf("token key blob not reported as legacy")
	}
	if string(dec) != string(blob) {
		t.Fatalf("decrypt mismatch, got %q", string(dec))
	}
}

func TestDecryptCredentialBlobInvalid(t *testing.T) {
	setupManagers(t)
	dec, legacy := DecryptCredentialBlob([]byte("not-a-valid-blob"))
	if dec != nil || legacy {
		t.Fatalf("invalid blob decrypted: %q legacy=%v", string(dec), legacy)
	}
}
