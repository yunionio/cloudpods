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
	"yunion.io/x/onecloud/pkg/multicloud/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DBInstanceListOptions struct {
	}
	shellutils.R(&DBInstanceListOptions{}, "dbinstance-list", "List dbinstances", func(cli *huawei.SRegion, args *DBInstanceListOptions) error {
		dbinstances, err := cli.GetDBInstances()
		if err != nil {
			return err
		}
		printList(dbinstances, 0, 0, 0, nil)
		return nil
	})

	type DBInstanceIdOptions struct {
		ID string `help:"DBInstance ID"`
	}

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-show", "Show dbinstance", func(cli *huawei.SRegion, args *DBInstanceIdOptions) error {
		dbinstance, err := cli.GetDBInstance(args.ID)
		if err != nil {
			return err
		}
		printObject(dbinstance)
		return nil
	})

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-parameter-list", "Show dbinstance parameters", func(cli *huawei.SRegion, args *DBInstanceIdOptions) error {
		parameters, err := cli.GetDBInstanceParameters(args.ID)
		if err != nil {
			return err
		}
		printList(parameters, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-database-list", "Show dbinstance databases", func(cli *huawei.SRegion, args *DBInstanceIdOptions) error {
		databases, err := cli.GetDBInstanceDatabases(args.ID)
		if err != nil {
			return err
		}
		printList(databases, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-account-list", "Show dbinstance accounts", func(cli *huawei.SRegion, args *DBInstanceIdOptions) error {
		accounts, err := cli.GetDBInstanceAccounts(args.ID)
		if err != nil {
			return err
		}
		printList(accounts, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-backup-list", "Show dbinstance backups", func(cli *huawei.SRegion, args *DBInstanceIdOptions) error {
		backups, err := cli.GetDBInstanceBackups(args.ID)
		if err != nil {
			return err
		}
		printList(backups, 0, 0, 0, nil)
		return nil
	})

}
