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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

var _ IServerBaseDeployTask = new(SBaremetalServerRebuildTask)

type SBaremetalServerRebuildTask struct {
	SBaremetalServerBaseDeployTask
}

func NewBaremetalServerRebuildTask(
	userCred mcclient.TokenCredential,
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) ITask {
	task := &SBaremetalServerRebuildTask{
		SBaremetalServerBaseDeployTask: newBaremetalServerBaseDeployTask(userCred, baremetal, taskId, data),
	}
	task.SetVirtualObject(task)
	task.SetStage(task.InitPXEBootTask)
	return task
}

func (self *SBaremetalServerRebuildTask) GetName() string {
	return "BaremetalServerRebuildTask"
}

func (self *SBaremetalServerRebuildTask) RemoveEFIOSEntry() bool {
	return true
}

func (self *SBaremetalServerRebuildTask) DoDeploys(ctx context.Context, term *ssh.Client) (jsonutils.JSONObject, error) {
	tool, err := self.Baremetal.GetServer().NewConfigedSSHPartitionTool(term)
	if err != nil {
		return nil, errors.Wrap(err, "NewConfigedSSHPartitionTool")
	}
	parts, err := self.Baremetal.GetServer().DoRebuildRootDisk(tool, term, self.IsDisableImageCache())
	if err != nil {
		return nil, fmt.Errorf("Rebuild root disk: %v", err)
	}
	disks, err := self.Baremetal.GetServer().SyncPartitionSize(term, parts)
	if err != nil {
		return nil, fmt.Errorf("SyncPartitionSize: %v", err)
	}
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewArray(disks...), "disks")
	deployInfo, err := self.Baremetal.GetServer().DoDeploy(tool, term, self.data, true)
	if err != nil {
		return nil, fmt.Errorf("DoDeploy: %v", err)
	}
	data.Update(deployInfo)
	return data, nil
}

func (self *SBaremetalServerRebuildTask) PostDeploys(ctx context.Context, term *ssh.Client) error {
	if self.Baremetal.HasBMC() {
		return doPoweroff(term)
	}
	return nil
}
