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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
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
		srv, err := modules.Servers.PerformAction(s, args.SERVER, "detach-isolated-device", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})
}
