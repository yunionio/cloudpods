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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/util/panicutils"
)

var taskWorkMan *appsrv.SWorkerManager
var taskWorkerTable map[string]*appsrv.SWorkerManager

func init() {
	taskWorkMan = appsrv.NewWorkerManager("TaskWorkerManager", 4, 1024, true)
	taskWorkerTable = make(map[string]*appsrv.SWorkerManager)
}

func runTask(taskId string, data jsonutils.JSONObject) {
	taskName := TaskManager.getTaskName(taskId)
	if len(taskName) == 0 {
		log.Errorf("no such task??? task_id=%s", taskId)
		return
	}
	worker := taskWorkMan
	if workerMan, ok := taskWorkerTable[taskName]; ok {
		worker = workerMan
	}
	worker.Run(func() {
		TaskManager.execTask(taskId, data)
	}, nil, func(err error) {
		panicutils.SendPanicMessage(context.TODO(), err)
	})
}
