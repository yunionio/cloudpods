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

	type BillCloudCheckListOptions struct {
		options.BaseListOptions
		ACCOUNTID string `help:"accountId of the bill_cloudcheck"`
		SUMMONTH  string `help:"sum_month of the bill_cloudcheck"`
		QUERYTYPE string `help:"query_type of the bill_cloudcheck"`
		QueryItem string `help:"query_item of the bill_cloudcheck"`
	}
	R(&BillCloudCheckListOptions{}, "billcloudcheck-list", "List all BillCloudChecks ", func(s *mcclient.ClientSession, args *BillCloudCheckListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err
			}
		}

		params.Add(jsonutils.NewString(args.ACCOUNTID), "account_id")
		params.Add(jsonutils.NewString(args.SUMMONTH), "sum_month")
		params.Add(jsonutils.NewString(args.QUERYTYPE), "query_type")
		if len(args.QueryItem) > 0 {
			params.Add(jsonutils.NewString(args.QueryItem), "query_item")
		}

		result, err := modules.BillCloudChecks.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.BillCloudChecks.GetColumns(s))
		return nil
	})
}
