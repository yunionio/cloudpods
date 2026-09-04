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
	"net/http/httptest"
	"testing"

	"yunion.io/x/pkg/appctx"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func TestCsrfEndpointRequiresAuth(t *testing.T) {
	app := appsrv.NewApplication("test-csrf", 1, 1, false)
	NewCSRFResourceHandler("/api").Bind(app)

	for _, url := range []string{
		"/api/v1/csrf/servers/some-server?region=region0",
		"/api/v1/csrf/networks/some-network",
	} {
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("GET %s: status = %d, want %d", url, w.Code, http.StatusUnauthorized)
		}
	}
}

func TestCsrfEndpointRejectsInvalidCookie(t *testing.T) {
	app := appsrv.NewApplication("test-csrf", 1, 1, false)
	NewCSRFResourceHandler("/api").Bind(app)

	req := httptest.NewRequest("GET", "/api/v1/csrf/servers/some-server?region=region0", nil)
	req.AddCookie(&http.Cookie{Name: "yunionauth", Value: "invalid-session-value"})
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("invalid cookie GET: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestGetHandlerCsrfGuardClauses(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/csrf/servers/some-server?region=region0", nil)
	w := httptest.NewRecorder()
	getHandlerCsrf(context.Background(), w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no token: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	ctx := context.WithValue(context.Background(), appctx.APP_CONTEXT_KEY_AUTH_TOKEN, &mcclient.SSimpleToken{})
	req = httptest.NewRequest("GET", "/api/v1/csrf/servers/some-server", nil)
	w = httptest.NewRecorder()
	getHandlerCsrf(ctx, w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("empty region: status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
