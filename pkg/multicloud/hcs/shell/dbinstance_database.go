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
		INSTANCE string `help:"DBInstance ID"`
	}

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-database-list", "Show dbinstance databases", func(cli *huawei.SRegion, args *DBInstanceIdOptions) error {
		databases, err := cli.GetDBInstanceDatabases(args.INSTANCE)
		if err != nil {
			return err
		}
		printList(databases, 0, 0, 0, nil)
		return nil
	})

	type DBInstanceDatabaseDeleteOptions struct {
		INSTANCE string
		DATABASE string
	}

	shellutils.R(&DBInstanceDatabaseDeleteOptions{}, "dbinstance-database-delete", "Delete dbinstance database", func(cli *huawei.SRegion, args *DBInstanceDatabaseDeleteOptions) error {
		return cli.DeleteDBInstanceDatabase(args.INSTANCE, args.DATABASE)
	})

	type DBInstanceDatabaseCreateOptions struct {
		INSTANCE      string
		DATABASE      string
		CHARACTER_SET string
	}

	shellutils.R(&DBInstanceDatabaseCreateOptions{}, "dbinstance-database-create", "Create dbinstance database", func(cli *huawei.SRegion, args *DBInstanceDatabaseCreateOptions) error {
		return cli.CreateDBInstanceDatabase(args.INSTANCE, args.DATABASE, args.CHARACTER_SET)
	})

}
