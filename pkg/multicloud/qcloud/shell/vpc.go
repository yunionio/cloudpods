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
	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VpcListOptions struct {
		Ids []string
	}
	shellutils.R(&VpcListOptions{}, "vpc-list", "List vpcs", func(cli *qcloud.SRegion, args *VpcListOptions) error {
		vpcs, err := cli.GetVpcs(args.Ids)
		if err != nil {
			return err
		}
		printList(vpcs, 0, 0, 0, []string{})
		return nil
	})

	type VpcCreateOptions struct {
		NAME string `help:"Name for vpc"`
		CIDR string `help:"Cidr for vpc" choices:"10.0.0.0/16|172.16.0.0/12|192.168.0.0/16"`
	}
	shellutils.R(&VpcCreateOptions{}, "vpc-create", "Create vpc", func(cli *qcloud.SRegion, args *VpcCreateOptions) error {
		opts := &cloudprovider.VpcCreateOptions{
			NAME: args.NAME,
			CIDR: args.CIDR,
		}
		vpc, err := cli.CreateIVpc(opts)
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

	type VpcPCListOption struct {
		VPCID  string
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&VpcPCListOption{}, "vpcPC-list", "List vpc peering connections", func(cli *qcloud.SRegion, args *VpcPCListOption) error {
		vpcs, total, err := cli.DescribeVpcPeeringConnections(args.VPCID, "", args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(vpcs, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type VpcPCShowOption struct {
		VPCPCID string
	}
	shellutils.R(&VpcPCShowOption{}, "vpcPC-show", "show vpc peering connection", func(cli *qcloud.SRegion, args *VpcPCShowOption) error {
		vpcPC, err := cli.GetVpcPeeringConnectionbyId(args.VPCPCID)
		if err != nil {
			return err
		}
		printObject(vpcPC)
		return nil
	})

	type VpcPCCreateOPtion struct {
		NAME         string
		VPCID        string
		PEERVPCID    string
		PEERREGIONID string
		PEEROWNERID  string
		Bandwidth    int
	}
	shellutils.R(&VpcPCCreateOPtion{}, "vpcPC-create", "create vpc peering connection", func(cli *qcloud.SRegion, args *VpcPCCreateOPtion) error {
		opts := cloudprovider.VpcPeeringConnectionCreateOptions{}
		opts.Name = args.NAME
		opts.PeerVpcId = args.PEERVPCID
		opts.PeerAccountId = args.PEEROWNERID
		opts.PeerRegionId = args.PEERREGIONID
		vpcPCId, err := cli.CreateVpcPeeringConnection(args.VPCID, &opts)
		if err != nil {
			return err
		}
		printObject(vpcPCId)
		return nil
	})

	shellutils.R(&VpcPCCreateOPtion{}, "vpcPC-createEx", "create Ex vpc peering connection", func(cli *qcloud.SRegion, args *VpcPCCreateOPtion) error {
		opts := cloudprovider.VpcPeeringConnectionCreateOptions{}
		opts.Name = args.NAME
		opts.PeerVpcId = args.PEERVPCID
		opts.PeerAccountId = args.PEEROWNERID
		opts.PeerRegionId = args.PEERREGIONID
		opts.Bandwidth = args.Bandwidth
		taskId, err := cli.CreateVpcPeeringConnectionEx(args.VPCID, &opts)
		if err != nil {
			return err
		}
		printObject(taskId)
		return nil
	})

	type VpcPeeringAcceptOPtion struct {
		VPCPEERINGID string
	}
	shellutils.R(&VpcPeeringAcceptOPtion{}, "vpcPC-accept", "Accept vpcPeering", func(cli *qcloud.SRegion, args *VpcPeeringAcceptOPtion) error {
		err := cli.AcceptVpcPeeringConnection(args.VPCPEERINGID)
		if err != nil {
			return err
		}
		return nil
	})

	shellutils.R(&VpcPeeringAcceptOPtion{}, "vpcPC-acceptEx", "Accept Ex vpcPeering", func(cli *qcloud.SRegion, args *VpcPeeringAcceptOPtion) error {
		_, err := cli.AcceptVpcPeeringConnectionEx(args.VPCPEERINGID)
		if err != nil {
			return err
		}
		return nil
	})

	type VpcPeeringDeleteOPtion struct {
		VPCPEERINGID string
	}
	shellutils.R(&VpcPeeringAcceptOPtion{}, "vpcPC-delete", "Delete vpcPeering", func(cli *qcloud.SRegion, args *VpcPeeringAcceptOPtion) error {
		err := cli.DeleteVpcPeeringConnection(args.VPCPEERINGID)
		if err != nil {
			return err
		}
		return nil
	})

	shellutils.R(&VpcPeeringAcceptOPtion{}, "vpcPC-deleteEx", "Delete Ex vpcPeering", func(cli *qcloud.SRegion, args *VpcPeeringAcceptOPtion) error {
		status, err := cli.DeleteVpcPeeringConnectionEx(args.VPCPEERINGID)
		if err != nil {
			return err
		}
		println(status)
		return nil
	})

	type VpcTaskShowOption struct {
		TASKID string
	}
	shellutils.R(&VpcTaskShowOption{}, "vpcTask-show", "show vpc task", func(cli *qcloud.SRegion, args *VpcTaskShowOption) error {
		status, err := cli.DescribeVpcTaskResult(args.TASKID)
		if err != nil {
			return err
		}
		println(status)
		return nil
	})

}
