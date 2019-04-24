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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
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
		auth.Authenticate(getQuotaHanlder), nil, "get_quota", nil)

	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s", prefix, _manager.Keyword()),
		auth.Authenticate(setQuotaHanlder), nil, "set_quota", nil)

	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/<tenantid>", prefix, _manager.Keyword()),
		auth.Authenticate(setQuotaHanlder), nil, "set_quota", nil)

	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/<tenantid>/<action>", prefix, _manager.Keyword()),
		auth.Authenticate(checkQuotaHanlder), nil, "check_quota", nil)
}

func queryQuota(ctx context.Context, projectId string) (*jsonutils.JSONDict, error) {
	ret := jsonutils.NewDict()

	quota := _manager.newQuota()
	err := _manager.GetQuota(ctx, projectId, quota)
	if err != nil {
		return nil, err
	}
	usage := _manager.newQuota()
	err = usage.FetchUsage(ctx, projectId)
	if err != nil {
		return nil, err
	}
	pending := _manager.newQuota()
	err = _manager.GetPendingUsage(ctx, projectId, pending)
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

	projectId := params["<tenantid>"]
	if len(projectId) == 0 {
		projectId = userCred.GetProjectId()
		if consts.IsRbacEnabled() {
			result := policy.PolicyManager.Allow(false, userCred, consts.GetServiceType(),
				_manager.Keyword(), policy.PolicyActionGet)
			if result == rbacutils.Deny {
				httperrors.ForbiddenError(w, "not allow to get quota")
				return
			}
		}
	} else {
		isAllow := false
		if consts.IsRbacEnabled() {
			result := policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
				policy.PolicyDelegation, policy.PolicyActionGet)
			isAllow = result == rbacutils.AdminAllow
		} else {
			isAllow = userCred.IsAdminAllow(consts.GetServiceType(), policy.PolicyDelegation, policy.PolicyActionGet)
		}
		if !isAllow {
			httperrors.ForbiddenError(w, "not allow to delegate query quota")
			return
		}
		if consts.IsRbacEnabled() {
			if policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
				_manager.Keyword(), policy.PolicyActionGet) != rbacutils.AdminAllow {
				httperrors.ForbiddenError(w, "not allow to query quota")
				return
			}
		}

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
		projectId = tenant.Id
	}
	quota, err := queryQuota(ctx, projectId)

	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	body := jsonutils.NewDict()
	body.Add(quota, _manager.Keyword())

	appsrv.SendJSON(w, body)
}

func setQuotaHanlder(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)

	var isAllow bool
	if consts.IsRbacEnabled() {
		isAllow = policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
			_manager.Keyword(), policy.PolicyActionUpdate) == rbacutils.AdminAllow
	} else {
		isAllow = userCred.IsAdminAllow(consts.GetServiceType(),
			_manager.Keyword(), policy.PolicyActionUpdate)
	}
	if !isAllow {
		httperrors.ForbiddenError(w, "not allow to set quota")
		return
	}
	params := appctx.AppContextParams(ctx)
	projectId := params["<tenantid>"]
	if len(projectId) == 0 {
		projectId = userCred.GetProjectId()
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
		projectId = tenant.Id
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
	err = _manager.GetQuota(ctx, projectId, oquota)
	if err != nil {
		log.Errorf("get quota fail %s", err)
		httperrors.GeneralServerError(w, err)
		return
	}
	oquota.Update(quota)
	err = _manager.SetQuota(ctx, userCred, projectId, oquota)
	if err != nil {
		log.Errorf("set quota fail %s", err)
		httperrors.GeneralServerError(w, err)
		return
	}
	rbody := jsonutils.NewDict()
	rbody.Add(oquota.ToJSON(""), _manager.Keyword())
	appsrv.SendJSON(w, rbody)
}

func checkQuotaHanlder(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)

	isAllow := false
	if consts.IsRbacEnabled() {
		isAllow = policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
			policy.PolicyDelegation, policy.PolicyActionGet) == rbacutils.AdminAllow
	} else {
		isAllow = userCred.IsAdminAllow(consts.GetServiceType(),
			policy.PolicyDelegation, policy.PolicyActionGet)
	}
	if !isAllow {
		httperrors.ForbiddenError(w, "not allow to delegate check quota")
		return
	}
	if consts.IsRbacEnabled() {
		if policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
			_manager.Keyword(), policy.PolicyActionGet) != rbacutils.AdminAllow {
			httperrors.ForbiddenError(w, "not allow to query quota")
			return
		}
	}

	params := appctx.AppContextParams(ctx)
	projectId := params["<tenantid>"]
	if len(projectId) == 0 {
		projectId = userCred.GetProjectId()
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
		projectId = tenant.Id
	}
	body, err := appsrv.FetchJSON(r)
	quota := _manager.newQuota()
	err = body.Unmarshal(quota, _manager.Keyword())
	if err != nil {
		log.Errorf("Fail to decode JSON request body: %s", err)
		httperrors.InvalidInputError(w, "fail to decode body")
		return
	}
	used, err := _manager.CheckQuota(ctx, userCred, projectId, quota)
	if err != nil {
		httperrors.OutOfQuotaError(w, "Out of quota: %s", err)
		return
	}
	rbody := jsonutils.NewDict()
	rbody.Add(used.ToJSON(""), _manager.Keyword())
	appsrv.SendJSON(w, rbody)
}
