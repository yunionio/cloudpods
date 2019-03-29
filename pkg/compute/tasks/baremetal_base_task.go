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
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SBaremetalBaseTask struct {
	taskman.STask
}

func (self *SBaremetalBaseTask) Action() string {
	actionMap := map[string]string{
		"start":         logclient.ACT_VM_START,
		"stop":          logclient.ACT_VM_STOP,
		"maintenance":   logclient.ACT_BM_MAINTENANCE,
		"unmaintenance": logclient.ACT_BM_UNMAINTENANCE,
	}
	if self.Params.Contains("action") {
		action, _ := self.Params.GetString("action")
		self.Params.Remove("action")
		if val, ok := actionMap[action]; len(action) > 0 && ok {
			return val
		}
	}
	return ""
}

func (self *SBaremetalBaseTask) GetBaremetal() *models.SHost {
	obj := self.GetObject()
	return obj.(*models.SHost)
}
