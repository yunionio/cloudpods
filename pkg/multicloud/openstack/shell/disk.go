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
	type DiskListOptions struct {
		Category    string `help:"Storage type for disk"`
		BackendName string `help:"Storage backend eg:lvm, rbd"`
	}
	shellutils.R(&DiskListOptions{}, "disk-list", "List disks", func(cli *openstack.SRegion, args *DiskListOptions) error {
		disks, err := cli.GetDisks(args.Category, args.BackendName)
		if err != nil {
			return err
		}
		printList(disks, 0, 0, 0, []string{})
		return nil
	})

	type DiskOptions struct {
		ID string `help:"ID of disk"`
	}

	shellutils.R(&DiskOptions{}, "disk-show", "Show disk", func(cli *openstack.SRegion, args *DiskOptions) error {
		disk, err := cli.GetDisk(args.ID)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})

	shellutils.R(&DiskOptions{}, "disk-delete", "Delete disk", func(cli *openstack.SRegion, args *DiskOptions) error {
		return cli.DeleteDisk(args.ID)
	})

	type DiskCreateOptions struct {
		ImageRef  string `help:"ImageRef"`
		CATEGORY  string `help:"Disk category"`
		NAME      string `help:"Disk Name"`
		SIZE      int    `help:"Disk Size GB"`
		Desc      string `help:"Description of disk"`
		ProjectId string `help:"ProjectId"`
	}
	shellutils.R(&DiskCreateOptions{}, "disk-create", "Create disk", func(cli *openstack.SRegion, args *DiskCreateOptions) error {
		disk, err := cli.CreateDisk(args.ImageRef, args.CATEGORY, args.NAME, args.SIZE, args.Desc, args.ProjectId)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})

	type DiskResetOptions struct {
		DISK     string `help:"ID of disk"`
		SNAPSHOT string `help:"ID of snapshot"`
	}

	shellutils.R(&DiskResetOptions{}, "disk-reset", "Reset disk", func(cli *openstack.SRegion, args *DiskResetOptions) error {
		return cli.ResetDisk(args.DISK, args.SNAPSHOT)
	})

	type DiskResizeOptions struct {
		DISK string `help:"ID of disk"`
		SIZE int64  `help:"Disk size GB"`
	}

	shellutils.R(&DiskResizeOptions{}, "disk-resize", "Resize disk", func(cli *openstack.SRegion, args *DiskResizeOptions) error {
		return cli.ResizeDisk(args.DISK, args.SIZE*1024)
	})

}
