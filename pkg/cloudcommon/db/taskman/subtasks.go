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
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

const (
	SUBTASK_INIT = "init"
	SUBTASK_FAIL = "fail"
	SUBTASK_SUCC = "succ"
)

type SSubTaskmanager struct {
	db.SModelBaseManager
}

var SubTaskManager *SSubTaskmanager

func init() {
	SubTaskManager = &SSubTaskmanager{SModelBaseManager: db.NewModelBaseManager(
		SSubTask{},
		"subtasks_tbl",
		"subtask",
		"subtasks",
	)}
	SubTaskManager.SetVirtualObject(SubTaskManager)
	SubTaskManager.TableSpec().AddIndex(true, "task_id", "stage", "subtask_id", "status")
}

type SSubTask struct {
	db.SModelBase

	TaskId    string `width:"36" charset:"ascii" nullable:"false" primary:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=False, primary_key=True)
	Stage     string `width:"64" charset:"ascii" nullable:"false" primary:"true"` // Column(VARCHAR(64, charset='ascii'), nullable=False, primary_key=True)
	SubtaskId string `width:"36" charset:"ascii" nullable:"false" primary:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=False, primary_key=True)
	Status    string `width:"36" charset:"ascii" nullable:"false" default:"init"` // Column(VARCHAR(36, charset='ascii'), nullable=False, default=SUBTASK_INIT)
	Result    string `length:"medium" charset:"utf8" nullable:"true"`             // Column(MEDIUMTEXT(charset='ascii'), nullable=True)
}

func (manager *SSubTaskmanager) GetSubTask(ptaskId string, subtaskId string) *SSubTask {
	subtask := SSubTask{}
	subtask.SetModelManager(manager, &subtask)
	err := manager.Query().Equals("task_id", ptaskId).Equals("subtask_id", subtaskId).First(&subtask)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("GetSubTask fail %s %s %s", err, ptaskId, subtaskId)
		}
		return nil
	}
	return &subtask
}

func (manager *SSubTaskmanager) getTotalSubtasksQuery(taskId string, stage string, status string) *sqlchemy.SQuery {
	q := manager.Query().Equals("task_id", taskId)
	if len(stage) > 0 {
		q = q.Equals("stage", stage)
	}
	if len(status) > 0 {
		q = q.Equals("status", status)
	}
	return q
}

func (manager *SSubTaskmanager) GetSubtasks(taskId string, stage string, status string) []SSubTask {
	subtasks := make([]SSubTask, 0)
	q := manager.getTotalSubtasksQuery(taskId, stage, status)
	err := db.FetchModelObjects(manager, q, &subtasks)
	if err != nil {
		log.Errorf("GetInitSubtasks fail %s", err)
		return nil
	}
	return subtasks
}

func (manager *SSubTaskmanager) GetSubtasksCount(taskId string, stage string, status string) (int, error) {
	q := manager.getTotalSubtasksQuery(taskId, stage, status)
	return q.CountWithError()
}

func (manager *SSubTaskmanager) GetTotalSubtasks(taskId string, stage string) []SSubTask {
	return manager.GetSubtasks(taskId, stage, "")
}

func (manager *SSubTaskmanager) GetTotalSubtasksCount(taskId string, stage string) (int, error) {
	return manager.GetSubtasksCount(taskId, stage, "")
}

func (manager *SSubTaskmanager) GetInitSubtasks(taskId string, stage string) []SSubTask {
	return manager.GetSubtasks(taskId, stage, SUBTASK_INIT)
}

func (manager *SSubTaskmanager) GetInitSubtasksCount(taskId string, stage string) (int, error) {
	return manager.GetSubtasksCount(taskId, stage, SUBTASK_INIT)
}

func (self *SSubTask) SaveResults(failed bool, result jsonutils.JSONObject) error {
	_, err := db.Update(self, func() error {
		if failed {
			self.Status = SUBTASK_FAIL
		} else {
			self.Status = SUBTASK_SUCC
		}
		self.Result = result.String()
		return nil
	})
	if err != nil {
		log.Errorf("SaveUpdate save update fail %s", err)
		return err
	}
	return nil
}
