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
	type CcnListOption struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&CcnListOption{}, "ccn-list", "List cloud connect network", func(cli *qcloud.SRegion, args *CcnListOption) error {
		vpcs, total, err := cli.DescribeCcns(nil, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(vpcs, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type CcnShowOption struct {
		CCNID string
	}
	shellutils.R(&CcnShowOption{}, "ccn-show", "show cloud connect network", func(cli *qcloud.SRegion, args *CcnShowOption) error {
		ccn, err := cli.GetCcnById(args.CCNID)
		if err != nil {
			return err
		}
		printObject(ccn)
		return nil
	})

	type CcnChildListOption struct {
		CCNID  string
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&CcnChildListOption{}, "ccn-child-list", "List cloud connect network attatched instance", func(cli *qcloud.SRegion, args *CcnChildListOption) error {
		vpcs, total, err := cli.DescribeCcnAttachedInstances(args.CCNID, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(vpcs, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type CenCreateOptions struct {
		Name        string
		Description string
	}
	shellutils.R(&CenCreateOptions{}, "ccn-create", "create cloud connect network", func(cli *qcloud.SRegion, args *CenCreateOptions) error {
		opts := cloudprovider.SInterVpcNetworkCreateOptions{}
		opts.Name = args.Name
		opts.Desc = args.Description
		ccnId, err := cli.CreateCcn(&opts)
		if err != nil {
			return err
		}
		print(ccnId)
		return nil
	})

	type CenDeleteOptions struct {
		ID string
	}
	shellutils.R(&CenDeleteOptions{}, "ccn-delete", "delete cloud connect network", func(cli *qcloud.SRegion, args *CenDeleteOptions) error {
		err := cli.DeleteCcn(args.ID)
		if err != nil {
			return err
		}
		return nil
	})

	type CenAddVpcOptions struct {
		ID          string
		OwnerId     string
		VpcId       string
		VpcRegionId string
	}
	shellutils.R(&CenAddVpcOptions{}, "ccn-add-vpc", "add vpc to cloud connect network attatched instance", func(cli *qcloud.SRegion, args *CenAddVpcOptions) error {

		instance := qcloud.SCcnAttachInstanceInput{
			InstanceType:   "VPC",
			InstanceId:     args.VpcId,
			InstanceRegion: args.VpcRegionId,
		}
		err := cli.AttachCcnInstances(args.ID, args.OwnerId, []qcloud.SCcnAttachInstanceInput{instance})
		if err != nil {
			return err
		}
		return nil
	})

	type CenRemoveVpcOptions struct {
		ID          string
		VpcId       string
		VpcRegionId string
	}
	shellutils.R(&CenAddVpcOptions{}, "ccn-remove-vpc", "remove vpc to cloud connect network attatched instance", func(cli *qcloud.SRegion, args *CenAddVpcOptions) error {
		instance := qcloud.SCcnAttachInstanceInput{
			InstanceType:   "VPC",
			InstanceId:     args.VpcId,
			InstanceRegion: args.VpcRegionId,
		}
		err := cli.DetachCcnInstances(args.ID, []qcloud.SCcnAttachInstanceInput{instance})
		if err != nil {
			return err
		}
		return nil
	})
}
