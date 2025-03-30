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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SArchivedTaskManager struct {
	db.SLogBaseManager
	db.SStatusResourceBaseManager
}

var ArchivedTaskManager *SArchivedTaskManager

func initArchivedTaskManager() {
	ArchivedTaskManager = &SArchivedTaskManager{
		SLogBaseManager: db.NewLogBaseManager(
			SArchivedTask{},
			"archived_tasks_tbl",
			"archivedtask",
			"archivedtasks",
			"start_at",
			consts.OpsLogWithClickhouse,
		),
	}
	ArchivedTaskManager.SetVirtualObject(ArchivedTaskManager)
}

type SArchivedTask struct {
	db.SLogBase

	TaskId string `width:"36" charset:"ascii" index:"true" list:"user"`

	STaskBase `params->list:"user" user_cred->list:"user"`

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

// 操作日志列表
func (manager *SArchivedTaskManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input apis.ArchivedTaskListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SLogBaseManager.ListItemFilter(ctx, q, userCred, input.LogBaseListInput)
	if err != nil {
		return q, errors.Wrap(err, "SLogBaseManager.ListItemFilter")
	}
	q, err = manager.SStatusResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusResourceBaseListInput)
	if err != nil {
		return q, errors.Wrap(err, "SStatusResourceBaseManager.ListItemFilter")
	}

	if len(input.TaskId) > 0 {
		q = q.In("task_id", input.TaskId)
	}

	if len(input.ObjId) > 0 {
		if len(input.ObjId) == 1 {
			q = q.Contains("obj_ids", input.ObjId[0])
		} else {
			filters := make([]sqlchemy.ICondition, 0)
			for i := range input.ObjId {
				filters = append(filters, sqlchemy.Contains(q.Field("obj_ids"), input.ObjId[i]))
			}
			q = q.Filter(sqlchemy.OR(filters...))
		}
	}

	if len(input.ObjName) > 0 {
		if len(input.ObjName) == 1 {
			q = q.Contains("obj_names", input.ObjName[0])
		} else {
			filters := make([]sqlchemy.ICondition, 0)
			for i := range input.ObjName {
				filters = append(filters, sqlchemy.Contains(q.Field("obj_names"), input.ObjName[i]))
			}
			q = q.Filter(sqlchemy.OR(filters...))
		}
	}

	if len(input.ObjType) > 0 {
		q = q.In("obj_type", input.ObjType)
	}

	if len(input.TaskName) > 0 {
		q = q.In("task_name", input.TaskName)
	}

	if len(input.Stage) > 0 {
		q = q.In("stage", input.Stage)
	}

	if len(input.NotStage) > 0 {
		q = q.NotIn("stage", input.NotStage)
	}

	if len(input.ParentId) > 0 {
		q = q.In("parent_task_id", input.ParentId)
	}

	if input.IsRoot != nil {
		if *input.IsRoot {
			q = q.IsNullOrEmpty("parent_task_id")
		} else {
			q = q.IsNotEmpty("parent_task_id")
		}
	}

	if len(input.ParentTaskId) > 0 {
		q = q.Equals("parent_task_id", input.ParentTaskId)
	}

	return q, nil
}

func (manager *SArchivedTaskManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SModelBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SModelBaseManager.ListItemExportKeys")
	}
	// q, err = manager.SProjectizedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	// if err != nil {
	//	return nil, errors.Wrap(err, "SProjectizedResourceBaseManager.ListItemExportKeys")
	// }
	return q, nil
}

func (manager *SArchivedTaskManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SModelBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	// q, err = manager.SProjectizedResourceBaseManager.QueryDistinctExtraField(q, field)
	// if err == nil {
	//	return q, nil
	// }
	return q, httperrors.ErrNotFound
}

func (manager *SArchivedTaskManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeProject
}

func (manager *SArchivedTaskManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.TaskDetails {
	rows := make([]apis.TaskDetails, len(objs))
	projectIds := make([]string, 0)
	domainIds := make([]string, 0)

	for i := range objs {
		task := objs[i].(*SArchivedTask)
		if len(task.ProjectIds) > 0 {
			projectIds = append(projectIds, task.ProjectIds[0])
		} else if len(task.DomainIds) > 0 {
			domainIds = append(domainIds, task.DomainIds[0])
		}
	}
	var projectsMap map[string]db.STenant
	var domainsMap map[string]db.STenant
	if len(projectIds) > 0 {
		projectsMap = db.DefaultProjectsFetcher(ctx, projectIds, false)
	}
	if len(domainIds) > 0 {
		domainsMap = db.DefaultProjectsFetcher(ctx, domainIds, true)
	}
	for i := range objs {
		task := objs[i].(*SArchivedTask)
		if len(task.ProjectIds) > 0 {
			rows[i].ProjectId = task.ProjectIds[0]
			rows[i].DomainId = task.DomainIds[0]
			if proj, ok := projectsMap[task.ProjectIds[0]]; ok {
				rows[i].Project = proj.Name
				rows[i].ProjectDomain = proj.Domain
			}
		} else if len(task.DomainIds) > 0 {
			rows[i].DomainId = task.DomainIds[0]
			if dom, ok := domainsMap[task.DomainIds[0]]; ok {
				rows[i].ProjectDomain = dom.Name
			}
		}
	}
	return rows
}

func (manager *SArchivedTaskManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.ArchivedTaskListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SModelBaseManager.OrderByExtraFields(ctx, q, userCred, query.ModelBaseListInput)
	if err != nil {
		return q, errors.Wrap(err, "SModelBaseManager.OrderByExtraField")
	}
	return q, nil
}
