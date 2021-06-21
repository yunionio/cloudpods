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
	"yunion.io/x/onecloud/pkg/multicloud/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DBInstanceListOptions struct {
		TYPE string `help:"dbinstance type" choices:"Microsoft.DBForMariaDB/servers|Microsoft.DBforMySQL/servers|Microsoft.DBforMySQL/flexibleServers|Microsoft.DBforPostgreSQL/servers|Microsoft.DBforPostgreSQL/flexibleServers|Microsoft.Sql/servers|Microsoft.Sql/managedInstances"`
	}
	shellutils.R(&DBInstanceListOptions{}, "dbinstance-list", "List rds intances", func(cli *azure.SRegion, args *DBInstanceListOptions) error {
		instances, err := cli.ListDBInstance(args.TYPE)
		if err != nil {
			return err
		}
		printList(instances, 0, 0, len(instances), []string{})
		return nil
	})

	type DBInstanceIdOptions struct {
		ID string
	}

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-show", "Show rds intance", func(cli *azure.SRegion, args *DBInstanceIdOptions) error {
		instance, err := cli.GetDBInstanceById(args.ID)
		if err != nil {
			return err
		}
		printObject(instance)
		return nil
	})

}
