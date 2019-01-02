package shell

import (
	"yunion.io/x/onecloud/pkg/util/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SnapshotListOptions struct {
		DiskId string `help:"Disk ID for filter snapshot"`
	}
	shellutils.R(&SnapshotListOptions{}, "snapshot-list", "List snapshots", func(cli *openstack.SRegion, args *SnapshotListOptions) error {
		snapshots, err := cli.GetSnapshots(args.DiskId)
		if err != nil {
			return err
		}
		printList(snapshots, 0, 0, 0, []string{})
		return nil
	})

	type SnapshotShowOptions struct {
		ID string `help:"ID of snapshot"`
	}

	shellutils.R(&SnapshotShowOptions{}, "snapshot-show", "Show snapshot", func(cli *openstack.SRegion, args *SnapshotShowOptions) error {
		snapshot, err := cli.GetISnapshotById(args.ID)
		if err != nil {
			return err
		}
		printObject(snapshot)
		return nil
	})

}
