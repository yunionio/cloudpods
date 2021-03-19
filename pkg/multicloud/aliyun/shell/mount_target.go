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
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type MountTargetListOptions struct {
		ID         string `help:"Id"`
		DomainName string
		PageSize   int `help:"page size"`
		PageNum    int `help:"page num"`
	}
	shellutils.R(&MountTargetListOptions{}, "mount-target-list", "List MountTargets", func(cli *aliyun.SRegion, args *MountTargetListOptions) error {
		mounts, _, err := cli.GetMountTargets(args.ID, args.DomainName, args.PageSize, args.PageNum)
		if err != nil {
			return err
		}
		printList(mounts, 0, 0, 0, []string{})
		return nil
	})

	shellutils.R(&cloudprovider.SMountTargetCreateOptions{}, "mount-target-create", "Create Nas MountTarget", func(cli *aliyun.SRegion, args *cloudprovider.SMountTargetCreateOptions) error {
		mt, err := cli.CreateMountTarget(args)
		if err != nil {
			return err
		}
		printObject(mt)
		return nil
	})

	type MountTargetDeleteOptions struct {
		FILE_SYSTEM_ID string
		MOUT_POINT_ID  string
	}
	shellutils.R(&MountTargetDeleteOptions{}, "mount-target-delete", "Delete Nas MountTarget", func(cli *aliyun.SRegion, args *MountTargetDeleteOptions) error {
		return cli.DeleteMountTarget(args.FILE_SYSTEM_ID, args.MOUT_POINT_ID)
	})

}
