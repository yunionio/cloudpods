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

package guestdrivers

import (
	"yunion.io/x/onecloud/pkg/util/ansible"
	"yunion.io/x/onecloud/pkg/util/cloudinit"
)

func generateUserData(adminPublicKey, projectPublicKey, oUserData string) string {
	var oCloudConfig *cloudinit.SCloudConfig = nil

	if len(oUserData) > 0 {
		oCloudConfig, _ = cloudinit.ParseUserDataBase64(oUserData)
	}

	ansibleUser := cloudinit.NewUser(ansible.PUBLIC_CLOUD_ANSIBLE_USER)
	ansibleUser.SshKey(adminPublicKey).SshKey(projectPublicKey).SudoPolicy(cloudinit.USER_SUDO_NOPASSWD)

	cloudConfig := cloudinit.SCloudConfig{
		DisableRoot: 0,
		SshPwauth:   1,

		Users: []cloudinit.SUser{
			ansibleUser,
		},
	}

	if oCloudConfig != nil {
		cloudConfig.Merge(oCloudConfig)
	}

	return cloudConfig.UserDataBase64()
}
