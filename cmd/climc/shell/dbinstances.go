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
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
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

	type DBInstanceCreateOptions struct {
		NAME          string `help:"DBInstance Name"`
		InstanceType  string `help:"InstanceType for DBInstance"`
		VcpuCount     int    `help:"Core of cpu for DBInstance"`
		VmemSizeMb    int    `help:"Memory size of DBInstance"`
		Port          int    `help:"Port of DBInstance"`
		Category      string `help:"Category of DBInstance"`
		Network       string `help:"Network of DBInstance"`
		Address       string `help:"Address of DBInstance"`
		Engine        string `help:"Engine of DBInstance"`
		EngineVersion string `help:"EngineVersion of DBInstance Engine"`
		StorageType   string `help:"StorageTyep of DBInstance"`
		Secgroup      string `help:"Secgroup name or Id for DBInstance"`
		Zone          string `help:"ZoneId or name for DBInstance"`
		DiskSizeGB    int    `help:"Storage size for DBInstance"`
		Duration      string `help:"Duration for DBInstance"`
		AllowDelete   *bool  `help:"not lock dbinstance" `
	}

	R(&DBInstanceCreateOptions{}, "dbinstance-create", "Create DB instance", func(s *mcclient.ClientSession, opts *DBInstanceCreateOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		if opts.AllowDelete != nil && *opts.AllowDelete {
			params.Add(jsonutils.JSONFalse, "disable_delete")
		}
		result, err := modules.DBInstance.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type DBInstanceRenewOptions struct {
		ID       string `help:"ID or name of server to renew"`
		DURATION string `help:"Duration of renew, ADMIN only command"`
	}
	R(&DBInstanceRenewOptions{}, "dbinstance-renew", "Renew a dbinstance", func(s *mcclient.ClientSession, args *DBInstanceRenewOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.DURATION), "duration")
		result, err := modules.DBInstance.PerformAction(s, args.ID, "renew", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type DBInstanceUpdateOptions struct {
		ID          string `help:"ID or name of server to renew"`
		Name        string
		Description string
		Delete      string `help:"Lock or not lock dbinstance" choices:"enable|disable"`
	}
	R(&DBInstanceUpdateOptions{}, "dbinstance-update", "Update a dbinstance", func(s *mcclient.ClientSession, args *DBInstanceUpdateOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		if len(args.Delete) > 0 {
			if args.Delete == "disable" {
				params.Add(jsonutils.JSONTrue, "disable_delete")
			} else {
				params.Add(jsonutils.JSONFalse, "disable_delete")
			}
		}
		result, err := modules.DBInstance.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type DBInstanceChangeConfigOptions struct {
		ID           string `help:"ID or name of server to renew"`
		DiskSizeGb   int64  `help:"Change DBInstance storage size"`
		VcpuCount    int64  `help:"Change DBInstance vcpu count"`
		VmemSizeMb   int64  `help:"Change DBInstance vmem size mb"`
		InstanceType string `help:"Change DBInstance instanceType"`
		Category     string `help:"Change DBInstance category"`
	}
	R(&DBInstanceChangeConfigOptions{}, "dbinstance-change-config", "ChangeConfig a dbinstance", func(s *mcclient.ClientSession, args *DBInstanceChangeConfigOptions) error {
		params := jsonutils.NewDict()
		if len(args.Category) > 0 {
			params.Add(jsonutils.NewString(args.Category), "category")
		}
		if len(args.InstanceType) > 0 {
			params.Add(jsonutils.NewString(args.InstanceType), "instance_type")
		}
		if args.DiskSizeGb > 0 {
			params.Add(jsonutils.NewInt(args.DiskSizeGb), "disk_size_gb")
		}
		if args.VcpuCount > 0 {
			params.Add(jsonutils.NewInt(args.VcpuCount), "vcpu_count")
		}
		if args.VmemSizeMb > 0 {
			params.Add(jsonutils.NewInt(args.VmemSizeMb), "vmeme_size_mb")
		}
		result, err := modules.DBInstance.PerformAction(s, args.ID, "change-config", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type DBInstanceIdOptions struct {
		ID string `help:"ID of dbinstance"`
	}

	R(&DBInstanceIdOptions{}, "dbinstance-open-public-connection", "Open DB instance public connection", func(s *mcclient.ClientSession, opts *DBInstanceIdOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.JSONTrue, "open")
		result, err := modules.DBInstance.PerformAction(s, opts.ID, "public-connection", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&DBInstanceIdOptions{}, "dbinstance-close-public-connection", "Close DB instance public connection", func(s *mcclient.ClientSession, opts *DBInstanceIdOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.JSONFalse, "open")
		result, err := modules.DBInstance.PerformAction(s, opts.ID, "public-connection", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type DBInstanceRecoveryOptions struct {
		ID        string
		BACKUP    string
		Databases []string
	}

	R(&DBInstanceRecoveryOptions{}, "dbinstance-recovery", "Recovery DB instance database from backup", func(s *mcclient.ClientSession, opts *DBInstanceRecoveryOptions) error {
		params := jsonutils.NewDict()
		params.Set("dbinstancebackup", jsonutils.NewString(opts.BACKUP))
		dbs := jsonutils.NewDict()
		for _, database := range opts.Databases {
			if len(database) > 0 {
				dbInfo := strings.Split(database, ":")
				if len(dbInfo) == 1 {
					dbs.Add(jsonutils.NewString(dbInfo[0]), dbInfo[0])
				} else if len(dbInfo) == 2 {
					dbs.Add(jsonutils.NewString(dbInfo[1]), dbInfo[0])
				} else {
					return fmt.Errorf("Invalid dbinfo: %s", database)
				}
			}
		}
		if dbs.Length() > 0 {
			params.Add(dbs, "databases")
		}
		result, err := modules.DBInstance.PerformAction(s, opts.ID, "recovery", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&DBInstanceIdOptions{}, "dbinstance-show", "Show DB instance", func(s *mcclient.ClientSession, opts *DBInstanceIdOptions) error {
		result, err := modules.DBInstance.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&DBInstanceIdOptions{}, "dbinstance-reboot", "Reboot DB instance", func(s *mcclient.ClientSession, opts *DBInstanceIdOptions) error {
		result, err := modules.DBInstance.PerformAction(s, opts.ID, "reboot", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&DBInstanceIdOptions{}, "dbinstance-delete", "Delete DB instance", func(s *mcclient.ClientSession, opts *DBInstanceIdOptions) error {
		result, err := modules.DBInstance.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&DBInstanceIdOptions{}, "dbinstance-purge", "Purge DB instance", func(s *mcclient.ClientSession, opts *DBInstanceIdOptions) error {
		result, err := modules.DBInstance.PerformAction(s, opts.ID, "purge", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&DBInstanceIdOptions{}, "dbinstance-sync-status", "Sync status for DB instance", func(s *mcclient.ClientSession, opts *DBInstanceIdOptions) error {
		result, err := modules.DBInstance.PerformAction(s, opts.ID, "sync-status", nil)
		if err != nil {
			return err
		}
		printObject(result)
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
