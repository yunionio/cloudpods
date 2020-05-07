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
	/**
	 * 列出列表
	 */
	type RateListOptions struct {
		options.BaseListOptions
		RESTYPE    string `help:"res_type of the rate"`
		ACTION     string `help:"action of list :querygroup or queryhistory"`
		SubResType string `help:"query the subResType"`
		Id         string `help:"ID of rate"`
	}
	R(&RateListOptions{}, "rate-list", "List all rates ", func(s *mcclient.ClientSession, args *RateListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}

		params.Add(jsonutils.NewString(args.RESTYPE), "res_type")
		params.Add(jsonutils.NewString(args.ACTION), "action")

		if len(args.SubResType) > 0 {
			params.Add(jsonutils.NewString(args.SubResType), "sub_res_type")
		}
		if len(args.Id) > 0 {
			params.Add(jsonutils.NewString(args.Id), "id")
		}

		result, err := modules.Rates.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.Rates.GetColumns(s))
		return nil
	})

	/**
	 * 根据ID查询详情
	 */
	type RateShowOptions struct {
		ID string `help:"ID of the rate to show"`
	}
	R(&RateShowOptions{}, "rate-show", "Show rate details", func(s *mcclient.ClientSession, args *RateShowOptions) error {
		result, err := modules.Rates.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	/**
	 * 删除
	 */
	type RateDeleteOptions struct {
		ID string `help:"ID of rate"`
	}
	R(&RateDeleteOptions{}, "rate-delete", "Delete a rate", func(s *mcclient.ClientSession, args *RateDeleteOptions) error {
		rate, e := modules.Rates.Delete(s, args.ID, nil)
		if e != nil {
			return e
		}
		printObject(rate)
		return nil
	})

	/**
	 * 创建
	 */
	type RateCreateOptions struct {
		RESTYPE       string `help:"RESTYPE of the rate"`
		SubResType    string `help:"sub_res_type of the rate"`
		DURATION      string `help:"DURATION of the rate"`
		UNIT          string `help:"UNIT of the rate"`
		Spec          string `help:"Spec of the rate"`
		RATE          string `help:"RATE of the rate"`
		EFFECTIVEDATE string `help:"EFFECTIVEDATE of the rate"`
		Platform      string `help:"Platform of the rate"`
	}

	R(&RateCreateOptions{}, "rate-create", "Create a rate", func(s *mcclient.ClientSession, args *RateCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.RESTYPE), "res_type")
		params.Add(jsonutils.NewString(args.SubResType), "sub_res_type")
		params.Add(jsonutils.NewString(args.DURATION), "duration")
		params.Add(jsonutils.NewString(args.UNIT), "unit")
		params.Add(jsonutils.NewString(args.Spec), "spec")
		params.Add(jsonutils.NewString(args.RATE), "rate")
		params.Add(jsonutils.NewString(args.EFFECTIVEDATE), "effective_date")
		params.Add(jsonutils.NewString(args.Platform), "platform")

		rate, err := modules.Rates.Create(s, params)
		if err != nil {
			return err
		}
		printObject(rate)
		return nil
	})

	/**
	 * 修改
	 */
	type RateUpdateOptions struct {
		ID            string `help:"ID of the rate"`
		ResType       string `help:"ResType of the rate"`
		SubResType    string `help:"SubResType of the rate"`
		Duration      string `help:"Duration of the rate"`
		Unit          string `help:"Unit of the rate"`
		Spec          string `help:"Spec of the rate"`
		Rate          string `help:"Rate of the rate"`
		EffectiveDate string `help:"Effective Date of the rate"`
		Platform      string `help:"Platform of the rate"`
	}
	R(&RateUpdateOptions{}, "rate-update", "Update a rate", func(s *mcclient.ClientSession, args *RateUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.ResType) > 0 {
			params.Add(jsonutils.NewString(args.ResType), "res_type")
		}
		if len(args.SubResType) > 0 {
			params.Add(jsonutils.NewString(args.SubResType), "sub_res_type")
		}
		if len(args.Duration) > 0 {
			params.Add(jsonutils.NewString(args.Duration), "duration")
		}
		if len(args.Unit) > 0 {
			params.Add(jsonutils.NewString(args.Unit), "unit")
		}
		if len(args.Spec) > 0 {
			params.Add(jsonutils.NewString(args.Spec), "spec")
		}
		if len(args.Rate) > 0 {
			params.Add(jsonutils.NewString(args.Rate), "rate")
		}
		if len(args.EffectiveDate) > 0 {
			params.Add(jsonutils.NewString(args.EffectiveDate), "effective_date")
		}
		if len(args.Platform) > 0 {
			params.Add(jsonutils.NewString(args.Platform), "platform")
		}
		rate, err := modules.Rates.Put(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(rate)
		return nil
	})
}
