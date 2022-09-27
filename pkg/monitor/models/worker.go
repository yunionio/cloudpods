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

package models

import (
	"fmt"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

var modelTaskMan *appsrv.SWorkerManager

func GetModelTaskManager() *appsrv.SWorkerManager {
	if modelTaskMan == nil {
		modelTaskMan = appsrv.NewWorkerManager("ModelTaskWorkerManager", 4, 1024, true)
	}
	return modelTaskMan
}

type modelTask struct {
	name           string
	object         db.IStandaloneModel
	backgroundFunc func() error
}

func newModelTask(name string, obj db.IStandaloneModel, f func() error) appsrv.IWorkerTask {
	return &modelTask{
		name:           name,
		object:         obj,
		backgroundFunc: f,
	}
}

func (t *modelTask) getObjectDesc() string {
	return fmt.Sprintf("%s(%s)", t.object.GetName(), t.object.GetId())
}

func (t *modelTask) Run() {
	if err := t.backgroundFunc(); err != nil {
		log.Errorf("execute %s for model %s: %v", t.name, t.getObjectDesc(), err)
	}
}

func (t *modelTask) Dump() string {
	return fmt.Sprintf("Task %s for model %s", t.name, t.getObjectDesc())
}

func RunModelTask(name string, obj db.IStandaloneModel, f func() error) {
	GetModelTaskManager().Run(newModelTask(name, obj, f), nil, nil)
}
