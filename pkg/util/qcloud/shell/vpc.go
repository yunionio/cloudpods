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
	"yunion.io/x/onecloud/pkg/util/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VpcListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&VpcListOptions{}, "vpc-list", "List vpcs", func(cli *qcloud.SRegion, args *VpcListOptions) error {
		vpcs, total, err := cli.GetVpcs(nil, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(vpcs, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type VpcCreateOptions struct {
		NAME string `help:"Name for vpc"`
		CIDR string `help:"Cidr for vpc" choices:"10.0.0.0/16|172.16.0.0/12|192.168.0.0/16"`
	}
	shellutils.R(&VpcCreateOptions{}, "vpc-create", "Create vpc", func(cli *qcloud.SRegion, args *VpcCreateOptions) error {
		vpc, err := cli.CreateIVpc(args.NAME, "", args.CIDR)
		if err != nil {
			return err
		}
		printObject(vpc)
		return nil
	})

	type VpcDeleteOptions struct {
		ID string `help:"VPC ID or Name"`
	}
	shellutils.R(&VpcDeleteOptions{}, "vpc-delete", "Delete vpc", func(cli *qcloud.SRegion, args *VpcDeleteOptions) error {
		return cli.DeleteVpc(args.ID)
	})
}
