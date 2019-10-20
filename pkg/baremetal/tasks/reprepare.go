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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/util/ssh"
)

type SBaremetalReprepareTask struct {
	SBaremetalServerBaseDeployTask
}

func NewBaremetalReprepareTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) ITask {
	task := &SBaremetalReprepareTask{
		SBaremetalServerBaseDeployTask: newBaremetalServerBaseDeployTask(baremetal, taskId, data),
	}
	task.SetVirtualObject(task)
	task.SetStage(task.InitPXEBootTask)
	return task
}

func (self *SBaremetalReprepareTask) GetName() string {
	return "BaremetalReprepareTask"
}

func (self *SBaremetalReprepareTask) DoDeploys(term *ssh.Client) (jsonutils.JSONObject, error) {
	task := newBaremetalPrepareTask(self.Baremetal)
	err := task.DoPrepare(term)
	return nil, err
}
