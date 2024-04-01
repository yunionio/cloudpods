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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/cloudid"
	"yunion.io/x/onecloud/pkg/mcclient/options/cloudid"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Cloudusers).WithKeyword("cloud-user")
	cmd.List(&cloudid.ClouduserListOptions{})
	cmd.Create(&cloudid.ClouduserCreateOptions{})
	cmd.Show(&cloudid.ClouduserIdOption{})
	cmd.Custom(shell.CustomActionGet, "login-info", &cloudid.ClouduserIdOption{})
	cmd.Delete(&cloudid.ClouduserIdOption{})
	cmd.Perform("syncstatus", &cloudid.ClouduserIdOption{})
	cmd.Perform("attach-policy", &cloudid.ClouduserPolicyOptions{})
	cmd.Perform("detach-policy", &cloudid.ClouduserPolicyOptions{})
	cmd.Perform("change-owner", &cloudid.ClouduserChangeOwnerOptions{})
	cmd.Perform("reset-password", &cloudid.ClouduserResetPasswordOptions{})
	cmd.Perform("join-group", &cloudid.ClouduserGroupOptions{})
	cmd.Perform("leave-group", &cloudid.ClouduserGroupOptions{})
	cmd.GetWithCustomShow("access-keys", func(result jsonutils.JSONObject) {
		ret := modulebase.JSON2ListResult(result)
		shell.PrintList(ret, nil)
	}, &cloudid.ClouduserListAccessKeyInput{})
	cmd.Perform("delete-access-key", &cloudid.ClouduserDeleteAccessKeyInput{})
	cmd.Perform("create-access-key", &cloudid.ClouduserCreateAccessKeyInput{})
}
