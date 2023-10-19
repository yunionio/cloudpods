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

package sshkeys

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func AddSshKeysHandler(prefix string, app *appsrv.Application) {
	app.AddHandler2("GET", fmt.Sprintf("%s/sshkeypairs", prefix), auth.Authenticate(sshKeysHandler), nil, "get_sshkeys", nil)
	app.AddHandler2("GET", fmt.Sprintf("%s/sshkeypairs/<tenant_id>", prefix), auth.Authenticate(adminSshKeysHandler), nil, "get_sshkeys", nil)
}

func adminSshKeysHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	publicOnly := false
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	if !userCred.IsAllow(rbacscope.ScopeDomain, consts.GetServiceType(), "sshkeypairs", policy.PolicyActionGet).Result.IsAllow() {
		publicOnly = true
	}
	params, query, _ := appsrv.FetchEnv(ctx, w, r)
	projectId := params["<tenant_id>"]
	if len(projectId) == 0 {
		httperrors.InputParameterError(ctx, w, "empty project_id/tenant_id")
		return
	}
	domainId, _ := jsonutils.GetAnyString2(query, db.DomainFetchKeys)
	tenant, err := db.TenantCacheManager.FetchTenantByIdOrNameInDomain(ctx, projectId, domainId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			httperrors.ResourceNotFoundError(ctx, w, "tenant/project %s not found", projectId)
			return
		} else {
			httperrors.GeneralServerError(ctx, w, err)
			return
		}
	}

	// get project key of specific project
	sendSshKey(ctx, w, userCred, tenant.Id, false, publicOnly)
}

func sshKeysHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	query, err := jsonutils.ParseQueryString(r.URL.RawQuery)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	isAdmin := jsonutils.QueryBoolean(query, "admin", false)

	// get owner project key or get admin key if admin presents
	sendSshKey(ctx, w, userCred, userCred.GetProjectId(), isAdmin, false)
}

func sendSshKey(ctx context.Context, w http.ResponseWriter, userCred mcclient.TokenCredential, projectId string, isAdmin bool, publicOnly bool) {
	var privKey, pubKey string

	if isAdmin {
		if userCred.IsAllow(rbacscope.ScopeSystem, consts.GetServiceType(), "sshkeypairs", policy.PolicyActionGet).Result.IsAllow() {
			privKey, pubKey, _ = GetSshAdminKeypair(ctx)
		} else {
			httperrors.ForbiddenError(ctx, w, "not allow to access admin key")
			return
		}
	} else {
		privKey, pubKey, _ = GetSshProjectKeypair(ctx, projectId)
	}

	ret := jsonutils.NewDict()

	if !publicOnly {
		ret.Add(jsonutils.NewString(privKey), "private_key")
	}
	ret.Add(jsonutils.NewString(pubKey), "public_key")
	body := jsonutils.NewDict()
	body.Add(ret, "sshkeypair")
	appsrv.SendJSON(w, body)
}
