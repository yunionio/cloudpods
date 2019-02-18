package shell

import (
	"yunion.io/x/onecloud/pkg/util/openstack"
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

}
