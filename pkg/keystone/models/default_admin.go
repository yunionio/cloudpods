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

package models

import (
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	defaultAdminCred mcclient.TokenCredential
)

func GetDefaultAdminCred() mcclient.TokenCredential {
	if defaultAdminCred == nil {
		defaultAdminCred = getDefaultAdminCred()
	}
	return defaultAdminCred
}

func getDefaultAdminCred() mcclient.TokenCredential {
	token := mcclient.SSimpleToken{}
	usr, _ := UserManager.FetchUserExtended("", options.Options.AdminUserName, options.Options.AdminUserDomainId, "")
	token.UserId = usr.Id
	token.User = usr.Name
	token.DomainId = usr.DomainId
	token.Domain = usr.DomainName
	prj, _ := ProjectManager.FetchProject("", options.Options.AdminProjectName, options.Options.AdminProjectDomainId, "")
	token.ProjectId = prj.Id
	token.Project = prj.Name
	token.ProjectDomainId = prj.DomainId
	token.ProjectDomain = prj.GetDomain().Name
	token.Roles = "admin"
	return &token
}
