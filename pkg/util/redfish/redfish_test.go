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

package redfish

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"yunion.io/x/jsonutils"
)

const (
	testBasePath = "/redfish/v1"
	testUser     = "admin"
	testPassword = "secret"
	testToken    = "session-token"

	testRoot = `{
		"RedfishVersion": "1.0.0",
		"Systems": {"@odata.id": "/redfish/v1/Systems"},
		"Managers": {"@odata.id": "/redfish/v1/Managers"}
	}`

	testSystemsCollection = `{
		"Members": [
			{"@odata.id": "/redfish/v1/Systems/system"}
		]
	}`

	testSystem = `{
		"Id": "system",
		"SerialNumber": " SN123 ",
		"SKU": " SKU ",
		"Model": " Inspur CS5280H ",
		"Manufacturer": " Inspur ",
		"PowerState": "on",
		"MemorySummary": {
			"TotalSystemMemoryGiB": "64"
		},
		"ProcessorSummary": {
			"Count": 2,
			"Model": "Intel Xeon"
		},
		"Boot": {
			"BootSourceOverrideTarget": "Pxe",
			"BootSourceOverrideSupported": ["Pxe"]
		},
		"Actions": {
			"#ComputerSystem.Reset": {
				"ResetType@Redfish.AllowableValues": ["ForceRestart"]
			}
		}
	}`
)

type testRedfishApi struct {
	SBaseRedfishClient
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestRedfishApi(handler func(req *http.Request) *http.Response) *testRedfishApi {
	api := &testRedfishApi{
		SBaseRedfishClient: NewBaseRedfishClient("http://redfish.test", testUser, testPassword, false),
	}
	api.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return handler(req), nil
		}),
	}
	api.SetVirtualObject(api)
	return api
}

func (r *testRedfishApi) BasePath() string {
	return testBasePath
}

func (r *testRedfishApi) GetParent(parent jsonutils.JSONObject) jsonutils.JSONObject {
	return parent
}

func (r *testRedfishApi) VersionKey() string {
	return "RedfishVersion"
}

func (r *testRedfishApi) LinkKey() string {
	return "@odata.id"
}

func (r *testRedfishApi) MemberKey() string {
	return "Members"
}

func (r *testRedfishApi) LogItemsKey() string {
	return "Members"
}

func TestProbeBasicAuthSucceedsWithoutLogin(t *testing.T) {
	rootGets := 0
	loginCalls := 0
	sawBasicAuth := false

	api := newTestRedfishApi(func(r *http.Request) *http.Response {
		switch r.URL.Path {
		case testBasePath:
			rootGets++
			user, password, ok := r.BasicAuth()
			if ok && user == testUser && password == testPassword {
				sawBasicAuth = true
			}
			if token := r.Header.Get("X-Auth-Token"); len(token) > 0 {
				t.Errorf("unexpected X-Auth-Token on basic auth request: %s", token)
			}
			return jsonResponse(r, http.StatusOK, testRoot)
		case "/redfish/v1/Sessions":
			loginCalls++
			return jsonResponse(r, http.StatusCreated, `{}`)
		default:
			return notFoundResponse(r)
		}
	})

	if err := api.Probe(context.Background()); err != nil {
		t.Fatalf("Probe error: %v", err)
	}
	if rootGets != 1 {
		t.Fatalf("root GET count got %d want 1", rootGets)
	}
	if loginCalls != 0 {
		t.Fatalf("login call count got %d want 0", loginCalls)
	}
	if !sawBasicAuth {
		t.Fatalf("expected Probe to use basic auth")
	}
	if len(api.SessionToken) > 0 {
		t.Fatalf("SessionToken got %q want empty", api.SessionToken)
	}
	if got := api.Registries["Systems"]; got != "/redfish/v1/Systems" {
		t.Fatalf("Systems registry got %q want /redfish/v1/Systems", got)
	}
}

func TestProbeFallsBackToSessionTokenAuth(t *testing.T) {
	rootGets := 0
	loginCalls := 0
	managerGets := 0

	api := newTestRedfishApi(func(r *http.Request) *http.Response {
		switch r.URL.Path {
		case testBasePath:
			rootGets++
			if r.Header.Get("X-Auth-Token") != testToken {
				return unauthorizedResponse(r, "basic auth rejected")
			}
			return jsonResponse(r, http.StatusOK, testRoot)
		case "/redfish/v1/Sessions":
			loginCalls++
			requireBasicAuth(t, r)
			hdr := http.Header{}
			hdr.Set("X-Auth-Token", testToken)
			hdr.Set("Location", "/redfish/v1/Sessions/1")
			return jsonResponseWithHeader(r, http.StatusCreated, `{}`, hdr)
		case "/redfish/v1/Managers":
			managerGets++
			if got := r.Header.Get("X-Auth-Token"); got != testToken {
				t.Errorf("subsequent request X-Auth-Token got %q want %q", got, testToken)
			}
			return jsonResponse(r, http.StatusOK, `{"Members":[]}`)
		default:
			return notFoundResponse(r)
		}
	})

	if err := api.Probe(context.Background()); err != nil {
		t.Fatalf("Probe error: %v", err)
	}
	if rootGets != 2 {
		t.Fatalf("root GET count got %d want 2", rootGets)
	}
	if loginCalls != 1 {
		t.Fatalf("login call count got %d want 1", loginCalls)
	}
	if api.SessionToken != testToken {
		t.Fatalf("SessionToken got %q want %q", api.SessionToken, testToken)
	}
	if _, err := api.Get(context.Background(), "/redfish/v1/Managers"); err != nil {
		t.Fatalf("subsequent token-authenticated GET error: %v", err)
	}
	if managerGets != 1 {
		t.Fatalf("manager GET count got %d want 1", managerGets)
	}
}

