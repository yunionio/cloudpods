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

package handler

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var delayTaskWorkerMan *appsrv.SWorkerManager

func init() {
	delayTaskWorkerMan = appsrv.NewWorkerManager("DelayTaskWorkerManager", 8, 1024, false)
}

type ProcessFunc func(data jsonutils.JSONObject) (jsonutils.JSONObject, error)

type delayTask struct {
	process ProcessFunc
	taskId  string
	session *mcclient.ClientSession
	data    jsonutils.JSONObject
}

func (t *delayTask) Run() {
	ret, err := t.process(t.data)
	if err != nil {
		modules.ComputeTasks.TaskFailed(t.session, t.taskId, err)
		return
	}
	if len(t.taskId) > 0 {
		modules.ComputeTasks.TaskComplete(t.session, t.taskId, ret)
	}
}

func (t *delayTask) Dump() string {
	return fmt.Sprintf("taskId: %s data: %v", t.taskId, t.data)
}

func newDelayTask(process ProcessFunc, session *mcclient.ClientSession, taskId string, data jsonutils.JSONObject) *delayTask {
	return &delayTask{
		process: process,
		taskId:  taskId,
		session: session,
		data:    data,
	}
}

func DelayProcess(process ProcessFunc, session *mcclient.ClientSession, taskId string, data jsonutils.JSONObject) {
	task := &delayTask{
		process: process,
		taskId:  taskId,
		session: session,
		data:    data,
	}
	delayTaskWorkerMan.Run(task, nil, nil)
}
