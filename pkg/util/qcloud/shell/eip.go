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
	type EipListOptions struct {
		Eip        string `help:"EIP ID"`
		InstanceId string `help:"Instance Id"`
		Offset     int    `help:"List offset"`
		Limit      int    `help:"List limit"`
	}
	shellutils.R(&EipListOptions{}, "eip-list", "List eips", func(cli *qcloud.SRegion, args *EipListOptions) error {
		eips, total, err := cli.GetEips(args.Eip, args.InstanceId, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(eips, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type EipAllocateOptions struct {
		BANDWIDTH  int                        `help:"EIP bandwoidth"`
		NAME       string                     `help:"EIP Name"`
		ChargeType qcloud.TInternetChargeType `help:"EIP ChargeType"`
	}
	shellutils.R(&EipAllocateOptions{}, "eip-create", "Allocate an EIP", func(cli *qcloud.SRegion, args *EipAllocateOptions) error {
		eip, err := cli.AllocateEIP(args.NAME, args.BANDWIDTH, args.ChargeType)
		if err != nil {
			return err
		}
		printObject(eip)
		return nil
	})

	type EipReleaseOptions struct {
		ID string `help:"EIP allocation ID"`
	}
	shellutils.R(&EipReleaseOptions{}, "eip-delete", "Release an EIP", func(cli *qcloud.SRegion, args *EipReleaseOptions) error {
		return cli.DeallocateEIP(args.ID)
	})

	type EipShowOptions struct {
		ID string `help:"EIP ID"`
	}
	shellutils.R(&EipShowOptions{}, "eip-show", "Show an EIP", func(cli *qcloud.SRegion, args *EipShowOptions) error {
		eip, err := cli.GetEip(args.ID)
		if err != nil {
			return err
		}
		printObject(eip)
		return nil
	})

	type EipAssociateOptions struct {
		ID       string `help:"EIP allocation ID"`
		INSTANCE string `help:"Instance ID"`
	}
	shellutils.R(&EipAssociateOptions{}, "eip-associate", "Associate an EIP", func(cli *qcloud.SRegion, args *EipAssociateOptions) error {
		return cli.AssociateEip(args.ID, args.INSTANCE)
	})

	type EipDissociateOptions struct {
		ID string `help:"EIP allocation ID"`
	}

	shellutils.R(&EipDissociateOptions{}, "eip-dissociate", "Dissociate an EIP", func(cli *qcloud.SRegion, args *EipDissociateOptions) error {
		return cli.DissociateEip(args.ID)
	})
}
