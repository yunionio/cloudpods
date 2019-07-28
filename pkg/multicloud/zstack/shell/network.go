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
	"yunion.io/x/onecloud/pkg/multicloud/zstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type NetworkListOptions struct {
		ZoneId    string
		WireId    string
		VpcId     string
		NetworkId string
	}
	shellutils.R(&NetworkListOptions{}, "network-list", "List networks", func(cli *zstack.SRegion, args *NetworkListOptions) error {
		networks, err := cli.GetNetworks(args.ZoneId, args.WireId, args.VpcId, args.NetworkId)
		if err != nil {
			return err
		}
		printList(networks, len(networks), 0, 0, []string{})
		return nil
	})

	type NetworkDeleteOptions struct {
		ID string
	}

	shellutils.R(&NetworkDeleteOptions{}, "network-delete", "Delete network", func(cli *zstack.SRegion, args *NetworkDeleteOptions) error {
		return cli.DeleteNetwork(args.ID)
	})

	type NetworkCreateOptions struct {
		NAME string
		Desc string
		WIRE string
		CIDR string
	}

	shellutils.R(&NetworkCreateOptions{}, "network-create", "Create network", func(cli *zstack.SRegion, args *NetworkCreateOptions) error {
		network, err := cli.CreateNetwork(args.NAME, args.CIDR, args.WIRE, args.Desc)
		if err != nil {
			return err
		}
		printObject(network)
		return nil
	})

	type L3NetworkListOptions struct {
		ZoneId string
		WireId string
		Id     string
	}

	shellutils.R(&L3NetworkListOptions{}, "l3network-list", "List networks", func(cli *zstack.SRegion, args *L3NetworkListOptions) error {
		l3networks, err := cli.GetL3Networks(args.ZoneId, args.WireId, args.Id)
		if err != nil {
			return err
		}
		printList(l3networks, len(l3networks), 0, 0, []string{})
		return nil
	})

	type NetworkServicesOptions struct {
	}

	shellutils.R(&NetworkServicesOptions{}, "network-service-list", "List network services", func(cli *zstack.SRegion, args *NetworkServicesOptions) error {
		service, err := cli.GetNetworkServices()
		if err != nil {
			return err
		}
		printObject(service)
		return nil
	})

	type NetworkServiceProviderOptions struct {
		Type string
	}

	shellutils.R(&NetworkServiceProviderOptions{}, "network-service-provider-list", "List network service providers", func(cli *zstack.SRegion, args *NetworkServiceProviderOptions) error {
		providers, err := cli.GetNetworkServiceProviders(args.Type)
		if err != nil {
			return err
		}
		printList(providers, len(providers), 0, 0, []string{})
		return nil
	})

	type NetworkServiceAttachOptions struct {
		Services []string
		ID       string
	}

	shellutils.R(&NetworkServiceAttachOptions{}, "network-service-attach", "Attach network service to l3network", func(cli *zstack.SRegion, args *NetworkServiceAttachOptions) error {
		cli.AttachServiceForl3Network(args.ID, args.Services)
		return nil
	})

	type NetworkServiceRefOptions struct {
		L3Id string
		Type string
	}

	shellutils.R(&NetworkServiceRefOptions{}, "network-service-ref", "List network service ref", func(cli *zstack.SRegion, args *NetworkServiceRefOptions) error {
		refs, err := cli.GetNetworkServiceRef(args.L3Id, args.Type)
		if err != nil {
			return err
		}
		printList(refs, len(refs), 0, 0, []string{})
		return nil
	})

	type NetworkServiceRemoveOptions struct {
		L3ID    string
		SERVICE string `choices:"Userdata|DHCP|VipQos|HostRoute|Eip|SecurityGroup|DNS|SNAT"`
	}

	shellutils.R(&NetworkServiceRemoveOptions{}, "remove-network-service", "Remove l3network service", func(cli *zstack.SRegion, args *NetworkServiceRemoveOptions) error {
		return cli.RemoveNetworkService(args.L3ID, args.SERVICE)
	})

}
