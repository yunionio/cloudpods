package shell

import (
	"fmt"

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

	type SnapshotOptions struct {
		ID string `help:"ID of snapshot"`
	}

	shellutils.R(&SnapshotOptions{}, "snapshot-show", "Show snapshot", func(cli *openstack.SRegion, args *SnapshotOptions) error {
		snapshot, err := cli.GetISnapshotById(args.ID)
		if err != nil {
			return err
		}
		printObject(snapshot)
		return nil
	})

	shellutils.R(&SnapshotOptions{}, "snapshot-delete", "Delete snapshot", func(cli *openstack.SRegion, args *SnapshotOptions) error {
		return cli.DeleteSnapshot(args.ID)
	})

	type SnapshotCreateOptions struct {
		DISKID string `help:"Disk ID"`
		Name   string `help:"Disk Name"`
		Desc   string `help:"Disk description"`
	}

	shellutils.R(&SnapshotCreateOptions{}, "snapshot-create", "Create snapshot", func(cli *openstack.SRegion, args *SnapshotCreateOptions) error {
		snapshotId, err := cli.CreateSnapshot(args.DISKID, args.Name, args.Desc)
		if err != nil {
			return err
		}
		fmt.Println(snapshotId)
		return nil
	})

}
