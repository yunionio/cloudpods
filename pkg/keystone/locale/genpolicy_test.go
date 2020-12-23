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

package locale

import (
	"testing"

	"golang.org/x/text/language"

	"yunion.io/x/jsonutils"
)

func TestGenerateAllPolicies(t *testing.T) {
	policies := GenerateAllPolicies()
	for _, p := range policies {
		t.Logf("name %s description %s scope %s policy %s", p.Name, p.Description, p.Scope, p.Policy)
	}
	t.Logf("total: %d", len(policies))
}

func TestPolicyDescriptions(t *testing.T) {
	out := jsonutils.NewDict()
	policies := GenerateAllPolicies()
	for _, p := range policies {
		item := jsonutils.NewDict()
		item.Add(jsonutils.NewString(p.Description), language.English.String())
		item.Add(jsonutils.NewString(p.DescriptionCN), language.Chinese.String())
		out.Add(item, p.Name)
	}
	t.Logf("%s", out.PrettyString())
}

func TestRoleDescriptions(t *testing.T) {
	out := jsonutils.NewDict()
	for _, r := range RoleDefinitions {
		item := jsonutils.NewDict()
		item.Add(jsonutils.NewString(r.Description), language.English.String())
		item.Add(jsonutils.NewString(r.DescriptionCN), language.Chinese.String())
		out.Add(item, r.Name)
	}
	t.Logf("%s", out.PrettyString())
}
