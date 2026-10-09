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
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/identity"
	webconsoleapi "yunion.io/x/onecloud/pkg/apis/webconsole"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type WebconsoleHandler struct {
	prefix string
}

func NewWebconsoleHandler(prefix string) *WebconsoleHandler {
	return &WebconsoleHandler{prefix}
}

func (h *WebconsoleHandler) GetPrefix() string {
	return h.prefix
}

func (h *WebconsoleHandler) Bind(app *appsrv.Application) {
	prefix := h.prefix

	app.AddHandler("GET", prefix+"/<service>/<sid>/list", FetchAuthToken(h.forwardToWebconsole))
	app.AddHandler("GET", prefix+"/<service>/<sid>/download", FetchAuthToken(h.forwardToWebconsole)).SetProcessTimeout(6 * time.Hour)
	app.AddHandler("POST", prefix+"/<service>/<sid>/upload", FetchAuthToken(h.forwardToWebconsole)).SetProcessTimeout(6 * time.Hour)
}

// forwardToWebconsole proxies /api/v1/webconsole/<service>/<sid>/<action>
// to the webconsole service /webconsole/<service>/<sid>/<action>.
func (h *WebconsoleHandler) forwardToWebconsole(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params := appctx.AppContextParams(ctx)
	service := params["<service>"]
	if !utils.IsInStringArray(service, []string{"sftp", "container"}) {
		httperrors.BadRequestError(ctx, w, "service %s not supported", service)
		return
	}
	sid := params["<sid>"]
	action := path.Base(r.URL.Path)
	if service == "" || sid == "" || action == "" || action == "." {
		httperrors.MissingParameterError(ctx, w, "service or sid")
		return
	}

	var body io.Reader
	headers := http.Header{}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		body = r.Body
		if ct := r.Header.Get("Content-Type"); ct != "" {
			headers.Set("Content-Type", ct)
		}
		if r.ContentLength >= 0 {
			headers.Set("Content-Length", fmt.Sprintf("%d", r.ContentLength))
		}
	}

	s := auth.GetSession(ctx, AppContextToken(ctx), FetchRegion(r))
	resp, err := s.RawRequest(
		webconsoleapi.SERVICE_TYPE,
		identity.EndpointInterfaceInternal,
		httputils.THttpMethod(r.Method),
		webconsoleBackendPath(service, sid, action, r.URL.RawQuery),
		headers,
		body,
	)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, errors.Wrap(err, "forward webconsole"))
		return
	}
	defer resp.Body.Close()

	copyHTTPHeader(w.Header(), resp.Header)
	w.Header().Del("Content-Length")
	w.Header().Del("Content-Encoding")
	w.Header().Del("Transfer-Encoding")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Errorf("copy webconsole %s response: %v", action, err)
	}
}

func webconsoleBackendPath(service, sid, action, rawQuery string) string {
	p := fmt.Sprintf("/webconsole/%s/%s/%s", url.PathEscape(service), url.PathEscape(sid), url.PathEscape(action))
	if rawQuery != "" {
		p += "?" + rawQuery
	}
	return p
}
