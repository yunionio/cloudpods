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
	"context"

	"yunion.io/x/onecloud/pkg/multicloud/ctyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VDiskListOptions struct {
	}
	shellutils.R(&VDiskListOptions{}, "disk-list", "List disks", func(cli *ctyun.SRegion, args *VDiskListOptions) error {
		disks, e := cli.GetDisks()
		if e != nil {
			return e
		}
		printList(disks, 0, 0, 0, nil)
		return nil
	})

	type DiskCreateOptions struct {
		ZoneId   string `help:"zone id"`
		Name     string `help:"disk name"`
		DiskType string `help:"disk type" choice:"SSD|SAS|SATA"`
		Size     string `help:"disk size"`
	}
	shellutils.R(&DiskCreateOptions{}, "disk-create", "Create disk", func(cli *ctyun.SRegion, args *DiskCreateOptions) error {
		disk, e := cli.CreateDisk(args.ZoneId, args.Name, args.DiskType, args.Size)
		if e != nil {
			return e
		}
		printObject(disk)
		return nil
	})

	type DiskResizeOptions struct {
		DiskId string `help:"disk id"`
		Size   int64  `help:"disk size GB"`
	}
	shellutils.R(&DiskResizeOptions{}, "disk-resize", "Resize disk", func(cli *ctyun.SRegion, args *DiskResizeOptions) error {
		disk, err := cli.GetDisk(args.DiskId)
		if err != nil {
			return err
		}

		e := disk.Resize(context.Background(), args.Size*1024)
		if e != nil {
			return e
		}
		printObject(disk)
		return nil
	})

	type VDiskRestoreOptions struct {
		DiskId     string `help:"disk id"`
		SnapshotId string `help:"snapshot id"`
	}
	shellutils.R(&VDiskRestoreOptions{}, "disk-restore", "Restore disk", func(cli *ctyun.SRegion, args *VDiskRestoreOptions) error {
		disk, err := cli.GetDisk(args.DiskId)
		if err != nil {
			return err
		}

		_, e := disk.Reset(context.Background(), args.SnapshotId)
		if e != nil {
			return e
		}
		printObject(disk)
		return nil
	})
}
