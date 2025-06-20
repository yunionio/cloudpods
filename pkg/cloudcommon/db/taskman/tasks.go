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
	"fmt"
	"net/http"
	"reflect"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/yunionconf"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	PARENT_TASK_ID_KEY     = "parent_task_id"
	PENDING_USAGE_KEY      = "__pending_usage__"
	PARENT_TASK_NOTIFY_KEY = "__parent_task_notifyurl"
	REQUEST_CONTEXT_KEY    = "__request_context"

	TASK_STAGE_FAILED   = "failed"
	TASK_STAGE_COMPLETE = "complete"

	MAX_REMOTE_NOTIFY_TRIES = 5

	MULTI_OBJECTS_ID = "[--MULTI_OBJECTS--]"

	TASK_INIT_STAGE = "on_init"

	CONVERT_TASK = "convert_task"

	LANG = "lang"

	taskStatusDone    = "done"
	TASK_STATUS_QUEUE = "queue"
)

type STaskManager struct {
	db.SModelBaseManager
	db.SProjectizedResourceBaseManager
	db.SStatusResourceBaseManager
}

var TaskManager *STaskManager
var userCredWidthLimit = 0

func init() {
	TaskManager = &STaskManager{
		SModelBaseManager: db.NewModelBaseManager(STask{}, "tasks_tbl", "task", "tasks"),
	}
	TaskManager.SetVirtualObject(TaskManager)
	if field, ok := reflect.TypeOf(&STask{}).Elem().FieldByName("UserCred"); ok {
		if widthStr := field.Tag.Get(sqlchemy.TAG_WIDTH); len(widthStr) > 0 {
			userCredWidthLimit, _ = strconv.Atoi(widthStr)
		}
	}
	TaskManager.TableSpec().AddIndex(true, "id", "created_at", "parent_task_id", "stage")
}

type STask struct {
	db.SModelBase

	// 资源创建时间
	CreatedAt time.Time `nullable:"false" created_at:"true" index:"true" get:"user" list:"user" json:"created_at"`
	// 资源更新时间
	UpdatedAt time.Time `nullable:"false" updated_at:"true" list:"user" json:"updated_at"`
	// 资源被更新次数
	UpdateVersion int `default:"0" nullable:"false" auto_version:"true" list:"user" json:"update_version"`

	Id string `width:"36" charset:"ascii" primary:"true" list:"user"` // Column(VARCHAR(36, charset='ascii'), primary_key=True, default=get_uuid)

	STaskBase

	db.SProjectizedResourceBase

	taskObject  db.IStandaloneModel   `ignore:"true"`
	taskObjects []db.IStandaloneModel `ignore:"true"`

	SubTaskCount   int `ignore:"true" json:"sub_task_count"`
	FailSubTaskCnt int `ignore:"true" json:"fail_sub_task_cnt"`
	SuccSubTaskCnt int `ignore:"true" json:"succ_sub_task_cnt"`
}

func (manager *STaskManager) CreateByInsertOrUpdate() bool {
	return false
}

func (manager *STaskManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (manager *STaskManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (manager *STaskManager) FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q.Equals("id", idStr)
}

func (manager *STaskManager) FilterByNotId(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q.NotEquals("id", idStr)
}

func (manager *STaskManager) FilterByName(q *sqlchemy.SQuery, name string) *sqlchemy.SQuery {
	return q.Equals("id", name)
}

func (manager *STaskManager) PerformAction(ctx context.Context, userCred mcclient.TokenCredential, taskId string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := runTask(taskId, data)
	if err != nil {
		return nil, errors.Wrapf(err, "runTask")
	}
	resp := jsonutils.NewDict()
	resp.Add(jsonutils.NewString("ok"), "result")
	return resp, nil
}

func (manager *STask) PreCheckPerformAction(
	ctx context.Context, userCred mcclient.TokenCredential,
	action string, query jsonutils.JSONObject, data jsonutils.JSONObject,
) error {
	return nil
}

func (task *STask) GetOwnerId() mcclient.IIdentityProvider {
	return task.SProjectizedResourceBase.GetOwnerId()
}

func (manager *STaskManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	taskQ := TaskObjectManager.Query("task_id")
	taskQ = taskQ.Snapshot()
	taskQ = manager.SProjectizedResourceBaseManager.FilterByOwner(ctx, taskQ, man, userCred, owner, scope)
	if taskQ.IsAltered() {
		taskSubQ := taskQ.SubQuery()
		q = q.Join(taskSubQ, sqlchemy.Equals(q.Field("id"), taskSubQ.Field("task_id")))
	}

	return q
}

func (manager *STaskManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return manager.SProjectizedResourceBaseManager.FetchOwnerId(ctx, data)
}

func (manager *STaskManager) FetchTaskById(taskId string) *STask {
	return manager.fetchTask(taskId)
}

func (task *STask) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	return httperrors.NewForbiddenError("forbidden")
}

func (task *STask) ValidateUpdateCondition(ctx context.Context) error {
	return httperrors.NewForbiddenError("forbidden")
}

func (task *STask) BeforeInsert() {
	if len(task.Id) == 0 {
		task.Id = stringutils.UUID4()
	}
}

func (task *STask) GetId() string {
	return task.Id
}

func (task *STask) GetName() string {
	return task.TaskName
}

func (task *STask) saveStartAt() {
	if !task.StartAt.IsZero() {
		return
	}
	_, err := db.Update(task, func() error {
		task.StartAt = timeutils.UtcNow()
		return nil
	})
	if err != nil {
		log.Errorf("task %s save start_at fail: %s", task.String(), err)
	}
}

