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

func newDelayTask(process ProcessFunc, session *mcclient.ClientSession, taskId string, data jsonutils.JSONObject) *delayTask {
	return &delayTask{
		process: process,
		taskId:  taskId,
		session: session,
		data:    data,
	}
}

func DelayProcess(process ProcessFunc, session *mcclient.ClientSession, taskId string, data jsonutils.JSONObject) {
	delayTaskWorkerMan.Run(func() {
		executeDelayProcess(newDelayTask(process, session, taskId, data))
	}, nil, nil)
}

func executeDelayProcess(task *delayTask) {
	ret, err := task.process(task.data)
	if err != nil {
		modules.ComputeTasks.TaskFailed(task.session, task.taskId, err)
		return
	}
	modules.ComputeTasks.TaskComplete(task.session, task.taskId, ret)
}
