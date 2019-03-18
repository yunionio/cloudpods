package secrules

import "testing"

func TestRules(t *testing.T) {
	rules := []string{"in:allow 192.168.0.1/32 tcp 80", "out:deny any", "in:allow any", "out:allow udp 3232-3000", "in:allow tcp"}
	for _, r := range rules {
		rule, err := ParseSecurityRule(r)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(rule)
	}

	ruleSet := [][]string{
		[]string{"in:allow any"},
		[]string{"in:allow any", "in:deny tcp 25", "in:deny tcp 80"},
		[]string{"in:allow tcp 80", "in:allow udp"},
		[]string{"in:allow any", "in:allow tcp 80"},
		[]string{"in:allow tcp 1000-1999", "in:allow tcp 2000", "in:allow tcp 2200"},
		[]string{"in:allow tcp 1000-1999", "in:allow tcp 2000", "in:allow tcp 2200",
			"in:deny tcp 500-1500", "in:deny tcp 2000-3000", "in:deny tcp 1888"},
		[]string{"in:allow 192.168.0.0/16 tcp", "in:allow tcp 80",
			"out:deny 192.168.0.0/16 tcp 80", "out:deny tcp 80", "in:allow any",
			"in:allow 192.168.0.0/24 tcp", "out:deny 192.168.0.0/24 udp"},
	}

	for _, rs := range ruleSet {
		srs := SecurityGroupRuleSet{}
		for _, r := range rs {
			rule, err := ParseSecurityRule(r)
			if err != nil {
				t.Fatalf("parse rule %s error: %v", r, err)
			}
			srs.AddRule(*rule)
		}
		t.Log(srs.String())
	}
}
