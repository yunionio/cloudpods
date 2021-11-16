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
	"yunion.io/x/onecloud/pkg/multicloud/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type EipListOptions struct {
		InstanceId string
		EipId      string
		IpAddr     string
	}
	shellutils.R(&EipListOptions{}, "eip-list", "List eips", func(cli *aws.SRegion, args *EipListOptions) error {
		eips, e := cli.GetEips(args.InstanceId, args.EipId, args.IpAddr)
		if e != nil {
			return e
		}
		printList(eips, 0, 0, 0, []string{})
		return nil
	})

	type EipAllocateOptions struct {
		NAME       string
		DomainType string `choices:"vpc" defualt:"vpc"`
	}
	shellutils.R(&EipAllocateOptions{}, "eip-create", "Allocate an EIP", func(cli *aws.SRegion, args *EipAllocateOptions) error {
		eip, err := cli.AllocateEIP(args.NAME, args.DomainType)
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
	shellutils.R(&EipAssociateOptions{}, "eip-dissociate", "Dissociate an EIP", func(cli *aws.SRegion, args *EipAssociateOptions) error {
		err := cli.DissociateEip(args.ID, args.INSTANCE)
		return err
	})
}
