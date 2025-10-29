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

package taskman

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

var _saveTaskUpcallWorkMan *appsrv.SWorkerManager

func init() {
	_saveTaskUpcallWorkMan = appsrv.NewWorkerManager("SaveTaskUpcallResultWorkerManager", 1, 1024, true)
}

type saveUpcallStatusTask struct {
	taskId  string
	target  string
	success bool
	errMsg  string
	tm      time.Time
}

func (task *saveUpcallStatusTask) Run() {
	err := task.runInternal()
	if err != nil {
		log.Errorf("Task %s save upcall status fail %s", task.Dump(), err)
	}
}

func (task *saveUpcallStatusTask) Dump() string {
	return fmt.Sprintf("taskId: %s, success: %t, errMsg: %s", task.taskId, task.success, task.errMsg)
}

func saveTaskUpCallStatus(taskId string, target string, success bool, errMsg string) {
	_saveTaskUpcallWorkMan.Run(&saveUpcallStatusTask{
		taskId:  taskId,
		target:  target,
		success: success,
		errMsg:  errMsg,
		tm:      time.Now(),
	}, nil, nil)
}

func (t *saveUpcallStatusTask) runInternal() error {
	task := TaskManager.FetchTaskById(t.taskId)
	_, err := db.Update(task, func() error {
		params := jsonutils.NewDict()
		params.Update(task.Params)

		upcalls, _ := params.Get("__upcalls")
		if upcalls == nil {
			upcalls = jsonutils.NewArray()
			params.Add(upcalls, "__upcalls")
		}
		upcallsList := upcalls.(*jsonutils.JSONArray)
		upcallData := jsonutils.NewDict()
		upcallData.Add(jsonutils.NewString(t.target), "target")
		upcallData.Add(jsonutils.NewBool(t.success), "result")
		upcallData.Add(jsonutils.NewString(t.errMsg), "msg")
		upcallData.Add(jsonutils.NewTimeString(t.tm), "complete_at")
		upcallsList.Add(upcallData)
		task.Params = params
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "Update %s", t.taskId)
	}
	return nil
}
