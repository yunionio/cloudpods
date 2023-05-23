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
	cmd := shell.NewResourceCmd(&modules.Cloudpolicies).WithKeyword("cloud-policy")
	cmd.Create(&cloudid.CloudpolicyCreateOption{})
	cmd.List(&cloudid.CloudpolicyListOptions{})
	cmd.Show(&cloudid.CloudpolicyIdOptions{})
	cmd.Delete(&cloudid.CloudpolicyIdOptions{})
	cmd.Update(&cloudid.CloudpolicyUpdateOption{})
	cmd.Perform("syncstauts", &cloudid.CloudpolicyIdOptions{})
	cmd.Perform("cache", &cloudid.CloudpolicyCacheOption{})
	cmd.Perform("lock", &cloudid.CloudpolicyIdOptions{})
	cmd.Perform("unlock", &cloudid.CloudpolicyIdOptions{})
	cmd.Perform("assign-group", &cloudid.CloudpolicyGroupOptions{})
	cmd.Perform("revoke-group", &cloudid.CloudpolicyGroupOptions{})
}
