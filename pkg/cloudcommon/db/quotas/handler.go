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
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

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

const (
	QUOTA_ACTION_ADD     = "add"
	QUOTA_ACTION_RESET   = "reset"
	QUOTA_ACTION_REPLACE = "replace"
)

func AddQuotaHandler(manager *SQuotaBaseManager, prefix string, app *appsrv.Application) {
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s", prefix, manager.KeywordPlural()),
		auth.Authenticate(manager.getQuotaHanlder), nil, "get_quota", nil)

	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<tenantid>", prefix, manager.KeywordPlural()),
		auth.Authenticate(manager.getQuotaHanlder), nil, "get_quota_for_project", nil)

	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/domains", prefix, manager.KeywordPlural()),
		auth.Authenticate(manager.listDomainQuotaHanlder), nil, "list_quotas_for_all_domains", nil)

	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/domains/<domainid>", prefix, manager.KeywordPlural()),
		auth.Authenticate(manager.getQuotaHanlder), nil, "get_quota_for_domain", nil)

	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/projects", prefix, manager.KeywordPlural()),
		auth.Authenticate(manager.listProjectQuotaHanlder), nil, "list_quotas_for_all_projects", nil)

	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/projects/<tenantid>", prefix, manager.KeywordPlural()),
		auth.Authenticate(manager.getQuotaHanlder), nil, "get_quota_for_project", nil)

	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s", prefix, manager.KeywordPlural()),
		auth.Authenticate(manager.setQuotaHanlder), nil, "set_quota", nil)

	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/<tenantid>", prefix, manager.KeywordPlural()),
		auth.Authenticate(manager.setQuotaHanlder), nil, "set_quota_for_project", nil)

	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/domains/<domainid>", prefix, manager.KeywordPlural()),
		auth.Authenticate(manager.setQuotaHanlder), nil, "set_quota_for_domain", nil)

	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/projects/<tenantid>", prefix, manager.KeywordPlural()),
		auth.Authenticate(manager.setQuotaHanlder), nil, "set_quota_for_project", nil)

	/*app.AddHandler2("POST",
	fmt.Sprintf("%s/%s/<tenantid>/<action>", prefix, _manager.Keyword()),
	auth.Authenticate(checkQuotaHanlder), nil, "check_quota", nil)*/
}

func (manager *SQuotaBaseManager) queryQuota(ctx context.Context, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platforma []string, refresh bool) (*jsonutils.JSONDict, IQuota, error) {
	ret := jsonutils.NewDict()

	quota := manager.newQuota()
	err := manager.GetQuota(ctx, scope, ownerId, platforma, quota)
	if err != nil {
		return nil, nil, err
	}
	usage := manager.newQuota()
	err = manager.usageStore.GetQuota(ctx, scope, ownerId, platforma, usage)
	if err != nil {
		return nil, nil, err
	}
	if usage.IsEmpty() || refresh {
		usageChan := make(chan IQuota)
		manager.PostUsageJob(scope, ownerId, nil, usageChan, false, true)

		usage = <-usageChan
	}

	pending := manager.newQuota()
	err = manager.GetPendingUsage(ctx, scope, ownerId, nil, pending)
	if err != nil {
		return nil, nil, err
	}

	ret.Update(quota.ToJSON(""))
	ret.Update(usage.ToJSON("usage"))
	if !pending.IsEmpty() {
		ret.Update(pending.ToJSON("pending"))
	}

	if scope == rbacutils.ScopeDomain {
		total, err := manager.getDomainTotalQuota(ctx, ownerId.GetProjectDomainId(), nil)
		if err == nil {
			ret.Update(total.ToJSON("total"))
		}
	}

	return ret, usage, nil
}

