package service

import (
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/notify/models"
	"yunion.io/x/onecloud/pkg/notify/oldmodels"
)

const (
	API_VERSION = "v2"
)

func InitHandlers(app *appsrv.Application) {
	db.InitAllManagers()

	db.RegistUserCredCacheUpdater()

	db.AddScopeResourceCountHandler(API_VERSION, app)

	// Data migration
	db.RegisterModelManager(oldmodels.NotificationManager)
	db.RegisterModelManager(oldmodels.ContactManager)
	db.RegisterModelManager(oldmodels.ConfigManager)
	db.RegisterModelManager(oldmodels.TemplateManager)
	db.RegisterModelManager(oldmodels.UserCacheManager)

	taskman.AddTaskHandler(API_VERSION, app)
	for _, manager := range []db.IModelManager{
		taskman.TaskManager,
		taskman.SubTaskManager,
		taskman.TaskObjectManager,

		db.UserCacheManager,
		db.TenantCacheManager,
		db.RoleCacheManager,
		models.SubContactManager,
		db.SharedResourceManager,
		models.VerificationManager,
		models.SubscriptionReceiverManager,
	} {
		db.RegisterModelManager(manager)
	}
	for _, manager := range []db.IModelManager{
		db.OpsLog,
		db.Metadata,

		models.ReceiverManager,
		models.NotificationManager,
		models.ConfigManager,
		models.TemplateManager,
		models.SubscriptionManager,
		models.RobotManager,
	} {
		db.RegisterModelManager(manager)
		handler := db.NewModelHandler(manager)
		dispatcher.AddModelDispatcher(API_VERSION, app, handler)
	}
	for _, manager := range []db.IJointModelManager{
		models.ReceiverNotificationManager,
	} {
		db.RegisterModelManager(manager)
		handler := db.NewJointModelHandler(manager)
		dispatcher.AddJointModelDispatcher(API_VERSION, app, handler)
	}
}
