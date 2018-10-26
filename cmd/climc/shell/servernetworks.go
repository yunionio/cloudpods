package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
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
		if len(args.Network) > 0 {
			params.Add(jsonutils.NewString(args.Network), "network_id")
		}
		var result *modules.ListResult
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
	}
	R(&ServerNetworkDetailOptions{}, "server-network-show", "Show server network details", func(s *mcclient.ClientSession, args *ServerNetworkDetailOptions) error {
		result, err := modules.Servernetworks.Get(s, args.SERVER, args.NETWORK, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerNetworkUpdateOptions struct {
		SERVER  string `help:"ID or Name of Server"`
		NETWORK string `help:"ID or Name of Wire"`
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
		result, err := modules.Servernetworks.Update(s, args.SERVER, args.NETWORK, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerNetworkBWOptions struct {
		SERVER string `help:"ID or Name of server"`
		INDEX  int64  `help:"Index of NIC"`
		BW     int64  `help:"Bandwidth in Mbps"`
	}
	R(&ServerNetworkBWOptions{}, "server-change-bandwidth", "Change server network bandwidth in Mbps", func(s *mcclient.ClientSession, args *ServerNetworkBWOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewInt(args.INDEX), "index")
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
		NETWORK string `help:"ID or Name of network to detach"`
		Reserve bool   `help:"Put the release IP address into reserved address pool"`
	}
	R(&ServerDetachNetworkOptions{}, "server-detach-network", "Detach the virtual network fron a virtual server", func(s *mcclient.ClientSession, args *ServerDetachNetworkOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NETWORK), "net_id")
		if args.Reserve {
			params.Add(jsonutils.JSONTrue, "reserve")
		}
		srv, err := modules.Servers.PerformAction(s, args.SERVER, "detachnetwork", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

}
