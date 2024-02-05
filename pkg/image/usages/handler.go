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

package usages

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

func AddUsageHandler(prefix string, app *appsrv.Application) {
	prefix = fmt.Sprintf("%s/usages", prefix)
	app.AddHandler2("GET", prefix, auth.Authenticate(ReportGeneralUsage), nil, "get_usage", nil)
}

func ReportGeneralUsage(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)

	ownerId, scope, err, result := db.FetchUsageOwnerScope(ctx, userCred, query)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	tags := rbacutils.SPolicyResult{Result: rbacutils.Allow}
	query.Unmarshal(&tags)
	result = result.Merge(tags)

	usages := jsonutils.NewDict()
	if scope == rbacscope.ScopeSystem {
		adminUsage := models.ImageManager.Usage(ctx, rbacscope.ScopeSystem, ownerId, "all", result)
		usages.Update(jsonutils.Marshal(adminUsage))
		adminUsage = models.GuestImageManager.Usage(ctx, rbacscope.ScopeSystem, ownerId, "all", result)
		usages.Update(jsonutils.Marshal(adminUsage))
	}

	if scope.HigherEqual(rbacscope.ScopeDomain) {
		domainUsage := models.ImageManager.Usage(ctx, rbacscope.ScopeDomain, ownerId, "domain", result)
		usages.Update(jsonutils.Marshal(domainUsage))
		domainUsage = models.GuestImageManager.Usage(ctx, rbacscope.ScopeDomain, ownerId, "domain", result)
		usages.Update(jsonutils.Marshal(domainUsage))
	}

	if scope.HigherEqual(rbacscope.ScopeProject) {
		projectUsage := models.ImageManager.Usage(ctx, rbacscope.ScopeProject, ownerId, "", result)
		usages.Update(jsonutils.Marshal(projectUsage))
		projectUsage = models.GuestImageManager.Usage(ctx, rbacscope.ScopeProject, ownerId, "", result)
		usages.Update(jsonutils.Marshal(projectUsage))
	}

	body := jsonutils.NewDict()
	body.Add(usages, "usage")
	appsrv.SendJSON(w, body)
}
