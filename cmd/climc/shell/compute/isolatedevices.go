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
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type DeviceListOptions struct {
		options.BaseListOptions
		Unused bool   `help:"Only show unused devices"`
		Gpu    bool   `help:"Only show gpu devices"`
		Host   string `help:"Host ID or Name"`
		Region string `help:"Cloudregion ID or Name"`
		Zone   string `help:"Zone ID or Name"`
		Server string `help:"Server ID or Name"`
	}
	R(&DeviceListOptions{}, "isolated-device-list", "List isolated devices like GPU", func(s *mcclient.ClientSession, args *DeviceListOptions) error {
		var params *jsonutils.JSONDict
		{
			param, err := args.BaseListOptions.Params()
			if err != nil {
				return err
			}
			params = param.(*jsonutils.JSONDict)
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
		if len(args.Region) > 0 {
			params.Add(jsonutils.NewString(args.Region), "region")
		}
		if args.Zone != "" {
			params.Add(jsonutils.NewString(args.Zone), "zone")
		}
		if args.Server != "" {
			params.Add(jsonutils.NewString(args.Server), "guest_id")
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

	type DeviceDeleteOptions struct {
		IDS []string `help:"IDs of the isolated device"`
	}
	R(&DeviceDeleteOptions{}, "isolated-device-delete", "Delete a isolated device", func(s *mcclient.ClientSession, args *DeviceDeleteOptions) error {
		result := modules.IsolatedDevices.BatchDelete(s, args.IDS, nil)
		printBatchResults(result, modules.IsolatedDevices.GetColumns(s))
		return nil
	})

	type DeviceUpdateOptions struct {
		ID              []string `help:"ID of the isolated device" json:"-"`
		ReservedCpu     *int     `help:"reserved cpu for isolated device"`
		ReservedMem     *int     `help:"reserved mem for isolated device"`
		ReservedStorage *int     `help:"reserved storage for isolated device"`
		DevType         string   `help:"Device type" choices:"GPU-HPC|GPU-VGA"`
	}
	R(&DeviceUpdateOptions{}, "isolated-device-update", "Update a isolated device", func(s *mcclient.ClientSession, args *DeviceUpdateOptions) error {
		res := modules.IsolatedDevices.BatchUpdate(s, args.ID, jsonutils.Marshal(args))
		printBatchResults(res, modules.IsolatedDevices.GetColumns(s))
		return nil
	})
}
