package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type DeviceListOptions struct {
		options.BaseListOptions
		Unused bool   `help:"Only show unused devices"`
		Gpu    bool   `help:"Only show gpu devices"`
		Host   string `help:"Host ID or Name"`
	}
	R(&DeviceListOptions{}, "isolated-device-list", "List isolated devices like GPU", func(s *mcclient.ClientSession, args *DeviceListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if len(args.Host) > 0 {
			params.Add(jsonutils.NewString(args.Host), "host")
		}
		if args.Unused {
			params.Add(jsonutils.JSONTrue, "unused")
		}
		if args.Gpu {
			params.Add(jsonutils.JSONTrue, "gpu")
		}
		result, err := modules.IsolatedDevices.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.IsolatedDevices.GetColumns(s))
		return nil
	})

	type DeviceShowOptions struct {
		ID string `help:"ID of the isolated device"`
	}
	R(&DeviceShowOptions{}, "isolated-device-show", "Show isolated device details", func(s *mcclient.ClientSession, args *DeviceShowOptions) error {
		result, err := modules.IsolatedDevices.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&DeviceShowOptions{}, "isolated-device-delete", "Delete a isolated device", func(s *mcclient.ClientSession, args *DeviceShowOptions) error {
		result, err := modules.IsolatedDevices.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
