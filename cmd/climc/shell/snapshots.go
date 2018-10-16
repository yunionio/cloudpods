package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type SnapshotsListOptions struct {
		options.BaseListOptions
		Disk        string `help:"Disk snapshots"`
		FakeDeleted bool   `help:"Show fake deleted snapshot or not"`
		DiskType    string `help: "Filter by disk type" choices:"sys|data"`
		Provider    string `help: "Cloud provider" choices:"Aliyun|VMware|Azure"`
	}
	R(&SnapshotsListOptions{}, "snapshot-list", "Show snapshots", func(s *mcclient.ClientSession, args *SnapshotsListOptions) error {
		params, err := args.BaseListOptions.Params()
		if err != nil {
			return err
		}
		if len(args.Disk) > 0 {
			params.Add(jsonutils.NewString(args.Disk), "disk_id")
		}
		params.Add(jsonutils.NewBool(args.FakeDeleted), "fake_deleted")
		if len(args.Disk) > 0 {
			params.Add(jsonutils.NewString(args.Disk), "disk_id")
		}
		if len(args.DiskType) > 0 {
			params.Add(jsonutils.NewString(args.DiskType), "disk_type")
		}
		if len(args.Provider) > 0 {
			params.Add(jsonutils.NewString(args.Provider), "provider")
		}
		result, err := modules.Snapshots.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Snapshots.GetColumns(s))
		return nil
	})

	type SnapshotDeleteOptions struct {
		ID string `help:"Delete snapshot id"`
	}
	R(&SnapshotDeleteOptions{}, "snapshot-delete", "Delete snapshots", func(s *mcclient.ClientSession, args *SnapshotDeleteOptions) error {
		result, err := modules.Snapshots.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	type DiskDeleteSnapshotsOptions struct {
		DISK string `help:"ID of disk"`
	}
	R(&DiskDeleteSnapshotsOptions{}, "disk-delete-snapshots", "Delete a disk snapshots", func(s *mcclient.ClientSession, args *DiskDeleteSnapshotsOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.DISK), "disk_id")
		result, err := modules.Snapshots.PerformClassAction(s, "delete-disk-snapshots", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	type SnapshotShowOptions struct {
		ID string `help:"ID or Name of snapshot"`
	}
	R(&SnapshotShowOptions{}, "snapshot-show", "Show snapshot details", func(s *mcclient.ClientSession, args *SnapshotShowOptions) error {
		result, err := modules.Snapshots.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
