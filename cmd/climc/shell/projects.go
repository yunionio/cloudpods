package shell

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/mcclient/modules"
)

func init() {
	type ProjectListOptions struct {
		Domain string `help:"Domain ID or Name"`
		Search string `help:"Search project name"`
		Limit  int64  `help:"Items per page" default:"20"`
		Offset int64  `help:"Offset"`
	}
	R(&ProjectListOptions{}, "project-list", "List projects", func(s *mcclient.ClientSession, args *ProjectListOptions) error {
		params := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(domainId), "domain_id")
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
			params.Add(jsonutils.JSONTrue, "disabled")
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
		_, err = modules.RolesV3.PutInContexts(s, rid, nil, []modules.ManagerContext{{InstanceManager: &modules.Projects, InstanceId: pid}, {InstanceManager: &modules.UsersV3, InstanceId: uid}})
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

		_, err = modules.RolesV3.HeadInContexts(s, rid, nil, []modules.ManagerContext{{InstanceManager: &modules.Projects, InstanceId: pid}, {InstanceManager: &modules.UsersV3, InstanceId: uid}})
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
		_, err = modules.RolesV3.DeleteInContexts(s, rid, nil, []modules.ManagerContext{{InstanceManager: &modules.Projects, InstanceId: pid}, {InstanceManager: &modules.UsersV3, InstanceId: uid}})
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
		result, err := modules.RolesV3.ListInContexts(s, nil, []modules.ManagerContext{{InstanceManager: &modules.Projects, InstanceId: pid}, {InstanceManager: &modules.UsersV3, InstanceId: uid}})
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
		_, err = modules.RolesV3.PutInContexts(s, rid, nil, []modules.ManagerContext{{InstanceManager: &modules.Projects, InstanceId: pid}, {InstanceManager: &modules.Groups, InstanceId: gid}})
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
		_, err = modules.RolesV3.HeadInContexts(s, rid, nil, []modules.ManagerContext{{InstanceManager: &modules.Projects, InstanceId: pid}, {InstanceManager: &modules.Groups, InstanceId: gid}})
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
		_, err = modules.RolesV3.DeleteInContexts(s, rid, nil, []modules.ManagerContext{{InstanceManager: &modules.Projects, InstanceId: pid}, {InstanceManager: &modules.Groups, InstanceId: gid}})
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
		result, err := modules.RolesV3.ListInContexts(s, nil, []modules.ManagerContext{{InstanceManager: &modules.Projects, InstanceId: pid}, {InstanceManager: &modules.Groups, InstanceId: gid}})
		if err != nil {
			return err
		}
		printList(result, modules.RolesV3.GetColumns(s))
		return nil
	})

	R(&ProjectShowOptions{}, "project-shared-images", "Show shared images of a project", func(s *mcclient.ClientSession, args *ProjectShowOptions) error {
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
}
