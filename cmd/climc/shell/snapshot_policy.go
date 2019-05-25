package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type SnapshotPolicyListOptions struct {
		options.BaseListOptions
	}
	R(&SnapshotPolicyListOptions{}, "snapshot-policy-list", "List snapshot policy", func(s *mcclient.ClientSession, args *SnapshotPolicyListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.SnapshotPoliciy.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.SnapshotPoliciy.GetColumns(s))
		return nil
	})

	type SnapshotPolicyDeleteOptions struct {
		ID string `help:"Delete snapshot id"`
	}
	R(&SnapshotPolicyDeleteOptions{}, "snapshot-policy-delete", "Delete snapshot policy", func(s *mcclient.ClientSession, args *SnapshotPolicyDeleteOptions) error {
		result, err := modules.SnapshotPoliciy.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type SnapshotPolicyCreateOptions struct {
		NAME        string
		Manager     string `help:"Manager id or name"`
		Cloudregion string `help:"Cloudregion id or name"`

		RetentionDays  int   `help:"snapshot retention days"`
		RepeatWeekdays []int `help:"snapshot create days on week"`
		TimePoints     []int `help:"snapshot create time points on one day`
	}

	R(&SnapshotPolicyCreateOptions{}, "snapshot-policy-create", "Create snapshot policy", func(s *mcclient.ClientSession, args *SnapshotPolicyCreateOptions) error {
		params := jsonutils.Marshal(args).(*jsonutils.JSONDict)
		snapshot, err := modules.SnapshotPoliciy.Create(s, params)
		if err != nil {
			return err
		}
		printObject(snapshot)
		return nil
	})

	type SnapshotPolicyApplyOptions struct {
		ID   string   `help:"ID or Name of SnapshotPolicy" json:"-"`
		Disk []string `help:"Disks id to apply snapshot policy"`
	}

	R(&SnapshotPolicyApplyOptions{}, "snapshot-policy-apply", "Apply snapshot policy to disks", func(s *mcclient.ClientSession, args *SnapshotPolicyApplyOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		snapshot, err := modules.SnapshotPoliciy.PerformAction(s, args.ID, "apply-to-disk", params)
		if err != nil {
			return err
		}
		printObject(snapshot)
		return nil
	})
}
