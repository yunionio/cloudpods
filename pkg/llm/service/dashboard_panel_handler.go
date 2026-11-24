// Copyright 2026 Yunion
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
	"io"
	"net/http"
	"time"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/dashboardcodegen"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func registerDashboardPanelHandlers(app *appsrv.Application) {
	gen := app.AddHandler2("POST", "/dashboard_panels/generate", auth.Authenticate(handleDashboardPanelGenerate), nil, "dashboard_panel_generate", nil)
	gen.SetProcessTimeout(time.Minute * 5).SetWorkerManager(models.GetMCPAgentWorkerManager())

	app.AddHandler2("POST", "/dashboard_panels/validate", auth.Authenticate(handleDashboardPanelValidate), nil, "dashboard_panel_validate", nil)
	save := app.AddHandler2("POST", "/dashboard_panels/save", auth.Authenticate(handleDashboardPanelSave), nil, "dashboard_panel_save", nil)
	save.SetProcessTimeout(time.Minute * 2)
}

func readDashboardPanelBody(r *http.Request, dest interface{}) error {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(bodyBytes) == 0 {
		return httperrors.ErrEmptyRequest
	}
	body, err := jsonutils.Parse(bodyBytes)
	if err != nil {
		return err
	}
	if body == nil {
		return httperrors.ErrEmptyRequest
	}
	// mcclient / 前端 Manager 常经 params2Body 包一层 keyword：
	// {"dashboard_panel": {...}} 或 {"dashboard_panels": {...}}；与 mcp_agents 自定义 handler 一致做解包。
	for _, key := range []string{"dashboard_panel", "dashboard_panels"} {
		if body.Contains(key) {
			if inner, _ := body.Get(key); inner != nil {
				body = inner
			}
			break
		}
	}
	return body.Unmarshal(dest)
}

// PerformClassAction / 前端 Manager 用 KeywordPlural 包请求、解响应。
const dashboardPanelRespKeyword = "dashboard_panels"

// sendDashboardPanelJSON wraps the payload as {"dashboard_panels": ...}.
func sendDashboardPanelJSON(w http.ResponseWriter, out interface{}) {
	wrapped := jsonutils.NewDict()
	wrapped.Set(dashboardPanelRespKeyword, jsonutils.Marshal(out))
	appsrv.SendJSON(w, wrapped)
}

func handleDashboardPanelGenerate(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	if userCred == nil {
		httperrors.UnauthorizedError(ctx, w, "Unauthorized")
		return
	}
	var input api.DashboardPanelGenerateInput
	if err := readDashboardPanelBody(r, &input); err != nil {
		httperrors.InvalidInputError(ctx, w, "invalid body: %v", err)
		return
	}
	out, err := dashboardcodegen.Generate(ctx, userCred, input)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	sendDashboardPanelJSON(w, out)
}

func handleDashboardPanelValidate(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	if userCred == nil {
		httperrors.UnauthorizedError(ctx, w, "Unauthorized")
		return
	}
	var input api.DashboardPanelValidateInput
	if err := readDashboardPanelBody(r, &input); err != nil {
		httperrors.InvalidInputError(ctx, w, "invalid body: %v", err)
		return
	}
	out, err := dashboardcodegen.ValidateOnly(input)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	sendDashboardPanelJSON(w, out)
}

func handleDashboardPanelSave(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	if userCred == nil {
		httperrors.UnauthorizedError(ctx, w, "Unauthorized")
		return
	}
	var input api.DashboardPanelSaveInput
	if err := readDashboardPanelBody(r, &input); err != nil {
		httperrors.InvalidInputError(ctx, w, "invalid body: %v", err)
		return
	}
	out, err := dashboardcodegen.Save(ctx, userCred, input)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	sendDashboardPanelJSON(w, out)
}
