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
	"fmt"

	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type NetworkListOptions struct {
		Zone  string
		VpcId string
		Ids   []string
	}
	shellutils.R(&NetworkListOptions{}, "network-list", "List networks", func(cli *qcloud.SRegion, args *NetworkListOptions) error {
		networks, e := cli.GetNetworks(args.Ids, args.VpcId, args.Zone)
		if e != nil {
			return e
		}
		printList(networks, 0, 0, 0, []string{})
		return nil
	})

	type NetworkOptions struct {
		ID string `help:"Network ID"`
	}
	shellutils.R(&NetworkOptions{}, "network-delete", "Delete network", func(cli *qcloud.SRegion, args *NetworkOptions) error {
		return cli.DeleteNetwork(args.ID)
	})

	shellutils.R(&NetworkOptions{}, "network-show", "Show network", func(cli *qcloud.SRegion, args *NetworkOptions) error {
		network, err := cli.GetNetwork(args.ID)
		if err != nil {
			return err
		}
		printObject(network)
		return nil
	})

	type NetworkCreateOptions struct {
		ZONE string `help:"Zone ID"`
		VPC  string `help:"VPC ID"`
		CIDR string `help:"Network CIDR"`
		NAME string `help:"Network Name"`
	}
	shellutils.R(&NetworkCreateOptions{}, "network-create", "Create network", func(cli *qcloud.SRegion, args *NetworkCreateOptions) error {
		networkId, err := cli.CreateNetwork(args.ZONE, args.VPC, args.NAME, args.CIDR, "")
		if err != nil {
			return err
		}
		fmt.Println(networkId)
		return nil
	})
}
