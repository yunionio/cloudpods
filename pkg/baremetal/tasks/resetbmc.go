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

	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type SBaremetalResetBMCTask struct {
	SBaremetalPXEBootTaskBase
	term *ssh.Client
}

func NewBaremetalResetBMCTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) ITask {
	task := &SBaremetalResetBMCTask{
		SBaremetalPXEBootTaskBase: newBaremetalPXEBootTaskBase(baremetal, taskId, data),
	}
	task.SetVirtualObject(task)
	task.SetStage(task.InitPXEBootTask)
	return task
}

func (self *SBaremetalResetBMCTask) GetName() string {
	return "BaremetalResetBMCTask"
}

func (self *SBaremetalResetBMCTask) GetIPMITool() *ipmitool.SSHIPMI {
	return ipmitool.NewSSHIPMI(self.term)
}

func (self *SBaremetalResetBMCTask) OnPXEBoot(ctx context.Context, term *ssh.Client, args interface{}) error {
	self.term = term
	self.SetStage(self.WaitForBMCReady)
	err := ipmitool.DoBMCReset(self.GetIPMITool())
	if err != nil {
		return err
	}
	time.Sleep(10 * time.Second)
	ExecuteTask(self, nil)
	return nil
}

func (self *SBaremetalResetBMCTask) WaitForBMCReady(ctx context.Context, args interface{}) error {
	self.SetStage(self.OnBMCReady)
	status, err := ipmitool.GetChassisPowerStatus(self.GetIPMITool())
	if err != nil {
		return err
	}
	if status != "" && status == types.POWER_STATUS_ON {
		ExecuteTask(self, nil)
	}
	return nil
}

func (self *SBaremetalResetBMCTask) OnBMCReady(ctx context.Context, args interface{}) error {
	time.Sleep(20 * time.Second)
	SetTaskComplete(self, nil)
	return nil
}
