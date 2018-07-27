package taskman

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/pkg/appsrv"
	"github.com/yunionio/pkg/appsrv/dispatcher"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
)

var taskWorkMan *appsrv.WorkerManager

func init() {
	taskWorkMan = appsrv.NewWorkerManager("TaskWorkerManager", 4, 10)
}

func AddTaskHandler(prefix string, app *appsrv.Application) {
	handler := db.NewModelHandler(TaskManager)
	dispatcher.AddModelDispatcher(prefix, app, handler)
}

func runTask(taskId string, data jsonutils.JSONObject) {
	taskWorkMan.Run(func() {
		TaskManager.execTask(taskId, data)
	}, nil)
}
