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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"yunion.io/x/onecloud/pkg/appsrv"
)

func TestLLMRouterAgentRouteRequiresAuth(t *testing.T) {
	app := appsrv.NewApplication("test-llm-router-route", 1, 1, false)
	registerLLMRouterAgentRoute(app)

	req := httptest.NewRequest("POST", "/llm_router_agents/some-id/route", strings.NewReader(`{"prompt":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("POST /llm_router_agents/<id>/route: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleLLMRouterAgentRouteUnauthorized(t *testing.T) {
	req := httptest.NewRequest("POST", "/llm_router_agents/some-id/route", strings.NewReader(`{"prompt":"hi"}`))
	w := httptest.NewRecorder()
	handleLLMRouterAgentRoute(context.Background(), w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no session: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
