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

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type GroupListOptions struct {
		options.BaseListOptions
		Name          string `help:"Filter by name"`
		OrderByDomain string `help:"order by domain name" choices:"asc|desc"`
	}
	R(&GroupListOptions{}, "group-list", "List groups", func(s *mcclient.ClientSession, args *GroupListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Groups.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Groups.GetColumns(s))
		return nil
	})

	type GroupShowOptions struct {
		ID     string `help:"ID or Name of group"`
		Domain string `help:"Id or Name of domain"`
	}
	R(&GroupShowOptions{}, "group-show", "Show details of a group", func(s *mcclient.ClientSession, args *GroupShowOptions) error {
		params := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(domainId), "domain_id")
		}
		grpId, err := modules.Groups.GetId(s, args.ID, params)
		if err != nil {
			return err
		}
		result, err := modules.Groups.GetById(s, grpId, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	R(&GroupShowOptions{}, "group-user-list", "Show members of a group", func(s *mcclient.ClientSession, args *GroupShowOptions) error {
		params := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(domainId), "domain_id")
		}
		grpId, err := modules.Groups.GetId(s, args.ID, params)
		if err != nil {
			return err
		}
		users, err := modules.Groups.GetUsers(s, grpId)
		if err != nil {
			return err
		}
		printList(users, modules.UsersV3.GetColumns(s))
		return nil
	})

	type GroupCreateOptions struct {
		NAME   string `help:"Name of the group"`
		Desc   string `help:"Description"`
		Domain string `help:"Domain ID or Name"`
	}
	R(&GroupCreateOptions{}, "group-create", "Create a group", func(s *mcclient.ClientSession, args *GroupCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.Domain) > 0 {
			domainId, e := modules.Domains.GetId(s, args.Domain, nil)
			if e != nil {
				return e
			}
			params.Add(jsonutils.NewString(domainId), "domain_id")
		}
		result, e := modules.Groups.Create(s, params)
		if e != nil {
			return e
		}
		printObject(result)
		return nil
	})

	R(&GroupShowOptions{}, "group-delete", "Delete a group", func(s *mcclient.ClientSession, args *GroupShowOptions) error {
		params := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(domainId), "domain_id")
		}
		grpId, err := modules.Groups.GetId(s, args.ID, params)
		if err != nil {
			return err
		}
		result, err := modules.Groups.Delete(s, grpId, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&GroupShowOptions{}, "group-project-list", "List projects of group", func(s *mcclient.ClientSession, args *GroupShowOptions) error {
		query := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "domain_id")
		}
		uid, err := modules.Groups.GetId(s, args.ID, query)
		if err != nil {
			return err
		}
		projects, e := modules.Groups.GetProjects(s, uid)
		if e != nil {
			return e
		}
		printList(projects, modules.Projects.GetColumns(s))
		return nil
	})

	type GroupJoinProjectOptions struct {
		Group   string   `help:"Group Id or name" optional:"false" positional:"true"`
		Project []string `help:"Projects to join" nargs:"+"`
		Role    []string `help:"User join project with roles" nargs:"+"`
	}
	R(&GroupJoinProjectOptions{}, "group-join-project", "Group join projects with roles", func(s *mcclient.ClientSession, args *GroupJoinProjectOptions) error {
		input := api.SJoinProjectsInput{}
		input.Projects = args.Project
		input.Roles = args.Role
		result, err := modules.Groups.PerformAction(s, args.Group, "join", jsonutils.Marshal(input))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type GroupLeaveProjectsOptions struct {
		Group   string   `help:"group id or name" optional:"false" positional:"true"`
		Project string   `help:"project id or name" optional:"false" positional:"true"`
		Role    []string `help:"roles to remove" nargs:"+"`
	}
	R(&GroupLeaveProjectsOptions{}, "group-leave-project", "Leave a group from projects", func(s *mcclient.ClientSession, args *GroupLeaveProjectsOptions) error {
		input := api.SLeaveProjectsInput{}
		input.ProjectRoles = make([]api.SProjectRole, len(args.Role))
		for i := range args.Role {
			input.ProjectRoles[i].Project = args.Project
			input.ProjectRoles[i].Role = args.Role[i]
		}
		result, err := modules.Groups.PerformAction(s, args.Group, "leave", jsonutils.Marshal(input))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
