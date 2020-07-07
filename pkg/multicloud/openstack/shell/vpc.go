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
	type VpcListOptions struct {
		TenantId string
	}
	shellutils.R(&VpcListOptions{}, "vpc-list", "List vpcs", func(cli *openstack.SRegion, args *VpcListOptions) error {
		vpcs, err := cli.GetVpcs(args.TenantId)
		if err != nil {
			return err
		}
		printList(vpcs, 0, 0, 0, nil)
		return nil
	})

	type VpcIdOptions struct {
		ID string `help:"ID of vpc"`
	}
	shellutils.R(&VpcIdOptions{}, "vpc-show", "Show vpc", func(cli *openstack.SRegion, args *VpcIdOptions) error {
		vpc, err := cli.GetVpc(args.ID)
		if err != nil {
			return err
		}
		printObject(vpc)
		return nil
	})

	shellutils.R(&VpcIdOptions{}, "vpc-delete", "Delete vpc", func(cli *openstack.SRegion, args *VpcIdOptions) error {
		return cli.DeleteVpc(args.ID)
	})

	type VpcCreateOptions struct {
		NAME string
		Desc string
	}

	shellutils.R(&VpcCreateOptions{}, "vpc-create", "Create vpc", func(cli *openstack.SRegion, args *VpcCreateOptions) error {
		vpc, err := cli.CreateVpc(args.NAME, args.Desc)
		if err != nil {
			return err
		}
		printObject(vpc)
		return nil
	})
}
