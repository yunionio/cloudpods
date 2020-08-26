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

package meter

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	type PriceInfoListOptions struct {
		options.BaseListOptions
		PROVIDER string `help:"provider of the priceinfo to show"`
		QUANTITY string `help:"quantity of the priceinfo to show"`
		Period   string `help:"period of the priceinfo to show"`
		Spec     string `help:"spec of the priceinfo to show"`
		PriceKey string `help:"priceKey of the priceinfo to show"`
		RegionId string `help:"regionId of the priceinfo to show"`
	}
	R(&PriceInfoListOptions{}, "priceinfo-list", "List all PriceInfos ", func(s *mcclient.ClientSession, args *PriceInfoListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err
			}
		}

		params.Add(jsonutils.NewString(args.PROVIDER), "provider")
		params.Add(jsonutils.NewString(args.QUANTITY), "quantity")
		if len(args.Period) > 0 {
			params.Add(jsonutils.NewString(args.Period), "period")
		}
		if len(args.Spec) > 0 {
			params.Add(jsonutils.NewString(args.Spec), "spec")
		}
		if len(args.PriceKey) > 0 {
			params.Add(jsonutils.NewString(args.PriceKey), "price_key")
		}
		if len(args.RegionId) > 0 {
			params.Add(jsonutils.NewString(args.RegionId), "region_id")
		}
		result, err := modules.PriceInfos.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.PriceInfos.GetColumns(s))
		return nil
	})

}
