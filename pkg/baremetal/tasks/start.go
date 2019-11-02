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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SBaremetalServerStartTask struct {
	SBaremetalTaskBase
}

func NewBaremetalServerStartTask(
	userCred mcclient.TokenCredential,
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) ITask {
	task := &SBaremetalServerStartTask{
		SBaremetalTaskBase: newBaremetalTaskBase(userCred, baremetal, taskId, data),
	}
	task.SetVirtualObject(task)
	task.SetStage(task.DoBoot)
	return task
}

func (self *SBaremetalServerStartTask) DoBoot(ctx context.Context, args interface{}) error {
	conf := self.Baremetal.GetRawIPMIConfig()
	if !conf.CdromBoot {
		err := self.Baremetal.DoPXEBoot()
		if err != nil {
			return errors.Wrap(err, "DoPXEBoot")
		}
	} else {
		err := self.Baremetal.DoRedfishPowerOn()
		if err != nil {
			return errors.Wrap(err, "DoRedfishPowerOn")
		}
	}
	self.SetStage(self.WaitForStart)
	ExecuteTask(self, nil)
	return nil
}

func (self *SBaremetalServerStartTask) GetName() string {
	return "BaremetalServerStartTask"
}

func (self *SBaremetalServerStartTask) WaitForStart(ctx context.Context, args interface{}) error {
	status, err := self.Baremetal.GetPowerStatus()
	if err != nil {
		return errors.Wrap(err, "GetPowerStatus")
	}
	log.Infof("%s WaitForStart status=%s", self.GetName(), status)
	if status == types.POWER_STATUS_ON {
		self.SetStage(self.OnStartComplete)
	}
	ExecuteTask(self, nil)
	return nil
}

func (self *SBaremetalServerStartTask) OnStartComplete(ctx context.Context, args interface{}) error {
	self.Baremetal.SyncAllStatus(types.POWER_STATUS_ON)
	SetTaskComplete(self, nil)
	return nil
}
