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
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {

	type DBInstanceIdExtraOptions struct {
		ID     string `help:"ID of instances to show"`
		Limit  int    `help:"page size"`
		Offset int    `help:"page offset"`
	}

	shellutils.R(&DBInstanceIdExtraOptions{}, "dbinstance-account-list", "List dbintance accounts", func(cli *aliyun.SRegion, args *DBInstanceIdExtraOptions) error {
		accounts, _, err := cli.GetDBInstanceAccounts(args.ID, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(accounts, 0, 0, 0, []string{})
		return nil
	})

	type DBInstanceAccountCreateOptions struct {
		INSTANCE string `help:"ID of instances"`
		NAME     string `help:"account name"`
		PASSWORD string `help:"account password"`
		Desc     string
	}

	shellutils.R(&DBInstanceAccountCreateOptions{}, "dbinstance-account-create", "Create dbintance account", func(cli *aliyun.SRegion, args *DBInstanceAccountCreateOptions) error {
		return cli.CreateDBInstanceAccount(args.INSTANCE, args.NAME, args.PASSWORD, args.Desc)
	})

	type DBInstanceAccountDeleteOptions struct {
		INSTANCE string
		NAME     string
	}

	shellutils.R(&DBInstanceAccountDeleteOptions{}, "dbinstance-account-delete", "Delete dbintance account", func(cli *aliyun.SRegion, args *DBInstanceAccountDeleteOptions) error {
		return cli.DeleteDBInstanceAccount(args.INSTANCE, args.NAME)
	})

	type DBInstanceAccountResetOptions struct {
		INSTANCE    string `help:"ID of instances"`
		NAME        string `help:"account name"`
		PASSWORD    string `help:"account password"`
		AccountType string `help:"account type" choices:"Normal|Super" default:"Normal"`
	}

	shellutils.R(&DBInstanceAccountResetOptions{}, "dbinstance-account-reset-password", "Reset dbintance account password", func(cli *aliyun.SRegion, args *DBInstanceAccountResetOptions) error {
		return cli.ResetDBInstanceAccountPassword(args.INSTANCE, args.NAME, args.PASSWORD, args.AccountType)
	})

}
