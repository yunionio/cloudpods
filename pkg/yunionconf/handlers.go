package yunionconf

import (
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/yunionconf/models"
)

func InitHandlers(app *appsrv.Application) {
	for _, manager := range []db.IModelManager{
		models.ParameterManager,
	} {
		db.RegisterModelManager(manager)
		handler := db.NewModelHandler(manager)
		dispatcher.AddModelDispatcher("", app, handler)
		dispatcher.AddModelDispatcher("/users/<user_id>", app, handler)
		dispatcher.AddModelDispatcher("/services/<service_id>", app, handler)
	}
}
