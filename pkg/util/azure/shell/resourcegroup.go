package shell

import (
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ResourceGroupListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&ResourceGroupListOptions{}, "group-list", "List group", func(cli *azure.SRegion, args *ResourceGroupListOptions) error {
		if groups, err := cli.GetResourceGroups(); err != nil {
			return err
		} else {
			printList(groups, len(groups), args.Offset, args.Limit, []string{})
			return nil
		}
	})

	type ResourceGroupOptions struct {
		GROUP string `help:"ResourceGrop Name"`
	}

	shellutils.R(&ResourceGroupOptions{}, "group-show", "Show group detail", func(cli *azure.SRegion, args *ResourceGroupOptions) error {
		if group, err := cli.GetResourceGroupDetail(args.GROUP); err != nil {
			return err
		} else {
			printObject(group)
			return nil
		}
	})

	shellutils.R(&ResourceGroupOptions{}, "group-create", "Create group", func(cli *azure.SRegion, args *ResourceGroupOptions) error {
		if group, err := cli.CreateResourceGroup(args.GROUP); err != nil {
			return err
		} else {
			printObject(group)
			return nil
		}
	})

}
