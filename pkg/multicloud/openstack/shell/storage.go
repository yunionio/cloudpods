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
	"yunion.io/x/onecloud/pkg/multicloud/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type StorageListOptions struct {
	}
	shellutils.R(&StorageListOptions{}, "storage-list", "List storages", func(cli *openstack.SRegion, args *StorageListOptions) error {
		storages, err := cli.GetStorageTypes()
		if err != nil {
			return err
		}
		printList(storages, 0, 0, 0, []string{})
		return nil
	})

	type CinderServiceListOptions struct {
	}

	shellutils.R(&CinderServiceListOptions{}, "cinder-service-list", "List cinder services", func(cli *openstack.SRegion, args *CinderServiceListOptions) error {
		services, err := cli.GetCinderServices()
		if err != nil {
			return err
		}
		printList(services, 0, 0, 0, []string{})
		return nil
	})
}
