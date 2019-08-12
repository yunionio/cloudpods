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

	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DBInstanceListOptions struct {
		Id     []string `help:"IDs of instances to show"`
		Limit  int      `help:"page size"`
		Offset int      `help:"page offset"`
	}
	shellutils.R(&DBInstanceListOptions{}, "dbinstance-list", "List dbintances", func(cli *aliyun.SRegion, args *DBInstanceListOptions) error {
		instances, total, e := cli.GetDBInstances(args.Id, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(instances, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type DBInstanceIdOptions struct {
		ID string `help:"ID of instances to show"`
	}
	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-show", "Show dbintance", func(cli *aliyun.SRegion, args *DBInstanceIdOptions) error {
		instance, err := cli.GetDBInstanceDetail(args.ID)
		if err != nil {
			return err
		}
		printObject(instance)
		return nil
	})

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-open-public-connection", "Open dbintance public connection", func(cli *aliyun.SRegion, args *DBInstanceIdOptions) error {
		return cli.OpenPublicConnection(args.ID)
	})

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-close-public-connection", "Close dbintance public connection", func(cli *aliyun.SRegion, args *DBInstanceIdOptions) error {
		return cli.ClosePublicConnection(args.ID)
	})

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-delete", "Delete dbintance", func(cli *aliyun.SRegion, args *DBInstanceIdOptions) error {
		return cli.DeleteDBInstance(args.ID)
	})

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-restart", "Restart dbintance", func(cli *aliyun.SRegion, args *DBInstanceIdOptions) error {
		return cli.RebootDBInstance(args.ID)
	})

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-network-list", "Show dbintance network info", func(cli *aliyun.SRegion, args *DBInstanceIdOptions) error {
		networks, err := cli.GetDBInstanceNetInfo(args.ID)
		if err != nil {
			return err
		}
		printList(networks, 0, 0, 0, []string{})
		return nil
	})

	type DBInstanceRecoveryOptions struct {
		ID        string
		BACKUP    string
		Databases []string
	}

	shellutils.R(&DBInstanceRecoveryOptions{}, "dbinstance-recovery", "Recovery dbintance from backup", func(cli *aliyun.SRegion, args *DBInstanceRecoveryOptions) error {
		databases := map[string]string{}
		for _, database := range args.Databases {
			if len(database) > 0 {
				dbInfo := strings.Split(database, ":")
				if len(dbInfo) == 1 {
					databases[dbInfo[0]] = dbInfo[0]

				} else if len(dbInfo) == 2 {
					databases[dbInfo[0]] = dbInfo[1]
				} else {
					return fmt.Errorf("Invalid dbinfo: %s", database)
				}
			}
		}
		return cli.RecoveryDBInstanceFromBackup(args.ID, args.BACKUP, databases)
	})

}
