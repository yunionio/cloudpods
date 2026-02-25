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

package compute

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	compute_options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

func init() {

	cmd := shell.NewResourceCmd(&modules.Networks).WithContextManager(&modules.Wires)
	cmd.List(&compute_options.NetworkListOptions{})
	cmd.Create(&compute_options.NetworkCreateOptions{})
	cmd.Update(&compute_options.NetworkUpdateOptions{})
	cmd.Show(&compute_options.NetworkIdOptions{})
	cmd.Delete(&compute_options.NetworkIdOptions{})
	cmd.GetMetadata(&compute_options.NetworkIdOptions{})
	cmd.Perform("private", &compute_options.NetworkIdOptions{})
	cmd.Perform("public", &options.SharableResourcePublicOptions{})
	cmd.Perform("syncstatus", &compute_options.NetworkIdOptions{})
	cmd.Perform("sync", &compute_options.NetworkIdOptions{})
	cmd.Perform("purge", &compute_options.NetworkIdOptions{})
	cmd.Get("change-owner-candidate-domains", &compute_options.NetworkIdOptions{})
	cmd.Perform("set-class-metadata", &options.ResourceMetadataOptions{})
	cmd.Perform("switch-wire", &compute_options.NetworkSwitchWireOptions{})
	cmd.Perform("sync-additional-wires", &compute_options.NetworkSyncAdditionalWiresOptions{})
	cmd.Get("available-addresses", &compute_options.NetworkIdOptions{})

	type NetworkCreateOptions2 struct {
		Wire   string `help:"ID or Name of wire in which the network is created"`
		Vpc    string `help:"ID or Name of vpc in which the network is created"`
		Zone   string `help:"ID or Name of zone in which the network is created"`
		NAME   string `help:"Name of new network"`
		PREFIX string `help:"IPv4 prefix"`

		Prefix6 string `help:"IPv6 prefix"`

		BgpType        string `help:"Internet service provider name" positional:"false"`
		AssignPublicIp bool
		Desc           string `help:"Description" metavar:"DESCRIPTION"`
	}
	R(&NetworkCreateOptions2{}, "network-create2", "Create a virtual network", func(s *mcclient.ClientSession, args *NetworkCreateOptions2) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.PREFIX), "guest_ip_prefix")

		if len(args.Prefix6) > 0 {
			params.Add(jsonutils.NewString(args.Prefix6), "guest_ip6_prefix")
		}

		params.Set("assign_public_ip", jsonutils.NewBool(args.AssignPublicIp))
		if len(args.BgpType) > 0 {
			params.Add(jsonutils.NewString(args.BgpType), "bgp_type")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.Wire) > 0 {
			params.Add(jsonutils.NewString(args.Wire), "wire")
		} else if len(args.Vpc) > 0 {
			if len(args.Zone) > 0 {
				params.Add(jsonutils.NewString(args.Zone), "zone")
				params.Add(jsonutils.NewString(args.Vpc), "vpc")
			} else {
				return fmt.Errorf("Either wire or VPC/Zone must be provided")
			}
		} else {
			return fmt.Errorf("Either wire or VPC/Zone must be provided")
		}
		net, e := modules.Networks.Create(s, params)
		if e != nil {
			return e
		}
		printObject(net)
		return nil
	})

	type NetworkCreate3Options struct {
		compute_options.NetworkCreateOptions `start_ip->positional:"false" end_ip->positional:"false" net_mask->positional:"false"`
	}
	R(&NetworkCreate3Options{}, "network-create3", "Create a dual-stack virtual network", func(s *mcclient.ClientSession, args *NetworkCreate3Options) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		net, err := modules.Networks.Create(s, params)
		if err != nil {
			return err
		}
		printObject(net)
		return nil
	})

	type NetworkSplitOptions struct {
		NETWORK string `help:"ID or name of network to split"`
		IP      string `help:"Start ip of the split network"`
		Name    string `help:"Name of the new network"`
	}
	R(&NetworkSplitOptions{}, "network-split", "Split a network at the specified IP address", func(s *mcclient.ClientSession, args *NetworkSplitOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.IP), "split_ip")
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		net, err := modules.Networks.PerformAction(s, args.NETWORK, "split", params)
		if err != nil {
			return err
		}
		printObject(net)
		return nil
	})

	type NetworkMergeOptions struct {
		FROM   string `help:"ID or name of merge network from"`
		TARGET string `help:"ID or name of merge network target"`
	}
	R(&NetworkMergeOptions{}, "network-merge", "Merge two network to be one", func(s *mcclient.ClientSession, args *NetworkMergeOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.TARGET), "target")
		net, err := modules.Networks.PerformAction(s, args.FROM, "merge", params)
		if err != nil {
			return err
		}
		printObject(net)
		return nil
	})

	type NetworkStaticRoutesOptions struct {
		NETWORK string   `help:"ID or name of the network"`
		Net     []string `help:"destination network of static route"`
		Gw      []string `help:"gateway address for the static route"`
	}
	R(&NetworkStaticRoutesOptions{}, "network-set-static-routes", "Set static routes for a network", func(s *mcclient.ClientSession, args *NetworkStaticRoutesOptions) error {
		params := jsonutils.NewDict()
		if len(args.Net) > 0 && len(args.Gw) > 0 {
			if len(args.Net) != len(args.Gw) {
				return fmt.Errorf("Inconsistent network and gateway pairs")
			}
			routes := jsonutils.NewDict()
			for i := 0; i < len(args.Net); i += 1 {
				routes.Add(jsonutils.NewString(args.Gw[i]), args.Net[i])
			}
			params.Add(routes, "static_routes")
		} else {
			params.Add(jsonutils.JSONNull, "static_routes")
		}
		result, err := modules.Networks.PerformAction(s, args.NETWORK, "metadata", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type NetworkAddressOptions struct {
		NETWORK string `help:"id or name of network to query"`

		api.GetNetworkAddressesInput
	}
	R(&NetworkAddressOptions{}, "network-addresses", "Query used addresses of network", func(s *mcclient.ClientSession, args *NetworkAddressOptions) error {
		result, err := modules.Networks.GetSpecific(s, args.NETWORK, "addresses", jsonutils.Marshal(args.GetNetworkAddressesInput))
		if err != nil {
			return err
		}
		addrList, _ := result.GetArray("addresses")
		if addrList == nil {
			fmt.Println("no result")
			return nil
		}
		listResult := printutils.ListResult{Data: addrList}
		printList(&listResult, nil)
		return nil
	})

	type NetworkStatusOptions struct {
		NETWORK string `help:"id or name of network to sync" json:"-"`
		STATUS  string `help:"status of network" choices:"available|unavailable" json:"status"`
		Reason  string `help:"reason to change status" json:"reason"`
	}
	R(&NetworkStatusOptions{}, "network-status", "Set on-premise network status", func(s *mcclient.ClientSession, args *NetworkStatusOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Networks.PerformAction(s, args.NETWORK, "status", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type NetworkChangeOwnerOptions struct {
		ID      string `help:"Network to change owner" json:"-"`
		PROJECT string `help:"Project ID or change" json:"tenant"`
	}
	R(&NetworkChangeOwnerOptions{}, "network-change-owner", "Change owner project of a network", func(s *mcclient.ClientSession, args *NetworkChangeOwnerOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		net, err := modules.Networks.PerformAction(s, args.ID, "change-owner", params)
		if err != nil {
			return err
		}
		printObject(net)
		return nil
	})
	type NetworkSetBgpTypeOptions struct {
		ID      string  `help:"Network to set BgpType" json:"-"`
		BgpType *string `help:"new BgpType name"`
	}
	R(&NetworkSetBgpTypeOptions{}, "network-set-bgp-type", "Set BgpType of a network", func(s *mcclient.ClientSession, args *NetworkSetBgpTypeOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		net, err := modules.Networks.PerformAction(s, args.ID, "set-bgp-type", params)
		if err != nil {
			return err
		}
		printObject(net)
		return nil
	})
	type NetworkInterfaceInfoOptions struct {
		DEV string `help:"name of device, e.g. eth0"`
	}
	R(&NetworkInterfaceInfoOptions{}, "network-interface-info", "Show addr and routes of interface", func(s *mcclient.ClientSession, args *NetworkInterfaceInfoOptions) error {
		netIf := netutils2.NewNetInterface(args.DEV)

		fmt.Println("[Slave Addresses]")
		for _, addr := range netIf.GetSlaveAddresses() {
			fmt.Printf("%s/%d\n", addr.Addr, addr.MaskLen)
		}
		fmt.Println("[Routes]")
		for _, r := range netIf.GetRouteSpecs() {
			fmt.Printf("%s via %s dev %d\n", r.Dst.String(), r.Gw.String(), r.LinkIndex)
		}

		return nil
	})
}
