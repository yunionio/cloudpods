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
	type DiskListOptions struct {
		ZONE        string
		StorageType string
		MaxResults  int
		PageToken   string
	}
	shellutils.R(&DiskListOptions{}, "disk-list", "List disks", func(cli *google.SRegion, args *DiskListOptions) error {
		disks, err := cli.GetDisks(args.ZONE, args.StorageType, args.MaxResults, args.PageToken)
		if err != nil {
			return err
		}
		printList(disks, 0, 0, 0, nil)
		return nil
	})

	type DiskShowOptions struct {
		ID string
	}
	shellutils.R(&DiskShowOptions{}, "disk-show", "Show disk", func(cli *google.SRegion, args *DiskShowOptions) error {
		disk, err := cli.GetDisk(args.ID)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})

	type RegionDiskListOptions struct {
		StorageType string
		MaxResults  int
		PageToken   string
	}
	shellutils.R(&RegionDiskListOptions{}, "region-disk-list", "List region disks", func(cli *google.SRegion, args *RegionDiskListOptions) error {
		disks, err := cli.GetRegionDisks(args.StorageType, args.MaxResults, args.PageToken)
		if err != nil {
			return err
		}
		printList(disks, 0, 0, 0, nil)
		return nil
	})

	type RegionDiskShowOptions struct {
		ID string
	}
	shellutils.R(&RegionDiskShowOptions{}, "region-disk-show", "Show region disk", func(cli *google.SRegion, args *RegionDiskShowOptions) error {
		disk, err := cli.GetRegionDisk(args.ID)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})

}
