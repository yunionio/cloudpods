package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type HostNetworkListOptions struct {
		options.BaseListOptions
		Host    string `help:"ID or Name of Host"`
		Network string `help:"ID or name of network"`
	}
	R(&HostNetworkListOptions{}, "host-network-list", "List baremetal network pairs", func(s *mcclient.ClientSession, args *HostNetworkListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if len(args.Network) > 0 {
			params.Add(jsonutils.NewString(args.Network), "network_id")
		}
		var result *modules.ListResult
		var err error
		if len(args.Host) > 0 {
			result, err = modules.Baremetalnetworks.ListDescendent(s, args.Host, params)
		} else {
			result, err = modules.Baremetalnetworks.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Baremetalnetworks.GetColumns(s))
		return nil
	})

	type HostNetworkDetailOptions struct {
		HOST    string `help:"ID or Name of Host"`
		NETWORK string `help:"ID or Name of Wire"`
	}
	R(&HostNetworkDetailOptions{}, "host-network-show", "Show baremetal network details", func(s *mcclient.ClientSession, args *HostNetworkDetailOptions) error {
		result, err := modules.Baremetalnetworks.Get(s, args.HOST, args.NETWORK, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
