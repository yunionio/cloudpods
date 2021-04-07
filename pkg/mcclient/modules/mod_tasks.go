// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package modules

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

var (
	Tasks modulebase.ResourceManager

	ComputeTasks ComputeTasksManager
	DevtoolTasks modulebase.ResourceManager
)

type ComputeTasksManager struct {
	modulebase.ResourceManager
}

func init() {
	cols := []string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "instance_id", "business_id", "task_id", "task_type", "task_name", "task_status", "current_approver", "approver_name", "receive_time", "finish_time", "result", "content", "common_start_string"}
	Tasks = NewITSMManager("task", "taskman", cols, cols)
	register(&Tasks)

	ComputeTasks = ComputeTasksManager{
		ResourceManager: NewComputeManager("task", "tasks",
			[]string{},
			[]string{"Id", "Obj_name", "Obj_Id", "Task_name", "Stage", "Created_at"}),
	}
	registerCompute(&ComputeTasks)

	DevtoolTasks = NewDevtoolManager("task", "tasks",
		[]string{},
		[]string{"Id", "Obj_name", "Obj_Id", "Task_name", "Stage", "Created_at"},
	)
}

type ITaskResourceManager interface {
	PerformClassAction(session *mcclient.ClientSession, action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
}

func TaskComplete(man ITaskResourceManager, session *mcclient.ClientSession, taskId string, params jsonutils.JSONObject) {
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

func TaskFailed(man ITaskResourceManager, session *mcclient.ClientSession, taskId string, reason string) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString("error"), "__status__")
	params.Add(jsonutils.NewString(reason), "__reason__")
	TaskComplete(man, session, taskId, params)
}

func (man ComputeTasksManager) TaskComplete(session *mcclient.ClientSession, taskId string, params jsonutils.JSONObject) {
	TaskComplete(&man, session, taskId, params)
}

func (man ComputeTasksManager) TaskFailed(session *mcclient.ClientSession, taskId string, err error) {
	man.TaskFailed2(session, taskId, err.Error())
}

func (man ComputeTasksManager) TaskFailed2(session *mcclient.ClientSession, taskId string, reason string) {
	man.TaskFailed3(session, taskId, reason, nil)
}

func (man ComputeTasksManager) TaskFailed3(session *mcclient.ClientSession, taskId string, reason string, params *jsonutils.JSONDict) {
	if params == nil {
		params = jsonutils.NewDict()
	}
	params.Add(jsonutils.NewString("error"), "__status__")
	params.Add(jsonutils.NewString(reason), "__reason__")
	man.TaskComplete(session, taskId, params)
}
