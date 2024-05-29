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
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type STaskObjectManager struct {
	db.SModelBaseManager
	db.SProjectizedResourceBaseManager
}

var TaskObjectManager *STaskObjectManager

func init() {
	TaskObjectManager = &STaskObjectManager{SModelBaseManager: db.NewModelBaseManager(STaskObject{}, "taskobjects_tbl", "taskobject", "taskobjects")}
}

type STaskObject struct {
	db.SModelBase
	db.SProjectizedResourceBase

	TaskId string `width:"36" charset:"ascii" nullable:"false" primary:"true" index:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=False, primary_key=True, index=True)
	ObjId  string `width:"36" charset:"ascii" nullable:"false" primary:"true"`              // Column(VARCHAR(36, charset='ascii'), nullable=False, primary_key=True)
	Object string `json:"object" width:"128" charset:"utf8" nullable:"false" list:"user"`
}

func (manager *STaskObjectManager) GetObjectIds(task *STask) []string {
	ret := make([]string, 0)
	taskobjs := manager.Query().SubQuery()
	q := taskobjs.Query(taskobjs.Field("obj_id")).Equals("task_id", task.Id)
	rows, err := q.Rows()
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("TaskObjectManager GetObjectIds fail %s", err)
		}
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var objId string
		err = rows.Scan(&objId)
		if err != nil {
			log.Errorf("TaskObjectManager GetObjects fetch row fail %s", err)
			return nil
		}
		ret = append(ret, objId)
	}
	return ret
}

func (manager *STaskObjectManager) GetObjectNames(task *STask) []string {
	ret := make([]string, 0)
	taskobjs := manager.Query().SubQuery()
	q := taskobjs.Query(taskobjs.Field("object")).Equals("task_id", task.Id)
	rows, err := q.Rows()
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("TaskObjectManager GetObjectIds fail %s", err)
		}
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var objId string
		err = rows.Scan(&objId)
		if err != nil {
			log.Errorf("TaskObjectManager GetObjects fetch row fail %s", err)
			return nil
		}
		ret = append(ret, objId)
	}
	return ret
}

func (manager *STaskObjectManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return manager.SProjectizedResourceBaseManager.FetchOwnerId(ctx, data)
}

func (manager *STaskObjectManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	return manager.SProjectizedResourceBaseManager.FilterByOwner(ctx, q, man, userCred, owner, scope)
}

func (manager *STaskObjectManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SModelBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SModelBaseManager.ListItemExportKeys")
	}
	q, err = manager.SProjectizedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SProjectizedResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (manager *STaskObjectManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SModelBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SProjectizedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *STaskObjectManager) ResourceScope() rbacscope.TRbacScope {
	return manager.SProjectizedResourceBaseManager.ResourceScope()
}

func (taskObj *STaskObject) GetOwnerId() mcclient.IIdentityProvider {
	return taskObj.SProjectizedResourceBase.GetOwnerId()
}
