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
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ProjectListOptions struct {
		options.BaseListOptions
		OrderByDomain string `help:"order by domain name" choices:"asc|desc"`
	}
	R(&ProjectListOptions{}, "project-list", "List projects", func(s *mcclient.ClientSession, args *ProjectListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Projects.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Projects.GetColumns(s))
		return nil
	})
	type ProjectShowOptions struct {
		ID     string `help:"ID or Name of project"`
		Domain string `help:"Domain"`
	}
	R(&ProjectShowOptions{}, "project-show", "Show details of project", func(s *mcclient.ClientSession, args *ProjectShowOptions) error {
		query := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "domain_id")
		}
		result, err := modules.Projects.Get(s, args.ID, query)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	R(&ProjectShowOptions{}, "project-delete", "Delete a project", func(s *mcclient.ClientSession, args *ProjectShowOptions) error {
		query := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "domain_id")
		}
		projectId, err := modules.Projects.GetId(s, args.ID, query)
		if err != nil {
			return err
		}
		_, err = modules.Projects.Delete(s, projectId, nil)
		if err != nil {
			return err
		}
		return nil
	})

	type ProjectCreateOptions struct {
		NAME     string `help:"Name of new project"`
		Domain   string `help:"Domain"`
		Desc     string `help:"Description"`
		Enabled  bool   `help:"Project is enabled"`
		Disabled bool   `help:"Project is disabled"`
	}
	R(&ProjectCreateOptions{}, "project-create", "Create a project", func(s *mcclient.ClientSession, args *ProjectCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(domainId), "domain_id")
		}
		if args.Enabled && !args.Disabled {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if !args.Enabled && args.Disabled {
			params.Add(jsonutils.JSONTrue, "disabled")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		result, err := modules.Projects.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ProjectUpdateOptions struct {
		ID       string `help:"ID or name of the project to update"`
		Domain   string `help:"Domain of the project if name is given"`
		Name     string `help:"New name of the project"`
		Desc     string `help:"Description"`
		Enabled  bool   `help:"Project is enabled"`
		Disabled bool   `help:"Project is disabled"`
	}
	R(&ProjectUpdateOptions{}, "project-update", "Update a project", func(s *mcclient.ClientSession, args *ProjectUpdateOptions) error {
		query := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "domain_id")
		}
		pId, err := modules.Projects.GetId(s, args.ID, query)
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if args.Enabled && !args.Disabled {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if !args.Enabled && args.Disabled {
			params.Add(jsonutils.JSONFalse, "enabled")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		project, err := modules.Projects.Patch(s, pId, params)
		if err != nil {
			return err
		}
		printObject(project)
		return nil
	})

	type ProjectUserRoleOptions struct {
		PROJECT       string `help:"ID or Name of Project"`
		USER          string `help:"ID or Name of User"`
		ROLE          string `help:"ID or Name of Role"`
		UserDomain    string `help:"Domain of user"`
		ProjectDomain string `help:"Domain of project"`
		RoleDomain    string `help:"Domain of role"`
	}
	R(&ProjectUserRoleOptions{}, "project-add-user", "Add user to project with role", func(s *mcclient.ClientSession, args *ProjectUserRoleOptions) error {
		uid, err := getUserId(s, args.USER, args.UserDomain)
		if err != nil {
			return err
		}
		pid, err := getProjectId(s, args.PROJECT, args.ProjectDomain)
		if err != nil {
			return err
		}
		rid, err := getRoleId(s, args.ROLE, args.RoleDomain)
		if err != nil {
			return err
		}
		_, err = modules.RolesV3.PutInContexts(s, rid, nil, []modulebase.ManagerContext{{InstanceManager: &modules.Projects, InstanceId: pid}, {InstanceManager: &modules.UsersV3, InstanceId: uid}})
		if err != nil {
			return err
		}
		return nil
	})
	R(&ProjectUserRoleOptions{}, "project-has-user", "Check a user in a project with a role", func(s *mcclient.ClientSession, args *ProjectUserRoleOptions) error {
		uid, err := getUserId(s, args.USER, args.UserDomain)
		if err != nil {
			return err
		}
		pid, err := getProjectId(s, args.PROJECT, args.ProjectDomain)
		if err != nil {
			return err
		}
		rid, err := getRoleId(s, args.ROLE, args.RoleDomain)
		if err != nil {
			return err
		}

		_, err = modules.RolesV3.HeadInContexts(s, rid, nil, []modulebase.ManagerContext{{InstanceManager: &modules.Projects, InstanceId: pid}, {InstanceManager: &modules.UsersV3, InstanceId: uid}})
		if err != nil {
			return err
		}
		return nil
	})
	R(&ProjectUserRoleOptions{}, "project-remove-user", "Remove a user role from a project", func(s *mcclient.ClientSession, args *ProjectUserRoleOptions) error {
		uid, err := getUserId(s, args.USER, args.UserDomain)
		if err != nil {
			return err
		}
		pid, err := getProjectId(s, args.PROJECT, args.ProjectDomain)
		if err != nil {
			return err
		}
		rid, err := getRoleId(s, args.ROLE, args.RoleDomain)
		if err != nil {
			return err
		}
		_, err = modules.RolesV3.DeleteInContexts(s, rid, nil, []modulebase.ManagerContext{{InstanceManager: &modules.Projects, InstanceId: pid}, {InstanceManager: &modules.UsersV3, InstanceId: uid}})
		if err != nil {
			return err
		}
		return nil
	})

	type ProjectUserRolesListOptions struct {
		PROJECT       string `help:"ID or Name of Project"`
		USER          string `help:"ID or Name of User"`
		UserDomain    string `help:"Domain of user"`
		ProjectDomain string `help:"Domain of project"`
	}
	R(&ProjectUserRolesListOptions{}, "project-user-roles", "Get roles of a user in a project", func(s *mcclient.ClientSession, args *ProjectUserRolesListOptions) error {
		uid, err := getUserId(s, args.USER, args.UserDomain)
		if err != nil {
			return err
		}
		pid, err := getProjectId(s, args.PROJECT, args.ProjectDomain)
		if err != nil {
			return err
		}
		result, err := modules.RolesV3.ListInContexts(s, nil, []modulebase.ManagerContext{{InstanceManager: &modules.Projects, InstanceId: pid}, {InstanceManager: &modules.UsersV3, InstanceId: uid}})
		if err != nil {
			return err
		}
		printList(result, modules.RolesV3.GetColumns(s))
		return nil
	})

	type ProjectGroupRoleOptions struct {
		PROJECT       string `help:"ID or Name of Project"`
		GROUP         string `help:"ID or Name of Group"`
		ROLE          string `help:"ID or Name of Role"`
		GroupDomain   string `help:"Domain of group"`
		ProjectDomain string `help:"Domain of project"`
		RoleDomain    string `help:"Domain of role"`
	}
	R(&ProjectGroupRoleOptions{}, "project-add-group", "Add group to project with role", func(s *mcclient.ClientSession, args *ProjectGroupRoleOptions) error {
		gid, err := getGroupId(s, args.GROUP, args.GroupDomain)
		if err != nil {
			return err
		}
		pid, err := getProjectId(s, args.PROJECT, args.ProjectDomain)
		if err != nil {
			return err
		}
		rid, err := getRoleId(s, args.ROLE, args.RoleDomain)
		if err != nil {
			return err
		}
		_, err = modules.RolesV3.PutInContexts(s, rid, nil, []modulebase.ManagerContext{{InstanceManager: &modules.Projects, InstanceId: pid}, {InstanceManager: &modules.Groups, InstanceId: gid}})
		if err != nil {
			return err
		}
		return nil
	})
	R(&ProjectGroupRoleOptions{}, "project-has-group", "Check a group in a project with a role", func(s *mcclient.ClientSession, args *ProjectGroupRoleOptions) error {
		gid, err := getGroupId(s, args.GROUP, args.GroupDomain)
		if err != nil {
			return err
		}
		pid, err := getProjectId(s, args.PROJECT, args.ProjectDomain)
		if err != nil {
			return err
		}
		rid, err := getRoleId(s, args.ROLE, args.RoleDomain)
		if err != nil {
			return err
		}
		_, err = modules.RolesV3.HeadInContexts(s, rid, nil, []modulebase.ManagerContext{{InstanceManager: &modules.Projects, InstanceId: pid}, {InstanceManager: &modules.Groups, InstanceId: gid}})
		if err != nil {
			return err
		}
		return nil
	})
	R(&ProjectGroupRoleOptions{}, "project-remove-group", "Remove a role for a group in a project", func(s *mcclient.ClientSession, args *ProjectGroupRoleOptions) error {
		gid, err := getGroupId(s, args.GROUP, args.GroupDomain)
		if err != nil {
			return err
		}
		pid, err := getProjectId(s, args.PROJECT, args.ProjectDomain)
		if err != nil {
			return err
		}
		rid, err := getRoleId(s, args.ROLE, args.RoleDomain)
		if err != nil {
			return err
		}
		_, err = modules.RolesV3.DeleteInContexts(s, rid, nil, []modulebase.ManagerContext{{InstanceManager: &modules.Projects, InstanceId: pid}, {InstanceManager: &modules.Groups, InstanceId: gid}})
		if err != nil {
			return err
		}
		return nil
	})
	type ProjectGroupRolesListOptions struct {
		PROJECT       string `help:"ID or Name of Project"`
		GROUP         string `help:"ID or Name of Group"`
		GroupDomain   string `help:"Domain of group"`
		ProjectDomain string `help:"Domain of project"`
	}
	R(&ProjectGroupRolesListOptions{}, "project-group-roles", "Get roles for group in project", func(s *mcclient.ClientSession, args *ProjectGroupRolesListOptions) error {
		gid, err := getGroupId(s, args.GROUP, args.GroupDomain)
		if err != nil {
			return err
		}
		pid, err := getProjectId(s, args.PROJECT, args.ProjectDomain)
		if err != nil {
			return err
		}
		result, err := modules.RolesV3.ListInContexts(s, nil, []modulebase.ManagerContext{{InstanceManager: &modules.Projects, InstanceId: pid}, {InstanceManager: &modules.Groups, InstanceId: gid}})
		if err != nil {
			return err
		}
		printList(result, modules.RolesV3.GetColumns(s))
		return nil
	})

	/*R(&ProjectShowOptions{}, "project-shared-images", "Show shared images of a project", func(s *mcclient.ClientSession, args *ProjectShowOptions) error {
		query := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "domain_id")
		}
		projectId, err := modules.Projects.GetId(s, args.ID, query)
		if err != nil {
			return err
		}
		imgs, err := modules.Images.ListSharedImages(s, projectId)
		if err != nil {
			return err
		}
		printList(imgs, modules.Images.GetColumns(s))
		return nil
	})

	type ProjectAddTagsOptions struct {
		ID   string   `help:"ID or name of project"`
		Tags []string `help:"tags added to project"`
	}
	R(&ProjectAddTagsOptions{}, "project-add-tags", "Add project with tags", func(s *mcclient.ClientSession, args *ProjectAddTagsOptions) error {
		err := modules.Projects.AddTags(s, args.ID, args.Tags)
		if err != nil {
			return err
		}
		return nil
	})*/

	// Deprecated
	type ProjectBatchJoinOptions struct {
		Ids      []string `help:"user ids or group ids"`
		Resource string   `help:"resource type" choices:"users|groups"`
		Rid      string   `help:"role id"`
		Pid      string   `help:"project id"`
	}
	R(&ProjectBatchJoinOptions{}, "project-batch-join", "Batch join users or groups into project with role", func(s *mcclient.ClientSession, args *ProjectBatchJoinOptions) error {
		params := jsonutils.Marshal(args)
		_, err := modules.Projects.DoProjectBatchJoin(s, params)
		if err != nil {
			return err
		}
		return nil
	})

	type ProjectAddUserGroupOptions struct {
		Project string   `help:"ID or name of project to add users/groups" positional:"true" optional:"false"`
		User    []string `help:"ID of user to add"`
		Group   []string `help:"ID of group to add"`
		Role    []string `help:"ID of role to add"`
	}
	R(&ProjectAddUserGroupOptions{}, "project-add-user-group", "Batch add users/groups to project", func(s *mcclient.ClientSession, args *ProjectAddUserGroupOptions) error {
		input := api.SProjectAddUserGroupInput{}
		input.Users = args.User
		input.Groups = args.Group
		input.Roles = args.Role
		err := input.Validate()
		if err != nil {
			return err
		}
		result, err := modules.Projects.PerformAction(s, args.Project, "join", jsonutils.Marshal(input))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ProjectRemoveUserGroup struct {
		Project string   `help:"ID or name of project to remove user/group" optional:"false" positional:"true"`
		User    string   `help:"user to remove"`
		Group   string   `help:"group to remove"`
		Role    []string `help:"roles to remove"`
	}
	R(&ProjectRemoveUserGroup{}, "project-remove-user-group", "Remove users/groups from project", func(s *mcclient.ClientSession, args *ProjectRemoveUserGroup) error {
		input := api.SProjectRemoveUserGroupInput{}
		input.UserRoles = make([]api.SUserRole, len(args.Role))
		input.GroupRoles = make([]api.SGroupRole, len(args.Role))
		for i := range args.Role {
			input.UserRoles[i].User = args.User
			input.UserRoles[i].Role = args.Role[i]
			input.GroupRoles[i].Group = args.Group
			input.GroupRoles[i].Role = args.Role[i]
		}
		err := input.Validate()
		if err != nil {
			return err
		}
		result, err := modules.Projects.PerformAction(s, args.Project, "leave", jsonutils.Marshal(input))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
