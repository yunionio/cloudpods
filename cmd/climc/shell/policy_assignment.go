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
		Policydefinition string `help:"filter by policydefinition"`
	}
	R(&PolicyListOptions{}, "policy-assignment-list", "List policy assignments", func(s *mcclient.ClientSession, args *PolicyListOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		if len(args.Policydefinition) > 0 {
			params.Add(jsonutils.NewString(args.Policydefinition), "policydefinition")
		}
		result, err := modules.PolicyAssignment.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.PolicyAssignment.GetColumns(s))
		return nil
	})

	type PolicyIdOptions struct {
		ID string `help:"policy assignment id or name"`
	}

	R(&PolicyIdOptions{}, "policy-assignment-delete", "Delete policy assignment", func(s *mcclient.ClientSession, args *PolicyIdOptions) error {
		result, err := modules.PolicyAssignment.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&PolicyIdOptions{}, "policy-assignment-show", "Show policy assignment details", func(s *mcclient.ClientSession, args *PolicyIdOptions) error {
		result, err := modules.PolicyAssignment.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type PolicyAssignmentCreateOptions struct {
		NAME             string
		ProjectDomain    string
		POLICYDEFINITION string
	}

	R(&PolicyAssignmentCreateOptions{}, "policy-assignment-create", "Create policy assignment", func(s *mcclient.ClientSession, args *PolicyAssignmentCreateOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.PolicyAssignment.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
