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

package firewalld

import (
	"testing"
)

func TestDirect(t *testing.T) {
	d := NewDirect(
		NewIP4Rule(0, "nat", "PREROUTING", "-j DNAT --to-destination 10.168.222.99:1099"),
		NewIP4Rule(0, "nat", "POSTROUTING", "-j SNAT --to-source :80"),
		NewIP4Rule(100, "nat", "POSTROUTING", "-j wg0 -j MASQUERADE"),
	)
	ruleWant := []string{
		`<rule priority="0" table="nat" ipv="ipv4" chain="PREROUTING">-j DNAT --to-destination 10.168.222.99:1099</rule>`,
		`<rule priority="0" table="nat" ipv="ipv4" chain="POSTROUTING">-j SNAT --to-source :80</rule>`,
		`<rule priority="100" table="nat" ipv="ipv4" chain="POSTROUTING">-j wg0 -j MASQUERADE</rule>`,
	}
	for i, r := range d.Rules {
		got := r.String()
		want := ruleWant[i]
		if got != want {
			t.Errorf("rule %d\n  got\n    %s\n  want\n    %s", i, got, want)
		}
	}
	want := `<direct>
  <rule priority="0" table="nat" ipv="ipv4" chain="PREROUTING">-j DNAT --to-destination 10.168.222.99:1099</rule>
  <rule priority="0" table="nat" ipv="ipv4" chain="POSTROUTING">-j SNAT --to-source :80</rule>
  <rule priority="100" table="nat" ipv="ipv4" chain="POSTROUTING">-j wg0 -j MASQUERADE</rule>
</direct>`
	got := d.String()
	if got != want {
		t.Errorf("direct:\n  got:\n    %s\n  want:\n    %s", got, want)
	}
}
