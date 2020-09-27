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

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type MysqlSkuListOptions struct {
	}
	shellutils.R(&MysqlSkuListOptions{}, "mysql-sku-list", "List mysql instance types", func(cli *qcloud.SRegion, args *MysqlSkuListOptions) error {
		skus, err := cli.ListMysqlSkus()
		if err != nil {
			return errors.Wrapf(err, "ListMysqlSkus")
		}
		printList(skus, 0, 0, 0, nil)
		return nil
	})

	type MysqlInstanceListOptions struct {
		Ids    []string
		Offset int
		Limit  int
	}

	shellutils.R(&MysqlInstanceListOptions{}, "mysql-instance-list", "List mysql instance", func(cli *qcloud.SRegion, args *MysqlInstanceListOptions) error {
		result, _, err := cli.ListMySQLInstances(args.Ids, args.Offset, args.Limit)
		if err != nil {
			return errors.Wrapf(err, "ListMySQLInstances")
		}
		printList(result, 0, 0, 0, nil)
		return nil
	})

	type MySQLInstanceIdOptions struct {
		ID string
	}

	shellutils.R(&MySQLInstanceIdOptions{}, "mysql-instance-reboot", "Reboot mysql instance", func(cli *qcloud.SRegion, args *MySQLInstanceIdOptions) error {
		return cli.RebootMySQLInstance(args.ID)
	})

	shellutils.R(&MySQLInstanceIdOptions{}, "mysql-instance-isolate", "Isolate mysql instance", func(cli *qcloud.SRegion, args *MySQLInstanceIdOptions) error {
		return cli.IsolateMySQLDBInstance(args.ID)
	})

	shellutils.R(&MySQLInstanceIdOptions{}, "mysql-instance-offline-isolate", "Offline Isolate mysql instance", func(cli *qcloud.SRegion, args *MySQLInstanceIdOptions) error {
		return cli.OfflineIsolatedMySQLInstances([]string{args.ID})
	})

	shellutils.R(&MySQLInstanceIdOptions{}, "mysql-instance-release-isolate", "Release Isolate mysql instance", func(cli *qcloud.SRegion, args *MySQLInstanceIdOptions) error {
		return cli.ReleaseIsolatedMySQLDBInstances([]string{args.ID})
	})

	shellutils.R(&MySQLInstanceIdOptions{}, "mysql-instance-secgroup-list", "List mysql instance secgroups", func(cli *qcloud.SRegion, args *MySQLInstanceIdOptions) error {
		secgroups, err := cli.DescribeMySQLDBSecurityGroups(args.ID)
		if err != nil {
			return err
		}
		printList(secgroups, 0, 0, 0, nil)
		return nil
	})

	type MySQLInstanceDBListOptions struct {
		MySQLInstanceIdOptions
		Offset int
		Limit  int
	}

	shellutils.R(&MySQLInstanceDBListOptions{}, "mysql-instance-database-list", "List mysql instance database", func(cli *qcloud.SRegion, args *MySQLInstanceDBListOptions) error {
		databases, totalCount, err := cli.DescribeMySQLDatabases(args.ID, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(databases, 0, 0, 0, nil)
		fmt.Println("TotalCount: ", totalCount)
		return nil
	})

	shellutils.R(&MySQLInstanceDBListOptions{}, "mysql-instance-account-list", "List mysql instance accounts", func(cli *qcloud.SRegion, args *MySQLInstanceDBListOptions) error {
		accounts, totalCount, err := cli.DescribeMySQLAccounts(args.ID, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(accounts, 0, 0, 0, nil)
		fmt.Println("TotalCount: ", totalCount)
		return nil
	})

	shellutils.R(&MySQLInstanceDBListOptions{}, "mysql-instance-backup-list", "List mysql instance backups", func(cli *qcloud.SRegion, args *MySQLInstanceDBListOptions) error {
		backups, totalCount, err := cli.DescribeMySQLBackups(args.ID, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(backups, 0, 0, 0, nil)
		fmt.Println("TotalCount: ", totalCount)
		return nil
	})

	shellutils.R(&MySQLInstanceIdOptions{}, "mysql-instance-show", "Show mysql instance", func(cli *qcloud.SRegion, args *MySQLInstanceIdOptions) error {
		result, err := cli.DescribeMySQLDBInstanceInfo(args.ID)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type MySQLRenewOptions struct {
		MySQLInstanceIdOptions
		MONTH int `choices:"1|2|3|4|5|6|7|8|9|10|11|12|24|36"`
	}

	shellutils.R(&MySQLRenewOptions{}, "mysql-instance-renew", "Renew mysql instance", func(cli *qcloud.SRegion, args *MySQLRenewOptions) error {
		return cli.RenewMySQLDBInstance(args.ID, args.MONTH)
	})

	shellutils.R(&MySQLInstanceIdOptions{}, "mysql-instance-backup-create", "Create mysql instance backup", func(cli *qcloud.SRegion, args *MySQLInstanceIdOptions) error {
		backup, err := cli.CreateMySQLBackup(args.ID, nil)
		if err != nil {
			return err
		}
		printObject(backup)
		return nil
	})

	type RestAccountPasswordOptions struct {
		INSTANCE_ID string
		PASSWORD    string
		USER        string
		Host        string `default:"%"`
	}

	shellutils.R(&RestAccountPasswordOptions{}, "mysql-account-reset-password", "Reset mysql account password", func(cli *qcloud.SRegion, args *RestAccountPasswordOptions) error {
		return cli.ModifyMySQLAccountPassword(args.INSTANCE_ID, args.PASSWORD, map[string]string{args.USER: args.Host})
	})

	type MySQLAccountPrivilegeShowOptions struct {
		INSTANCE_ID string
		USER        string
		Host        string `default:"%"`
	}

	shellutils.R(&MySQLAccountPrivilegeShowOptions{}, "mysql-account-privilege-show", "Show account privileges", func(cli *qcloud.SRegion, args *MySQLAccountPrivilegeShowOptions) error {
		result, err := cli.DescribeAccountPrivileges(args.INSTANCE_ID, args.USER, args.Host)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
