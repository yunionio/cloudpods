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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type SBaremetalServerCreateTask struct {
	SBaremetalServerBaseDeployTask
}

func NewBaremetalServerCreateTask(
	userCred mcclient.TokenCredential,
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) ITask {
	task := &SBaremetalServerCreateTask{
		SBaremetalServerBaseDeployTask: newBaremetalServerBaseDeployTask(userCred, baremetal, taskId, data),
	}
	task.SetVirtualObject(task)
	task.SetStage(task.InitPXEBootTask)
	return task
}

func (self *SBaremetalServerCreateTask) GetName() string {
	return "BaremetalServerCreateTask"
}

func (self *SBaremetalServerCreateTask) DoDeploys(term *ssh.Client) (jsonutils.JSONObject, error) {
	// Build raid
	err := self.Baremetal.GetServer().DoDiskConfig(term)
	if err != nil {
		return nil, self.onError(term, err)
	}
	time.Sleep(2 * time.Second)
	if err := self.Baremetal.GetServer().DoEraseDisk(term); err != nil {
		return nil, self.onError(term, err)
	}
	time.Sleep(2 * time.Second)
	parts, err := self.Baremetal.GetServer().DoPartitionDisk(term)
	if err != nil {
		return nil, self.onError(term, err)
	}
	data := jsonutils.NewDict()
	disks, err := self.Baremetal.GetServer().SyncPartitionSize(term, parts)
	if err != nil {
		return nil, self.onError(term, err)
	}
	data.Add(jsonutils.Marshal(disks), "disks")
	rootImageId := self.Baremetal.GetServer().GetRootTemplateId()
	if len(rootImageId) > 0 {
		deployInfo, err := self.Baremetal.GetServer().DoDeploy(term, self.data, true)
		if err != nil {
			return nil, self.onError(term, err)
		}
		if deployInfo != nil {
			data.Update(deployInfo)
		}
	}
	return data, nil
}

func doPoweroff(term *ssh.Client) error {
	if _, err := term.Run("/sbin/poweroff"); err != nil {
		log.Errorf("poweroff error: %s", err)
		return nil
	}
	time.Sleep(2 * time.Second)
	return nil
}

func (self *SBaremetalServerCreateTask) PostDeploys(term *ssh.Client) error {
	return doPoweroff(term)
}

func (self *SBaremetalServerCreateTask) onError(term *ssh.Client, err error) error {
	log.Errorf("Create server error: %+v", err)
	if err1 := self.Baremetal.GetServer().DoEraseDisk(term); err1 != nil {
		log.Warningf("EraseDisk error: %v", err1)
	}
	self.Baremetal.AutoSyncStatus()
	return err
}
