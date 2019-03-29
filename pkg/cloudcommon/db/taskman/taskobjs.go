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

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type STaskObjectManager struct {
	db.SModelBaseManager
}

var TaskObjectManager *STaskObjectManager

func init() {
	TaskObjectManager = &STaskObjectManager{SModelBaseManager: db.NewModelBaseManager(STaskObject{}, "taskobjects_tbl", "taskobject", "taskobjects")}
}

type STaskObject struct {
	db.SModelBase

	TaskId string `width:"36" charset:"ascii" nullable:"false" primary:"true" index:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=False, primary_key=True, index=True)
	ObjId  string `width:"36" charset:"ascii" nullable:"false" primary:"true"`              // Column(VARCHAR(36, charset='ascii'), nullable=False, primary_key=True)
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
