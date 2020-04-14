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

	R(&PolicyIdOptions{}, "policy-definition-delete", "Delete policy definition", func(s *mcclient.ClientSession, args *PolicyIdOptions) error {
		result, err := modules.PolicyDefinition.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

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

	type PolicyDefinitionCreateOptions struct {
		NAME         string `help:"policy name"`
		CATEGORY     string `help:"policy definition category" choices:"cloudregion|tag|expired|billing_type|batch_create"`
		CONDITION    string `help:"policy condition"`
		Domains      string `help:"domains"`
		Duration     string
		Cloudregions []string
		Tags         []string
		Count        int
		BillingType  string `help:"server billing type" choices:"postpaid|prepaid"`
	}

	R(&PolicyDefinitionCreateOptions{}, "policy-definition-create", "Create policy definition", func(s *mcclient.ClientSession, args *PolicyDefinitionCreateOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.PolicyDefinition.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
