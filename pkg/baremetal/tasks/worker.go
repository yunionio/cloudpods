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
	"runtime/debug"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var baremetalTaskWorkerMan *appsrv.SWorkerManager

func init() {
	baremetalTaskWorkerMan = appsrv.NewWorkerManager("BaremetalTaskWorkerManager", 8, 1024, false)
}

func GetWorkManager() *appsrv.SWorkerManager {
	return baremetalTaskWorkerMan
}

func OnStop() {
	for GetWorkManager().ActiveWorkerCount() > 0 {
		log.Warningf("Busy workers count %d, waiting them finish", GetWorkManager().ActiveWorkerCount())
		time.Sleep(5 * time.Second)
	}
}

type baremtalTask struct {
	task ITask
	args interface{}
}

func (t *baremtalTask) Run() {
	executeTask(t.task, t.args)
}

func (t *baremtalTask) Dump() string {
	return fmt.Sprintf("Task %s(%s) params: %v", t.task.GetName(), t.task.GetTaskId(), t.args)
}

func ExecuteTask(task ITask, args interface{}) {
	t := &baremtalTask{
		task: task,
		args: args,
	}
	baremetalTaskWorkerMan.Run(t, nil, nil)
}

func executeTask(task ITask, args interface{}) {
	if task == nil {
		return
	}
	curStage := task.GetStage()
	if curStage == nil {
		return
	}
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Execute task panic: %v", err)
			debug.PrintStack()
			SetTaskFail(task, fmt.Errorf("%v", err))
		}
	}()
	err := curStage(context.Background(), args)
	if err != nil {
		log.Errorf("Execute task %s error: %v", task.GetName(), err)
		SetTaskFail(task, err)
	}
}

func SetTaskComplete(task ITask, data jsonutils.JSONObject) {
	taskId := task.GetTaskId()
	if taskId != "" {
		session := task.GetClientSession()
		modules.ComputeTasks.TaskComplete(session, taskId, data)
	}
	onTaskEnd(task)
}

func SetTaskFail(task ITask, err error) {
	taskId := task.GetTaskId()
	if taskId != "" {
		session := task.GetClientSession()
		modules.ComputeTasks.TaskFailed(session, taskId, err)
	}
	onTaskEnd(task)
}

func onTaskEnd(task ITask) {
	task.SetStage(nil)
	ExecuteTask(task.GetTaskQueue().PopTask(), nil)
}

func OnInitStage(task ITask) error {
	log.Infof("Start task %s", task.GetName())
	return nil
}
