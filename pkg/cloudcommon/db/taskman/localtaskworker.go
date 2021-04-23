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
	"runtime/debug"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
)

var localTaskWorkerMan *appsrv.SWorkerManager

func init() {
	localTaskWorkerMan = appsrv.NewWorkerManager("LocalTaskWorkerManager", 4, 1024, false)
}

func Error2TaskData(err error) jsonutils.JSONObject {
	errJson := jsonutils.NewDict()
	errJson.Add(jsonutils.NewString("ERROR"), "__status__")
	errJson.Add(jsonutils.NewString(err.Error()), "__reason__")
	return errJson
}

func LocalTaskRunWithWorkers(task ITask, proc func() (jsonutils.JSONObject, error), wm *appsrv.SWorkerManager) {
	wm.Run(func() {

		log.Debugf("XXXXXXXXXXXXXXXXXXLOCAL TASK RUN STARTXXXXXXXXXXXXXXXXX")
		defer log.Debugf("XXXXXXXXXXXXXXXXXXLOCAL TASK RUN END  XXXXXXXXXXXXXXXXX")

		defer func() {
			if r := recover(); r != nil {
				log.Errorf("LocalTaskRun error: %s", r)
				debug.PrintStack()
				task.ScheduleRun(Error2TaskData(fmt.Errorf("LocalTaskRun error: %s", r)))
			}
		}()
		data, err := proc()
		if err != nil {
			task.ScheduleRun(Error2TaskData(err))
		} else {
			task.ScheduleRun(data)
		}

	}, nil, nil)
}

func LocalTaskRun(task ITask, proc func() (jsonutils.JSONObject, error)) {
	LocalTaskRunWithWorkers(task, proc, localTaskWorkerMan)
}
