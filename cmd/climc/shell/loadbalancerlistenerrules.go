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
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	R(&options.LoadbalancerListenerRuleCreateOptions{}, "lblistenerrule-create", "Create lblistenerrule", func(s *mcclient.ClientSession, opts *options.LoadbalancerListenerRuleCreateOptions) error {
		params := jsonutils.Marshal(opts)
		lblistenerrule, err := modules.LoadbalancerListenerRules.Create(s, params)
		if err != nil {
			return err
		}
		printObject(lblistenerrule)
		return nil
	})
	R(&options.LoadbalancerListenerRuleGetOptions{}, "lblistenerrule-show", "Show lblistenerrule", func(s *mcclient.ClientSession, opts *options.LoadbalancerListenerRuleGetOptions) error {
		lblistenerrule, err := modules.LoadbalancerListenerRules.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lblistenerrule)
		return nil
	})
	R(&options.LoadbalancerListenerRuleListOptions{}, "lblistenerrule-list", "List lblistenerrules", func(s *mcclient.ClientSession, opts *options.LoadbalancerListenerRuleListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.LoadbalancerListenerRules.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.LoadbalancerListenerRules.GetColumns(s))
		return nil
	})
	R(&options.LoadbalancerListenerRuleUpdateOptions{}, "lblistenerrule-update", "Update lblistenerrule", func(s *mcclient.ClientSession, opts *options.LoadbalancerListenerRuleUpdateOptions) error {
		params, err := options.StructToParams(opts)
		lblistenerrule, err := modules.LoadbalancerListenerRules.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printObject(lblistenerrule)
		return nil
	})
	R(&options.LoadbalancerListenerRuleDeleteOptions{}, "lblistenerrule-delete", "Delete lblistenerrule", func(s *mcclient.ClientSession, opts *options.LoadbalancerListenerRuleDeleteOptions) error {
		lblistenerrule, err := modules.LoadbalancerListenerRules.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lblistenerrule)
		return nil
	})
	R(&options.LoadbalancerListenerRuleDeleteOptions{}, "lblistenerrule-purge", "Purge lblistenerrule", func(s *mcclient.ClientSession, opts *options.LoadbalancerListenerRuleDeleteOptions) error {
		lblistenerrule, err := modules.LoadbalancerListenerRules.PerformAction(s, opts.ID, "purge", nil)
		if err != nil {
			return err
		}
		printObject(lblistenerrule)
		return nil
	})
	R(&options.LoadbalancerListenerRuleActionStatusOptions{}, "lblistenerrule-status", "Change lblistenerrule status", func(s *mcclient.ClientSession, opts *options.LoadbalancerListenerRuleActionStatusOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		lblistenerrule, err := modules.LoadbalancerListenerRules.PerformAction(s, opts.ID, "status", params)
		if err != nil {
			return err
		}
		printObject(lblistenerrule)
		return nil
	})
	R(&options.LoadbalancerListenerRuleGetBackendStatusOptions{}, "lblistenerrule-backend-status", "Get lblistenerrule backend status", func(s *mcclient.ClientSession, opts *options.LoadbalancerListenerRuleGetBackendStatusOptions) error {
		backendStatus, err := modules.LoadbalancerListenerRules.GetSpecific(s, opts.ID, "backend-status", nil)
		if err != nil {
			return err
		}
		return printLbBackendStatus(backendStatus)
	})
}
