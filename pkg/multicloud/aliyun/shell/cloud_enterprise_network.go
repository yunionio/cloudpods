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
// PageSizeations under the License.

package shell

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type CenListOptions struct {
		PageSize   int `help:"page size"`
		PageNumber int `help:"page PageNumber"`
	}
	shellutils.R(&CenListOptions{}, "cen-list", "List cloud enterprise network", func(cli *aliyun.SRegion, args *CenListOptions) error {
		scens, e := cli.GetClient().DescribeCens(args.PageNumber, args.PageSize)
		if e != nil {
			return e
		}
		printList(scens.Cens.Cen, scens.TotalCount, args.PageNumber, args.PageSize, []string{})
		return nil
	})

	type CenChildListOptions struct {
		ID         string `help:"cen id"`
		PageSize   int    `help:"page size"`
		PageNumber int    `help:"page PageNumber"`
	}
	shellutils.R(&CenChildListOptions{}, "cen-child-list", "List cloud enterprise network childs", func(cli *aliyun.SRegion, args *CenChildListOptions) error {
		schilds, e := cli.GetClient().DescribeCenAttachedChildInstances(args.ID, args.PageNumber, args.PageSize)
		if e != nil {
			return e
		}
		printList(schilds.ChildInstances.ChildInstance, schilds.TotalCount, args.PageNumber, args.PageSize, []string{})
		return nil
	})

	type CenCreateOptions struct {
		Name        string
		Description string
	}
	shellutils.R(&CenCreateOptions{}, "cen-create", "Create cloud enterprise network", func(cli *aliyun.SRegion, args *CenCreateOptions) error {
		opts := cloudprovider.SInterVpcNetworkCreateOptions{}
		opts.Name = args.Name
		opts.Desc = args.Description
		scen, e := cli.GetClient().CreateCen(&opts)
		if e != nil {
			return e
		}
		print(scen)
		return nil
	})

	type CenDeleteOptions struct {
		ID string
	}
	shellutils.R(&CenDeleteOptions{}, "cen-delete", "delete cloud enterprise network", func(cli *aliyun.SRegion, args *CenDeleteOptions) error {
		e := cli.GetClient().DeleteCen(args.ID)
		if e != nil {
			return e
		}
		return nil
	})

	type CenAddVpcOptions struct {
		ID          string
		VpcId       string
		VpcRegionId string
	}
	shellutils.R(&CenAddVpcOptions{}, "cen-add-vpc", "add vpc to cloud enterprise network", func(cli *aliyun.SRegion, args *CenAddVpcOptions) error {
		instance := aliyun.SCenAttachInstanceInput{
			InstanceType:   "VPC",
			InstanceId:     args.VpcId,
			InstanceRegion: args.VpcRegionId,
		}

		e := cli.GetClient().AttachCenChildInstance(args.ID, instance)
		if e != nil {
			return e
		}
		return nil
	})

	type CenRemoveVpcOptions struct {
		ID          string
		VpcId       string
		VpcRegionId string
	}
	shellutils.R(&CenRemoveVpcOptions{}, "cen-remove-vpc", "remove vpc to cloud enterprise network", func(cli *aliyun.SRegion, args *CenRemoveVpcOptions) error {
		instance := aliyun.SCenAttachInstanceInput{
			InstanceType:   "VPC",
			InstanceId:     args.VpcId,
			InstanceRegion: args.VpcRegionId,
		}

		e := cli.GetClient().DetachCenChildInstance(args.ID, instance)
		if e != nil {
			return e
		}
		return nil
	})
}
