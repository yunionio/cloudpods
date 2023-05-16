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

package app

import (
	"context"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

func ExportOptionsHandler(app *appsrv.Application, options interface{}) {
	ExportOptionsHandlerWithPrefix(app, "", options)
}

func ExportOptionsHandlerWithPrefix(app *appsrv.Application, prefix string, options interface{}) {
	hf := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
		result := policy.PolicyManager.Allow(rbacscope.ScopeSystem, userCred, consts.GetServiceType(), "app-options", "list")
		if result.Result == rbacutils.Deny {
			httperrors.ForbiddenError(ctx, w, "Not allow to access")
			return
		}
		appsrv.SendJSON(w, jsonutils.Marshal(options))
	}
	ahf := auth.Authenticate(hf)
	name := "get_app_options"
	pref := httputils.JoinPath(prefix, "app-options")
	app.AddHandler2("GET", pref, ahf, nil, name, nil)
}
