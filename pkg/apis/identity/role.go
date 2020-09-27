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

package identity

import (
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type RoleDetails struct {
	IdentityBaseResourceDetails
	apis.SharableResourceBaseInfo

	SRole

	UserCount    int `json:"user_count"`
	GroupCount   int `json:"group_count"`
	ProjectCount int `json:"project_count"`

	MatchPolicies []string `json:"match_policies"`

	Policies map[rbacutils.TRbacScope][]string `json:"policies"`
}
