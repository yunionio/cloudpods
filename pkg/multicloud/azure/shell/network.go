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
		VPC    string `help:"Vpc Id"`
		Limit  int    `help:"page size"`
		Offset int    `help:"page offset"`
	}
	shellutils.R(&NetworkListOptions{}, "network-list", "List networks", func(cli *azure.SRegion, args *NetworkListOptions) error {
		vpc, err := cli.GetVpc(args.VPC)
		if err != nil {
			return err
		}
		networks := vpc.GetNetworks()
		printList(networks, len(networks), args.Offset, args.Limit, []string{})
		return nil
	})

	type NetworkCreateOptions struct {
		VPC  string
		NAME string
		CIDR string
	}

	shellutils.R(&NetworkCreateOptions{}, "network-create", "Create network", func(cli *azure.SRegion, args *NetworkCreateOptions) error {
		network, err := cli.CreateNetwork(args.VPC, args.NAME, args.CIDR, "")
		if err != nil {
			return err
		}
		printObject(network)
		return nil
	})

	type NetworkInterfaceListOptions struct {
	}

	shellutils.R(&NetworkInterfaceListOptions{}, "network-interface-list", "List network interface", func(cli *azure.SRegion, args *NetworkInterfaceListOptions) error {
		nics, err := cli.GetNetworkInterfaces()
		if err != nil {
			return err
		}
		printList(nics, len(nics), 0, 0, []string{})
		return nil
	})

	type NetworkInterfaceOptions struct {
		ID string `help:"Network ineterface ID"`
	}

	shellutils.R(&NetworkInterfaceOptions{}, "network-interface-show", "Show network interface", func(cli *azure.SRegion, args *NetworkInterfaceOptions) error {
		nic, err := cli.GetNetworkInterface(args.ID)
		if err != nil {
			return err
		}
		printObject(nic)
		return nil
	})

	type NetworkInterfaceCreateOptions struct {
		ResourceGroup string `help:"ResourceGroup Name"`
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
