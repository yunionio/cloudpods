package shell

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
)

func init() {
	type HostWireListOptions struct {
		BaseListOptions
		Host string `help:"ID or Name of Host"`
		Wire string `help:"ID or Name of Wire"`
	}
	R(&HostWireListOptions{}, "host-wire-list", "List host wire", func(s *mcclient.ClientSession, args *HostWireListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)
		var result *modules.ListResult
		var err error
		if len(args.Host) > 0 {
			result, err = modules.Hostwires.ListDescendent(s, args.Host, params)
		} else if len(args.Wire) > 0 {
			result, err = modules.Hostwires.ListDescendent2(s, args.Wire, params)
		} else {
			result, err = modules.Hostwires.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Hostwires.GetColumns(s))
		return nil
	})

	type HostWireDetailOptions struct {
		HOST string `help:"ID or Name of Host"`
		WIRE string `help:"ID or Name of Wire"`
	}
	R(&HostWireDetailOptions{}, "host-wire-show", "Show host wire details", func(s *mcclient.ClientSession, args *HostWireDetailOptions) error {
		result, err := modules.Hostwires.Get(s, args.HOST, args.WIRE, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type HostWireUpdateOptions struct {
		HOST      string `help:"ID or Name of Host"`
		WIRE      string `help:"ID or Name of Wire"`
		Mac       string `help:"Mac address"`
		Interface string `help:"Interface"`
		Bridge    string `help:"Bridge"`
		IsMaster  string `help:"Master interface" choices:"true|false"`
	}
	R(&HostWireUpdateOptions{}, "host-wire-update", "Update host wire information", func(s *mcclient.ClientSession, args *HostWireUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Mac) > 0 {
			params.Add(jsonutils.NewString(args.Mac), "mac_addr")
		}
		if len(args.Interface) > 0 {
			params.Add(jsonutils.NewString(args.Interface), "interface")
		}
		if len(args.Bridge) > 0 {
			params.Add(jsonutils.NewString(args.Bridge), "bridge")
		}
		if len(args.IsMaster) > 0 {
			if args.IsMaster == "true" {
				params.Add(jsonutils.JSONTrue, "is_master")
			} else {
				params.Add(jsonutils.JSONFalse, "is_master")
			}
		}
		result, err := modules.Hostwires.Update(s, args.HOST, args.WIRE, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type HostWireDetachOptions struct {
		HOST string `help:"ID or Name of Host"`
		WIRE string `help:"ID or Name of Wire"`
	}
	R(&HostWireDetachOptions{}, "host-wire-detach", "Detach host from wire", func(s *mcclient.ClientSession, args *HostWireDetachOptions) error {
		result, err := modules.Hostwires.Detach(s, args.HOST, args.WIRE)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
