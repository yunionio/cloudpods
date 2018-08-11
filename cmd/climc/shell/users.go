package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type UserListOptions struct {
		Domain string `help:"Filter by domain"`
		Name   string `help:"Filter by name"`
		Limit  int64  `help:"Limit, default 0, i.e. no limit"`
		Offset int64  `help:"Offset, default 0, i.e. no offset"`
		Search string `help:"Search by name"`
	}
	R(&UserListOptions{}, "user-list", "List users", func(s *mcclient.ClientSession, args *UserListOptions) error {
		mod, err := modules.GetModule(s, "users")
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(domainId), "domain_id")
		}
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Search) > 0 {
			params.Add(jsonutils.NewString(args.Search), "name__icontains")
		}
		if args.Limit > 0 {
			params.Add(jsonutils.NewInt(args.Limit), "limit")
		}
		if args.Offset > 0 {
			params.Add(jsonutils.NewInt(args.Offset), "offset")
		}
		result, err := mod.List(s, params)
		if err != nil {
			return err
		}
		printList(result, mod.GetColumns(s))
		return nil
	})

	type UserDetailOptions struct {
		ID     string `help:"ID of user"`
		Domain string `help:"Domain"`
	}
	R(&UserDetailOptions{}, "user-show", "Show details of user", func(s *mcclient.ClientSession, args *UserDetailOptions) error {
		mod, e := modules.GetModule(s, "users")
		if e != nil {
			return e
		}
		query := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "domain_id")
		}
		user, e := mod.Get(s, args.ID, query)
		if e != nil {
			return e
		}
		printObject(user)
		return nil
	})
	R(&UserDetailOptions{}, "user-delete", "Delete user", func(s *mcclient.ClientSession, args *UserDetailOptions) error {
		mod, e := modules.GetModule(s, "users")
		if e != nil {
			return e
		}
		query := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "domain_id")
		}
		uid, e := mod.GetId(s, args.ID, query)
		if e != nil {
			return e
		}
		_, e = mod.Delete(s, uid, nil)
		if e != nil {
			return e
		}
		return nil
	})

	R(&UserDetailOptions{}, "user-project-list", "List projects of user", func(s *mcclient.ClientSession, args *UserDetailOptions) error {
		query := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "domain_id")
		}
		uid, err := modules.UsersV3.GetId(s, args.ID, query)
		if err != nil {
			return err
		}
		projects, e := modules.UsersV3.GetProjects(s, uid)
		if e != nil {
			return e
		}
		printList(projects, modules.Projects.GetColumns(s))
		return nil
	})

	R(&UserDetailOptions{}, "user-group-list", "List groups of user", func(s *mcclient.ClientSession, args *UserDetailOptions) error {
		query := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "domain_id")
		}
		uid, err := modules.UsersV3.GetId(s, args.ID, query)
		if err != nil {
			return err
		}
		projects, e := modules.UsersV3.GetGroups(s, uid)
		if e != nil {
			return e
		}
		printList(projects, modules.Groups.GetColumns(s))
		return nil
	})

	type UserTenantRoleOptions struct {
		ID     string `help:"ID of user"`
		Tenant string `help:"ID of tenant"`
	}
	R(&UserTenantRoleOptions{}, "user-role-list", "List roles of user", func(s *mcclient.ClientSession, args *UserTenantRoleOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.ID), "id")
		if len(args.Tenant) > 0 {
			params.Add(jsonutils.NewString(args.Tenant), "tenantId")
		}
		result, err := modules.Users.GetTenantRoleList(s, params)
		if err != nil {
			return err
		}
		printList(modules.JSON2ListResult(result), nil)
		return nil
	})

	type UserCreateOptions struct {
		NAME        string `help:"Name of the new user"`
		Domain      string `help:"Domain"`
		Desc        string `help:"Description"`
		Password    string `help:"Password"`
		Displayname string `help:"Displayname"`
		Email       string `help:"Email"`
		Mobile      string `help:"Mobile"`
		Enabled     bool   `help:"Enabled"`
		Disabled    bool   `help:"Disabled"`
	}
	R(&UserCreateOptions{}, "user-create", "Create a user", func(s *mcclient.ClientSession, args *UserCreateOptions) error {
		mod, err := modules.GetModule(s, "users")
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(domainId), "domain_id")
		}
		if len(args.Password) > 0 {
			params.Add(jsonutils.NewString(args.Password), "password")
		}
		if len(args.Displayname) > 0 {
			params.Add(jsonutils.NewString(args.Displayname), "displayname")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.Email) > 0 {
			params.Add(jsonutils.NewString(args.Email), "email")
		}
		if len(args.Mobile) > 0 {
			params.Add(jsonutils.NewString(args.Mobile), "mobile")
		}
		if args.Enabled && !args.Disabled {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if !args.Enabled && args.Disabled {
			params.Add(jsonutils.JSONFalse, "enabled")
		}
		user, err := mod.Create(s, params)
		if err != nil {
			return err
		}
		printObject(user)
		return nil
	})

	type UserUpdateOptions struct {
		ID          string `help:"ID or name of the user"`
		Domain      string `help:"Domain"`
		Name        string `help:"New name of the user"`
		Password    string `help:"New password"`
		Desc        string `help:"Description"`
		Displayname string `help:"Displayname"`
		Email       string `help:"Email"`
		Mobile      string `help:"Mobile"`
		Enabled     bool   `help:"Enabled"`
		Disabled    bool   `help:"Disabled"`
		// Option []string `help:"User options"`
	}
	R(&UserUpdateOptions{}, "user-update", "Update a user", func(s *mcclient.ClientSession, args *UserUpdateOptions) error {
		query := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "domain_id")
		}
		uid, err := modules.UsersV3.GetId(s, args.ID, query)
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Password) > 0 {
			params.Add(jsonutils.NewString(args.Password), "password")
		}
		if len(args.Displayname) > 0 {
			params.Add(jsonutils.NewString(args.Displayname), "displayname")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.Email) > 0 {
			params.Add(jsonutils.NewString(args.Email), "email")
		}
		if len(args.Mobile) > 0 {
			params.Add(jsonutils.NewString(args.Mobile), "mobile")
		}
		if args.Enabled && !args.Disabled {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if !args.Enabled && args.Disabled {
			params.Add(jsonutils.JSONFalse, "enabled")
		}
		/*
		   if len(args.Option) > 0 {
		       uoptions := jsonutils.NewDict()
		       for _, opt := range args.Option {
		           pos := strings.IndexByte(opt, ':')
		           key := opt[:pos]
		           val := opt[pos+1:]
		           uoptions.Add(jsonutils.NewString(val), key)
		       }
		       params.Add(uoptions, "_resource_options")
		   }
		*/
		user, err := modules.UsersV3.Patch(s, uid, params)
		if err != nil {
			return err
		}
		printObject(user)
		return nil
	})

	type UserGroupOptions struct {
		USER   string `help:"User ID or Name"`
		GROUP  string `help:"group ID or Name"`
		Domain string `help:"Domain"`
	}
	R(&UserGroupOptions{}, "user-join-group", "Add a user to a group", func(s *mcclient.ClientSession, args *UserGroupOptions) error {
		uid, gid, err := getUserGroupId(s, args.USER, args.GROUP, args.Domain)
		if err != nil {
			return err
		}
		_, err = modules.UsersV3.PutInContext(s, uid, nil, &modules.Groups, gid)
		if err != nil {
			return err
		}
		return nil
	})
	R(&UserGroupOptions{}, "user-in-group", "Check whether a user belongs a group", func(s *mcclient.ClientSession, args *UserGroupOptions) error {
		uid, gid, err := getUserGroupId(s, args.USER, args.GROUP, args.Domain)
		if err != nil {
			return err
		}
		_, err = modules.UsersV3.HeadInContext(s, uid, nil, &modules.Groups, gid)
		if err != nil {
			return err
		}
		return nil
	})
	R(&UserGroupOptions{}, "user-leave-group", "Remove a user from a group", func(s *mcclient.ClientSession, args *UserGroupOptions) error {
		uid, gid, err := getUserGroupId(s, args.USER, args.GROUP, args.Domain)
		if err != nil {
			return err
		}
		_, err = modules.UsersV3.DeleteInContext(s, uid, nil, &modules.Groups, gid)
		if err != nil {
			return err
		}
		return nil
	})

}
