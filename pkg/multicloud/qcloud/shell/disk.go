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
	"fmt"

	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DiskListOptions struct {
		Instance string `help:"Instance ID"`
		Zone     string `help:"Zone ID"`
		Category string `help:"Disk category"`
		Offset   int    `help:"List offset"`
		Limit    int    `help:"List limit"`
	}
	shellutils.R(&DiskListOptions{}, "disk-list", "List disks", func(cli *qcloud.SRegion, args *DiskListOptions) error {
		disks, total, e := cli.GetDisks(args.Instance, args.Zone, args.Category, nil, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(disks, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type DiskOptions struct {
		ID string `help:"Disk ID"`
	}
	shellutils.R(&DiskOptions{}, "disk-delete", "Delete disks", func(cli *qcloud.SRegion, args *DiskOptions) error {
		e := cli.DeleteDisk(args.ID)
		if e != nil {
			return e
		}
		return nil
	})

	shellutils.R(&DiskOptions{}, "disk-show", "Show disk", func(cli *qcloud.SRegion, args *DiskOptions) error {
		disk, err := cli.GetDisk(args.ID)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})

	type DiskCreateOptions struct {
		ZONE      string `help:"Zone ID"`
		CATEGORY  string `help:"Disk category" choices:"CLOUD_BASIC|CLOUD_PREMIUM|CLOUD_SSD"`
		NAME      string `help:"Disk Name"`
		SIZE      int    `help:"Disk Size GB"`
		Desc      string `help:"Description"`
		ProjectId string `help:"Project Id"`
	}
	shellutils.R(&DiskCreateOptions{}, "disk-create", "Create disk", func(cli *qcloud.SRegion, args *DiskCreateOptions) error {
		diskId, err := cli.CreateDisk(args.ZONE, args.CATEGORY, args.NAME, args.SIZE, args.Desc, args.ProjectId)
		if err != nil {
			return err
		}
		fmt.Println(diskId)
		return nil
	})

	type DiskResizeOptions struct {
		ID   string `help:"Disk ID"`
		SIZE int64  `help:"Disk Size GB"`
	}
	shellutils.R(&DiskResizeOptions{}, "disk-resize", "Resize disk", func(cli *qcloud.SRegion, args *DiskResizeOptions) error {
		ctx := context.Background()
		return cli.ResizeDisk(ctx, args.ID, args.SIZE)
	})

	type DiskResetOptions struct {
		ID       string `help:"Disk ID"`
		SNAPSHOT string `help:"Snapshot ID"`
	}
	shellutils.R(&DiskResetOptions{}, "disk-reset", "Reset disk", func(cli *qcloud.SRegion, args *DiskResetOptions) error {
		return cli.ResetDisk(args.ID, args.SNAPSHOT)
	})
}
