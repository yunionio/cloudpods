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

package proxmox

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type STask struct {
	User      string `json:"user"`
	Type      string `json:"type"`
	Saved     string `json:"saved"`
	Status    string `json:"status"`
	Starttime int    `json:"starttime"`
	Node      string `json:"node"`
	ID        string `json:"id"`
	Endtime   int    `json:"endtime"`
	Upid      string `json:"upid"`
}

func (c *SProxmoxClient) getTaskId(taskResponse map[string]interface{}) string {
	if taskResponse["errors"] != nil {
		errJSON, _ := json.MarshalIndent(taskResponse["errors"], "", "  ")
		return string(errJSON)
	}
	if taskResponse["data"] == nil {
		return ""
	}

	taskUpid := taskResponse["data"].(string)
	return taskUpid
}

func (self *SProxmoxClient) waitTask(id string) (string, error) {
	resId := ""
	return resId, cloudprovider.Wait(time.Second*10, time.Minute*10, func() (bool, error) {
		tasks := []STask{}
		err := self.get("/cluster/tasks", nil, &tasks)
		if err != nil {
			return false, errors.Wrapf(err, "get task %s", id)
		}

		for _, task := range tasks {
			if task.Upid == id {
				switch strings.ToLower(task.Status) {
				case "running":
					log.Debugf("task %s state: %s", task.Upid, task.Status)
					return false, nil
				case "ok":
					log.Debugf("task %s state: %s", task.Upid, task.Status)
					resId = task.Upid
					return true, nil
				case "":
					log.Debugf("task %s not finish", task.Upid)
					return false, nil
				default:
					log.Debugf("task %s state: %s", task.Upid, task.Status)
					return false, fmt.Errorf("task %s state: %s", task.Upid, task.Status)
				}

			}
		}

		return false, fmt.Errorf("not find task %s ", id)
	})
}
