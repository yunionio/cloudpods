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
	"yunion.io/x/onecloud/pkg/multicloud/proxmox"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DiskListOptions struct {
		STORAGE_ID string
	}
	shellutils.R(&DiskListOptions{}, "disk-list", "list disks", func(cli *proxmox.SRegion, args *DiskListOptions) error {
		disks, err := cli.GetDisks(args.STORAGE_ID)
		if err != nil {
			return err
		}
		printList(disks, 0, 0, 0, []string{})
		return nil
	})

	type DiskIdOptions struct {
		ID string
	}

	shellutils.R(&DiskIdOptions{}, "disk-show", "show disk", func(cli *proxmox.SRegion, args *DiskIdOptions) error {
		ret, err := cli.GetDisk(args.ID)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

}
