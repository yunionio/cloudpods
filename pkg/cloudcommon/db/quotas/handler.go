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
	"reflect"
	"sort"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/reflectutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

const (
	QUOTA_ACTION_ADD     = "add"
	QUOTA_ACTION_SUB     = "sub"
	QUOTA_ACTION_RESET   = "reset"
	QUOTA_ACTION_REPLACE = "replace"
	QUOTA_ACTION_UPDATE  = "update"
	QUOTA_ACTION_DELETE  = "delete"
)

type SBaseQuotaSetInput struct {
	// 设置配额操作
	//
	// | action  | 说明                                      |
	// |---------|-------------------------------------------|
	// | add     | 增量增加配额                              |
	// | sub     | 增量减少配额                              |
	// | reset   | 重置所有配额为0                           |
	// | replace | 替换所有配额，对于不存在的配额项，设置为0 |
	// | update  | 更新存在的配额                            |
	// | delete  | 删除配额                                  |
	//
	Action string `json:"action"`
}

type SBaseQuotaQueryInput struct {
	// 只列出主配额
	// require:false
	Primary bool `json:"primary"`
	// 强制刷新使用量
	// require:false
	Refresh bool `json:"refresh"`
}

func AddQuotaHandler(manager *SQuotaBaseManager, prefix string, app *appsrv.Application) {
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s", prefix, manager.KeywordPlural()),
		auth.Authenticate(manager.getQuotaHandler), nil, "get_quota", nil)

	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/domains", prefix, manager.KeywordPlural()),
		auth.Authenticate(manager.listDomainQuotaHandler), nil, "list_quotas_for_all_domains", nil)

	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/domains/<domainid>", prefix, manager.KeywordPlural()),
		auth.Authenticate(manager.getQuotaHandler), nil, "get_quota_for_domain", nil)

	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s", prefix, manager.KeywordPlural()),
		auth.Authenticate(manager.setQuotaHandler), nil, "set_quota", nil)

	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/domains/<domainid>", prefix, manager.KeywordPlural()),
		auth.Authenticate(manager.setQuotaHandler), nil, "set_quota_for_domain", nil)

	app.AddHandler2("DELETE",
		fmt.Sprintf("%s/%s/pending", prefix, manager.KeywordPlural()),
		auth.Authenticate(manager.cleanPendingUsageHandler), nil, "clean_pending_usage", nil)

	app.AddHandler2("DELETE",
		fmt.Sprintf("%s/%s/domains/<domainid>/pending", prefix, manager.KeywordPlural()),
		auth.Authenticate(manager.cleanPendingUsageHandler), nil, "clean_pending_usage_for_domain", nil)

	if manager.scope == rbacscope.ScopeProject {
		app.AddHandler2("GET",
			fmt.Sprintf("%s/%s/<tenantid>", prefix, manager.KeywordPlural()),
			auth.Authenticate(manager.getQuotaHandler), nil, "get_quota_for_project", nil)

		app.AddHandler2("GET",
			fmt.Sprintf("%s/%s/projects", prefix, manager.KeywordPlural()),
			auth.Authenticate(manager.listProjectQuotaHandler), nil, "list_quotas_for_all_projects", nil)

		app.AddHandler2("GET",
			fmt.Sprintf("%s/%s/projects/<tenantid>", prefix, manager.KeywordPlural()),
			auth.Authenticate(manager.getQuotaHandler), nil, "get_quota_for_project", nil)

		app.AddHandler2("POST",
			fmt.Sprintf("%s/%s/<tenantid>", prefix, manager.KeywordPlural()),
			auth.Authenticate(manager.setQuotaHandler), nil, "set_quota_for_project", nil)

		app.AddHandler2("POST",
			fmt.Sprintf("%s/%s/projects/<tenantid>", prefix, manager.KeywordPlural()),
			auth.Authenticate(manager.setQuotaHandler), nil, "set_quota_for_project", nil)

		app.AddHandler2("DELETE",
			fmt.Sprintf("%s/%s/projects/<tenantid>/pending", prefix, manager.KeywordPlural()),
			auth.Authenticate(manager.cleanPendingUsageHandler), nil, "clean_pending_usage_for_project", nil)
	}
}

