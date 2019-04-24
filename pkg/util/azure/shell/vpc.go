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
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VpcListOptions struct {
		Classic bool `help:"List classic vpcs"`
		Limit   int  `help:"page size"`
		Offset  int  `help:"page offset"`
	}
	shellutils.R(&VpcListOptions{}, "vpc-list", "List vpcs", func(cli *azure.SRegion, args *VpcListOptions) error {
		vpcs, err := cli.GetIVpcs()
		if err != nil {
			return err
		}
		printList(vpcs, len(vpcs), args.Offset, args.Limit, []string{})
		return nil
	})

	type VpcOptions struct {
		ID string `help:"vpc ID"`
	}

	shellutils.R(&VpcOptions{}, "vpc-show", "Show vpc details", func(cli *azure.SRegion, args *VpcOptions) error {
		vpc, err := cli.GetVpc(args.ID)
		if err != nil {
			return err
		}
		printObject(vpc)
		return nil
	})

	shellutils.R(&VpcOptions{}, "vpc-delete", "Delete vpc", func(cli *azure.SRegion, args *VpcOptions) error {
		return cli.DeleteVpc(args.ID)
	})

	type VpcCreateOptions struct {
		NAME string `help:"vpc Name"`
		CIDR string `help:"vpc cidr"`
		Desc string `help:"vpc description"`
	}

	shellutils.R(&VpcCreateOptions{}, "vpc-create", "Create vpc", func(cli *azure.SRegion, args *VpcCreateOptions) error {
		vpc, err := cli.CreateIVpc(args.NAME, args.Desc, args.CIDR)
		if err != nil {
			return err
		}
		printObject(vpc)
		return nil
	})
}
