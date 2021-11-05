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
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

func init() {
	type RoleAssignmentsOptions struct {
		Effective     bool     `help:"Include role assignment of group members"`
		System        bool     `help:"Include system user account"`
		Policy        bool     `help:"Show matched policies"`
		Domain        string   `help:"Role assignments for domain"`
		User          string   `help:"For user"`
		UserDomain    string   `help:"Domain for user"`
		Group         string   `help:"For group"`
		GroupDomain   string   `help:"Domain for group"`
		Project       string   `help:"Role assignments for project"`
		ProjectDomain string   `help:"Domain for project"`
		Role          string   `help:"Role assignments for role"`
		RoleDomain    string   `help:"Domain for role"`
		Limit         int64    `help:"maximal returned number of rows"`
		Offset        int64    `help:"offset index of returned results"`
		Users         []string `help:"fitler by user id or name"`
		Groups        []string `help:"fitler by group id or name"`
		Roles         []string `help:"fitler by role id or name"`
		Projects      []string `help:"fitler by project id or name"`
		Domains       []string `help:"fitler by domain id or name"`

		ProjectDomainId string
		ProjectDomains  []string `help:"filter by project's domain id or name"`
	}
	R(&RoleAssignmentsOptions{}, "role-assignments", "List all role assignments", func(s *mcclient.ClientSession, args *RoleAssignmentsOptions) error {
		query := jsonutils.NewDict()
		query.Add(jsonutils.JSONNull, "include_names")
		if args.Effective {
			query.Add(jsonutils.JSONNull, "effective")
		}
		if args.System {
			query.Add(jsonutils.JSONNull, "include_system")
		}
		if args.Policy {
			query.Add(jsonutils.JSONNull, "include_policies")
		}
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "scope", "domain", "id")
		}
		if len(args.Project) > 0 {
			pid, err := getProjectId(s, args.Project, args.ProjectDomain)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(pid), "scope", "project", "id")
		}
		if len(args.ProjectDomainId) > 0 {
			query.Add(jsonutils.NewString(args.ProjectDomainId), "project_domain_id")
		}
		if len(args.User) > 0 {
			uid, err := getUserId(s, args.User, args.UserDomain)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(uid), "user", "id")
		}
		if len(args.Group) > 0 {
			gid, err := getGroupId(s, args.Group, args.GroupDomain)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(gid), "group", "id")
		}
		if len(args.Role) > 0 {
			rid, err := getRoleId(s, args.Role, args.RoleDomain)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(rid), "role", "id")
		}
		if len(args.Users) > 0 {
			query.Add(jsonutils.NewStringArray(args.Users), "users")
		}
		if len(args.Groups) > 0 {
			query.Add(jsonutils.NewStringArray(args.Groups), "groups")
		}
		if len(args.Roles) > 0 {
			query.Add(jsonutils.NewStringArray(args.Roles), "roles")
		}
		if len(args.Projects) > 0 {
			query.Add(jsonutils.NewStringArray(args.Projects), "projects")
		}
		if len(args.Domains) > 0 {
			query.Add(jsonutils.NewStringArray(args.Domains), "domains")
		}
		if len(args.ProjectDomains) > 0 {
			query.Add(jsonutils.NewStringArray(args.ProjectDomains), "project_domains")
		}
		if args.Limit > 0 {
			query.Add(jsonutils.NewInt(args.Limit), "limit")
		}
		if args.Offset > 0 {
			query.Add(jsonutils.NewInt(args.Offset), "offset")
		}
		result, err := modules.RoleAssignments.List(s, query)
		if err != nil {
			return err
		}
		printList(result, modules.RoleAssignments.GetColumns(s))
		return nil
	})
}