func (manager *SQuotaBaseManager) queryQuota(ctx context.Context, quota IQuota, refresh bool) (*jsonutils.JSONDict, error) {
	ret := jsonutils.NewDict()

	keys := quota.GetKeys()
	ret.Update(jsonutils.Marshal(keys))
	ret.Update(quota.ToJSON(""))

	if !consts.EnableQuotaCheck() {
		return ret, nil
	}

	usage := manager.newQuota()
	err := manager.usageStore.GetQuota(ctx, keys, usage)
	if err != nil {
		return nil, errors.Wrap(err, "manager.usageStore.GetQuota")
	}
	if refresh {
		usageChan := make(chan IQuota)
		manager.PostUsageJob(keys, usageChan, true)

		usage = <-usageChan
	}

	pendings, err := manager.GetPendingUsages(ctx, keys)
	if err != nil {
		return nil, errors.Wrap(err, "manager.GetPendingUsages")
	}

	if usage != nil {
		ret.Update(usage.ToJSON("usage"))
	}
	if len(pendings) > 0 {
		pendingArray := jsonutils.NewArray()
		for _, q := range pendings {
			pending := q.ToJSON("")
			pending.(*jsonutils.JSONDict).Update(jsonutils.Marshal(q.GetKeys()))
			pendingArray.Add(pending)
		}
		ret.Add(pendingArray, "pending")
	}

	return ret, nil
}

func (manager *SQuotaBaseManager) getQuotaHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, query, _ := appsrv.FetchEnv(ctx, w, r)
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)

	var ownerId mcclient.IIdentityProvider
	var scope rbacscope.TRbacScope
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
		ownerId, scope, err, _ = db.FetchCheckQueryOwnerScope(ctx, userCred, data, manager.GetIQuotaManager(), policy.PolicyActionGet, true)
		if err != nil {
			httperrors.GeneralServerError(ctx, w, err)
			return
		}
	} else {
		scopeStr, _ := query.GetString("scope")
		if scopeStr == "project" && manager.scope == rbacscope.ScopeProject {
			scope = rbacscope.ScopeProject
		} else if scopeStr == "domain" {
			scope = rbacscope.ScopeDomain
		} else {
			scope = manager.scope
		}
		ownerId = userCred
	}

	keys := OwnerIdProjectQuotaKeys(scope, ownerId)
	refresh := jsonutils.QueryBoolean(query, "refresh", false)
	primary := jsonutils.QueryBoolean(query, "primary", false)
	quotaList, err := manager.listQuotas(ctx, userCred, keys.DomainId, keys.ProjectId, scope == rbacscope.ScopeDomain, primary, refresh)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	if len(quotaList) == 0 {
		quota := manager.newQuota()
		var baseKeys IQuotaKeys
		if manager.scope == rbacscope.ScopeProject {
			baseKeys = OwnerIdProjectQuotaKeys(scope, ownerId)
		} else {
			baseKeys = OwnerIdDomainQuotaKeys(ownerId)
		}
		reflectutils.FillEmbededStructValue(reflect.Indirect(reflect.ValueOf(quota)), reflect.ValueOf(baseKeys))
		quota.FetchSystemQuota()
		manager.SetQuota(ctx, userCred, quota)

		quotaList, err = manager.listQuotas(ctx, userCred, keys.DomainId, keys.ProjectId, scope == rbacscope.ScopeDomain, primary, refresh)
		if err != nil {
			httperrors.GeneralServerError(ctx, w, err)
			return
		}
	}
	manager.sendQuotaList(w, quotaList)
}

func (manager *SQuotaBaseManager) fetchSetQuotaScope(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	data jsonutils.JSONObject,
	isBaseQuotaKeys bool,
) (
	mcclient.IIdentityProvider,
	rbacscope.TRbacScope,
	rbacscope.TRbacScope,
	error,
) {
	var scope rbacscope.TRbacScope
	ownerId, err := db.FetchProjectInfo(ctx, data)
	if err != nil {
		return nil, scope, scope, err
	}
	var requestScope rbacscope.TRbacScope
	if ownerId != nil {
		if len(ownerId.GetProjectId()) > 0 {
			// project level
			scope = rbacscope.ScopeProject
			if ownerId.GetProjectDomainId() == userCred.GetProjectDomainId() {
				if isBaseQuotaKeys || ownerId.GetProjectId() != userCred.GetProjectId() {
					requestScope = rbacscope.ScopeDomain
				} else {
					requestScope = rbacscope.ScopeProject
				}
			} else {
				requestScope = rbacscope.ScopeSystem
			}
		} else {
			// domain level if len(ownerId.GetProjectDomainId()) > 0 {
			scope = rbacscope.ScopeDomain
			if isBaseQuotaKeys || ownerId.GetProjectDomainId() != userCred.GetProjectDomainId() {
				requestScope = rbacscope.ScopeSystem
			} else {
				requestScope = rbacscope.ScopeDomain
			}
		}
	} else {
		ownerId = userCred
		scope = rbacscope.ScopeProject
		if isBaseQuotaKeys {
			requestScope = rbacscope.ScopeDomain
		} else {
			requestScope = rbacscope.ScopeProject
		}
	}

	return ownerId, scope, requestScope, nil
}

