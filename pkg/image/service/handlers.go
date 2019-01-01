package service

import (
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/image/usages"
)

const (
	API_VERSION = "v1"
)

func initHandlers(app *appsrv.Application) {
	db.InitAllManagers()

	quotas.AddQuotaHandler(models.QuotaManager, API_VERSION, app)
	usages.AddUsageHandler(API_VERSION, app)
	taskman.AddTaskHandler(API_VERSION, app)

	for _, manager := range []db.IModelManager{
		taskman.TaskManager,
		taskman.SubTaskManager,
		taskman.TaskObjectManager,
		// db.UserCacheManager,
		db.TenantCacheManager,
		db.Metadata,
		models.ImageTagManager,
		models.ImageMemberManager,
		models.ImagePropertyManager,
		models.ImageSubformatManager,
	} {
		db.RegisterModelManager(manager)
	}

	for _, manager := range []db.IModelManager{
		db.OpsLog,
		models.ImageManager,
	} {
		db.RegisterModelManager(manager)
		handler := db.NewModelHandler(manager)
		dispatcher.AddModelDispatcher(API_VERSION, app, handler)
	}
}
