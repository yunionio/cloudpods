package worker

import (
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/options"
)

var (
	localTaskWorkerManLock *sync.Mutex

	startTaskWorkerMan *appsrv.SWorkerManager
)

func init() {
	localTaskWorkerManLock = &sync.Mutex{}
}

func getStartTaskWorkerMan() *appsrv.SWorkerManager {
	localTaskWorkerManLock.Lock()
	defer localTaskWorkerManLock.Unlock()

	if startTaskWorkerMan != nil {
		return startTaskWorkerMan
	}
	log.Infof("StartTaskWorkerCount %d", options.Options.StartTaskWorkerCount)
	startTaskWorkerMan = appsrv.NewWorkerManager("StartTaskWorkerManager", options.Options.StartTaskWorkerCount, 1024, false)
	return startTaskWorkerMan
}

func StartTaskRun(task taskman.ITask, proc func() (jsonutils.JSONObject, error)) {
	taskman.LocalTaskRunWithWorkers(task, proc, getStartTaskWorkerMan())
}
