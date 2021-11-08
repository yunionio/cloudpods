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

package compute

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type PolicyListOptions struct {
		options.BaseListOptions
	}
	R(&PolicyListOptions{}, "policy-definition-list", "List policy definitions", func(s *mcclient.ClientSession, args *PolicyListOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		result, err := modules.PolicyDefinition.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.PolicyDefinition.GetColumns(s))
		return nil
	})

	type PolicyIdOptions struct {
		ID string `help:"policy definition name or id"`
	}

	R(&PolicyIdOptions{}, "policy-definition-show", "Show policy definition", func(s *mcclient.ClientSession, args *PolicyIdOptions) error {
		result, err := modules.PolicyDefinition.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&PolicyIdOptions{}, "policy-definition-syncstatus", "Sync policy definition status", func(s *mcclient.ClientSession, args *PolicyIdOptions) error {
		result, err := modules.PolicyDefinition.PerformAction(s, args.ID, "syncstatus", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