func fetchTaskParams(
	ctx context.Context,
	taskName string,
	taskData *jsonutils.JSONDict,
	parentTaskId string,
	parentTaskNotifyUrl string,
	pendingUsages []quotas.IQuota,
) *jsonutils.JSONDict {
	var data *jsonutils.JSONDict
	if taskData != nil {
		excludeKeys := []string{
			PARENT_TASK_ID_KEY, PARENT_TASK_NOTIFY_KEY, PENDING_USAGE_KEY,
		}
		for i := 1; taskData.Contains(pendingUsageKey(i)); i += 1 {
			excludeKeys = append(excludeKeys, pendingUsageKey(i))
		}
		data = taskData.CopyExcludes(excludeKeys...)
	} else {
		data = jsonutils.NewDict()
	}
	reqContext := appctx.FetchAppContextData(ctx)
	if !reqContext.IsZero() {
		data.Add(jsonutils.Marshal(&reqContext), REQUEST_CONTEXT_KEY)
	}
	if len(parentTaskId) > 0 || len(parentTaskNotifyUrl) > 0 {
		if len(parentTaskId) > 0 {
			data.Add(jsonutils.NewString(parentTaskId), PARENT_TASK_ID_KEY)
		}
		if len(parentTaskNotifyUrl) > 0 {
			data.Add(jsonutils.NewString(parentTaskNotifyUrl), PARENT_TASK_NOTIFY_KEY)
			log.Infof("%s notify parent url: %s", taskName, parentTaskNotifyUrl)
		}
	} else {
		if !reqContext.IsZero() {
			if len(reqContext.TaskId) > 0 && len(reqContext.TaskNotifyUrl) == 0 {
				data.Add(jsonutils.NewString(reqContext.TaskId), PARENT_TASK_ID_KEY)
			}
			if len(reqContext.TaskNotifyUrl) > 0 {
				data.Add(jsonutils.NewString(reqContext.TaskNotifyUrl), PARENT_TASK_NOTIFY_KEY)
				log.Infof("%s notify parent url: %s", taskName, reqContext.TaskNotifyUrl)
			}
		}
	}
	if len(pendingUsages) > 0 {
		for i := range pendingUsages {
			pendingUsage := pendingUsages[i]
			if gotypes.IsNil(pendingUsage) {
				continue
			}
			key := pendingUsageKey(i)
			data.Add(jsonutils.Marshal(pendingUsage), key)
		}
	}
	return data
}

func (manager *STaskManager) NewTask(
	ctx context.Context,
	taskName string,
	obj db.IStandaloneModel,
	userCred mcclient.TokenCredential,
	taskData *jsonutils.JSONDict,
	parentTaskId string,
	parentTaskNotifyUrl string,
	pendingUsage ...quotas.IQuota,
) (*STask, error) {
	if userCredWidthLimit > 0 && len(userCred.String()) > userCredWidthLimit {
		return nil, fmt.Errorf("Too many permissions for user %s", userCred.GetUserName())
	}

	if !isTaskExist(taskName) {
		return nil, fmt.Errorf("task %s not found", taskName)
	}

	data := fetchTaskParams(ctx, taskName, taskData, parentTaskId, parentTaskNotifyUrl, pendingUsage)
	task := &STask{
		STaskBase: STaskBase{
			ObjType:      obj.Keyword(),
			ObjId:        obj.GetId(),
			Object:       obj.GetName(),
			TaskName:     taskName,
			UserCred:     userCred,
			Params:       data,
			Stage:        TASK_INIT_STAGE,
			ParentTaskId: parentTaskId,
		},
	}

	task.SetModelManager(manager, task)
	err := manager.TableSpec().Insert(ctx, task)
	if err != nil {
		log.Errorf("Task insert error %s", err)
		return nil, err
	}
	task.SetProgressAndStatus(0, TASK_STATUS_QUEUE)

	{
		to, err := TaskObjectManager.insertObject(ctx, task.Id, obj)
		if err != nil {
			log.Errorf("Taskobject insert error %s", err)
			return nil, errors.Wrap(err, "TaskObjectManager.insertObject")
		}
		db.Update(task, func() error {
			task.ProjectId = to.ProjectId
			task.DomainId = to.DomainId
			return nil
		})
	}

	parentTask := task.GetParentTask()
	if parentTask != nil {
		st := &SSubTask{TaskId: parentTask.Id, Stage: parentTask.Stage, SubtaskId: task.Id}
		st.SetModelManager(SubTaskManager, st)
		err := SubTaskManager.TableSpec().Insert(ctx, st)
		if err != nil {
			log.Errorf("Subtask insert error %s", err)
			return nil, err
		}
	}
	return task, nil
}

func (manager *STaskManager) NewParallelTask(
	ctx context.Context,
	taskName string,
	objs []db.IStandaloneModel,
	userCred mcclient.TokenCredential,
	taskData *jsonutils.JSONDict,
	parentTaskId string,
	parentTaskNotifyUrl string,
	pendingUsage ...quotas.IQuota,
) (*STask, error) {
	if !isTaskExist(taskName) {
		return nil, fmt.Errorf("task %s not found", taskName)
	}

	if len(objs) == 0 {
		return nil, fmt.Errorf("failed to do task %s with zero objs", taskName)
	}

	log.Debugf("number of objs: %d", len(objs))

	data := fetchTaskParams(ctx, taskName, taskData, parentTaskId, parentTaskNotifyUrl, pendingUsage)
	task := &STask{
		STaskBase: STaskBase{
			ObjType:      objs[0].Keyword(),
			Object:       MULTI_OBJECTS_ID,
			ObjId:        MULTI_OBJECTS_ID,
			TaskName:     taskName,
			UserCred:     userCred,
			Params:       data,
			Stage:        TASK_INIT_STAGE,
			ParentTaskId: parentTaskId,
		},
	}
	task.SetModelManager(manager, task)
	err := manager.TableSpec().Insert(ctx, task)
	if err != nil {
		log.Errorf("Task insert error %s", err)
		return nil, err
	}
	task.SetProgressAndStatus(0, TASK_STATUS_QUEUE)

	domainIds := stringutils2.NewSortedStrings(nil)
	tenantIds := stringutils2.NewSortedStrings(nil)
	for i := range objs {
		to, err := TaskObjectManager.insertObject(ctx, task.Id, objs[i])
		if err != nil {
			log.Errorf("Taskobject insert error %s", err)
			return nil, errors.Wrap(err, "insert task object")
		}
		tenantIds = tenantIds.Append(to.ProjectId)
		domainIds = domainIds.Append(to.DomainId)
	}

	db.Update(task, func() error {
		task.DomainId = strings.Join(domainIds, ",")
		task.ProjectId = strings.Join(tenantIds, ",")
		return nil
	})

	parentTask := task.GetParentTask()
	if parentTask != nil {
		st := &SSubTask{TaskId: parentTask.Id, Stage: parentTask.Stage, SubtaskId: task.Id}
		st.SetModelManager(SubTaskManager, st)
		err := SubTaskManager.TableSpec().Insert(ctx, st)
		if err != nil {
			log.Errorf("Subtask insert error %s", err)
			return nil, err
		}
	}
	return task, nil
}

