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

	type AccountBalancesListOptions struct {
		options.BaseListOptions
		StatMonth string `help:"stat_month of the query"`
		StartDate string `help:"start_date of the query"`
		EndDate   string `help:"end_date of the query"`
		QueryType string `help:"query_type of the query"`
		Platform  string `help:"platform of the query"`
		ProjectId string `help:"project_id of the query"`
	}
	R(&AccountBalancesListOptions{}, "accountbalances-list", "List all account balances ", func(s *mcclient.ClientSession, args *AccountBalancesListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}

		if len(args.StatMonth) > 0 {
			params.Add(jsonutils.NewString(args.StatMonth), "stat_month")
		}
		if len(args.StartDate) > 0 {
			params.Add(jsonutils.NewString(args.StartDate), "start_date")
		}
		if len(args.EndDate) > 0 {
			params.Add(jsonutils.NewString(args.EndDate), "end_date")
		}
		if len(args.QueryType) > 0 {
			params.Add(jsonutils.NewString(args.QueryType), "query_type")
		}
		if len(args.Platform) > 0 {
			params.Add(jsonutils.NewString(args.Platform), "platform")
		}
		if len(args.ProjectId) > 0 {
			params.Add(jsonutils.NewString(args.ProjectId), "project_id")
		}

		result, err := modules.AccountBalances.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.AccountBalances.GetColumns(s))
		return nil
	})
}
