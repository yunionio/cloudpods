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

package client

import (
	"testing"
)

func TestParseWWWAuthenticate(t *testing.T) {
	cases := []struct {
		header string
		scheme string
		realm  string
		svc    string
	}{
		{
			header: `Bearer realm="https://auth.docker.io/token",service="registry.docker.io"`,
			scheme: "bearer",
			realm:  "https://auth.docker.io/token",
			svc:    "registry.docker.io",
		},
		{
			header: `Basic realm="Registry Realm"`,
			scheme: "basic",
			realm:  "Registry Realm",
		},
		{
			header: `Bearer realm="https://registry.cn-beijing.aliyuncs.com/auth",service="registry.aliyuncs.com:cn-beijing:26842",scope="repository:yunion/foo:pull"`,
			scheme: "bearer",
			realm:  "https://registry.cn-beijing.aliyuncs.com/auth",
			svc:    "registry.aliyuncs.com:cn-beijing:26842",
		},
	}
	for _, c := range cases {
		scheme, params, err := parseWWWAuthenticate(c.header)
		if err != nil {
			t.Fatalf("parse %q: %v", c.header, err)
		}
		if scheme != c.scheme {
			t.Errorf("scheme want %q got %q", c.scheme, scheme)
		}
		if params["realm"] != c.realm {
			t.Errorf("realm want %q got %q", c.realm, params["realm"])
		}
		if c.svc != "" && params["service"] != c.svc {
			t.Errorf("service want %q got %q", c.svc, params["service"])
		}
	}
}
