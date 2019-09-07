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

package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"

	"yunion.io/x/onecloud/pkg/apigateway/clientman"
	"yunion.io/x/onecloud/pkg/apigateway/constants"
	"yunion.io/x/onecloud/pkg/apigateway/options"
	policytool "yunion.io/x/onecloud/pkg/apigateway/policy"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

func AppContextToken(ctx context.Context) mcclient.TokenCredential {
	val := ctx.Value(appctx.AppContextKey(constants.AUTH_TOKEN))
	if val == nil {
		return nil
	}
	return val.(mcclient.TokenCredential)
}

type AuthHandlers struct {
	*SHandlers
	preLoginHook PreLoginFunc
}

func NewAuthHandlers(prefix string, preLoginHook PreLoginFunc) *AuthHandlers {
	return &AuthHandlers{
		SHandlers:    NewHandlers(prefix),
		preLoginHook: preLoginHook,
	}
}

func (h *AuthHandlers) AddMethods() {
	// no middleware handler
	h.AddByMethod(GET, nil,
		NewHP(h.getRegions, "regions"),
		NewHP(h.listTotpRecoveryQuestions, "recovery"),
	)
	h.AddByMethod(POST, nil,
		NewHP(h.resetTotpSecrets, "credential"),
		NewHP(h.validatePasscode, "passcode"),
		NewHP(h.resetTotpRecoveryQuestions, "recovery"),
		NewHP(h.postLoginHandler, "login"),
	)

	// auth middleware handler
	h.AddByMethod(GET, FetchAuthToken,
		NewHP(h.getUser, "user"),
		NewHP(h.getPermissionDetails, "permissions"),
		NewHP(h.getAdminResources, "admin_resources"),
		NewHP(h.getResources, "scoped_resources"),
		NewHP(h.postLogoutHandler, "logout"),
	)
	h.AddByMethod(POST, FetchAuthToken,
		NewHP(h.resetUserPassword, "password"),
		NewHP(h.getPermissionDetails, "permissions"),
		NewHP(h.doCreatePolicies, "policies"),
	)
	h.AddByMethod(PATCH, FetchAuthToken,
		NewHP(h.doPatchPolicy, "policies", "<policy_id>"),
	)
	h.AddByMethod(DELETE, FetchAuthToken,
		NewHP(h.doDeletePolicies, "policies"),
	)
}

func (h *AuthHandlers) Bind(app *appsrv.Application) {
	h.AddMethods()
	h.SHandlers.Bind(app)
}

func (h *AuthHandlers) GetRegionsResponse(ctx context.Context, w http.ResponseWriter, req *http.Request) (*jsonutils.JSONDict, error) {
	adminToken := auth.AdminCredential()
	if adminToken == nil {
		return nil, errors.New("failed to get admin credential")
	}
	regions := adminToken.GetRegions()
	if len(regions) == 0 {
		return nil, errors.New("region is empty")
	}
	regionsJson := jsonutils.NewStringArray(regions)
	s := auth.GetAdminSession(ctx, regions[0], "")
	filters := jsonutils.NewDict()
	filters.Add(jsonutils.NewInt(1000), "limit")
	result, e := modules.Domains.List(s, filters)
	if e != nil {
		return nil, errors.Wrap(e, "list domain")
	}
	domains := jsonutils.NewArray()
	for _, d := range result.Data {
		dn, e := d.Get("name")
		if e == nil {
			if status, err := d.Bool("enabled"); err == nil && status {
				domains.Add(dn)
			}
		}
	}
	resp := jsonutils.NewDict()
	resp.Add(domains, "domains")
	resp.Add(regionsJson, "regions")

	filters = jsonutils.NewDict()
	filters.Add(jsonutils.NewStringArray([]string{"cas"}), "driver")
	filters.Add(jsonutils.NewInt(1000), "limit")
	idps, err := modules.IdentityProviders.List(s, filters)
	if err != nil {
		return nil, errors.Wrap(err, "list idp")
	}
	retIdps := make([]jsonutils.JSONObject, 0)
	for i := range idps.Data {
		retIdp := jsonutils.NewDict()
		id, _ := idps.Data[i].GetString("id")
		name, _ := idps.Data[i].GetString("name")
		driver, _ := idps.Data[i].GetString("driver")
		retIdp.Add(jsonutils.NewString(id), "id")
		retIdp.Add(jsonutils.NewString(name), "name")
		retIdp.Add(jsonutils.NewString(driver), "driver")
		conf, err := modules.IdentityProviders.GetSpecific(s, id, "config", nil)
		if err != nil {
			return nil, errors.Wrap(err, "idp get config spec")
		}
		retIdp.Update(conf)
		retIdps = append(retIdps, retIdp)
	}

	resp.Add(jsonutils.NewArray(retIdps...), "idps")

	return resp, nil
}

