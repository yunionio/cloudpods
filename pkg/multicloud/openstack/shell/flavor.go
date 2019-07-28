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
	"fmt"

	"yunion.io/x/onecloud/pkg/multicloud/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type FlavorkListOptions struct {
	}
	shellutils.R(&FlavorkListOptions{}, "flavor-list", "List flavors", func(cli *openstack.SRegion, args *FlavorkListOptions) error {
		flavors, err := cli.GetFlavors()
		if err != nil {
			return err
		}
		printList(flavors, 0, 0, 0, []string{})
		return nil
	})

	type FlavorOptions struct {
		ID string `help:"ID of flavor"`
	}

	shellutils.R(&FlavorOptions{}, "flavor-show", "Show flavor", func(cli *openstack.SRegion, args *FlavorOptions) error {
		flavor, err := cli.GetFlavor(args.ID)
		if err != nil {
			return err
		}
		printObject(flavor)
		return nil
	})

	shellutils.R(&FlavorOptions{}, "flavor-delete", "Delete flavor", func(cli *openstack.SRegion, args *FlavorOptions) error {
		return cli.DeleteFlavor(args.ID)
	})

	type FlavorCreateOptions struct {
		NAME      string `help:"Name of flavor"`
		CPU       int    `help:"Core num of cpu"`
		MEMORY_MB int    `help:"Memory of flavor"`
		DISK      int    `help:"Disk size of flavor"`
	}

	shellutils.R(&FlavorCreateOptions{}, "flavor-create", "Create flavor", func(cli *openstack.SRegion, args *FlavorCreateOptions) error {
		flavor, err := cli.CreateFlavor(args.NAME, args.CPU, args.MEMORY_MB, args.DISK)
		if err != nil {
			return err
		}
		printObject(flavor)
		return nil
	})

	shellutils.R(&FlavorCreateOptions{}, "flavor-sync", "Sync flavor", func(cli *openstack.SRegion, args *FlavorCreateOptions) error {
		flavorId, err := cli.SyncFlavor(args.NAME, args.CPU, args.MEMORY_MB, args.DISK)
		if err != nil {
			return err
		}
		fmt.Println(flavorId)
		return nil
	})

}
