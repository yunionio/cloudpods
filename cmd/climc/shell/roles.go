package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type RoleListOptions struct {
		Name   string `help:"Name filter"`
		Domain string `help:"Domain filter"`
		Limit  int64  `help:"Items per page" default:"20"`
		Offset int64  `help:"Offset"`
		Search string `help:"search text"`
	}
	R(&RoleListOptions{}, "role-list", "List keystone Roles", func(s *mcclient.ClientSession, args *RoleListOptions) error {
		mod, err := modules.GetModule(s, "roles")
		if err != nil {
			return err
		}
		query := jsonutils.NewDict()
		if len(args.Name) > 0 {
			query.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "domain_id")
		}
		if args.Limit > 0 {
			query.Add(jsonutils.NewInt(args.Limit), "limit")
		}
		if args.Offset > 0 {
			query.Add(jsonutils.NewInt(args.Offset), "offset")
		}
		if len(args.Search) > 0 {
			query.Add(jsonutils.NewString(args.Search), "name__icontains")
		}
		result, err := mod.List(s, query)
		if err != nil {
			return err
		}
		printList(result, mod.GetColumns(s))
		return nil
	})

	type RoleDetailOptions struct {
		ID     string `help:"ID or name of role"`
		Domain string `help:"Domain"`
	}
	R(&RoleDetailOptions{}, "role-show", "Show details of a role", func(s *mcclient.ClientSession, args *RoleDetailOptions) error {
		mod, err := modules.GetModule(s, "roles")
		if err != nil {
			return err
		}
		query := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "domain_id")
		}
		role, err := mod.Get(s, args.ID, query)
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
		mod, err := modules.GetModule(s, "roles")
		if err != nil {
			return err
		}
		rid, err := mod.GetId(s, args.ID, query)
		if err != nil {
			return err
		}
		role, err := mod.Delete(s, rid, nil)
		if err != nil {
			return err
		}
		printObject(role)
		return nil
	})

	type RoleCreateOptions struct {
		NAME   string `help:"Role name"`
		Domain string `help:"Domain"`
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
		mod, err := modules.GetModule(s, "roles")
		if err != nil {
			return err
		}
		role, err := mod.Create(s, params)
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
		mod, err := modules.GetModule(s, "roles")
		if err != nil {
			return err
		}
		rid, err := mod.GetId(s, args.ID, query)
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		role, err := mod.Patch(s, rid, params)
		if err != nil {
			return err
		}
		printObject(role)
		return nil
	})
}
