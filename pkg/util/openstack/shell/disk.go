package shell

import (
	"yunion.io/x/onecloud/pkg/util/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DiskListOptions struct {
		Category string `help:"Storage type for disk"`
	}
	shellutils.R(&DiskListOptions{}, "disk-list", "List disks", func(cli *openstack.SRegion, args *DiskListOptions) error {
		disks, err := cli.GetDisks(args.Category)
		if err != nil {
			return err
		}
		printList(disks, 0, 0, 0, []string{})
		return nil
	})

	type DiskShowOptions struct {
		ID string `help:"Storage type for disk"`
	}

	shellutils.R(&DiskShowOptions{}, "disk-show", "Show disk", func(cli *openstack.SRegion, args *DiskShowOptions) error {
		disk, err := cli.GetDisk(args.ID)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})
}
