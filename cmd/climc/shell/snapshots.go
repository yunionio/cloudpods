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
		Disk string `help:"Disk snapshots"`
	}
	R(&SnapshotsListOptions{}, "snapshot-list", "Show snapshots", func(s *mcclient.ClientSession, args *SnapshotsListOptions) error {
		params, err := args.BaseListOptions.Params()
		if err != nil {
			return err
		}
		params.Add(jsonutils.NewString(args.Disk), "disk_id")
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
}
