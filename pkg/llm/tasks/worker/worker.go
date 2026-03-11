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

	backupTaskWorkerMan *appsrv.SWorkerManager

	startTaskWorkerMan *appsrv.SWorkerManager

	importTaskWorkerMan *appsrv.SWorkerManager
)

func init() {
	localTaskWorkerManLock = &sync.Mutex{}
}

func BackupTaskRun(task taskman.ITask, proc func() (jsonutils.JSONObject, error)) {
	taskman.LocalTaskRunWithWorkers(task, proc, getBackupTaskWorkerMan())
}

func StartTaskRun(task taskman.ITask, proc func() (jsonutils.JSONObject, error)) {
	taskman.LocalTaskRunWithWorkers(task, proc, getStartTaskWorkerMan())
}

func ImportTaskRun(task taskman.ITask, proc func() (jsonutils.JSONObject, error)) {
	taskman.LocalTaskRunWithWorkers(task, proc, getImportTaskWorkerMan())
}

func getBackupTaskWorkerMan() *appsrv.SWorkerManager {
	localTaskWorkerManLock.Lock()
	defer localTaskWorkerManLock.Unlock()

	if backupTaskWorkerMan != nil {
		return backupTaskWorkerMan
	}
	log.Infof("BackupTaskWorkerCount %d", options.Options.BackupTaskWorkerCount)
	backupTaskWorkerMan = appsrv.NewWorkerManager("BackupTaskWorkerManager", options.Options.BackupTaskWorkerCount, 1024, false)
	return backupTaskWorkerMan
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

func getImportTaskWorkerMan() *appsrv.SWorkerManager {
	localTaskWorkerManLock.Lock()
	defer localTaskWorkerManLock.Unlock()

	if importTaskWorkerMan != nil {
		return importTaskWorkerMan
	}
	log.Infof("ImportTaskWorkerCount %d", options.Options.ImportTaskWorkerCount)
	importTaskWorkerMan = appsrv.NewWorkerManager("ImportTaskWorkerManager", options.Options.ImportTaskWorkerCount, 1024, false)
	return importTaskWorkerMan
}
