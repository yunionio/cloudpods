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

package handler

import (
	"context"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type CSRFResourceHandler struct {
	*SHandlers
}

func NewCSRFResourceHandler(prefix string) *CSRFResourceHandler {
	return &CSRFResourceHandler{NewHandlers(prefix)}
}

func (h *CSRFResourceHandler) Bind(app *appsrv.Application) {
	h.AddByMethod(GET, FetchAuthToken, NewHP(getHandlerCsrf, APIVer, "csrf", ResName, ResID))
	h.SHandlers.Bind(app)
}

func getHandlerCsrf(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	token := AppContextToken(ctx)
	if token == nil {
		httperrors.UnauthorizedError(ctx, w, "No valid auth token found")
		return
	}
	params := appctx.AppContextParams(ctx)
	region := r.URL.Query().Get("region")
	if len(region) < 1 {
		httperrors.NotFoundError(ctx, w, "region %s is empty", region)
		return
	}
	session := auth.GetSession(ctx, token, region)
	if session == nil {
		httperrors.GeneralServerError(ctx, w, httperrors.ErrInvalidCredential)
		return
	}
	module, e := modulebase.GetModule(session, params[ResName])
	if e != nil || module == nil {
		httperrors.NotFoundError(ctx, w, "resource %s not found", params[ResName])
		return
	}
	query, e := jsonutils.ParseQueryString(r.URL.RawQuery)
	if e != nil {
		log.Errorf("Parse query string %s: %v", r.URL.RawQuery, e)
	}
	obj, e := module.Get(session, params[ResID], query)
	if e != nil {
		httperrors.GeneralServerError(ctx, w, e)
		return
	}
	appsrv.SendJSON(w, obj)
}
