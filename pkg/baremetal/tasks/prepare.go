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

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/baremetal/status"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type SBaremetalServerPrepareTask struct {
	SBaremetalTaskBase
}

func NewBaremetalServerPrepareTask(
	baremetal IBaremetal,
) *SBaremetalServerPrepareTask {
	task := &SBaremetalServerPrepareTask{
		SBaremetalTaskBase: newBaremetalTaskBase(auth.AdminCredential(), baremetal, "", nil),
	}
	task.SetVirtualObject(task)
	task.SetSSHStage(task.OnPXEBootRequest)
	return task
}

func (self *SBaremetalServerPrepareTask) NeedPXEBoot() bool {
	return true
}

func (self *SBaremetalServerPrepareTask) GetName() string {
	return "BaremetalServerPrepareTask"
}

// OnPXEBootRequest called by notify api handler
func (self *SBaremetalServerPrepareTask) OnPXEBootRequest(ctx context.Context, cli *ssh.Client, args interface{}) error {
	err := newBaremetalPrepareTask(self.Baremetal, self.userCred).DoPrepare(cli)
	if err != nil {
		log.Errorf("Prepare failed: %v", err)
		self.Baremetal.SyncStatus(status.PREPARE_FAIL, err.Error())
		return err
	}
	self.Baremetal.AutoSyncStatus()
	SetTaskComplete(self, nil)
	return nil
}
