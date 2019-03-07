package k8s

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	KubeTasks *KubeTasksManager
)

type KubeTasksManager struct {
	ResourceManager
}

func init() {
	KubeTasks = &KubeTasksManager{
		ResourceManager: *NewResourceManager("task", "tasks", NewColumns(), NewColumns("Id", "Obj_name", "Obj_Id", "Task_name", "Stage", "Created_at")),
	}
}

func (m *KubeTasksManager) TaskComplete(session *mcclient.ClientSession, taskId string, params jsonutils.JSONObject) {
	modules.TaskComplete(m, session, taskId, params)
}

func (m *KubeTasksManager) TaskFailed(session *mcclient.ClientSession, taskId string, reason string) {
	modules.TaskFailed(m, session, taskId, reason)
}
