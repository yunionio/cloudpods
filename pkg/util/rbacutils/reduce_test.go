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
	"reflect"
	"sort"
	"testing"
)

func TestReduce(t *testing.T) {

	actions := []string{"create", "delete", "get", "list", "perform", "update"}
	compute := []string{"disks", "snapshotpolicies", "snapshots"}
	image := []string{"images"}
	caseIn := make([]SRbacRule, 0, len(actions)*(len(compute)+len(image)))

	generate := func(serviceName string, resources []string, denyFunc func(resource, action string) bool) {
		for _, resource := range resources {
			for _, action := range actions {
				if denyFunc(resource, action) {
					caseIn = append(caseIn, SRbacRule{
						Service:  serviceName,
						Resource: resource,
						Action:   action,
						Result:   Deny,
					})
					continue
				}
				caseIn = append(caseIn, SRbacRule{
					Service:  serviceName,
					Resource: resource,
					Action:   action,
					Result:   Allow,
				})
			}
		}
	}

	generate("compute", compute, func(resource, action string) bool {
		if resource == "snapshotpolicies" {
			return true
		}
		if resource == "snapshots" && action == "list" {
			return true
		}
		return false
	})
	generate("image", image, func(resource, actions string) bool {
		return true
	})

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
					Action:   "perform",
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
			in: caseIn,
			want: []SRbacRule{
				{
					Service:  "compute",
					Resource: "disks",
					Action:   "",
					Result:   Allow,
				},
				{
					Service:  "compute",
					Resource: "snapshotpolicies",
					Action:   "",
					Result:   Deny,
				},
				{
					Service:  "compute",
					Resource: "snapshots",
					Action:   "*",
					Result:   Allow,
				},
				{
					Service:  "compute",
					Resource: "snapshots",
					Action:   "list",
					Result:   Deny,
				},
				{
					Service:  "image",
					Resource: "images",
					Action:   "",
					Result:   Deny,
				},
			},
		},
	}

	for name, c := range cases {
		got := reduceRules(c.in)
		for i := range c.want {
			c.want[i].Extra = []string{}
		}
		sort.Slice(got, genLess(got))
		sort.Slice(c.want, genLess(c.want))
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("[%s]: want: %s got: %s", name, c.want, got)
		}
	}
}

func genLess(rules []SRbacRule) func(i, j int) bool {
	return func(i, j int) bool {
		if rules[i].Service == rules[j].Service {
			return rules[i].Resource < rules[j].Resource
		}
		return rules[i].Service < rules[j].Service
	}
}
