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
	"yunion.io/x/onecloud/pkg/i18n"
)

var PredefinedPolicyI18nTable = i18n.Table{}
var PredefinedRoleI18nTable = i18n.Table{}

func init() {
	policies := GenerateAllPolicies()
	for _, p := range policies {
		PredefinedPolicyI18nTable.Set(p.Name, i18n.NewTableEntry().
			EN(p.Description).
			CN(p.DescriptionCN),
		)
	}

	for _, r := range RoleDefinitions {
		PredefinedRoleI18nTable.Set(r.Name, i18n.NewTableEntry().
			EN(r.Description).
			CN(r.DescriptionCN),
		)
	}
}
