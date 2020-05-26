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

package regiondrivers

import (
	"fmt"
	"testing"

	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type TestData struct {
	Name        string
	LocalRules  secrules.SecurityRuleSet
	RemoteRules []cloudprovider.SecurityRule
	Common      []cloudprovider.SecurityRule
	InAdds      []cloudprovider.SecurityRule
	OutAdds     []cloudprovider.SecurityRule
	InDels      []cloudprovider.SecurityRule
	OutDels     []cloudprovider.SecurityRule
}

var check = func(t *testing.T, name string, ret, expect []cloudprovider.SecurityRule) {
	var show = func(info string, rules []cloudprovider.SecurityRule) {
		t.Logf("%s: %d\n", info, len(rules))
		for _, r := range rules {
			t.Logf("Name: %s priority: %d %s\n", r.Name, r.Priority, r.String())
		}
	}
	if len(ret) != len(expect) {
		show(fmt.Sprintf("%s rule", name), ret)
		show(fmt.Sprintf("%s expect", name), expect)
		t.Fatalf("invalid rules for %s current is %d expect %d", name, len(ret), len(expect))
	}
	for i := range ret {
		if ret[i].Name != expect[i].Name {
			show(fmt.Sprintf("%s rule", name), ret)
			show(fmt.Sprintf("%s expect", name), expect)
			t.Fatalf("invalid index(%d) %s rule name %s expect %s", i, name, ret[i].Name, expect[i].Name)
		}
		if ret[i].Priority != expect[i].Priority {
			show(fmt.Sprintf("%s rule", name), ret)
			show(fmt.Sprintf("%s expect", name), expect)
			t.Fatalf("invalid index(%d) %s rule priority %d expect %d", i, name, ret[i].Priority, expect[i].Priority)
		}
		if ret[i].String() != expect[i].String() {
			show(fmt.Sprintf("%s rule", name), ret)
			show(fmt.Sprintf("%s expect", name), expect)
			t.Fatalf("invalid index(%d) %s rules %s expect %s", i, name, ret[i].String(), expect[i].String())
		}
	}
}
