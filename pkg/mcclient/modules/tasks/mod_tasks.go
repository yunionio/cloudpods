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

package tasks

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type TasksManager struct {
	modulebase.ResourceManager
}

func NewTaskManagers(taskManager func(string, string, []string, []string) modulebase.ResourceManager) (TasksManager, TasksManager) {
	task := TasksManager{
		ResourceManager: taskManager("task", "tasks",
			[]string{},
			[]string{"Id", "Obj_name", "Obj_Id", "Task_name", "Stage", "Created_at"}),
	}
	modules.Register(&task)
	archivedTask := TasksManager{
		ResourceManager: taskManager("archivedtask", "archivedtasks",
			[]string{},
			[]string{"Id", "task_id", "Obj_name", "Obj_Id", "Task_name", "Stage", "Start_at"}),
	}
	modules.Register(&archivedTask)
	return task, archivedTask
}

func (man *TasksManager) TaskComplete(session *mcclient.ClientSession, taskId string, params jsonutils.JSONObject) {
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

func (man *TasksManager) TaskFailed(session *mcclient.ClientSession, taskId string, err error) {
	man.TaskFailed2(session, taskId, err.Error())
}

func (man *TasksManager) TaskFailed2(session *mcclient.ClientSession, taskId string, reason string) {
	man.TaskFailed3(session, taskId, reason, nil)
}

func (man *TasksManager) TaskFailed3(session *mcclient.ClientSession, taskId string, reason string, params *jsonutils.JSONDict) {
	if params == nil {
		params = jsonutils.NewDict()
	}
	params.Add(jsonutils.NewString("error"), "__status__")
	params.Add(jsonutils.NewString(reason), "__reason__")
	man.TaskComplete(session, taskId, params)
}

/*func (self *TasksManager) getManager(session *mcclient.ClientSession, params jsonutils.JSONObject) (*modulebase.ResourceManager, error) {
	serviceType := apis.SERVICE_TYPE_REGION
	if params.Contains("service_type") {
		serviceType, _ = params.GetString("service_type")
	}

	version := ""
	switch serviceType {
	case apis.SERVICE_TYPE_KEYSTONE:
		version = "v3"
	case apis.SERVICE_TYPE_REGION, apis.SERVICE_TYPE_NOTIFY:
		version = "v2"
	case apis.SERVICE_TYPE_IMAGE:
		version = "v1"
	}

	_, err := session.GetServiceURL(serviceType, "")
	if err != nil {
		return nil, httperrors.NewNotFoundError("service %s not found error: %v", serviceType, err)
	}

	return &modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(serviceType, "", version, []string{}, []string{}),
		Keyword:     "task", KeywordPlural: "tasks",
	}, nil
}

func (this *TasksManager) List(session *mcclient.ClientSession, params jsonutils.JSONObject) (*printutils.ListResult, error) {
	man, err := this.getManager(session, params)
	if err != nil {
		return nil, err
	}
	return man.List(session, params)
}*/
