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
	type CloudproviderQuotaListOptions struct {
		options.BaseListOptions
		Cloudregion string `help:"ID or Name of Cloudregion"`
		QuotaType   string `help:"Quota type"`
		QuotaRange  string `help:"Quota range" choices:"cloudregion|cloudprovider"`
	}

	R(&CloudproviderQuotaListOptions{}, "cloud-provider-quota-list", "List cloudprovider quota", func(s *mcclient.ClientSession, args *CloudproviderQuotaListOptions) error {
		param, err := args.Params()
		if err != nil {
			return err
		}
		params := param.(*jsonutils.JSONDict)

		if len(args.QuotaType) > 0 {
			params.Add(jsonutils.NewString(args.QuotaType), "quota_type")
		}

		if len(args.QuotaRange) > 0 {
			params.Add(jsonutils.NewString(args.QuotaRange), "quota_range")
		}

		if len(args.Cloudregion) > 0 {
			params.Add(jsonutils.NewString(args.Cloudregion), "cloudregion")
		}

		result, err := modules.CloudproviderQuotas.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.CloudproviderQuotas.GetColumns(s))
		return nil
	})

}
