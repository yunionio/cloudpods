package shell

import (
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DiskListOptions struct {
		Offset int `help:"List offset"`
		Limit  int `help:"List limit"`
	}
	shellutils.R(&DiskListOptions{}, "disk-list", "List disks", func(cli *azure.SRegion, args *DiskListOptions) error {
		if disks, err := cli.GetDisks(); err != nil {
			return err
		} else {
			printList(disks, len(disks), args.Offset, args.Limit, []string{})
			return nil
		}
		return nil
	})

	type DiskCreateOptions struct {
		NAME        string `help:"Disk name"`
		StorageType string `help:"Storage type" choices:"Standard_LRS|Premium_LRS"`
		SizeGb      int32  `help:"Disk size"`
		Desc        string `help:"description for disk"`
	}

	shellutils.R(&DiskCreateOptions{}, "disk-create", "Create disk", func(cli *azure.SRegion, args *DiskCreateOptions) error {
		if diskId, err := cli.CreateDisk(args.StorageType, args.NAME, args.SizeGb, args.Desc); err != nil {
			return err
		} else if disk, err := cli.GetDisk(diskId); err != nil {
			return err
		} else {
			printObject(disk)
		}
		return nil
	})

	type DiskOptions struct {
		ID string `help:"Disk ID"`
	}

	shellutils.R(&DiskOptions{}, "disk-show", "Show disk", func(cli *azure.SRegion, args *DiskOptions) error {
		if disk, err := cli.GetDisk(args.ID); err != nil {
			return err
		} else {
			printObject(disk)
			return nil
		}
	})

	shellutils.R(&DiskOptions{}, "disk-delete", "Delete disks", func(cli *azure.SRegion, args *DiskOptions) error {
		return cli.DeleteDisk(args.ID)
	})

	type DiskResizeOptions struct {
		ID   string `help:"Disk ID"`
		SIZE int32  `help:"Disk SizeGb"`
	}

	shellutils.R(&DiskResizeOptions{}, "disk-resize", "Delete disks", func(cli *azure.SRegion, args *DiskResizeOptions) error {
		return cli.ResizeDisk(args.ID, args.SIZE)
	})
}
