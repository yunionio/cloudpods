package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type WireListOptions struct {
		options.BaseListOptions

		Region string `help:"List hosts in region"`
		Zone   string `help:"list wires in zone"`
		Vpc    string `help:"List wires in vpc"`

		Manager  string `help:"List hosts belongs to the cloud provider"`
		Account  string `help:"List hosts belongs to the cloud account"`
		Provider string `help:"List hosts belongs to the provider" choices:"VMware|Aliyun|Qcloud|Azure|Aws|Huawei"`
	}
	R(&WireListOptions{}, "wire-list", "List wires", func(s *mcclient.ClientSession, args *WireListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if len(args.Vpc) > 0 {
			params.Add(jsonutils.NewString(args.Vpc), "vpc")
		}
		if len(args.Region) > 0 {
			params.Add(jsonutils.NewString(args.Region), "region")
		}
		if len(args.Manager) > 0 {
			params.Add(jsonutils.NewString(args.Manager), "manager")
		}
		if len(args.Account) > 0 {
			params.Add(jsonutils.NewString(args.Account), "account")
		}
		if len(args.Provider) > 0 {
			params.Add(jsonutils.NewString(args.Provider), "provider")
		}

		var result *modules.ListResult
		var err error
		if len(args.Zone) > 0 {
			result, err = modules.Wires.ListInContext(s, params, &modules.Zones, args.Zone)
		} else {
			result, err = modules.Wires.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Wires.GetColumns(s))
		return nil
	})

	type WireUpdateOptions struct {
		ID   string `help:"ID or Name of zone to update"`
		Name string `help:"Name of wire"`
		Desc string `metavar:"<DESCRIPTION>" help:"Description"`
		Bw   int64  `help:"Bandwidth in mbps"`
	}
	R(&WireUpdateOptions{}, "wire-update", "Update wire", func(s *mcclient.ClientSession, args *WireUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.Bw > 0 {
			params.Add(jsonutils.NewInt(args.Bw), "bandwidth")
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Wires.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type WireCreateOptions struct {
		ZONE string `help:"Zone ID or Name"`
		Vpc  string `help:"VPC ID or Name" default:"default"`
		NAME string `help:"Name of wire"`
		BW   int64  `help:"Bandwidth in mbps"`
		Desc string `metavar:"<DESCRIPTION>" help:"Description"`
	}
	R(&WireCreateOptions{}, "wire-create", "Create a wire", func(s *mcclient.ClientSession, args *WireCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewInt(args.BW), "bandwidth")
		if len(args.Vpc) > 0 {
			params.Add(jsonutils.NewString(args.Vpc), "vpc")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		result, err := modules.Wires.CreateInContext(s, params, &modules.Zones, args.ZONE)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type WireShowOptions struct {
		ID string `help:"ID or Name of the wire to show"`
	}
	R(&WireShowOptions{}, "wire-show", "Show wire details", func(s *mcclient.ClientSession, args *WireShowOptions) error {
		result, err := modules.Wires.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&WireShowOptions{}, "wire-delete", "Delete wire", func(s *mcclient.ClientSession, args *WireShowOptions) error {
		result, err := modules.Wires.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
