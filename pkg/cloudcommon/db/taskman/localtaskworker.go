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
	"context"
	"fmt"
	"runtime/debug"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/version"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/mcclient/modules/yunionconf"
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

type localTask struct {
	task ITask
	proc func() (jsonutils.JSONObject, error)
}

func (t *localTask) Run() {
	log.Debugf("XXXXXXXXXXXXXXXXXXLOCAL TASK RUN STARTXXXXXXXXXXXXXXXXX")
	defer log.Debugf("XXXXXXXXXXXXXXXXXXLOCAL TASK RUN END  XXXXXXXXXXXXXXXXX")

	defer func() {
		if r := recover(); r != nil {
			yunionconf.BugReport.SendBugReport(context.Background(), version.GetShortString(), string(debug.Stack()), errors.Errorf("%s", r))
			log.Errorf("LocalTaskRun error: %s", r)
			debug.PrintStack()
			t.task.ScheduleRun(Error2TaskData(fmt.Errorf("LocalTaskRun error: %s", r)))
		}
	}()
	data, err := t.proc()
	if err != nil {
		t.task.ScheduleRun(Error2TaskData(err))
	} else {
		t.task.ScheduleRun(data)
	}
}

func (t *localTask) Dump() string {
	return fmt.Sprintf("StartTime: %s TaskId: %s Params: %s", t.task.GetStartTime(), t.task.GetTaskId(), t.task.GetParams())
}

func LocalTaskRunWithWorkers(task ITask, proc func() (jsonutils.JSONObject, error), wm *appsrv.SWorkerManager) {
	t := localTask{
		task: task,
		proc: proc,
	}
	wm.Run(&t, nil, nil)
}

func LocalTaskRun(task ITask, proc func() (jsonutils.JSONObject, error)) {
	LocalTaskRunWithWorkers(task, proc, localTaskWorkerMan)
}
