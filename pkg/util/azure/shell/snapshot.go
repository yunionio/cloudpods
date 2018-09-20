package shell

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SnapshotListOptions struct {
		Disk   string `help:"List snapshot for disk"`
		Limit  int    `help:"page size"`
		Offset int    `help:"page offset"`
	}
	shellutils.R(&SnapshotListOptions{}, "snapshot-list", "List snapshot", func(cli *azure.SRegion, args *SnapshotListOptions) error {
		if snapshots, err := cli.GetSnapShots(args.Disk); err != nil {
			return err
		} else {
			printList(snapshots, len(snapshots), args.Offset, args.Limit, []string{})
			return nil
		}
	})

	type SnapshotCreateOptions struct {
		DISK string `help:"SourceID"`
		NAME string `help:"Snapshot name"`
		Desc string `help:"Snapshot description"`
	}

	shellutils.R(&SnapshotCreateOptions{}, "snapshot-create", "Create snapshot", func(cli *azure.SRegion, args *SnapshotCreateOptions) error {
		if snapshot, err := cli.CreateSnapshot(args.DISK, args.NAME, args.Desc); err != nil {
			return err
		} else {
			printObject(snapshot)
			return nil
		}
	})

	type SnapshotOptions struct {
		ID string `help:"Snapshot ID"`
	}

	shellutils.R(&SnapshotOptions{}, "snapshot-delete", "Delete snapshot", func(cli *azure.SRegion, args *SnapshotOptions) error {
		return cli.DeleteSnapshot(args.ID)
	})

	shellutils.R(&SnapshotOptions{}, "snapshot-show", "List snapshot", func(cli *azure.SRegion, args *SnapshotOptions) error {
		if snapshot, err := cli.GetSnapshotDetail(args.ID); err != nil {
			return err
		} else {
			printObject(snapshot)
			return nil
		}
	})

	shellutils.R(&SnapshotOptions{}, "snapshot-grant-access", "Grant access for snapshot", func(cli *azure.SRegion, args *SnapshotOptions) error {
		if uri, err := cli.GrantAccessSnapshot(args.ID); err != nil {
			return err
		} else {
			fmt.Printf("download link %s", uri)
			return nil
		}
	})

}
