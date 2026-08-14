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

package models

import (
	"strings"
	"testing"

	"yunion.io/x/pkg/utils"
)

func TestEncryptDecryptAtRest(t *testing.T) {
	id := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	plain := "sk-upstream-secret"
	enc, err := encryptAtRest(id, plain)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(enc, atRestPrefix) {
		t.Fatalf("expected prefix %q, got %q", atRestPrefix, enc)
	}
	if !secretLooksEncrypted(enc) {
		t.Fatal("ciphertext should look encrypted")
	}
	if got := decryptAtRest(id, enc); got != plain {
		t.Fatalf("decrypt got %q want %q", got, plain)
	}
	if got := decryptAtRest(id, plain); got != plain {
		t.Fatalf("plaintext fallback got %q", got)
	}
	if secretLooksEncrypted(plain) {
		t.Fatal("plaintext should not look encrypted")
	}
}

func TestDecryptLegacyCiphertextWithoutPrefix(t *testing.T) {
	id := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	plain := "sk-legacy-secret"
	legacy, err := utils.EncryptAESBase64(id, plain)
	if err != nil {
		t.Fatal(err)
	}
	if secretLooksEncrypted(legacy) {
		t.Fatal("legacy ciphertext has no prefix")
	}
	if got := decryptAtRest(id, legacy); got != plain {
		t.Fatalf("legacy decrypt got %q want %q", got, plain)
	}
	k := &SAiKey{Secret: legacy}
	k.Id = id
	if err := k.sealSecret(); err != nil {
		t.Fatal(err)
	}
	if !secretLooksEncrypted(k.Secret) {
		t.Fatal("reseal should add prefix")
	}
	if got := k.GetSecret(); got != plain {
		t.Fatalf("GetSecret after reseal=%q", got)
	}
}

func TestAiKeySealAndGetSecret(t *testing.T) {
	k := &SAiKey{Secret: "sk-plain-secret"}
	k.Id = "key-id-1"
	if err := k.sealSecret(); err != nil {
		t.Fatal(err)
	}
	if !secretLooksEncrypted(k.Secret) {
		t.Fatal("secret should be encrypted with prefix")
	}
	if got := k.GetSecret(); got != "sk-plain-secret" {
		t.Fatalf("GetSecret=%q", got)
	}
	if err := k.sealSecret(); err != nil {
		t.Fatal(err)
	}
	if got := k.GetSecret(); got != "sk-plain-secret" {
		t.Fatalf("reseal changed plaintext: %q", got)
	}
}

func TestAiKeyGetSecretPlaintext(t *testing.T) {
	k := &SAiKey{Secret: "still-plain"}
	k.Id = "key-id-2"
	if got := k.GetSecret(); got != "still-plain" {
		t.Fatalf("unmigrated GetSecret=%q", got)
	}
}

func TestVirtualKeySealHashAndLookup(t *testing.T) {
	plain := "sk-0123456789abcdef0123456789abcdef"
	m := &SAiVirtualKey{VirtualKey: plain}
	m.Id = "vk-id-1"
	if err := m.sealVirtualKey(); err != nil {
		t.Fatal(err)
	}
	if !secretLooksEncrypted(m.VirtualKey) {
		t.Fatal("virtual_key should be encrypted with prefix")
	}
	if m.VirtualKeyHash != virtualKeyDigest(plain) {
		t.Fatalf("hash=%q want %q", m.VirtualKeyHash, virtualKeyDigest(plain))
	}
	if got := m.GetVirtualKey(); got != plain {
		t.Fatalf("GetVirtualKey=%q", got)
	}
	if virtualKeyDigest("sk-other") == m.VirtualKeyHash {
		t.Fatal("different keys should not share hash")
	}
	if err := m.sealVirtualKey(); err != nil {
		t.Fatal(err)
	}
	if got := m.GetVirtualKey(); got != plain {
		t.Fatalf("reseal changed plaintext: %q", got)
	}
}

func TestMigrateVirtualKeySetsHash(t *testing.T) {
	plain := "sk-legacyplaintext0000000000000000"
	m := &SAiVirtualKey{VirtualKey: plain}
	m.Id = "vk-id-legacy"
	if secretLooksEncrypted(m.VirtualKey) {
		t.Fatal("legacy value should be plaintext")
	}
	if err := m.sealVirtualKey(); err != nil {
		t.Fatal(err)
	}
	if !secretLooksEncrypted(m.VirtualKey) {
		t.Fatal("migrated value should be encrypted")
	}
	if m.VirtualKeyHash == "" || !strings.EqualFold(m.VirtualKeyHash, virtualKeyDigest(plain)) {
		t.Fatalf("migrated hash=%q", m.VirtualKeyHash)
	}
}
