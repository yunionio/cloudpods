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

package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"yunion.io/x/onecloud/pkg/util/probe"
)

func TestDoHTTPProbeRequestDefaultsToGet(t *testing.T) {
	var gotMethod, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	result, _, err := DoHTTPProbeRequestWithDialContext(srv.URL, "", nil, nil, time.Second, nil)
	if err != nil || result != probe.Success {
		t.Fatalf("probe result=%v err=%v", result, err)
	}
	if gotMethod != http.MethodGet || gotBody != "" {
		t.Fatalf("method=%q body=%q", gotMethod, gotBody)
	}
}

func TestDoHTTPProbeRequestPostsJSONBody(t *testing.T) {
	var gotMethod, gotBody, gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	body := []byte(`{"model":"m","input":"test"}`)
	result, _, err := DoHTTPProbeRequestWithDialContext(srv.URL, "POST", body, nil, time.Second, nil)
	if err != nil || result != probe.Success {
		t.Fatalf("probe result=%v err=%v", result, err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotCT != "application/json" {
		t.Fatalf("content-type=%q", gotCT)
	}
	if !strings.Contains(gotBody, `"input":"test"`) {
		t.Fatalf("body=%q", gotBody)
	}
}

func TestDoHTTPProbeRequestFailureStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	result, _, err := DoHTTPProbeRequestWithDialContext(srv.URL, "POST", []byte("{}"), nil, time.Second, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != probe.Failure {
		t.Fatalf("result=%v want failure", result)
	}
}
