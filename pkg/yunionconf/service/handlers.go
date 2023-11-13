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

package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/yunionconf/models"
)

func InitHandlers(app *appsrv.Application) {
	db.InitAllManagers()

	db.AddScopeResourceCountHandler("", app)
	addBugReportHandler("", app)

	for _, manager := range []db.IModelManager{
		db.UserCacheManager,
		db.TenantCacheManager,
		db.SharedResourceManager,
		db.OpsLog,
	} {
		db.RegisterModelManager(manager)
	}

	for _, manager := range []db.IModelManager{
		db.Metadata,
		models.ParameterManager,
		models.ScopedPolicyBindingManager,
		models.ScopedPolicyManager,
	} {
		db.RegisterModelManager(manager)
		handler := db.NewModelHandler(manager)
		dispatcher.AddModelDispatcher("", app, handler)
		if manager == models.ParameterManager {
			dispatcher.AddModelDispatcher("/users/<user_id>", app, handler)
			dispatcher.AddModelDispatcher("/services/<service_id>", app, handler)
		}
	}
}

func addBugReportHandler(prefix string, app *appsrv.Application) {
	app.AddHandler("GET", fmt.Sprintf("%s/bug-report-status", prefix), bugReportStatusHandler)
	app.AddHandler("POST", fmt.Sprintf("%s/enable-bug-report", prefix), enableBugReportHandler)
	app.AddHandler("POST", fmt.Sprintf("%s/disable-bug-report", prefix), disableBugReportHandler)
	app.AddHandler("POST", fmt.Sprintf("%s/send-bug-report", prefix), sendBugReportHandler)
}

func bugReportStatusHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	enabled := models.ParameterManager.GetBugReportEnabled()
	appsrv.SendJSON(w, jsonutils.Marshal(map[string]bool{"enabled": enabled}))
}

func enableBugReportHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	enabled := models.ParameterManager.EnableBugReport(ctx)
	appsrv.SendJSON(w, jsonutils.Marshal(map[string]bool{"enabled": enabled}))
}

func disableBugReportHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	models.ParameterManager.DisableBugReport(ctx)
	appsrv.SendJSON(w, jsonutils.Marshal(map[string]bool{"enabled": false}))
}

func sendBugReportHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if !models.ParameterManager.GetBugReportEnabled() {
		appsrv.SendJSON(w, jsonutils.Marshal(map[string]bool{"status": false}))
		return
	}
	_, _, body := appsrv.FetchEnv(ctx, w, r)
	apiServer := options.Options.ApiServer
	if len(apiServer) == 0 {
		s := auth.GetAdminSession(ctx, options.Options.Region)
		commonCfg, _ := identity.ServicesV3.GetSpecific(s, "common", "config", nil)
		if commonCfg != nil {
			_apiServer, _ := commonCfg.GetString("config", "default", "api_server")
			if len(_apiServer) > 0 {
				apiServer = _apiServer
			}
		}
	}
	params := jsonutils.NewDict()
	params.Set("api_server", jsonutils.NewString(apiServer))
	params.Update(body)
	url, _ := base64.StdEncoding.DecodeString("aHR0cHM6Ly9jbG91ZC55dW5pb24uY24vYXBpL3YyL2J1Zy1yZXBvcnQ=")
	_, _, err := httputils.JSONRequest(nil, ctx, httputils.POST, string(url), nil, params, false)
	if err != nil {
		appsrv.SendJSON(w, jsonutils.Marshal(map[string]bool{"status": false}))
		return
	}
	appsrv.SendJSON(w, jsonutils.Marshal(map[string]bool{"status": true}))
}
