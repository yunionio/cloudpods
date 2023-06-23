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
	orgNodeCmd := shell.NewResourceCmd(&modules.OrganizationNodes)
	orgNodeCmd.List(&identity_options.OrganizationNodeListOptions{})
	orgNodeCmd.Show(&identity_options.OrganizationNodeIdOptions{})
	orgNodeCmd.Update(&identity_options.OrganizationNodeUpdateOptions{})
	orgNodeCmd.Perform("bind", &identity_options.OrganizationNodeBindOptions{})
}
