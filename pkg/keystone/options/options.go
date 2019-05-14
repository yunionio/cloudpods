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

package options

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/options"
)

type SKeystoneOptions struct {
	options.BaseOptions

	options.DBOptions

	AdminPort int `default:"35357" help:"listening port for admin API(deprecated)"`

	TokenExpirationSeconds  int    `default:"86400" help:"token expiration seconds" token:"expiration"`
	TokenKeyRepository      string `help:"fernet key repo directory" token:"key_repository" default:"/etc/yunion/keystone/fernet-keys"`
	CredentialKeyRepository string `help:"fernet key repo directory for credential" token:"credential_key_repository"`

	AdminUserName        string `help:"Administrative user name" default:"sysadmin"`
	AdminUserDomainId    string `help:"Domain id of administrative user" default:"default"`
	AdminProjectName     string `help:"Administrative project name" default:"system"`
	AdminProjectDomainId string `help:"Domain id of administrative project" default:"default"`
	AdminRoleName        string `help:"Administrative user role" default:"admin"`
	AdminRoleDomainId    string `help:"Domain id of administrative role" default:"default"`

	BootstrapAdminUserPassword string `help:"bootstreap sysadmin user password" default:"sysadmin"`
}

var (
	Options SKeystoneOptions
)
