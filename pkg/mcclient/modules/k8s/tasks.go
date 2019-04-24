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
