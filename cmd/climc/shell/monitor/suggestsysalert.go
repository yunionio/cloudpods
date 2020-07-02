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

package monitor

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	aN := cmdN("suggestsysalert")
	R(&options.SuggestSysAlertListOptions{}, aN("list"), "List all suggestsysrules",
		func(s *mcclient.ClientSession, args *options.SuggestSysAlertListOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			if len(args.Type) > 0 {
				params.Add(jsonutils.NewString(args.Type), "type")
			}
			ret, err := monitor.SuggestSysAlertManager.List(s, params)
			if err != nil {
				return err
			}
			printList(ret, monitor.SuggestSysAlertManager.GetColumns(s))
			return nil
		})

	R(&options.SSuggestAlertShowOptions{}, aN("show"), "Show details of a alert rule",
		func(s *mcclient.ClientSession, args *options.SSuggestAlertShowOptions) error {
			ret, err := monitor.SuggestSysAlertManager.Get(s, args.ID, nil)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

	R(&options.SuggestAlertIgnoreOptions{}, aN("ignore"), "Ignore alert result",
		func(s *mcclient.ClientSession, args *options.SuggestAlertIgnoreOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := monitor.SuggestSysAlertManager.PerformAction(s, args.ID, "ignore", params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})
}
