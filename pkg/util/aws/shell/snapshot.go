package shell

import (
	"yunion.io/x/onecloud/pkg/util/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SnapshotListOptions struct {
		DiskId      string   `help:"Disk ID"`
		InstanceId  string   `help:"Instance ID"`
		SnapshotIds []string `helo:"Snapshot ids"`
		Name        string   `help:"Snapshot Name"`
		Limit       int      `help:"page size"`
		Offset      int      `help:"page offset"`
	}
	shellutils.R(&SnapshotListOptions{}, "snapshot-list", "List snapshot", func(cli *aws.SRegion, args *SnapshotListOptions) error {
		if snapshots, total, err := cli.GetSnapshots(args.InstanceId, args.DiskId, args.Name, args.SnapshotIds, args.Offset, args.Limit); err != nil {
			return err
		} else {
			printList(snapshots, total, args.Offset, args.Limit, []string{})
			return nil
		}
	})

	type SnapshotDeleteOptions struct {
		ID string `help:"Snapshot ID"`
	}

	shellutils.R(&SnapshotDeleteOptions{}, "snapshot-delete", "Delete snapshot", func(cli *aws.SRegion, args *SnapshotDeleteOptions) error {
		return cli.DeleteSnapshot(args.ID)
	})

	type SnapshotCreateOptions struct {
		DiskId string `help:"Disk ID"`
		Name   string `help:"Snapeshot Name"`
		Desc   string `help:"Snapshot Desc"`
	}

	shellutils.R(&SnapshotCreateOptions{}, "snapshot-create", "Create snapshot", func(cli *aws.SRegion, args *SnapshotCreateOptions) error {
		_, err := cli.CreateSnapshot(args.DiskId, args.Name, args.Desc)
		return err
	})

}
