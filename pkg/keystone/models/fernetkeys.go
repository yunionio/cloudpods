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
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/fernet/fernet-go"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/keystone/keys"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

// +onecloud:swagger-gen-ignore
type SFernetKeyManager struct {
	db.SModelBaseManager
}

var (
	FernetKeyManager *SFernetKeyManager
)

func init() {
	FernetKeyManager = &SFernetKeyManager{
		SModelBaseManager: db.NewModelBaseManager(
			SFernetKey{},
			"fernetkey",
			"fernetkey",
			"fernetkeys",
		),
	}
	FernetKeyManager.SetVirtualObject(FernetKeyManager)
}

type SFernetKey struct {
	db.SModelBase

	Type  string `width:"36" charset:"ascii" nullable:"false" primary:"true"`
	Index int    `nullable:"false" primary:"true"`
	Key   string `width:"64" charset:"ascii" nullable:"false"`
}

func (manager *SFernetKeyManager) InitializeData() error {
	fkeys, err := manager.getKeys(api.FernetKeyForToken)
	if err != nil {
		return errors.Wrap(err, "manager.getKeys")
	}
	if len(fkeys) == 0 {
		fkeys, err = manager.setupKeys(api.FernetKeyForToken, options.Options.FernetKeyRepository)
		if err != nil {
			return errors.Wrap(err, "manager.setupKeys")
		}
	}
	keys.TokenKeysManager.SetKeys(fkeys)

	if options.Options.SetupCredentialKeys {
		fkeys, err := manager.getKeys(api.FernetKeyForToken)
		if err != nil {
			return errors.Wrap(err, "manager.getKeys")
		}
		if len(fkeys) == 0 {
			fkeys, err = manager.setupKeys(api.FernetKeyForCredential, "")
			if err != nil {
				return errors.Wrap(err, "manager.setupKeys")
			}
		}
		keys.CredentialKeyManager.SetKeys(fkeys)
	} else {
		err = keys.CredentialKeyManager.InitEmpty()
		if err != nil {
			return errors.Wrap(err, "keys.TokenKeysManager.InitEmpty")
		}
	}
	return nil
}

func (manager *SFernetKeyManager) getKeys(keyType string) ([]*fernet.Key, error) {
	q := manager.Query().Equals("type", keyType).Asc("index")
	keys := make([]SFernetKey, 0)
	err := db.FetchModelObjects(manager, q, &keys)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	fkeys := make([]*fernet.Key, len(keys))
	for i := range keys {
		fkeys[i], err = fernet.DecodeKey(keys[i].Key)
		if err != nil {
			return nil, errors.Wrap(err, "fernet.DecodeKey")
		}
	}
	return fkeys, nil
}

func (manager *SFernetKeyManager) setupKeys(keyType string, repoDir string) ([]*fernet.Key, error) {
	maxKeyCount := 2
	ret := make([]*fernet.Key, maxKeyCount)
	for i := 0; i < maxKeyCount; i += 1 {
		var fkey *fernet.Key
		if len(repoDir) > 0 {
			keyPath := filepath.Join(repoDir, fmt.Sprintf("%d", i))
			if fileutils2.Exists(keyPath) {
				keyCrypt, err := fileutils2.FileGetContents(keyPath)
				if err != nil {
					return nil, errors.Wrap(err, "fileutils.FileGetContent")
				}
				fkey, err = fernet.DecodeKey(keyCrypt)
				if err != nil {
					return nil, errors.Wrap(err, "fernet.DecodeKey")
				}
			}
		}
		if fkey == nil {
			fkey = &fernet.Key{}
			err := fkey.Generate()
			if err != nil {
				return nil, errors.Wrap(err, "fkey.Generate")
			}
		}
		key := SFernetKey{
			Type:  keyType,
			Index: i,
			Key:   fkey.Encode(),
		}
		err := manager.TableSpec().Insert(&key)
		if err != nil {
			return nil, errors.Wrap(err, "insertFernetKeys")
		}
		ret[i] = fkey
	}
	return ret, nil
}
