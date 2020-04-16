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
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	aN := cmdN("suggestsysrule")
	R(&options.SuggestRuleListOptions{}, aN("list"), "List all suggestsysrules",
		func(s *mcclient.ClientSession, args *options.SuggestRuleListOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := monitor.SuggestSysRuleManager.List(s, params)
			if err != nil {
				return err
			}
			printList(ret, monitor.SuggestSysRuleManager.GetColumns(s))
			return nil
		})

	R(&options.SuggestRuleCreateOptions{}, aN("create"), "Create suggestsys rule",
		func(s *mcclient.ClientSession, args *options.SuggestRuleCreateOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := monitor.SuggestSysRuleManager.Create(s, params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

	R(&options.SuggestRuleShowOptions{}, aN("show"), "Show details of a alert rule",
		func(s *mcclient.ClientSession, args *options.SuggestRuleShowOptions) error {
			ret, err := monitor.SuggestSysRuleManager.Get(s, args.ID, nil)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

	R(&options.SuggestRuleUpdateOptions{}, aN("update"), "Update a alert rule",
		func(s *mcclient.ClientSession, args *options.SuggestRuleUpdateOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := monitor.SuggestSysRuleManager.Update(s, args.ID, params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

	R(&options.SuggestRuleDeleteOptions{}, aN("delete"), "Delete alerts",
		func(s *mcclient.ClientSession, args *options.SuggestRuleDeleteOptions) error {
			ret := monitor.SuggestSysRuleManager.BatchDelete(s, args.ID, nil)
			printBatchResults(ret, monitor.SuggestSysRuleManager.GetColumns(s))
			return nil
		})

}
