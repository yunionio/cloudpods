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

func (manager *SQuotaBaseManager) queryQuota(ctx context.Context, quota IQuota, refresh bool) (*jsonutils.JSONDict, error) {
	ret := jsonutils.NewDict()

	keys := quota.GetKeys()

	usage := manager.newQuota()
	err := manager.usageStore.GetQuota(ctx, keys, usage)
	if err != nil {
		return nil, err
	}
	if refresh {
		usageChan := make(chan IQuota)
		manager.PostUsageJob(keys, usageChan, true)

		usage = <-usageChan
	}

	pendings, err := manager.GetPendingUsages(ctx, keys)
	if err != nil {
		return nil, err
	}

	ret.Update(jsonutils.Marshal(keys))
	ret.Update(quota.ToJSON(""))
	ret.Update(usage.ToJSON("usage"))
	if len(pendings) > 0 {
		pendingArray := jsonutils.NewArray()
		for _, q := range pendings {
			pendingArray.Add(q.ToJSON(""))
		}
		ret.Add(pendingArray, "pending")
	}

	return ret, nil
}

func (manager *SQuotaBaseManager) getQuotaHanlder(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, query, _ := appsrv.FetchEnv(ctx, w, r)
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)

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

	keys := OwnerIdQuotaKeys(scope, ownerId)
	refresh := jsonutils.QueryBoolean(query, "refresh", false)
	quotaList, err := manager.listQuotas(ctx, userCred, keys.DomainId, keys.ProjectId, refresh)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	manager.sendQuotaList(w, quotaList)
}

func (manager *SQuotaBaseManager) fetchSetQuotaScope(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (mcclient.IIdentityProvider, rbacutils.TRbacScope, rbacutils.TRbacScope, error) {
	var scope rbacutils.TRbacScope
	ownerId, err := db.FetchProjectInfo(ctx, data)
	if err != nil {
		return nil, scope, scope, err
	}
	var requestScope rbacutils.TRbacScope
	ownerScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionUpdate)
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
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)

	projectId := params["<tenantid>"]
	domainId := params["<domainid>"]
	data := jsonutils.NewDict()
	if len(projectId) > 0 {
		data.Add(jsonutils.NewString(projectId), "project")
	} else if len(domainId) > 0 {
		data.Add(jsonutils.NewString(domainId), "project_domain")
	}
	ownerId, scope, _, err := manager.fetchSetQuotaScope(ctx, userCred, data)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	if body == nil {
		httperrors.InvalidInputError(w, "fail to decode body")
		return
	}

	keys := SBaseQuotaKeys{}
	body.Unmarshal(&keys, manager.KeywordPlural())
	paramKeys := OwnerIdQuotaKeys(scope, ownerId)
	keys.DomainId = paramKeys.DomainId
	keys.ProjectId = paramKeys.ProjectId

	quota := manager.newQuota()
	err = body.Unmarshal(quota, manager.KeywordPlural())
	if err != nil {
		log.Errorf("Fail to decode JSON request body: %s", err)
		httperrors.InvalidInputError(w, "fail to decode body")
		return
	}
	quota.SetKeys(keys)

	oquota := manager.newQuota()
	err = manager.GetQuota(ctx, keys, oquota)
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
		oquota.FetchSystemQuota()
	case QUOTA_ACTION_REPLACE:
		oquota = quota
	default:
		oquota.Update(quota)
	}

	err = manager.SetQuota(ctx, userCred, oquota)
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
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
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

	refresh := jsonutils.QueryBoolean(query, "refresh", false)
	quotaList, err := manager.listQuotas(ctx, userCred, "", "", refresh)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	manager.sendQuotaList(w, quotaList)
}

func (manager *SQuotaBaseManager) sendQuotaList(w http.ResponseWriter, quotaList []jsonutils.JSONObject) {
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
	refresh := jsonutils.QueryBoolean(query, "refresh", false)
	quotaList, err := manager.listQuotas(ctx, userCred, domainId, "", refresh)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	manager.sendQuotaList(w, quotaList)
}

func (manager *SQuotaBaseManager) listQuotas(ctx context.Context, userCred mcclient.TokenCredential, targetDomainId string, targetProjectId string, refresh bool) ([]jsonutils.JSONObject, error) {
	q := manager.Query()
	if len(targetDomainId) > 0 {
		q = q.Equals("domain_id", targetDomainId)
		if len(targetProjectId) > 0 {
			q = q.Equals("tenant_id", targetProjectId)
		}else {
			q = q.IsNotEmpty("tenant_id")
		}
	} else {
		// domain only
		q = q.IsNotEmpty("domain_id")
		q = q.IsNullOrEmpty("tenant_id")
	}
	rows, err := q.Rows()
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return nil, httperrors.NewInternalServerError("query quotas %s", err)
		}else {
			return nil, nil
		}
	}
	defer rows.Close()

	quotaList := make([]IQuota, 0)
	keyList := make([]IQuotaKeys, 0)
	ret := make([]jsonutils.JSONObject, 0)
	for rows.Next() {
		quota := manager.newQuota()
		err := q.Row2Struct(rows, quota)
		if err != nil {
			return nil, errors.Wrap(err, "q.Row2Struct")
		}
		quotaList = append(quotaList, quota)
		keyList = append(keyList, quota.GetKeys())
	}
	idNameMap, err := manager.keyList2IdNameMap(ctx, keyList)
	var fields []string
	for i := range quotaList {
		keys := quotaList[i].GetKeys()
		keyJson := jsonutils.Marshal(keys).(*jsonutils.JSONDict)
		if idNameMap != nil {
			// no error, do check
			if len(fields) == 0 {
				fields = keys.Fields()
			}
			values := keys.Values()
			for i := range fields {
				if strings.HasSuffix(fields[i], "_id") && len(values[i]) > 0 {
				    if len(idNameMap[fields[i]][values[i]]) == 0 {
						manager.DeleteAllQuotas(ctx, userCred, keys)
						manager.pendingStore.DeleteAllQuotas(ctx, userCred, keys)
						manager.usageStore.DeleteAllQuotas(ctx, userCred, keys)
						continue
					}else {
						keyJson.Add(jsonutils.NewString(idNameMap[fields[i]][values[i]]), fields[i][:len(fields[i])-3])
					}
				}
			}
		}
		quotaJson, err := manager.queryQuota(ctx, quotaList[i], refresh)
		if err != nil {
			return nil, errors.Wrap(err, "manager.queryQuota")
		}
		quotaJson.Update(keyJson)
		ret = append(ret, quotaJson)
	}
	return ret, nil
}