func (h *AuthHandlers) getRegions(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	resp, err := h.GetRegionsResponse(ctx, w, req)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, resp)
}

func (h *AuthHandlers) getUser(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	t := AppContextToken(ctx)
	s := auth.GetAdminSession(ctx, FetchRegion(req), "")
	data, err := getUserInfo(ctx, s, t, req)
	if err != nil {
		httperrors.NotFoundError(w, err.Error())
		return
	}
	body := jsonutils.NewDict()
	body.Add(data, "data")

	appsrv.SendJSON(w, body)
}

func (h *AuthHandlers) resetTotpSecrets(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	ResetTotpSecrets(ctx, w, req)
}

func (h *AuthHandlers) validatePasscode(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	ValidatePasscodeHandler(ctx, w, req)
}

func (h *AuthHandlers) resetTotpRecoveryQuestions(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	ResetTotpRecoveryQuestions(ctx, w, req)
}

func (h *AuthHandlers) listTotpRecoveryQuestions(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	ListTotpRecoveryQuestions(ctx, w, req)
}

// 返回 token及totp验证状态
func doTenantLogin(ctx context.Context, w http.ResponseWriter, req *http.Request, body jsonutils.JSONObject) (mcclient.TokenCredential, bool) {
	otpVerified := false
	tenantId, e := body.GetString("tenantId")
	if e != nil {
		httperrors.InvalidInputError(w, "not found tenantId in body")
		return nil, otpVerified
	}
	authTokenStr := getAuthToken(req)
	if len(authTokenStr) == 0 {
		httperrors.InvalidCredentialError(w, "not found auth token")
		return nil, otpVerified
	}
	token := clientman.TokenMan.Get(authTokenStr)
	if token == nil || !token.IsValid() {
		httperrors.InvalidCredentialError(w, "auth token %q is invalid", authTokenStr)
		return nil, otpVerified
	}

	if isUserEnableTotp(ctx, req, token) {
		totp := clientman.TokenMan.GetTotp(authTokenStr)
		if !totp.IsVerified() {
			httperrors.UnauthorizedError(w, "invalid totp token %q", authTokenStr)
			return nil, otpVerified
		} else {
			otpVerified = true
		}
	}

	token, e = auth.Client().SetProject(tenantId, "", "", token)
	if e != nil {
		httperrors.InvalidCredentialError(w, "failed to change project")
		return nil, otpVerified
	}
	return token, otpVerified
}

func isUserEnableTotp(ctx context.Context, req *http.Request, token mcclient.TokenCredential) bool {
	if !options.Options.EnableTotp {
		return false
	}
	s := auth.GetAdminSession(ctx, FetchRegion(req), "")
	usr, err := modules.UsersV3.Get(s, token.GetUserId(), nil)
	if err != nil {
		return false
	}
	return jsonutils.QueryBoolean(usr, "enable_mfa", true)
}

