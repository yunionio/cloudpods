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
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type DBInstanceListOptions struct {
		options.BaseListOptions
		BillingType string `help:"billing type" choices:"postpaid|prepaid"`
	}
	R(&DBInstanceListOptions{}, "dbinstance-list", "List DB instance", func(s *mcclient.ClientSession, opts *DBInstanceListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.DBInstance.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.DBInstance.GetColumns(s))
		return nil
	})

	type DBInstanceNetworkListOptions struct {
		options.BaseListOptions
		DBInstance string `help:"ID or Name of DBInstance" json:"dbinstance"`
		Network    string `help:"Network ID or name"`
	}
	R(&DBInstanceNetworkListOptions{}, "dbinstance-network-list", "List DB instance networks", func(s *mcclient.ClientSession, opts *DBInstanceNetworkListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		var result *modulebase.ListResult
		if len(opts.DBInstance) > 0 {
			result, err = modules.DBInstanceNetworks.ListDescendent(s, opts.DBInstance, params)
		} else if len(opts.Network) > 0 {
			result, err = modules.DBInstanceNetworks.ListDescendent2(s, opts.Network, params)
		} else {
			result, err = modules.DBInstanceNetworks.List(s, params)
		}

		if err != nil {
			return err
		}
		printList(result, modules.DBInstanceNetworks.GetColumns(s))
		return nil
	})

	type DBInstanceParameterListOptions struct {
		options.BaseListOptions
		DBInstance string `help:"ID or Name of DBInstance" json:"dbinstance"`
	}
	R(&DBInstanceParameterListOptions{}, "dbinstance-parameter-list", "List DB instance parameters", func(s *mcclient.ClientSession, opts *DBInstanceParameterListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.DBInstanceParameters.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.DBInstanceParameters.GetColumns(s))
		return nil
	})

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

	type DBInstancePrivilegeListOptions struct {
		options.BaseListOptions
		DBInstanceaccount  string `help:"ID or Name of DBInstanceaccount" json:"dbinstanceaccount"`
		DBInstancedatabase string `help:"ID or Name of DBInstancedatabase" json:"dbinstancedatabase"`
	}
	R(&DBInstancePrivilegeListOptions{}, "dbinstance-privilege-list", "List DB instance accounts", func(s *mcclient.ClientSession, opts *DBInstancePrivilegeListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.DBInstancePrivileges.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.DBInstancePrivileges.GetColumns(s))
		return nil
	})

}