func (manager *STaskManager) fetchTask(idStr string) *STask {
	iTask, err := db.NewModelObject(manager)
	if err != nil {
		log.Errorf("New task object fail: %s", err)
		return nil
	}
	err = manager.Query().Equals("id", idStr).First(iTask)
	if err != nil {
		log.Errorf("GetTask %s fail: %s", idStr, err)
		return nil
	}
	task := iTask.(*STask)
	task.fixParams()
	return task
}

func (task *STask) fixParams() {
	if task.Params == nil {
		task.Params = jsonutils.NewDict()
	}
}

func (manager *STaskManager) execTask(taskId string, data jsonutils.JSONObject) {
	baseTask := manager.fetchTask(taskId)
	if baseTask == nil {
		return
	}
	manager.execTaskObject(baseTask, data)
}

func (manager *STaskManager) execTaskObject(baseTask *STask, data jsonutils.JSONObject) {
	taskType, ok := taskTable[baseTask.TaskName]
	if !ok {
		log.Errorf("Cannot find task %s", baseTask.TaskName)
		return
	}
	log.Debugf("Do task %s(%s) with data %s at stage %s", taskType, baseTask.Id, data, baseTask.Stage)
	taskValue := reflect.New(taskType)
	if taskValue.Type().Implements(ITaskType) {
		execITask(taskValue, baseTask, data, false)
	} else if taskValue.Type().Implements(IBatchTaskType) {
		execITask(taskValue, baseTask, data, true)
	} else {
		log.Errorf("Unsupported task type?? %s", taskValue.Type())
	}
}

type sSortedObjects []db.IStandaloneModel

func (a sSortedObjects) Len() int           { return len(a) }
func (a sSortedObjects) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a sSortedObjects) Less(i, j int) bool { return a[i].GetId() < a[j].GetId() }

func execITask(taskValue reflect.Value, task *STask, odata jsonutils.JSONObject, isMulti bool) {
	ctxData := task.GetRequestContext()
	ctx := ctxData.GetContext()

	task.saveStartAt()

	taskFailed := false

	var data jsonutils.JSONObject
	if odata != nil {
		switch dictdata := odata.(type) {
		case *jsonutils.JSONDict:
			taskStatus, _ := odata.GetString("__status__")
			if len(taskStatus) > 0 && taskStatus != "OK" {
				taskFailed = true
				dictdata.Set("__stage__", jsonutils.NewString(task.Stage))
				if !dictdata.Contains("__reason__") {
					reasonJson := dictdata.CopyExcludes("__status__", "__stage__")
					dictdata.Set("__reason__", reasonJson)
				}
				/*if vdata, ok := data.(*jsonutils.JSONDict); ok {
					reason, err := vdata.Get("__reason__") // only dict support Get
					if err != nil {
						reason = jsonutils.NewString(fmt.Sprintf("Task failed due to unknown remote errors! %s", odata))
						vdata.Set("__reason__", reason)
					}
				}*/
			}
			data = dictdata
		default:
			data = odata
		}
	} else {
		data = jsonutils.NewDict()
	}

	stageName := task.Stage
	if taskFailed {
		stageName = fmt.Sprintf("%sFailed", task.Stage)
		if strings.Contains(stageName, "_") {
			stageName = fmt.Sprintf("%s_failed", task.Stage)
		}
	}

	if strings.Contains(stageName, "_") {
		stageName = utils.Kebab2Camel(stageName, "_")
	}

	funcValue := taskValue.MethodByName(stageName)

	if !funcValue.IsValid() || funcValue.IsNil() {
		msg := fmt.Sprintf("Stage %s not found", stageName)
		if taskFailed {
			// failed handler is optional, ignore the error
			log.Warningf(msg)
			msg, _ = data.GetString()
		} else {
			log.Errorf(msg)
		}
		task.SetStageFailed(ctx, jsonutils.NewString(msg))
		task.SaveRequestContext(&ctxData)
		return
	}

	objManager := db.GetModelManager(task.ObjType)
	if objManager == nil {
		msg := fmt.Sprintf("model %s %s(%s) not found??? ...", task.ObjType, task.Object, task.ObjId)
		log.Errorf(msg)
		task.SetStageFailed(ctx, jsonutils.NewString(msg))
		task.SaveRequestContext(&ctxData)
		return
	}
	// log.Debugf("objManager: %s", objManager)
	objResManager, ok := objManager.(db.IStandaloneModelManager)
	if !ok {
		msg := fmt.Sprintf("model %s %s(%s) is not a resource??? ...", task.ObjType, task.Object, task.ObjId)
		log.Errorf(msg)
		task.SetStageFailed(ctx, jsonutils.NewString(msg))
		task.SaveRequestContext(&ctxData)
		return
	}

	params := make([]reflect.Value, 3)
	params[0] = reflect.ValueOf(ctx)

	if isMulti {
		objIds := TaskObjectManager.GetObjectIds(task)
		objs := make([]db.IStandaloneModel, len(objIds))
		for i, objId := range objIds {
			obj, err := objResManager.FetchById(objId)
			if err != nil {
				msg := fmt.Sprintf("fail to find %s object %s", task.ObjType, objId)
				log.Errorf(msg)
				task.SetStageFailed(ctx, jsonutils.NewString(msg))
				task.SaveRequestContext(&ctxData)
				return
			}
			objs[i] = obj.(db.IStandaloneModel)
		}
		task.taskObjects = objs

		// sort objects by ids to avoid deadlock
		sort.Sort(sSortedObjects(objs))

		for i := range objs {
			lockman.LockObject(ctx, objs[i])
			defer lockman.ReleaseObject(ctx, objs[i])
		}

		params[1] = reflect.ValueOf(objs)
	} else {
		obj, err := objResManager.FetchById(task.ObjId)
		if err != nil {
			msg := fmt.Sprintf("fail to find %s object %s", task.ObjType, task.ObjId)
			log.Errorf(msg)
			task.SetStageFailed(ctx, jsonutils.NewString(msg))
			task.SaveRequestContext(&ctxData)
			return
		}
		task.taskObject = obj.(db.IStandaloneModel)

		lockman.LockObject(ctx, obj)
		defer lockman.ReleaseObject(ctx, obj)

		params[1] = reflect.ValueOf(obj)
	}

	params[2] = reflect.ValueOf(data)

	filled := reflectutils.FillEmbededStructValue(taskValue.Elem(), reflect.Indirect(reflect.ValueOf(task)))
	if !filled {
		log.Errorf("Cannot locate baseTask embedded struct, give up...")
		return
	}

	defer func() {
		if r := recover(); r != nil {
			// call set stage failed, should not call task.SetStageFailed
			// func SetStageFailed may be overloading
			yunionconf.BugReport.SendBugReport(ctx, version.GetShortString(), string(debug.Stack()), errors.Errorf("%s", r))
			log.Errorf("Task %s PANIC on stage %s: %v \n%s", task.TaskName, stageName, r, debug.Stack())
			SetStageFailedFuncValue := taskValue.MethodByName("SetStageFailed")
			SetStageFailedFuncValue.Call(
				[]reflect.Value{
					reflect.ValueOf(ctx),
					reflect.ValueOf(jsonutils.NewString(fmt.Sprintf("%v", r))),
				},
			)
			obj, err := objResManager.FetchById(task.ObjId)
			if err != nil {
				return
			}
			statusObj, ok := obj.(db.IStatusStandaloneModel)
			if ok {
				db.StatusBaseSetStatus(ctx, statusObj, task.GetUserCred(), apis.STATUS_UNKNOWN, fmt.Sprintf("%v", r))
			}
			notes := map[string]interface{}{
				"Stack":   string(debug.Stack()),
				"Version": version.GetShortString(),
				"Task":    task.TaskName,
				"Stage":   stageName,
				"Message": fmt.Sprintf("%v", r),
			}
			logclient.AddSimpleActionLog(obj, logclient.ACT_PANIC, notes, task.GetUserCred(), false)
		}
	}()

	log.Debugf("Call %s(%s) %s %#v", task.TaskName, task.Id, stageName, params)
	funcValue.Call(params)

	// call save request context
	saveRequestContextFuncValue := taskValue.MethodByName("SaveRequestContext")
	saveRequestContextFuncValue.Call([]reflect.Value{reflect.ValueOf(&ctxData)})
}

