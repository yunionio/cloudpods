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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
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
)

type STaskManager struct {
	db.SResourceBaseManager
}

var TaskManager *STaskManager

func init() {
	TaskManager = &STaskManager{SResourceBaseManager: db.NewResourceBaseManager(STask{}, "tasks_tbl", "task", "tasks")}
}

type STask struct {
	db.SResourceBase

	Id string `width:"36" charset:"ascii" primary:"true" list:"user"` // Column(VARCHAR(36, charset='ascii'), primary_key=True, default=get_uuid)

	ObjName  string                   `width:"128" charset:"utf8" nullable:"false" list:"user"`               //  Column(VARCHAR(128, charset='utf8'), nullable=False)
	ObjId    string                   `width:"128" charset:"ascii" nullable:"false" list:"user" index:"true"` // Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=False)
	TaskName string                   `width:"64" charset:"ascii" nullable:"false" list:"user"`               // Column(VARCHAR(64, charset='ascii'), nullable=False)
	UserCred mcclient.TokenCredential `width:"1024" charset:"ascii" nullable:"false" get:"user"`              // Column(VARCHAR(1024, charset='ascii'), nullable=False)
	// OwnerCred string `width:"512" charset:"ascii" nullable:"true"` // Column(VARCHAR(512, charset='ascii'), nullable=True)
	Params *jsonutils.JSONDict `charset:"ascii" length:"medium" nullable:"false" get:"user"` // Column(MEDIUMTEXT(charset='ascii'), nullable=False)

	Stage string `width:"64" charset:"ascii" nullable:"false" default:"on_init" list:"user"` // Column(VARCHAR(64, charset='ascii'), nullable=False, default='on_init')

	taskObject  db.IStandaloneModel   `ignore:"true"`
	taskObjects []db.IStandaloneModel `ignore:"true"`
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
	return q
}

func (manager *STaskManager) AllowPerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
}

func (manager *STaskManager) PerformAction(ctx context.Context, userCred mcclient.TokenCredential, taskId string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	runTask(taskId, data)
	resp := jsonutils.NewDict()
	// 'result': 'ok'
	resp.Add(jsonutils.NewString("ok"), "result")
	return resp, nil
}

func (manager *STaskManager) GetOwnerId(userCred mcclient.IIdentityProvider) string {
	return userCred.GetProjectId()
}

func (self *STask) GetOwnerProjectId() string {
	return self.UserCred.GetProjectId()
}

func (manager *STaskManager) FilterByOwner(q *sqlchemy.SQuery, owner string) *sqlchemy.SQuery {
	q = q.Contains("user_cred", owner)
	return q
}

func (self *STask) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self) || userCred.GetProjectId() == self.UserCred.GetProjectId()
}

func (self *STask) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

func (self *STask) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (self *STask) ValidateDeleteCondition(ctx context.Context) error {
	return fmt.Errorf("forbidden")
}

func (self *STask) BeforeInsert() {
	if len(self.Id) == 0 {
		self.Id = stringutils.UUID4()
	}
}

func (self *STask) GetId() string {
	return self.Id
}

func (self *STask) GetName() string {
	return self.TaskName
}

