package taskman

import (
	"context"
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

/*type TaskStageFunc func(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject)
type BatchTaskStageFunc func(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject)
*/

type IBatchTask interface {
	OnInit(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject)
	ScheduleRun(data jsonutils.JSONObject)
}

type ISingleTask interface {
	OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject)
	ScheduleRun(data jsonutils.JSONObject)
}

var ITaskType reflect.Type
var IBatchTaskType reflect.Type

var taskTable map[string]reflect.Type

func init() {
	ITaskType = reflect.TypeOf((*ISingleTask)(nil)).Elem()
	IBatchTaskType = reflect.TypeOf((*IBatchTask)(nil)).Elem()

	taskTable = make(map[string]reflect.Type)
}

func RegisterTask(task interface{}) {
	taskName := gotypes.GetInstanceTypeName(task)
	if _, ok := taskTable[taskName]; ok {
		log.Fatalf("Task %s already registered!", taskName)
	}
	taskType := reflect.Indirect(reflect.ValueOf(task)).Type()
	taskTable[taskName] = taskType
	// log.Infof("Task %s registerd", taskName)
}

func isTaskExist(taskName string) bool {
	_, ok := taskTable[taskName]
	return ok
}
