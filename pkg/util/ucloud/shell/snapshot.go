package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/ucloud"
)

func init() {
	type SnapshotListOptions struct {
		DiskId string `help:"Disk ID"`
		Name   string `help:"Snapshot Name"`
	}
	shellutils.R(&SnapshotListOptions{}, "snapshot-list", "List snapshot", func(cli *ucloud.SRegion, args *SnapshotListOptions) error {
		snapshots, err := cli.GetSnapshots(args.DiskId, args.Name)
		if err != nil {
			return err
		}
		printList(snapshots, 0, 0, 0, nil)
		return nil
	})

	type SnapshotDeleteOptions struct {
		ID string `help:"Snapshot ID"`
	}

	shellutils.R(&SnapshotDeleteOptions{}, "snapshot-delete", "Delete snapshot", func(cli *ucloud.SRegion, args *SnapshotDeleteOptions) error {
		return cli.DeleteSnapshot(args.ID)
	})

	type SnapshotCreateOptions struct {
		ZoneId string `help:"Zone ID"`
		DiskId string `help:"Disk ID"`
		Name   string `help:"Snapeshot Name"`
		Desc   string `help:"Snapshot Desc"`
	}

	shellutils.R(&SnapshotCreateOptions{}, "snapshot-create", "Create snapshot", func(cli *ucloud.SRegion, args *SnapshotCreateOptions) error {
		_, err := cli.CreateSnapshot(args.ZoneId, args.DiskId, args.Name, args.Desc)
		return err
	})

}
