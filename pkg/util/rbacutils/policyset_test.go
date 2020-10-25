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

func TestTPolicySet_Violate(t *testing.T) {
	cases := []struct {
		name string
		p1   TPolicySet
		p2   TPolicySet
		want bool
	}{
		{
			name: "case1",
			p1: TPolicySet{
				{
					{
						Service:  "compute",
						Resource: "servers",
						Action:   "list",
						Result:   Deny,
					},
					{
						Service:  "compute",
						Resource: "servers",
						Action:   WILD_MATCH,
						Result:   Allow,
					},
				},
			},
			p2: TPolicySet{
				{
					{
						Service:  "compute",
						Resource: "servers",
						Action:   WILD_MATCH,
						Result:   Allow,
					},
				},
			},
			want: true,
		},
		{
			name: "case2",
			p1: TPolicySet{
				{
					{
						Service:  "comptue",
						Resource: "servers",
						Action:   "list",
						Result:   Deny,
					},
					{
						Service:  "compute",
						Resource: "servers",
						Action:   WILD_MATCH,
						Result:   Allow,
					},
				},
			},
			p2: TPolicySet{
				{
					{
						Service: WILD_MATCH,
						Result:  Allow,
					},
				},
				{
					{
						Service:  "compute",
						Resource: "servers",
						Action:   "list",
						Result:   Deny,
					},
				},
			},
			want: true,
		},
		{
			name: "case3",
			p1: TPolicySet{
				{
					{
						Service: WILD_MATCH,
						Result:  Allow,
					},
					{
						Service:  "compute",
						Resource: "servers",
						Action:   "create",
						Result:   Deny,
					},
				},
				{
					{
						Service:  "comptue",
						Resource: "servers",
						Action:   "list",
						Result:   Deny,
					},
					{
						Service:  "compute",
						Resource: "servers",
						Action:   WILD_MATCH,
						Result:   Allow,
					},
				},
			},
			p2: TPolicySet{
				{
					{
						Service: WILD_MATCH,
						Result:  Deny,
					},
				},
				{
					{
						Service:  "comptue",
						Resource: "servers",
						Action:   WILD_MATCH,
						Result:   Deny,
					},
					{
						Service:  "compute",
						Resource: "servers",
						Action:   "get",
						Result:   Allow,
					},
				},
			},
			want: false,
		},
		{
			name: "case4",
			p2: TPolicySet{
				{
					{
						Service: WILD_MATCH,
						Result:  Allow,
					},
				},
			},
			p1: TPolicySet{
				{
					{
						Service: WILD_MATCH,
						Result:  Allow,
					},
					{
						Service:  "compute",
						Resource: "servers",
						Action:   "list",
						Result:   Deny,
					},
				},
			},
			want: true,
		},
	}
	for _, c := range cases {
		got := c.p1.ViolatedBy(c.p2)
		if got != c.want {
			t.Errorf("[%s] want %v got %v", c.name, c.want, got)
		}
	}
}