func (task *STask) ScheduleRun(data jsonutils.JSONObject) error {
	return runTask(task.Id, data)
}

func (task *STask) IsSubtask() bool {
	return task.HasParentTask()
}

func (task *STask) GetParentTaskId() string {
	if len(task.ParentTaskId) > 0 {
		return task.ParentTaskId
	}
	parentTaskId, _ := task.Params.GetString(PARENT_TASK_ID_KEY)
	if len(parentTaskId) > 0 {
		return parentTaskId
	}
	return ""
}

func (task *STask) HasParentTask() bool {
	parentTaskId := task.GetParentTaskId()
	if len(parentTaskId) > 0 {
		return true
	}
	return false
}

func (task *STask) GetParentTask() *STask {
	parentTaskId := task.GetParentTaskId()
	if len(parentTaskId) > 0 {
		return TaskManager.fetchTask(parentTaskId)
	}
	return nil
}

func (task *STask) GetRequestContext() appctx.AppContextData {
	ctxData := appctx.AppContextData{}
	if task.Params != nil {
		ctxJson, _ := task.Params.Get(REQUEST_CONTEXT_KEY)
		if ctxJson != nil {
			ctxJson.Unmarshal(&ctxData)
		}
	}
	return ctxData
}

func (task *STask) SaveRequestContext(data *appctx.AppContextData) {
	jsonData := jsonutils.Marshal(data)
	log.Debugf("SaveRequestContext %s(%s) %s param %s", task.TaskName, task.Id, jsonData, task.Params)
	_, err := db.Update(task, func() error {
		params := task.Params.CopyExcludes(REQUEST_CONTEXT_KEY)
		params.Add(jsonData, REQUEST_CONTEXT_KEY)
		task.Params = params
		task.EndAt = timeutils.UtcNow()
		return nil
	})
	log.Debugf("Params: %s(%s) %s", task.TaskName, task.Id, task.Params)
	if err != nil {
		log.Errorf("save_request_context fail %s", err)
	}
}

func (task *STask) SaveParams(data *jsonutils.JSONDict) error {
	return task.SetStage("", data)
}

func (task *STask) SetStage(stageName string, data *jsonutils.JSONDict) error {
	_, err := db.Update(task, func() error {
		params := jsonutils.NewDict()
		params.Update(task.Params)
		if data != nil {
			params.Update(data)
		}
		if len(stageName) > 0 {
			stages, _ := params.Get("__stages")
			if stages == nil {
				stages = jsonutils.NewArray()
				params.Add(stages, "__stages")
			}
			stageList := stages.(*jsonutils.JSONArray)
			stageData := jsonutils.NewDict()
			stageData.Add(jsonutils.NewString(task.Stage), "name")
			stageData.Add(jsonutils.NewTimeString(time.Now()), "complete_at")
			stageList.Add(stageData)
			task.Stage = stageName
		}
		task.Params = params
		return nil
	})
	if err != nil {
		log.Errorf("Task %s(%s) set_stage %s fail %s", task.TaskName, task.Id, stageName, err)
	}
	return err
}

func (task *STask) GetObjectIdStr() string {
	if task.ObjId == MULTI_OBJECTS_ID {
		return strings.Join(TaskObjectManager.GetObjectIds(task), ",")
	} else {
		return task.ObjId
	}
}

func (task *STask) GetObjectStr() string {
	if task.ObjId == MULTI_OBJECTS_ID {
		return strings.Join(TaskObjectManager.GetObjectNames(task), ",")
	} else {
		return task.Object
	}
}

func (task *STask) SetStageComplete(ctx context.Context, data *jsonutils.JSONDict) {
	log.Infof("XXX TASK %s(%s) complete", task.TaskName, task.Id)
	task.SetStage(TASK_STAGE_COMPLETE, data)
	task.SetProgressAndStatus(100, taskStatusDone)
	if data == nil {
		data = jsonutils.NewDict()
	}
	if data.Size() == 0 {
		data.Add(jsonutils.NewString(task.GetObjectIdStr()), "id")
		data.Add(jsonutils.NewString(task.GetObjectStr()), "name")
		data.Add(jsonutils.NewString(task.ObjType), "type")
	}
	task.NotifyParentTaskComplete(ctx, data, false)
}