func (manager *SQuotaBaseManager) getQuotaHanlder(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	params := appctx.AppContextParams(ctx)

	var ownerId mcclient.IIdentityProvider
	var scope rbacutils.TRbacScope
	var err error

	projectId := params["<tenantid>"]
	domainId := params["<domainid>"]
	if len(projectId) > 0 || len(domainId) > 0 {
		data := jsonutils.NewDict()
		if len(domainId) > 0 {
			data.Add(jsonutils.NewString(domainId), "project_domain")
		} else if len(projectId) > 0 {
			data.Add(jsonutils.NewString(projectId), "project")
		}
		ownerId, scope, err = db.FetchCheckQueryOwnerScope(ctx, userCred, data, manager, policy.PolicyActionGet, true)
		if err != nil {
			httperrors.GeneralServerError(w, err)
			return
		}
	} else {
		ownerId = userCred
		scope = rbacutils.ScopeProject
	}

	quota, _, err := manager.queryQuota(ctx, scope, ownerId, nil, true)

	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	body := jsonutils.NewDict()
	body.Add(quota, manager.KeywordPlural())

	appsrv.SendJSON(w, body)
}

func FetchSetQuotaScope(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (mcclient.IIdentityProvider, rbacutils.TRbacScope, rbacutils.TRbacScope, error) {
	var scope rbacutils.TRbacScope
	ownerId, err := db.FetchProjectInfo(ctx, data)
	if err != nil {
		return nil, scope, scope, err
	}
	var requestScope rbacutils.TRbacScope
	ownerScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), quotaKeywords, policy.PolicyActionUpdate)
	if ownerId != nil {
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
	} else {
		ownerId = userCred
		scope = rbacutils.ScopeProject
		requestScope = rbacutils.ScopeDomain
	}
	if requestScope.HigherThan(ownerScope) {
		return nil, scope, scope, httperrors.NewForbiddenError("not enough privilleges")
	}
	return ownerId, scope, ownerScope, nil
}

func (manager *SQuotaBaseManager) setQuotaHanlder(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	params := appctx.AppContextParams(ctx)

	projectId := params["<tenantid>"]
	domainId := params["<domainid>"]
	data := jsonutils.NewDict()
	if len(projectId) > 0 {
		data.Add(jsonutils.NewString(projectId), "project")
	} else if len(domainId) > 0 {
		data.Add(jsonutils.NewString(domainId), "project_domain")
	}
	ownerId, scope, allowScope, err := FetchSetQuotaScope(ctx, userCred, data)
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
	quota := manager.newQuota()
	err = body.Unmarshal(quota, manager.KeywordPlural())
	if err != nil {
		log.Errorf("Fail to decode JSON request body: %s", err)
		httperrors.InvalidInputError(w, "fail to decode body")
		return
	}
	oquota := manager.newQuota()
	err = manager.GetQuota(ctx, scope, ownerId, nil, oquota)
	if err != nil {
		log.Errorf("get quota fail %s", err)
		httperrors.GeneralServerError(w, err)
		return
	}
	action, _ := body.GetString(manager.KeywordPlural(), "action")
	switch action {
	case QUOTA_ACTION_ADD:
		oquota.Add(quota)
	case QUOTA_ACTION_RESET:
		oquota.FetchSystemQuota(scope, ownerId)
	case QUOTA_ACTION_REPLACE:
		oquota = quota
	default:
		oquota.Update(quota)
	}

	if consts.GetNonDefaultDomainProjects() {
		// only if non-default-domain-project is turned ON, check the conformance between project quotas and domain quotas
		if scope == rbacutils.ScopeProject {
			total, err := manager.getDomainTotalQuota(ctx, ownerId.GetProjectDomainId(), []string{ownerId.GetProjectId()})
			if err != nil {
				log.Errorf("get total quota fail %s", err)
				httperrors.GeneralServerError(w, err)
				return
			}

			domainQuota := manager.newQuota()
			err = manager.GetQuota(ctx, rbacutils.ScopeDomain, ownerId, nil, domainQuota)
			if err != nil {
				log.Errorf("GetQuota for domain %s fail %s", ownerId.GetProjectDomainId(), err)
				httperrors.GeneralServerError(w, err)
				return
			}

			total.Add(oquota)
			err = total.Exceed(quota, domainQuota)
			if err != nil {
				// exeed domain quota
				cascade, _ := body.Bool(manager.KeywordPlural(), "cascade")
				if !cascade {
					log.Errorf("project quota exeed domain quota: %s", err)
					httperrors.OutOfQuotaError(w, "project quota exeed domain quota")
					return
				} else {
					if allowScope != rbacutils.ScopeSystem {
						httperrors.OutOfQuotaError(w, "project quota exeed domain quota, no previlige to cascade set")
						return
					} else {
						// cascade set domain quota
						err = manager.SetQuota(ctx, userCred, rbacutils.ScopeDomain, ownerId, nil, total)
						if err != nil {
							log.Errorf("cascade set quota fail %s", err)
							httperrors.GeneralServerError(w, err)
							return
						}
					}
				}
			}
		} else {
			total, err := manager.getDomainTotalQuota(ctx, ownerId.GetProjectDomainId(), nil)
			if err != nil {
				log.Errorf("get total quota fail %s", err)
				httperrors.GeneralServerError(w, err)
				return
			}
			err = total.Exceed(quota, oquota)
			if err != nil {
				log.Errorf("project quota exeed domain quota: %s", err)
				httperrors.OutOfQuotaError(w, "project quota exeed domain quota")
				return
			}
		}
	}

	err = manager.SetQuota(ctx, userCred, scope, ownerId, nil, oquota)
	if err != nil {
		log.Errorf("set quota fail %s", err)
		httperrors.GeneralServerError(w, err)
		return
	}
	rbody := jsonutils.NewDict()
	rbody.Add(oquota.ToJSON(""), manager.KeywordPlural())
	appsrv.SendJSON(w, rbody)
}

