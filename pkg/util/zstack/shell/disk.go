package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type DiskListOptions struct {
		StorageId string
		DiskIds   []string
		DiskType  string
	}
	shellutils.R(&DiskListOptions{}, "disk-list", "List disks", func(cli *zstack.SRegion, args *DiskListOptions) error {
		disks, err := cli.GetDisks(args.StorageId, args.DiskIds, args.DiskType)
		if err != nil {
			return err
		}
		printList(disks, len(disks), 0, 0, []string{})
		return nil
	})

	type DiskCreateOptions struct {
		NAME        string
		Description string
		SizeGB      int
		HostId      string
		PoolId      string
		STORAGE_ID  string
	}

	shellutils.R(&DiskCreateOptions{}, "disk-create", "Create disk", func(cli *zstack.SRegion, args *DiskCreateOptions) error {
		disk, err := cli.CreateDisk(args.NAME, args.STORAGE_ID, args.HostId, args.PoolId, args.SizeGB, args.Description)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})

	type DiskDelete struct {
		ID string
	}

	shellutils.R(&DiskDelete{}, "disk-delete", "Delete disk", func(cli *zstack.SRegion, args *DiskDelete) error {
		return cli.DeleteDisk(args.ID)
	})

	type DiskResize struct {
		ID     string
		SIZEGB int64
	}

	shellutils.R(&DiskResize{}, "disk-resize", "Resize disk", func(cli *zstack.SRegion, args *DiskResize) error {
		return cli.ResizeDisk(args.ID, args.SIZEGB*1024)
	})

}
