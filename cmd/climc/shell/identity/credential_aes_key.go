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

package identity

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

func init() {
	type CredentialAesKeyOptions struct {
		User       string `help:"User"`
		UserDomain string `help:"domain of user"`
	}

	type CredentialCreateAesKeyOptions struct {
		CredentialAesKeyOptions
		NAME string `help:"name of key"`
		ALG  string `help:"name of alg" choices:"aes-256|sm4"`
	}
	R(&CredentialCreateAesKeyOptions{}, "credential-create-enckey", "Create AES 256 Secret Key", func(s *mcclient.ClientSession, args *CredentialCreateAesKeyOptions) error {
		var uid string
		var err error
		if len(args.User) > 0 {
			uid, err = modules.UsersV3.FetchId(s, args.User, args.UserDomain)
			if err != nil {
				return err
			}
		}
		secret, err := modules.Credentials.CreateEncryptKey(s, uid, args.NAME, args.ALG)
		if err != nil {
			return err
		}
		printObject(secret.Marshal())
		return nil
	})

	R(&CredentialAesKeyOptions{}, "credential-get-enckey", "Get encryption secret keys for user", func(s *mcclient.ClientSession, args *CredentialAesKeyOptions) error {
		var uid string
		var err error
		if len(args.User) > 0 {
			uid, err = modules.UsersV3.FetchId(s, args.User, args.UserDomain)
			if err != nil {
				return err
			}
		}
		secrets, err := modules.Credentials.GetEncryptKeys(s, uid)
		if err != nil {
			return err
		}
		result := printutils.ListResult{}
		result.Data = make([]jsonutils.JSONObject, len(secrets))
		for i := range secrets {
			result.Data[i] = secrets[i].Marshal()
		}
		printList(&result, nil)
		return nil
	})

	type CredentialAesKeyEncryptOptions struct {
		ID     string `help:"id or name of credential"`
		SECRET string `help:"secret to encrypt"`
	}
	R(&CredentialAesKeyEncryptOptions{}, "credential-enckey-encrypt", "Encrypt with a encryption key", func(s *mcclient.ClientSession, args *CredentialAesKeyEncryptOptions) error {
		sec, err := modules.Credentials.EncryptKeyEncryptBase64(s, args.ID, []byte(args.SECRET))
		if err != nil {
			return err
		}
		fmt.Println(sec)
		return nil
	})

	type CredentialAesKeyDecryptOptions struct {
		ID     string `help:"id or name of credential"`
		SECRET string `help:"secret to decrypt"`
	}
	R(&CredentialAesKeyDecryptOptions{}, "credential-enckey-decrypt", "Decrypt with a encryption key", func(s *mcclient.ClientSession, args *CredentialAesKeyDecryptOptions) error {
		sec, err := modules.Credentials.EncryptKeyDecryptBase64(s, args.ID, args.SECRET)
		if err != nil {
			return err
		}
		fmt.Println(string(sec))
		return nil
	})
}
