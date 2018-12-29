package taskman

import (
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

func AddTaskHandler(prefix string, app *appsrv.Application) {
	handler := db.NewModelHandler(TaskManager)
	dispatcher.AddModelDispatcher(prefix, app, handler)
}
