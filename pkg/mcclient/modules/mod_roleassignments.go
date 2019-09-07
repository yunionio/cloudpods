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
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"

	"yunion.io/x/onecloud/pkg/mcclient"
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

var (
	RoleAssignments RoleAssignmentManagerV3
)

// get project users for given project
func (this *RoleAssignmentManagerV3) GetProjectUsers(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	query := jsonutils.NewDict()

	effective, e := params.GetString("effective")
	if e == nil && effective == "true" {
		query.Add(jsonutils.JSONNull, "effective")
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

	effective, e := params.GetString("effective")
	if e == nil && effective == "true" {
		query.Add(jsonutils.JSONNull, "effective")
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

	query.Add(jsonutils.JSONNull, "include_names")

	if scope {
		query.Add(jsonutils.NewString(id), "scope", resource, "id")
	} else {
		query.Add(jsonutils.NewString(id), resource, "id")
	}

	result, err := this.List(s, query)
	if err != nil {
		return jsonutils.JSONNull, err
	}

	projects := make(map[string]*projectRoles)
	for _, roleAssign := range result.Data {
		roleId, _ := roleAssign.GetString("role", "id")
		roleName, _ := roleAssign.GetString("role", "name")
		projectId, _ := roleAssign.GetString("scope", "project", "id")
		projectName, _ := roleAssign.GetString("scope", "project", "name")
		_, ok := projects[projectId]

		if ok {
			projects[projectId].add(roleId, roleName)
		} else {
			projects[projectId] = newProjectRoles(projectId, projectName, roleId, roleName, "")
		}
	}

	projJson := jsonutils.NewArray()
	for _, proj := range projects {
		projJson.Add(proj.json())
	}
	data.Add(projJson, "data")
	data.Add(jsonutils.NewInt(int64(len(projects))), "total")
	return data, nil
}

func init() {
	RoleAssignments = RoleAssignmentManagerV3{NewIdentityV3Manager("role_assignment", "role_assignments",
		[]string{"Scope", "User", "Group", "Role", "Policies"},
		[]string{})}
	register(&RoleAssignments)
}
