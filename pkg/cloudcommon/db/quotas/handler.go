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

package quotas

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

var _manager *SQuotaManager

func AddQuotaHandler(manager *SQuotaManager, prefix string, app *appsrv.Application) {
	_manager = manager

	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s", prefix, _manager.Keyword()),
		auth.Authenticate(getQuotaHanlder), nil, "get_quota", nil)

	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<tenantid>", prefix, _manager.Keyword()),
		auth.Authenticate(getQuotaHanlder), nil, "get_quota_for_project", nil)

	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/domains/<domainid>", prefix, _manager.Keyword()),
		auth.Authenticate(getQuotaHanlder), nil, "get_quota_for_domain", nil)

	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/projects/<tenantid>", prefix, _manager.Keyword()),
		auth.Authenticate(getQuotaHanlder), nil, "get_quota_for_project", nil)

	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s", prefix, _manager.Keyword()),
		auth.Authenticate(setQuotaHanlder), nil, "set_quota", nil)

	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/<tenantid>", prefix, _manager.Keyword()),
		auth.Authenticate(setQuotaHanlder), nil, "set_quota_for_project", nil)

	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/domains/<domainid>", prefix, _manager.Keyword()),
		auth.Authenticate(setQuotaHanlder), nil, "set_quota_for_domain", nil)

	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/projects/<tenantid>", prefix, _manager.Keyword()),
		auth.Authenticate(setQuotaHanlder), nil, "set_quota_for_project", nil)

	/*app.AddHandler2("POST",
	fmt.Sprintf("%s/%s/<tenantid>/<action>", prefix, _manager.Keyword()),
	auth.Authenticate(checkQuotaHanlder), nil, "check_quota", nil)*/
}

func queryQuota(ctx context.Context, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) (*jsonutils.JSONDict, error) {
	ret := jsonutils.NewDict()

	quota := _manager.newQuota()
	err := _manager.GetQuota(ctx, scope, ownerId, quota)
	if err != nil {
		return nil, err
	}
	usage := _manager.newQuota()
	err = usage.FetchUsage(ctx, scope, ownerId)
	if err != nil {
		return nil, err
	}
	pending := _manager.newQuota()
	err = _manager.GetPendingUsage(ctx, scope, ownerId, pending)
	if err != nil {
		return nil, err
	}

	ret.Update(quota.ToJSON(""))
	ret.Update(usage.ToJSON("usage"))
	ret.Update(pending.ToJSON("pending"))

	return ret, nil
}

func getQuotaHanlder(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	params := appctx.AppContextParams(ctx)

	var ownerId mcclient.IIdentityProvider
	var scope rbacutils.TRbacScope
	var err error

	log.Debugf("%s", params)
	projectId := params["<tenantid>"]
	domainId := params["<domainid>"]
	if len(projectId) > 0 || len(domainId) > 0 {
		data := jsonutils.NewDict()
		if len(domainId) > 0 {
			data.Add(jsonutils.NewString(domainId), "domain")
		} else if len(projectId) > 0 {
			data.Add(jsonutils.NewString(projectId), "project")
		}
		ownerId, scope, err = db.FetchQueryOwnerScope(ctx, userCred, data, _manager, policy.PolicyActionGet)
		if err != nil {
			httperrors.GeneralServerError(w, err)
			return
		}
	} else {
		ownerId = userCred
		scope = rbacutils.ScopeProject
	}

	quota, err := queryQuota(ctx, scope, ownerId)

	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	body := jsonutils.NewDict()
	body.Add(quota, _manager.Keyword())

	appsrv.SendJSON(w, body)
}

func FetchSetQuotaScope(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (mcclient.IIdentityProvider, rbacutils.TRbacScope, error) {
	var scope rbacutils.TRbacScope
	ownerId, err := db.FetchProjectInfo(ctx, data)
	if err != nil {
		return nil, scope, err
	}
	ownerScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), _manager.Keyword(), policy.PolicyActionUpdate)
	if ownerId != nil {
		var requestScope rbacutils.TRbacScope
		if len(ownerId.GetProjectId()) > 0 {
			// project level
			scope = rbacutils.ScopeProject
			if ownerId.GetProjectDomainId() == userCred.GetProjectDomainId() {
				requestScope = rbacutils.ScopeDomain
			} else {
				requestScope = rbacutils.ScopeSystem
			}
		} else {
			// domain level if len(ownerId.GetProjectDomainId()) > 0 {
			scope = rbacutils.ScopeDomain
			requestScope = rbacutils.ScopeSystem
		}
		if requestScope.HigherThan(ownerScope) {
			return nil, scope, httperrors.NewForbiddenError("not enough privilleges")
		}
	} else {
		ownerId = userCred
		scope = rbacutils.ScopeProject
	}
	return ownerId, scope, nil
}

