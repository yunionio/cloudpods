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

	"yunion.io/x/onecloud/cmd/climc/shell"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	identity_options "yunion.io/x/onecloud/pkg/mcclient/options/identity"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.UsersV3)
	cmd.List(&identity_options.UserListOptions{})
	cmd.Perform("user-metadata", &options.ResourceMetadataOptions{})
	cmd.Perform("set-user-metadata", &options.ResourceMetadataOptions{})
	cmd.Perform("class-metadata", &options.ResourceMetadataOptions{})
	cmd.Perform("set-class-metadata", &options.ResourceMetadataOptions{})
	cmd.Perform("metadata", &options.ResourceMetadataOptions{})
	cmd.Perform("enable", &identity_options.UserDetailOptions{})
	cmd.Perform("disable", &identity_options.UserDetailOptions{})

	/*type UserListOptions struct {
		options.BaseListOptions
		Name                    string `help:"Filter by name"`
		OrderByDomain           string `help:"order by domain name" choices:"asc|desc"`
		Role                    string `help:"Filter by role"`
		RoleAssignmentDomainId  string `help:"filter role assignment domain"`
		RoleAssignmentProjectId string `help:"filter role assignment project"`
		IdpId                   string `help:"filter by idp_id"`
		IdpEntityId             string `help:"filter by idp_entity_id"`
	}
	R(&UserListOptions{}, "user-list", "List users", func(s *mcclient.ClientSession, args *UserListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.UsersV3.List(s, params)
		if err != nil {
			return err
		}
		if len(args.ExportFile) > 0 {
			exportList(result, args.ExportFile, args.ExportKeys, args.ExportTexts, modules.UsersV3.GetColumns(s))
		} else {
			printList(result, modules.UsersV3.GetColumns(s))
		}
		return nil
	})*/

	type UserDetailOptions struct {
		ID     string `help:"ID of user"`
		Domain string `help:"Domain"`
		System bool   `help:"show system user"`
	}
	R(&UserDetailOptions{}, "user-show", "Show details of user", func(s *mcclient.ClientSession, args *UserDetailOptions) error {
		query := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "domain_id")
		}
		if args.System {
			query.Add(jsonutils.JSONTrue, "system")
		}
		user, e := modules.UsersV3.Get(s, args.ID, query)
		if e != nil {
			return e
		}
		printObject(user)
		return nil
	})

	R(&UserDetailOptions{}, "user-delete", "Delete user", func(s *mcclient.ClientSession, args *UserDetailOptions) error {
		query := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "domain_id")
		}
		uid, e := modules.UsersV3.GetId(s, args.ID, query)
		if e != nil {
			return e
		}
		_, e = modules.UsersV3.Delete(s, uid, nil)
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
		groups, e := modules.UsersV3.GetGroups(s, uid)
		if e != nil {
			return e
		}
		printList(groups, modules.Groups.GetColumns(s))
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
		printList(modulebase.JSON2ListResult(result), nil)
		return nil
	})

	type UserCreateOptions struct {
		NAME        string  `help:"Name of the new user"`
		Domain      string  `help:"Domain"`
		Desc        string  `help:"Description"`
		Password    *string `help:"Password"`
		Displayname string  `help:"Displayname"`
		Email       string  `help:"Email"`
		Mobile      string  `help:"Mobile"`
		Enabled     bool    `help:"Enabled"`
		Disabled    bool    `help:"Disabled"`

		SkipPasswordComplexityCheck bool `help:"do password complexity check, default is false"`

		// DefaultProject string `help:"Default project"`
		SystemAccount bool `help:"is a system account?"`
		NoWebConsole  bool `help:"allow web console access"`
		EnableMfa     bool `help:"enable TOTP mfa"`

		IdpId       string `help:"Id of identity provider to link with"`
		IdpEntityId string `help:"Entity id of identity provider to link with"`

		Lang string `help:"user default language"`
	}
	R(&UserCreateOptions{}, "user-create", "Create a user", func(s *mcclient.ClientSession, args *UserCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(domainId), "domain_id")
		}
		if args.Password != nil {
			params.Add(jsonutils.NewString(*args.Password), "password")
			if args.SkipPasswordComplexityCheck {
				params.Add(jsonutils.JSONTrue, "skip_password_complexity_check")
			}
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

		if args.SystemAccount {
			params.Add(jsonutils.JSONTrue, "is_system_account")
		}
		if args.NoWebConsole {
			params.Add(jsonutils.JSONFalse, "allow_web_console")
		}
		if args.EnableMfa {
			params.Add(jsonutils.JSONTrue, "enable_mfa")
		}

		if len(args.IdpId) > 0 {
			params.Add(jsonutils.NewString(args.IdpId), "idp_id")
			params.Add(jsonutils.NewString(args.IdpEntityId), "idp_entity_id")
		}

		if len(args.Lang) > 0 {
			params.Add(jsonutils.NewString(args.Lang), "lang")
		}

		/*if len(args.DefaultProject) > 0 {
			projId, err := modules.Projects.GetId(s, args.DefaultProject, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(projId), "default_project_id")
		}*/

		user, err := modules.UsersV3.Create(s, params)
		if err != nil {
			return err
		}
		printObject(user)
		return nil
	})

	type UserUpdateOptions struct {
		ID          string  `help:"ID or name of the user"`
		Domain      string  `help:"Domain"`
		Name        string  `help:"New name of the user"`
		Password    *string `help:"New password"`
		Desc        string  `help:"Description"`
		Displayname string  `help:"Displayname"`
		Email       string  `help:"Email"`
		Mobile      string  `help:"Mobile"`
		Enabled     bool    `help:"Enabled"`
		Disabled    bool    `help:"Disabled"`

		SystemAccount    bool `help:"Turn on is_system_account"`
		NotSystemAccount bool `help:"Turn off is_system_account"`

		AllowWebConsole    bool `help:"Turn on allow_web_console"`
		DisallowWebConsole bool `help:"Turn off allow_web_console"`

		EnableMfa  bool `help:"turn on enable_mfa"`
		DisableMfa bool `help:"turn off enable_mfa"`

		// DefaultProject string `help:"Default project"`
		// Option []string `help:"User options"`

		SkipPasswordComplexityCheck bool `help:"skip_password_complexity_check"`

		Lang string `help:"update user language"`
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
		if args.Password != nil {
			params.Add(jsonutils.NewString(*args.Password), "password")
			if args.SkipPasswordComplexityCheck {
				params.Add(jsonutils.JSONTrue, "skip_password_complexity_check")
			}
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
		if args.SystemAccount {
			params.Add(jsonutils.JSONTrue, "is_system_account")
		} else if args.NotSystemAccount {
			params.Add(jsonutils.JSONFalse, "is_system_account")
		}
		if args.AllowWebConsole {
			params.Add(jsonutils.JSONTrue, "allow_web_console")
		} else if args.DisallowWebConsole {
			params.Add(jsonutils.JSONFalse, "allow_web_console")
		}
		if args.EnableMfa {
			params.Add(jsonutils.JSONTrue, "enable_mfa")
		} else if args.DisableMfa {
			params.Add(jsonutils.JSONFalse, "enable_mfa")
		}
		if len(args.Lang) > 0 {
			params.Add(jsonutils.NewString(args.Lang), "lang")
		}
		// if len(args.DefaultProject) > 0 {
		// 	projId, err := modules.Projects.GetId(s, args.DefaultProject, nil)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	params.Add(jsonutils.NewString(projId), "default_project_id")
		// }
		//
		//   if len(args.Option) > 0 {
		//       uoptions := jsonutils.NewDict()
		//       for _, opt := range args.Option {
		//           pos := strings.IndexByte(opt, ':')
		//           key := opt[:pos]
		//           val := opt[pos+1:]
		//           uoptions.Add(jsonutils.NewString(val), key)
		//       }
		//       params.Add(uoptions, "_resource_options")
		//   }
		//
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

	type UserGroupsOptions struct {
		USER    string   `help:"User ID or Name"`
		Gids    []string `help:"group ID or Name"`
		Action  string   `default:"join" choices:"join|leave"`
		Enabled bool
	}

	R(&UserGroupsOptions{}, "user-join-groups", "Add a user to groups", func(s *mcclient.ClientSession, args *UserGroupsOptions) error {
		_, err := modules.UsersV3.DoJoinGroups(s, args.USER, jsonutils.Marshal(args))
		if err != nil {
			return err
		}
		return nil
	})

	type UserJoinProjectOptions struct {
		User    string   `help:"User Id or name" optional:"false" positional:"true"`
		Project []string `help:"Projects to join" nargs:"+"`
		Role    []string `help:"User join project with roles" nargs:"+"`
		Enabled bool
	}
	R(&UserJoinProjectOptions{}, "user-join-project", "User join projects with roles", func(s *mcclient.ClientSession, args *UserJoinProjectOptions) error {
		input := api.SJoinProjectsInput{}
		input.Projects = args.Project
		input.Roles = args.Role
		input.Enabled = args.Enabled
		result, err := modules.UsersV3.PerformAction(s, args.User, "join", jsonutils.Marshal(input))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type UserLeaveProjectsOptions struct {
		User    string   `help:"user id or name" optional:"false" positional:"true"`
		Project string   `help:"project id or name" optional:"false" positional:"true"`
		Role    []string `help:"roles to remove" nargs:"+"`
	}
	R(&UserLeaveProjectsOptions{}, "user-leave-project", "Leave a user from projects", func(s *mcclient.ClientSession, args *UserLeaveProjectsOptions) error {
		input := api.SLeaveProjectsInput{}
		input.ProjectRoles = make([]api.SProjectRole, len(args.Role))
		for i := range args.Role {
			input.ProjectRoles[i].Project = args.Project
			input.ProjectRoles[i].Role = args.Role[i]
		}
		result, err := modules.UsersV3.PerformAction(s, args.User, "leave", jsonutils.Marshal(input))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type UserLinkIdpOptions struct {
		USER        string `help:"ID or name of user to operate" json:"-"`
		IdpId       string `help:"Id of identity provider to link with" required:"true" json:"idp_id"`
		IdpEntityId string `help:"Id of entity in identity provider to link with" required:"true" json:"idp_entity_id"`
	}
	R(&UserLinkIdpOptions{}, "user-link-idp", "Link user with an entity in the speicified identity provider", func(s *mcclient.ClientSession, args *UserLinkIdpOptions) error {
		result, err := modules.UsersV3.PerformAction(s, args.USER, "link-idp", jsonutils.Marshal(args))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	R(&UserLinkIdpOptions{}, "user-unlink-idp", "Unlink user from an entity in the speicified identity provider", func(s *mcclient.ClientSession, args *UserLinkIdpOptions) error {
		result, err := modules.UsersV3.PerformAction(s, args.USER, "unlink-idp", jsonutils.Marshal(args))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type UserResetCredentialOptions struct {
		USER string `json:"-" help:"ID or name of user to operate"`
		TYPE string `json:"type" help:"Crednetial type of reset" choices:"totp|recovery|aksk|enc_key"`
	}
	R(&UserResetCredentialOptions{}, "user-reset-credentials", "Reset user credential", func(s *mcclient.ClientSession, args *UserResetCredentialOptions) error {
		result, err := modules.UsersV3.PerformAction(s, args.USER, "reset-credentials", jsonutils.Marshal(args))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
