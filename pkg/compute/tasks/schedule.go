package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

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
)

const (
	SCHEDULE        = models.VM_SCHEDULE
	SCHEDULE_FAILED = models.VM_SCHEDULE_FAILED
)

type IScheduleModel interface {
	db.IStandaloneModel

	SetStatus(userCred mcclient.TokenCredential, status string, reason string) error
}

type IScheduleTask interface {
	GetUserCred() mcclient.TokenCredential
	GetSchedParams() *jsonutils.JSONDict
	GetPendingUsage(quota quotas.IQuota) error
	SetStage(stageName string, data *jsonutils.JSONDict) error
	SetStageFailed(ctx context.Context, reason string)

	OnStartSchedule(obj IScheduleModel)
	OnScheduleFailCallback(obj IScheduleModel, reason string)
	OnScheduleComplete(ctx context.Context, items []db.IStandaloneModel, data *jsonutils.JSONDict)
	SaveScheduleResult(ctx context.Context, obj IScheduleModel, hostId string)
	SaveScheduleResultWithBackup(ctx context.Context, obj IScheduleModel, master, slave string)
	OnScheduleFailed(ctx context.Context, reason string)
}

type SSchedTask struct {
	taskman.STask
}

func (self *SSchedTask) GetSchedParams() *jsonutils.JSONDict {
	return self.GetParams()
}

func (self *SSchedTask) OnStartSchedule(obj IScheduleModel) {
	db.OpsLog.LogEvent(obj, db.ACT_ALLOCATING, nil, self.GetUserCred())
	obj.SetStatus(self.GetUserCred(), SCHEDULE, "")
}

func (self *SSchedTask) OnScheduleFailCallback(obj IScheduleModel, reason string) {
	obj.SetStatus(self.GetUserCred(), SCHEDULE_FAILED, reason)
	db.OpsLog.LogEvent(obj, db.ACT_ALLOCATE_FAIL, reason, self.GetUserCred())
	notifyclient.NotifySystemError(obj.GetId(), obj.GetName(), SCHEDULE_FAILED, reason)
}

func (self *SSchedTask) OnScheduleComplete(ctx context.Context, items []db.IStandaloneModel, data *jsonutils.JSONDict) {
	self.SetStageComplete(ctx, nil)
}

func (self *SSchedTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, hostId string) {
	// ...
}

func (self *SSchedTask) SaveScheduleResultWithBackup(ctx context.Context, obj IScheduleModel, master, slave string) {
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
	parmas := task.GetSchedParams()
	schedtags := models.ApplySchedPolicies(parmas)

	task.SetStage("OnScheduleComplete", schedtags)

	s := auth.GetAdminSession(ctx, options.Options.Region, "")
	results, err := modules.SchedManager.DoSchedule(s, parmas, len(objs))
	if err != nil {
		onSchedulerRequestFail(ctx, task, objs, fmt.Sprintf("Scheduler fail: %s", err))
		return
	}
	onSchedulerResults(ctx, task, objs, results)
}

func cancelPendingUsage(ctx context.Context, task IScheduleTask) {
	pendingUsage := models.SQuota{}
	err := task.GetPendingUsage(&pendingUsage)
	if err != nil {
		log.Errorf("Taks GetPendingUsage fail %s", err)
		return
	}
	ownerProjectId, _ := task.GetSchedParams().GetString("owner_tenant_id")
	err = models.QuotaManager.CancelPendingUsage(ctx, task.GetUserCred(), ownerProjectId, &pendingUsage, &pendingUsage)
	if err != nil {
		log.Errorf("cancelpendingusage error %s", err)
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
	task.OnScheduleFailCallback(obj, reason)
}

func onSchedulerResults(
	ctx context.Context,
	task IScheduleTask,
	objs []IScheduleModel,
	results []jsonutils.JSONObject,
) {
	succCount := 0
	for idx := 0; idx < len(objs); idx += 1 {
		obj := objs[idx]
		result := results[idx]
		if result.Contains("candidate", "id") {
			hostId, _ := result.GetString("candidate", "id")
			onScheduleSucc(ctx, task, obj, hostId)
			succCount += 1
		} else if result.Contains("candidate", "master_id") {
			master, _ := result.GetString("candidate", "master_id")
			slave, _ := result.GetString("candidate", "slave_id")
			if len(master) == 0 || len(slave) == 0 {
				onObjScheduleFail(ctx, task, obj, "Scheduler candidates not match")
			} else {
				onMasterSlaveScheduleSucc(ctx, task, obj, master, slave)
			}
		} else if result.Contains("error") {
			msg, _ := result.Get("error")
			onObjScheduleFail(ctx, task, obj, fmt.Sprintf("%s", msg))
		} else {
			msg := fmt.Sprintf("Unknown scheduler result %s", result)
			onObjScheduleFail(ctx, task, obj, msg)
			return
		}
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
	master, slave string,
) {
	lockman.LockObject(ctx, obj)
	defer lockman.ReleaseObject(ctx, obj)
	task.SaveScheduleResultWithBackup(ctx, obj, master, slave)
	models.HostManager.ClearSchedDescCache(master)
	models.HostManager.ClearSchedDescCache(slave)
}

func onScheduleSucc(
	ctx context.Context,
	task IScheduleTask,
	obj IScheduleModel,
	hostId string,
) {
	lockman.LockRawObject(ctx, models.HostManager.KeywordPlural(), hostId)
	defer lockman.ReleaseRawObject(ctx, models.HostManager.KeywordPlural(), hostId)

	task.SaveScheduleResult(ctx, obj, hostId)
	models.HostManager.ClearSchedDescCache(hostId)
}
