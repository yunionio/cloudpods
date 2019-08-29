package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type InstanceSnapshotAndCloneTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(InstanceSnapshotAndCloneTask{})
}

func (self *InstanceSnapshotAndCloneTask) taskFailed(
	ctx context.Context, isp *models.SInstanceSnapshot, reason string) {
	guest := models.GuestManager.FetchGuestById(isp.GuestId)
	guest.SetStatus(self.UserCred, compute.VM_SNAPSHOT_AND_CLONE_FAILED, reason)
	logclient.AddActionLogWithContext(
		ctx, guest, logclient.ACT_VM_SNAPSHOT_AND_CLONE, reason, self.UserCred, false,
	)
	db.OpsLog.LogEvent(guest, db.ACT_VM_SNAPSHOT_AND_CLONE_FAILED, reason, self.UserCred)
	self.SetStageFailed(ctx, reason)
}

func (self *InstanceSnapshotAndCloneTask) taskComplete(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {
	self.finalReleasePendingUsage(ctx)
	guest := models.GuestManager.FetchGuestById(isp.GuestId)
	guest.StartSyncstatus(ctx, self.UserCred, "")
	db.OpsLog.LogEvent(guest, db.ACT_VM_SNAPSHOT_AND_CLONE, "", self.UserCred)
	logclient.AddActionLogWithContext(
		ctx, guest, logclient.ACT_VM_SNAPSHOT_AND_CLONE, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *InstanceSnapshotAndCloneTask) SetStageFailed(ctx context.Context, reason string) {
	self.finalReleasePendingUsage(ctx)
	self.STask.SetStageFailed(ctx, reason)
}

func (self *InstanceSnapshotAndCloneTask) finalReleasePendingUsage(ctx context.Context) {
	pendingUsage := models.SQuota{}
	err := self.GetPendingUsage(&pendingUsage)
	if err == nil && !pendingUsage.IsEmpty() {
		isp := self.GetObject().(*models.SInstanceSnapshot)
		guest := models.GuestManager.FetchGuestById(isp.GuestId)
		quotaPlatform := guest.GetQuotaPlatformID()
		models.QuotaManager.CancelPendingUsage(
			ctx, self.UserCred, rbacutils.ScopeProject,
			guest.GetOwnerId(), quotaPlatform, &pendingUsage, &pendingUsage,
		)
	}
}

func (self *InstanceSnapshotAndCloneTask) OnInit(
	ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {

	isp := obj.(*models.SInstanceSnapshot)
	guest := models.GuestManager.FetchGuestById(isp.GuestId)

	self.SetStage("OnCreateInstanceSnapshot", nil)
	err := isp.StartCreateInstanceSnapshotTask(ctx, self.UserCred, guest.GetOwnerId(), nil, self.Id)
	if err != nil {
		self.taskFailed(ctx, isp, err.Error())
		return
	}
}

func (self *InstanceSnapshotAndCloneTask) OnCreateInstanceSnapshot(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {
	// start create server
	params, err := self.Params.Get("guest_params")
	if err != nil {
		self.taskFailed(ctx, isp, "Failed get new guest params")
		return
	}
	newGuest, input, err := models.GuestManager.CreateGuestFromInstanceSnapshot(
		ctx, self.UserCred, params.(*jsonutils.JSONDict), isp)
	if err != nil {
		self.taskFailed(ctx, isp, err.Error())
		return
	}
	input.Set("parent_task_id", jsonutils.NewString(self.GetTaskId()))
	self.SetStage("OnGuestCreated", nil)
	models.GuestManager.OnCreateComplete(ctx, []db.IModel{newGuest}, self.UserCred, nil, input)
}

func (self *InstanceSnapshotAndCloneTask) OnGuestCreated(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {

	self.taskComplete(ctx, isp, data)
}

func (self *InstanceSnapshotAndCloneTask) OnCreateInstanceSnapshotFailed(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {
	self.taskFailed(ctx, isp, data.String())
}