func fetchTaskParams(ctx context.Context, taskName string, taskData *jsonutils.JSONDict,
	parentTaskId string, parentTaskNotifyUrl string,
	pendingUsage quotas.IQuota) *jsonutils.JSONDict {
	var data *jsonutils.JSONDict
	if taskData != nil {
		data = taskData.CopyExcludes(PARENT_TASK_ID_KEY, PARENT_TASK_NOTIFY_KEY, PENDING_USAGE_KEY)
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
	if pendingUsage != nil {
		data.Add(jsonutils.Marshal(pendingUsage), PENDING_USAGE_KEY)
	}
	return data
}

func (manager *STaskManager) NewTask(ctx context.Context, taskName string, obj db.IStandaloneModel,
	userCred mcclient.TokenCredential, taskData *jsonutils.JSONDict,
	parentTaskId string, parentTaskNotifyUrl string,
	pendingUsage quotas.IQuota) (*STask, error) {
	if !isTaskExist(taskName) {
		return nil, fmt.Errorf("task %s not found", taskName)
	}

	lockman.LockObject(ctx, obj)
	defer lockman.ReleaseObject(ctx, obj)

	data := fetchTaskParams(ctx, taskName, taskData, parentTaskId, parentTaskNotifyUrl, pendingUsage)
	task := STask{
		ObjName:  obj.Keyword(),
		ObjId:    obj.GetId(),
		TaskName: taskName,
		UserCred: userCred,
		Params:   data,
		Stage:    TASK_INIT_STAGE,
	}
	err := manager.TableSpec().Insert(&task)
	if err != nil {
		log.Errorf("Task insert error %s", err)
		return nil, err
	}
	parentTask := task.GetParentTask()
	if parentTask != nil {
		st := SSubTask{TaskId: parentTask.Id, Stage: parentTask.Stage, SubtaskId: task.Id}
		err := SubTaskManager.TableSpec().Insert(&st)
		if err != nil {
			log.Errorf("Subtask insert error %s", err)
			return nil, err
		}
	}
	return &task, nil
}

func (manager *STaskManager) NewParallelTask(ctx context.Context, taskName string, objs []db.IStandaloneModel,
	userCred mcclient.TokenCredential, taskData *jsonutils.JSONDict,
	parentTaskId string, parentTaskNotifyUrl string,
	pendingUsage quotas.IQuota) (*STask, error) {
	if !isTaskExist(taskName) {
		return nil, fmt.Errorf("task %s not found", taskName)
	}

	if len(objs) == 0 {
		return nil, fmt.Errorf("failed to do task %s with zero objs", taskName)
	}

	log.Debugf("number of objs: %d", len(objs))
	lockman.LockClass(ctx, objs[0].GetModelManager(), userCred.GetProjectId())
	defer lockman.ReleaseClass(ctx, objs[0].GetModelManager(), userCred.GetProjectId())

	data := fetchTaskParams(ctx, taskName, taskData, parentTaskId, parentTaskNotifyUrl, pendingUsage)
	task := STask{
		ObjName:  objs[0].Keyword(),
		ObjId:    MULTI_OBJECTS_ID,
		TaskName: taskName,
		UserCred: userCred,
		Params:   data,
		Stage:    TASK_INIT_STAGE,
	}
	err := manager.TableSpec().Insert(&task)
	if err != nil {
		log.Errorf("Task insert error %s", err)
		return nil, err
	}
	for _, obj := range objs {
		to := STaskObject{TaskId: task.Id, ObjId: obj.GetId()}
		err := TaskObjectManager.TableSpec().Insert(&to)
		if err != nil {
			log.Errorf("Taskobject insert error %s", err)
			return nil, err
		}
	}
	parentTask := task.GetParentTask()
	if parentTask != nil {
		st := SSubTask{TaskId: parentTask.Id, Stage: parentTask.Stage, SubtaskId: task.Id}
		err := SubTaskManager.TableSpec().Insert(&st)
		if err != nil {
			log.Errorf("Subtask insert error %s", err)
			return nil, err
		}
	}
	return &task, nil
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
	if task.Params == nil {
		task.Params = jsonutils.NewDict()
	}
	return task
}

func (manager *STaskManager) getTaskName(taskId string) string {
	baseTask := manager.fetchTask(taskId)
	if baseTask == nil {
		return ""
	}
	return baseTask.TaskName
}

func (manager *STaskManager) execTask(taskId string, data jsonutils.JSONObject) {
	baseTask := manager.fetchTask(taskId)
	if baseTask == nil {
		return
	}
	taskType, ok := taskTable[baseTask.TaskName]
	if !ok {
		log.Errorf("Cannot find task %s", baseTask.TaskName)
		return
	}
	log.Debugf("Do task %s(%s) with data %s at stage %s", taskType, taskId, data, baseTask.Stage)
	taskValue := reflect.New(taskType)
	if taskValue.Type().Implements(ITaskType) {
		execITask(taskValue, baseTask, data, false)
	} else if taskValue.Type().Implements(IBatchTaskType) {
		execITask(taskValue, baseTask, data, true)
	} else {
		log.Errorf("Unsupported task type?? %s", taskValue.Type())
	}
}

func execITask(taskValue reflect.Value, task *STask, odata jsonutils.JSONObject, isMulti bool) {
	ctxData := task.GetRequestContext()
	ctx := ctxData.GetContext()

	taskFailed := false

	data := odata
	if data != nil {
		taskStatus, _ := data.GetString("__status__")
		if len(taskStatus) > 0 && taskStatus != "OK" {
			taskFailed = true
			if vdata, ok := data.(*jsonutils.JSONDict); ok {
				reason, err := vdata.Get("__reason__") // only dict support Get
				if err != nil {
					reason = jsonutils.NewString(fmt.Sprintf("Task failed due to unknown remote errors! %s", odata))
					vdata.Set("__reason__", reason)
				}
			}
		}
	} else {
		data = jsonutils.NewDict()
	}

	var stageName string
	if taskFailed {
		stageName = fmt.Sprintf("%sFailed", task.Stage)
	} else {
		stageName = task.Stage
	}

	funcValue := taskValue.MethodByName(stageName)

	if !funcValue.IsValid() || funcValue.IsNil() {
		log.Debugf("Stage %s not found, try kebab to camel and find again", stageName)
		if taskFailed {
			stageName = fmt.Sprintf("%s_failed", task.Stage)
		}
		stageName = utils.Kebab2Camel(stageName, "_")
		funcValue = taskValue.MethodByName(stageName)

		if !funcValue.IsValid() || funcValue.IsNil() {
			msg := fmt.Sprintf("Stage %s not found", stageName)
			if taskFailed {
				// failed handler is optional, ignore the error
				log.Warningf(msg)
				msg, _ = data.GetString()
			} else {
				log.Errorf(msg)
			}
			task.SetStageFailed(ctx, msg)
			task.SaveRequestContext(&ctxData)
			return
		}
	}

	objManager := db.GetModelManager(task.ObjName)
	if objManager == nil {
		msg := fmt.Sprintf("model %s not found??? ...", task.ObjName)
		log.Errorf(msg)
		task.SetStageFailed(ctx, msg)
		task.SaveRequestContext(&ctxData)
		return
	}
	// log.Debugf("objManager: %s", objManager)
	objResManager, ok := objManager.(db.IStandaloneModelManager)
	if !ok {
		msg := fmt.Sprintf("mode %s is not a resource??? ...", task.ObjName)
		log.Errorf(msg)
		task.SetStageFailed(ctx, msg)
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
				msg := fmt.Sprintf("fail to find %s object %s", task.ObjName, objId)
				log.Errorf(msg)
				task.SetStageFailed(ctx, msg)
				task.SaveRequestContext(&ctxData)
				return
			}
			objs[i] = obj
		}
		task.taskObjects = objs

		lockman.LockClass(ctx, objResManager, task.UserCred.GetProjectId())
		defer lockman.ReleaseClass(ctx, objResManager, task.UserCred.GetProjectId())

		params[1] = reflect.ValueOf(objs)
	} else {
		obj, err := objResManager.FetchById(task.ObjId)
		if err != nil {
			msg := fmt.Sprintf("fail to find %s object %s", task.ObjName, task.ObjId)
			log.Errorf(msg)
			task.SetStageFailed(ctx, msg)
			task.SaveRequestContext(&ctxData)
			return
		}
		task.taskObject = obj

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

	log.Debugf("Call %s %s %#v", task.TaskName, stageName, params)
	funcValue.Call(params)

	// call save request context
	saveRequestContextFuncValue := taskValue.MethodByName("SaveRequestContext")
	saveRequestContextFuncValue.Call([]reflect.Value{reflect.ValueOf(&ctxData)})
}

