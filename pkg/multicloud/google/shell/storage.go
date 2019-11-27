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
	type StorageListOptions struct {
		ZONE       string
		MaxResults int
		PageToken  string
	}
	shellutils.R(&StorageListOptions{}, "storage-list", "List storages", func(cli *google.SRegion, args *StorageListOptions) error {
		storages, err := cli.GetStorages(args.ZONE, args.MaxResults, args.PageToken)
		if err != nil {
			return err
		}
		printList(storages, 0, 0, 0, nil)
		return nil
	})

	type StorageShowOptions struct {
		ID string
	}
	shellutils.R(&StorageShowOptions{}, "storage-show", "Show storage", func(cli *google.SRegion, args *StorageShowOptions) error {
		storage, err := cli.GetStorage(args.ID)
		if err != nil {
			return err
		}
		printObject(storage)
		return nil
	})

	type RegionStorageListOptions struct {
		MaxResults int
		PageToken  string
	}
	shellutils.R(&RegionStorageListOptions{}, "region-storage-list", "List region storages", func(cli *google.SRegion, args *RegionStorageListOptions) error {
		storages, err := cli.GetRegionStorages(args.MaxResults, args.PageToken)
		if err != nil {
			return err
		}
		printList(storages, 0, 0, 0, nil)
		return nil
	})

	type RegionStorageShowOptions struct {
		ID string
	}
	shellutils.R(&RegionStorageShowOptions{}, "region-storage-show", "Show region storage", func(cli *google.SRegion, args *RegionStorageShowOptions) error {
		storage, err := cli.GetRegionStorage(args.ID)
		if err != nil {
			return err
		}
		printObject(storage)
		return nil
	})

}
