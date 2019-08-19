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
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/util/ssh"
)

type SBaremetalServerRebuildTask struct {
	*SBaremetalServerBaseDeployTask
}

func NewBaremetalServerRebuildTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) (ITask, error) {
	task := new(SBaremetalServerRebuildTask)
	baseTask, err := newBaremetalServerBaseDeployTask(baremetal, taskId, data, task)
	task.SBaremetalServerBaseDeployTask = baseTask
	return task, err
}

func (self *SBaremetalServerRebuildTask) GetName() string {
	return "BaremetalServerRebuildTask"
}

func (self *SBaremetalServerRebuildTask) DoDeploys(term *ssh.Client) (jsonutils.JSONObject, error) {
	parts, err := self.Baremetal.GetServer().DoRebuildRootDisk(term)
	if err != nil {
		return nil, fmt.Errorf("Rebuild root disk: %v", err)
	}
	disks, err := self.Baremetal.GetServer().SyncPartitionSize(term, parts)
	if err != nil {
		return nil, fmt.Errorf("SyncPartitionSize: %v", err)
	}
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewArray(disks...), "disks")
	deployInfo, err := self.Baremetal.GetServer().DoDeploy(term, self.data, false)
	if err != nil {
		return nil, fmt.Errorf("DoDeploy: %v", err)
	}
	data.Update(deployInfo)
	return data, nil
}

func (self *SBaremetalServerRebuildTask) PostDeploys(term *ssh.Client) error {
	return doPoweroff(term)
}
