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
