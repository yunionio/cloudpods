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
}
