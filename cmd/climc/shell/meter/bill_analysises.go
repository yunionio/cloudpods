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

	type BillAnalysisListOptions struct {
		options.BaseListOptions
		QUERYTYPE  string `help:"query_type of the bill_analysis"`
		STARTDATE  string `help:"start_date of the bill_analysis"`
		ENDDATE    string `help:"end_date of the bill_analysis"`
		DataType   string `help:"data_type of the bill_analysis"`
		ChargeType string `help:"charge_type of the bill_analysis"`

		Platform    string `help:"platform of the bill_analysis"`
		AccountId   string `help:"account_id of the bill_analysis"`
		RegionExtId string `help:"region_ext_id of the bill_analysis"`
		ProjectId   string `help:"project_id of the bill_analysis"`
	}
	R(&BillAnalysisListOptions{}, "billanalysis-list", "List all bill analysises", func(s *mcclient.ClientSession, args *BillAnalysisListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}

		if len(args.QUERYTYPE) > 0 {
			params.Add(jsonutils.NewString(args.QUERYTYPE), "query_type")
		}

		if len(args.STARTDATE) > 0 {
			params.Add(jsonutils.NewString(args.STARTDATE), "start_date")
		}

		if len(args.ENDDATE) > 0 {
			params.Add(jsonutils.NewString(args.ENDDATE), "end_date")
		}

		if len(args.DataType) > 0 {
			params.Add(jsonutils.NewString(args.DataType), "data_type")
		}

		if len(args.ChargeType) > 0 {
			params.Add(jsonutils.NewString(args.ChargeType), "charge_type")
		}

		if len(args.Platform) > 0 {
			params.Add(jsonutils.NewString(args.Platform), "platform")
		}

		if len(args.AccountId) > 0 {
			params.Add(jsonutils.NewString(args.AccountId), "account_id")
		}

		if len(args.RegionExtId) > 0 {
			params.Add(jsonutils.NewString(args.RegionExtId), "region_ext_id")
		}

		if len(args.ProjectId) > 0 {
			params.Add(jsonutils.NewString(args.ProjectId), "project_id")
		}

		result, err := modules.BillAnalysises.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.BillAnalysises.GetColumns(s))
		return nil
	})
}
