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

package handlers

import (
	"net/http"
	"net/url"
	"testing"
)

// the object key must be percent-decoded exactly once (the form the
// signature covers); decoding twice would let a signature for one key be
// replayed against another
func TestGetObjectRequestSingleDecode(t *testing.T) {
	cases := []struct {
		name    string
		urlPath string // URL.Path as net/http sets it: raw path decoded once
		wantKey string
	}{
		// raw /bucket/a%20b -> Path /bucket/a b
		{name: "simple space", urlPath: "/bucket/a b", wantKey: "a b"},
		// raw /bucket/a%252Fb -> Path /bucket/a%2Fb, must stay literal
		{name: "literal percent", urlPath: "/bucket/a%2Fb", wantKey: "a%2Fb"},
		// raw /bucket/a%252e%252e%252fc -> Path /bucket/a%2e%2e%2fc,
		// must not resolve to a/../c
		{name: "no double dot resolution", urlPath: "/bucket/a%2e%2e%2fc", wantKey: "a%2e%2e%2fc"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := &http.Request{
				Host: "127.0.0.1",
				URL:  &url.URL{Path: c.urlPath},
			}
			o, err := getObjectRequest(req)
			if err != nil {
				t.Fatalf("getObjectRequest: %v", err)
			}
			if o.Key != c.wantKey {
				t.Fatalf("key = %q, want %q", o.Key, c.wantKey)
			}
		})
	}
}
