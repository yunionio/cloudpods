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
	"yunion.io/x/onecloud/pkg/multicloud/bingocloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type StorageListOptions struct {
		MaxRecords int
		NextToken  string
	}
	shellutils.R(&StorageListOptions{}, "storage-list", "list storage", func(cli *bingocloud.SRegion, args *StorageListOptions) error {
		storages, err := cli.GetStorages()
		if err != nil {
			return err
		}
		printList(storages, 0, 0, 0, []string{})
		return nil
	})

	type StorageCacheListOptions struct{}

	shellutils.R(&StorageCacheListOptions{}, "storagecache-list", "list storagecache", func(cli *bingocloud.SRegion, args *StorageCacheListOptions) error {
		storagecaches, err := cli.GetStorages()
		if err != nil {
			return err
		}
		printList(storagecaches, 0, 0, 0, []string{})
		return nil
	})

	type StorageIdOptions struct {
		ID string
	}

	shellutils.R(&StorageIdOptions{}, "storage-show", "show storage", func(cli *bingocloud.SRegion, args *StorageIdOptions) error {
		storage, err := cli.GetStorage(args.ID)
		if err != nil {
			return err
		}
		printObject(storage)
		return nil
	})

}
