package models

import (
	"testing"

	"yunion.io/x/onecloud/pkg/mcclient/models"
)

func TestLoadbalancerListenerRules_OrderedEnabledList(t *testing.T) {
	set := LoadbalancerListenerRules(map[string]*LoadbalancerListenerRule{
		"empty": {
			LoadbalancerListenerRule: &models.LoadbalancerListenerRule{
				Domain: "",
				Path:   "",
			},
		},
		"/": {
			LoadbalancerListenerRule: &models.LoadbalancerListenerRule{
				Domain: "",
				Path:   "/",
			},
		},
		"/img": {
			LoadbalancerListenerRule: &models.LoadbalancerListenerRule{
				Domain: "",
				Path:   "/img",
			},
		},
		"a.com": {
			LoadbalancerListenerRule: &models.LoadbalancerListenerRule{
				Domain: "a.com",
				Path:   "",
			},
		},
		"a.com/": {
			LoadbalancerListenerRule: &models.LoadbalancerListenerRule{
				Domain: "a.com",
				Path:   "/",
			},
		},
		"a.com/img": {
			LoadbalancerListenerRule: &models.LoadbalancerListenerRule{
				Domain: "a.com",
				Path:   "/img",
			},
		},
		"m.a.com": {
			LoadbalancerListenerRule: &models.LoadbalancerListenerRule{
				Domain: "m.a.com",
				Path:   "",
			},
		},
		"m.a.com/": {
			LoadbalancerListenerRule: &models.LoadbalancerListenerRule{
				Domain: "m.a.com",
				Path:   "/",
			},
		},
		"m.a.com/img": {
			LoadbalancerListenerRule: &models.LoadbalancerListenerRule{
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
