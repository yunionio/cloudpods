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

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type RoleListOptions struct {
		options.BaseListOptions
		OrderByDomain string `help:"order by domain name" choices:"asc|desc"`
	}
	R(&RoleListOptions{}, "role-list", "List keystone Roles", func(s *mcclient.ClientSession, args *RoleListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.RolesV3.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.RolesV3.GetColumns(s))
		return nil
	})

	type RoleDetailOptions struct {
		ID     string `help:"ID or name of role"`
		Domain string `help:"Domain"`
	}
	R(&RoleDetailOptions{}, "role-show", "Show details of a role", func(s *mcclient.ClientSession, args *RoleDetailOptions) error {
		query := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "domain_id")
		}
		role, err := modules.RolesV3.Get(s, args.ID, query)
		if err != nil {
			return err
		}
		printObject(role)
		return nil
	})
	R(&RoleDetailOptions{}, "role-delete", "Delete a role", func(s *mcclient.ClientSession, args *RoleDetailOptions) error {
		query := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "domain_id")
		}
		rid, err := modules.RolesV3.GetId(s, args.ID, query)
		if err != nil {
			return err
		}
		role, err := modules.RolesV3.Delete(s, rid, nil)
		if err != nil {
			return err
		}
		printObject(role)
		return nil
	})

	type RoleCreateOptions struct {
		NAME   string `help:"Role name"`
		Domain string `help:"Domain"`
		Desc   string `help:"Description"`

		PublicScope string `help:"public scope" choices:"none|system"`
	}
	R(&RoleCreateOptions{}, "role-create", "Create a new role", func(s *mcclient.ClientSession, args *RoleCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(domainId), "domain_id")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.PublicScope) > 0 {
			params.Add(jsonutils.NewString(args.PublicScope), "public_scope")
		}
		role, err := modules.RolesV3.Create(s, params)
		if err != nil {
			return err
		}
		printObject(role)
		return nil
	})

	type RoleUpdateOptions struct {
		ID     string `help:"Role ID or Name"`
		Domain string `help:"Domain"`
		Name   string `help:"Name to alter"`
		Desc   string `help:"Description"`
	}
	R(&RoleUpdateOptions{}, "role-update", "Update role", func(s *mcclient.ClientSession, args *RoleUpdateOptions) error {
		query := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "domain_id")
		}
		rid, err := modules.RolesV3.GetId(s, args.ID, query)
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		role, err := modules.RolesV3.Patch(s, rid, params)
		if err != nil {
			return err
		}
		printObject(role)
		return nil
	})

	type RolePerformOptions struct {
		ID string `help:"ID of role to update"`
	}
	R(&RolePerformOptions{}, "role-public", "Mark a role public", func(s *mcclient.ClientSession, args *RolePerformOptions) error {
		result, err := modules.RolesV3.PerformAction(s, args.ID, "public", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&RolePerformOptions{}, "role-private", "Mark a role private", func(s *mcclient.ClientSession, args *RolePerformOptions) error {
		result, err := modules.RolesV3.PerformAction(s, args.ID, "private", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
