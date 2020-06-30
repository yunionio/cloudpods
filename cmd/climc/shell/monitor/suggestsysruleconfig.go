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
	n := cmdN("suggestruleconfig")
	man := monitor.SuggestSysRuleConfigManager
	R(&options.SuggestRuleConfigSupportTypesOptions{}, n("support-types"), "Get support driver types",
		func(s *mcclient.ClientSession, opt *options.SuggestRuleConfigSupportTypesOptions) error {
			ret, err := man.Get(s, "support-types", nil)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})
	R(&options.SuggestRuleConfigCreateOptions{}, n("create"), "Create suggestsysrule config",
		func(s *mcclient.ClientSession, opt *options.SuggestRuleConfigCreateOptions) error {
			params, err := opt.Params()
			if err != nil {
				return err
			}
			ret, err := man.Create(s, params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

	R(&options.SuggestRuleConfigListOptions{}, n("list"), "List suggestsysrule configs",
		func(s *mcclient.ClientSession, opt *options.SuggestRuleConfigListOptions) error {
			params, err := opt.Params()
			if err != nil {
				return err
			}
			ret, err := man.List(s, params)
			if err != nil {
				return err
			}
			printList(ret, man.GetColumns(s))
			return nil
		})

	R(&options.SuggestRuleConfigIdOptions{}, n("toggle-alert"), "Toggle suggest alert",
		func(s *mcclient.ClientSession, opt *options.SuggestRuleConfigIdOptions) error {
			ret, err := man.PerformAction(s, opt.ID, "toggle-alert", nil)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

	R(&options.SuggestRuleConfigIdsOptions{}, n("delete"), "Delete suggest rule config",
		func(s *mcclient.ClientSession, opt *options.SuggestRuleConfigIdsOptions) error {
			ret := man.BatchDelete(s, opt.ID, nil)
			printBatchResults(ret, man.GetColumns(s))
			return nil
		})
}