func setQuotaHanlder(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	params := appctx.AppContextParams(ctx)

	projectId := params["<tenantid>"]
	domainId := params["<domainid>"]
	data := jsonutils.NewDict()
	if len(projectId) > 0 {
		data.Add(jsonutils.NewString(projectId), "project")
	} else if len(domainId) > 0 {
		data.Add(jsonutils.NewString(domainId), "domain")
	}
	ownerId, scope, err := FetchSetQuotaScope(ctx, userCred, data)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	body, err := appsrv.FetchJSON(r)
	if err != nil {
		log.Errorf("Fail to decode JSON request body: %s", err)
		httperrors.InvalidInputError(w, "fail to decode body")
		return
	}
	quota := _manager.newQuota()
	err = body.Unmarshal(quota, _manager.Keyword())
	if err != nil {
		log.Errorf("Fail to decode JSON request body: %s", err)
		httperrors.InvalidInputError(w, "fail to decode body")
		return
	}
	oquota := _manager.newQuota()
	err = _manager.GetQuota(ctx, scope, ownerId, oquota)
	if err != nil {
		log.Errorf("get quota fail %s", err)
		httperrors.GeneralServerError(w, err)
		return
	}
	oquota.Update(quota)
	err = _manager.SetQuota(ctx, userCred, scope, ownerId, oquota)
	if err != nil {
		log.Errorf("set quota fail %s", err)
		httperrors.GeneralServerError(w, err)
		return
	}
	rbody := jsonutils.NewDict()
	rbody.Add(oquota.ToJSON(""), _manager.Keyword())
	appsrv.SendJSON(w, rbody)
}

/*func checkQuotaHanlder(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)

	isAllow := false
	if consts.IsRbacEnabled() {
		isAllow = policy.PolicyManager.Allow(rbacutils.ScopeSystem, userCred, consts.GetServiceType(),
			policy.PolicyDelegation, policy.PolicyActionGet) == rbacutils.Allow
	} else {
		isAllow = userCred.IsAllow(rbacutils.ScopeSystem, consts.GetServiceType(),
			policy.PolicyDelegation, policy.PolicyActionGet)
	}
	if !isAllow {
		httperrors.ForbiddenError(w, "not allow to delegate check quota")
		return
	}
	if consts.IsRbacEnabled() {
		if policy.PolicyManager.Allow(rbacutils.ScopeSystem, userCred, consts.GetServiceType(),
			_manager.Keyword(), policy.PolicyActionGet) != rbacutils.Allow {
			httperrors.ForbiddenError(w, "not allow to query quota")
			return
		}
	}

	params := appctx.AppContextParams(ctx)

	var ownerId mcclient.IIdentityProvider

	projectId := params["<tenantid>"]
	if len(projectId) == 0 {
		ownerId = userCred
	} else {
		tenant, err := db.TenantCacheManager.FetchTenantByIdOrName(ctx, projectId)
		if err != nil {
			if err == sql.ErrNoRows {
				httperrors.TenantNotFoundError(w, "project %s not found", projectId)
				return
			} else {
				httperrors.GeneralServerError(w, err)
				return
			}
		}
		ownerId = &db.SOwnerId{
			DomainId:  tenant.DomainId,
			Domain:    tenant.Domain,
			ProjectId: tenant.Id,
			Project:   tenant.Name,
		}
	}
	body, err := appsrv.FetchJSON(r)
	quota := _manager.newQuota()
	err = body.Unmarshal(quota, _manager.Keyword())
	if err != nil {
		log.Errorf("Fail to decode JSON request body: %s", err)
		httperrors.InvalidInputError(w, "fail to decode body")
		return
	}
	used, err := _manager.CheckQuota(ctx, userCred, rbacutils.ScopeProject, ownerId, quota)
	if err != nil {
		httperrors.OutOfQuotaError(w, "Out of quota: %s", err)
		return
	}
	rbody := jsonutils.NewDict()
	rbody.Add(used.ToJSON(""), _manager.Keyword())
	appsrv.SendJSON(w, rbody)
}*/
