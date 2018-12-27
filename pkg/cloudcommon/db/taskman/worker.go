package taskman

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
)

var taskWorkMan *appsrv.SWorkerManager
var taskWorkerTable map[string]*appsrv.SWorkerManager

func init() {
	taskWorkMan = appsrv.NewWorkerManager("TaskWorkerManager", 4, 1024, true)
	taskWorkerTable = make(map[string]*appsrv.SWorkerManager)
}

func runTask(taskId string, data jsonutils.JSONObject) {
	taskName := TaskManager.getTaskName(taskId)
	if len(taskName) == 0 {
		log.Errorf("no such task??? task_id=%s", taskId)
		return
	}
	worker := taskWorkMan
	if workerMan, ok := taskWorkerTable[taskName]; ok {
		worker = workerMan
	}
	worker.Run(func() {
		TaskManager.execTask(taskId, data)
	}, nil, nil)
}
