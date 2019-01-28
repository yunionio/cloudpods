package shell

import (
	"fmt"

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

	type DiskOptions struct {
		ID string `help:"ID of disk"`
	}

	shellutils.R(&DiskOptions{}, "disk-show", "Show disk", func(cli *openstack.SRegion, args *DiskOptions) error {
		disk, err := cli.GetDisk(args.ID)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})

	shellutils.R(&DiskOptions{}, "disk-delete", "Delete disk", func(cli *openstack.SRegion, args *DiskOptions) error {
		return cli.DeleteDisk(args.ID)
	})

	type DiskCreateOptions struct {
		ZONE     string `help:"Zone name"`
		CATEGORY string `help:"Disk category"`
		NAME     string `help:"Disk Name"`
		SIZE     int    `help:"Disk Size GB"`
	}
	shellutils.R(&DiskCreateOptions{}, "disk-create", "Create disk", func(cli *openstack.SRegion, args *DiskCreateOptions) error {
		diskId, err := cli.CreateDisk(args.ZONE, args.CATEGORY, args.NAME, args.SIZE, "")
		if err != nil {
			return err
		}
		fmt.Println(diskId)
		return nil
	})
}