func (h *AuthHandlers) doCredentialLogin(ctx context.Context, req *http.Request, body jsonutils.JSONObject) (mcclient.TokenCredential, error) {
	var token mcclient.TokenCredential
	var err error
	var tenant string
	cliIp := netutils2.GetHttpRequestIp(req)
	if body.Contains("username") {
		uname, _ := body.GetString("username")

		if h.preLoginHook != nil {
			if err := h.preLoginHook(ctx, req, uname, body); err != nil {
				return nil, err
			}
		}

		passwd, err := body.GetString("password")
		if err != nil {
			return nil, httperrors.NewInputParameterError("get password in body")
		}
		if len(uname) == 0 || len(passwd) == 0 {
			return nil, httperrors.NewInputParameterError("username or password is empty")
		}

		tenant, uname = parseLoginUser(uname)
		// var token mcclient.TokenCredential
		domain, _ := body.GetString("domain")
		token, err = auth.Client().AuthenticateWeb(uname, passwd, domain, "", "", cliIp)
	} else if body.Contains("cas_ticket") {
		ticket, _ := body.GetString("cas_ticket")
		if len(ticket) == 0 {
			return nil, httperrors.NewInputParameterError("cas_ticket is empty")
		}
		token, err = auth.Client().AuthenticateCAS(ticket, "", "", "", cliIp)
	} else {
		return nil, httperrors.NewInputParameterError("missing credential")
	}
	if err != nil {
		switch httperr := err.(type) {
		case *httputils.JSONClientError:
			if httperr.Code == 409 {
				return nil, err
			}
		}
		return nil, httperrors.NewInvalidCredentialError("username/password incorrect")
	}
	uname := token.GetUserName()
	if len(tenant) > 0 {
		s := auth.GetAdminSession(ctx, FetchRegion(req), "")
		jsonProj, e := modules.Projects.GetById(s, tenant, nil)
		if e != nil {
			log.Errorf("fail to find preset project %s, reset to empty", tenant)
			tenant = ""
		} else {
			projId, _ := jsonProj.GetString("id")
			// projName, _ := jsonProj.GetString("name")
			ntoken, e := auth.Client().SetProject(projId, "", "", token)
			if e != nil {
				log.Errorf("fail to change to preset project %s(%s), reset to empty", tenant, e)
				tenant = ""
			} else {
				token = ntoken
			}
		}
	}
	if len(tenant) == 0 {
		s := auth.GetAdminSession(ctx, FetchRegion(req), "")
		projects, e := modules.UsersV3.GetProjects(s, token.GetUserId())
		if e == nil && len(projects.Data) > 0 {
			projectJson := projects.Data[0]
			for _, pJson := range projects.Data {
				pname, _ := pJson.GetString("name")
				if pname == uname {
					projectJson = pJson
					break
				}
			}
			pid, e := projectJson.GetString("id")
			if e == nil {
				ntoken, e := auth.Client().SetProject(pid, "", "", token)
				if e == nil {
					token = ntoken
				} else {
					log.Errorf("fail to change to default project %s(%s), reset to empty", pid, e)
				}
			}
		} else {
			log.Errorf("GetProjects for login user error %s project count %d", e, len(projects.Data))
		}
	}
	return token, nil
}

func parseLoginUser(uname string) (string, string) {
	slashpos := strings.IndexByte(uname, '/')
	tenant := ""
	if slashpos > 0 {
		tenant = uname[0:slashpos]
		uname = uname[slashpos+1:]
	}

	return tenant, uname
}

func isUserAllowWebconsole(ctx context.Context, w http.ResponseWriter, req *http.Request, token mcclient.TokenCredential) bool {
	s := auth.GetAdminSession(ctx, FetchRegion(req), "")
	usr, err := modules.UsersV3.Get(s, token.GetUserId(), nil)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return false
	}
	if !jsonutils.QueryBoolean(usr, "allow_web_console", true) {
		httperrors.ForbiddenError(w, "forbidden user %q login from web")
		return false
	}
	return true
}

func saveCookie(w http.ResponseWriter, name, val string, expire time.Time, base64 bool) {
	diff := time.Until(expire)
	maxAge := int(diff.Seconds())
	// log.Println("Set cookie", name, expire, maxAge, val)
	var valenc string
	if base64 {
		valenc = Base64UrlEncode([]byte(val))
	} else {
		valenc = val
	}
	// log.Printf("Set coookie: %s - %s\n", val, valenc)
	cookie := &http.Cookie{Name: name, Value: valenc, Path: "/", Expires: expire, MaxAge: maxAge, HttpOnly: false}
	http.SetCookie(w, cookie)
}

func getCookie(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		log.Errorf("Cookie not found %q", name)
		return ""
		// } else if cookie.Expires.Before(time.Now()) {
		//     fmt.Println("Cookie expired ", cookie.Expires, time.Now())
		//     return ""
	} else {
		val, err := Base64UrlDecode(cookie.Value)
		if err != nil {
			log.Errorf("Cookie %q fail to decode: %v", name, err)
			return ""
		}
		return string(val)
	}
}

