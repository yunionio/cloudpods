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
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/hcs"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ElasticcacheListOptions struct {
	}
	shellutils.R(&ElasticcacheListOptions{}, "redis-list", "List elasticcaches", func(cli *hcs.SRegion, args *ElasticcacheListOptions) error {
		instances, e := cli.GetElasticCaches()
		if e != nil {
			return e
		}
		printList(instances, len(instances), 0, 0, []string{})
		return nil
	})

	type ElasitccacheInstanceTypeListOptions struct {
	}

	shellutils.R(&ElasitccacheInstanceTypeListOptions{}, "redis-instance-type-list", "List elasticcache instancetypes", func(cli *hcs.SRegion, args *ElasitccacheInstanceTypeListOptions) error {
		ret, e := cli.GetRedisInstnaceTypes()
		if e != nil {
			return e
		}
		printList(ret, 0, 0, 0, []string{})
		return nil
	})

	shellutils.R(&cloudprovider.SCloudElasticCacheInput{}, "redis-create", "Create elasticcache", func(cli *hcs.SRegion, args *cloudprovider.SCloudElasticCacheInput) error {
		instance, e := cli.CreateElasticcache(args)
		if e != nil {
			return e
		}
		printObject(instance)
		return nil
	})

	type ElasticcacheIdOptions struct {
		ID string `help:"ID of instances to show"`
	}
	shellutils.R(&ElasticcacheIdOptions{}, "redis-show", "Show elasticcache", func(cli *hcs.SRegion, args *ElasticcacheIdOptions) error {
		instance, err := cli.GetElasticCache(args.ID)
		if err != nil {
			return err
		}
		printObject(instance)
		return nil
	})

	shellutils.R(&ElasticcacheIdOptions{}, "redis-delete", "Delete elasticcache", func(cli *hcs.SRegion, args *ElasticcacheIdOptions) error {
		return cli.DeleteElasticcache(args.ID)
	})

	type ElasticcacheBackupsListOptions struct {
		ID        string `help:"ID of instances to show"`
		StartTime string `help:"backup start time. format: 20060102150405"`
		EndTime   string `help:"backup end time. format: 20060102150405 "`
	}

	shellutils.R(&ElasticcacheBackupsListOptions{}, "redis-backup-list", "List elasticcache backups", func(cli *hcs.SRegion, args *ElasticcacheBackupsListOptions) error {
		backups, err := cli.GetElasticCacheBackups(args.ID, args.StartTime, args.EndTime)
		if err != nil {
			return err
		}
		printList(backups, 0, 0, 0, []string{})
		return nil
	})

	shellutils.R(&ElasticcacheIdOptions{}, "redis-parameter-list", "List elasticcache parameters", func(cli *hcs.SRegion, args *ElasticcacheIdOptions) error {
		parameters, err := cli.GetElasticCacheParameters(args.ID)
		if err != nil {
			return err
		}
		printList(parameters, 0, 0, 0, []string{})
		return nil
	})
}
