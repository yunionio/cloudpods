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
	"yunion.io/x/onecloud/pkg/multicloud/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type EipListOptions struct {
		Classic bool `help:"List classic eips"`
		Offset  int  `help:"List offset"`
		Limit   int  `help:"List limit"`
	}
	shellutils.R(&EipListOptions{}, "eip-list", "List eips", func(cli *azure.SRegion, args *EipListOptions) error {
		if args.Classic {
			eips, err := cli.GetClassicEips()
			if err != nil {
				return err
			}
			printList(eips, len(eips), args.Offset, args.Limit, []string{})
			return nil
		} else {
			eips, err := cli.GetEips()
			if err != nil {
				return err
			}
			printList(eips, len(eips), args.Offset, args.Limit, []string{})
			return nil
		}
	})

	type EipAllocateOptions struct {
		NAME          string `help:"Eip Name"`
		ResourceGroup string `help:"ResourceGroup Name"`
	}
	shellutils.R(&EipAllocateOptions{}, "eip-create", "Allocate an EIP", func(cli *azure.SRegion, args *EipAllocateOptions) error {
		if eip, err := cli.AllocateEIP(args.NAME, args.ResourceGroup); err != nil {
			return err
		} else {
			printObject(eip)
			return nil
		}
	})

	type EipReleaseOptions struct {
		ID string `help:"EIP allocation ID"`
	}
	shellutils.R(&EipReleaseOptions{}, "eip-delete", "Release an EIP", func(cli *azure.SRegion, args *EipReleaseOptions) error {
		return cli.DeallocateEIP(args.ID)
	})

	type EipShowOptions struct {
		ID string `help:"EIP ID"`
	}
	shellutils.R(&EipShowOptions{}, "eip-show", "Show an EIP", func(cli *azure.SRegion, args *EipShowOptions) error {
		if eip, err := cli.GetEip(args.ID); err != nil {
			return err
		} else {
			printObject(eip)
			return nil
		}
	})

	type EipAssociateOptions struct {
		ID       string `help:"EIP allocation ID"`
		INSTANCE string `help:"Instance ID"`
	}
	shellutils.R(&EipAssociateOptions{}, "eip-associate", "Associate an EIP", func(cli *azure.SRegion, args *EipAssociateOptions) error {
		err := cli.AssociateEip(args.ID, args.INSTANCE)
		return err
	})

	type EipDissociateOptions struct {
		ID string `help:"EIP allocation ID"`
	}

	shellutils.R(&EipDissociateOptions{}, "eip-dissociate", "Dissociate an EIP", func(cli *azure.SRegion, args *EipDissociateOptions) error {
		err := cli.DissociateEip(args.ID)
		return err
	})
}
