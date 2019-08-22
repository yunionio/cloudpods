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

import (
	"testing"
)

func TestReduce(t *testing.T) {
	cases := map[string]struct {
		in   []SRbacRule
		want []SRbacRule
	}{
		"merge1": {
			in: []SRbacRule{
				{
					Service:  "compute",
					Resource: "servers",
					Action:   "list",
					Result:   Allow,
				},
				{
					Service:  "compute",
					Resource: "servers",
					Action:   "get",
					Result:   Allow,
				},
				{
					Service:  "compute",
					Resource: "servers",
					Action:   "delete",
					Result:   Allow,
				},
			},
			want: []SRbacRule{
				{
					Service:  "compute",
					Resource: "servers",
					Result:   Allow,
				},
			},
		},
		"merge2": {
			in: []SRbacRule{
				{
					Service:  "compute",
					Resource: "servers",
					Action:   "get",
					Result:   Allow,
				},
				{
					Service:  "compute",
					Resource: "servers",
					Action:   "create",
					Result:   Allow,
				},
				{
					Service:  "compute",
					Resource: "networks",
					Action:   "get",
					Result:   Allow,
				},
				{
					Service:  "compute",
					Resource: "networks",
					Action:   "create",
					Result:   Deny,
				},
			},
			want: []SRbacRule{
				{
					Service:  "compute",
					Resource: "servers",
					Action:   "",
					Result:   Allow,
				},
				{
					Service:  "compute",
					Resource: "networks",
					Action:   "get",
					Result:   Allow,
				},
				{
					Service:  "compute",
					Resource: "networks",
					Action:   "create",
					Result:   Deny,
				},
			},
		},
		"merge3": {
			in: []SRbacRule{
				{
					Service:  "compute",
					Resource: "servers",
					Action:   "get",
					Result:   Allow,
				},
				{
					Service:  "compute",
					Resource: "servers",
					Action:   "get",
					Extra:    []string{"vnc"},
					Result:   Deny,
				},
				{
					Service:  "compute",
					Resource: "servers",
					Action:   "create",
					Result:   Allow,
				},
				{
					Service:  "compute",
					Resource: "servers",
					Action:   "update",
					Result:   Allow,
				},
				{
					Service:  "compute",
					Resource: "servers",
					Action:   "delete",
					Result:   Deny,
				},
			},
			want: []SRbacRule{
				{
					Service:  "compute",
					Resource: "servers",
					Result:   Allow,
				},
				{
					Service:  "compute",
					Resource: "servers",
					Action:   "get",
					Extra:    []string{"vnc"},
					Result:   Deny,
				},
				{
					Service:  "compute",
					Resource: "servers",
					Action:   "delete",
					Result:   Deny,
				},
			},
		},
	}
	for name, c := range cases {
		got := reduceRules(c.in)
		t.Logf("[%s]: want: %s got: %s", name, c.want, got)
	}
}
