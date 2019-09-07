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

package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type CloudproviderRegionListOptions struct {
		options.BaseListOptions
		Region string `help:"ID or Name of Host"`
	}
	R(&CloudproviderRegionListOptions{}, "cloud-provider-region-list", "List cloudprovider region synchronization status", func(s *mcclient.ClientSession, args *CloudproviderRegionListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		var result *modulebase.ListResult
		var err error
		if len(args.Manager) > 0 {
			result, err = modules.CloudproviderregionManager.ListDescendent(s, args.Manager, params)
		} else if len(args.Region) > 0 {
			result, err = modules.CloudproviderregionManager.ListDescendent2(s, args.Region, params)
		} else {
			result, err = modules.CloudproviderregionManager.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.CloudproviderregionManager.GetColumns(s))
		return nil
	})

	type CloudproviderRegionOpsOptions struct {
		PROVIDER string `help:"ID or name of cloud provider"`
		REGION   string `help:"ID or name of cloud region"`
	}
	R(&CloudproviderRegionOpsOptions{}, "cloud-provider-region-enable", "Enable automatic synchronization for cloudprovider/region pair", func(s *mcclient.ClientSession, args *CloudproviderRegionOpsOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.JSONTrue, "enabled")
		result, err := modules.CloudproviderregionManager.Update(s, args.PROVIDER, args.REGION, nil, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CloudproviderRegionOpsOptions{}, "cloud-provider-region-disable", "Disable automatic synchronization for cloudprovider/region pair", func(s *mcclient.ClientSession, args *CloudproviderRegionOpsOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.JSONFalse, "enabled")
		result, err := modules.CloudproviderregionManager.Update(s, args.PROVIDER, args.REGION, nil, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