func (manager *SQuotaBaseManager) listDomainQuotaHanlder(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)

	if consts.IsRbacEnabled() {
		allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList)
		if allowScope != rbacutils.ScopeSystem {
			httperrors.ForbiddenError(w, "not allow to list domain quotas")
			return
		}
	} else {
		if !userCred.HasSystemAdminPrivilege() {
			httperrors.ForbiddenError(w, "out of privileges")
			return
		}
	}

	quotaList, err := manager.listQuotas(ctx, "")
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	rbody := jsonutils.NewDict()
	rbody.Set(manager.KeywordPlural(), jsonutils.NewArray(quotaList...))
	appsrv.SendJSON(w, rbody)
}

func (manager *SQuotaBaseManager) listProjectQuotaHanlder(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	owner, err := db.FetchDomainInfo(ctx, query)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	if owner == nil {
		owner = userCred
	}

	if consts.IsRbacEnabled() {
		allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList)
		if (allowScope == rbacutils.ScopeDomain && userCred.GetProjectDomainId() == owner.GetProjectDomainId()) || allowScope == rbacutils.ScopeSystem {
		} else {
			httperrors.ForbiddenError(w, "not allow to list project quotas")
			return
		}
	} else {
		if !userCred.HasSystemAdminPrivilege() {
			httperrors.ForbiddenError(w, "out of privileges")
			return
		}
	}

	domainId := owner.GetProjectDomainId()
	quotaList, err := manager.listQuotas(ctx, domainId)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	rbody := jsonutils.NewDict()
	rbody.Set(manager.KeywordPlural(), jsonutils.NewArray(quotaList...))
	appsrv.SendJSON(w, rbody)
}

func (manager *SQuotaBaseManager) getDomainTotalQuota(ctx context.Context, targetDomainId string, excludes []string) (IQuota, error) {
	q := manager.Query("domain_id", "tenant_id", "platform")
	q = q.Equals("domain_id", targetDomainId)
	q = q.IsNotEmpty("tenant_id")
	if len(excludes) > 0 {
		q = q.NotIn("tenant_id", excludes)
	}
	// dsable platform
	q = q.IsNullOrEmpty("platform")
	rows, err := q.Rows()
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	defer rows.Close()

	ret := manager.newQuota()
	for rows.Next() {
		var domainId, projectId, platformStr string
		err := rows.Scan(&domainId, &projectId, &platformStr)
		if err != nil {
			return nil, errors.Wrap(err, "scan")
		}
		scope := rbacutils.ScopeProject
		owner := db.SOwnerId{
			DomainId:  domainId,
			ProjectId: projectId,
		}
		platform := strings.Split(platformStr, nameSeparator)

		quota := manager.newQuota()
		err = manager.GetQuota(ctx, scope, &owner, platform, quota)
		if err != nil {
			return nil, errors.Wrap(err, "GetQuota")
		}
		ret.Add(quota)
	}
	return ret, nil
}

