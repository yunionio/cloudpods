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
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type CSRFResourceHandler struct {
	*SHandlers
}

func NewCSRFResourceHandler(prefix string) *CSRFResourceHandler {
	return &CSRFResourceHandler{NewHandlers(prefix)}
}

func (h *CSRFResourceHandler) Bind(app *appsrv.Application) {
	h.AddByMethod(GET, nil, NewHP(getHandlerCsrf, APIVer, "csrf", ResName, ResID))
	h.SHandlers.Bind(app)
}

func getAdminSession(ctx context.Context, apiVer string, region string, w http.ResponseWriter) *mcclient.ClientSession {
	adminToken := auth.AdminCredential()
	if adminToken == nil {
		httperrors.NotFoundError(w, "get admin credential is nil")
		return nil
	}
	regions := adminToken.GetRegions()
	log.Infof("CSRF regions: %v", regions)
	if len(regions) == 0 {
		httperrors.NotFoundError(w, "no usable regions, please contact admin")
		return nil
	}
	ret, _ := sets.InArray(region, regions)
	if !ret {
		httperrors.NotFoundError(w, "illegal region %s, please contact admin", region)
	}
	s := auth.GetAdminSession(ctx, region, apiVer)
	return s
}

func fetchEnv3Csrf(ctx context.Context, w http.ResponseWriter, r *http.Request) (modulebase.Manager, modulebase.Manager, modulebase.Manager, *mcclient.ClientSession, map[string]string, jsonutils.JSONObject, jsonutils.JSONObject) {
	module, module2, session, params, query, body := fetchEnv2Csrf(ctx, w, r)
	if module == nil || module2 == nil {
		return nil, nil, nil, nil, nil, nil, nil
	}
	module3, e := modulebase.GetModule(session, params[ResName3])
	if e != nil || module == nil {
		httperrors.NotFoundError(w, fmt.Sprintf("resource %s not found", params[ResName3]))
		return nil, nil, nil, nil, nil, nil, nil
	}
	return module, module2, module3, session, params, query, body
}

func fetchEnv2Csrf(ctx context.Context, w http.ResponseWriter, r *http.Request) (modulebase.Manager, modulebase.Manager, *mcclient.ClientSession, map[string]string, jsonutils.JSONObject, jsonutils.JSONObject) {
	module, session, params, query, body := fetchEnvCsrf(ctx, w, r)
	if module == nil {
		return nil, nil, nil, nil, nil, nil
	}
	module2, e := modulebase.GetModule(session, params[ResName2])
	if e != nil || module == nil {
		httperrors.NotFoundError(w, fmt.Sprintf("resource %s not found", params[ResName2]))
		return nil, nil, nil, nil, nil, nil
	}
	return module, module2, session, params, query, body
}

func fetchEnvCsrf(ctx context.Context, w http.ResponseWriter, r *http.Request) (modulebase.Manager, *mcclient.ClientSession, map[string]string, jsonutils.JSONObject, jsonutils.JSONObject) {
	session, params, query, body := fetchEnvCsrf0(ctx, w, r)
	module, e := modulebase.GetModule(session, params[ResName])
	if e != nil || module == nil {
		httperrors.NotFoundError(w, fmt.Sprintf("resource %s not found", params[ResName]))
		return nil, nil, nil, nil, nil
	}
	return module, session, params, query, body
}

func fetchEnvCsrf0(ctx context.Context, w http.ResponseWriter, r *http.Request) (*mcclient.ClientSession, map[string]string, jsonutils.JSONObject, jsonutils.JSONObject) {
	params := appctx.AppContextParams(ctx)
	region := r.URL.Query().Get("region")
	log.Println("csrf region from url:", region)
	if len(region) < 1 {
		httperrors.NotFoundError(w, fmt.Sprintf("region %s is empty", region))
		return nil, nil, nil, nil
	}
	log.Infof("csrf region from url: %s", region)
	session := getAdminSession(ctx, params[APIVer], region, w)
	log.Infof("csrf got session: %s", region)
	if session == nil {
		return nil, nil, nil, nil
	}
	query, e := jsonutils.ParseQueryString(r.URL.RawQuery)
	if e != nil {
		log.Errorf("Parse query string %s: %v", r.URL.RawQuery, e)
	}
	var body jsonutils.JSONObject = nil
	if r.Method == PUT || r.Method == POST || r.Method == DELETE || r.Method == PATCH {
		body, e = appsrv.FetchJSON(r)
		if e != nil {
			log.Errorf("Fail to decode JSON request body: %v", e)
		}
	}
	return session, params, query, body
}

func getHandlerCsrf(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	module, session, params, query, _ := fetchEnvCsrf(ctx, w, r)
	if module == nil {
		return
	}
	obj, e := module.Get(session, params[ResID], query)
	if e != nil {
		httperrors.GeneralServerError(w, e)
	} else {
		appsrv.SendJSON(w, obj)
	}
}
