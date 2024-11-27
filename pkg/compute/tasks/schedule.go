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

package tasks

import (
	"context"
	"fmt"
	"sort"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/scheduler"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type IScheduleModel interface {
	db.IStandaloneModel

	SetStatus(ctx context.Context, userCred mcclient.TokenCredential, status string, reason string) error
}

type IScheduleTask interface {
	GetUserCred() mcclient.TokenCredential
	GetSchedParams() (*schedapi.ScheduleInput, error)
	GetPendingUsage(quota quotas.IQuota, index int) error
	SetStage(stageName string, data *jsonutils.JSONDict) error
	SetStageFailed(ctx context.Context, reason jsonutils.JSONObject)

	OnStartSchedule(obj IScheduleModel)
	OnScheduleFailCallback(ctx context.Context, obj IScheduleModel, reason jsonutils.JSONObject, index int)
	// OnScheduleComplete(ctx context.Context, items []db.IStandaloneModel, data *jsonutils.JSONDict)
	SaveScheduleResult(ctx context.Context, obj IScheduleModel, candidate *schedapi.CandidateResource, index int)
	SaveScheduleResultWithBackup(ctx context.Context, obj IScheduleModel, master, slave *schedapi.CandidateResource, index int)
	OnScheduleFailed(ctx context.Context, reason jsonutils.JSONObject)
}

type SSchedTask struct {
	taskman.STask
	input *schedapi.ScheduleInput
}

func (self *SSchedTask) OnStartSchedule(obj IScheduleModel) {
	db.OpsLog.LogEvent(obj, db.ACT_ALLOCATING, nil, self.GetUserCred())
	obj.SetStatus(context.Background(), self.GetUserCred(), api.VM_SCHEDULE, "")
}

func (self *SSchedTask) OnScheduleFailCallback(ctx context.Context, obj IScheduleModel, reason jsonutils.JSONObject, index int) {
	obj.SetStatus(ctx, self.GetUserCred(), api.VM_SCHEDULE_FAILED, reason.String())
	db.OpsLog.LogEvent(obj, db.ACT_ALLOCATE_FAIL, reason, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, obj, logclient.ACT_ALLOCATE, reason, self.GetUserCred(), false)
	notifyclient.NotifySystemErrorWithCtx(ctx, obj.GetId(), obj.GetName(), api.VM_SCHEDULE_FAILED, reason.String())
}

func (self *SSchedTask) OnScheduleComplete(ctx context.Context, items []db.IStandaloneModel, data *jsonutils.JSONDict) {
	self.SetStageComplete(ctx, nil)
}

func (self *SSchedTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, candidate *schedapi.CandidateResource, index int) {
	// ...
}

func (self *SSchedTask) SaveScheduleResultWithBackup(ctx context.Context, obj IScheduleModel, master, slave *schedapi.CandidateResource, index int) {
	// ...
}

func (self *SSchedTask) OnScheduleFailed(ctx context.Context, reason jsonutils.JSONObject) {
	self.SetStageFailed(ctx, reason)
}

func StartScheduleObjects(
	ctx context.Context,
	task IScheduleTask,
	objs []db.IStandaloneModel,
) {
	schedObjs := make([]IScheduleModel, len(objs))
	for i, obj := range objs {
		schedObj := obj.(IScheduleModel)
		schedObjs[i] = schedObj
		task.OnStartSchedule(schedObj)
	}
	doScheduleObjects(ctx, task, schedObjs)
}

func doScheduleWithInput(
	ctx context.Context,
	task IScheduleTask,
	schedInput *schedapi.ScheduleInput,
	count int,
) (*schedapi.ScheduleOutput, error) {
	computeUsage := models.SQuota{}
	task.GetPendingUsage(&computeUsage, 0)
	regionUsage := models.SRegionQuota{}
	task.GetPendingUsage(&regionUsage, 1)

	schedInput.PendingUsages = []jsonutils.JSONObject{
		jsonutils.Marshal(&computeUsage),
		jsonutils.Marshal(&regionUsage),
	}

	var params *jsonutils.JSONDict
	if count > 0 {
		// if object count <=0, don't need update schedule params
		params = jsonutils.Marshal(schedInput).(*jsonutils.JSONDict)
	}
	task.SetStage("OnScheduleComplete", params)
	s := auth.GetSession(ctx, task.GetUserCred(), options.Options.Region)
	return scheduler.SchedManager.DoSchedule(s, schedInput, count)
}

