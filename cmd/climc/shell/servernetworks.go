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
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ServerNetworkListOptions struct {
		options.BaseListOptions
		Server  string `help:"ID or Name of Server"`
		Mac     string `help:"search the MAC address"`
		Ip      string `help:"search the IP address"`
		Network string `help:"Network ID or name"`
	}
	R(&ServerNetworkListOptions{}, "server-network-list", "List server network pairs", func(s *mcclient.ClientSession, args *ServerNetworkListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if len(args.Mac) > 0 {
			params.Add(jsonutils.NewString(args.Mac), "mac_addr")
		}
		if len(args.Ip) > 0 {
			params.Add(jsonutils.NewString(args.Ip), "ip_addr")
		}
		var result *modulebase.ListResult
		var err error
		if len(args.Server) > 0 {
			result, err = modules.Servernetworks.ListDescendent(s, args.Server, params)
		} else if len(args.Network) > 0 {
			result, err = modules.Servernetworks.ListDescendent2(s, args.Network, params)
		} else {
			result, err = modules.Servernetworks.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Servernetworks.GetColumns(s))
		return nil
	})

	type ServerNetworkDetailOptions struct {
		SERVER  string `help:"ID or Name of Server"`
		NETWORK string `help:"ID or Name of Network"`
		Mac     string `help:"Mac of the guest NIC"`
	}
	R(&ServerNetworkDetailOptions{}, "server-network-show", "Show server network details", func(s *mcclient.ClientSession, args *ServerNetworkDetailOptions) error {
		query := jsonutils.NewDict()
		if len(args.Mac) > 0 {
			query.Add(jsonutils.NewString(args.Mac), "mac")
		}
		result, err := modules.Servernetworks.Get(s, args.SERVER, args.NETWORK, query)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerNetworkUpdateOptions struct {
		SERVER  string `help:"ID or Name of Server"`
		NETWORK string `help:"ID or Name of Wire"`
		Mac     string `help:"Mac of NIC"`
		Driver  string `help:"Driver model of vNIC" choices:"virtio|e1000|vmxnet3|rtl8139"`
		Index   int64  `help:"Index of NIC" default:"-1"`
		Ifname  string `help:"Interface name of vNIC on host"`
	}
	R(&ServerNetworkUpdateOptions{}, "server-network-update", "Update server network settings", func(s *mcclient.ClientSession, args *ServerNetworkUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Driver) > 0 {
			params.Add(jsonutils.NewString(args.Driver), "driver")
		}
		if args.Index >= 0 {
			params.Add(jsonutils.NewInt(args.Index), "index")
		}
		if len(args.Ifname) > 0 {
			params.Add(jsonutils.NewString(args.Ifname), "ifname")
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		query := jsonutils.NewDict()
		if len(args.Mac) > 0 {
			query.Add(jsonutils.NewString(args.Mac), "mac")
		}
		result, err := modules.Servernetworks.Update(s, args.SERVER, args.NETWORK, query, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerNetworkBWOptions struct {
		SERVER  string `help:"ID or Name of server"`
		MACORIP string `help:"IP, Mac, or Index of NIC"`
		BW      int64  `help:"Bandwidth in Mbps"`
	}
	R(&ServerNetworkBWOptions{}, "server-change-bandwidth", "Change server network bandwidth in Mbps", func(s *mcclient.ClientSession, args *ServerNetworkBWOptions) error {
		params := jsonutils.NewDict()
		if regutils.MatchMacAddr(args.MACORIP) {
			params.Add(jsonutils.NewString(args.MACORIP), "mac")
		} else if regutils.MatchIP4Addr(args.MACORIP) {
			params.Add(jsonutils.NewString(args.MACORIP), "ip_addr")
		} else if regutils.MatchInteger(args.MACORIP) {
			index, err := strconv.ParseInt(args.MACORIP, 10, 64)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewInt(index), "index")
		} else {
			return fmt.Errorf("Please specify Ip or Mac")
		}
		params.Add(jsonutils.NewInt(args.BW), "bandwidth")
		server, err := modules.Servers.PerformAction(s, args.SERVER, "change-bandwidth", params)
		if err != nil {
			return err
		}
		printObject(server)
		return nil
	})

	type ServerAttachNetworkOptions struct {
		SERVER  string `help:"ID or Name of server"`
		NETDESC string `help:"Network description"`
	}
	R(&ServerAttachNetworkOptions{}, "server-attach-network", "Attach a server to a virtual network", func(s *mcclient.ClientSession, args *ServerAttachNetworkOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NETDESC), "net_desc")
		srv, err := modules.Servers.PerformAction(s, args.SERVER, "attachnetwork", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	type ServerDetachNetworkOptions struct {
		SERVER  string `help:"ID or Name of server"`
		MACORIP string `help:"Mac Or IP of NIC"`
		Reserve bool   `help:"Put the release IP address into reserved address pool"`
	}
	R(&ServerDetachNetworkOptions{}, "server-detach-network", "Detach the virtual network fron a virtual server", func(s *mcclient.ClientSession, args *ServerDetachNetworkOptions) error {
		params := jsonutils.NewDict()
		// params.Add(jsonutils.NewString(args.NETWORK), "net_id")
		if args.Reserve {
			params.Add(jsonutils.JSONTrue, "reserve")
		}
		if regutils.MatchMacAddr(args.MACORIP) {
			params.Add(jsonutils.NewString(args.MACORIP), "mac")
		} else if regutils.MatchIP4Addr(args.MACORIP) {
			params.Add(jsonutils.NewString(args.MACORIP), "ip_addr")
		} else if regutils.MatchInteger(args.MACORIP) {
			index, err := strconv.ParseInt(args.MACORIP, 10, 64)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewInt(index), "index")
		} else if len(args.MACORIP) > 0 {
			params.Add(jsonutils.NewString(args.MACORIP), "net_id")
		} else {
			return fmt.Errorf("Please specify Ip or Mac")
		}
		srv, err := modules.Servers.PerformAction(s, args.SERVER, "detachnetwork", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	type ServerChangeIPAddressOptions struct {
		SERVER  string `help:"ID or Name of server"`
		MACORIP string `help:"Mac Or IP of NIC"`
		NETDESC string `help:"Network description"`
		Reserve bool   `help:"Put the release IP address into reserved address pool"`
	}
	R(&ServerChangeIPAddressOptions{}, "server-change-ipaddr", "Change ipaddr of a virtual server", func(s *mcclient.ClientSession, args *ServerChangeIPAddressOptions) error {
		params := jsonutils.NewDict()
		if args.Reserve {
			params.Add(jsonutils.JSONTrue, "reserve")
		}
		if regutils.MatchMacAddr(args.MACORIP) {
			params.Add(jsonutils.NewString(args.MACORIP), "mac")
		} else if regutils.MatchIP4Addr(args.MACORIP) {
			params.Add(jsonutils.NewString(args.MACORIP), "ip_addr")
		} else if regutils.MatchInteger(args.MACORIP) {
			index, err := strconv.ParseInt(args.MACORIP, 10, 64)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewInt(index), "index")
		} else {
			return fmt.Errorf("Please specify Ip or Mac")
		}
		conf, err := cmdline.ParseNetworkConfig(args.NETDESC, 0)
		if err != nil {
			return err
		}
		params.Add(conf.JSON(conf), "net_desc")
		srv, err := modules.Servers.PerformAction(s, args.SERVER, "change-ipaddr", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

}
