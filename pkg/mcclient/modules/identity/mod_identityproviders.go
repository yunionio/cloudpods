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
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type IdentityProviderManager struct {
	modulebase.ResourceManager
}

var (
	IdentityProviders IdentityProviderManager
)

func init() {
	IdentityProviders = IdentityProviderManager{
		modules.NewIdentityV3Manager("identity_provider",
			"identity_providers",
			[]string{},
			[]string{"ID", "Name", "Driver", "Template", "auto_create_user", "Enabled", "Status", "Sync_Status", "Error_count", "Sync_Interval_Seconds", "target_domain_id"}),
	}

	modules.Register(&IdentityProviders)
}
