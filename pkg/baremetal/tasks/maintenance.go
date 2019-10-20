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

	"yunion.io/x/onecloud/pkg/util/ssh"
)

type SBaremetalMaintenanceTask struct {
	SBaremetalPXEBootTaskBase
}

func NewBaremetalMaintenanceTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) ITask {
	task := &SBaremetalMaintenanceTask{
		SBaremetalPXEBootTaskBase: newBaremetalPXEBootTaskBase(baremetal, taskId, data),
	}
	task.SetVirtualObject(task)
	task.SetStage(task.InitPXEBootTask)
	return task
}

func (self *SBaremetalMaintenanceTask) OnPXEBoot(ctx context.Context, term *ssh.Client, args interface{}) error {
	sshConfig := term.GetConfig()
	dataObj := map[string]interface{}{
		"username": sshConfig.Username,
		"password": sshConfig.Password,
		"ip":       sshConfig.Host,
	}
	if jsonutils.QueryBoolean(self.data, "guest_running", false) {
		dataObj["guest_running"] = true
	}
	self.Baremetal.AutoSyncStatus()
	SetTaskComplete(self, jsonutils.Marshal(dataObj))
	return nil
}
