package modules

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	Tasks ResourceManager

	ComputeTasks ComputeTasksManager
)

type ComputeTasksManager struct {
	ResourceManager
}

func init() {
	Tasks = NewITSMManager("task", "taskman",
		[]string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "instance_id", "business_id", "task_id", "task_type", "task_name", "task_status", "current_approver", "approver_name", "receive_time", "finish_time", "result", "content", "common_start_string"},
		[]string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "instance_id", "business_id", "task_id", "task_type", "task_name", "task_status", "current_approver", "approver_name", "receive_time", "finish_time", "result", "content", "common_start_string"},
	)
	register(&Tasks)

	ComputeTasks = ComputeTasksManager{
		ResourceManager: NewComputeManager("task", "tasks",
			[]string{},
			[]string{"Id", "Obj_name", "Obj_Id", "Task_name", "Stage", "Created_at"}),
	}
	registerCompute(&ComputeTasks)
}

func (man ComputeTasksManager) TaskComplete(session *mcclient.ClientSession, taskId string, params jsonutils.JSONObject) {
	for i := 0; i < 3; i++ {
		_, err := man.PerformClassAction(session, taskId, params)
		if err == nil {
			log.Infof("Sync task %s complete succ", taskId)
			break
		}
		log.Errorf("Sync task %s complete error: %v", taskId, err)
		time.Sleep(5 * time.Second)
	}
}

func (man ComputeTasksManager) TaskFailed(session *mcclient.ClientSession, taskId string, err error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString("error"), "__status__")
	params.Add(jsonutils.NewString(err.Error()), "__reason__")
	man.TaskComplete(session, taskId, params)
}
