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

package cloudid

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/cloudid"
	"yunion.io/x/onecloud/pkg/mcclient/options/cloudid"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Cloudgroups).WithKeyword("cloud-group")
	cmd.List(&cloudid.CloudgroupListOptions{})
	cmd.Create(&cloudid.CloudgroupCreateOptions{})
	cmd.Show(&cloudid.CloudgroupIdOptions{})
	cmd.Delete(&cloudid.CloudgroupIdOptions{})
	cmd.Perform("syncstatus", &cloudid.CloudgroupIdOptions{})
	cmd.Perform("attach-policy", &cloudid.CloudgroupPolicyOptions{})
	cmd.Perform("detach-policy", &cloudid.CloudgroupPolicyOptions{})
	cmd.Perform("add-user", &cloudid.CloudgroupUserOptions{})
	cmd.Perform("remove-user", &cloudid.CloudgroupUserOptions{})
	cmd.Perform("set-policies", &cloudid.CloudgroupPolicyOptions{})
	cmd.Perform("set-users", &cloudid.CloudgroupUserOptions{})
	cmd.Perform("public", &cloudid.CloudgroupPublicOptions{})
	cmd.Perform("private", &cloudid.CloudgroupIdOptions{})
	cmd.Get("saml", &cloudid.CloudgroupIdOptions{})
}
