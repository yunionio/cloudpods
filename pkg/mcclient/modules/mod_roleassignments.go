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

package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type RoleAssignmentManagerV3 struct {
	modulebase.ResourceManager
}

type role struct {
	id   string
	name string
}

type projectRoles struct {
	id    string
	name  string
	typ   string
	roles []role
}

func newProjectRoles(projectId, projectName, roleId, roleName, typ string) *projectRoles {
	return &projectRoles{id: projectId, name: projectName, typ: typ, roles: []role{{id: roleId, name: roleName}}}
}

func (this *projectRoles) add(roleId, roleName string) {
	this.roles = append(this.roles, role{id: roleId, name: roleName})
}

func (this *projectRoles) json() jsonutils.JSONObject {
	obj := jsonutils.NewDict()
	obj.Add(jsonutils.NewString(this.id), "id")
	obj.Add(jsonutils.NewString(this.name), "name")
	obj.Add(jsonutils.NewString(this.typ), "type")
	roles := jsonutils.NewArray()
	for _, r := range this.roles {
		role := jsonutils.NewDict()
		role.Add(jsonutils.NewString(r.id), "id")
		role.Add(jsonutils.NewString(r.name), "name")
		roles.Add(role)
	}
	obj.Add(roles, "roles")
	return obj
}

type sRole struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type sGroupRole struct {
	Id       string  `json:"id"`
	Name     string  `json:"name"`
	Roles    []sRole `json:"roles"`
	Policies struct {
		Project []string `json:"project"`
		Domain  []string `json:"domain"`
		System  []string `json:"system"`
	} `json:"policies"`
}

type sProjectGroupRole struct {
	Id     string       `json:"id"`
	Name   string       `json:"name"`
	Groups []sGroupRole `json:"groups"`
}

func (pgr *sProjectGroupRole) add(groupId, groupName, roleId, roleName string, projectPolicies, domainPolicies, systemPolicies []string) {
	groupIdx := -1
	for i := range pgr.Groups {
		if pgr.Groups[i].Id == groupId {
			groupIdx = i
			break
		}
	}
	if groupIdx < 0 {
		groupIdx = len(pgr.Groups)
		pgr.Groups = append(pgr.Groups, sGroupRole{
			Id:   groupId,
			Name: groupName,
		})
	}
	pgr.Groups[groupIdx].add(roleId, roleName, projectPolicies, domainPolicies, systemPolicies)
}

func (gr *sGroupRole) add(roleId, roleName string, projectPolicies, domainPolicies, systemPolicies []string) {
	gr.Roles = append(gr.Roles, sRole{
		Id:   roleId,
		Name: roleName,
	})
	for _, p := range projectPolicies {
		if !utils.IsInStringArray(p, gr.Policies.Project) {
			gr.Policies.Project = append(gr.Policies.Project, p)
		}
	}
	for _, p := range domainPolicies {
		if !utils.IsInStringArray(p, gr.Policies.Domain) {
			gr.Policies.Domain = append(gr.Policies.Domain, p)
		}
	}
	for _, p := range systemPolicies {
		if !utils.IsInStringArray(p, gr.Policies.System) {
			gr.Policies.System = append(gr.Policies.System, p)
		}
	}
}

var (
	RoleAssignments RoleAssignmentManagerV3
)

// get users for given project
func (this *RoleAssignmentManagerV3) GetProjectUsers(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	query := jsonutils.NewDict()

	if params.Contains("effective") {
		query.Add(jsonutils.JSONNull, "effective")
	}

	if jsonutils.QueryBoolean(params, "system", false) {
		query.Add(jsonutils.JSONNull, "include_system")
	}

	resource, e := params.GetString("resource")
	if e != nil {
		return jsonutils.JSONNull, e
	}

	scope := false
	switch resource {
	case "domain",
		"project":
		{
			scope = true
		}
	case "user",
		"group",
		"role":
		{
			scope = false
		}
	default:
		return jsonutils.JSONNull, fmt.Errorf("not allowed resource %s", resource)
	}

	query.Add(jsonutils.JSONNull, "include_names")

	if scope {
		query.Add(jsonutils.NewString(id), "scope", resource, "id")
	} else {
		query.Add(jsonutils.NewString(id), resource, "id")
	}

	query.Add(jsonutils.JSONNull, "include_policies")

	result, err := this.List(s, query)
	if err != nil {
		return jsonutils.JSONNull, err
	}

	projects := make(map[string]*projectRoles)
	for _, roleAssign := range result.Data {

		typ := "user"

		roleId, _ := roleAssign.GetString("role", "id")
		roleName, _ := roleAssign.GetString("role", "name")
		userId, _ := roleAssign.GetString("user", "id")
		userName, _ := roleAssign.GetString("user", "name")

		if len(userName) == 0 {
			typ = "group"
			userName, _ = roleAssign.GetString("group", "name")
			userId, _ = roleAssign.GetString("group", "id")
		}

		_, ok := projects[userId]

		if ok {
			projects[userId].add(roleId, roleName)
		} else {
			projects[userId] = newProjectRoles(userId, userName, roleId, roleName, typ)
		}
	}

	projJson := jsonutils.NewArray()
	for _, proj := range projects {
		projJson.Add(proj.json())
	}

	data := jsonutils.NewDict()
	data.Add(projJson, "data")
	data.Add(jsonutils.NewInt(int64(len(projects))), "total")
	return data, nil
}

