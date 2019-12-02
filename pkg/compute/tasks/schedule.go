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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type IScheduleModel interface {
	db.IStandaloneModel

	SetStatus(userCred mcclient.TokenCredential, status string, reason string) error
}

type IScheduleTask interface {
	GetUserCred() mcclient.TokenCredential
	GetSchedParams() (*schedapi.ScheduleInput, error)
	GetPendingUsage(quota quotas.IQuota, index int) error
	SetStage(stageName string, data *jsonutils.JSONDict) error
	SetStageFailed(ctx context.Context, reason string)

	OnStartSchedule(obj IScheduleModel)
	OnScheduleFailCallback(ctx context.Context, obj IScheduleModel, reason string)
	// OnScheduleComplete(ctx context.Context, items []db.IStandaloneModel, data *jsonutils.JSONDict)
	SaveScheduleResult(ctx context.Context, obj IScheduleModel, candidate *schedapi.CandidateResource)
	SaveScheduleResultWithBackup(ctx context.Context, obj IScheduleModel, master, slave *schedapi.CandidateResource)
	OnScheduleFailed(ctx context.Context, reason string)
}

type SSchedTask struct {
	taskman.STask
	input *schedapi.ScheduleInput
}

func (self *SSchedTask) GetSchedParams() (*schedapi.ScheduleInput, error) {
	params := self.GetParams()
	input, err := cmdline.FetchScheduleInputByJSON(params)
	if err != nil {
		return nil, fmt.Errorf("Unmarsh to schedule input: %v", err)
	}
	return input, err
}

func (self *SSchedTask) GetDisks() ([]*api.DiskConfig, error) {
	input, err := self.GetSchedParams()
	if err != nil {
		return nil, err
	}
	return input.Disks, nil
}

func (self *SSchedTask) GetFirstDisk() (*api.DiskConfig, error) {
	disks, err := self.GetDisks()
	if err != nil {
		return nil, err
	}
	if len(disks) == 0 {
		return nil, fmt.Errorf("Empty disks to schedule")
	}
	return disks[0], nil
}

func (self *SSchedTask) OnStartSchedule(obj IScheduleModel) {
	db.OpsLog.LogEvent(obj, db.ACT_ALLOCATING, nil, self.GetUserCred())
	obj.SetStatus(self.GetUserCred(), api.VM_SCHEDULE, "")
}

func (self *SSchedTask) OnScheduleFailCallback(ctx context.Context, obj IScheduleModel, reason string) {
	obj.SetStatus(self.GetUserCred(), api.VM_SCHEDULE_FAILED, reason)
	db.OpsLog.LogEvent(obj, db.ACT_ALLOCATE_FAIL, reason, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, obj, logclient.ACT_ALLOCATE, reason, self.GetUserCred(), false)
	notifyclient.NotifySystemError(obj.GetId(), obj.GetName(), api.VM_SCHEDULE_FAILED, reason)
}

func (self *SSchedTask) OnScheduleComplete(ctx context.Context, items []db.IStandaloneModel, data *jsonutils.JSONDict) {
	self.SetStageComplete(ctx, nil)
}

func (self *SSchedTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, candidate *schedapi.CandidateResource) {
	// ...
}

func (self *SSchedTask) SaveScheduleResultWithBackup(ctx context.Context, obj IScheduleModel, master, slave *schedapi.CandidateResource) {
	// ...
}

