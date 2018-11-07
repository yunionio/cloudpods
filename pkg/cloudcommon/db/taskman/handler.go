package taskman

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

var taskWorkMan *appsrv.SWorkerManager

func init() {
	taskWorkMan = appsrv.NewWorkerManager("TaskWorkerManager", 4, 100)
}

func AddTaskHandler(prefix string, app *appsrv.Application) {
	handler := db.NewModelHandler(TaskManager)
	dispatcher.AddModelDispatcher(prefix, app, handler)
}

func runTask(taskId string, data jsonutils.JSONObject) {
	taskWorkMan.Run(func() {
		TaskManager.execTask(taskId, data)
	}, nil, nil)
}
