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
	"yunion.io/x/onecloud/pkg/multicloud/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ElasticcacheListOptions struct {
	}
	shellutils.R(&ElasticcacheListOptions{}, "dcs-list", "List elasticcaches", func(cli *huawei.SRegion, args *ElasticcacheListOptions) error {
		instances, e := cli.GetElasticCaches()
		if e != nil {
			return e
		}
		printList(instances, len(instances), 0, 0, []string{})
		return nil
	})

	type ElasticcacheIdOptions struct {
		ID string `help:"ID of instances to show"`
	}
	shellutils.R(&ElasticcacheIdOptions{}, "dcs-show", "Show elasticcache", func(cli *huawei.SRegion, args *ElasticcacheIdOptions) error {
		instance, err := cli.GetElasticCache(args.ID)
		if err != nil {
			return err
		}
		printObject(instance)
		return nil
	})

	type ElasticcacheBackupsListOptions struct {
		ID        string `help:"ID of instances to show"`
		StartTime string `help:"backup start time. format: 20060102150405"`
		EndTime   string `help:"backup end time. format: 20060102150405 "`
	}

	shellutils.R(&ElasticcacheBackupsListOptions{}, "dcs-backup-list", "List elasticcache backups", func(cli *huawei.SRegion, args *ElasticcacheBackupsListOptions) error {
		backups, err := cli.GetElasticCacheBackups(args.ID, args.StartTime, args.EndTime)
		if err != nil {
			return err
		}
		printList(backups, 0, 0, 0, []string{})
		return nil
	})

	shellutils.R(&ElasticcacheIdOptions{}, "dcs-parameter-list", "List elasticcache parameters", func(cli *huawei.SRegion, args *ElasticcacheIdOptions) error {
		parameters, err := cli.GetElasticCacheParameters(args.ID)
		if err != nil {
			return err
		}
		printList(parameters, 0, 0, 0, []string{})
		return nil
	})
}
