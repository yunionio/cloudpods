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
	type VSwitchListOptions struct {
		Ids   []string
		VpcId string
	}
	shellutils.R(&VSwitchListOptions{}, "network-list", "List vswitches", func(cli *aws.SRegion, args *VSwitchListOptions) error {
		networks, err := cli.GetNetwroks(args.Ids, args.VpcId)
		if err != nil {
			return err
		}
		printList(networks, 0, 0, 0, []string{})
		return nil
	})

	type NetworkCreateOptions struct {
		NAME string
		ZONE string
		VPC  string
		CIDR string
		Desc string
	}

	shellutils.R(&NetworkCreateOptions{}, "network-create", "Create network", func(cli *aws.SRegion, args *NetworkCreateOptions) error {
		network, err := cli.CreateNetwork(args.ZONE, args.VPC, args.NAME, args.CIDR, args.Desc)
		if err != nil {
			return err
		}
		printObject(network)
		return nil
	})

}