func (task *STask) SetStageFailed(ctx context.Context, reason jsonutils.JSONObject) {
	if task.Stage == TASK_STAGE_FAILED {
		log.Warningf("Task %s(%s) has been failed", task.TaskName, task.Id)
		return
	}
	log.Infof("XXX TASK %s(%s) failed: %s on stage %s", task.TaskName, task.Id, reason, task.Stage)
	reasonDict := jsonutils.NewDict()
	reasonDict.Add(jsonutils.NewString(task.Stage), "stage")
	if reason != nil {
		reasonDict.Add(reason, "reason")
	}
	reason = reasonDict
	prevFailed, _ := task.Params.Get("__failed_reason")
	if prevFailed != nil {
		switch prevFailed.(type) {
		case *jsonutils.JSONArray:
			prevFailed.(*jsonutils.JSONArray).Add(reason)
			reason = prevFailed
		default:
			reason = jsonutils.NewArray(prevFailed, reason)
		}
	}
	data := jsonutils.NewDict()
	data.Add(reason, "__failed_reason")
	task.SetStage(TASK_STAGE_FAILED, data)
	task.SetProgressAndStatus(100, taskStatusDone)
	task.NotifyParentTaskFailure(ctx, reason)
}

func (task *STask) NotifyParentTaskComplete(ctx context.Context, body *jsonutils.JSONDict, failed bool) {
	log.Infof("notify_parent_task_complete: %s(%s) params %s", task.TaskName, task.Id, task.Params)
	parentTaskId := task.GetParentTaskId()
	parentTaskNotify, _ := task.Params.GetString(PARENT_TASK_NOTIFY_KEY)
	if len(parentTaskId) > 0 {
		subTask := SubTaskManager.GetSubTask(parentTaskId, task.Id)
		if subTask != nil {
			subTask.SaveResults(failed, body)
		}
		func() {
			lockman.LockRawObject(ctx, "tasks", parentTaskId)
			defer lockman.ReleaseRawObject(ctx, "tasks", parentTaskId)

			pTask := TaskManager.fetchTask(parentTaskId)
			if pTask == nil {
				log.Errorf("Parent task %s not found", parentTaskId)
				return
			}
			if pTask.IsCurrentStageComplete() {
				pTask.ScheduleRun(body)
			}
		}()
	}
	if len(parentTaskNotify) > 0 {
		notifyRemoteTask(ctx, parentTaskNotify, parentTaskId, body, 0)
	}
}

func notifyRemoteTask(ctx context.Context, notifyUrl string, taskid string, body jsonutils.JSONObject, tried int) {
	client := httputils.GetDefaultClient()
	header := http.Header{}
	if len(taskid) > 0 {
		header.Set("X-Task-Id", taskid)
	}
	_, body, err := httputils.JSONRequest(client, ctx, "POST", notifyUrl, header, body, true)
	if err != nil {
		log.Errorf("notifyRemoteTask fail %s", err)
		if tried > MAX_REMOTE_NOTIFY_TRIES {
			log.Errorf("notifyRemoteTask max tried reached, give up...")
		} else {
			notifyRemoteTask(ctx, notifyUrl, taskid, body, tried+1)
		}
		return
	}
	log.Infof("Notify remote URL %s(%s) get acked: %s!", notifyUrl, taskid, body.String())
}

func (task *STask) NotifyParentTaskFailure(ctx context.Context, reason jsonutils.JSONObject) {
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString("error"), "__status__")
	body.Add(jsonutils.NewString(task.TaskName), "__task_name__")
	body.Add(reason, "__reason__")
	task.NotifyParentTaskComplete(ctx, body, true)
}

func (task *STask) IsCurrentStageComplete() bool {
	totalSubtasksCnt, _ := SubTaskManager.GetTotalSubtasksCount(task.Id, task.Stage)
	initSubtasksCnt, _ := SubTaskManager.GetInitSubtasksCount(task.Id, task.Stage)
	log.Debugf("Task %s IsCurrentStageComplete totalSubtasks %d initSubtasks %d ", task.String(), totalSubtasksCnt, initSubtasksCnt)
	task.SetProgress(float32(totalSubtasksCnt-initSubtasksCnt) * 100 / float32(totalSubtasksCnt))
	if totalSubtasksCnt > 0 && initSubtasksCnt == 0 {
		return true
	} else {
		return false
	}
}

func (task *STask) GetPendingUsage(quota quotas.IQuota, index int) error {
	key := pendingUsageKey(index)
	if task.Params.Contains(key) {
		quotaJson, err := task.Params.Get(key)
		if err != nil {
			return errors.Wrapf(err, "task.Params.Get %s", key)
		}
		err = quotaJson.Unmarshal(quota)
		if err != nil {
			return errors.Wrap(err, "quotaJson.Unmarshal")
		}
	}
	return nil
}

func pendingUsageKey(index int) string {
	key := PENDING_USAGE_KEY
	if index > 0 {
		key += "." + strconv.FormatInt(int64(index), 10)
	}
	return key
}

func (task *STask) SetPendingUsage(quota quotas.IQuota, index int) error {
	_, err := db.Update(task, func() error {
		key := pendingUsageKey(index)
		params := task.Params.CopyExcludes(key)
		params.Add(jsonutils.Marshal(quota), key)
		task.Params = params
		return nil
	})
	if err != nil {
		log.Errorf("set_pending_usage fail %s", err)
	}
	return err
}

func (task *STask) ClearPendingUsage(index int) error {
	_, err := db.Update(task, func() error {
		key := pendingUsageKey(index)
		params := task.Params.CopyExcludes(key)
		task.Params = params
		return nil
	})
	if err != nil {
		log.Errorf("clear_pending_usage fail %s", err)
	}
	return err
}

func (task *STask) GetParams() *jsonutils.JSONDict {
	return task.Params
}

func (task *STask) GetUserCred() mcclient.TokenCredential {
	return task.UserCred
}

func (task *STask) GetTaskId() string {
	return task.GetId()
}

func (task *STask) GetObject() db.IStandaloneModel {
	return task.taskObject
}

func (task *STask) GetObjects() []db.IStandaloneModel {
	return task.taskObjects
}

