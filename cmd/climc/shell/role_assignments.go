package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type RoleAssignmentsOptions struct {
		Effective     bool   `help:"Include role assignment of group members"`
		Domain        string `help:"Role assignments for domain"`
		User          string `help:"For user"`
		UserDomain    string `help:"Domain for user"`
		Group         string `help:"For group"`
		GroupDomain   string `help:"Domain for group"`
		Project       string `help:"Role assignments for project"`
		ProjectDomain string `help:"Domain for project"`
		Role          string `help:"Role assignments for role"`
		RoleDomain    string `help:"Domain for role"`
	}
	R(&RoleAssignmentsOptions{}, "role-assignments", "List all role assignments", func(s *mcclient.ClientSession, args *RoleAssignmentsOptions) error {
		query := jsonutils.NewDict()
		query.Add(jsonutils.JSONNull, "include_names")
		if args.Effective {
			query.Add(jsonutils.JSONNull, "effective")
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
		result, err := modules.RoleAssignments.List(s, query)
		if err != nil {
			return err
		}
		printList(result, modules.RoleAssignments.GetColumns(s))
		return nil
	})
}
