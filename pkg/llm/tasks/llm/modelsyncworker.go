package llm

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
)

func ModelSyncTaskRun(task taskman.ITask, llmId string, proc func() (jsonutils.JSONObject, error)) {
	taskman.LocalTaskRunWithWorkers(task, proc, instantModelSyncTaskWorkerMan.GetWorkerManager(llmId))
}
