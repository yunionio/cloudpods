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

package identity

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

func init() {
	type RolePolicyListOptions struct {
		api.RolePolicyListInput
	}
	R(&RolePolicyListOptions{}, "role-policy-list", "List associated policies of a role", func(s *mcclient.ClientSession, args *RolePolicyListOptions) error {
		results, err := modules.RolePolicies.List(s, jsonutils.Marshal(args))
		if err != nil {
			return err
		}
		printList(results, modules.RolePolicies.GetColumns(s))
		return nil
	})

	type RoleAddPolicyOptions struct {
		ID string `json:"-" help:"role id or name to add policy"`
		api.RolePerformAddPolicyInput
	}
	R(&RoleAddPolicyOptions{}, "role-add-policy", "Add policy to a role", func(s *mcclient.ClientSession, args *RoleAddPolicyOptions) error {
		result, err := modules.RolesV3.PerformAction(s, args.ID, "add-policy", jsonutils.Marshal(args))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type RoleRemovePolicyOptions struct {
		ID string `json:"-" help:"role id or name to remove policy"`
		api.RolePerformRemovePolicyInput
	}
	R(&RoleRemovePolicyOptions{}, "role-remove-policy", "Remove policy from a role", func(s *mcclient.ClientSession, args *RoleRemovePolicyOptions) error {
		result, err := modules.RolesV3.PerformAction(s, args.ID, "remove-policy", jsonutils.Marshal(args))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type RoleSetPolicyOptions struct {
		ID string `json:"-" help:"role id or name to set policies"`
		api.RolePerformSetPoliciesInput
	}
	R(&RoleSetPolicyOptions{}, "role-set-policies", "Set policies for a role", func(s *mcclient.ClientSession, args *RoleSetPolicyOptions) error {
		result, err := modules.RolesV3.PerformAction(s, args.ID, "set-policies", jsonutils.Marshal(args))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type RolePolicyDeleteOptions struct {
		ID string `json:"-" help:"id or role policy binding"`
	}
	R(&RolePolicyDeleteOptions{}, "role-policy-delete", "Delete role policy binding", func(s *mcclient.ClientSession, args *RolePolicyDeleteOptions) error {
		result, err := modules.RolePolicies.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
