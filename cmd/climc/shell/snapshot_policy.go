// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		NAME string

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

	type SnapshotPolicyBindDisksOptions struct {
		ID   string   `help:"ID"`
		Disk []string `help:"ids of disk"`
	}

	R(&SnapshotPolicyBindDisksOptions{}, "snapshot-policy-bind-disk", "bind snapshotpolicy to disks",
		func(s *mcclient.ClientSession, opts *SnapshotPolicyBindDisksOptions) error {
			params, err := options.StructToParams(opts)
			if err != nil {
				return err
			}
			sp, err := modules.SnapshotPoliciy.PerformAction(s, opts.ID, "bind-disks", params)
			if err != nil {
				return err
			}
			printObject(sp)
			return nil
		})

	R(&SnapshotPolicyBindDisksOptions{}, "snapshot-policy-unbind-disk", "bind snapshotpolicy to disks",
		func(s *mcclient.ClientSession, opts *SnapshotPolicyBindDisksOptions) error {
			params, err := options.StructToParams(opts)
			if err != nil {
				return err
			}
			sp, err := modules.SnapshotPoliciy.PerformAction(s, opts.ID, "unbind-disks", params)
			if err != nil {
				return err
			}
			printObject(sp)
			return nil
		})
}