func (self *SSchedTask) OnScheduleFailed(ctx context.Context, reason string) {
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

func doScheduleObjects(
	ctx context.Context,
	task IScheduleTask,
	objs []IScheduleModel,
) {
	schedInput, err := task.GetSchedParams()
	if err != nil {
		onSchedulerRequestFail(ctx, task, objs, fmt.Sprintf("Scheduler fail: %s", err))
		return
	}
	//schedInput = models.ApplySchedPolicies(schedInput)

	params := jsonutils.Marshal(schedInput).(*jsonutils.JSONDict)
	task.SetStage("OnScheduleComplete", params)

	s := auth.GetAdminSession(ctx, options.Options.Region, "")
	output, err := modules.SchedManager.DoSchedule(s, schedInput, len(objs))
	if err != nil {
		onSchedulerRequestFail(ctx, task, objs, fmt.Sprintf("Scheduler fail: %s", err))
		return
	}
	onSchedulerResults(ctx, task, objs, output.Candidates)
}

func cancelPendingUsage(ctx context.Context, task IScheduleTask) {
	pendingUsage := models.SQuota{}
	err := task.GetPendingUsage(&pendingUsage, 0)
	if err != nil {
		log.Errorf("Taks GetPendingUsage fail %s", err)
		return
	}
	if !pendingUsage.IsEmpty() {
		err = models.QuotaManager.CancelPendingUsage(ctx, task.GetUserCred(), &pendingUsage, &pendingUsage)
		if err != nil {
			log.Errorf("cancelpendingusage error %s", err)
		}
	}

	pendingRegionUsage := models.SRegionQuota{}
	err = task.GetPendingUsage(&pendingRegionUsage, 0)
	if err != nil {
		log.Errorf("Taks GetRegionPendingUsage fail %s", err)
		return
	}
	if !pendingRegionUsage.IsEmpty() {
		err = models.RegionQuotaManager.CancelPendingUsage(ctx, task.GetUserCred(), &pendingRegionUsage, &pendingRegionUsage)
		if err != nil {
			log.Errorf("cancelpendingusage error %s", err)
		}
	}
}

func onSchedulerRequestFail(
	ctx context.Context,
	task IScheduleTask,
	objs []IScheduleModel,
	reason string,
) {
	for _, obj := range objs {
		onObjScheduleFail(ctx, task, obj, reason)
	}
	task.OnScheduleFailed(ctx, fmt.Sprintf("Schedule failed: %s", reason))
	cancelPendingUsage(ctx, task)
}

func onObjScheduleFail(
	ctx context.Context,
	task IScheduleTask,
	obj IScheduleModel,
	msg string,
) {
	lockman.LockObject(ctx, obj)
	defer lockman.ReleaseObject(ctx, obj)

	reason := "No matching resources"
	if len(msg) > 0 {
		reason = fmt.Sprintf("%s: %s", reason, msg)
	}
	task.OnScheduleFailCallback(ctx, obj, reason)
}

func onSchedulerResults(
	ctx context.Context,
	task IScheduleTask,
	objs []IScheduleModel,
	results []*schedapi.CandidateResource,
) {
	succCount := 0
	for idx := 0; idx < len(objs); idx += 1 {
		obj := objs[idx]
		result := results[idx]

		if len(result.Error) != 0 {
			onObjScheduleFail(ctx, task, obj, fmt.Sprintf("%s", result.Error))
			continue
		}

		if result.BackupCandidate == nil {
			// normal schedule
			onScheduleSucc(ctx, task, obj, result)
		} else {
			// backup schedule
			onMasterSlaveScheduleSucc(ctx, task, obj, result, result.BackupCandidate)
		}
		succCount += 1
	}
	if succCount == 0 {
		task.OnScheduleFailed(ctx, "Schedule failed")
	}
	cancelPendingUsage(ctx, task)
}

func onMasterSlaveScheduleSucc(
	ctx context.Context,
	task IScheduleTask,
	obj IScheduleModel,
	master, slave *schedapi.CandidateResource,
) {
	lockman.LockObject(ctx, obj)
	defer lockman.ReleaseObject(ctx, obj)
	task.SaveScheduleResultWithBackup(ctx, obj, master, slave)
	models.HostManager.ClearSchedDescSessionCache(master.HostId, master.SessionId)
	models.HostManager.ClearSchedDescSessionCache(slave.HostId, slave.SessionId)
}

func onScheduleSucc(
	ctx context.Context,
	task IScheduleTask,
	obj IScheduleModel,
	candidate *schedapi.CandidateResource,
) {
	hostId := candidate.HostId
	lockman.LockRawObject(ctx, models.HostManager.KeywordPlural(), hostId)
	defer lockman.ReleaseRawObject(ctx, models.HostManager.KeywordPlural(), hostId)

	task.SaveScheduleResult(ctx, obj, candidate)
	models.HostManager.ClearSchedDescSessionCache(candidate.HostId, candidate.SessionId)
}