func clearCookie(w http.ResponseWriter, name string) {
	cookie := &http.Cookie{Name: name, Expires: time.Now(), Path: "/", MaxAge: -1, HttpOnly: false}
	http.SetCookie(w, cookie)
}

type PreLoginFunc func(ctx context.Context, req *http.Request, uname string, body jsonutils.JSONObject) error

func (h *AuthHandlers) postLoginHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	body, e := appsrv.FetchJSON(req)
	if e != nil {
		httperrors.InvalidInputError(w, "fetch json for request: %v", e)
		return
	}
	var token mcclient.TokenCredential
	otpVerified := false
	if body.Contains("tenantId") { // switch project
		token, otpVerified = doTenantLogin(ctx, w, req, body)
	} else if body.Contains("username") || body.Contains("cas_ticket") {
		// user/password authenticate
		// cas authentication
		token, e = h.doCredentialLogin(ctx, req, body)
		if e != nil {
			httperrors.GeneralServerError(w, e)
			return
		}
	} else {
		httperrors.InvalidInputError(w, "no login credential")
		return
	}
	if token == nil {
		return
	}

	if !isUserAllowWebconsole(ctx, w, req, token) {
		return
	}

	//if len(token.GetProjectId()) == 0 {
	// no vaid project, return 403
	//	httperrors.NoProjectError(w, "no valid project")
	//	return
	//}

	tid := clientman.TokenMan.Save(token)

	// 切换项目时，如果之前totp已经验证通过，自动放行
	if otpVerified {
		totp := clientman.TokenMan.GetTotp(tid)
		totp.MarkVerified()
		clientman.TokenMan.SaveTotp(tid)
	}
	// log.Debugf("auth %s token %s", tid, token)
	s := auth.GetAdminSession(ctx, FetchRegion(req), "")
	authCookie, err := getUserAuthCookie(ctx, s, token, tid, req)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	saveCookie(w, constants.YUNION_AUTH_COOKIE, authCookie, token.GetExpires(), true)

	if len(token.GetProjectId()) > 0 {
		if body.Contains("isadmin") {
			adminVal := "false"
			if policy.PolicyManager.IsScopeCapable(token, rbacutils.ScopeSystem) {
				adminVal, _ = body.GetString("isadmin")
			}
			saveCookie(w, "isadmin", adminVal, token.GetExpires(), false)
		}
		if body.Contains("scope") {
			scopeStr, _ := body.GetString("scope")
			if !policy.PolicyManager.IsScopeCapable(token, rbacutils.TRbacScope(scopeStr)) {
				scopeStr = string(rbacutils.ScopeProject)
			}
			saveCookie(w, "scope", scopeStr, token.GetExpires(), false)
		}
		if body.Contains("domain") {
			domainStr, _ := body.GetString("domain")
			saveCookie(w, "domain", domainStr, token.GetExpires(), false)
		}
		saveCookie(w, "tenant", token.GetProjectId(), token.GetExpires(), false)
	}

	setAuthHeader(w, tid)

	// 开启Totp的状态下，如果用户未设置
	qrcode := ""
	if isUserEnableTotp(ctx, req, token) {
		qrcode, err = initializeUserTotpCred(s, token)
		if err != nil {
			httperrors.GeneralServerError(w, err)
			return
		}
	}

	appsrv.Send(w, qrcode)
}

func (h *AuthHandlers) postLogoutHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	tid := getAuthToken(req)
	if len(tid) > 0 {
		clientman.TokenMan.Remove(tid)
	}
	clearCookie(w, constants.YUNION_AUTH_COOKIE)
	appsrv.Send(w, "")
}

func FetchRegion(req *http.Request) string {
	r, e := req.Cookie("region")
	if e != nil {
		return options.Options.DefaultRegion
	}
	return r.Value
}

func fetchDomain(req *http.Request) string {
	r, e := req.Cookie("domain")
	if e != nil {
		return ""
	}
	return r.Value
}

type role struct {
	id   string
	name string
}

type projectRoles struct {
	id       string
	name     string
	domain   string
	domainId string
	roles    []role
}

func newProjectRoles(projectId, projectName, roleId, roleName string, domainId, domainName string) *projectRoles {
	return &projectRoles{
		id:       projectId,
		name:     projectName,
		domainId: domainId,
		domain:   domainName,
		roles:    []role{{id: roleId, name: roleName}},
	}
}

