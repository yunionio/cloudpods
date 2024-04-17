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
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	baseoptions "yunion.io/x/onecloud/pkg/mcclient/options"
	options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.IsolatedDeviceModels)
	cmd.List(new(options.IsolatedDeviceModelListOptions))
	cmd.Create(new(options.IsolatedDeviceModelCreateOptions))
	cmd.Update(new(options.IsolatedDeviceModelUpdateOptions))
	cmd.Delete(new(options.IsolatedDeviceIdsOptions))
	cmd.Perform("set-hardware-info", new(options.IsolatedDeviceModelSetHardwareInfoOptions))
	cmd.Get("hardware-info", new(options.IsolatedDeviceIdsOptions))

	type HostIsolatedDeviceModelListOptions struct {
		baseoptions.BaseListOptions
		Host                string `help:"ID or Name of Host"`
		IsolatedDeviceModel string `help:"ID or Name of Isolated device model"`
	}
	R(&HostIsolatedDeviceModelListOptions{}, "host-isolated-device-model-list",
		"List host Isolated device model pairs", func(s *mcclient.ClientSession, args *HostIsolatedDeviceModelListOptions) error {
			var params *jsonutils.JSONDict
			{
				var err error
				params, err = args.BaseListOptions.Params()
				if err != nil {
					return err

				}
			}
			var result *printutils.ListResult
			var err error
			if len(args.Host) > 0 {
				result, err = modules.HostIsolatedDeviceModels.ListDescendent(s, args.Host, params)
			} else if len(args.IsolatedDeviceModel) > 0 {
				result, err = modules.HostIsolatedDeviceModels.ListDescendent2(s, args.IsolatedDeviceModel, params)
			} else {
				result, err = modules.HostIsolatedDeviceModels.List(s, params)
			}
			if err != nil {
				return err
			}
			printList(result, modules.HostIsolatedDeviceModels.GetColumns(s))
			return nil
		})

	type HostIsolatedDeviceModelDetailOptions struct {
		HOST                string `help:"ID or Name of Host"`
		IsolatedDeviceModel string `help:"ID or Name of Isolated device model"`
	}
	R(&HostIsolatedDeviceModelDetailOptions{}, "host-isolated-device-model-show", "Show host isolated-device-model details", func(s *mcclient.ClientSession, args *HostIsolatedDeviceModelDetailOptions) error {
		result, err := modules.HostIsolatedDeviceModels.Get(s, args.HOST, args.IsolatedDeviceModel, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&HostIsolatedDeviceModelDetailOptions{}, "host-isolated-device-model-detach", "Detach a isolated-device-model from a host", func(s *mcclient.ClientSession, args *HostIsolatedDeviceModelDetailOptions) error {
		result, err := modules.HostIsolatedDeviceModels.Detach(s, args.HOST, args.IsolatedDeviceModel, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&HostIsolatedDeviceModelDetailOptions{}, "host-isolated-device-model-attach", "Attach a isolated-device-model to a host", func(s *mcclient.ClientSession, args *HostIsolatedDeviceModelDetailOptions) error {
		params := jsonutils.NewDict()
		result, err := modules.HostIsolatedDeviceModels.Attach(s, args.HOST, args.IsolatedDeviceModel, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
