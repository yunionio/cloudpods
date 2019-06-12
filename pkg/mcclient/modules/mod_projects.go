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
	"sync"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type ProjectManagerV3 struct {
	ResourceManager
}

var (
	Projects ProjectManagerV3
)

func (this *ProjectManagerV3) _join(s *mcclient.ClientSession, pid, uid, rid, resource string, ch chan int) error {
	defer func() {
		ch <- 1
	}()
	if resource == "users" {
		_, err := RolesV3.PutInContexts(s, rid, nil, []ManagerContext{{&Projects, pid}, {&UsersV3, uid}})
		if err != nil {
			return err
		}
	} else if resource == "groups" {
		_, err := RolesV3.PutInContexts(s, rid, nil, []ManagerContext{{&Projects, pid}, {&Groups, uid}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *ProjectManagerV3) _leave(s *mcclient.ClientSession, pid string, resource string, uid string, rid string, ch chan int) error {
	defer func() {
		ch <- 1
	}()
	var err error
	if resource == "users" {
		_, err = RolesV3.DeleteInContexts(s, rid, nil, []ManagerContext{{&Projects, pid}, {&UsersV3, uid}})
	} else if resource == "groups" {
		_, err = RolesV3.DeleteInContexts(s, rid, nil, []ManagerContext{{&Projects, pid}, {&Groups, uid}})
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
		return ret, e
	}
	pids, e := params.GetArray("pids")
	if e != nil {
		return ret, e
	}

	resource, _ := params.GetString("resource")

	if len(resource) == 0 {
		resource = "users"
	}

	chs := make([]chan int, len(pids))

	for i, pid := range pids {
		_pid, e := pid.GetString("pid")
		if e != nil {
			return ret, e
		}
		_rid, e := pid.GetString("rid")
		if e != nil {
			return ret, e
		}

		chs[i] = make(chan int)
		go this._leave(s, _pid, resource, uid, _rid, chs[i])
	}

	for _, ch := range chs {
		<-ch
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
		return ret, e
	}
	ridsA, e := params.Get("rid")
	if e != nil {
		return ret, e
	}
	rids := ridsA.(*jsonutils.JSONArray).GetStringArray()
	pidsA, e := params.Get("pids")
	if e != nil {
		return ret, e
	}
	pids := pidsA.(*jsonutils.JSONArray).GetStringArray()

	resource, _ := params.GetString("resource")

	if len(resource) == 0 {
		resource = "users"
	}

	chs := make([]chan int, len(pids))

	for _, rid := range rids {
		for i, pid := range pids {
			chs[i] = make(chan int)
			go this._join(s, pid, uid, rid, resource, chs[i])
		}

		for _, ch := range chs {
			<-ch
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
	ids := make([]string, 0)
	for _, u := range _ids {
		name, _ := u.GetString()
		ids = append(ids, name)
	}
	resource, e := params.GetString("resource")
	if e != nil {
		return ret, e
	}
	if resource != "users" && resource != "groups" {
		return ret, fmt.Errorf("不支持的 resource type")
	}
	rid, e := params.GetString("rid")
	if e != nil {
		return ret, e
	}
	pid, e := params.GetString("pid")
	if e != nil {
		return ret, e
	}

	BatchDo(ids, func(id string) (jsonutils.JSONObject, error) {
		if resource == "users" {
			return RolesV3.PutInContexts(
				s,
				rid,
				nil,
				[]ManagerContext{
					{&Projects,
						pid},
					{&UsersV3, id}})
		}
		return RolesV3.PutInContexts(
			s,
			rid,
			nil,
			[]ManagerContext{
				{&Projects,
					pid},
				{&Groups, id}})

	})

	return ret, nil
}

// Add Many user[uids] to project(pid) with role(rid)
func (this *ProjectManagerV3) DoProjectBatchDeleteUserGroup(s *mcclient.ClientSession, pid string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// params format:
	// {
	//     id: project id;
	//     params: {items: [
	//         id:, role_id:, res_type:
	//     ]}
	// }

	var wg sync.WaitGroup
	ret := jsonutils.NewDict()
	items, e := params.GetArray("items")

	if e != nil {
		return ret, e
	}

	for _, item := range items {

		wg.Add(1)
		id, _ := item.GetString("id")
		role_id, _ := item.GetString("role_id")
		res_type, _ := item.GetString("res_type")

		go func(id string) {
			defer wg.Done()
			if res_type == "user" {
				RolesV3.DeleteInContexts(s, role_id, nil, []ManagerContext{{&Projects, pid}, {&UsersV3, id}})
			} else if res_type == "group" {
				RolesV3.DeleteInContexts(s, role_id, nil, []ManagerContext{{&Projects, pid}, {&Groups, id}})
			}
		}(id)
	}

	wg.Wait()
	return ret, nil
}

func (this *ProjectManagerV3) Delete(session *mcclient.ClientSession, id string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.DeleteInContexts(session, id, body, nil)
}

func (this *ProjectManagerV3) DeleteInContexts(session *mcclient.ClientSession, id string, body jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error) {
	if ctxs == nil {
		p := jsonutils.NewDict()
		p.Add(jsonutils.JSONTrue, "admin")
		p.Add(jsonutils.JSONTrue, "system")
		p.Add(jsonutils.NewString(id), "tenant")
		ret, e := Servers.List(session, p)
		if e != nil {
			return nil, e
		} else {
			if ret.Total > 0 {
				err := &httputils.JSONClientError{}
				err.Code = 403
				err.Details = fmt.Sprintf("该项目（%s）下存在云服务器，请清除后重试", id)
				return nil, err
			}
		}
	}

	return this.deleteInContexts(session, id, nil, body, ctxs)
}

func (this *ProjectManagerV3) BatchDelete(session *mcclient.ClientSession, idlist []string, body jsonutils.JSONObject) []SubmitResult {
	return this.BatchDeleteInContexts(session, idlist, body, nil)
}

func (this *ProjectManagerV3) BatchDeleteInContexts(session *mcclient.ClientSession, idlist []string, body jsonutils.JSONObject, ctxs []ManagerContext) []SubmitResult {
	return BatchDo(idlist, func(id string) (jsonutils.JSONObject, error) {
		return this.DeleteInContexts(session, id, body, ctxs)
	})
}

func (this *ProjectManagerV3) AddTags(session *mcclient.ClientSession, id string, tags []string) error {
	path := fmt.Sprintf("/projects/%s/tags", id)
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewStringArray(tags), "tags")
	_, err := this._put(session, path, body, "")
	if err != nil {
		return err
	}
	return nil
}

func init() {
	Projects = ProjectManagerV3{NewIdentityV3Manager("project", "projects",
		[]string{},
		[]string{"ID", "Name", "Domain_Id", "Domain", "Parent_Id", "Enabled", "Description", "Created_At"})}

	register(&Projects)
}