func (this *projectRoles) add(roleId, roleName string) {
	this.roles = append(this.roles, role{id: roleId, name: roleName})
}

func (this *projectRoles) getToken(scope rbacutils.TRbacScope, user, userId, domain, domainId string, ip string) mcclient.TokenCredential {
	return &mcclient.SSimpleToken{
		Domain:          domain,
		DomainId:        domainId,
		User:            user,
		UserId:          userId,
		Project:         this.name,
		ProjectId:       this.id,
		ProjectDomain:   this.domain,
		ProjectDomainId: this.domainId,
		Roles:           strings.Join(this.getRoles(), ","),
		Context: mcclient.SAuthContext{
			Ip: ip,
		},
	}
	// return policy.PolicyManager.IsScopeCapable(&t, scope)
}

func (this *projectRoles) getRoles() []string {
	roles := make([]string, 0)
	for _, r := range this.roles {
		roles = append(roles, r.name)
	}
	return roles
}

func (this *projectRoles) json(user, userId, domain, domainId string, ip string) jsonutils.JSONObject {
	obj := jsonutils.NewDict()
	obj.Add(jsonutils.NewString(this.id), "id")
	obj.Add(jsonutils.NewString(this.name), "name")
	obj.Add(jsonutils.NewString(this.domain), "domain")
	obj.Add(jsonutils.NewString(this.domainId), "domain_id")
	roles := jsonutils.NewArray()
	for _, r := range this.roles {
		role := jsonutils.NewDict()
		role.Add(jsonutils.NewString(r.id), "id")
		role.Add(jsonutils.NewString(r.name), "name")
		roles.Add(role)
	}
	obj.Add(roles, "roles")
	for _, scope := range []rbacutils.TRbacScope{
		rbacutils.ScopeSystem,
		rbacutils.ScopeDomain,
		rbacutils.ScopeProject,
	} {
		token := this.getToken(scope, user, userId, domain, domainId, ip)
		matches := policy.PolicyManager.MatchedPolicies(scope, token)
		obj.Add(jsonutils.NewStringArray(matches), fmt.Sprintf("%s_policies", scope))
		if len(matches) > 0 {
			obj.Add(jsonutils.JSONTrue, fmt.Sprintf("%s_capable", scope))
		} else {
			obj.Add(jsonutils.JSONFalse, fmt.Sprintf("%s_capable", scope))
		}
		// backward compatible
		if scope == rbacutils.ScopeSystem {
			if len(matches) > 0 {
				obj.Add(jsonutils.JSONTrue, "admin_capable")
			} else {
				obj.Add(jsonutils.JSONFalse, "admin_capable")
			}
		}
	}
	return obj
}

func getUserAuthCookie(ctx context.Context, s *mcclient.ClientSession, token mcclient.TokenCredential, sid string, req *http.Request) (string, error) {
	// info, err := getUserInfo(s, token, req)
	// if err != nil {
	// 	return "", err
	// }
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewTimeString(token.GetExpires()), "exp")
	info.Add(jsonutils.NewString(sid), "session")
	info.Add(jsonutils.NewBool(isUserEnableTotp(ctx, req, token)), "totp_on") // 用户totp 开启状态。 True（已开启）|False(未开启)
	info.Add(jsonutils.NewBool(options.Options.EnableTotp), "system_totp_on") // 全局totp 开启状态。 True（已开启）|False(未开启)
	return info.String(), nil
}

func getLBAgentInfo(s *mcclient.ClientSession, token mcclient.TokenCredential) (*jsonutils.JSONDict, error) {

	lbagents, err := modules.LoadbalancerAgents.List(s, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "user %s get lbagent", token.GetUserName())
	}

	item := jsonutils.NewDict()
	item.Add(jsonutils.NewString(""), "name")
	item.Add(jsonutils.NewString("lbagent"), "type")
	item.Add(jsonutils.JSONTrue, "as_menu")
	if len(lbagents.Data) > 0 {
		item.Add(jsonutils.JSONTrue, "status")
	} else {
		item.Add(jsonutils.JSONFalse, "status")
	}
	return item, nil
}

