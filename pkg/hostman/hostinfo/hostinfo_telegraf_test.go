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

package hostinfo

import (
	"fmt"
	"testing"
)

func TestResolveKunlunxinXpuLibDir(t *testing.T) {
	cases := []struct {
		name     string
		xreHome  string
		existing []string
		want     string
	}{
		{
			name:    "default home when empty",
			xreHome: "",
			want:    "/usr/local/xpu/so",
		},
		{
			name:     "prefers so",
			xreHome:  "/opt/xpu",
			existing: []string{"/opt/xpu/so", "/opt/xpu/lib64", "/opt/xpu/lib"},
			want:     "/opt/xpu/so",
		},
		{
			name:     "falls back to lib64",
			xreHome:  "/opt/xpu",
			existing: []string{"/opt/xpu/lib64", "/opt/xpu/lib"},
			want:     "/opt/xpu/lib64",
		},
		{
			name:     "falls back to lib",
			xreHome:  "/opt/xpu",
			existing: []string{"/opt/xpu/lib"},
			want:     "/opt/xpu/lib",
		},
		{
			name:    "defaults to so when nothing exists",
			xreHome: "/opt/xpu",
			want:    "/opt/xpu/so",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			exists := func(p string) bool {
				for _, e := range c.existing {
					if e == p {
						return true
					}
				}
				return false
			}
			if got := resolveKunlunxinXpuLibDir(c.xreHome, exists); got != c.want {
				t.Errorf("resolveKunlunxinXpuLibDir(%q) = %q, expected %q", c.xreHome, got, c.want)
			}
		})
	}

	if got := resolveKunlunxinXpuLibDir("", nil); got != "/usr/local/xpu/so" {
		t.Errorf("resolveKunlunxinXpuLibDir with nil exists = %q, expected %q", got, "/usr/local/xpu/so")
	}
}

func TestResolveSmiBinPathWithReadlink(t *testing.T) {
	cases := []struct {
		name     string
		binPath  string
		resolved string
		readErr  error
		want     string
	}{
		{
			name:     "resolve symlink",
			binPath:  "/usr/local/bin/ixsmi",
			resolved: "/usr/local/corex-4.3.0/bin/ixsmi",
			want:     "/usr/local/corex-4.3.0/bin/ixsmi",
		},
		{
			name:    "fallback on error",
			binPath: "/usr/local/bin/ixsmi",
			readErr: fmt.Errorf("no such file"),
			want:    "/usr/local/bin/ixsmi",
		},
		{
			name:     "fallback on empty result",
			binPath:  "/usr/local/bin/ixsmi",
			resolved: "",
			want:     "/usr/local/bin/ixsmi",
		},
		{
			name:    "empty input",
			binPath: "",
			want:    "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			readlink := func(string) (string, error) {
				return c.resolved, c.readErr
			}
			if got := resolveSmiBinPathWithReadlink(c.binPath, readlink); got != c.want {
				t.Errorf("resolveSmiBinPathWithReadlink(%q) = %q, expected %q", c.binPath, got, c.want)
			}
		})
	}
}
