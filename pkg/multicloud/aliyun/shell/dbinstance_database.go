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

	shellutils.R(&DBInstanceIdExtraOptions{}, "dbinstance-database-list", "List dbintance databases", func(cli *aliyun.SRegion, args *DBInstanceIdExtraOptions) error {
		databases, _, err := cli.GetDBInstanceDatabases(args.ID, "", args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(databases, 0, 0, 0, []string{})
		return nil
	})

	type DBInstanceDatabaseCreateOptions struct {
		INSTANCE     string `help:"ID of instances"`
		NAME         string `help:"database name"`
		CHARACTERSET string `help:"character set for database"`
		Desc         string
	}

	shellutils.R(&DBInstanceDatabaseCreateOptions{}, "dbinstance-database-create", "Create dbintance database", func(cli *aliyun.SRegion, args *DBInstanceDatabaseCreateOptions) error {
		return cli.CreateDBInstanceDatabae(args.INSTANCE, args.CHARACTERSET, args.NAME, args.Desc)
	})

	type DBInstanceDatabaseDeleteOptions struct {
		INSTANCE string
		NAME     string
	}

	shellutils.R(&DBInstanceDatabaseDeleteOptions{}, "dbinstance-database-delete", "Delete dbintance database", func(cli *aliyun.SRegion, args *DBInstanceDatabaseDeleteOptions) error {
		return cli.DeleteDBInstanceDatabase(args.INSTANCE, args.NAME)
	})

}
