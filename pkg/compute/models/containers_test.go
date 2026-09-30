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

package models

import "testing"

func TestParseContainerImageRepository(t *testing.T) {
	cases := []struct {
		repo           string
		wantImageName  string
		wantImageLabel string
		wantErr        bool
	}{
		{"registry.example.com/myapp:20260102150405", "registry.example.com/myapp", "20260102150405", false},
		{"registry.example.com:5000/ns/myapp:latest", "registry.example.com:5000/ns/myapp", "latest", false},
		{"myapp", "", "", true},
		{"myapp:", "", "", true},
		{":myapp", "", "", true},
	}
	for _, c := range cases {
		gotName, gotLabel, err := parseContainerImageRepository(c.repo)
		if (err != nil) != c.wantErr {
			t.Fatalf("repo %q: wantErr=%v, got err=%v", c.repo, c.wantErr, err)
		}
		if !c.wantErr && (gotName != c.wantImageName || gotLabel != c.wantImageLabel) {
			t.Fatalf("repo %q: got (%q, %q), want (%q, %q)", c.repo, gotName, gotLabel, c.wantImageName, c.wantImageLabel)
		}
	}
}
