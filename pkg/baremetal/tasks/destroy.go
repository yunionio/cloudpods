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
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

var _ IServerBaseDeployTask = new(SBaremetalServerDestroyTask)

type SBaremetalServerDestroyTask struct {
	SBaremetalServerBaseDeployTask
}

func NewBaremetalServerDestroyTask(
	userCred mcclient.TokenCredential,
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) ITask {
	task := &SBaremetalServerDestroyTask{
		SBaremetalServerBaseDeployTask: newBaremetalServerBaseDeployTask(userCred, baremetal, taskId, data),
	}
	task.SetVirtualObject(task)
	task.SetStage(task.InitPXEBootTask)
	return task
}

func (self *SBaremetalServerDestroyTask) GetName() string {
	return "BaremetalServerDestroyTask"
}

func (self *SBaremetalServerDestroyTask) RemoveEFIOSEntry() bool {
	return true
}

func (self *SBaremetalServerDestroyTask) DoDeploys(ctx context.Context, term *ssh.Client) (jsonutils.JSONObject, error) {
	if err := self.Baremetal.GetServer().DoEraseDisk(term); err != nil {
		log.Errorf("Delete server do erase disk: %v", err)
	}
	if err := self.Baremetal.GetServer().DoDiskUnconfig(term); err != nil {
		log.Errorf("Baremetal do disk unconfig: %v", err)
	}
	self.Baremetal.RemoveServer()
	return nil, nil
}
