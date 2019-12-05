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
	"yunion.io/x/onecloud/pkg/multicloud/ctyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VEipListOptions struct {
	}
	shellutils.R(&VEipListOptions{}, "eip-list", "List eips", func(cli *ctyun.SRegion, args *VEipListOptions) error {
		eips, e := cli.GetEips()
		if e != nil {
			return e
		}
		printList(eips, 0, 0, 0, nil)
		return nil
	})

	type EipCreateOptions struct {
		ZoneId    string `help:"zone id"`
		Name      string `help:"eip name"`
		Size      string `help:"size"`
		ShareType string `help:"share type" choice:"PER|WHOLE"`
	}
	shellutils.R(&EipCreateOptions{}, "eip-create", "Create eip", func(cli *ctyun.SRegion, args *EipCreateOptions) error {
		eip, e := cli.CreateEip(args.ZoneId, args.Name, args.Size, args.ShareType)
		if e != nil {
			return e
		}
		printObject(eip)
		return nil
	})
}
