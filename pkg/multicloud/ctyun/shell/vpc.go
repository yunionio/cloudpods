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
	type VpcListOptions struct {
	}
	shellutils.R(&VpcListOptions{}, "vpc-list", "List vpcs", func(cli *ctyun.SRegion, args *VpcListOptions) error {
		vpcs, e := cli.GetVpcs()
		if e != nil {
			return e
		}
		printList(vpcs, 0, 0, 0, nil)
		return nil
	})

	type VpcCreateOptions struct {
		NAME string `help:"vpc name"`
		CIDR string `help:"10.0.0.0/8~10.255.255.0/24或者172.16.0.0/12 ~ 172.31.255.0/24或者192.168.0.0/16 ~ 192.168.255.0/24"`
	}
	shellutils.R(&VpcCreateOptions{}, "vpc-create", "Create vpc", func(cli *ctyun.SRegion, args *VpcCreateOptions) error {
		vpc, e := cli.CreateVpc(args.NAME, args.CIDR)
		if e != nil {
			return e
		}
		printObject(vpc)
		return nil
	})
}
