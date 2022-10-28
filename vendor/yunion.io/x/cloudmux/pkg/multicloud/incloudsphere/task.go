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

package incloudsphere

import (
	"fmt"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type STask struct {
	Id                string
	Name              string
	Detail            string
	Error             string
	State             string
	CreatedResourceId string
}

func (self *SphereClient) waitTask(id string) (string, error) {
	resId := ""
	return resId, cloudprovider.Wait(time.Second*10, time.Minute*10, func() (bool, error) {
		task := &STask{}
		err := self.get("/tasks/"+id, nil, task)
		if err != nil {
			return false, errors.Wrapf(err, "get task %s", id)
		}
		switch task.State {
		case "WAITING", "READY", "RUNNING":
			log.Debugf("task %s with %s state: %s", task.Id, task.Detail, task.State)
			return false, nil
		case "FINISHED":
			resId = task.CreatedResourceId
			return true, nil
		case "ERROR":
			return false, fmt.Errorf("%s: %s", task.Detail, task.Error)
		default:
			log.Debugf("task %s with %s state: %s", task.Id, task.Detail, task.State)
			return false, fmt.Errorf("%s: %s invalid state: %v", task.Detail, task.Error, task.State)
		}
	})
}
