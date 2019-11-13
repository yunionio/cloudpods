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
	type DBInstanceAccountListOptions struct {
		options.BaseListOptions
		DBInstance string `help:"ID or Name of DBInstance" json:"dbinstance"`
	}
	R(&DBInstanceAccountListOptions{}, "dbinstance-account-list", "List DB instance accounts", func(s *mcclient.ClientSession, opts *DBInstanceAccountListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.DBInstanceAccounts.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.DBInstanceAccounts.GetColumns(s))
		return nil
	})

	type DBInstanceAccountIdOptions struct {
		ID string `help:"ID or Name of DBInstanceaccount"`
	}

	R(&DBInstanceAccountIdOptions{}, "dbinstance-account-show", "Show DB instance account", func(s *mcclient.ClientSession, opts *DBInstanceAccountIdOptions) error {
		account, err := modules.DBInstanceAccounts.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(account)
		return nil
	})

	R(&DBInstanceAccountIdOptions{}, "dbinstance-account-delete", "Delete DB instance account", func(s *mcclient.ClientSession, opts *DBInstanceAccountIdOptions) error {
		account, err := modules.DBInstanceAccounts.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(account)
		return nil
	})

}