func (task *STask) GetTaskRequestHeader() http.Header {
	userCred := task.GetUserCred()
	if !userCred.IsValid() {
		userCred = auth.AdminCredential()
	}
	header := mcclient.GetTokenHeaders(userCred)
	header.Set(mcclient.TASK_ID, task.GetTaskId())
	if len(serviceUrl) > 0 {
		notifyUrl := fmt.Sprintf("%s/tasks/%s", serviceUrl, task.GetTaskId())
		header.Set(mcclient.TASK_NOTIFY_URL, notifyUrl)
	}
	return header
}

func (task *STask) String() string {
	return fmt.Sprintf("%s(%s,%s)", task.Id, task.TaskName, task.Stage)
}

var serviceUrl string

func SetServiceUrl(url string) {
	serviceUrl = url
}

func (task *STask) GetStartTime() time.Time {
	return task.CreatedAt
}

func (manager *STaskManager) QueryTasksOfObject(obj db.IStandaloneModel, since time.Time, isOpen *bool) *sqlchemy.SQuery {
	subq1 := manager.Query()
	{
		subq1 = subq1.Equals("obj_id", obj.GetId())
		subq1 = subq1.Equals("obj_type", obj.Keyword())
		if !since.IsZero() {
			subq1 = subq1.GE("created_at", since)
		}
		if isOpen != nil {
			if *isOpen {
				subq1 = subq1.Filter(sqlchemy.NOT(
					sqlchemy.In(subq1.Field("stage"), []string{"complete", "failed"}),
				))
			} else if !*isOpen {
				subq1 = subq1.In("stage", []string{"complete", "failed"})
			}
		}
	}

	subq2 := manager.Query()
	{
		taskObjs := TaskObjectManager.TableSpec().Instance()
		subq2 = subq2.Join(taskObjs, sqlchemy.AND(
			sqlchemy.Equals(taskObjs.Field("task_id"), subq2.Field("id")),
			sqlchemy.Equals(taskObjs.Field("obj_id"), obj.GetId()),
		))
		subq2 = subq2.Filter(sqlchemy.Equals(subq2.Field("obj_id"), MULTI_OBJECTS_ID))
		subq2 = subq2.Filter(sqlchemy.Equals(subq2.Field("obj_type"), obj.Keyword()))
		if !since.IsZero() {
			subq2 = subq2.Filter(sqlchemy.GE(subq2.Field("created_at"), since))
		}
		if isOpen != nil {
			if *isOpen {
				subq2 = subq2.Filter(sqlchemy.NOT(
					sqlchemy.In(subq2.Field("stage"), []string{"complete", "failed"}),
				))
			} else if !*isOpen {
				subq2 = subq2.In("stage", []string{"complete", "failed"})
			}
		}
	}

	// subq1 and subq2 do not overlap for the fact that they have
	// different conditions on tasks_tbl.obj_id field
	return sqlchemy.Union(subq1, subq2).Query().Desc("created_at")
}

func (manager *STaskManager) IsInTask(obj db.IStandaloneModel) bool {
	tasks, err := manager.FetchIncompleteTasksOfObject(obj)
	if err == nil && len(tasks) == 0 {
		return false
	}
	return true
}

func (manager *STaskManager) FetchIncompleteTasksOfObject(obj db.IStandaloneModel) ([]STask, error) {
	isOpen := true
	return manager.FetchTasksOfObjectLatest(obj, 1*time.Hour, &isOpen)
}

func (manager *STaskManager) FetchTasksOfObjectLatest(obj db.IStandaloneModel, interval time.Duration, isOpen *bool) ([]STask, error) {
	since := timeutils.UtcNow().Add(-1 * interval)
	return manager.FetchTasksOfObject(obj, since, isOpen)
}

func (manager *STaskManager) FetchTasksOfObject(obj db.IStandaloneModel, since time.Time, isOpen *bool) ([]STask, error) {
	q := manager.QueryTasksOfObject(obj, since, isOpen)

	tasks := make([]STask, 0)
	err := db.FetchModelObjects(manager, q, &tasks)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return tasks, nil
}

