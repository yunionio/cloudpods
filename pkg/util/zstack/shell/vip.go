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
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type VipListOptions struct {
		VipId     string
		NetworkId string
	}
	shellutils.R(&VipListOptions{}, "vip-list", "List vips", func(cli *zstack.SRegion, args *VipListOptions) error {
		vips, err := cli.GetVirtualIPs(args.NetworkId, args.VipId)
		if err != nil {
			return err
		}
		printList(vips, 0, 0, 0, []string{})
		return nil
	})

	type VipCreateOptions struct {
		NAME string
		Desc string
		Ip   string
		L3ID string
	}

	shellutils.R(&VipCreateOptions{}, "vip-create", "Create vip", func(cli *zstack.SRegion, args *VipCreateOptions) error {
		vip, err := cli.CreateVirtualIP(args.NAME, args.Desc, args.Ip, args.L3ID)
		if err != nil {
			return err
		}
		printObject(vip)
		return nil
	})

}
