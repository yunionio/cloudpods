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
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

/*type TaskStageFunc func(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject)
type BatchTaskStageFunc func(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject)
*/

type IBatchTask interface {
	OnInit(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject)
	ScheduleRun(data jsonutils.JSONObject) error
}

type ISingleTask interface {
	OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject)
	ScheduleRun(data jsonutils.JSONObject) error
}

var ITaskType reflect.Type
var IBatchTaskType reflect.Type

var taskTable map[string]reflect.Type

func init() {
	ITaskType = reflect.TypeOf((*ISingleTask)(nil)).Elem()
	IBatchTaskType = reflect.TypeOf((*IBatchTask)(nil)).Elem()

	taskTable = make(map[string]reflect.Type)
}

func RegisterTaskAndWorker(task interface{}, workerMan *appsrv.SWorkerManager) {
	taskName := gotypes.GetInstanceTypeName(task)
	if _, ok := taskTable[taskName]; ok {
		log.Fatalf("Task %s already registered!", taskName)
	}
	taskType := reflect.Indirect(reflect.ValueOf(task)).Type()
	taskTable[taskName] = taskType
	// log.Infof("Task %s registerd", taskName)
	if workerMan != nil {
		taskWorkerTable[taskName] = workerMan
	}
}

func RegisterTask(task interface{}) {
	RegisterTaskAndWorker(task, nil)
}

func isTaskExist(taskName string) bool {
	_, ok := taskTable[taskName]
	return ok
}
