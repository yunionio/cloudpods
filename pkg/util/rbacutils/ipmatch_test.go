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

package rbacutils

import "testing"

func TestMatchIPStrings(t *testing.T) {
	cases := []struct {
		prefixes string
		ip       string
		want     bool
	}{
		{
			prefixes: "",
			ip:       "127.0.0.1",
			want:     true,
		},
		{
			prefixes: "10.0.0.0/8",
			ip:       "10.8.0.1",
			want:     true,
		},
		{
			prefixes: "10.0.0.0/8,192.168.0.0/16",
			ip:       "172.16.0.23",
			want:     false,
		},
		{
			prefixes: "10.0.0.0/8,192.168.0.0/16",
			ip:       "10.16.0.23",
			want:     true,
		},
		{
			prefixes: "10.0.0.0/8,192.168.0.0/16",
			ip:       "192.168.0.23",
			want:     true,
		},
	}
	for _, c := range cases {
		got := MatchIPStrings(c.prefixes, c.ip)
		if got != c.want {
			t.Errorf("prefix %s ip %s got %v want %v", c.prefixes, c.ip, got, c.want)
		}
	}
}
