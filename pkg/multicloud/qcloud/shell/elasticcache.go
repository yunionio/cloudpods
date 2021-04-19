package shell

import (
	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type RedisListOptions struct {
	}
	shellutils.R(&RedisListOptions{}, "redis-list", "List redis", func(cli *qcloud.SRegion, args *RedisListOptions) error {
		redis, err := cli.GetCloudElasticcaches("")
		if err != nil {
			return err
		}
		printList(redis, 0, 0, 0, []string{})
		return nil
	})

	type RedisParameterListOptions struct {
		INSTANCEID string `json:"instanceid"`
	}

	shellutils.R(&RedisParameterListOptions{}, "redis-parameter-list", "List redis parameters", func(cli *qcloud.SRegion, args *RedisParameterListOptions) error {
		parameters, err := cli.GetCloudElasticcacheParameters(args.INSTANCEID)
		if err != nil {
			return err
		}
		printList(parameters, 0, 0, 0, []string{})
		return nil
	})

	type RedisBackupListOptions struct {
		INSTANCEID string `json:"instanceid"`
	}

	shellutils.R(&RedisBackupListOptions{}, "redis-backup-list", "List redis backups", func(cli *qcloud.SRegion, args *RedisBackupListOptions) error {
		backups, err := cli.GetCloudElasticcacheBackups(args.INSTANCEID)
		if err != nil {
			return err
		}
		printList(backups, 0, 0, 0, []string{})
		return nil
	})

	type RedisSecGroupListOptions struct {
		INSTANCEID string `json:"instanceid"`
	}

	shellutils.R(&RedisSecGroupListOptions{}, "redis-secgroup-list", "List redis secgroups", func(cli *qcloud.SRegion, args *RedisSecGroupListOptions) error {
		secgroups, err := cli.GetCloudElasticcacheSecurityGroups(args.INSTANCEID)
		if err != nil {
			return err
		}
		printList(secgroups, 0, 0, 0, []string{})
		return nil
	})

	type RedisAccountListOptions struct {
		INSTANCEID string `json:"instanceid"`
	}

	shellutils.R(&RedisAccountListOptions{}, "redis-account-list", "List redis accounts", func(cli *qcloud.SRegion, args *RedisAccountListOptions) error {
		accounts, err := cli.GetCloudElasticcacheAccounts(args.INSTANCEID)
		if err != nil {
			return err
		}
		printList(accounts, 0, 0, 0, []string{})
		return nil
	})
}
