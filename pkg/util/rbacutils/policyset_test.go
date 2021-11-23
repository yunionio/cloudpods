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

func TestTPolicySet_Contains(t *testing.T) {
	cases := []struct {
		name  string
		p1    TPolicySet
		p2    TPolicySet
		p1cp2 bool
		p2cp1 bool
	}{
		{
			name: "case1",
			p1: TPolicySet{
				SPolicy{
					Rules: TPolicy{
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
			},
			p2: TPolicySet{
				SPolicy{
					Rules: TPolicy{
						{
							Service:  "compute",
							Resource: "servers",
							Action:   WILD_MATCH,
							Result:   Allow,
						},
					},
				},
			},
			p1cp2: false,
			p2cp1: true,
		},
		{
			name: "case2",
			p1: TPolicySet{
				SPolicy{
					Rules: TPolicy{
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
			},
			p2: TPolicySet{
				SPolicy{
					Rules: TPolicy{
						{
							Service: WILD_MATCH,
							Result:  Allow,
						},
					},
				},
				SPolicy{
					Rules: TPolicy{
						{
							Service:  "compute",
							Resource: "servers",
							Action:   "list",
							Result:   Deny,
						},
					},
				},
			},
			p1cp2: false,
			p2cp1: true,
		},
		{
			name: "case3",
			p1: TPolicySet{
				SPolicy{
					Rules: TPolicy{
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
				},
				SPolicy{
					Rules: TPolicy{
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
			},
			p2: TPolicySet{
				SPolicy{
					Rules: TPolicy{
						{
							Service: WILD_MATCH,
							Result:  Deny,
						},
					},
				},
				SPolicy{
					Rules: TPolicy{
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
			},
			p1cp2: true,
			p2cp1: false,
		},
		{
			name: "case4",
			p1: TPolicySet{
				SPolicy{
					Rules: TPolicy{
						{
							Service: WILD_MATCH,
							Result:  Allow,
						},
					},
				},
			},
			p2: TPolicySet{
				SPolicy{
					Rules: TPolicy{
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
			},
			p1cp2: true,
			p2cp1: false,
		},
	}
	for _, c := range cases {
		got := c.p1.Contains(c.p2)
		if got != c.p1cp2 {
			t.Errorf("[%s] p1 contains p2 want %v got %v", c.name, c.p1cp2, got)
		}
		got = c.p2.Contains(c.p1)
		if got != c.p2cp1 {
			t.Errorf("[%s] p2 contains p1 want %v got %v", c.name, c.p2cp1, got)
		}
	}
}
