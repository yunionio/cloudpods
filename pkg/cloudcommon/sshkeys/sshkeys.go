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

package sshkeys

import "yunion.io/x/jsonutils"

type SSHKeys struct {
	PublicKey        string
	DeletePublicKey  string
	AdminPublicKey   string
	ProjectPublicKey string
}

func GetKeys(data jsonutils.JSONObject) *SSHKeys {
	var ret = new(SSHKeys)
	ret.PublicKey, _ = data.GetString("public_key")
	ret.DeletePublicKey, _ = data.GetString("delete_public_key")
	ret.AdminPublicKey, _ = data.GetString("admin_public_key")
	ret.ProjectPublicKey, _ = data.GetString("project_public_key")
	return ret
}
