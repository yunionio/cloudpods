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
	type VpcListOptions struct {
		VpcIds []string
	}
	shellutils.R(&VpcListOptions{}, "vpc-list", "List vpcs", func(cli *aws.SRegion, args *VpcListOptions) error {
		vpcs, err := cli.GetVpcs(args.VpcIds)
		if err != nil {
			return err
		}
		printList(vpcs, 0, 0, 0, []string{})
		return nil
	})

	type VpcPeeringConnectionListOptions struct {
		AcceptId  string
		RequestId string
		Id        string
	}
	shellutils.R(&VpcPeeringConnectionListOptions{}, "vpc-peering-connection-list", "List vpcPeeringConnections", func(cli *aws.SRegion, args *VpcPeeringConnectionListOptions) error {
		vpcPCs, err := cli.DescribeVpcPeeringConnections(args.Id, args.AcceptId, args.RequestId)
		if err != nil {
			return err
		}
		printList(vpcPCs, len(vpcPCs), len(vpcPCs), len(vpcPCs), []string{})
		return nil
	})

	type VpcPeeringConnectionCreateOptions struct {
		NAME          string
		VPCID         string
		PEERVPCID     string
		PEERACCOUNTID string
		PEERREGIONID  string
		Desc          string
	}
	shellutils.R(&VpcPeeringConnectionCreateOptions{}, "vpc-peering-connection-create", "create vpcPeeringConnection", func(cli *aws.SRegion, args *VpcPeeringConnectionCreateOptions) error {
		opts := cloudprovider.VpcPeeringConnectionCreateOptions{}
		opts.Desc = args.Desc
		opts.Name = args.NAME
		opts.PeerAccountId = args.PEERACCOUNTID
		opts.PeerRegionId = args.PEERREGIONID
		opts.PeerVpcId = args.PEERVPCID

		vpcPC, err := cli.CreateVpcPeeringConnection(args.VPCID, &opts)
		if err != nil {
			return err
		}
		printObject(vpcPC)
		return nil
	})

	type VpcPeeringConnectionAcceptOptions struct {
		ID string
	}
	shellutils.R(&VpcPeeringConnectionAcceptOptions{}, "vpc-peering-connection-accept", "accept vpcPeeringConnection", func(cli *aws.SRegion, args *VpcPeeringConnectionAcceptOptions) error {
		vpcPC, err := cli.AcceptVpcPeeringConnection(args.ID)
		if err != nil {
			return err
		}
		printObject(vpcPC)
		return nil
	})

	type VpcPeeringConnectionDeleteOptions struct {
		ID string
	}
	shellutils.R(&VpcPeeringConnectionDeleteOptions{}, "vpc-peering-connection-delete", "delete vpcPeeringConnection", func(cli *aws.SRegion, args *VpcPeeringConnectionDeleteOptions) error {
		err := cli.DeleteVpcPeeringConnection(args.ID)
		if err != nil {
			return err
		}
		return nil
	})

	type VpcPeeringConnectionRouteDeleteOptions struct {
		ID string
	}
	shellutils.R(&VpcPeeringConnectionAcceptOptions{}, "vpc-peering-connection-route-delete", "delete vpc-peering-connection route", func(cli *aws.SRegion, args *VpcPeeringConnectionAcceptOptions) error {
		err := cli.DeleteVpcPeeringConnectionRoute(args.ID)
		if err != nil {
			printObject(err)
			return err
		}
		return nil
	})
}
