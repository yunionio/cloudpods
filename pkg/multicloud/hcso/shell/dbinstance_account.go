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
	huawei "yunion.io/x/onecloud/pkg/multicloud/hcso"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DBInstanceIdOptions struct {
		ID string `help:"DBInstance ID"`
	}

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-account-list", "Show dbinstance accounts", func(cli *huawei.SRegion, args *DBInstanceIdOptions) error {
		accounts, err := cli.GetDBInstanceAccounts(args.ID)
		if err != nil {
			return err
		}
		printList(accounts, 0, 0, 0, nil)
		return nil
	})

	type DBInstanceAccountDeleteOptions struct {
		INSTANCE string
		ACCOUNT  string
	}

	shellutils.R(&DBInstanceAccountDeleteOptions{}, "dbinstance-account-delete", "Delete dbinstance account", func(cli *huawei.SRegion, args *DBInstanceAccountDeleteOptions) error {
		return cli.DeleteDBInstanceAccount(args.INSTANCE, args.ACCOUNT)
	})

	type DBInstanceAccountCreateOptions struct {
		INSTANCE string
		ACCOUNT  string
		PASSWORD string
	}

	shellutils.R(&DBInstanceAccountCreateOptions{}, "dbinstance-account-create", "Create dbinstance account", func(cli *huawei.SRegion, args *DBInstanceAccountCreateOptions) error {
		return cli.CreateDBInstanceAccount(args.INSTANCE, args.ACCOUNT, args.PASSWORD)
	})

	type DBInstanceAccountRevokePrivilegeOptions struct {
		INSTANCE string
		ACCOUNT  string
		DATABASE string
	}

	shellutils.R(&DBInstanceAccountRevokePrivilegeOptions{}, "dbinstance-account-revoke-provilege", "Revoke dbinstance account privilege", func(cli *huawei.SRegion, args *DBInstanceAccountRevokePrivilegeOptions) error {
		return cli.RevokeDBInstancePrivilege(args.INSTANCE, args.ACCOUNT, args.DATABASE)
	})

	type DBInstanceAccountGrantPrivilegeOptions struct {
		INSTANCE  string
		ACCOUNT   string
		DATABASE  string
		PRIVILEGE string `help:"database privilege" choices:"r|rw"`
	}

	shellutils.R(&DBInstanceAccountGrantPrivilegeOptions{}, "dbinstance-account-grant-provilege", "Grant dbinstance account privilege", func(cli *huawei.SRegion, args *DBInstanceAccountGrantPrivilegeOptions) error {
		return cli.GrantDBInstancePrivilege(args.INSTANCE, args.ACCOUNT, args.DATABASE, args.PRIVILEGE)
	})

}
