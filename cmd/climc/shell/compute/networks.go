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
)

func init() {

	cmd := shell.NewResourceCmd(&modules.Networks).WithContextManager(&modules.Wires)
	cmd.List(&compute_options.NetworkListOptions{})
	cmd.Update(&compute_options.NetworkUpdateOptions{})
	cmd.Show(&compute_options.NetworkIdOptions{})
	cmd.Delete(&compute_options.NetworkIdOptions{})
	cmd.GetMetadata(&compute_options.NetworkIdOptions{})
	cmd.Perform("private", &compute_options.NetworkIdOptions{})
	cmd.Perform("syncstatus", &compute_options.NetworkIdOptions{})
	cmd.Perform("sync", &compute_options.NetworkIdOptions{})
	cmd.Perform("purge", &compute_options.NetworkIdOptions{})
	cmd.Get("change-owner-candidate-domains", &compute_options.NetworkIdOptions{})
	cmd.Perform("set-class-metadata", &options.ResourceMetadataOptions{})
	cmd.Perform("switch-wire", &compute_options.NetworkSwitchWireOptions{})
	cmd.Get("available-addresses", &compute_options.NetworkIdOptions{})

	type NetworkShareOptions struct {
		ID             string   `help:"ID or Name of the zone to show"`
		Scope          string   `help:"sharing scope" choices:"system|domain|project"`
		SharedProjects []string `help:"Share to prjects"`
		SharedDomains  []string `help:"share to domains"`
	}
	R(&NetworkShareOptions{}, "network-public", "Make a network public", func(s *mcclient.ClientSession, args *NetworkShareOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Networks.PerformAction(s, args.ID, "public", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type NetworkCreateOptions struct {
		WIRE        string `help:"ID or Name of wire in which the network is created"`
		NETWORK     string `help:"Name of new network"`
		STARTIP     string `help:"Start of IPv4 address range"`
		ENDIP       string `help:"End of IPv4 address rnage"`
		NETMASK     int64  `help:"Length of network mask"`
		Gateway     string `help:"Default gateway"`
		VlanId      int64  `help:"Vlan ID" default:"1"`
		IfnameHint  string `help:"Hint for ifname generation"`
		AllocPolicy string `help:"Address allocation policy" choices:"none|stepdown|stepup|random"`
		ServerType  string `help:"Server type" choices:"baremetal|container|eip|guest|ipmi|pxe"`
		IsAutoAlloc *bool  `help:"Auto allocation IP pool"`
		BgpType     string `help:"Internet service provider name" positional:"false"`
		Desc        string `help:"Description" metavar:"DESCRIPTION"`
	}
	R(&NetworkCreateOptions{}, "network-create", "Create a virtual network", func(s *mcclient.ClientSession, args *NetworkCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NETWORK), "name")
		params.Add(jsonutils.NewString(args.STARTIP), "guest_ip_start")
		params.Add(jsonutils.NewString(args.ENDIP), "guest_ip_end")
		params.Add(jsonutils.NewInt(args.NETMASK), "guest_ip_mask")
		if len(args.Gateway) > 0 {
			params.Add(jsonutils.NewString(args.Gateway), "guest_gateway")
		}
		if args.VlanId > 0 {
			params.Add(jsonutils.NewInt(args.VlanId), "vlan_id")
		}
		if len(args.ServerType) > 0 {
			params.Add(jsonutils.NewString(args.ServerType), "server_type")
		}
		if len(args.IfnameHint) > 0 {
			params.Add(jsonutils.NewString(args.IfnameHint), "ifname_hint")
		}
		if len(args.AllocPolicy) > 0 {
			params.Add(jsonutils.NewString(args.AllocPolicy), "alloc_policy")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.BgpType) > 0 {
			params.Add(jsonutils.NewString(args.BgpType), "bgp_type")
		}
		if args.IsAutoAlloc != nil {
			params.Add(jsonutils.NewBool(*args.IsAutoAlloc), "is_auto_alloc")
		}
		net, e := modules.Networks.CreateInContext(s, params, &modules.Wires, args.WIRE)
		if e != nil {
			return e
		}
		printObject(net)
		return nil
	})

	type NetworkCreateOptions2 struct {
		Wire           string `help:"ID or Name of wire in which the network is created"`
		Vpc            string `help:"ID or Name of vpc in which the network is created"`
		Zone           string `help:"ID or Name of zone in which the network is created"`
		NAME           string `help:"Name of new network"`
		PREFIX         string `help:"Start of IPv4 address range"`
		BgpType        string `help:"Internet service provider name" positional:"false"`
		AssignPublicIp bool
		Desc           string `help:"Description" metavar:"DESCRIPTION"`
	}
	R(&NetworkCreateOptions2{}, "network-create2", "Create a virtual network", func(s *mcclient.ClientSession, args *NetworkCreateOptions2) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.PREFIX), "guest_ip_prefix")
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
}
