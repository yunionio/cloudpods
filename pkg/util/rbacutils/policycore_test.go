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

func TestTPolicy_Contains(t *testing.T) {
	cases := []struct {
		name     string
		p1       TPolicy
		p2       TPolicy
		contains bool
	}{
		{
			name: "case1",
			p1: TPolicy{
				SRbacRule{
					Service: "*",
					Result:  Allow,
				},
			},
			p2: TPolicy{
				SRbacRule{
					Service: "compute",
					Result:  Allow,
				},
			},
			contains: true,
		},
		{
			name: "case2",
			p1: TPolicy{
				SRbacRule{
					Service:  "compute",
					Resource: "servers",
					Action:   "list",
					Result:   Allow,
				},
			},
			p2: TPolicy{
				SRbacRule{
					Service: "compute",
					Result:  Allow,
				},
			},
			contains: false,
		},
		{
			name: "case3",
			p1: TPolicy{
				SRbacRule{
					Service: "compute",
					Result:  Allow,
				},
				SRbacRule{
					Service:  "compute",
					Resource: "servers",
					Result:   Deny,
				},
			},
			p2: TPolicy{
				SRbacRule{
					Service: "compute",
					Result:  Allow,
				},
			},
			contains: false,
		},
		{
			name: "case4",
			p1: TPolicy{
				SRbacRule{
					Service:  "compute",
					Resource: "servers",
					Action:   "list",
					Result:   Allow,
				},
				SRbacRule{
					Service:  "compute",
					Resource: "servers",
					Action:   "get",
					Result:   Allow,
				},
				SRbacRule{
					Service:  "compute",
					Resource: "servers",
					Result:   Deny,
				},
			},
			p2: TPolicy{
				SRbacRule{
					Service:  "compute",
					Resource: "servers",
					Action:   "create",
					Result:   Allow,
				},
			},
			contains: false,
		},
	}
	for i, c := range cases {
		got := c.p1.Contains(c.p2)
		if got != c.contains {
			t.Errorf("[%d] want %v got %v", i, c.contains, got)
		}
	}
}
