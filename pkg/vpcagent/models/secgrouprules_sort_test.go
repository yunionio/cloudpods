package models

import (
	"reflect"
	"testing"

	"yunion.io/x/pkg/util/secrules"

	compute_models "yunion.io/x/onecloud/pkg/compute/models"
)

func newTestSecgroupRule(id string, priority int) *SecurityGroupRule {
	return &SecurityGroupRule{
		SSecurityGroupRule: compute_models.SSecurityGroupRule{
			Id:        id,
			Priority:  priority,
			Direction: string(secrules.SecurityRuleIngress),
			Action:    string(secrules.SecurityRuleAllow),
		},
	}
}

func newTestSecgroup(rules ...*SecurityGroupRule) *SecurityGroup {
	sg := &SecurityGroup{
		SecurityGroupRules: SecurityGroupRules{},
	}
	for _, rule := range rules {
		sg.SecurityGroupRules[rule.Id] = rule
	}
	return sg
}

// the rules go out to OVN in this order, so they have to come back sorted:
// the deny-all and ARP prefixes first, then the guest security groups (base
// 100) and the admin security group (base 1000) last.  The rules of a security
// group come out of a map, so without the sort their order is arbitrary.
func TestOrderedSecurityGroupRulesAreSortedByPriority(t *testing.T) {
	guest := &Guest{}
	guest.SecurityGroups = SecurityGroups{
		"sg1": newTestSecgroup(
			newTestSecgroupRule("r40", 40),
			newTestSecgroupRule("r10", 10),
			newTestSecgroupRule("r30", 30),
			newTestSecgroupRule("r20", 20),
			newTestSecgroupRule("r60", 60),
			newTestSecgroupRule("r50", 50),
		),
	}
	guest.AdminSecurityGroup = newTestSecgroup(newTestSecgroupRule("admin10", 10))

	rules := guest.OrderedSecurityGroupRules()
	if len(rules) != 2+6+1 {
		t.Fatalf("got %d rules, want %d", len(rules), 2+6+1)
	}
	if rules[0].Priority != 1 || rules[0].Action != string(secrules.SecurityRuleDeny) {
		t.Fatalf("the first rule is not the deny-all prefix: %+v", rules[0].SSecurityGroupRule)
	}
	if rules[1].Priority != 2 || rules[1].Protocol != "arp" {
		t.Fatalf("the second rule is not the ARP prefix: %+v", rules[1].SSecurityGroupRule)
	}
	for i := 1; i < len(rules); i++ {
		if rules[i-1].Priority > rules[i].Priority {
			t.Fatalf("rule %s (priority %d) comes before rule %s (priority %d)",
				rules[i-1].Id, rules[i-1].Priority, rules[i].Id, rules[i].Priority)
		}
	}
	// the admin security group rules are the last ones, on the top priorities
	if rules[len(rules)-1].Id != "admin10" || rules[len(rules)-1].Priority != 1010 {
		t.Fatalf("the last rule is %s (priority %d), want the admin rule 1010",
			rules[len(rules)-1].Id, rules[len(rules)-1].Priority)
	}
}

// rules that land on the same priority are ordered by id, so that the list
// built from a map comes out the same way every time
func TestOrderedSecurityGroupRulesAreStableForEqualPriority(t *testing.T) {
	guest := &Guest{}
	guest.SecurityGroups = SecurityGroups{
		"sg1": newTestSecgroup(
			newTestSecgroupRule("r-b", 10),
			newTestSecgroupRule("r-a", 10),
			newTestSecgroupRule("r-d", 10),
			newTestSecgroupRule("r-c", 10),
		),
	}

	var first []string
	for i := 0; i < 20; i++ {
		got := make([]string, 0)
		for _, rule := range guest.OrderedSecurityGroupRules() {
			got = append(got, rule.Id)
		}
		if first == nil {
			first = got
			continue
		}
		if !reflect.DeepEqual(got, first) {
			t.Fatalf("the order of equal priority rules is not stable:\n got: %v\nwant: %v", got, first)
		}
	}

	// the two prefix rules have no id, and come first on their priorities
	want := []string{"", "", "r-a", "r-b", "r-c", "r-d"}
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("got: %v\nwant: %v", first, want)
	}
}