func (task *STask) ScheduleRun(data jsonutils.JSONObject) {
	runTask(task.Id, data)
}

func (self *STask) IsSubtask() bool {
	return self.HasParentTask()
}

func (self *STask) HasParentTask() bool {
	parentTaskId, _ := self.Params.GetString(PARENT_TASK_ID_KEY)
	if len(parentTaskId) > 0 {
		return true
	}
	return false
}

func (self *STask) GetParentTask() *STask {
	parentTaskId, _ := self.Params.GetString(PARENT_TASK_ID_KEY)
	if len(parentTaskId) > 0 {
		return TaskManager.fetchTask(parentTaskId)
	}
	return nil
}

func (self *STask) GetRequestContext() appctx.AppContextData {
	ctxData := appctx.AppContextData{}
	if self.Params != nil {
		ctxJson, _ := self.Params.Get(REQUEST_CONTEXT_KEY)
		if ctxJson != nil {
			ctxJson.Unmarshal(&ctxData)
		}
	}
	return ctxData
}

func (self *STask) SaveRequestContext(data *appctx.AppContextData) {
	_, err := db.Update(self, func() error {
		params := self.Params.CopyExcludes(REQUEST_CONTEXT_KEY)
		params.Add(jsonutils.Marshal(data), REQUEST_CONTEXT_KEY)
		self.Params = params
		return nil
	})
	if err != nil {
		log.Errorf("save_request_context fail %s", err)
	}
}

func (self *STask) SaveParams(data *jsonutils.JSONDict) error {
	return self.SetStage("", data)
}

func (self *STask) SetStage(stageName string, data *jsonutils.JSONDict) error {
	_, err := db.Update(self, func() error {
		params := jsonutils.NewDict()
		params.Update(self.Params)
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
			stageData.Add(jsonutils.NewString(self.Stage), "name")
			stageData.Add(jsonutils.NewTimeString(time.Now()), "complete_at")
			stageList.Add(stageData)
			self.Stage = stageName
		}
		self.Params = params
		return nil
	})
	if err != nil {
		log.Errorf("set_stage fail %s", err)
	}
	return err
}

func (self *STask) GetObjectIdStr() string {
	if self.ObjId == MULTI_OBJECTS_ID {
		return strings.Join(TaskObjectManager.GetObjectIds(self), ",")
	} else {
		return self.ObjId
	}
}

