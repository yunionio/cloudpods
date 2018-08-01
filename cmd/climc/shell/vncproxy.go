package shell

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
)

func init() {
	type VNCConnectOptions struct {
		ID        string `help:"ID of server to connect"`
		Baremetal bool   `help:"Connect to baremetal"`
	}
	R(&VNCConnectOptions{}, "vnc-connect", "Show the VNC console of given server", func(s *mcclient.ClientSession, args *VNCConnectOptions) error {
		params := jsonutils.NewDict()
		if args.Baremetal {
			params.Add(jsonutils.NewString("hosts"), "objtype")
		}
		result, err := modules.VNCProxy.DoConnect(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
