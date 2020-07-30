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
	"yunion.io/x/onecloud/pkg/multicloud/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type NetworkListOptions struct {
		VpcId string `help:"Vpc ID for filter network list"`
	}
	shellutils.R(&NetworkListOptions{}, "network-list", "List networks", func(cli *openstack.SRegion, args *NetworkListOptions) error {
		networks, err := cli.GetNetworks(args.VpcId)
		if err != nil {
			return err
		}
		printList(networks, 0, 0, 0, nil)
		return nil
	})

	type NetworkOptions struct {
		ID string `help:"Network ID"`
	}
	shellutils.R(&NetworkOptions{}, "network-show", "Show network", func(cli *openstack.SRegion, args *NetworkOptions) error {
		network, err := cli.GetNetwork(args.ID)
		if err != nil {
			return err
		}
		printObject(network)
		return nil
	})

	type NetworkCreateOptions struct {
		VPCID     string
		NAME      string
		CIDR      string
		Desc      string
		ProjectId string
	}

	shellutils.R(&NetworkCreateOptions{}, "network-create", "Create network", func(cli *openstack.SRegion, args *NetworkCreateOptions) error {
		network, err := cli.CreateNetwork(args.VPCID, args.ProjectId, args.NAME, args.CIDR, args.Desc)
		if err != nil {
			return err
		}
		printObject(network)
		return nil
	})
}
