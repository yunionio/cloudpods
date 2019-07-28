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
	"yunion.io/x/onecloud/pkg/multicloud/zstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type StorageListOptions struct {
		ZoneId    string
		ClusterId string
		Id        string
	}
	shellutils.R(&StorageListOptions{}, "storage-list", "List storages", func(cli *zstack.SRegion, args *StorageListOptions) error {
		storages, err := cli.GetStorages(args.ZoneId, args.ClusterId, args.Id)
		if err != nil {
			return err
		}
		printList(storages, 0, 0, 0, []string{})
		return nil
	})

	type LocalStorageOptions struct {
		STORAGE_ID string
		HostId     string
	}

	shellutils.R(&LocalStorageOptions{}, "local-storage-list", "Show local storages", func(cli *zstack.SRegion, args *LocalStorageOptions) error {
		storages, err := cli.GetLocalStorages(args.STORAGE_ID, args.HostId)
		if err != nil {
			return err
		}
		printList(storages, 0, 0, 0, []string{})
		return nil
	})

}