func TestProbeReturnsOriginalErrorWhenSessionLoginFails(t *testing.T) {
	rootGets := 0
	loginCalls := 0

	api := newTestRedfishApi(func(r *http.Request) *http.Response {
		switch r.URL.Path {
		case testBasePath:
			rootGets++
			return unauthorizedResponse(r, "unauthorized-root")
		case "/redfish/v1/Sessions":
			loginCalls++
			return unauthorizedResponse(r, "login-failed")
		default:
			return notFoundResponse(r)
		}
	})

	err := api.Probe(context.Background())
	if err == nil {
		t.Fatalf("Probe returned nil error")
	}
	if rootGets != 1 {
		t.Fatalf("root GET count got %d want 1", rootGets)
	}
	if loginCalls != 1 {
		t.Fatalf("login call count got %d want 1", loginCalls)
	}
	if !strings.Contains(err.Error(), "unauthorized-root") {
		t.Fatalf("Probe error %q does not contain original root error", err.Error())
	}
	if strings.Contains(err.Error(), "login-failed") {
		t.Fatalf("Probe error %q should not contain login failure", err.Error())
	}
	if len(api.SessionToken) > 0 {
		t.Fatalf("SessionToken got %q want empty", api.SessionToken)
	}
}

func TestProbeDoesNotFallbackToSessionTokenAuthForNonAuthErrors(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		response   func(*http.Request) *http.Response
	}{
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			response:   notFoundResponse,
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			response:   serverErrorResponse,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rootGets := 0
			loginCalls := 0

			api := newTestRedfishApi(func(r *http.Request) *http.Response {
				switch r.URL.Path {
				case testBasePath:
					rootGets++
					return c.response(r)
				case "/redfish/v1/Sessions":
					loginCalls++
					hdr := http.Header{}
					hdr.Set("X-Auth-Token", testToken)
					hdr.Set("Location", "/redfish/v1/Sessions/1")
					return jsonResponseWithHeader(r, http.StatusCreated, `{}`, hdr)
				default:
					return notFoundResponse(r)
				}
			})

			err := api.Probe(context.Background())
			if err == nil {
				t.Fatalf("Probe returned nil error")
			}
			if rootGets != 1 {
				t.Fatalf("root GET count got %d want 1", rootGets)
			}
			if loginCalls != 0 {
				t.Fatalf("login call count got %d want 0 for status %d", loginCalls, c.statusCode)
			}
			if len(api.SessionToken) > 0 {
				t.Fatalf("SessionToken got %q want empty", api.SessionToken)
			}
		})
	}
}

func TestProbeDoesNotLoopWhenSessionRetryFails(t *testing.T) {
	rootGets := 0
	loginCalls := 0

	api := newTestRedfishApi(func(r *http.Request) *http.Response {
		switch r.URL.Path {
		case testBasePath:
			rootGets++
			if rootGets == 1 {
				return unauthorizedResponse(r, "basic auth rejected")
			}
			return plainResponse(r, http.StatusOK, "not-json")
		case "/redfish/v1/Sessions":
			loginCalls++
			hdr := http.Header{}
			hdr.Set("X-Auth-Token", testToken)
			hdr.Set("Location", "/redfish/v1/Sessions/1")
			return jsonResponseWithHeader(r, http.StatusCreated, `{}`, hdr)
		default:
			return notFoundResponse(r)
		}
	})

	err := api.Probe(context.Background())
	if err == nil {
		t.Fatalf("Probe returned nil error")
	}
	if rootGets != 2 {
		t.Fatalf("root GET count got %d want 2", rootGets)
	}
	if loginCalls != 1 {
		t.Fatalf("login call count got %d want 1", loginCalls)
	}
	if !strings.Contains(err.Error(), "Response is nil") {
		t.Fatalf("Probe error got %q want Response is nil", err.Error())
	}
}

