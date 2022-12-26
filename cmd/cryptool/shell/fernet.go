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

package shell

import (
	"fmt"

	"yunion.io/x/pkg/util/shellutils"
	"yunion.io/x/pkg/util/timeutils"

	"yunion.io/x/onecloud/pkg/keystone/tokens"
	"yunion.io/x/onecloud/pkg/util/fernetool"
)

func init() {
	type FernetInitKeysOptions struct {
		PATH  string `help:"path that stores fernet keys"`
		COUNT int    `help:"number of keys to init"`
	}
	shellutils.R(&FernetInitKeysOptions{}, "fernet-initkeys", "Initialze fernet keys", func(args *FernetInitKeysOptions) error {
		fm := fernetool.SFernetKeyManager{}
		err := fm.InitKeys(args.PATH, args.COUNT)
		if err != nil {
			return err
		}
		return nil
	})

	type FernetEncryptOptions struct {
		PATH string `help:"path that stores fernet keys"`
		MSG  string `help:"message to encrypt"`
	}
	shellutils.R(&FernetEncryptOptions{}, "fernet-encrypt", "Encrypt message with fernet keys", func(args *FernetEncryptOptions) error {
		fm := fernetool.SFernetKeyManager{}
		err := fm.LoadKeys(args.PATH)
		if err != nil {
			return err
		}
		ret, err := fm.Encrypt([]byte(args.MSG))
		if err != nil {
			return err
		}
		fmt.Println(string(ret))
		return nil
	})

	type FernetEncryptTokenOptions struct {
		PATH      string `help:"path that stores fernet keys"`
		USERID    string `help:"UserId"`
		METHOD    string `help:"auth method" choices:"password|token"`
		EXPIREAT  string `help:"expired time"`
		ProjectId string `help:"project Id"`
		DomainId  string `help:"domainId"`
		AUDITID   string `help:"audit ID"`
	}
	shellutils.R(&FernetEncryptTokenOptions{}, "fernet-encrypt-token", "Encrypt auth token with fernet keys", func(args *FernetEncryptTokenOptions) error {
		token := tokens.SAuthToken{}
		token.UserId = args.USERID
		token.Method = args.METHOD
		token.ProjectId = args.ProjectId
		token.DomainId = args.DomainId
		token.ExpiresAt, _ = timeutils.ParseFullIsoTime(args.EXPIREAT)
		token.AuditIds = []string{args.AUDITID}

		tk, err := token.Encode()
		if err != nil {
			return err
		}

		fmt.Println(len(tk))
		fmt.Println(string(tk))
		fmt.Println(tk)
		fmt.Printf("%x\n", tk)

		fm := fernetool.SFernetKeyManager{}
		err = fm.LoadKeys(args.PATH)
		if err != nil {
			return err
		}
		ret, err := fm.Encrypt(tk)
		if err != nil {
			return err
		}
		fmt.Println(string(ret))
		return nil
	})

	shellutils.R(&FernetEncryptOptions{}, "fernet-decrypt", "Decrypt message with fernet keys", func(args *FernetEncryptOptions) error {
		fm := fernetool.SFernetKeyManager{}
		if args.PATH == "empty" {
			fm.InitEmpty()
		} else {
			err := fm.LoadKeys(args.PATH)
			if err != nil {
				return err
			}
		}
		fmt.Println("primary key hash:", fm.PrimaryKeyHash())

		ret := fm.Decrypt([]byte(args.MSG))
		if len(ret) == 0 {
			return fmt.Errorf("invalid message")
		}
		fmt.Println(len(ret))
		fmt.Println(string(ret))
		fmt.Println(ret)
		fmt.Printf("%x\n", ret)
		token := tokens.SAuthToken{}
		err := token.Decode(ret)
		if err != nil {
			return err
		}
		fmt.Println(token)
		return nil
	})
}
