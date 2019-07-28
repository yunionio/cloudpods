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
	type DiskListOptions struct {
		StorageId string
		DiskIds   []string
		DiskType  string
	}
	shellutils.R(&DiskListOptions{}, "disk-list", "List disks", func(cli *zstack.SRegion, args *DiskListOptions) error {
		disks, err := cli.GetDisks(args.StorageId, args.DiskIds, args.DiskType)
		if err != nil {
			return err
		}
		printList(disks, len(disks), 0, 0, []string{})
		return nil
	})

	type DiskIdOptions struct {
		ID string
	}

	shellutils.R(&DiskIdOptions{}, "disk-show", "Show disk", func(cli *zstack.SRegion, args *DiskIdOptions) error {
		disk, err := cli.GetDisk(args.ID)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})

	shellutils.R(&DiskIdOptions{}, "disk-delete", "Delete disk", func(cli *zstack.SRegion, args *DiskIdOptions) error {
		return cli.DeleteDisk(args.ID)
	})

	type DiskCreateOptions struct {
		NAME        string
		Description string
		SizeGB      int
		HostId      string
		PoolId      string
		STORAGE_ID  string
	}

	shellutils.R(&DiskCreateOptions{}, "disk-create", "Create disk", func(cli *zstack.SRegion, args *DiskCreateOptions) error {
		disk, err := cli.CreateDisk(args.NAME, args.STORAGE_ID, args.HostId, args.PoolId, args.SizeGB, args.Description)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})

	type DiskResize struct {
		ID     string
		SIZEGB int64
	}

	shellutils.R(&DiskResize{}, "disk-resize", "Resize disk", func(cli *zstack.SRegion, args *DiskResize) error {
		return cli.ResizeDisk(args.ID, args.SIZEGB*1024)
	})

}