// 操作日志列表
func (manager *STaskManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input apis.TaskListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SModelBaseManager.ListItemFilter(ctx, q, userCred, input.ModelBaseListInput)
	if err != nil {
		return q, errors.Wrap(err, "SResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SStatusResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusResourceBaseListInput)
	if err != nil {
		return q, errors.Wrap(err, "SStatusResourceBaseManager.ListItemFilter")
	}

	if len(input.Id) > 0 {
		q = q.In("id", input.Id)
	}

	if len(input.ObjId) > 0 {
		taskObjQ := TaskObjectManager.Query("task_id").In("obj_id", input.ObjId).Distinct().SubQuery()
		q = q.Join(taskObjQ, sqlchemy.Equals(q.Field("id"), taskObjQ.Field("task_id")))
	}

	if len(input.ObjName) > 0 {
		q = q.In("object", input.ObjName)
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

	if input.IsComplete != nil {
		if *input.IsComplete {
			q = q.In("stage", []string{TASK_STAGE_FAILED, TASK_STAGE_COMPLETE})
		} else {
			q = q.NotIn("stage", []string{TASK_STAGE_FAILED, TASK_STAGE_COMPLETE})
		}
	}

	if input.IsInit != nil {
		if *input.IsInit {
			q = q.Equals("stage", TASK_INIT_STAGE)
		} else {
			q = q.NotEquals("stage", TASK_INIT_STAGE)
		}
	}

	if input.IsMulti != nil {
		if *input.IsMulti {
			q = q.Equals("obj_id", MULTI_OBJECTS_ID)
		} else {
			q = q.NotEquals("obj_id", MULTI_OBJECTS_ID)
		}
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

	if input.SubTask != nil && *input.SubTask {
		subSQFunc := func(status string, cntField string) *sqlchemy.SSubQuery {
			subQ := SubTaskManager.Query()
			if len(status) > 0 {
				subQ = subQ.Equals("status", status)
			}
			subQ = subQ.GroupBy(subQ.Field("task_id"))
			subQ = subQ.AppendField(subQ.Field("task_id"))
			subQ = subQ.AppendField(sqlchemy.COUNT(cntField))
			return subQ.SubQuery()
		}

		{
			subSQ := subSQFunc("", "sub_task_count")
			q = q.LeftJoin(subSQ, sqlchemy.Equals(subSQ.Field("task_id"), q.Field("id")))
			q = q.AppendField(subSQ.Field("sub_task_count"))
		}

		{
			failSubSQ := subSQFunc(SUBTASK_FAIL, "fail_sub_task_cnt")
			q = q.LeftJoin(failSubSQ, sqlchemy.Equals(failSubSQ.Field("task_id"), q.Field("id")))
			q = q.AppendField(failSubSQ.Field("fail_sub_task_cnt"))
		}

		{
			succSubSQ := subSQFunc(SUBTASK_SUCC, "succ_sub_task_cnt")
			q = q.LeftJoin(succSubSQ, sqlchemy.Equals(succSubSQ.Field("task_id"), q.Field("id")))
			q = q.AppendField(succSubSQ.Field("succ_sub_task_cnt"))
		}

		for _, c := range manager.TableSpec().Columns() {
			q = q.AppendField(q.Field(c.Name()))
		}
	}

	// q.DebugQuery2("taskQuery")

	return q, nil
}

func (manager *STaskManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
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

func (manager *STaskManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
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

func (manager *STaskManager) ResourceScope() rbacscope.TRbacScope {
	return manager.SProjectizedResourceBaseManager.ResourceScope()
}

func (manager *STaskManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.TaskDetails {
	rows := make([]apis.TaskDetails, len(objs))
	bases := manager.SModelBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	projs := manager.SProjectizedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range objs {
		rows[i] = apis.TaskDetails{
			ModelBaseDetails:        bases[i],
			ProjectizedResourceInfo: projs[i],
		}
	}
	return rows
}

func (manager *STaskManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.TaskListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SModelBaseManager.OrderByExtraFields(ctx, q, userCred, query.ModelBaseListInput)
	if err != nil {
		return q, errors.Wrap(err, "SModelBaseManager.OrderByExtraField")
	}
	// q, err = manager.SProjectizedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ProjectizedResourceListInput)
	// if err != nil {
	//	return q, errors.Wrap(err, "SProjectizedResourceBaseManager.OrderByExtraField")
	// }
	return q, nil
}

func (task *STask) SetProgressAndStatus(progress float32, status string) error {
	_, err := db.Update(task, func() error {
		task.SetStatusValue(status)
		task.SetProgressValue(progress)
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Update")
	}
	return nil
}

func (task *STask) SetProgress(progress float32) error {
	_, err := db.Update(task, func() error {
		task.SetProgressValue(progress)
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Update")
	}
	return nil
}

func (manager *STaskManager) InitializeData() error {
	{
		err := manager.failTimeoutTasks()
		if err != nil {
			return errors.Wrap(err, "failTimeoutTasks")
		}
	}
	{
		err := manager.migrateObjectInfo()
		if err != nil {
			return errors.Wrap(err, "migrateObjectInfo")
		}
	}
	{
		err := manager.clearnUpSubtasks()
		if err != nil {
			return errors.Wrap(err, "clearnUpSubtasks")
		}
	}
	{
		err := manager.clearnUpTaskObjects()
		if err != nil {
			return errors.Wrap(err, "clearnUpTaskObjects")
		}
	}
	return nil
}

func (manager *STaskManager) failTimeoutTasks() error {
	q := manager.Query().NotIn("stage", []string{TASK_STAGE_FAILED, TASK_STAGE_COMPLETE})

	tasks := make([]STask, 0)
	err := db.FetchModelObjects(manager, q, &tasks)
	if err != nil {
		return errors.Wrap(err, "FetchModelObjects")
	}
	reason := jsonutils.NewDict()
	reason.Add(jsonutils.NewString("service restart"), "__reason__")
	reason.Add(jsonutils.NewString("error"), "__status__")
	for i := range tasks {
		task := &tasks[i]
		task.fixParams()
		manager.execTaskObject(task, reason)
	}
	return nil
}

func (manager *STaskManager) clearnUpSubtasks() error {
	start := time.Now()
	log.Infof("start clearnUpSubtasks")

	defer func() {
		log.Infof("end clearnUpSubtasks, takes %f seconds", time.Since(start).Seconds())
	}()

	q := SubTaskManager.Query("task_id")
	tasksQ := TaskManager.Query().SubQuery()

	q = q.LeftJoin(tasksQ, sqlchemy.Equals(q.Field("task_id"), tasksQ.Field("id")))
	q = q.Filter(sqlchemy.IsNull(tasksQ.Field("id")))
	q = q.Distinct()

	rows, err := q.Rows()
	if err != nil {
		return errors.Wrap(err, "q.Rows")
	}
	defer rows.Close()

	for rows.Next() {
		val, err := q.Row2Map(rows)
		if err != nil {
			return errors.Wrap(err, "Row2Map")
		}

		sql := fmt.Sprintf("delete from `%s` where task_id = '%s'", SubTaskManager.TableSpec().Name(), val["task_id"])
		log.Infof("%s", sql)
		_, err = SubTaskManager.TableSpec().GetTableSpec().Database().Exec(sql)
		if err != nil {
			return errors.Wrap(err, "exec")
		}

		sql = fmt.Sprintf("delete from `%s` where task_id = '%s'", TaskObjectManager.TableSpec().Name(), val["task_id"])
		log.Infof("%s", sql)
		_, err = TaskObjectManager.TableSpec().GetTableSpec().Database().Exec(sql)
		if err != nil {
			return errors.Wrap(err, "exec")
		}
	}

	return nil
}

func (manager *STaskManager) clearnUpTaskObjects() error {
	start := time.Now()
	log.Infof("start clearnUpTaskObjects")

	defer func() {
		log.Infof("end clearnUpTaskObjects, takes %f seconds", time.Since(start).Seconds())
	}()

	q := TaskObjectManager.Query("task_id")
	tasksQ := TaskManager.Query().SubQuery()

	q = q.LeftJoin(tasksQ, sqlchemy.Equals(q.Field("task_id"), tasksQ.Field("id")))
	q = q.Filter(sqlchemy.IsNull(tasksQ.Field("id")))
	q = q.Distinct()

	rows, err := q.Rows()
	if err != nil {
		return errors.Wrap(err, "q.Rows")
	}
	defer rows.Close()

	for rows.Next() {
		val, err := q.Row2Map(rows)
		if err != nil {
			return errors.Wrap(err, "Row2Map")
		}

		sql := fmt.Sprintf("delete from `%s` where task_id = '%s'", TaskObjectManager.TableSpec().Name(), val["task_id"])
		log.Infof("%s", sql)
		_, err = TaskObjectManager.TableSpec().GetTableSpec().Database().Exec(sql)
		if err != nil {
			return errors.Wrap(err, "exec")
		}
	}

	return nil
}

func (manager *STaskManager) migrateObjectInfo() error {
	start := time.Now()
	log.Infof("start migrateObjectInfo")

	defer func() {
		log.Infof("end migrateObjectInfo, takes %f seconds", time.Since(start).Seconds())
	}()

	q := manager.Query().NotEquals("obj_id", MULTI_OBJECTS_ID)
	taskObj := TaskObjectManager.Query().SubQuery()

	q = q.LeftJoin(taskObj, sqlchemy.Equals(q.Field("id"), taskObj.Field("task_id")))
	q = q.Filter(sqlchemy.IsNull(taskObj.Field("task_id")))
	q = q.Asc("created_at")

	q.DebugQuery2("migrateObjectInfo")

	rows, err := q.Rows()
	if err != nil {
		return errors.Wrap(err, "query.Rows")
	}
	defer rows.Close()

	for rows.Next() {
		task := STask{}
		task.SetModelManager(manager, &task)
		err := q.Row2Struct(rows, &task)
		if err != nil {
			return errors.Wrap(err, "row2struct")
		}

		taskObj := STaskObject{}
		taskObj.ObjId = task.ObjId
		taskObj.Object = task.Object
		taskObj.TaskId = task.Id
		taskObj.DomainId = task.DomainId
		taskObj.ProjectId = task.ProjectId
		taskObj.SetModelManager(TaskObjectManager, &taskObj)

		err = TaskObjectManager.TableSpec().Insert(context.Background(), &taskObj)
		if err != nil {
			return errors.Wrap(err, "Insert taskObject")
		}
	}
	return nil
}

var (
	taskCleanuoWorkerManager = appsrv.NewWorkerManager(
		"taskCleanupWorkerManager",
		1,
		1024,
		true,
	)
)

func (manager *STaskManager) TaskCleanupJob(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	taskCleanuoWorkerManager.Run(&sTaskCleanupJob{}, nil, nil)
}

type sTaskCleanupJob struct{}

func (job *sTaskCleanupJob) Run() {
	count, err := TaskManager.doTaskCleanupJob()
	if err != nil {
		log.Errorf("doTaskCleanupJob fail %s", err)
		return
	}
	if count > 0 {
		taskCleanuoWorkerManager.Run(&sTaskCleanupJob{}, nil, nil)
	}
}

func (job *sTaskCleanupJob) Dump() string {
	return "TaskCleanupJob"
}

func (manager *STaskManager) doTaskCleanupJob() (int, error) {
	ctx := context.WithValue(context.Background(), "task_cleanup_job", true)

	q := manager.Query().LT("created_at", time.Now().Add(-time.Duration(consts.TaskArchiveThresholdHours())*time.Hour)).Asc("created_at")

	if consts.TaskArchiveBatchLimit() > 0 {
		q = q.Limit(consts.TaskArchiveBatchLimit())
	}

	rows, err := q.Rows()
	if err != nil {
		log.Errorf("query rows fail %s", err)
		return 0, errors.Wrap(err, "query rows")
	}
	defer rows.Close()

	taskStart := time.Now()
	count := 0
	for rows.Next() {
		task := STask{}

		err := q.Row2Struct(rows, &task)
		if err != nil {
			log.Errorf("Row2Struct fail %s", err)
			return 0, errors.Wrap(err, "row2struct")
		}

		task.SetModelManager(ArchivedTaskManager, &task)
		err = ArchivedTaskManager.Insert(ctx, &task)
		if err != nil {
			log.Errorf("insert archive fail %s", err)
			return 0, errors.Wrap(err, "insert archive")
		}

		// cleanup
		for _, sql := range []string{
			fmt.Sprintf("DELETE FROM `%s` WHERE id = ?", manager.TableSpec().Name()),
			fmt.Sprintf("DELETE FROM `%s` WHERE task_id = ?", TaskObjectManager.TableSpec().Name()),
			fmt.Sprintf("DELETE FROM `%s` WHERE task_id = ?", SubTaskManager.TableSpec().Name()),
		} {
			_, err := manager.TableSpec().GetTableSpec().Database().Exec(sql, task.Id)
			if err != nil {
				log.Errorf("exec %s %s fail: %s", sql, task.Id, err)
				return 0, errors.Wrap(err, "exec")
			}
		}

		count++
	}
	log.Infof("TaskCleanupJob migrate %d tasks, takes %f seconds, batch limit=%d threshold hours=%d", count, time.Since(taskStart).Seconds(), consts.TaskArchiveBatchLimit(), consts.TaskArchiveThresholdHours())
	return count, nil
}

func (task *STask) PerformCancel(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.TaskCancelInput,
) (jsonutils.JSONObject, error) {
	if utils.IsInArray(task.Stage, []string{TASK_STAGE_FAILED, TASK_STAGE_COMPLETE}) {
		return nil, errors.Wrapf(errors.ErrInvalidStatus, "cannot cancel stage in %s", task.Stage)
	}
	err := task.cancel(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "cancel")
	}
	return nil, nil
}

func (task *STask) fetchSubTasks() ([]STask, error) {
	q := task.GetModelManager().Query().Equals("parent_task_id", task.Id)

	tasks := make([]STask, 0)
	err := db.FetchModelObjects(task.GetModelManager(), q, &tasks)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return tasks, nil
}

func (task *STask) cancel(ctx context.Context, reason *jsonutils.JSONDict) error {
	if utils.IsInArray(task.Stage, []string{TASK_STAGE_FAILED, TASK_STAGE_COMPLETE}) {
		return nil
	}
	if reason == nil {
		reason = jsonutils.NewDict()
		reason.Add(jsonutils.NewString("cancel"), "__reason__")
		reason.Add(jsonutils.NewString("error"), "__status__")
	}

	subtasks, err := task.fetchSubTasks()
	if err != nil {
		return errors.Wrap(err, "fetchSubTasks")
	}
	for i := range subtasks {
		err := subtasks[i].cancel(ctx, reason)
		if err != nil {
			return errors.Wrap(err, "cancelTask")
		}
	}

	task.fixParams()
	TaskManager.execTask(task.GetTaskId(), reason)
	return nil
}
