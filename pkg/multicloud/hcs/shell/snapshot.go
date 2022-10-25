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
	"yunion.io/x/onecloud/pkg/multicloud/hcs"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SnapshotListOptions struct {
		DiskId string `help:"Disk ID"`
		Name   string `help:"Snapshot Name"`
	}
	shellutils.R(&SnapshotListOptions{}, "snapshot-list", "List snapshot", func(cli *hcs.SRegion, args *SnapshotListOptions) error {
		snapshots, err := cli.GetSnapshots(args.DiskId, args.Name)
		if err != nil {
			return err
		}
		printList(snapshots, 0, 0, 0, nil)
		return nil
	})

	type SnapshotIdOptions struct {
		ID string `help:"Snapshot ID"`
	}

	shellutils.R(&SnapshotIdOptions{}, "snapshot-delete", "Delete snapshot", func(cli *hcs.SRegion, args *SnapshotIdOptions) error {
		return cli.DeleteSnapshot(args.ID)
	})

	shellutils.R(&SnapshotIdOptions{}, "snapshot-show", "Show snapshot", func(cli *hcs.SRegion, args *SnapshotIdOptions) error {
		ret, err := cli.GetSnapshot(args.ID)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	type SnapshotCreateOptions struct {
		DiskId string `help:"Disk ID"`
		Name   string `help:"Snapeshot Name"`
		Desc   string `help:"Snapshot Desc"`
	}

	shellutils.R(&SnapshotCreateOptions{}, "snapshot-create", "Create snapshot", func(cli *hcs.SRegion, args *SnapshotCreateOptions) error {
		_, err := cli.CreateSnapshot(args.DiskId, args.Name, args.Desc)
		return err
	})
}
