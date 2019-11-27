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
	"yunion.io/x/onecloud/pkg/multicloud/google"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SnapshotListOptions struct {
		Disk       string
		MaxResults int
		PageToken  string
	}
	shellutils.R(&SnapshotListOptions{}, "snapshot-list", "List snapshots", func(cli *google.SRegion, args *SnapshotListOptions) error {
		snapshots, err := cli.GetSnapshots(args.Disk, args.MaxResults, args.PageToken)
		if err != nil {
			return err
		}
		printList(snapshots, 0, 0, 0, nil)
		return nil
	})

	type SnapshotShowOptions struct {
		ID string
	}
	shellutils.R(&SnapshotShowOptions{}, "snapshot-show", "Show snapshot", func(cli *google.SRegion, args *SnapshotShowOptions) error {
		snapshot, err := cli.GetSnapshot(args.ID)
		if err != nil {
			return err
		}
		printObject(snapshot)
		return nil
	})

}
