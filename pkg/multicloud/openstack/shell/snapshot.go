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
	"yunion.io/x/onecloud/pkg/multicloud/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SnapshotListOptions struct {
		DiskId string `help:"Disk ID for filter snapshot"`
	}
	shellutils.R(&SnapshotListOptions{}, "snapshot-list", "List snapshots", func(cli *openstack.SRegion, args *SnapshotListOptions) error {
		snapshots, err := cli.GetSnapshots(args.DiskId)
		if err != nil {
			return err
		}
		printList(snapshots, 0, 0, 0, []string{})
		return nil
	})

	type SnapshotOptions struct {
		ID string `help:"ID of snapshot"`
	}

	shellutils.R(&SnapshotOptions{}, "snapshot-show", "Show snapshot", func(cli *openstack.SRegion, args *SnapshotOptions) error {
		snapshot, err := cli.GetISnapshotById(args.ID)
		if err != nil {
			return err
		}
		printObject(snapshot)
		return nil
	})

	shellutils.R(&SnapshotOptions{}, "snapshot-delete", "Delete snapshot", func(cli *openstack.SRegion, args *SnapshotOptions) error {
		return cli.DeleteSnapshot(args.ID)
	})

	type SnapshotCreateOptions struct {
		DISKID string `help:"Disk ID"`
		Name   string `help:"Disk Name"`
		Desc   string `help:"Disk description"`
	}

	shellutils.R(&SnapshotCreateOptions{}, "snapshot-create", "Create snapshot", func(cli *openstack.SRegion, args *SnapshotCreateOptions) error {
		snapshot, err := cli.CreateSnapshot(args.DISKID, args.Name, args.Desc)
		if err != nil {
			return err
		}
		printObject(snapshot)
		return nil
	})

}
