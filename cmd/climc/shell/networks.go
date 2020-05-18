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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type NetworkListOptions struct {
		options.BaseListOptions

		Ip         string   `help:"search networks that contain this IP"`
		Zone       []string `help:"search networks in zones"`
		Wire       string   `help:"search networks belongs to a wire" json:"-"`
		Host       string   `help:"search networks attached to a host"`
		Vpc        string   `help:"search networks belongs to a VPC"`
		Region     string   `help:"search networks belongs to a CloudRegion" json:"cloudregion"`
		City       string   `help:"search networks belongs to a city"`
		Usable     *bool    `help:"search usable networks"`
		ServerType string   `help:"search networks belongs to a ServerType" choices:"guest|baremetal|container|pxe|ipmi"`
		Schedtag   string   `help:"filter networks by schedtag"`

		Status string `help:"filter by network status"`
	}
	R(&NetworkListOptions{}, "network-list", "List networks", func(s *mcclient.ClientSession, opts *NetworkListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		var result *modulebase.ListResult
		if len(opts.Wire) > 0 {
			result, err = modules.Networks.ListInContext(s, params, &modules.Wires, opts.Wire)
		} else {
			result, err = modules.Networks.List(s, params)
		}
		if err != nil {
			return err
		}
		if len(opts.ExportFile) > 0 {
			exportList(result, opts.ExportFile, opts.ExportKeys, opts.ExportTexts, modules.Networks.GetColumns(s))
		} else {
			printList(result, modules.Networks.GetColumns(s))
		}
		return nil
	})

	type NetworkUpdateOptions struct {
		ID          string `help:"ID or Name of zone to update"`
		Name        string `help:"Name of zone"`
		Desc        string `metavar:"<DESCRIPTION>" help:"Description"`
		ServerType  string `help:"server type," choices:"baremetal|guest|container|pxe|ipmi"`
		StartIp     string `help:"Start ip"`
		EndIp       string `help:"end ip"`
		NetMask     int64  `help:"Netmask"`
		Gateway     string `help:"IP of gateway"`
		Dns         string `help:"IP of DNS server"`
		Domain      string `help:"Domain"`
		Dhcp        string `help:"DHCP server IP"`
		VlanId      int64  `help:"Vlan ID" default:"1"`
		ExternalId  string `help:"External ID"`
		AllocPolicy string `help:"Address allocation policy" choices:"none|stepdown|stepup|random"`
	}
	R(&NetworkUpdateOptions{}, "network-update", "Update network", func(s *mcclient.ClientSession, args *NetworkUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.ServerType) > 0 {
			params.Add(jsonutils.NewString(args.ServerType), "server_type")
		}
		if len(args.StartIp) > 0 {
			params.Add(jsonutils.NewString(args.StartIp), "guest_ip_start")
		}
		if len(args.EndIp) > 0 {
			params.Add(jsonutils.NewString(args.EndIp), "guest_ip_end")
		}
		if args.NetMask > 0 {
			params.Add(jsonutils.NewInt(args.NetMask), "guest_ip_mask")
		}
		if len(args.Gateway) > 0 {
			params.Add(jsonutils.NewString(args.Gateway), "guest_gateway")
		}
		if len(args.Dns) > 0 {
			if args.Dns == "none" {
				params.Add(jsonutils.NewString(""), "guest_dns")
			} else {
				params.Add(jsonutils.NewString(args.Dns), "guest_dns")
			}
		}
		if len(args.Domain) > 0 {
			if args.Domain == "none" {
				params.Add(jsonutils.NewString(""), "guest_domain")
			} else {
				params.Add(jsonutils.NewString(args.Domain), "guest_domain")
			}
		}
		if len(args.Dhcp) > 0 {
			if args.Dhcp == "none" {
				params.Add(jsonutils.NewString(""), "guest_dhcp")
			} else {
				params.Add(jsonutils.NewString(args.Dhcp), "guest_dhcp")
			}
		}
		if args.VlanId > 0 {
			params.Add(jsonutils.NewInt(args.VlanId), "vlan_id")
		}
		if len(args.ExternalId) > 0 {
			params.Add(jsonutils.NewString(args.ExternalId), "external_id")
		}
		if len(args.AllocPolicy) > 0 {
			params.Add(jsonutils.NewString(args.AllocPolicy), "alloc_policy")
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Networks.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type NetworkIdOptions struct {
		ID string `help:"ID or Name of the zone to show"`
	}
	R(&NetworkIdOptions{}, "network-show", "Show network details", func(s *mcclient.ClientSession, args *NetworkIdOptions) error {
		result, err := modules.Networks.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&NetworkIdOptions{}, "network-metadata", "Show metadata of a network", func(s *mcclient.ClientSession, args *NetworkIdOptions) error {
		result, err := modules.Networks.GetMetadata(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&NetworkIdOptions{}, "network-private", "Make a network private", func(s *mcclient.ClientSession, args *NetworkIdOptions) error {
		result, err := modules.Networks.PerformAction(s, args.ID, "private", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&NetworkIdOptions{}, "network-syncstatus", "Sync network status", func(s *mcclient.ClientSession, args *NetworkIdOptions) error {
		result, err := modules.Networks.PerformAction(s, args.ID, "syncstatus", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type NetworkShareOptions struct {
		NetworkIdOptions
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

	R(&NetworkIdOptions{}, "network-delete", "Delete a network", func(s *mcclient.ClientSession, args *NetworkIdOptions) error {
		result, err := modules.Networks.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&NetworkIdOptions{}, "network-purge", "Purge a managed network, not delete the remote entity", func(s *mcclient.ClientSession, args *NetworkIdOptions) error {
		result, err := modules.Networks.PerformAction(s, args.ID, "purge", nil)
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
		ServerType  string `help:"Server type" choices:"baremetal|guest|container|pxe|ipmi"`
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
		net, e := modules.Networks.CreateInContext(s, params, &modules.Wires, args.WIRE)
		if e != nil {
			return e
		}
		printObject(net)
		return nil
	})

	type NetworkCreateOptions2 struct {
		Wire   string `help:"ID or Name of wire in which the network is created"`
		Vpc    string `help:"ID or Name of vpc in which the network is created"`
		Zone   string `help:"ID or Name of zone in which the network is created"`
		NAME   string `help:"Name of new network"`
		PREFIX string `help:"Start of IPv4 address range"`
		Desc   string `help:"Description" metavar:"DESCRIPTION"`
	}
	R(&NetworkCreateOptions2{}, "network-create2", "Create a virtual network", func(s *mcclient.ClientSession, args *NetworkCreateOptions2) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.PREFIX), "guest_ip_prefix")
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
	}
	R(&NetworkAddressOptions{}, "network-addresses", "Query used addresses of network", func(s *mcclient.ClientSession, args *NetworkAddressOptions) error {
		result, err := modules.Networks.GetSpecific(s, args.NETWORK, "addresses", nil)
		if err != nil {
			return err
		}
		addrList, err := result.GetArray("addresses")
		if err != nil {
			return err
		}
		listResult := modulebase.ListResult{Data: addrList}
		printList(&listResult, nil)
		return nil
	})

	type NetworkSyncOptions struct {
		NETWORK string `help:"id or name of network to sync"`
	}
	R(&NetworkSyncOptions{}, "network-sync", "Sync network status", func(s *mcclient.ClientSession, args *NetworkSyncOptions) error {
		result, err := modules.Networks.PerformAction(s, args.NETWORK, "sync", nil)
		if err != nil {
			return err
		}
		printObject(result)
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

	R(&NetworkIdOptions{}, "network-change-owner-candidate-domains", "Show candiate domains of a network for changing owner", func(s *mcclient.ClientSession, args *NetworkIdOptions) error {
		result, err := modules.Networks.GetSpecific(s, args.ID, "change-owner-candidate-domains", nil)
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
}
