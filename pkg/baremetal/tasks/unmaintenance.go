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
	"fmt"

	"yunion.io/x/jsonutils"

	baremetalstatus "yunion.io/x/onecloud/pkg/baremetal/status"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
)

type SBaremetalUnmaintenanceTask struct {
	*SBaremetalTaskBase
}

func NewBaremetalUnmaintenanceTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) (ITask, error) {
	task := new(SBaremetalUnmaintenanceTask)
	baseTask := newBaremetalTaskBase(baremetal, taskId, data)
	task.SBaremetalTaskBase = baseTask
	var err error
	if jsonutils.QueryBoolean(task.data, "guest_running", false) {
		err = task.EnsurePowerShutdown(false)
		if err != nil {
			return task, fmt.Errorf("EnsurePowerShutdown hard: %v", err)
		}
		err = task.EnsurePowerUp()
		if err != nil {
			return task, fmt.Errorf("EnsurePowerUp disk: %v", err)
		}
		task.Baremetal.SyncStatus(baremetalstatus.RUNNING, "")
		SetTaskComplete(task, nil)
		return task, nil
	}
	task.SetStage(task.WaitForStop)
	err = task.EnsurePowerShutdown(true)
	if err != nil {
		return task, fmt.Errorf("EnsurePowerShutdown soft: %v", err)
	}
	ExecuteTask(task, nil)
	return task, nil
}

func (self *SBaremetalUnmaintenanceTask) WaitForStop(ctx context.Context, args interface{}) error {
	status, err := self.Baremetal.GetPowerStatus()
	if err != nil {
		return err
	}
	self.SetStage(self.OnStopComplete)
	if status == types.POWER_STATUS_OFF {
		ExecuteTask(self, nil)
	}
	return nil
}

func (self *SBaremetalUnmaintenanceTask) OnStopComplete(ctx context.Context, args interface{}) error {
	self.Baremetal.SyncStatus(baremetalstatus.READY, "")
	SetTaskComplete(self, nil)
	return nil
}

func (self *SBaremetalUnmaintenanceTask) GetName() string {
	return "BaremetalUnmaintenanceTask"
}