func TestProbeSkipsSessionFallbackWhenSessionTokenAlreadySet(t *testing.T) {
	rootGets := 0
	loginCalls := 0

	api := newTestRedfishApi(func(r *http.Request) *http.Response {
		switch r.URL.Path {
		case testBasePath:
			rootGets++
			return unauthorizedResponse(r, "stale token rejected")
		case "/redfish/v1/Sessions":
			loginCalls++
			hdr := http.Header{}
			hdr.Set("X-Auth-Token", testToken)
			return jsonResponseWithHeader(r, http.StatusCreated, `{}`, hdr)
		default:
			return notFoundResponse(r)
		}
	})

	api.SessionToken = "stale-token"
	if err := api.Probe(context.Background()); err == nil {
		t.Fatalf("Probe returned nil error")
	}
	if rootGets != 1 {
		t.Fatalf("root GET count got %d want 1", rootGets)
	}
	if loginCalls != 0 {
		t.Fatalf("login call count got %d want 0", loginCalls)
	}
	if api.SessionToken != "stale-token" {
		t.Fatalf("SessionToken got %q want stale-token", api.SessionToken)
	}
}

func TestGetSystemInfoUsesMemberIndexForNonStandardSystemPath(t *testing.T) {
	cases := []struct {
		name           string
		requireSession bool
	}{
		{name: "basic", requireSession: false},
		{name: "session", requireSession: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			loginCalls := 0
			systemGets := 0

			api := newTestRedfishApi(func(r *http.Request) *http.Response {
				switch r.URL.Path {
				case testBasePath:
					if !requestAuthorized(c.requireSession, r) {
						return unauthorizedResponse(r, "root rejected")
					}
					return jsonResponse(r, http.StatusOK, testRoot)
				case "/redfish/v1/Sessions":
					loginCalls++
					hdr := http.Header{}
					hdr.Set("X-Auth-Token", testToken)
					hdr.Set("Location", "/redfish/v1/Sessions/1")
					return jsonResponseWithHeader(r, http.StatusCreated, `{}`, hdr)
				case "/redfish/v1/Systems":
					if !requestAuthorized(c.requireSession, r) {
						return unauthorizedResponse(r, "systems rejected")
					}
					return jsonResponse(r, http.StatusOK, testSystemsCollection)
				case "/redfish/v1/Systems/system":
					systemGets++
					if !requestAuthorized(c.requireSession, r) {
						return unauthorizedResponse(r, "system rejected")
					}
					return jsonResponse(r, http.StatusOK, testSystem)
				default:
					return notFoundResponse(r)
				}
			})

			if err := api.Probe(context.Background()); err != nil {
				t.Fatalf("Probe error: %v", err)
			}

			path, info, err := api.GetSystemInfo(context.Background())
			if err != nil {
				t.Fatalf("GetSystemInfo error: %v", err)
			}
			if path != "/redfish/v1/Systems/system" {
				t.Fatalf("system path got %q want /redfish/v1/Systems/system", path)
			}
			if info.Model != "Inspur CS5280H" {
				t.Fatalf("system model got %q want Inspur CS5280H", info.Model)
			}
			if systemGets != 1 {
				t.Fatalf("system GET count got %d want 1", systemGets)
			}
			if c.requireSession && loginCalls != 1 {
				t.Fatalf("login call count got %d want 1", loginCalls)
			}
			if !c.requireSession && loginCalls != 0 {
				t.Fatalf("login call count got %d want 0", loginCalls)
			}
		})
	}
}

func requestAuthorized(requireSession bool, r *http.Request) bool {
	if requireSession {
		return r.Header.Get("X-Auth-Token") == testToken
	}
	user, password, ok := r.BasicAuth()
	return ok && user == testUser && password == testPassword && len(r.Header.Get("X-Auth-Token")) == 0
}

func requireBasicAuth(t *testing.T, r *http.Request) {
	user, password, ok := r.BasicAuth()
	if !ok || user != testUser || password != testPassword {
		t.Errorf("basic auth got user=%q password=%q ok=%v", user, password, ok)
	}
}

func unauthorizedResponse(req *http.Request, message string) *http.Response {
	return jsonResponse(req, http.StatusUnauthorized, fmt.Sprintf(`{"error":{"code":"401","message":%q}}`, message))
}

func notFoundResponse(req *http.Request) *http.Response {
	return jsonResponse(req, http.StatusNotFound, `{"error":{"code":"404","message":"not found"}}`)
}

func serverErrorResponse(req *http.Request) *http.Response {
	return jsonResponse(req, http.StatusInternalServerError, `{"error":{"code":"500","message":"server error"}}`)
}

func jsonResponse(req *http.Request, status int, body string) *http.Response {
	return jsonResponseWithHeader(req, status, body, nil)
}

func jsonResponseWithHeader(req *http.Request, status int, body string, hdr http.Header) *http.Response {
	if hdr == nil {
		hdr = http.Header{}
	}
	hdr.Set("Content-Type", "application/json")
	return response(req, status, body, hdr)
}

func plainResponse(req *http.Request, status int, body string) *http.Response {
	return response(req, status, body, http.Header{"Content-Type": []string{"text/plain"}})
}

func response(req *http.Request, status int, body string, hdr http.Header) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     hdr,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}
