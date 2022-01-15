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
	"yunion.io/x/onecloud/pkg/multicloud/nutanix"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VpcListOptions struct {
	}
	shellutils.R(&VpcListOptions{}, "vpc-list", "list vpc", func(cli *nutanix.SRegion, args *VpcListOptions) error {
		vpcs, err := cli.GetVpcs()
		if err != nil {
			return err
		}
		printList(vpcs, 0, 0, 0, []string{})
		return nil
	})

	type VpcIdOptions struct {
		ID string
	}

	shellutils.R(&VpcIdOptions{}, "vpc-show", "show vpc", func(cli *nutanix.SRegion, args *VpcIdOptions) error {
		vpc, err := cli.GetVpc(args.ID)
		if err != nil {
			return err
		}
		printObject(vpc)
		return nil
	})

	shellutils.R(&VpcIdOptions{}, "vpc-delete", "delete vpc", func(cli *nutanix.SRegion, args *VpcIdOptions) error {
		return cli.DeleteVpc(args.ID)
	})

	type VpcCreateOptions struct {
		Name string
		Desc string
		CIDR string
	}

	shellutils.R(&VpcCreateOptions{}, "vpc-create", "Create vpc", func(cli *nutanix.SRegion, args *VpcCreateOptions) error {
		opts := cloudprovider.VpcCreateOptions{
			NAME: args.Name,
			CIDR: args.CIDR,
			Desc: args.Desc,
		}
		vpc, err := cli.CreateVpc(&opts)
		if err != nil {
			return err
		}
		printObject(vpc)
		return nil
	})

}
