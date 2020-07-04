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
	api "yunion.io/x/onecloud/pkg/apis/identity"
	common_policy "yunion.io/x/onecloud/pkg/cloudcommon/policy"
)

var (
	identitySystemResources = []string{
		"identity_providers",
		"domains",
		"services",
		"endpoints",
	}
	identityDomainResources = []string{
		"users",
		"groups",
		"projects",
		"roles",
		"policies",
	}
	identityUserResources = []string{
		"credentials",
	}
)

func init() {
	common_policy.RegisterSystemResources(api.SERVICE_TYPE, identitySystemResources)
	common_policy.RegisterDomainResources(api.SERVICE_TYPE, identityDomainResources)
	common_policy.RegisterUserResources(api.SERVICE_TYPE, identityUserResources)
}
