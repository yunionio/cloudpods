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
	huawei "yunion.io/x/onecloud/pkg/multicloud/hcs"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DBInstanceListOptions struct {
	}
	shellutils.R(&DBInstanceListOptions{}, "dbinstance-list", "List dbinstances", func(cli *huawei.SRegion, args *DBInstanceListOptions) error {
		dbinstances, err := cli.GetDBInstances()
		if err != nil {
			return err
		}
		printList(dbinstances, 0, 0, 0, nil)
		return nil
	})

	type DBInstanceIdOptions struct {
		ID string `help:"DBInstance ID"`
	}

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-open-public-connection", "Open dbinstance public connection", func(cli *huawei.SRegion, args *DBInstanceIdOptions) error {
		return cli.PublicConnectionAction(args.ID, "openRC")
	})

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-close-public-connection", "Close dbinstance public connection", func(cli *huawei.SRegion, args *DBInstanceIdOptions) error {
		return cli.PublicConnectionAction(args.ID, "closeRC")
	})

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-show", "Show dbinstance", func(cli *huawei.SRegion, args *DBInstanceIdOptions) error {
		dbinstance, err := cli.GetDBInstance(args.ID)
		if err != nil {
			return err
		}
		printObject(dbinstance)
		return nil
	})

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-parameter-list", "Show dbinstance parameters", func(cli *huawei.SRegion, args *DBInstanceIdOptions) error {
		parameters, err := cli.GetDBInstanceParameters(args.ID)
		if err != nil {
			return err
		}
		printList(parameters, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-backup-list", "Show dbinstance backups", func(cli *huawei.SRegion, args *DBInstanceIdOptions) error {
		backups, err := cli.GetDBInstanceBackups(args.ID, "")
		if err != nil {
			return err
		}
		printList(backups, 0, 0, 0, nil)
		return nil
	})

	type DBInstanceFlavorListOption struct {
		ENGINE  string `help:"DBInstance engine" choices:"MySQL|SQLServer|PostgreSQL"`
		VERSION string `help:"DBInstance engine version" choices:"5.6|5.7|9.5|9.6|10.0|2014 SE|2014 EE|2016 SE|2016 EE|2008 R2 EE|2008 R2 WEB|2014 WEB|2016 WEB"`
	}

	shellutils.R(&DBInstanceFlavorListOption{}, "dbinstance-flavor-list", "Show dbinstance backups", func(cli *huawei.SRegion, args *DBInstanceFlavorListOption) error {
		flavors, err := cli.GetDBInstanceFlavors(args.ENGINE, args.VERSION)
		if err != nil {
			return err
		}
		printList(flavors, 0, 0, 0, nil)
		return nil
	})

	type DBInstanceChangeConfigOptions struct {
		INSTANCE     string
		InstanceType string
		DiskSizeGB   int
	}

	shellutils.R(&DBInstanceChangeConfigOptions{}, "dbinstance-change-config", "Change dbinstance config", func(cli *huawei.SRegion, args *DBInstanceChangeConfigOptions) error {
		return cli.ChangeDBInstanceConfig(args.INSTANCE, args.InstanceType, args.DiskSizeGB)
	})

}
