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
	"yunion.io/x/onecloud/pkg/multicloud/zstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SnapshotListOptions struct {
		SnapshotId string
		DiskId     string
	}
	shellutils.R(&SnapshotListOptions{}, "snapshot-list", "List snapshots", func(cli *zstack.SRegion, args *SnapshotListOptions) error {
		snapshots, err := cli.GetSnapshots(args.SnapshotId, args.DiskId)
		if err != nil {
			return err
		}
		printList(snapshots, len(snapshots), 0, 0, []string{})
		return nil
	})

	type SnapshoDeleteOptions struct {
		ID string
	}

	shellutils.R(&SnapshoDeleteOptions{}, "snapshot-delete", "Delete snapshot", func(cli *zstack.SRegion, args *SnapshoDeleteOptions) error {
		return cli.DeleteSnapshot(args.ID)
	})

	type SnapshoCreateOptions struct {
		DISKID string
		NAME   string
		Desc   string
	}

	shellutils.R(&SnapshoCreateOptions{}, "snapshot-create", "Create snapshot", func(cli *zstack.SRegion, args *SnapshoCreateOptions) error {
		snapshot, err := cli.CreateSnapshot(args.NAME, args.DISKID, args.Desc)
		if err != nil {
			return err
		}
		printObject(snapshot)
		return nil
	})
}
