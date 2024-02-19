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

package compute

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.LoadbalancerAcls).WithKeyword("lbacl")
	cmd.List(&options.LoadbalancerAclListOptions{})
	cmd.Show(&options.LoadbalancerAclIdOptions{})
	cmd.Create(&options.LoadbalancerAclCreateOptions{})
	cmd.Delete(&options.LoadbalancerAclIdOptions{})
	cmd.Update(&options.LoadbalancerAclUpdateOptions{})
	cmd.Perform("purge", &options.LoadbalancerAclIdOptions{})
	cmd.Perform("syncstatus", &options.LoadbalancerAclIdOptions{})
	cmd.Perform("private", &options.LoadbalancerAclIdOptions{})
	cmd.Perform("public", &options.LoadbalancerAclPublicOptions{})
	cmd.Perform("patch", &options.LoadbalancerAclActionPatchOptions{})
}
