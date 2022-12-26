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
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	ComputeTasks TasksManager
	DevtoolTasks TasksManager
)

type TasksManager struct {
	modulebase.ResourceManager
}

func init() {
	ComputeTasks = TasksManager{
		ResourceManager: modules.NewComputeManager("task", "tasks",
			[]string{},
			[]string{"Id", "Obj_name", "Obj_Id", "Task_name", "Stage", "Created_at"}),
	}
	modules.RegisterCompute(&ComputeTasks)

	DevtoolTasks = TasksManager{
		ResourceManager: modules.NewDevtoolManager("task", "tasks",
			[]string{},
			[]string{"Id", "Obj_name", "Obj_Id", "Task_name", "Stage", "Created_at"},
		),
	}
}

func (man TasksManager) TaskComplete(session *mcclient.ClientSession, taskId string, params jsonutils.JSONObject) {
	modules.TaskComplete(&man, session, taskId, params)
}

func (man TasksManager) TaskFailed(session *mcclient.ClientSession, taskId string, err error) {
	man.TaskFailed2(session, taskId, err.Error())
}

func (man TasksManager) TaskFailed2(session *mcclient.ClientSession, taskId string, reason string) {
	man.TaskFailed3(session, taskId, reason, nil)
}

func (man TasksManager) TaskFailed3(session *mcclient.ClientSession, taskId string, reason string, params *jsonutils.JSONDict) {
	if params == nil {
		params = jsonutils.NewDict()
	}
	params.Add(jsonutils.NewString("error"), "__status__")
	params.Add(jsonutils.NewString(reason), "__reason__")
	man.TaskComplete(session, taskId, params)
}

func (self *TasksManager) getManager(session *mcclient.ClientSession, params jsonutils.JSONObject) (*modulebase.ResourceManager, error) {
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
}
