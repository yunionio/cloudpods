package modules

import (
	"fmt"
	"strings"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/mcclient"
)

type UserManager struct {
	ResourceManager
}

func (this *UserManager) GetTenantRoles(session *mcclient.ClientSession, uid string, tenantId string) (*ListResult, error) {
	url := fmt.Sprintf("/users/%s/roles", uid)
	if len(tenantId) > 0 {
		url = fmt.Sprintf("/tenants/%s/%s", tenantId, url)
	}
	return this._list(session, url, "roles")
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
	return ListResult2JSON(ret), nil
}

type UserManagerV3 struct {
	ResourceManager
}

func (this *UserManagerV3) GetProjects(session *mcclient.ClientSession, uid string) (*ListResult, error) {
	url := fmt.Sprintf("/users/%s/projects", uid)
	return this._list(session, url, "projects")
}

func (this *UserManagerV3) GetGroups(session *mcclient.ClientSession, uid string) (*ListResult, error) {
	url := fmt.Sprintf("/users/%s/groups", uid)
	return this._list(session, url, "groups")
}

func (this *UserManagerV3) GetProjectsRPC(s *mcclient.ClientSession, uid string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret, e := this.GetProjects(s, uid)
	if e != nil {
		return nil, e
	}
	return ListResult2JSON(ret), nil
}

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
}

func (this *UserManagerV3) _groupAction(s *mcclient.ClientSession, gid, uid, action string, ch chan int) error {

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

	defer func() {
		ch <- 1
	}()

	return nil
}

func (this *UserManagerV3) DoJoinGroups(s *mcclient.ClientSession, uid string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// params format:
	// {
	//     "uid": "CCCGwOsrpp6h",
	//     "action": "leave" / "join", select one of them
	//     "gids": ["L6ssbAJUG3rC", "pu8lkunxP4z8"]
	// }

	ret := jsonutils.NewDict()
	gids, e := params.GetArray("gids")
	if e != nil {
		return ret, e
	}
	action, e := params.GetString("action")
	if e != nil {
		return ret, e
	}

	if action != "join" && action != "leave" {
		return ret, nil
	}

	chs := make([]chan int, len(gids))

	for i, gid := range gids {
		_gid, e := gid.GetString()
		if e != nil {
			return ret, e
		}
		chs[i] = make(chan int)
		go this._groupAction(s, _gid, uid, action, chs[i])
	}

	for _, ch := range chs {
		<-ch
	}
	return ret, nil
}

var (
	Users   UserManager
	UsersV3 UserManagerV3
)

func init() {
	Users = UserManager{NewIdentityManager("user", "users",
		[]string{},
		[]string{"ID", "Name", "TenantId", "Tenant_name",
			"Enabled", "Email", "Mobile"})}

	register(&Users)

	UsersV3 = UserManagerV3{NewIdentityV3Manager("user", "users",
		[]string{},
		[]string{"ID", "Name", "Domain_Id",
			"Enabled", "Email", "Mobile"})}

	register(&UsersV3)
}
