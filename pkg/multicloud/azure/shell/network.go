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
	"yunion.io/x/onecloud/pkg/multicloud/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type NetworkListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&NetworkListOptions{}, "network-list", "List networks", func(cli *azure.SRegion, args *NetworkListOptions) error {
		if vpcs, err := cli.GetIVpcs(); err != nil {
			return nil
		} else {
			networks := make([]azure.SNetwork, 0)
			for _, _vpc := range vpcs {
				vpc := _vpc.(*azure.SVpc)
				if _networks := vpc.GetNetworks(); len(_networks) > 0 {
					networks = append(networks, _networks...)
				}

			}
			printList(networks, len(networks), args.Offset, args.Limit, []string{})
		}
		return nil
	})

	type NetworkInterfaceListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}

	shellutils.R(&NetworkInterfaceListOptions{}, "network-interface-list", "List network interface", func(cli *azure.SRegion, args *NetworkInterfaceListOptions) error {
		if interfaces, err := cli.GetNetworkInterfaces(); err != nil {
			return err
		} else {
			printList(interfaces, len(interfaces), args.Offset, args.Limit, []string{})
		}
		return nil
	})

	type NetworkInterfaceOptions struct {
		ID string `help:"Network ineterface ID"`
	}

	shellutils.R(&NetworkInterfaceOptions{}, "network-interface-show", "Show network interface", func(cli *azure.SRegion, args *NetworkInterfaceOptions) error {
		if networkInterface, err := cli.GetNetworkInterfaceDetail(args.ID); err != nil {
			return err
		} else {
			printObject(networkInterface)
			return nil
		}
	})

	type NetworkInterfaceCreateOptions struct {
		ResourceGroup string `help":"ResourceGroup Name"`
		NAME          string `help:"Nic interface name"`
		IP            string `help:"Nic private ip address"`
		NETWORK       string `help:"Netowrk ID"`
		SecurityGroup string `help:"SecurityGroup ID"`
	}

	shellutils.R(&NetworkInterfaceCreateOptions{}, "network-interface-create", "Create network interface", func(cli *azure.SRegion, args *NetworkInterfaceCreateOptions) error {
		nic, err := cli.CreateNetworkInterface(args.ResourceGroup, args.NAME, args.IP, args.NETWORK, args.SecurityGroup)
		if err != nil {
			return err
		}
		printObject(nic)
		return nil
	})

}