func (self *STask) SetStageComplete(ctx context.Context, data *jsonutils.JSONDict) {
	log.Infof("XXX TASK %s complete", self.TaskName)
	self.SetStage(TASK_STAGE_COMPLETE, data)
	if data == nil {
		data = jsonutils.NewDict()
	}
	if data.Size() == 0 {
		data.Add(jsonutils.NewString(self.GetObjectIdStr()), "id")
		data.Add(jsonutils.NewString(self.ObjName), "name")
	}
	self.NotifyParentTaskComplete(ctx, data, false)
}

func (self *STask) SetStageFailed(ctx context.Context, reason string) {
	if self.Stage == TASK_STAGE_FAILED {
		log.Warningf("Task %s has been failed", self.TaskName)
		return
	}
	log.Infof("XXX TASK %s failed: %s on stage %s", self.TaskName, reason, self.Stage)
	prevFailed, _ := self.Params.GetString("__failed_reason")
	if len(prevFailed) > 0 {
		reason = prevFailed + ";" + reason
	}
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(reason), "__failed_reason")
	self.SetStage(TASK_STAGE_FAILED, data)
	self.NotifyParentTaskFailure(ctx, reason)
}

func (self *STask) NotifyParentTaskComplete(ctx context.Context, body *jsonutils.JSONDict, failed bool) {
	log.Infof("notify_parent_task_complete: %s params %s", self.TaskName, self.Params)
	parentTaskId, _ := self.Params.GetString(PARENT_TASK_ID_KEY)
	parentTaskNotify, _ := self.Params.GetString(PARENT_TASK_NOTIFY_KEY)
	if len(parentTaskId) > 0 {
		subTask := SubTaskManager.GetSubTask(parentTaskId, self.Id)
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

func (self *STask) NotifyParentTaskFailure(ctx context.Context, reason string) {
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString("error"), "__status__")
	if len(reason) > 100 {
		reason = reason[:100] + "..."
	}
	body.Add(jsonutils.NewString(fmt.Sprintf("Subtask %s failed: %s", self.TaskName, reason)), "__reason__")
	self.NotifyParentTaskComplete(ctx, body, true)
}

func (self *STask) IsCurrentStageComplete() bool {
	totalSubtasks := SubTaskManager.GetTotalSubtasks(self.Id, self.Stage, "")
	initSubtasks := SubTaskManager.GetInitSubtasks(self.Id, self.Stage)
	if len(totalSubtasks) > 0 && len(initSubtasks) == 0 {
		return true
	} else {
		return false
	}
}

func (self *STask) GetPendingUsage(quota quotas.IQuota) error {
	quotaJson, err := self.Params.Get(PENDING_USAGE_KEY)
	if err != nil {
		return err
	}
	return quotaJson.Unmarshal(quota)
}

func (self *STask) SetPendingUsage(quota quotas.IQuota) error {
	_, err := db.Update(self, func() error {
		params := self.Params.CopyExcludes(PENDING_USAGE_KEY)
		params.Add(jsonutils.Marshal(quota), PENDING_USAGE_KEY)
		self.Params = params
		return nil
	})
	if err != nil {
		log.Errorf("set_pending_usage fail %s", err)
	}
	return err
}

func (self *STask) ClearPendingUsage() error {
	_, err := db.Update(self, func() error {
		params := self.Params.CopyExcludes(PENDING_USAGE_KEY)
		self.Params = params
		return nil
	})
	if err != nil {
		log.Errorf("clear_pending_usage fail %s", err)
	}
	return err
}

func (self *STask) GetParams() *jsonutils.JSONDict {
	return self.Params
}

func (self *STask) GetUserCred() mcclient.TokenCredential {
	return self.UserCred
}

func (self *STask) GetTaskId() string {
	return self.GetId()
}

func (self *STask) GetObject() db.IStandaloneModel {
	return self.taskObject
}

func (self *STask) GetObjects() []db.IStandaloneModel {
	return self.taskObjects
}

func (task *STask) GetTaskRequestHeader() http.Header {
	header := mcclient.GetTokenHeaders(task.GetUserCred())
	header.Set(mcclient.TASK_ID, task.GetTaskId())
	return header
}

func (task *STask) GetStartTime() time.Time {
	return task.CreatedAt
}

func (manager *STaskManager) QueryTasksOfObject(obj db.IStandaloneModel, since time.Time, isOpen *bool) *sqlchemy.SQuery {
	subq1 := manager.Query()
	{
		subq1 = subq1.Equals("obj_id", obj.GetId())
		subq1 = subq1.Equals("obj_name", obj.Keyword())
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
		subq2 = subq2.Filter(sqlchemy.Equals(subq2.Field("obj_name"), obj.Keyword()))
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

	// subq1 and subq2 do not intersect for the fact that they have
	// different condition on tasks_tbl.obj_id field
	uq := sqlchemy.Union(subq1, subq2)
	uq = uq.Desc("created_at")

	q := uq.SubQuery().Query()
	return q
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