func (manager *SQuotaBaseManager) cleanPendingUsageHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, query, _ := appsrv.FetchEnv(ctx, w, r)
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)

	var ownerId mcclient.IIdentityProvider
	var scope rbacscope.TRbacScope
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
		ownerId, scope, err, _ = db.FetchCheckQueryOwnerScope(ctx, userCred, data, manager, policy.PolicyActionGet, true)
		if err != nil {
			httperrors.GeneralServerError(ctx, w, err)
			return
		}
	} else {
		scopeStr, _ := query.GetString("scope")
		if scopeStr == "project" {
			scope = rbacscope.ScopeProject
		} else if scopeStr == "domain" {
			scope = rbacscope.ScopeDomain
		} else {
			scope = manager.scope
		}
		ownerId = userCred
	}
	var keys IQuotaKeys
	if manager.scope == rbacscope.ScopeProject {
		keys = OwnerIdProjectQuotaKeys(scope, ownerId)
	} else {
		keys = OwnerIdDomainQuotaKeys(ownerId)
	}
	err = manager.cleanPendingUsage(ctx, userCred, keys)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	rbody := jsonutils.NewDict()
	appsrv.SendJSON(w, rbody)
}

func (manager *SQuotaBaseManager) setQuotaHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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

	quota := manager.newQuota()
	err := body.Unmarshal(quota, manager.KeywordPlural())
	if err != nil {
		log.Errorf("Fail to decode JSON request body: %s", err)
		httperrors.InvalidInputError(appctx.WithRequestLang(ctx, r), w, "fail to decode body")
		return
	}

	// check is there any nonempty key other than domain_id and project_id
	isBaseQuota := false
	if manager.scope == rbacscope.ScopeDomain {
		isBaseQuota = IsBaseDomainQuotaKeys(quota.GetKeys())
	} else {
		isBaseQuota = IsBaseProjectQuotaKeys(quota.GetKeys())
	}
	ownerId, scope, requestScope, err := manager.fetchSetQuotaScope(ctx, userCred, data, isBaseQuota)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	// fill project_id and domain_id
	var baseKeys IQuotaKeys
	if manager.scope == rbacscope.ScopeDomain {
		baseKeys = OwnerIdDomainQuotaKeys(ownerId)
	} else {
		baseKeys = OwnerIdProjectQuotaKeys(scope, ownerId)
	}
	reflectutils.FillEmbededStructValue(reflect.Indirect(reflect.ValueOf(quota)), reflect.ValueOf(baseKeys))

	keys := quota.GetKeys()
	isNew := false
	oquota := manager.newQuota()
	oquota.SetKeys(keys)
	err = manager.getQuotaByKeys(ctx, keys, oquota)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("get quota %s fail %s", QuotaKeyString(keys), err)
			httperrors.GeneralServerError(ctx, w, err)
			return
		} else {
			isNew = true
		}
	}

	action, _ := body.GetString(manager.KeywordPlural(), "action")
	var policyAction string
	if action == QUOTA_ACTION_DELETE {
		if isNew {
			// no need to delete
			httperrors.NotFoundError(ctx, w, "Quota %s not found", QuotaKeyString(keys))
			return
		} else if isBaseQuota {
			// base quota is not deletable
			httperrors.ForbiddenError(ctx, w, "Default quota %s not allow to delete", QuotaKeyString(keys))
			return
		}
		policyAction = policy.PolicyActionDelete
	} else {
		if isNew {
			policyAction = policy.PolicyActionCreate
		} else {
			policyAction = policy.PolicyActionUpdate
		}
	}

	log.Debugf("is_new: %v action: %s origin: %s current: %s", isNew, action, jsonutils.Marshal(oquota), jsonutils.Marshal(quota))

	// check rbac policy
	ownerScope, policyResult := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policyAction)
	if policyResult.Result.IsAllow() && requestScope.HigherThan(ownerScope) {
		httperrors.ForbiddenError(ctx, w, "not enough privilleges")
		return
	}

	if action == QUOTA_ACTION_DELETE {
		err = manager.DeleteQuota(ctx, userCred, keys)
		if err != nil {
			httperrors.GeneralServerError(ctx, w, err)
			return
		}
	} else {
		switch action {
		case QUOTA_ACTION_ADD:
			oquota.Add(quota)
		case QUOTA_ACTION_SUB:
			oquota.Sub(quota)
		case QUOTA_ACTION_RESET:
			oquota.FetchSystemQuota()
		case QUOTA_ACTION_REPLACE:
			oquota = quota
		case QUOTA_ACTION_UPDATE:
			fallthrough
		default:
			oquota.Update(quota)
		}

		log.Debugf("To set %s", jsonutils.Marshal(oquota))

		err = manager.SetQuota(ctx, userCred, oquota)
		if err != nil {
			log.Errorf("set quota fail %s", err)
			httperrors.GeneralServerError(ctx, w, err)
			return
		}
	}

	quotaList, err := manager.listQuotas(ctx, userCred, baseKeys.OwnerId().GetProjectDomainId(), baseKeys.OwnerId().GetProjectId(), scope == rbacscope.ScopeDomain, false, true)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	manager.sendQuotaList(w, quotaList)
}

