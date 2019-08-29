package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type InstanceSnapshotsListOptions struct {
		options.BaseListOptions

		GuestId string `help:"guest id" json:"guest_id"`
	}
	R(&InstanceSnapshotsListOptions{}, "instance-snapshot-list", "Show instance snapshots", func(s *mcclient.ClientSession, args *InstanceSnapshotsListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.InstanceSnapshots.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.InstanceSnapshots.GetColumns(s))
		return nil
	})

	type InstanceSnapshotDeleteOptions struct {
		ID []string `help:"Delete snapshot id"`
	}
	R(&InstanceSnapshotDeleteOptions{}, "instance-snapshot-delete", "Delete snapshots", func(s *mcclient.ClientSession, args *InstanceSnapshotDeleteOptions) error {
		ret := modules.InstanceSnapshots.BatchDelete(s, args.ID, nil)
		printBatchResults(ret, modules.InstanceSnapshots.GetColumns(s))
		return nil
	})

	type InstanceSnapshotShowOptions struct {
		ID string `help:"ID or Name of snapshot"`
	}
	R(&InstanceSnapshotShowOptions{}, "snapshot-show", "Show snapshot details", func(s *mcclient.ClientSession, args *InstanceSnapshotShowOptions) error {
		result, err := modules.InstanceSnapshots.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
