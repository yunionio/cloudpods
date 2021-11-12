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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type UserManager struct {
	modulebase.ResourceManager
}

func (this *UserManager) GetTenantRoles(session *mcclient.ClientSession, uid string, tenantId string) (*modulebase.ListResult, error) {
	url := fmt.Sprintf("/users/%s/roles", uid)
	if len(tenantId) > 0 {
		url = fmt.Sprintf("/tenants/%s/%s", tenantId, url)
	}
	return modulebase.List(this.ResourceManager, session, url, "roles")
}

func (this *UserManager) GetTenantRoleList(session *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	uid, e := params.GetString("id")
	if e != nil {
		return nil, e
	}
	tenantId, _ := params.GetString("tenantId")
	ret, e := this.GetTenantRoles(session, uid, tenantId)
	if e != nil {
		return nil, e
	}
	return modulebase.ListResult2JSON(ret), nil
}

type UserManagerV3 struct {
	modulebase.ResourceManager
}

func (this *UserManagerV3) GetProjects(session *mcclient.ClientSession, uid string) (*modulebase.ListResult, error) {
	url := fmt.Sprintf("/users/%s/projects?admin=true", uid)
	return modulebase.List(this.ResourceManager, session, url, "projects")
}

func (this *UserManagerV3) GetGroups(session *mcclient.ClientSession, uid string) (*modulebase.ListResult, error) {
	url := fmt.Sprintf("/users/%s/groups?admin=true", uid)
	return modulebase.List(this.ResourceManager, session, url, "groups")
}

func (this *UserManagerV3) GetProjectsRPC(s *mcclient.ClientSession, uid string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret, e := this.GetProjects(s, uid)
	if e != nil {
		return nil, e
	}
	return modulebase.ListResult2JSON(ret), nil
}

/*
func (this *UserManagerV3) GetIsLdapUser(s *mcclient.ClientSession, uid string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.JSONFalse, "isldap")
	log.Infof("GetIsLdapUser ret: %s", ret)
	user, err := this.Get(s, uid, nil)
	if err != nil {
		return ret, err
	}
	domain_id, err := user.GetString("domain_id")
	if err != nil {
		return ret, nil
	}
	domain, err := Domains.GetConfig(s, domain_id)
	if err != nil {
		log.Errorf("domain config error: %v", err)
		return ret, nil
	}
	driver, err := domain.GetString("identity", "driver")
	if err != nil {
		return ret, nil
	}
	if strings.ToLower(driver) == "ldap" {
		// ret["isldap"] = jsonutils.JSONTrue
		ret.Set("isldap", jsonutils.JSONTrue)
	}

	return ret, nil
}*/

func (this *UserManagerV3) _groupAction(s *mcclient.ClientSession, gid, uid, action string) error {
	if action == "join" {
		_, err := this.PutInContext(s, uid, nil, &Groups, gid)
		if err != nil {
			return err
		}
	} else if action == "leave" {
		_, err := this.DeleteInContext(s, uid, nil, &Groups, gid)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *UserManagerV3) DoJoinGroups(s *mcclient.ClientSession, uid string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// params format:
	// {
	//     "uid": "CCCGwOsrpp6h",
	//     "action": "leave" / "join", select one of them
	//     "gids": ["L6ssbAJUG3rC", "pu8lkunxP4z8"]
	// }

	gids, err := params.GetArray("gids")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("gids")
	}
	action, err := params.GetString("action")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("action")
	}

	if action != "join" && action != "leave" {
		return nil, httperrors.NewInputParameterError("unsupported action %s", action)
	}

	errs := make([]error, 0)
	for _, gid := range gids {
		_gid, _ := gid.GetString()
		if len(_gid) > 0 {
			err := this._groupAction(s, _gid, uid, action)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		if len(errs) == len(gids) {
			return nil, httperrors.NewGeneralError(errors.NewAggregate(errs))
		}
		log.Errorf("join group error %s", errors.NewAggregate(errs))
	}
	return jsonutils.NewDict(), nil
}

// create user && assgin user with project_domain、project、role
func (this *UserManagerV3) DoCreateUser(s *mcclient.ClientSession, p jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// params format:
	// {
	//     "project": ["projectA", "projectB"],
	//     "role": ["RoleA", "RoleB"],
	// }
	params := p.(*jsonutils.JSONDict)
	_project, _ := params.Get("project")
	_role, _ := params.Get("role")
	params.Remove("project")
	params.Remove("role")

	response, err := UsersV3.Create(s, params)
	if err != nil {
		return nil, err
	}

	uid, err := response.GetString("id")
	if err != nil {
		return nil, err
	}

	// assgin project & role
	projects := []string{}
	roles := []string{}
	if _project != nil {
		if _p, ok := _project.(*jsonutils.JSONArray); ok {
			projects = _p.GetStringArray()
		}
	}

	if _role != nil {
		if _r, ok := _role.(*jsonutils.JSONArray); ok {
			roles = _r.GetStringArray()
		}
	}

	errs := make([]error, 0)
	if len(projects) > 0 && len(roles) > 0 {
		for i := range projects {
			pid := projects[i]
			for j := range roles {
				rid := roles[j]
				err := Projects.JoinProject(s, rid, pid, uid)
				if err != nil {
					errs = append(errs, err)
				}
			}
		}
	}

	if len(errs) > 0 {
		log.Errorf("join project errors: %s", errors.NewAggregate(errs))
	}
	return response, nil
}

func (this *UserManagerV3) FetchId(s *mcclient.ClientSession, user string, domain string) (string, error) {
	userQuery := jsonutils.NewDict()
	if len(domain) > 0 {
		domainId, err := Domains.GetId(s, domain, nil)
		if err != nil {
			return "", err
		}
		userQuery.Add(jsonutils.NewString(domainId), "domain_id")
	}
	return this.GetId(s, user, userQuery)
}

var (
	Users   UserManager
	UsersV3 UserManagerV3
)

func init() {
	Users = UserManager{modules.NewIdentityManager("user", "users",
		[]string{},
		[]string{"ID", "Name", "TenantId", "Tenant_name",
			"Enabled", "Email", "Mobile"})}

	modules.Register(&Users)

	UsersV3 = UserManagerV3{modules.NewIdentityV3Manager("user", "users",
		[]string{},
		[]string{"ID", "Name", "Domain_Id", "Project_Domain", "readonly", "idp_id",
			"Enabled", "Email", "Mobile", "Displayname",
			"is_system_account", "allow_web_console", "enable_mfa"})}

	modules.Register(&UsersV3)
}
