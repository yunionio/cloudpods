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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

var _ IServerBaseDeployTask = new(SBaremetalServerDeployTask)

type SBaremetalServerDeployTask struct {
	SBaremetalServerBaseDeployTask
}

func NewBaremetalServerDeployTask(
	userCred mcclient.TokenCredential,
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) ITask {
	task := &SBaremetalServerDeployTask{
		SBaremetalServerBaseDeployTask: newBaremetalServerBaseDeployTask(userCred, baremetal, taskId, data),
	}
	task.SetVirtualObject(task)
	task.SetStage(task.InitPXEBootTask)
	return task
}

func (self *SBaremetalServerDeployTask) GetName() string {
	return "BaremetalServerDeployTask"
}

func (self *SBaremetalServerDeployTask) RemoveEFIOSEntry() bool {
	return false
}

func (self *SBaremetalServerDeployTask) DoDeploys(ctx context.Context, term *ssh.Client) (jsonutils.JSONObject, error) {
	tool, err := self.Baremetal.GetServer().NewConfigedSSHPartitionTool(term)
	if err != nil {
		return nil, errors.Wrap(err, "NewConfigedSSHPartitionTool")
	}
	return self.Baremetal.GetServer().DoDeploy(tool, term, self.data, false)
}
