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
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type DBInstanceBackupListOptions struct {
		options.BaseListOptions
		DBInstance  string `help:"ID or Name of DBInstance" json:"dbinstance"`
		Cloudregion string `help:"ID or Name of cloudregion"`
	}
	R(&DBInstanceBackupListOptions{}, "dbinstance-backup-list", "List DB instance backups", func(s *mcclient.ClientSession, opts *DBInstanceBackupListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.DBInstanceBackups.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.DBInstanceBackups.GetColumns(s))
		return nil
	})

	type DBInstanceBackupCreateOptions struct {
		INSTANCE    string `help:"ID or Name of DBInstance" json:"dbinstance"`
		NAME        string
		Databases   []string
		Description string
	}

	R(&DBInstanceBackupCreateOptions{}, "dbinstance-backup-create", "Create DB instance backup", func(s *mcclient.ClientSession, opts *DBInstanceBackupCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(opts.INSTANCE), "dbinstance")
		params.Add(jsonutils.NewString(opts.NAME), "name")
		params.Add(jsonutils.NewString(opts.Description), "description")
		databases := jsonutils.NewArray()
		for _, database := range opts.Databases {
			databases.Add(jsonutils.NewString(database))
		}
		params.Add(databases, "databases")
		result, err := modules.DBInstanceBackups.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type DBInstanceBackupIdOptions struct {
		ID string
	}

	R(&DBInstanceBackupIdOptions{}, "dbinstance-backup-show", "Show DB instance backup", func(s *mcclient.ClientSession, opts *DBInstanceBackupIdOptions) error {
		result, err := modules.DBInstanceBackups.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&DBInstanceBackupIdOptions{}, "dbinstance-backup-delete", "Delete DB instance backup", func(s *mcclient.ClientSession, opts *DBInstanceBackupIdOptions) error {
		result, err := modules.DBInstanceBackups.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
