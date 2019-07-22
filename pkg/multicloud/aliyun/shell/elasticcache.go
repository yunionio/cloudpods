package shell

import (
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ElasticcacheListOptions struct {
	}
	shellutils.R(&ElasticcacheListOptions{}, "elasticcache-list", "List elasticcaches", func(cli *aliyun.SRegion, args *ElasticcacheListOptions) error {
		instances, e := cli.GetElasticCaches(nil)
		if e != nil {
			return e
		}
		printList(instances, len(instances), 0, 0, []string{})
		return nil
	})

	type ElasticcacheIdOptions struct {
		ID string `help:"ID of instances to show"`
	}
	shellutils.R(&ElasticcacheIdOptions{}, "elasticcache-show", "Show elasticcache", func(cli *aliyun.SRegion, args *ElasticcacheIdOptions) error {
		instance, err := cli.GetElasticCacheById(args.ID)
		if err != nil {
			return err
		}
		printObject(instance)
		return nil
	})

	type ElasticcacheBackupsListOptions struct {
		ID        string `help:"ID of instances to show"`
		StartTime string `help:"backup start time. format: 2019-03-11T10:00Z"`
		EndTime   string `help:"backup end time. format: 2019-03-11T10:00Z"`
	}

	shellutils.R(&ElasticcacheBackupsListOptions{}, "elasticcache-backup-list", "List elasticcache backups", func(cli *aliyun.SRegion, args *ElasticcacheBackupsListOptions) error {
		backups, err := cli.GetElasticCacheBackups(args.ID, args.StartTime, args.EndTime)
		if err != nil {
			return err
		}
		printList(backups, 0, 0, 0, []string{})
		return nil
	})

	shellutils.R(&ElasticcacheIdOptions{}, "elasticcache-parameter-list", "List elasticcache parameters", func(cli *aliyun.SRegion, args *ElasticcacheIdOptions) error {
		parameters, err := cli.GetElasticCacheParameters(args.ID)
		if err != nil {
			return err
		}
		printList(parameters, 0, 0, 0, []string{})
		return nil
	})

	shellutils.R(&ElasticcacheIdOptions{}, "elasticcache-account-list", "List elasticcache accounts", func(cli *aliyun.SRegion, args *ElasticcacheIdOptions) error {
		accounts, err := cli.GetElasticCacheAccounts(args.ID)
		if err != nil {
			return err
		}
		printList(accounts, 0, 0, 0, []string{})
		return nil
	})

	shellutils.R(&ElasticcacheIdOptions{}, "elasticcache-acl-list", "List elasticcache security ip rules", func(cli *aliyun.SRegion, args *ElasticcacheIdOptions) error {
		acls, err := cli.GetElasticCacheAcls(args.ID)
		if err != nil {
			return err
		}
		printList(acls, 0, 0, 0, []string{})
		return nil
	})
}
