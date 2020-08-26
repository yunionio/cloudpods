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
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Cloudusers).WithKeyword("cloud-user")
	cmd.List(&options.ClouduserListOptions{})
	cmd.Create(&options.ClouduserCreateOptions{})
	cmd.Show(&options.ClouduserIdOption{})
	cmd.Custom(shell.CustomActionGet, "login-info", &options.ClouduserIdOption{})
	cmd.Delete(&options.ClouduserIdOption{})
	cmd.Perform("sync", &options.ClouduserSyncOptions{})
	cmd.Perform("syncstatus", &options.ClouduserIdOption{})
	cmd.Perform("attach-policy", &options.ClouduserPolicyOptions{})
	cmd.Perform("detach-policy", &options.ClouduserPolicyOptions{})
	cmd.Perform("change-owner", &options.ClouduserChangeOwnerOptions{})
	cmd.Perform("cloud-user-join-group", &options.ClouduserGroupOptions{})
	cmd.Perform("cloud-user-leave-group", &options.ClouduserGroupOptions{})
}
