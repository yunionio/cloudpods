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
