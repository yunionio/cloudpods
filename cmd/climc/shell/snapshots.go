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
	type SnapshotsListOptions struct {
		options.BaseListOptions

		Disk        string `help:"Disk snapshots" json:"disk_id"`
		FakeDeleted bool   `help:"Show fake deleted snapshot or not"`
		Local       *bool  `help:"Show local snapshots"`
		Share       *bool  `help:"Show shared snapshots"`
		DiskType    string `help:"Filter by disk type" choices:"sys|data"`
	}
	R(&SnapshotsListOptions{}, "snapshot-list", "Show snapshots", func(s *mcclient.ClientSession, args *SnapshotsListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Snapshots.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Snapshots.GetColumns(s))
		return nil
	})

	type SnapshotDeleteOptions struct {
		ID []string `help:"Delete snapshot id"`
	}
	R(&SnapshotDeleteOptions{}, "snapshot-delete", "Delete snapshots", func(s *mcclient.ClientSession, args *SnapshotDeleteOptions) error {
		ret := modules.Snapshots.BatchDelete(s, args.ID, nil)
		printBatchResults(ret, modules.Snapshots.GetColumns(s))
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

	type SnapshotPurgeOptions struct {
		ID string `help:"ID or name of Snapshot"`
	}
	R(&SnapshotPurgeOptions{}, "snapshot-purge", "Purge Snapshot db records", func(s *mcclient.ClientSession, args *SnapshotPurgeOptions) error {
		result, err := modules.Snapshots.PerformAction(s, args.ID, "purge", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type SnapshotCreateOptions struct {
		Disk string `help:"Id of disk to take snapshot" json:"disk" required:"true"`
		NAME string `help:"Name of snapshot" json:"name"`
	}
	R(&SnapshotCreateOptions{}, "snapshot-create", "Create a snapshot", func(s *mcclient.ClientSession, args *SnapshotCreateOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		snapshot, err := modules.Snapshots.Create(s, params)
		if err != nil {
			return err
		}
		printObject(snapshot)
		return nil
	})
}
