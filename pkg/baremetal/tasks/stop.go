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
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
)

type SBaremetalServerStopTask struct {
	*SBaremetalTaskBase
	startTime time.Time
}

func NewBaremetalServerStopTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) (ITask, error) {
	baseTask := newBaremetalTaskBase(baremetal, taskId, data)
	self := &SBaremetalServerStopTask{
		SBaremetalTaskBase: baseTask,
	}
	self.SetStage(self.WaitForStop)
	if err := self.Baremetal.DoPowerShutdown(true); err != nil {
		return nil, fmt.Errorf("Do power shutdown error: %v", err)
	}
	self.startTime = time.Now()
	return self, nil
}

func (self *SBaremetalServerStopTask) GetName() string {
	return "BaremetalServerStopTask"
}

func (self *SBaremetalServerStopTask) WaitForStop(ctx context.Context, args interface{}) error {
	status, err := self.Baremetal.GetPowerStatus()
	if err != nil {
		return fmt.Errorf("GetPowerStatus: %v", err)
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
		return err
	}
	time.Sleep(10 * time.Second)
	ExecuteTask(self, nil)
	return nil
}

func (self *SBaremetalServerStopTask) OnStopComplete(ctx context.Context, args interface{}) error {
	SetTaskComplete(self, nil)
	return nil
}
