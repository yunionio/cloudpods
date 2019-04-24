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
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
)

type SBaremetalServerStartTask struct {
	*SBaremetalTaskBase
}

func NewBaremetalServerStartTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) (ITask, error) {
	baseTask := newBaremetalTaskBase(baremetal, taskId, data)
	self := &SBaremetalServerStartTask{
		SBaremetalTaskBase: baseTask,
	}
	if err := self.Baremetal.DoPXEBoot(); err != nil {
		return nil, fmt.Errorf("DoPXEBoot: %v", err)
	}
	self.SetStage(self.WaitForStart)
	ExecuteTask(self, nil)
	return self, nil
}

func (self *SBaremetalServerStartTask) GetName() string {
	return "BaremetalServerStartTask"
}

func (self *SBaremetalServerStartTask) WaitForStart(ctx context.Context, args interface{}) error {
	self.SetStage(self.OnStartComplete)
	status, err := self.Baremetal.GetPowerStatus()
	if err != nil {
		return fmt.Errorf("GetPowerStatus: %v", err)
	}
	log.Infof("%s WaitForStart status=%s", self.GetName(), status)
	if status == types.POWER_STATUS_ON {
		ExecuteTask(self, nil)
	}
	return nil
}

func (self *SBaremetalServerStartTask) OnStartComplete(ctx context.Context, args interface{}) error {
	self.Baremetal.SyncAllStatus(types.POWER_STATUS_ON)
	SetTaskComplete(self, nil)
	return nil
}
