package shell

import (
	"yunion.io/x/onecloud/pkg/util/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SnapshotListOptions struct {
		DiskId string `help:"Disk ID"`
		Name   string `help:"Snapshot Name"`
		Limit  int    `help:"page size"`
		Offset int    `help:"page offset"`
	}
	shellutils.R(&SnapshotListOptions{}, "snapshot-list", "List snapshot", func(cli *huawei.SRegion, args *SnapshotListOptions) error {
		if snapshots, total, err := cli.GetSnapshots(args.DiskId, args.Name, args.Offset, args.Limit); err != nil {
			return err
		} else {
			printList(snapshots, total, args.Offset, args.Limit, []string{})
			return nil
		}
	})

	type SnapshotDeleteOptions struct {
		ID string `help:"Snapshot ID"`
	}

	shellutils.R(&SnapshotDeleteOptions{}, "snapshot-delete", "Delete snapshot", func(cli *huawei.SRegion, args *SnapshotDeleteOptions) error {
		return cli.DeleteSnapshot(args.ID)
	})

	type SnapshotCreateOptions struct {
		DiskId string `help:"Disk ID"`
		Name   string `help:"Snapeshot Name"`
		Desc   string `help:"Snapshot Desc"`
	}

	shellutils.R(&SnapshotCreateOptions{}, "snapshot-create", "Create snapshot", func(cli *huawei.SRegion, args *SnapshotCreateOptions) error {
		_, err := cli.CreateSnapshot(args.DiskId, args.Name, args.Desc)
		return err
	})

}
