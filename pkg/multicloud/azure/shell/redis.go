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
	type RedisListOptions struct {
	}
	shellutils.R(&RedisListOptions{}, "redis-list", "List redis", func(cli *azure.SRegion, args *RedisListOptions) error {
		redis, err := cli.GetRedisCaches()
		if err != nil {
			return err
		}
		printList(redis, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&RedisListOptions{}, "enterprise-redis-list", "List enterprise redis", func(cli *azure.SRegion, args *RedisListOptions) error {
		redis, err := cli.GetEnterpriseRedisCaches()
		if err != nil {
			return err
		}
		printList(redis, 0, 0, 0, nil)
		return nil
	})

	type RedisIdOptions struct {
		ID string
	}
	shellutils.R(&RedisIdOptions{}, "redis-acl-list", "List redis acls", func(cli *azure.SRegion, args *RedisIdOptions) error {
		acls, err := cli.GetRedisAcls(args.ID)
		if err != nil {
			return err
		}
		printList(acls, 0, 0, 0, nil)
		return nil
	})

}
