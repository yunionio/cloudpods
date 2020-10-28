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
	type ResourceGroupListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&ResourceGroupListOptions{}, "resource-group-list", "List group", func(cli *azure.SRegion, args *ResourceGroupListOptions) error {
		groups, err := cli.GetClient().ListResourceGroups()
		if err != nil {
			return err
		}
		printList(groups, len(groups), 0, 0, []string{})
		return nil
	})

	type ResourceGroupOptions struct {
		GROUP string `help:"ResourceGrop Name"`
	}

	shellutils.R(&ResourceGroupOptions{}, "resource-group-show", "Show group detail", func(cli *azure.SRegion, args *ResourceGroupOptions) error {
		if group, err := cli.GetResourceGroupDetail(args.GROUP); err != nil {
			return err
		} else {
			printObject(group)
			return nil
		}
	})

	shellutils.R(&ResourceGroupOptions{}, "resource-group-create", "Create resource group", func(cli *azure.SRegion, args *ResourceGroupOptions) error {
		resp, err := cli.CreateResourceGroup(args.GROUP)
		if err != nil {
			return err
		}
		printObject(resp)
		return nil
	})

	shellutils.R(&ResourceGroupOptions{}, "resource-group-delete", "Delete resource group", func(cli *azure.SRegion, args *ResourceGroupOptions) error {
		err := cli.DeleteResourceGroup(args.GROUP)
		if err != nil {
			return err
		}
		return nil
	})

	type ResourceGroupUpdateOptions struct {
		GROUP string `help:"Name of resource group to update"`
		NAME  string `help:"New name of resource group"`
	}
	shellutils.R(&ResourceGroupUpdateOptions{}, "resource-group-update", "Update resource group detail", func(cli *azure.SRegion, args *ResourceGroupUpdateOptions) error {
		err := cli.UpdateResourceGroup(args.GROUP, args.NAME)
		if err != nil {
			return err
		}
		return nil
	})

}
