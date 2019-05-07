package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type SnapshotListOptions struct {
		SnapshotId string
		DiskId     string
	}
	shellutils.R(&SnapshotListOptions{}, "snapshot-list", "List snapshots", func(cli *zstack.SRegion, args *SnapshotListOptions) error {
		snapshots, err := cli.GetSnapshots(args.SnapshotId, args.DiskId)
		if err != nil {
			return err
		}
		printList(snapshots, len(snapshots), 0, 0, []string{})
		return nil
	})

	type SnapshoDeleteOptions struct {
		ID string
	}

	shellutils.R(&SnapshoDeleteOptions{}, "snapshot-delete", "Delete snapshot", func(cli *zstack.SRegion, args *SnapshoDeleteOptions) error {
		return cli.DeleteSnapshot(args.ID)
	})

	type SnapshoCreateOptions struct {
		DISKID string
		NAME   string
		Desc   string
	}

	shellutils.R(&SnapshoCreateOptions{}, "snapshot-create", "Create snapshot", func(cli *zstack.SRegion, args *SnapshoCreateOptions) error {
		snapshot, err := cli.CreateSnapshot(args.NAME, args.DISKID, args.Desc)
		if err != nil {
			return err
		}
		printObject(snapshot)
		return nil
	})
}
