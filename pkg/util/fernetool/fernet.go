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

package fernetool

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/fernet/fernet-go"
	"github.com/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

type SFernetKeyManager struct {
	keys []*fernet.Key
}

func (m *SFernetKeyManager) InitEmpty() error {
	src := [32]byte{}
	dst := base64.URLEncoding.EncodeToString(src[:])
	emptyKey, err := fernet.DecodeKey(dst)
	if err != nil {
		return errors.WithMessage(err, "InitEmpty with fernet.DecodeKey")
	}
	m.keys = []*fernet.Key{emptyKey}
	return nil
}

func (m *SFernetKeyManager) PrimaryKeyHash() string {
	if len(m.keys) == 0 {
		return ""
	}
	dst := m.keys[0].Encode()
	sum := sha1.Sum([]byte(dst))
	return hex.EncodeToString(sum[:])
}

func (m *SFernetKeyManager) SetKeys(keys []*fernet.Key) {
	m.keys = keys
}

func (m *SFernetKeyManager) LoadKeys(path string) error {
	filesInfos, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	keyStrs := make([]string, 0)
	for i := range filesInfos {
		fn := filepath.Join(path, filesInfos[i].Name())
		keyBytes, err := ioutil.ReadFile(fn)
		if err != nil {
			return err
		}
		keyStrs = append(keyStrs, string(keyBytes))
	}
	if len(keyStrs) == 0 {
		return fmt.Errorf("empty fernet keys")
	}
	keys, err := fernet.DecodeKeys(keyStrs...)
	if err != nil {
		return err
	}
	m.keys = keys
	return nil
}

func (m *SFernetKeyManager) Decrypt(tok []byte, ttl time.Duration) []byte {
	modReturned := len(tok) % 4
	if modReturned > 0 {
		for i := 0; i < 4-modReturned; i += 1 {
			tok = append(tok, '=')
		}
	}
	return fernet.VerifyAndDecrypt(tok, ttl, m.keys)
}

func (m *SFernetKeyManager) InitKeys(path string, cnt int) error {
	m.keys = make([]*fernet.Key, cnt)
	for i := 0; i < cnt; i += 1 {
		key := fernet.Key{}
		err := key.Generate()
		if err != nil {
			return err
		}
		if len(path) > 0 && fileutils2.IsDir(path) {
			fn := filepath.Join(path, fmt.Sprintf("%d", i))
			err = fileutils2.FileSetContents(fn, key.Encode())
			if err != nil {
				return err
			}
		}
		m.keys[i] = &key
	}
	return nil
}

func (m *SFernetKeyManager) Encrypt(msg []byte) ([]byte, error) {
	hash := 0
	for i := 0; i < len(msg) && i < 10; i += 1 {
		hash += int(msg[i])
	}
	idx := hash % len(m.keys)
	tok, err := fernet.EncryptAndSign([]byte(msg), m.keys[idx])
	if err != nil {
		return nil, err
	}
	endIdx := len(tok)
	for endIdx >= 0 && tok[endIdx-1] == '=' {
		endIdx -= 1
	}
	return tok[:endIdx], nil
}
