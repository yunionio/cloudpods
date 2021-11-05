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

package compute

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	ComputeTasks ComputeTasksManager
	DevtoolTasks modulebase.ResourceManager
)

type ComputeTasksManager struct {
	modulebase.ResourceManager
}

func init() {
	ComputeTasks = ComputeTasksManager{
		ResourceManager: modules.NewComputeManager("task", "tasks",
			[]string{},
			[]string{"Id", "Obj_name", "Obj_Id", "Task_name", "Stage", "Created_at"}),
	}
	modules.RegisterCompute(&ComputeTasks)

	DevtoolTasks = modules.NewDevtoolManager("task", "tasks",
		[]string{},
		[]string{"Id", "Obj_name", "Obj_Id", "Task_name", "Stage", "Created_at"},
	)
}

func (man ComputeTasksManager) TaskComplete(session *mcclient.ClientSession, taskId string, params jsonutils.JSONObject) {
	modules.TaskComplete(&man, session, taskId, params)
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
