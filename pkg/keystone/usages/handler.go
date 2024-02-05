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
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/tagutils"
)

func AddUsageHandler(prefix string, app *appsrv.Application) {
	prefix = fmt.Sprintf("%s/usages", prefix)
	app.AddHandler2("GET", prefix, auth.Authenticate(ReportGeneralUsage), nil, "get_usage", nil)
}

func ReportGeneralUsage(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)

	_, _, err, result := db.FetchUsageOwnerScope(ctx, userCred, query)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	projectTags := &tagutils.TTagSetList{}
	query.Unmarshal(projectTags, "project_tags")
	for i := range result.ProjectTags {
		projectTags.Append(result.ProjectTags[i])
	}
	result.ProjectTags = *projectTags

	isAdmin := false
	if policy.PolicyManager.Allow(rbacscope.ScopeSystem, userCred, consts.GetServiceType(),
		"usages", policy.PolicyActionGet).Result.IsAllow() {
		isAdmin = true
	}

	var adminUsage map[string]int
	// var projectUsage map[string]int64
	if isAdmin {
		adminUsage = models.Usage(ctx, result)
	}

	/*isProject := false
	if policy.PolicyManager.Allow(false, userCred, consts.GetServiceType(),
		"usages", policy.PolicyActionGet) == rbacutils.Deny {
		isProject = false
	} else {
		isProject = true
	}

	if isProject {
		projectUsage = models.Usage(userCred.GetProjectId(), "")
	}*/

	// if !isAdmin && !isProject {
	if !isAdmin {
		httperrors.ForbiddenError(ctx, w, "not allow to get usage")
		return
	}

	usages := jsonutils.NewDict()
	// if isProject {
	//	usages.Update(jsonutils.Marshal(projectUsage))
	// }

	if isAdmin {
		usages.Update(jsonutils.Marshal(adminUsage))
	}

	body := jsonutils.NewDict()
	body.Add(usages, "usage")
	appsrv.SendJSON(w, body)
}
