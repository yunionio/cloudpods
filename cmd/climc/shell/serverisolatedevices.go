package shell

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
)

func init() {
	type ServerAttachDeviceOptions struct {
		SERVER string `help:"ID or name of server"`
		DEVICE string `help:"ID of isolated device to attach"`
		Type   string `help:"Device type" choices:"GPU-HPC|GPU-VGA|PCI"`
	}
	R(&ServerAttachDeviceOptions{}, "server-attach-isolated-device", "Attach an existing isolated device to a virtual server", func(s *mcclient.ClientSession, args *ServerAttachDeviceOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.DEVICE), "device")
		if len(args.Type) > 0 {
			params.Add(jsonutils.NewString(args.Type), "dev_type")
		}
		srv, err := modules.Servers.PerformAction(s, args.SERVER, "attach-isolated-device", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	type ServerDetachDeviceOptions struct {
		SERVER string `help:"ID or name of server"`
		DEVICE string `help:"ID of isolated device to attach"`
	}
	R(&ServerDetachDeviceOptions{}, "server-detach-isolated-device", "Detach a isolated device from a virtual server", func(s *mcclient.ClientSession, args *ServerDetachDeviceOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.DEVICE), "device")
		srv, err := modules.Servers.PerformAction(s, args.SERVER, "detach-isolated_device", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})
}
