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

package appsrv

import (
	"net/http/httptest"
	"testing"
)

// with an unset or wildcard origin allowlist, credentials must never be
// honored: echoing any origin with credentials would let any website make
// authenticated cross-origin requests
func TestCorsWildcardDisablesCredentials(t *testing.T) {
	c := NewCors(CorsOptions{
		AllowedMethods:   []string{"GET"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	if c.allowCredentials {
		t.Fatalf("wildcard origins must not honor credentials")
	}
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "http://evil.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	c.handlePreflight(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Allow-Origin = %q, want *", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("Allow-Credentials must not be set, got %q", got)
	}

	// actual request
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://evil.com")
	w = httptest.NewRecorder()
	c.handleActualRequest(w, req)
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("Allow-Credentials must not be set, got %q", got)
	}
}

// explicitly listed origins keep credentials, other origins get nothing
func TestCorsExplicitOriginsKeepCredentials(t *testing.T) {
	c := NewCors(CorsOptions{
		AllowedOrigins:   []string{"foo.com"},
		AllowedMethods:   []string{"GET"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	if !c.allowCredentials {
		t.Fatalf("explicit origins must honor credentials")
	}
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "http://foo.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	c.handlePreflight(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://foo.com" {
		t.Fatalf("Allow-Origin = %q, want http://foo.com", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Allow-Credentials = %q, want true", got)
	}

	// a disallowed origin gets no CORS headers
	req = httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "http://evil.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w = httptest.NewRecorder()
	c.handlePreflight(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("disallowed origin must not get Allow-Origin, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("disallowed origin must not get Allow-Credentials, got %q", got)
	}
}
