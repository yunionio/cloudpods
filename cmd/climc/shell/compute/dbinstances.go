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

package compute

import (
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.DBInstance) //.WithContextManager(&modules.Networks)
	cmd.List(&compute.DBInstanceListOptions{})
	cmd.Create(&compute.DBInstanceCreateOptions{})
	cmd.Update(&compute.DBInstanceUpdateOptions{})
	cmd.Show(&compute.DBInstanceIdOptions{})
	cmd.Delete(&compute.DBInstanceDeleteOptions{})
	cmd.Perform("renew", &compute.DBInstanceRenewOptions{})
	cmd.Perform("change-config", &compute.DBInstanceChangeConfigOptions{})
	cmd.Perform("public-connection", &compute.DBInstancePublicConnectionOptions{})
	cmd.Perform("recovery", &compute.DBInstanceRecoveryOptions{})
	cmd.Perform("reboot", &compute.DBInstanceIdOptions{})
	cmd.Perform("purge", &compute.DBInstanceIdOptions{})
	cmd.Perform("syncstatus", &compute.DBInstanceIdOptions{})
	cmd.Perform("sync", &compute.DBInstanceIdOptions{})
	cmd.Perform("change-owner", &compute.DBInstanceChangeOwnerOptions{})
	cmd.PerformWithKeyword("add-tag", "user-metadata", &options.ResourceMetadataOptions{})
	cmd.PerformWithKeyword("set-tag", "set-user-metadata", &options.ResourceMetadataOptions{})
	cmd.Perform("remote-update", &compute.DBInstanceRemoteUpdateOptions{})
	cmd.Perform("set-secgroup", &compute.DBInstanceSetSecgroupOptions{})

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

		var result *printutils.ListResult
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