func getUserInfo(ctx context.Context, s *mcclient.ClientSession, token mcclient.TokenCredential, req *http.Request) (*jsonutils.JSONDict, error) {
	usr, err := modules.UsersV3.Get(s, token.GetUserId(), nil)
	if err != nil {
		log.Errorf("modules.UsersV3.Get fail %s", err)
		return nil, fmt.Errorf("not found user %s", token.GetUserId())
	}
	data := jsonutils.NewDict()
	for _, k := range []string{"displayname", "email", "id", "name", "enabled", "mobile", "allow_web_console", "created_at", "enable_mfa", "is_system_account", "last_active_at", "last_login_ip", "last_login_source", "idp_driver"} {
		v, e := usr.Get(k)
		if e == nil {
			data.Add(v, k)
		}
	}
	data.Add(jsonutils.NewString(token.GetDomainId()), "domain", "id")
	data.Add(jsonutils.NewString(token.GetDomainName()), "domain", "name")
	data.Add(jsonutils.NewStringArray(auth.AdminCredential().GetRegions()), "regions")
	data.Add(jsonutils.NewStringArray(token.GetRoles()), "roles")
	data.Add(jsonutils.NewString(token.GetProjectName()), "projectName")
	data.Add(jsonutils.NewString(token.GetProjectId()), "projectId")
	data.Add(jsonutils.NewString(token.GetProjectDomain()), "projectDomain")
	data.Add(jsonutils.NewString(token.GetProjectDomainId()), "projectDomainId")

	query := jsonutils.NewDict()
	query.Add(jsonutils.JSONNull, "effective")
	query.Add(jsonutils.JSONNull, "include_names")
	query.Add(jsonutils.NewInt(0), "limit")
	query.Add(jsonutils.NewString(token.GetUserId()), "user", "id")
	roleAssigns, err := modules.RoleAssignments.List(s, query)
	if err != nil {
		return nil, errors.Wrapf(err, "get RoleAssignments list")
	}
	projects := make(map[string]*projectRoles)
	for _, roleAssign := range roleAssigns.Data {
		roleId, _ := roleAssign.GetString("role", "id")
		roleName, _ := roleAssign.GetString("role", "name")
		projectId, _ := roleAssign.GetString("scope", "project", "id")
		projectName, _ := roleAssign.GetString("scope", "project", "name")
		domainId, _ := roleAssign.GetString("scope", "project", "domain", "id")
		domain, _ := roleAssign.GetString("scope", "project", "domain", "name")
		_, ok := projects[projectId]
		if ok {
			projects[projectId].add(roleId, roleName)
		} else {
			projects[projectId] = newProjectRoles(projectId, projectName, roleId, roleName, domainId, domain)
		}
	}
	projJson := jsonutils.NewArray()
	for _, proj := range projects {
		projJson.Add(proj.json(
			token.GetUserName(),
			token.GetUserId(),
			token.GetDomainName(),
			token.GetDomainId(),
			token.GetLoginIp(),
		))
	}
	data.Add(projJson, "projects")

	for _, scope := range []rbacutils.TRbacScope{
		rbacutils.ScopeSystem,
		rbacutils.ScopeDomain,
		rbacutils.ScopeProject,
	} {
		p := policy.PolicyManager.MatchedPolicies(scope, token)
		data.Add(jsonutils.NewStringArray(p), fmt.Sprintf("%s_policies", scope))
		if scope == rbacutils.ScopeSystem {
			data.Add(jsonutils.NewStringArray(p), "admin_policies")
		} else if scope == rbacutils.ScopeProject {
			data.Add(jsonutils.NewStringArray(p), "policies")
		}
	}
	allPolicies := policy.PolicyManager.AllPolicies()
	data.Add(jsonutils.Marshal(allPolicies), "all_policies")

	services := jsonutils.NewArray()
	menus := jsonutils.NewArray()
	k8s := jsonutils.NewArray()

	adminToken := auth.AdminCredential()
	curReg := FetchRegion(req)
	allsrv := adminToken.GetInternalServices(curReg)
	alleps := auth.Client().GetServiceCatalog().GetServicesByInterface(curReg, "console")

	if allsrv != nil && len(allsrv) > 0 {
		for _, srv := range allsrv {
			item := jsonutils.NewDict()
			item.Add(jsonutils.NewString(""), "name")
			item.Add(jsonutils.NewString(srv), "type")
			item.Add(jsonutils.JSONTrue, "status")
			if srv == "notify" {
				item.Add(jsonutils.JSONFalse, "as_menu")
			} else {
				item.Add(jsonutils.JSONTrue, "as_menu")
			}
			services.Add(item)
		}
	} else {
		log.Errorf("fail to find services????: %#v %s", adminToken, curReg)
	}

	lb, err := getLBAgentInfo(s, adminToken)
	if err != nil {
		log.Errorf("getLBAgentInfo fail %s", err)
	} else {
		services.Add(lb)
	}

	if alleps != nil {
		for _, ep := range alleps {
			item := jsonutils.NewDict()
			item.Add(jsonutils.NewString(ep.Url), "url")
			item.Add(jsonutils.NewString(ep.Name), "name")
			menus.Add(item)
		}
	}

	s2 := auth.GetSession(ctx, token, FetchRegion(req), "v2")
	cap, err := modules.Capabilities.List(s2, nil)
	if err != nil {
		log.Errorf("modules.Capabilities.List fail %s", err)
	} else {
		hypervisors, _ := cap.Data[0].Get("hypervisors")
		data.Add(hypervisors, "hypervisors")
	}

	data.Add(menus, "menus")
	data.Add(k8s, "k8sdashboard")
	data.Add(services, "services")

	if options.Options.NonDefaultDomainProjects {
		data.Add(jsonutils.JSONTrue, "non_default_domain_projects")
	} else {
		data.Add(jsonutils.JSONFalse, "non_default_domain_projects")
	}

	return data, nil
}

func getUserHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	t := AppContextToken(ctx)
	s := auth.GetAdminSession(ctx, FetchRegion(req), "")
	data, err := getUserInfo(ctx, s, t, req)
	if err != nil {
		httperrors.NotFoundError(w, err.Error())
		return
	}
	body := jsonutils.NewDict()
	body.Add(data, "data")

	appsrv.SendJSON(w, body)
}

func (h *AuthHandlers) getPermissionDetails(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	t := AppContextToken(ctx)

	_, query, body := appsrv.FetchEnv(ctx, w, req)
	if body == nil {
		httperrors.InvalidInputError(w, "body is empty")
		return
	}
	var name string
	if query != nil {
		name, _ = query.GetString("policy")
	}
	result, err := policy.PolicyManager.ExplainRpc(t, body, name)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	appsrv.SendJSON(w, result)
}

func (h *AuthHandlers) getAdminResources(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	res := policy.GetSystemResources()
	appsrv.SendJSON(w, jsonutils.Marshal(res))
}

func (h *AuthHandlers) getResources(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	res := policy.GetResources()
	appsrv.SendJSON(w, jsonutils.Marshal(res))
}

func (h *AuthHandlers) doCreatePolicies(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	t := AppContextToken(ctx)
	// if !utils.IsInStringArray("admin", t.GetRoles()) || t.GetProjectName() != "system" {
	// 	httperrors.ForbiddenError(w, "not allow to create policy")
	// 	return
	// }
	_, _, body := appsrv.FetchEnv(ctx, w, req)
	if body == nil {
		httperrors.InvalidInputError(w, "body is empty")
		return
	}
	s := auth.GetSession(ctx, t, FetchRegion(req), "")
	result, err := policytool.PolicyCreate(s, body)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, result)
}

func (h *AuthHandlers) doPatchPolicy(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	t := AppContextToken(ctx)
	// if !utils.IsInStringArray("admin", t.GetRoles()) || t.GetProjectName() != "system" {
	// 	httperrors.ForbiddenError(w, "not allow to create policy")
	// 	return
	// }
	params, _, body := appsrv.FetchEnv(ctx, w, req)
	if body == nil {
		httperrors.InvalidInputError(w, "request body is empty")
		return
	}
	s := auth.GetSession(ctx, t, FetchRegion(req), "")
	result, err := policytool.PolicyPatch(s, params["<policy_id>"], body)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, result)
}