func doScheduleObjects(
	ctx context.Context,
	task IScheduleTask,
	objs []IScheduleModel,
) {
	schedInput, err := task.GetSchedParams()
	if err != nil {
		onSchedulerRequestFail(ctx, task, objs, jsonutils.NewString(fmt.Sprintf("GetSchedParams fail: %s", err)))
		return
	}

	sort.Sort(sortedIScheduleModelList(objs))
	schedInput.GuestIds = make([]string, len(objs))
	for i := range objs {
		schedInput.GuestIds[i] = objs[i].GetId()
	}

	output, err := doScheduleWithInput(ctx, task, schedInput, len(objs))
	if err != nil {
		onSchedulerRequestFail(ctx, task, objs, jsonutils.NewString(err.Error()))
		return
	}
	onSchedulerResults(ctx, task, objs, output.Candidates)
}

func cancelPendingUsage(ctx context.Context, task IScheduleTask) {
	ClearTaskPendingUsage(ctx, task.(taskman.ITask))
	ClearTaskPendingRegionUsage(ctx, task.(taskman.ITask))
}

func onSchedulerRequestFail(
	ctx context.Context,
	task IScheduleTask,
	objs []IScheduleModel,
	reason jsonutils.JSONObject,
) {
	for i, obj := range objs {
		onObjScheduleFail(ctx, task, obj, reason, i)
	}
	task.OnScheduleFailed(ctx, reason)
	cancelPendingUsage(ctx, task)
}

func onObjScheduleFail(
	ctx context.Context,
	task IScheduleTask,
	obj IScheduleModel,
	msg jsonutils.JSONObject,
	idx int,
) {
	lockman.LockObject(ctx, obj)
	defer lockman.ReleaseObject(ctx, obj)

	var reason jsonutils.JSONObject
	reason = jsonutils.NewString("No matching resources")
	if msg != nil {
		reason = jsonutils.NewArray(reason, msg)
	}
	task.OnScheduleFailCallback(ctx, obj, reason, idx)
}

type sortedIScheduleModelList []IScheduleModel

func (a sortedIScheduleModelList) Len() int           { return len(a) }
func (a sortedIScheduleModelList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a sortedIScheduleModelList) Less(i, j int) bool { return a[i].GetName() < a[j].GetName() }

func onSchedulerResults(
	ctx context.Context,
	task IScheduleTask,
	objs []IScheduleModel,
	results []*schedapi.CandidateResource,
) {
	if len(objs) == 0 {
		// sched with out object can't clean sched cache immediately
		task.SaveScheduleResult(ctx, nil, results[0], 0)
		return
	}
	succCount := 0
	for idx := 0; idx < len(objs); idx += 1 {
		obj := objs[idx]
		result := results[idx]

		if len(result.Error) != 0 {
			onObjScheduleFail(ctx, task, obj, jsonutils.NewString(result.Error), idx)
			continue
		}

		if result.BackupCandidate == nil {
			// normal schedule
			onScheduleSucc(ctx, task, obj, result, idx)
		} else {
			// backup schedule
			onMasterSlaveScheduleSucc(ctx, task, obj, result, result.BackupCandidate, idx)
		}
		succCount += 1
	}
	if succCount == 0 {
		task.OnScheduleFailed(ctx, jsonutils.NewString("Schedule failed"))
	}
	cancelPendingUsage(ctx, task)
}

func onMasterSlaveScheduleSucc(
	ctx context.Context,
	task IScheduleTask,
	obj IScheduleModel,
	master, slave *schedapi.CandidateResource,
	index int,
) {
	lockman.LockObject(ctx, obj)
	defer lockman.ReleaseObject(ctx, obj)
	task.SaveScheduleResultWithBackup(ctx, obj, master, slave, index)
	models.HostManager.ClearSchedDescSessionCache(master.HostId, master.SessionId)
	models.HostManager.ClearSchedDescSessionCache(slave.HostId, slave.SessionId)
}

func onScheduleSucc(
	ctx context.Context,
	task IScheduleTask,
	obj IScheduleModel,
	candidate *schedapi.CandidateResource,
	index int,
) {
	hostId := candidate.HostId
	lockman.LockRawObject(ctx, models.HostManager.KeywordPlural(), hostId)
	defer lockman.ReleaseRawObject(ctx, models.HostManager.KeywordPlural(), hostId)

	task.SaveScheduleResult(ctx, obj, candidate, index)
	models.HostManager.ClearSchedDescSessionCache(candidate.HostId, candidate.SessionId)
}

func getBatchParamsAtIndex(task taskman.ITask, index int) *jsonutils.JSONDict {
	var data *jsonutils.JSONDict
	params := task.GetParams()
	paramsArray, _ := params.GetArray("data")
	if len(paramsArray) > 0 {
		if len(paramsArray) > index {
			data = paramsArray[index].(*jsonutils.JSONDict)
		} else {
			data = paramsArray[0].(*jsonutils.JSONDict)
		}
	} else {
		data = params
	}
	return data
}
