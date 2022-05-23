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
	"yunion.io/x/onecloud/pkg/multicloud/apsara"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ResourceGroupListOptions struct {
		PageSize   int
		PageNumber int
	}
	shellutils.R(&ResourceGroupListOptions{}, "resource-group-list", "List resource group", func(cli *apsara.SRegion, args *ResourceGroupListOptions) error {
		groups, _, err := cli.GetClient().GetResourceGroups(args.PageNumber, args.PageSize)
		if err != nil {
			return err
		}
		printList(groups, 0, 0, 0, nil)
		return nil
	})

	type ResourceGroupShowOptions struct {
		ID string
	}

	shellutils.R(&ResourceGroupShowOptions{}, "resource-group-show", "Show resource group", func(cli *apsara.SRegion, args *ResourceGroupShowOptions) error {
		group, err := cli.GetClient().GetResourceGroup(args.ID)
		if err != nil {
			return err
		}
		printObject(group)
		return nil
	})

	type ResourceGroupCreateOptions struct {
		NAME string
	}

	shellutils.R(&ResourceGroupCreateOptions{}, "resource-group-create", "Create resource group", func(cli *apsara.SRegion, args *ResourceGroupCreateOptions) error {
		group, err := cli.GetClient().CreateResourceGroup(args.NAME)
		if err != nil {
			return err
		}
		printObject(group)
		return nil
	})

	type OrganizationListOptions struct {
		Id string
	}

	shellutils.R(&OrganizationListOptions{}, "organization-tree", "List organization tree", func(cli *apsara.SRegion, args *OrganizationListOptions) error {
		org, err := cli.GetClient().GetOrganizationTree(args.Id)
		if err != nil {
			return err
		}
		printObject(org)
		projects := org.GetProject([]string{})
		printList(projects, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&OrganizationListOptions{}, "organization-list", "List organization", func(cli *apsara.SRegion, args *OrganizationListOptions) error {
		orgs, err := cli.GetClient().GetOrganizationList()
		if err != nil {
			return err
		}
		printList(orgs, 0, 0, 0, nil)
		return nil
	})

}
