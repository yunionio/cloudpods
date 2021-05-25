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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SBaremetalServerStopTask struct {
	SBaremetalTaskBase
	startTime time.Time
}

func NewBaremetalServerStopTask(
	userCred mcclient.TokenCredential,
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) ITask {
	task := &SBaremetalServerStopTask{
		SBaremetalTaskBase: newBaremetalTaskBase(userCred, baremetal, taskId, data),
	}
	task.SetVirtualObject(task)
	task.SetStage(task.DoStop)
	return task
}

func (task *SBaremetalServerStopTask) DoStop(ctx context.Context, args interface{}) error {
	if task.Baremetal.HasBMC() {
		task.SetStage(task.WaitForStop)
		if err := task.Baremetal.DoPowerShutdown(true); err != nil {
			log.Errorf("Do power shutdown error: %s", err)
		}
	} else {
		if err := task.Baremetal.SSHShutdown(); err != nil {
			return errors.Wrap(err, "Try ssh shutdown")
		}
		task.SetStage(task.OnStopComplete)
	}
	task.startTime = time.Now()
	ExecuteTask(task, nil)
	return nil
}

func (self *SBaremetalServerStopTask) GetName() string {
	return "BaremetalServerStopTask"
}

func (self *SBaremetalServerStopTask) WaitForStop(ctx context.Context, args interface{}) error {
	status, err := self.Baremetal.GetPowerStatus()
	if err != nil {
		return errors.Wrap(err, "GetPowerStatus")
	}
	if status == types.POWER_STATUS_OFF {
		self.SetStage(self.OnStopComplete)
		ExecuteTask(self, nil)
		return nil
	}
	isSoft := true
	if time.Since(self.startTime) >= 90*time.Second {
		isSoft = false
	}
	if err := self.Baremetal.DoPowerShutdown(isSoft); err != nil {
		log.Errorf("DoPowerShutdown soft=%v error: %s", isSoft, err)
	}
	time.Sleep(10 * time.Second)
	ExecuteTask(self, nil)
	return nil
}

func (self *SBaremetalServerStopTask) OnStopComplete(ctx context.Context, args interface{}) error {
	SetTaskComplete(self, nil)
	return nil
}