func (manager *SQuotaBaseManager) listQuotas(ctx context.Context, targetDomainId string) ([]jsonutils.JSONObject, error) {
	q := manager.Query("domain_id", "tenant_id", "platform")
	if len(targetDomainId) > 0 {
		q = q.Equals("domain_id", targetDomainId)
	} else {
		// domain only
		q = q.IsNullOrEmpty("tenant_id")
	}
	// if tuen OFF NonDefaultDOmainProjects
	// no domain quota should return
	if !consts.GetNonDefaultDomainProjects() {
		q = q.IsNotEmpty("tenant_id")
	}
	// dsable platform
	q = q.IsNullOrEmpty("platform")
	rows, err := q.Rows()
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("query quotas fail %s", err)
		return nil, httperrors.NewInternalServerError("query quotas %s", err)
	}
	defer rows.Close()

	ret := make([]jsonutils.JSONObject, 0)
	for rows.Next() {
		var domainId, projectId, platformStr string
		err := rows.Scan(&domainId, &projectId, &platformStr)
		if err != nil {
			log.Errorf("scan domain_id, project_id, platform error %s", err)
			return nil, httperrors.NewInternalServerError("scan quotas %s", err)
		}
		scope := rbacutils.ScopeProject
		owner := db.SOwnerId{
			DomainId:  domainId,
			ProjectId: projectId,
		}
		if len(projectId) == 0 {
			scope = rbacutils.ScopeDomain
		}
		platform := strings.Split(platformStr, nameSeparator)
		quota, _, err := manager.queryQuota(ctx, scope, &owner, platform, false)
		if err != nil {
			log.Errorf("query quota for %s fail %s", getMemoryStoreKey(scope, &owner, platform), err)
			continue
		}
		if len(projectId) > 0 {
			quota.Set("tenant_id", jsonutils.NewString(projectId))
			quota.Set("domain_id", jsonutils.NewString(domainId))
			// fetch without cache expiration check
			project, err := db.TenantCacheManager.FetchTenantByIdWithoutExpireCheck(ctx, projectId)
			if err != nil {
				return nil, err
			}
			quota.Set("tenant", jsonutils.NewString(project.Name))
			quota.Set("project_domain", jsonutils.NewString(project.Domain))
		} else {
			quota.Set("domain_id", jsonutils.NewString(domainId))
			// fetch without cache expiration check
			domain, err := db.TenantCacheManager.FetchDomainByIdWithoutExpireCheck(ctx, domainId)
			if err != nil {
				return nil, err
			}
			quota.Set("project_domain", jsonutils.NewString(domain.Name))
		}
		if len(platformStr) > 0 {
			quota.Set("platform", jsonutils.NewString(platformStr))
		}
		ret = append(ret, quota)
	}
	// if no projects for a domain and NonDefaultDomainProjects is turned ON
	if len(ret) == 0 && len(targetDomainId) > 0 && consts.GetNonDefaultDomainProjects() {
		// return the initial quota of targetDomainId
		scope := rbacutils.ScopeDomain
		owner := db.SOwnerId{
			DomainId: targetDomainId,
		}
		platform := []string{}
		quota, _, err := manager.queryQuota(ctx, scope, &owner, platform, false)
		if err != nil {
			return nil, httperrors.NewInternalServerError("query domain initial quotas %s", err)
		}
		quota.Set("domain_id", jsonutils.NewString(targetDomainId))
		// fetch without cache expiration check
		domain, err := db.TenantCacheManager.FetchDomainByIdWithoutExpireCheck(ctx, targetDomainId)
		if err != nil {
			return nil, err
		}
		quota.Set("project_domain", jsonutils.NewString(domain.Name))
		ret = append(ret, quota)
	}
	return ret, nil
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