func (manager *SQuotaBaseManager) listDomainQuotaHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)

	allowScope, policyResult := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList)
	if policyResult.Result.IsAllow() && allowScope != rbacscope.ScopeSystem {
		httperrors.ForbiddenError(ctx, w, "not allow to list domain quotas")
		return
	}

	refresh := jsonutils.QueryBoolean(query, "refresh", false)
	primary := jsonutils.QueryBoolean(query, "primary", false)
	quotaList, err := manager.listQuotas(ctx, userCred, "", "", false, primary, refresh)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	manager.sendQuotaList(w, sortQuotaByUsage(quotaList))
}

func (manager *SQuotaBaseManager) sendQuotaList(w http.ResponseWriter, quotaList []jsonutils.JSONObject) {
	rbody := jsonutils.NewDict()
	rbody.Set(manager.KeywordPlural(), jsonutils.NewArray(quotaList...))
	appsrv.SendJSON(w, rbody)
}

func (manager *SQuotaBaseManager) listProjectQuotaHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	owner, err := db.FetchDomainInfo(ctx, query)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	if owner == nil {
		owner = userCred
	}

	allowScope, _ := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList)
	if (allowScope == rbacscope.ScopeDomain && userCred.GetProjectDomainId() == owner.GetProjectDomainId()) || allowScope == rbacscope.ScopeSystem {
	} else {
		httperrors.ForbiddenError(ctx, w, "not allow to list project quotas")
		return
	}

	domainId := owner.GetProjectDomainId()
	refresh := jsonutils.QueryBoolean(query, "refresh", false)
	primary := jsonutils.QueryBoolean(query, "primary", false)
	quotaList, err := manager.listQuotas(ctx, userCred, domainId, "", false, primary, refresh)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	manager.sendQuotaList(w, sortQuotaByUsage(quotaList))
}

func (manager *SQuotaBaseManager) listQuotas(ctx context.Context, userCred mcclient.TokenCredential, targetDomainId string, targetProjectId string, domainOnly bool, primaryOnly bool, refresh bool) ([]jsonutils.JSONObject, error) {
	q := manager.Query()
	if len(targetDomainId) > 0 {
		q = q.Equals("domain_id", targetDomainId)
		if domainOnly {
			if manager.scope == rbacscope.ScopeProject {
				q = q.IsEmpty("tenant_id")
			}
		} else {
			if len(targetProjectId) > 0 {
				q = q.Equals("tenant_id", targetProjectId)
			}
			// otherwise, list all projects quotas for a domain
			// also include the domain's quota
			// else {
			// 	q = q.IsNotEmpty("tenant_id")
			// }
		}
	} else {
		// domain only
		q = q.IsNotEmpty("domain_id")
		if manager.scope == rbacscope.ScopeProject {
			q = q.IsNullOrEmpty("tenant_id")
		}
	}
	if primaryOnly {
		// list primary quota for domain or project
		fields := manager.getQuotaFields()
		for _, f := range fields {
			if f != "domain_id" && f != "tenant_id" {
				q = q.IsNullOrEmpty(f)
			}
		}
	}
	rows, err := q.Rows()
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return nil, httperrors.NewInternalServerError("query quotas %s", err)
		} else {
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
					} else {
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

type tQuotaResultList []jsonutils.JSONObject

func (a tQuotaResultList) Len() int           { return len(a) }
func (a tQuotaResultList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a tQuotaResultList) Less(i, j int) bool { return usageRateOfQuota(a[i]) > usageRateOfQuota(a[j]) }

func usageRateOfQuota(quota jsonutils.JSONObject) float32 {
	maxRate := float32(0)
	quotaMap, _ := quota.GetMap()
	for k, v := range quotaMap {
		usageK := fmt.Sprintf("usage.%s", k)
		if usageV, ok := quotaMap[usageK]; ok {
			intV, _ := v.Int()
			if intV > 0 {
				intUsageV, _ := usageV.Int()
				rate := float32(intUsageV) / float32(intV)
				if maxRate < rate {
					maxRate = rate
				}
			}
		}
	}
	return maxRate
}

func sortQuotaByUsage(quotaList []jsonutils.JSONObject) []jsonutils.JSONObject {
	sort.Sort(tQuotaResultList(quotaList))
	return quotaList
}
