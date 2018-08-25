package taskman

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
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

	ObjName  string                   `width:"128" charset:"utf8" nullable:"false" list:"user"`  //  Column(VARCHAR(128, charset='utf8'), nullable=False)
	ObjId    string                   `width:"128" charset:"ascii" nullable:"false" list:"user"` // Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=False)
	TaskName string                   `width:"64" charset:"ascii" nullable:"false" list:"user"`  // Column(VARCHAR(64, charset='ascii'), nullable=False)
	UserCred mcclient.TokenCredential `width:"1024" charset:"ascii" nullable:"false" get:"user"` // Column(VARCHAR(1024, charset='ascii'), nullable=False)
	// OwnerCred string `width:"512" charset:"ascii" nullable:"true"` // Column(VARCHAR(512, charset='ascii'), nullable=True)
	Params *jsonutils.JSONDict `charset:"ascii" length:"medium" nullable:"false" get:"user"` // Column(MEDIUMTEXT(charset='ascii'), nullable=False)

	Stage string `width:"64" charset:"ascii" nullable:"false" default:"on_init" list:"user"` // Column(VARCHAR(64, charset='ascii'), nullable=False, default='on_init')

	taskObject  db.IStandaloneModel   `ignore:"true"`
	taskObjects []db.IStandaloneModel `ignore:"true"`
}

func (manager *STaskManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
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

func (manager *STaskManager) FilterByOwner(q *sqlchemy.SQuery, ownerProjId string) *sqlchemy.SQuery {
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

func (self *STask) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin() || userCred.GetProjectId() == self.UserCred.GetProjectId()
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
			if len(reqContext.TaskId) > 0 {
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
	return &task, nil
}

func (manager *STaskManager) NewParallelTask(ctx context.Context, taskName string, objs []db.IStandaloneModel,
	userCred mcclient.TokenCredential, taskData *jsonutils.JSONDict,
	parentTaskId string, parentTaskNotifyUrl string,
	pendingUsage quotas.IQuota) (*STask, error) {
	if !isTaskExist(taskName) {
		return nil, fmt.Errorf("task %s not found", taskName)
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
	task, err := db.NewModelObject(manager)
	if err != nil {
		log.Errorf("New task object fail: %s", err)
		return nil
	}
	err = manager.Query().Equals("id", idStr).First(task)
	if err != nil {
		log.Errorf("GetTask %s fail: %s", idStr, err)
		return nil
	}
	return task.(*STask)
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
	filled := reflectutils.FillEmbededStructValue(taskValue.Elem(), reflect.Indirect(reflect.ValueOf(baseTask)))
	if !filled {
		log.Errorf("Cannot locate baseTask embedded struct, give up...")
		return
	}
	if taskValue.Type().Implements(ITaskType) {
		execITask(taskValue, baseTask, data, false)
	} else if taskValue.Type().Implements(IBatchTaskType) {
		execITask(taskValue, baseTask, data, true)
	} else {
		log.Errorf("Unsupported task type?? %s", taskValue.Type())
	}
}

func execITask(taskValue reflect.Value, task *STask, data jsonutils.JSONObject, isMulti bool) {
	var err error
	ctxData := task.GetRequestContext()
	ctx := ctxData.GetContext()

	taskFailed := false

	if data != nil {
		taskStatus, _ := data.GetString("__status__")
		if len(taskStatus) > 0 && taskStatus != "OK" {
			taskFailed = true
			data, err = data.Get("__reason__")
			if err != nil {
				data = jsonutils.NewString("Task failed due to unknown remote errors!")
			}
		}
	} else {
		data = jsonutils.NewDict()
	}

	var stageName string
	if taskFailed {
		stageName = fmt.Sprintf("%s_failed", task.Stage)
	} else {
		stageName = task.Stage
	}

	funcValue := taskValue.MethodByName(stageName)

	if !funcValue.IsValid() || funcValue.IsNil() {
		stageName = utils.Kebab2Camel(stageName, "_")
		funcValue = taskValue.MethodByName(stageName)

		if !funcValue.IsValid() || funcValue.IsNil() {
			msg := fmt.Sprintf("Stage %s not found", stageName)
			log.Errorf(msg)
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

		// lockman.LockClass(ctx, objResManager, task.UserCred.GetProjectId())
		// defer lockman.ReleaseClass(ctx, objResManager, task.UserCred.GetProjectId())

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

		// lockman.LockObject(ctx, obj)
		// defer lockman.ReleaseObject(ctx, obj)

		params[1] = reflect.ValueOf(obj)
	}

	params[2] = reflect.ValueOf(data)

	log.Debugf("Call %s with %s", funcValue, params)

	funcValue.Call(params)

	task.SaveRequestContext(&ctxData)
}

func (task *STask) ScheduleRun(data jsonutils.JSONObject) {
	runTask(task.Id, data)
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
	ctxJson, _ := self.Params.Get(REQUEST_CONTEXT_KEY)
	if ctxJson != nil {
		ctxJson.Unmarshal(&ctxData)
	}
	return ctxData
}

func (self *STask) SaveRequestContext(data *appctx.AppContextData) {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		params := self.Params.CopyExcludes(REQUEST_CONTEXT_KEY)
		params.Add(jsonutils.Marshal(data), REQUEST_CONTEXT_KEY)
		self.Params = params
		return nil
	})
	if err != nil {
		log.Errorf("save_request_context fail %s", err)
	}
}

func (self *STask) SetStage(stageName string, data *jsonutils.JSONDict) {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		params := jsonutils.NewDict()
		params.Update(self.Params)
		if data != nil {
			params.Update(data)
		}
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
		self.Params = params
		return nil
	})
	if err != nil {
		log.Errorf("set_stage fail %s", err)
	}
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
		pTask := TaskManager.fetchTask(parentTaskId)
		if pTask == nil {
			log.Errorf("Parent task %s not found", parentTaskId)
			return
		}
		if pTask.IsCurrentStageComplete() {
			pTask.ScheduleRun(body)
		}
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
	body.Add(jsonutils.NewString(fmt.Sprintf("Subtask %s failed: %s", self.TaskName, reason)))
	self.NotifyParentTaskComplete(ctx, body, true)
}

func (self *STask) IsCurrentStageComplete() bool {
	subtasks := SubTaskManager.GetInitSubtasks(self.Id, self.Stage)
	if len(subtasks) == 0 {
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
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
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
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
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
