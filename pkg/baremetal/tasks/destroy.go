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
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/ssh"
)

type SBaremetalServerDestroyTask struct {
	*SBaremetalServerBaseDeployTask
}

func NewBaremetalServerDestroyTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) (ITask, error) {
	task := new(SBaremetalServerDestroyTask)
	baseTask, err := newBaremetalServerBaseDeployTask(baremetal, taskId, data, task)
	task.SBaremetalServerBaseDeployTask = baseTask
	return task, err
}

func (self *SBaremetalServerDestroyTask) GetName() string {
	return "BaremetalServerDestroyTask"
}

func (self *SBaremetalServerDestroyTask) DoDeploys(term *ssh.Client) (jsonutils.JSONObject, error) {
	if err := self.Baremetal.GetServer().DoEraseDisk(term); err != nil {
		log.Errorf("Delete server do erase disk: %v", err)
	}
	if err := self.Baremetal.GetServer().DoDiskUnconfig(term); err != nil {
		log.Errorf("Baremetal do disk unconfig: %v", err)
	}
	self.Baremetal.RemoveServer()
	return nil, nil
}
