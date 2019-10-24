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
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type DBInstanceDatabaseListOptions struct {
		options.BaseListOptions
		DBInstance string `help:"ID or Name of DBInstance" json:"dbinstance"`
	}
	R(&DBInstanceDatabaseListOptions{}, "dbinstance-database-list", "List DB instance databases", func(s *mcclient.ClientSession, opts *DBInstanceDatabaseListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.DBInstanceDatabases.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.DBInstanceDatabases.GetColumns(s))
		return nil
	})

	type DBInstanceDatabaseCreateOptions struct {
		NAME         string
		DBINSTANCE   string `help:"ID or Name of DBInstance" json:"dbinstance"`
		CharacterSet string `help:"CharacterSet for database"`
	}

	R(&DBInstanceDatabaseCreateOptions{}, "dbinstance-database-create", "Create DB instance databases", func(s *mcclient.ClientSession, opts *DBInstanceDatabaseCreateOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.DBInstanceDatabases.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type DBInstanceDatabaseIdOptions struct {
		ID string
	}

	R(&DBInstanceDatabaseIdOptions{}, "dbinstance-database-delete", "Delete DB instance databases", func(s *mcclient.ClientSession, opts *DBInstanceDatabaseIdOptions) error {
		result, err := modules.DBInstanceDatabases.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
