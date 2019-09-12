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

package cloudnet

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/cloudnet"
	base_options "yunion.io/x/onecloud/pkg/mcclient/options"
	options "yunion.io/x/onecloud/pkg/mcclient/options/cloudnet"
)

func init() {
	R(&options.RuleCreateOptions{}, "router-rule-create", "Create router rule", func(s *mcclient.ClientSession, opts *options.RuleCreateOptions) error {
		params, err := base_options.StructToParams(opts)
		if err != nil {
			return err
		}
		router, err := modules.Rules.Create(s, params)
		if err != nil {
			return err
		}
		printObject(router)
		return nil
	})
	R(&options.RuleGetOptions{}, "router-rule-show", "Show router rule", func(s *mcclient.ClientSession, opts *options.RuleGetOptions) error {
		router, err := modules.Rules.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(router)
		return nil
	})
	R(&options.RuleListOptions{}, "router-rule-list", "List router rules", func(s *mcclient.ClientSession, opts *options.RuleListOptions) error {
		params, err := base_options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.Rules.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Rules.GetColumns(s))
		return nil
	})
	R(&options.RuleUpdateOptions{}, "router-rule-update", "Update router rule", func(s *mcclient.ClientSession, opts *options.RuleUpdateOptions) error {
		params, err := base_options.StructToParams(opts)
		router, err := modules.Rules.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printObject(router)
		return nil
	})
	R(&options.RuleDeleteOptions{}, "router-rule-delete", "Delete router rule", func(s *mcclient.ClientSession, opts *options.RuleDeleteOptions) error {
		router, err := modules.Rules.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(router)
		return nil
	})
}
