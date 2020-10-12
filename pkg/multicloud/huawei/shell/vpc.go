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
	"yunion.io/x/onecloud/pkg/multicloud/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VpcListOptions struct {
	}
	shellutils.R(&VpcListOptions{}, "vpc-list", "List vpcs", func(cli *huawei.SRegion, args *VpcListOptions) error {
		vpcs, e := cli.GetVpcs()
		if e != nil {
			return e
		}
		printList(vpcs, 0, 0, 0, nil)
		return nil
	})

	type VpcCreateOptions struct {
		NAME string
		CIDR string
		Desc string
	}

	shellutils.R(&VpcCreateOptions{}, "vpc-create", "Create vpc", func(cli *huawei.SRegion, args *VpcCreateOptions) error {
		vpc, err := cli.CreateVpc(args.NAME, args.CIDR, args.Desc)
		if err != nil {
			return err
		}
		printObject(vpc)
		return nil
	})

	type VpcIdOption struct {
		ID string
	}

	shellutils.R(&VpcIdOption{}, "vpc-delete", "Delete vpc", func(cli *huawei.SRegion, args *VpcIdOption) error {
		return cli.DeleteVpc(args.ID)
	})

	type VpcPeeringListOPtion struct {
		VPCID string
	}
	shellutils.R(&VpcPeeringListOPtion{}, "vpcPeering-list", "List vpcPeering", func(cli *huawei.SRegion, args *VpcPeeringListOPtion) error {
		vpcPeerings, err := cli.GetVpcPeerings(args.VPCID)
		if err != nil {
			return err
		}
		printList(vpcPeerings, 0, 0, 0, nil)
		return nil
	})

	type VpcPeeringShowOPtion struct {
		VPCPEERINGID string
	}
	shellutils.R(&VpcPeeringShowOPtion{}, "vpcPeering-show", "show vpcPeering", func(cli *huawei.SRegion, args *VpcPeeringShowOPtion) error {
		vpcPeering, err := cli.GetVpcPeering(args.VPCPEERINGID)
		if err != nil {
			return err
		}
		printObject(vpcPeering)
		return nil
	})

	type VpcPeeringCreateOPtion struct {
		NAME        string
		VPCID       string
		PEERVPCID   string
		PEEROWNERID string
	}
	shellutils.R(&VpcPeeringCreateOPtion{}, "vpcPeering-create", "create vpcPeering", func(cli *huawei.SRegion, args *VpcPeeringCreateOPtion) error {
		opts := cloudprovider.VpcPeeringConnectionCreateOptions{}
		opts.Name = args.NAME
		opts.PeerVpcId = args.PEERVPCID
		opts.PeerAccountId = args.PEEROWNERID
		vpcPeering, err := cli.CreateVpcPeering(args.VPCID, &opts)
		if err != nil {
			return err
		}
		printObject(vpcPeering)
		return nil
	})

	type VpcPeeringAcceptOPtion struct {
		VPCPEERINGID string
	}
	shellutils.R(&VpcPeeringAcceptOPtion{}, "vpcPeering-accept", "Accept vpcPeering", func(cli *huawei.SRegion, args *VpcPeeringAcceptOPtion) error {
		err := cli.AcceptVpcPeering(args.VPCPEERINGID)
		if err != nil {
			return err
		}
		return nil
	})

	type VpcPeeringDeleteOPtion struct {
		VPCPEERINGID string
	}
	shellutils.R(&VpcPeeringDeleteOPtion{}, "vpcPeering-delete", "Delete vpcPeering", func(cli *huawei.SRegion, args *VpcPeeringDeleteOPtion) error {
		err := cli.DeleteVpcPeering(args.VPCPEERINGID)
		if err != nil {
			return err
		}
		return nil
	})
}
