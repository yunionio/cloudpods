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
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	identity_options "yunion.io/x/onecloud/pkg/mcclient/options/identity"
)

func init() {
	orgCmd := shell.NewResourceCmd(&modules.Organizations)
	orgCmd.List(&identity_options.OrganizationListOptions{})
	orgCmd.Create(&identity_options.OrganizationCreateOptions{})
	orgCmd.Show(&identity_options.OrganizationIdOptions{})
	orgCmd.Perform("sync", &identity_options.OrganizationSyncOptions{})
	orgCmd.Perform("enable", &identity_options.OrganizationIdOptions{})
	orgCmd.Perform("disable", &identity_options.OrganizationIdOptions{})
	orgCmd.Delete(&identity_options.OrganizationIdOptions{})
	orgCmd.Perform("add-level", &identity_options.OrganizationAddLevelOptions{})
	orgCmd.Perform("add-node", &identity_options.OrganizationAddNodeOptions{})
	orgCmd.Perform("clean", &identity_options.OrganizationIdOptions{})
}
