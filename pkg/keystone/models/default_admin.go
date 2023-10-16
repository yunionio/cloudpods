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
	"context"

	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	defaultAdminCred mcclient.TokenCredential

	defaultClient *mcclient.Client = nil
)

func GetDefaultClient() *mcclient.Client {
	if defaultClient == nil {
		defaultClient = mcclient.NewClient("", 300, options.Options.DebugClient, true, "", "")
		refreshDefaultClientServiceCatalog()
	}
	return defaultClient
}

func refreshDefaultClientServiceCatalog() {
	cata, err := EndpointManager.FetchAll()
	if err != nil {
		log.Fatalf("fail to fetch endpoints")
	}
	defaultClient.SetServiceCatalog(cata.GetKeystoneCatalogV3())
}

func GetDefaultAdminCred() mcclient.TokenCredential {
	if defaultAdminCred == nil {
		defaultAdminCred = getDefaultAdminCred()
	}
	return defaultAdminCred
}

func getDefaultAdminCred() *mcclient.SSimpleToken {
	token := mcclient.SSimpleToken{}
	usr, _ := UserManager.FetchUserExtended("", api.SystemAdminUser, api.DEFAULT_DOMAIN_ID, "")
	token.UserId = usr.Id
	token.User = usr.Name
	token.DomainId = usr.DomainId
	token.Domain = usr.DomainName
	prj, _ := ProjectManager.FetchProject("", api.SystemAdminProject, api.DEFAULT_DOMAIN_ID, "")
	token.ProjectId = prj.Id
	token.Project = prj.Name
	token.ProjectDomainId = prj.DomainId
	token.ProjectDomain = prj.GetDomain().Name
	rol, _ := RoleManager.FetchRole("", api.SystemAdminRole, api.DEFAULT_DOMAIN_ID, "")
	token.Roles = rol.Name
	token.SystemAccount = true
	token.RoleIds = rol.Id
	return &token
}

func GetDefaultClientSession(ctx context.Context, token mcclient.TokenCredential, region string) *mcclient.ClientSession {
	return GetDefaultClient().NewSession(ctx, region, "", "", token)
}
