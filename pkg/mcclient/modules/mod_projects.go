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

	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type ProjectManagerV3 struct {
	modulebase.ResourceManager
}

var (
	Projects ProjectManagerV3
)

func (this *ProjectManagerV3) _join(s *mcclient.ClientSession, pid, uid, rid, resource string) error {
	if resource == "users" {
		_, err := RolesV3.PutInContexts(s, rid, nil, []modulebase.ManagerContext{
			{InstanceManager: &Projects, InstanceId: pid},
			{InstanceManager: &UsersV3, InstanceId: uid},
		})
		if err != nil {
			return err
		}
	} else if resource == "groups" {
		_, err := RolesV3.PutInContexts(s, rid, nil, []modulebase.ManagerContext{
			{InstanceManager: &Projects, InstanceId: pid},
			{InstanceManager: &Groups, InstanceId: uid},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *ProjectManagerV3) _leave(s *mcclient.ClientSession, pid string, resource string, uid string, rid string) error {
	var err error
	if resource == "users" {
		_, err = RolesV3.DeleteInContexts(s, rid, nil, []modulebase.ManagerContext{
			{InstanceManager: &Projects, InstanceId: pid},
			{InstanceManager: &UsersV3, InstanceId: uid},
		})
	} else if resource == "groups" {
		_, err = RolesV3.DeleteInContexts(s, rid, nil, []modulebase.ManagerContext{
			{InstanceManager: &Projects, InstanceId: pid},
			{InstanceManager: &Groups, InstanceId: uid},
		})
	}
	if err != nil {
		return err
	}
	return nil
}

func (this *ProjectManagerV3) DoLeaveProject(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// params format:
	// {
	//     "uid": "CCCGwOsrpp6h", // may be userid, may be group id
	//     "resource": "" default users, else: groups
	//     "pids": [{"rid": "arstarst", "pid": "ooienst"}, {}...]
	// }

	ret := jsonutils.NewDict()
	uid, e := params.GetString("uid")
	if e != nil {
		return nil, httperrors.NewInputParameterError("missing uid")
	}
	pids, e := params.GetArray("pids")
	if e != nil {
		return nil, httperrors.NewInputParameterError("missing pids")
	}

	resource, _ := params.GetString("resource")

	if len(resource) == 0 {
		resource = "users"
	}

	for _, pid := range pids {
		_pid, e := pid.GetString("pid")
		if e != nil {
			return nil, httperrors.NewInputParameterError("missing pid in pids")
		}
		_rid, e := pid.GetString("rid")
		if e != nil {
			return nil, httperrors.NewInputParameterError("missing rid in pids")
		}

		err := this._leave(s, _pid, resource, uid, _rid)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

// Add one user to Many Projects
func (this *ProjectManagerV3) DoJoinProject(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// params format:
	// {
	//     "uid": "CCCGwOsrpp6h",
	//     "rid": "aDeKQx1PmcTd",
	//     "pids": ["L6ssbAJUG3rC", "pu8lkunxP4z8"]
	//     "resource": "users" or "groups"
	// }

	ret := jsonutils.NewDict()
	uid, e := params.GetString("uid")
	if e != nil {
		return nil, httperrors.NewInputParameterError("missing uid")
	}
	ridsA, e := params.Get("rid")
	if e != nil {
		return nil, httperrors.NewInputParameterError("missing rid")
	}
	rids := ridsA.(*jsonutils.JSONArray).GetStringArray()
	pidsA, e := params.Get("pids")
	if e != nil {
		return nil, httperrors.NewInputParameterError("missing pids")
	}
	pids := pidsA.(*jsonutils.JSONArray).GetStringArray()

	resource, _ := params.GetString("resource")

	if len(resource) == 0 {
		resource = "users"
	}

	for _, rid := range rids {
		for _, pid := range pids {
			err := this._join(s, pid, uid, rid, resource)
			if err != nil {
				return nil, err
			}
		}
	}
	return ret, nil
}

// Add Many user[uids] to project(pid) with role(rid)
func (this *ProjectManagerV3) DoProjectBatchJoin(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// params format:
	// {
	//     "uids": ["CCCGwOsrpp6h", "aroisetna"],
	//     "rid": "aDeKQx1PmcTd", // role id
	//     "pid": "L6ssbAJUG3rC"
	// }

	ret := jsonutils.NewDict()
	_ids, e := params.GetArray("ids")

	if e != nil {
		return ret, e
	}
	// ids := make([]string, 0)
	// for _, u := range _ids {
	// 	name, _ := u.GetString()
	// 	ids = append(ids, name)
	// }
	resource, e := params.GetString("resource")
	if e != nil {
		return ret, e
	}
	if resource != "users" && resource != "groups" {
		return ret, fmt.Errorf("不支持的 resource type")
	}
	_rids, e := params.GetArray("rid")
	if e != nil {
		return ret, e
	}
	pid, e := params.GetString("pid")
	if e != nil {
		return ret, e
	}

	for i := range _rids {
		rid, _ := _rids[i].GetString()
		for _, u := range _ids {
			id, _ := u.GetString()
			if resource == "users" {
				_, err := RolesV3.PutInContexts(
					s,
					rid,
					nil,
					[]modulebase.ManagerContext{
						{InstanceManager: &Projects, InstanceId: pid},
						{InstanceManager: &UsersV3, InstanceId: id},
					})
				if err != nil {
					return nil, err
				}
			} else {
				_, err := RolesV3.PutInContexts(
					s,
					rid,
					nil,
					[]modulebase.ManagerContext{
						{InstanceManager: &Projects, InstanceId: pid},
						{InstanceManager: &Groups, InstanceId: id},
					})
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return ret, nil
}

// Remove Many user[uids] to project(pid) with role(rid)
func (this *ProjectManagerV3) DoProjectBatchDeleteUserGroup(s *mcclient.ClientSession, pid string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// params format:
	// {
	//     id: project id;
	//     params: {items: [
	//         id:, role_id:, res_type:
	//     ]}
	// }

	ret := jsonutils.NewDict()
	items, e := params.GetArray("items")

	if e != nil {
		return ret, e
	}

	for _, item := range items {

		id, _ := item.GetString("id")
		role_id, _ := item.GetString("role_id")
		res_type, _ := item.GetString("res_type")

		if res_type == "user" {
			_, err := RolesV3.DeleteInContexts(s, role_id, nil, []modulebase.ManagerContext{
				{InstanceManager: &Projects, InstanceId: pid},
				{InstanceManager: &UsersV3, InstanceId: id},
			})
			if err != nil {
				return nil, err
			}
		} else if res_type == "group" {
			_, err := RolesV3.DeleteInContexts(s, role_id, nil, []modulebase.ManagerContext{
				{InstanceManager: &Projects, InstanceId: pid},
				{InstanceManager: &Groups, InstanceId: id},
			})
			if err != nil {
				return nil, err
			}
		}
	}

	return ret, nil
}

func (this *ProjectManagerV3) AddTags(session *mcclient.ClientSession, id string, tags []string) error {
	path := fmt.Sprintf("/projects/%s/tags", id)
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewStringArray(tags), "tags")
	_, err := modulebase.Put(this.ResourceManager, session, path, body, "")
	if err != nil {
		return err
	}
	return nil
}

func (this *ProjectManagerV3) FetchId(s *mcclient.ClientSession, project string, domain string) (string, error) {
	query := jsonutils.NewDict()
	if len(domain) > 0 {
		domainId, err := Domains.GetId(s, domain, nil)
		if err != nil {
			return "", err
		}
		query.Add(jsonutils.NewString(domainId), "domain_id")
	}
	return this.GetId(s, project, query)
}

func (this *ProjectManagerV3) JoinProject(s *mcclient.ClientSession, rid, pid, uid string) error {
	_, err := RolesV3.PutInContexts(s, rid, nil, []modulebase.ManagerContext{
		{InstanceManager: &Projects, InstanceId: pid},
		{InstanceManager: &UsersV3, InstanceId: uid},
	})
	if err != nil {
		return err
	}

	return nil
}

// create project and attach users & roles
func (this *ProjectManagerV3) DoCreateProject(s *mcclient.ClientSession, p jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	/*
		params format:
		{
			user : ["TestA", "TestB"],
			role : ["RoleA", "RoleB"],
		}
	*/
	params := p.(*jsonutils.JSONDict)
	_user, _ := params.Get("user")
	_role, _ := params.Get("role")
	params.Remove("user")
	params.Remove("role")

	// create project
	response, err := Projects.Create(s, params)
	if err != nil {
		return nil, err
	}

	pid, err := response.GetString("id")
	if err != nil {
		return nil, httperrors.NewResourceNotFoundError("project is not found")
	}

	// assgin users to project
	users := []string{}
	roles := []string{}
	if _user != nil {
		if _u, ok := _user.(*jsonutils.JSONArray); ok {
			users = _u.GetStringArray()
		}
	}

	if _role != nil {
		if _r, ok := _role.(*jsonutils.JSONArray); ok {
			roles = _r.GetStringArray()
		}
	}

	if len(users) > 0 && len(roles) > 0 {
		var projectG errgroup.Group

		for i := range users {
			uid := users[i]
			for j := range roles {
				rid := roles[j]

				projectG.Go(func() error {
					return this.JoinProject(s, rid, pid, uid)
				})
			}
		}

		if err := projectG.Wait(); err != nil {
			return nil, err
		}
	}

	return response, nil
}

func init() {
	Projects = ProjectManagerV3{NewIdentityV3Manager("project", "projects",
		[]string{},
		[]string{"ID", "Name", "Domain_Id", "Project_Domain", "Parent_Id", "Enabled", "Description", "Created_At", "Displayname"})}

	register(&Projects)
}
