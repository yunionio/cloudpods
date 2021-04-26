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

package service

import (
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/keystone/cronjobs"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/keystone/tokens"
	"yunion.io/x/onecloud/pkg/keystone/usages"
)

const (
	API_VERSION = "v3"
)

func InitHandlers(app *appsrv.Application) {
	db.InitAllManagers()

	// add version handler with API_VERSION prefix
	app.AddDefaultHandler("GET", API_VERSION+"/version", appsrv.VersionHandler, "version")
	cronjobs.AddRefreshHandler(API_VERSION, app)

	quotas.AddQuotaHandler(&models.IdentityQuotaManager.SQuotaBaseManager, API_VERSION, app)

	usages.AddUsageHandler(API_VERSION, app)
	taskman.AddTaskHandler(API_VERSION, app)

	tokens.AddHandler(app)

	for _, manager := range []db.IModelManager{
		taskman.TaskManager,
		taskman.SubTaskManager,
		taskman.TaskObjectManager,
		models.SensitiveConfigManager,
		models.WhitelistedConfigManager,
		models.IdmappingManager,
		models.LocalUserManager,
		models.NonlocalUserManager,
		models.PasswordManager,
		models.UsergroupManager,

		models.FederatedUserManager,
		models.FederationProtocolManager,
		models.ImpliedRoleManager,
		models.UserOptionManager,
		models.IdpRemoteIdsManager,

		models.FernetKeyManager,

		models.ScopeResourceManager,

		db.SharedResourceManager,

		models.IdentityQuotaManager,
		models.IdentityUsageManager,
		models.IdentityPendingUsageManager,
	} {
		db.RegisterModelManager(manager)
	}

	for _, manager := range []db.IModelManager{
		db.OpsLog,
		db.Metadata,

		models.UserManager,
		models.GroupManager,
		models.ProjectManager,
		models.DomainManager,
		models.RoleManager,
		models.ServiceManager,
		models.RegionManager,
		models.EndpointManager,
		models.AssignmentManager,
		models.PolicyManager,
		models.CredentialManager,
		models.IdentityProviderManager,
		models.ServiceCertificateManager,
		models.RolePolicyManager,
	} {
		db.RegisterModelManager(manager)
		handler := db.NewModelHandler(manager)
		dispatcher.AddModelDispatcher(API_VERSION, app, handler)
	}

	models.AddAdhocHandlers(API_VERSION, app)
}
