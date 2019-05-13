package service

import (
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/keystone/tokens"
	"yunion.io/x/onecloud/pkg/keystone/usages"
)

const (
	API_VERSION = "v3"
)

func initHandlers(app *appsrv.Application) {
	db.InitAllManagers()

	// quotas.AddQuotaHandler(models.QuotaManager, API_VERSION, app)
	usages.AddUsageHandler(API_VERSION, app)
	// taskman.AddTaskHandler(API_VERSION, app)

	tokens.AddHandler(app)

	for _, manager := range []db.IModelManager{
		// taskman.TaskManager,
		// taskman.SubTaskManager,
		// taskman.TaskObjectManager,
		db.Metadata,
		models.SensitiveConfigManager,
		models.WhitelistedConfigManager,
		models.IdmappingManager,
		models.LocalUserManager,
		models.NonlocalUserManager,
		models.PasswordManager,
		models.UsergroupManager,

		models.FederatedUserManager,
		models.FederationProtocolManager,
		models.IdentityProviderManager,
		models.ImpliedRoleManager,
		models.UserOptionManager,
		models.IdpRemoteIdsManager,
	} {
		db.RegisterModelManager(manager)
	}

	for _, manager := range []db.IModelManager{
		db.OpsLog,
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
	} {
		db.RegisterModelManager(manager)
		handler := db.NewModelHandler(manager)
		dispatcher.AddModelDispatcher(API_VERSION, app, handler)
	}

	models.AddAdhocHandlers(API_VERSION, app)
}
