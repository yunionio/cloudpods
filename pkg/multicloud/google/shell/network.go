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
	"yunion.io/x/onecloud/pkg/multicloud/google"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type NetworkListOptions struct {
	}
	shellutils.R(&NetworkListOptions{}, "vpc-list", "List networks", func(cli *google.SRegion, args *NetworkListOptions) error {
		networks, err := cli.GetVpcs()
		if err != nil {
			return err
		}
		printList(networks, 0, 0, 0, nil)
		return nil
	})

	type NetworkIdOptions struct {
		ID string
	}

	shellutils.R(&NetworkIdOptions{}, "vpc-show", "Show network", func(cli *google.SRegion, args *NetworkIdOptions) error {
		vpc, err := cli.GetVpc(args.ID)
		if err != nil {
			return err
		}
		printObject(vpc)
		return nil
	})

	shellutils.R(&NetworkIdOptions{}, "vpc-delete", "Delete network", func(cli *google.SRegion, args *NetworkIdOptions) error {
		return cli.Delete(args.ID)
	})

	type NetworkCreateOptions struct {
		NAME string
		VPC  string
		CIDR string
		Desc string
	}

	shellutils.R(&NetworkCreateOptions{}, "vpc-create", "Create network", func(cli *google.SRegion, args *NetworkCreateOptions) error {
		network, err := cli.CreateVpc(args.NAME, args.VPC, args.CIDR, args.Desc)
		if err != nil {
			return err
		}
		printObject(network)
		return nil
	})

}
