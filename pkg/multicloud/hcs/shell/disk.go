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
	type DiskListOptions struct {
		Zone string `help:"Zone ID"`
	}
	shellutils.R(&DiskListOptions{}, "disk-list", "List disks", func(cli *hcs.SRegion, args *DiskListOptions) error {
		disks, e := cli.GetDisks(args.Zone)
		if e != nil {
			return e
		}
		printList(disks, 0, 0, 0, nil)
		return nil
	})

	type DiskCreateOptions struct {
		ZONE         string
		STORAGE_TYPE string
		NAME         string
		SIZE_GB      int
		SnapshotId   string
		Desc         string
		ProjectId    string
	}

	shellutils.R(&DiskCreateOptions{}, "disk-create", "Create disk", func(cli *hcs.SRegion, args *DiskCreateOptions) error {
		disk, e := cli.CreateDisk(args.ZONE, args.STORAGE_TYPE, args.NAME, args.SIZE_GB, args.SnapshotId, args.Desc, args.ProjectId)
		if e != nil {
			return e
		}
		printObject(disk)
		return nil
	})

	type DiskIdOptions struct {
		ID string `help:"Disk ID"`
	}

	shellutils.R(&DiskIdOptions{}, "disk-delete", "Delete disks", func(cli *hcs.SRegion, args *DiskIdOptions) error {
		e := cli.DeleteDisk(args.ID)
		if e != nil {
			return e
		}
		return nil
	})

	shellutils.R(&DiskIdOptions{}, "disk-show", "Show disks", func(cli *hcs.SRegion, args *DiskIdOptions) error {
		disk, e := cli.GetDisk(args.ID)
		if e != nil {
			return e
		}
		printObject(disk)
		return nil
	})

	shellutils.R(&DiskListOptions{}, "disk-types", "List disk types", func(cli *hcs.SRegion, args *DiskListOptions) error {
		ret, e := cli.GetDiskTypes()
		if e != nil {
			return e
		}
		printList(ret, 0, 0, 0, nil)
		return nil
	})
}
