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
	"yunion.io/x/onecloud/pkg/multicloud/jdcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DBInstanceListOptions struct {
		PageNumber int `default:"1"`
		PageSize   int `default:"100"`
	}

	shellutils.R(&DBInstanceListOptions{}, "dbinstance-list", "List rds", func(cli *jdcloud.SRegion, args *DBInstanceListOptions) error {
		dbs, _, err := cli.GetDBInstances(args.PageNumber, args.PageSize)
		if err != nil {
			return err
		}
		printList(dbs, 0, 0, 0, nil)
		return nil
	})

	type DBInstanceIdOptions struct {
		ID string
	}

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-show", "Show rds", func(cli *jdcloud.SRegion, args *DBInstanceIdOptions) error {
		rds, err := cli.GetDBInstance(args.ID)
		if err != nil {
			return err
		}
		printObject(rds)
		return nil
	})

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-delete", "Delete rds", func(cli *jdcloud.SRegion, args *DBInstanceIdOptions) error {
		return cli.DeleteDBInstance(args.ID)
	})

	type DBInstanceAccountListOptions struct {
		PageNumber int `default:"1"`
		PageSize   int `default:"100"`
		ID         string
	}

	shellutils.R(&DBInstanceAccountListOptions{}, "dbinstance-account-list", "List rds account", func(cli *jdcloud.SRegion, args *DBInstanceAccountListOptions) error {
		accounts, _, err := cli.GetDBInstanceAccounts(args.ID, args.PageNumber, args.PageSize)
		if err != nil {
			return err
		}
		printList(accounts, 0, 0, 0, nil)
		return nil
	})

	type DBInstanceDatabaseListOptions struct {
		PageNumber int `default:"1"`
		PageSize   int `default:"100"`
		ID         string
	}

	shellutils.R(&DBInstanceDatabaseListOptions{}, "dbinstance-database-list", "List rds database", func(cli *jdcloud.SRegion, args *DBInstanceDatabaseListOptions) error {
		databases, _, err := cli.GetDBInstanceDatabases(args.ID, args.PageNumber, args.PageSize)
		if err != nil {
			return err
		}
		printList(databases, 0, 0, 0, nil)
		return nil
	})

	type DBInstanceBackupListOptions struct {
		PageNumber int `default:"1"`
		PageSize   int `default:"100"`
		ID         string
	}

	shellutils.R(&DBInstanceBackupListOptions{}, "dbinstance-backup-list", "List rds backup", func(cli *jdcloud.SRegion, args *DBInstanceBackupListOptions) error {
		backups, _, err := cli.GetDBInstanceBackups(args.ID, args.PageNumber, args.PageSize)
		if err != nil {
			return err
		}
		printList(backups, 0, 0, 0, nil)
		return nil
	})

}
