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
	"yunion.io/x/onecloud/pkg/multicloud/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type EipListOptions struct {
		Id          string
		Addr        string
		AssociateId string
	}
	shellutils.R(&EipListOptions{}, "eip-list", "List eips", func(cli *aws.SRegion, args *EipListOptions) error {
		eips, err := cli.GetEips(args.Id, args.Addr, args.AssociateId)
		if err != nil {
			return err
		}
		printList(eips, 0, 0, 0, []string{})
		return nil
	})

	type EipAllocateOptions struct {
		Name string
	}
	shellutils.R(&EipAllocateOptions{}, "eip-create", "Allocate an EIP", func(cli *aws.SRegion, args *EipAllocateOptions) error {
		opts := cloudprovider.SEip{Name: args.Name}
		eip, err := cli.AllocateEIP(&opts)
		if err != nil {
			return err
		}
		printObject(eip)
		return nil
	})

	type EipReleaseOptions struct {
		ID string `help:"EIP allocation ID"`
	}
	shellutils.R(&EipReleaseOptions{}, "eip-delete", "Release an EIP", func(cli *aws.SRegion, args *EipReleaseOptions) error {
		err := cli.DeallocateEIP(args.ID)
		return err
	})

	type EipAssociateOptions struct {
		ID       string `help:"EIP allocation ID"`
		INSTANCE string `help:"Instance ID"`
	}
	shellutils.R(&EipAssociateOptions{}, "eip-associate", "Associate an EIP", func(cli *aws.SRegion, args *EipAssociateOptions) error {
		err := cli.AssociateEip(args.ID, args.INSTANCE)
		return err
	})

	type EipDissociateOptions struct {
		INSTANCE string `help:"Instance ID"`
	}

	shellutils.R(&EipDissociateOptions{}, "eip-dissociate", "Dissociate an EIP", func(cli *aws.SRegion, args *EipDissociateOptions) error {
		err := cli.DissociateEip(args.INSTANCE)
		return err
	})
}
