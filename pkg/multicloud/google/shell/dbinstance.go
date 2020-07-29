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
	"yunion.io/x/onecloud/pkg/multicloud/google"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DBInstanceListOptions struct {
		MaxResults int
		PageToken  string
	}
	shellutils.R(&DBInstanceListOptions{}, "dbinstance-list", "List dbinstances", func(cli *google.SRegion, args *DBInstanceListOptions) error {
		instances, err := cli.GetDBInstances(args.MaxResults, args.PageToken)
		if err != nil {
			return err
		}
		printList(instances, 0, 0, 0, nil)
		return nil
	})

	type DBInstanceIdOptions struct {
		INSTANCE string
	}

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-show", "Show dbinstance", func(cli *google.SRegion, args *DBInstanceIdOptions) error {
		instance, err := cli.GetDBInstance(args.INSTANCE)
		if err != nil {
			return err
		}
		printObject(instance)
		return nil
	})

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-close-public-connection", "Close dbinstance public connection", func(cli *google.SRegion, args *DBInstanceIdOptions) error {
		return cli.DBInstancePublicConnectionOperation(args.INSTANCE, false)
	})

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-open-public-connection", "Open dbinstance public connection", func(cli *google.SRegion, args *DBInstanceIdOptions) error {
		return cli.DBInstancePublicConnectionOperation(args.INSTANCE, true)
	})

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-delete", "Delete dbinstance", func(cli *google.SRegion, args *DBInstanceIdOptions) error {
		return cli.DeleteDBInstance(args.INSTANCE)
	})

	type DBInstanceCreateOptions struct {
		NAME            string
		ENGINE          string
		DATABASEVERSION string
		Category        string `default:"Zonal" choices:"Zonal|Regional"`
		INSTANCE_TYPE   string
		STORAGE_TYPE    string
		DISK_SIZE_GB    int
		VpcId           string
		ZoneId          string
		Password        string
	}

	shellutils.R(&DBInstanceCreateOptions{}, "dbinstance-create", "Create dbinstance", func(cli *google.SRegion, args *DBInstanceCreateOptions) error {
		instance, err := cli.CreateRds(args.NAME, args.ENGINE, args.DATABASEVERSION, args.Category, args.INSTANCE_TYPE, args.STORAGE_TYPE, args.DISK_SIZE_GB, args.VpcId, args.ZoneId, args.Password)
		if err != nil {
			return err
		}
		printObject(instance)
		return nil
	})

	type DBInstanceChangeConfigOptions struct {
		INSTANCE     string
		DiskSizeGb   int
		InstnaceType string
	}

	shellutils.R(&DBInstanceChangeConfigOptions{}, "dbinstance-change-config", "Change dbinstance config", func(cli *google.SRegion, args *DBInstanceChangeConfigOptions) error {
		return cli.ChangeDBInstanceConfig(args.INSTANCE, args.DiskSizeGb, args.InstnaceType)
	})

	type DBInstanceRecoveryOptions struct {
		NAME   string
		TARGET string
		BACKUP string
	}

	shellutils.R(&DBInstanceRecoveryOptions{}, "dbinstance-restore", "restore dbinstance from backup", func(cli *google.SRegion, args *DBInstanceRecoveryOptions) error {
		return cli.RecoverFromBackup(args.NAME, args.TARGET, args.BACKUP)
	})

}
