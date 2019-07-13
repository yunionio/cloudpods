package shell

import (
	"yunion.io/x/onecloud/pkg/util/aliyun"
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

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-delete", "Delete dbintance", func(cli *aliyun.SRegion, args *DBInstanceIdOptions) error {
		return cli.DeleteDBInstance(args.ID)
	})

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-network-list", "Show dbintance network info", func(cli *aliyun.SRegion, args *DBInstanceIdOptions) error {
		networks, err := cli.GetDBInstanceNetInfo(args.ID)
		if err != nil {
			return err
		}
		printList(networks, 0, 0, 0, []string{})
		return nil
	})

	type DBInstanceIdExtraOptions struct {
		ID     string `help:"ID of instances to show"`
		Limit  int    `help:"page size"`
		Offset int    `help:"page offset"`
	}

	shellutils.R(&DBInstanceIdExtraOptions{}, "dbinstance-backup-list", "List dbintance backups", func(cli *aliyun.SRegion, args *DBInstanceIdExtraOptions) error {
		backups, _, err := cli.GetDBInstanceBackups(args.ID, "", args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(backups, 0, 0, 0, []string{})
		return nil
	})

	shellutils.R(&DBInstanceIdExtraOptions{}, "dbinstance-database-list", "List dbintance databases", func(cli *aliyun.SRegion, args *DBInstanceIdExtraOptions) error {
		databases, _, err := cli.GetDBInstanceDatabases(args.ID, "", args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(databases, 0, 0, 0, []string{})
		return nil
	})

	shellutils.R(&DBInstanceIdExtraOptions{}, "dbinstance-account-list", "List dbintance account", func(cli *aliyun.SRegion, args *DBInstanceIdExtraOptions) error {
		accounts, _, err := cli.GetDBInstanceAccounts(args.ID, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(accounts, 0, 0, 0, []string{})
		return nil
	})

}
