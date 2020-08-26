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
	"yunion.io/x/onecloud/pkg/multicloud/google"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DBInstanceDatabaseListOptions struct {
		RDS string
	}
	shellutils.R(&DBInstanceDatabaseListOptions{}, "dbinstance-database-list", "List dbinstance database", func(cli *google.SRegion, args *DBInstanceDatabaseListOptions) error {
		databases, err := cli.GetDBInstanceDatabases(args.RDS)
		if err != nil {
			return err
		}
		printList(databases, 0, 0, 0, nil)
		return nil
	})

	type DBInstanceDatabaseCreateOptions struct {
		RDS     string
		NAME    string
		CHARSET string
	}

	shellutils.R(&DBInstanceDatabaseCreateOptions{}, "dbinstance-database-create", "Create dbinstance database", func(cli *google.SRegion, args *DBInstanceDatabaseCreateOptions) error {
		return cli.CreateDatabase(args.RDS, args.NAME, args.CHARSET)
	})

}
