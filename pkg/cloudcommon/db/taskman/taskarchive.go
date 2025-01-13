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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SArchivedTaskManager struct {
	db.SLogBaseManager
	db.SStatusResourceBaseManager
}

var ArchivedTaskManager *SArchivedTaskManager

func InitArchivedTaskManager() {
	ArchivedTaskManager = &SArchivedTaskManager{
		SLogBaseManager: db.NewLogBaseManager(
			SArchivedTask{},
			"archived_tasks_tbl",
			"achivedtask",
			"achivedtasks",
			"start_at",
			consts.OpsLogWithClickhouse,
		),
	}
	ArchivedTaskManager.SetVirtualObject(ArchivedTaskManager)
}

type SArchivedTask struct {
	db.SLogBase

	TaskId string `width:"36" charset:"ascii" index:"true" list:"user"`

	STaskBase

	ObjIds     []string `charset:"ascii" list:"user"`
	ObjNames   []string `charset:"utf8" list:"user"`
	ProjectIds []string `charset:"ascii" list:"user"`
	DomainIds  []string `charset:"ascii" list:"user"`

	SubTaskCount   int `json:"sub_task_count" list:"user"`
	FailSubTaskCnt int `json:"fail_sub_task_cnt" list:"user"`
	SuccSubTaskCnt int `json:"succ_sub_task_cnt" list:"user"`
}

func (manager *SArchivedTaskManager) NamespaceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeProject
}

func (manager *SArchivedTaskManager) Insert(ctx context.Context, task *STask) error {
	archivedTask := SArchivedTask{}
	archivedTask.SetModelManager(manager, &archivedTask)
	archivedTask.TaskId = task.Id
	archivedTask.STaskBase = task.STaskBase
	archivedTask.ObjIds = TaskObjectManager.GetObjectIds(task)
	archivedTask.ObjNames = TaskObjectManager.GetObjectNames(task)
	archivedTask.FailSubTaskCnt, _ = SubTaskManager.GetSubtasksCount(task.Id, "", SUBTASK_FAIL)
	archivedTask.SuccSubTaskCnt, _ = SubTaskManager.GetSubtasksCount(task.Id, "", SUBTASK_SUCC)
	archivedTask.SubTaskCount = archivedTask.FailSubTaskCnt + archivedTask.SuccSubTaskCnt
	archivedTask.ProjectIds = TaskObjectManager.GetProjectIds(task)
	archivedTask.DomainIds = TaskObjectManager.GetDomainIds(task)

	err := manager.TableSpec().Insert(ctx, &archivedTask)
	if err != nil {
		return errors.Wrap(err, "Insert")
	}
	return nil
}

func (task *SArchivedTask) GetOwnerId() mcclient.IIdentityProvider {
	return nil
}

func (manager *SArchivedTaskManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	return q
}

func (manager *SArchivedTaskManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return nil, nil
}
