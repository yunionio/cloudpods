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