// get projects-roles for given resource, like domain, project, user, group, role
func (this *RoleAssignmentManagerV3) GetProjectRole(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	data := jsonutils.NewDict()
	query := jsonutils.NewDict()

	if params.Contains("effective") {
		query.Add(jsonutils.JSONNull, "effective")
	}

	if jsonutils.QueryBoolean(params, "system", false) {
		query.Add(jsonutils.JSONNull, "include_system")
	}

	resource, err := params.GetString("resource")
	if err != nil {
		return jsonutils.JSONNull, err
	}
	scope := false
	switch resource {
	case "domain",
		"project":
		{
			scope = true
		}
	case "user",
		"group",
		"role":
		{
		}
	default:
		return jsonutils.JSONNull, fmt.Errorf("not allowed resource %s", resource)
	}

	// search by project id or name
	searchProjs := jsonutils.GetQueryStringArray(params, "projects")
	if len(searchProjs) > 0 {
		query.Add(jsonutils.NewStringArray(searchProjs), "projects")
	}
	// search by user id or name
	searchUsers := jsonutils.GetQueryStringArray(params, "users")
	if len(searchUsers) > 0 {
		query.Add(jsonutils.NewStringArray(searchUsers), "users")
	}

	groupBy, _ := params.GetString("group_by")
	if len(groupBy) == 0 {
		groupBy = "project"
	}
	if groupBy == "project" {
	} else if groupBy == "user" {
	} else {
		return nil, errors.Wrapf(httperrors.ErrInputParameter, "unsupported group_by value %s", groupBy)
	}

	query.Add(jsonutils.JSONNull, "include_names")

	if scope {
		query.Add(jsonutils.NewString(id), "scope", resource, "id")
	} else {
		query.Add(jsonutils.NewString(id), resource, "id")
	}
	query.Add(jsonutils.JSONNull, "include_policies")

	result, err := this.List(s, query)
	if err != nil {
		return jsonutils.JSONNull, err
	}

	lines := make([]sProjectGroupRole, 0)
	for _, roleAssign := range result.Data {
		roleId, _ := roleAssign.GetString("role", "id")
		roleName, _ := roleAssign.GetString("role", "name")

		var groupById string
		var groupByName string

		if groupBy == "project" {
			groupById, _ = roleAssign.GetString("scope", "project", "id")
			groupByName, _ = roleAssign.GetString("scope", "project", "name")
		} else if groupBy == "user" {
			groupById, _ = roleAssign.GetString("user", "id")
			groupByName, _ = roleAssign.GetString("user", "name")
		}

		groupId, _ := roleAssign.GetString("group", "id")
		groupName, _ := roleAssign.GetString("group", "name")
		projPolicies, _ := jsonutils.GetStringArray(roleAssign, "policies", "project")
		domPolicies, _ := jsonutils.GetStringArray(roleAssign, "policies", "domain")
		sysPolicies, _ := jsonutils.GetStringArray(roleAssign, "policies", "system")

		lineIdx := -1
		for i := range lines {
			if lines[i].Id == groupById {
				lineIdx = i
				break
			}
		}

		if lineIdx < 0 {
			lineIdx = len(lines)
			lines = append(lines, sProjectGroupRole{
				Id:   groupById,
				Name: groupByName,
			})
		}

		lines[lineIdx].add(groupId, groupName, roleId, roleName, projPolicies, domPolicies, sysPolicies)
	}

	lineJson := jsonutils.NewArray()
	for _, line := range lines {
		lineJson.Add(jsonutils.Marshal(line))
	}
	data.Add(lineJson, "data")
	data.Add(jsonutils.NewInt(int64(len(lines))), "total")
	return data, nil
}

func init() {
	RoleAssignments = RoleAssignmentManagerV3{NewIdentityV3Manager("role_assignment", "role_assignments",
		[]string{"Scope", "User", "Group", "Role", "Policies"},
		[]string{})}
	register(&RoleAssignments)
}
