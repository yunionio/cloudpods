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

import (
	"testing"

	compute_models "yunion.io/x/onecloud/pkg/compute/models"
)

func TestLoadbalancerListenerRules_OrderedEnabledList(t *testing.T) {
	set := LoadbalancerListenerRules(map[string]*LoadbalancerListenerRule{
		"empty": {
			SLoadbalancerListenerRule: &compute_models.SLoadbalancerListenerRule{
				Domain: "",
				Path:   "",
			},
		},
		"/": {
			SLoadbalancerListenerRule: &compute_models.SLoadbalancerListenerRule{
				Domain: "",
				Path:   "/",
			},
		},
		"/img": {
			SLoadbalancerListenerRule: &compute_models.SLoadbalancerListenerRule{
				Domain: "",
				Path:   "/img",
			},
		},
		"a.com": {
			SLoadbalancerListenerRule: &compute_models.SLoadbalancerListenerRule{
				Domain: "a.com",
				Path:   "",
			},
		},
		"a.com/": {
			SLoadbalancerListenerRule: &compute_models.SLoadbalancerListenerRule{
				Domain: "a.com",
				Path:   "/",
			},
		},
		"a.com/img": {
			SLoadbalancerListenerRule: &compute_models.SLoadbalancerListenerRule{
				Domain: "a.com",
				Path:   "/img",
			},
		},
		"m.a.com": {
			SLoadbalancerListenerRule: &compute_models.SLoadbalancerListenerRule{
				Domain: "m.a.com",
				Path:   "",
			},
		},
		"m.a.com/": {
			SLoadbalancerListenerRule: &compute_models.SLoadbalancerListenerRule{
				Domain: "m.a.com",
				Path:   "/",
			},
		},
		"m.a.com/img": {
			SLoadbalancerListenerRule: &compute_models.SLoadbalancerListenerRule{
				Domain: "m.a.com",
				Path:   "/img",
			},
		},
	})
	for _, rule := range set {
		rule.Status = "enabled"
	}
	rules := set.OrderedEnabledList()
	for i, rule := range rules {
		ok := true
		if i > 0 {
			rulep := rules[i-1]
			if len(rulep.Domain) < len(rule.Domain) {
				ok = false
			} else if len(rulep.Domain) == len(rule.Domain) {
				if len(rulep.Path) < len(rule.Path) {
					ok = false
				}
			}
		}
		if ok {
			t.Logf("%v: %s%s", ok, rule.Domain, rule.Path)
		} else {
			t.Errorf("%v: %s%s", ok, rule.Domain, rule.Path)
		}
	}
}
