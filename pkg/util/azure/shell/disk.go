package shell

import (
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DiskListOptions struct {
		// Instance string `help:"Instance ID"`
		// Zone     string `help:"Zone ID"`
		// Category string `help:"Disk category"`
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
		StorageType string `help:"Storage type" choices:""`
		SizeGb      int32  `help:"Disk size"`
		Desc        string `help:"description for disk"`
	}

	shellutils.R(&DiskCreateOptions{}, "disk-create", "Create disk", func(cli *azure.SRegion, args *DiskCreateOptions) error {
		resourceGroup, diskName := azure.PareResourceGroupWithName(args.NAME, azure.DISK_RESOURCE)
		if err := cli.CreateDisk(args.StorageType, args.NAME, args.SizeGb, args.Desc); err != nil {
			return err
		} else if disk, err := cli.GetDisk(resourceGroup, diskName); err != nil {
			return err
		} else {
			printObject(disk)
		}
		return nil
	})

	type DiskDeleteOptions struct {
		NAME          string `help:"Disk name"`
		ResourceGroup string `help:"Disk resourceGroup name"`
	}

	shellutils.R(&DiskDeleteOptions{}, "disk-delete", "Delete disks", func(cli *azure.SRegion, args *DiskDeleteOptions) error {
		resourceGroup := args.ResourceGroup
		if len(resourceGroup) == 0 {
			resourceGroup, _ = azure.PareResourceGroupWithName(args.NAME, azure.DISK_RESOURCE)
		}
		if disk, err := cli.GetDisk(resourceGroup, args.NAME); err != nil {
			return err
		} else if err := cli.DeleteDisk(disk.ID); err != nil {
			return err
		} else {
			return nil
		}
	})
}