func (h *AuthHandlers) doDeletePolicies(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	t := AppContextToken(ctx)
	// if !utils.IsInStringArray("admin", t.GetRoles()) || t.GetProjectName() != "system" {
	// 	httperrors.ForbiddenError(w, "not allow to create policy")
	// 	return
	// }
	_, query, _ := appsrv.FetchEnv(ctx, w, req)
	s := auth.GetSession(ctx, t, FetchRegion(req), "")

	idlist, e := query.GetArray("id")
	if e != nil || len(idlist) == 0 {
		httperrors.InvalidInputError(w, "missing id")
		return
	}
	idStrList := jsonutils.JSONArray2StringArray(idlist)
	ret := make([]modulebase.SubmitResult, len(idStrList))
	for i := range idStrList {
		err := policytool.PolicyDelete(s, idStrList[i])
		if err != nil {
			ret[i] = modulebase.SubmitResult{
				Status: 400,
				Id:     idStrList[i],
				Data:   jsonutils.NewString(err.Error()),
			}
		} else {
			ret[i] = modulebase.SubmitResult{
				Status: 200,
				Id:     idStrList[i],
				Data:   jsonutils.NewDict(),
			}
		}
	}
	w.WriteHeader(207)
	appsrv.SendJSON(w, modulebase.SubmitResults2JSON(ret))
}

/*
重置密码
1.验证新密码正确
2.验证原密码正确，且idp_driver为空
3.如果已开启MFA，验证 随机密码正确
4.重置密码，清除认证token
*/
func (h *AuthHandlers) resetUserPassword(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	ctx, err := SetAuthToken(ctx, w, req)
	if err != nil {
		httperrors.InvalidCredentialError(w, "set auth token %v", err)
		return
	}

	t := AppContextToken(ctx)
	s := auth.GetAdminSession(ctx, FetchRegion(req), "")
	_, _, body := appsrv.FetchEnv(ctx, w, req)
	if body == nil {
		httperrors.InvalidInputError(w, "body is empty")
		return
	}

	uid := t.GetUserId()
	if len(uid) == 0 {
		httperrors.ConflictError(w, "uid is empty")
		return
	}

	user, err := modules.UsersV3.Get(s, uid, nil)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	oldPwd, _ := body.GetString("password_old")
	newPwd, _ := body.GetString("password_new")
	confirmPwd, _ := body.GetString("password_confirm")
	passcode, _ := body.GetString("passcode")

	// 1.验证新密码正确
	if len(newPwd) < 6 {
		httperrors.InputParameterError(w, "new password must have at least 6 characters")
		return
	}

	if newPwd != confirmPwd {
		httperrors.InputParameterError(w, "new password mismatch")
		return
	}

	// 2.验证原密码正确，且idp_driver为空
	if isLdapUser(user) {
		httperrors.ForbiddenError(w, "not support reset ldap user password")
		return
	}

	cliIp := netutils2.GetHttpRequestIp(req)
	_, err = auth.Client().AuthenticateWeb(t.GetUserName(), oldPwd, t.GetDomainName(), "", "", cliIp)
	if err != nil {
		switch httperr := err.(type) {
		case *httputils.JSONClientError:
			if httperr.Code == 409 {
				httperrors.GeneralServerError(w, err)
				return
			}
		}
		httperrors.InputParameterError(w, "密码错误")
		return
	}

	// 3.如果已开启MFA，验证 随机密码正确
	tid := getAuthToken(req)
	if isMfaEnabled(user) {
		totp := clientman.TokenMan.GetTotp(tid)
		err = totp.VerifyTotpPasscode(s, uid, passcode)
		if err != nil {
			httperrors.InputParameterError(w, "invalid passcode")
			return
		}
	}

	// 4.重置密码，清除认证token
	params := jsonutils.NewDict()
	params.Set("password", jsonutils.NewString(newPwd))
	ret, err := modules.UsersV3.Patch(s, uid, params)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	} else {
		clientman.TokenMan.Remove(tid)
	}

	appsrv.SendJSON(w, ret)
}

func isLdapUser(user jsonutils.JSONObject) bool {
	if driver, _ := user.GetString("idp_driver"); driver == "ldap" {
		return true
	}

	return false
}

// refer: isUserEnableTotp
func isMfaEnabled(user jsonutils.JSONObject) bool {
	if !options.Options.EnableTotp {
		return false
	}

	if ok, _ := user.Bool("enable_mfa"); ok {
		return true
	}

	return false
}
