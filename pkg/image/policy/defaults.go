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

package policy

import (
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	common_policy "yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

const (
	PolicyActionPerform = common_policy.PolicyActionPerform
	PolicyActionGet     = common_policy.PolicyActionGet
	PolicyActionList    = common_policy.PolicyActionList
)

var (
	predefinedDefaultPolicies = []rbacutils.SRbacPolicy{
		{
			Auth:  true,
			Scope: rbacscope.ScopeProject,
			Rules: []rbacutils.SRbacRule{
				{
					Service:  api.SERVICE_TYPE,
					Resource: "image_quotas",
					Action:   PolicyActionGet,
					Result:   rbacutils.Allow,
				},
				{
					Service:  api.SERVICE_TYPE,
					Resource: "image_quotas",
					Action:   PolicyActionList,
					Result:   rbacutils.Allow,
				},
			},
		},
		{
			// for anonymous update torrent status
			Auth:  false,
			Scope: rbacscope.ScopeSystem,
			Rules: []rbacutils.SRbacRule{
				{
					Service:  api.SERVICE_TYPE,
					Resource: "images",
					Action:   PolicyActionPerform,
					Extra:    []string{"update-torrent-status"},
					Result:   rbacutils.Allow,
				},
			},
		},
	}
)

func Init() {
	if consts.IsEnableDefaultPolicy() {
		common_policy.AppendDefaultPolicies(predefinedDefaultPolicies)
	}
}
